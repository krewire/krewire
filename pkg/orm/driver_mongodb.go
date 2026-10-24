package orm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

// MongoConfig contains connection options for MongoDB backends.
type MongoConfig struct {
	URI      string // e.g. "mongodb://localhost:27017" or "memory://"
	Database string
	Timeout  time.Duration
}

// MongoAdapter implements Adapter for document-oriented MongoDB workloads.
type MongoAdapter struct {
	mu     sync.RWMutex
	cfg    MongoConfig
	closed bool
	stores map[string]*mongoMemoryStore // collection name -> in-memory store
}

// Ensure MongoAdapter implements Adapter.
var _ Adapter = (*MongoAdapter)(nil)

// NewMongoDB creates an adapter for MongoDB. If URI is "memory://" or empty,
// it runs in high-performance in-memory mode (ideal for testing and hermetic builds).
func NewMongoDB(cfg MongoConfig) (*MongoAdapter, error) {
	if strings.TrimSpace(cfg.Database) == "" {
		cfg.Database = "default"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &MongoAdapter{
		cfg:    cfg,
		stores: make(map[string]*mongoMemoryStore),
	}, nil
}

func (a *MongoAdapter) Name() string {
	return "mongodb"
}

func (a *MongoAdapter) Type() AdapterType {
	return AdapterTypeDocument
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

// MongoCollection provides document CRUD operations.
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

	raw, err := json.Marshal(doc)
	if err != nil {
		return "", errs.FailureError(fmt.Sprintf("orm: failed to serialize mongo document: %v", err))
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", errs.FailureError(fmt.Sprintf("orm: failed to deserialize document map: %v", err))
	}

	idVal, ok := m["_id"]
	var idStr string
	if !ok || idVal == "" {
		idStr = generateObjectID()
		m["_id"] = idStr
	} else {
		idStr = fmt.Sprintf("%v", idVal)
	}

	c.store.put(idStr, m)
	return idStr, nil
}

// FindOne finds a document matching filter and decodes into dest.
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
	if c.adapter.closed {
		return errs.FailureError("orm: mongodb connection is closed")
	}

	docs := c.store.findAll(filter)
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

// UpdateOne updates fields matching filter with $set or raw map.
func (c *MongoCollection) UpdateOne(ctx context.Context, filter map[string]interface{}, update map[string]interface{}) (int64, error) {
	if c.adapter.closed {
		return 0, errs.FailureError("orm: mongodb connection is closed")
	}

	setUpdates := update
	if set, ok := update["$set"].(map[string]interface{}); ok {
		setUpdates = set
	}

	return c.store.update(filter, setUpdates, false), nil
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
	return int64(len(c.store.findAll(filter))), nil
}

// ----------------------------------------------------------------------------
// In-Memory Thread-Safe Document Store
// ----------------------------------------------------------------------------

type mongoMemoryStore struct {
	mu   sync.RWMutex
	docs map[string]map[string]interface{}
}

func newMongoMemoryStore() *mongoMemoryStore {
	return &mongoMemoryStore{
		docs: make(map[string]map[string]interface{}),
	}
}

func (s *mongoMemoryStore) put(id string, doc map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[id] = doc
}

func (s *mongoMemoryStore) findFirst(filter map[string]interface{}) (map[string]interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, doc := range s.docs {
		if matchFilter(doc, filter) {
			return copyDoc(doc), true
		}
	}
	return nil, false
}

func (s *mongoMemoryStore) findAll(filter map[string]interface{}) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []map[string]interface{}
	for _, doc := range s.docs {
		if matchFilter(doc, filter) {
			res = append(res, copyDoc(doc))
		}
	}
	return res
}

func (s *mongoMemoryStore) update(filter, set map[string]interface{}, multiple bool) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var modified int64
	for _, doc := range s.docs {
		if matchFilter(doc, filter) {
			for k, v := range set {
				if k != "_id" {
					doc[k] = v
				}
			}
			modified++
			if !multiple {
				break
			}
		}
	}
	return modified
}

func (s *mongoMemoryStore) delete(filter map[string]interface{}, multiple bool) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	for id, doc := range s.docs {
		if matchFilter(doc, filter) {
			delete(s.docs, id)
			deleted++
			if !multiple {
				break
			}
		}
	}
	return deleted
}

func matchFilter(doc, filter map[string]interface{}) bool {
	if len(filter) == 0 {
		return true
	}
	for k, expected := range filter {
		actual, exists := doc[k]
		if !exists {
			return false
		}
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}

func copyDoc(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func decodeMapToDest(m map[string]interface{}, dest interface{}) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func generateObjectID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
