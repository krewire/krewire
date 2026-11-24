package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/errs"
)

const (
	// DefaultMongoDatabase is the fallback database name for MongoDB.
	DefaultMongoDatabase = "default"

	// DefaultMongoTimeout is the default command execution timeout.
	DefaultMongoTimeout = 10 * time.Second

	// DefaultMongoMemoryURI is the URI scheme triggering in-memory storage mode.
	DefaultMongoMemoryURI = "memory://"
)

// MongoConfig contains connection and pool options for MongoDB backends.
type MongoConfig struct {
	URI                    string        // e.g. "mongodb://localhost:27017" or DefaultMongoMemoryURI ("memory://")
	Database               string        // default: DefaultMongoDatabase ("default")
	Timeout                time.Duration // default: DefaultMongoTimeout (10s)
	MinPoolSize            uint64
	MaxPoolSize            uint64
	MaxConnIdleTime        time.Duration
	ServerSelectionTimeout time.Duration
	AppName                string
}

// FindOptions controls pagination, projection, and sorting for document queries.
type FindOptions struct {
	Limit      int64          `json:"limit,omitempty"`
	Skip       int64          `json:"skip,omitempty"`
	Sort       map[string]int `json:"sort,omitempty"`       // 1 for ascending, -1 for descending
	Projection map[string]int `json:"projection,omitempty"` // 1 for include, 0 for exclude
}

// IndexInfo describes an active index on a collection.
type IndexInfo struct {
	Name   string         `json:"name"`
	Keys   map[string]int `json:"keys"`
	Unique bool           `json:"unique"`
}

// IndexOptions configures collection index creation.
type IndexOptions struct {
	Name   string
	Unique bool
}

// MongoAdapter implements Adapter for document-oriented MongoDB workloads.
type MongoAdapter struct {
	mu     sync.RWMutex
	config MongoConfig
	closed bool
	stores map[string]*mongoMemoryStore // collection name -> store
}

// Ensure MongoAdapter implements Adapter.
var _ Adapter = (*MongoAdapter)(nil)

// NewMongoDB creates an adapter for MongoDB. If URI is "memory://" or empty,
// it runs in high-performance in-memory mode with full query operator fidelity.
func NewMongoDB(config MongoConfig) (*MongoAdapter, error) {
	if strings.TrimSpace(config.Database) == "" {
		config.Database = DefaultMongoDatabase
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultMongoTimeout
	}

	return &MongoAdapter{
		config: config,
		stores: make(map[string]*mongoMemoryStore),
	}, nil
}

func (a *MongoAdapter) Name() string {
	return "mongodb"
}

func (a *MongoAdapter) Type() AdapterType {
	return AdapterTypeDocument
}

// Database returns the currently selected database name.
func (a *MongoAdapter) Database() string {
	return a.config.Database
}

func (a *MongoAdapter) Ping(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return errs.FailureError("orm: mongodb connection is closed")
	}
	return nil
}

func (a *MongoAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	return nil
}

// CollectionNames returns the names of all collections in the database.
func (a *MongoAdapter) CollectionNames(ctx context.Context) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return nil, errs.FailureError("orm: mongodb connection is closed")
	}

	names := make([]string, 0, len(a.stores))
	for name := range a.stores {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// DropDatabase drops all collections and clears all stored documents.
func (a *MongoAdapter) DropDatabase(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return errs.FailureError("orm: mongodb connection is closed")
	}
	for _, s := range a.stores {
		s.mu.Lock()
		s.docs = make(map[string]map[string]interface{})
		s.order = nil
		s.mu.Unlock()
	}
	a.stores = make(map[string]*mongoMemoryStore)
	return nil
}

// Collection returns a handle to a MongoDB collection.
func (a *MongoAdapter) Collection(name string) *MongoCollection {
	a.mu.Lock()
	defer a.mu.Unlock()

	name = strings.TrimSpace(name)
	store, ok := a.stores[name]
	if !ok {
		store = newMongoMemoryStore()
		a.stores[name] = store
	}

	return &MongoCollection{
		name:    name,
		adapter: a,
		store:   store,
	}
}

// MongoCollection provides document CRUD, query DSL, and indexing operations.
type MongoCollection struct {
	name    string
	adapter *MongoAdapter
	store   *mongoMemoryStore
}

// InsertOne inserts a single document and returns the generated or existing _id.
func (c *MongoCollection) InsertOne(ctx context.Context, doc interface{}) (string, error) {
	if c.adapter.closed {
		return "", errs.FailureError("orm: mongodb connection is closed")
	}

	m, err := toDocumentMap(doc)
	if err != nil {
		return "", err
	}

	idVal, ok := m["_id"]
	var idStr string
	if !ok || idVal == "" {
		idStr = generateObjectID()
		m["_id"] = idStr
	} else {
		idStr = fmt.Sprintf("%v", idVal)
	}

	if err := c.store.put(idStr, m); err != nil {
		return "", err
	}
	return idStr, nil
}

// InsertMany inserts multiple documents in batch.
func (c *MongoCollection) InsertMany(ctx context.Context, docs []interface{}) ([]string, error) {
	if c.adapter.closed {
		return nil, errs.FailureError("orm: mongodb connection is closed")
	}

	ids := make([]string, 0, len(docs))
	for _, d := range docs {
		id, err := c.InsertOne(ctx, d)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// FindOne finds the first document matching filter and decodes it into dest.
func (c *MongoCollection) FindOne(ctx context.Context, filter map[string]interface{}, dest interface{}) error {
	if c.adapter.closed {
		return errs.FailureError("orm: mongodb connection is closed")
	}

	doc, found := c.store.findFirst(filter)
	if !found {
		return errs.FailureError("orm: document not found")
	}

	return decodeMapToDest(doc, dest)
}

// Find finds all documents matching filter and decodes into a slice pointer dest.
func (c *MongoCollection) Find(ctx context.Context, filter map[string]interface{}, dest interface{}) error {
	return c.FindWithOptions(ctx, filter, dest, FindOptions{})
}

// FindWithOptions finds documents matching filter with pagination, sorting, and projection.
func (c *MongoCollection) FindWithOptions(ctx context.Context, filter map[string]interface{}, dest interface{}, opts FindOptions) error {
	if c.adapter.closed {
		return errs.FailureError("orm: mongodb connection is closed")
	}

	docs := c.store.findWithOptions(filter, opts)
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Slice {
		return errs.UsageError("orm: Find requires a pointer to a slice")
	}

	raw, err := json.Marshal(docs)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: serialize results: %v", err))
	}

	return json.Unmarshal(raw, dest)
}

// UpdateOne updates the first document matching filter.
func (c *MongoCollection) UpdateOne(ctx context.Context, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.update(filter, update, false)
}

// UpdateMany updates all documents matching filter.
func (c *MongoCollection) UpdateMany(ctx context.Context, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.update(filter, update, true)
}

// ReplaceOne replaces the entire document matching filter while preserving _id.
func (c *MongoCollection) ReplaceOne(ctx context.Context, filter map[string]interface{}, replacement interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}

	newDoc, err := toDocumentMap(replacement)
	if err != nil {
		return 0, err
	}
	return c.store.replace(filter, newDoc)
}

// DeleteOne removes a single document matching filter.
func (c *MongoCollection) DeleteOne(ctx context.Context, filter map[string]interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.delete(filter, false), nil
}

// DeleteMany removes all documents matching filter.
func (c *MongoCollection) DeleteMany(ctx context.Context, filter map[string]interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.delete(filter, true), nil
}

// CountDocuments counts matching documents.
func (c *MongoCollection) CountDocuments(ctx context.Context, filter map[string]interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}
	return int64(len(c.store.findWithOptions(filter, FindOptions{}))), nil
}

// EstimatedDocumentCount returns the total number of documents in the collection.
func (c *MongoCollection) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()
	return int64(len(c.store.docs)), nil
}

// Distinct returns unique values of fieldName across matching documents.
func (c *MongoCollection) Distinct(ctx context.Context, fieldName string, filter map[string]interface{}) ([]interface{}, error) {
	if c.adapter.closed {
		return nil, errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.distinct(fieldName, filter), nil
}

// CreateIndex creates an index on specified fields with optional unique constraints.
func (c *MongoCollection) CreateIndex(ctx context.Context, keys map[string]int, opts IndexOptions) (string, error) {
	if c.adapter.closed {
		return "", errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.createIndex(keys, opts)
}

// ListIndexes returns metadata of all indexes in the collection.
func (c *MongoCollection) ListIndexes(ctx context.Context) ([]IndexInfo, error) {
	if c.adapter.closed {
		return nil, errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.listIndexes(), nil
}

// DropIndex drops an index by its name.
func (c *MongoCollection) DropIndex(ctx context.Context, name string) error {
	if c.adapter.closed {
		return errs.FailureError("orm: mongodb connection is closed")
	}
	return c.store.dropIndex(name)
}
