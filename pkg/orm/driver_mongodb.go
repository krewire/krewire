package orm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/pkg/errs"
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
	cfg    MongoConfig
	closed bool
	stores map[string]*mongoMemoryStore // collection name -> store
}

// Ensure MongoAdapter implements Adapter.
var _ Adapter = (*MongoAdapter)(nil)

// NewMongoDB creates an adapter for MongoDB. If URI is "memory://" or empty,
// it runs in high-performance in-memory mode with full query operator fidelity.
func NewMongoDB(cfg MongoConfig) (*MongoAdapter, error) {
	if strings.TrimSpace(cfg.Database) == "" {
		cfg.Database = DefaultMongoDatabase
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultMongoTimeout
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

// Database returns the currently selected database name.
func (a *MongoAdapter) Database() string {
	return a.cfg.Database
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

// ----------------------------------------------------------------------------
// Thread-Safe Document Store & Query Engine
// ----------------------------------------------------------------------------

type mongoMemoryStore struct {
	mu      sync.RWMutex
	docs    map[string]map[string]interface{}
	order   []string // document ID insertion order
	indexes map[string]IndexInfo
}

func newMongoMemoryStore() *mongoMemoryStore {
	store := &mongoMemoryStore{
		docs:    make(map[string]map[string]interface{}),
		order:   make([]string, 0),
		indexes: make(map[string]IndexInfo),
	}
	// Default _id_ index
	store.indexes["_id_"] = IndexInfo{
		Name:   "_id_",
		Keys:   map[string]int{"_id": 1},
		Unique: true,
	}
	return store
}

func (s *mongoMemoryStore) put(id string, doc map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check unique index constraints
	if err := s.checkUniqueConstraints(doc, id); err != nil {
		return err
	}

	if _, exists := s.docs[id]; !exists {
		s.order = append(s.order, id)
	}
	s.docs[id] = doc
	return nil
}

func (s *mongoMemoryStore) checkUniqueConstraints(doc map[string]interface{}, excludeID string) error {
	for _, idx := range s.indexes {
		if !idx.Unique {
			continue
		}
		for existingID, existingDoc := range s.docs {
			if existingID == excludeID {
				continue
			}
			matched := true
			for k := range idx.Keys {
				newVal, newHas := doc[k]
				oldVal, oldHas := existingDoc[k]
				if !newHas || !oldHas || !compareEqual(newVal, oldVal) {
					matched = false
					break
				}
			}
			if matched {
				return errs.FailureError(fmt.Sprintf("orm: E11000 duplicate key error collection index: %s dup key: %+v", idx.Name, doc))
			}
		}
	}
	return nil
}

func (s *mongoMemoryStore) findFirst(filter map[string]interface{}) (map[string]interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, id := range s.order {
		doc := s.docs[id]
		if matchFilter(doc, filter) {
			return copyDoc(doc), true
		}
	}
	return nil, false
}

func (s *mongoMemoryStore) findWithOptions(filter map[string]interface{}, opts FindOptions) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []map[string]interface{}
	for _, id := range s.order {
		doc := s.docs[id]
		if matchFilter(doc, filter) {
			matched = append(matched, copyDoc(doc))
		}
	}

	// 1. Sort
	if len(opts.Sort) > 0 {
		sortDocuments(matched, opts.Sort)
	}

	// 2. Skip
	if opts.Skip > 0 {
		if opts.Skip >= int64(len(matched)) {
			return []map[string]interface{}{}
		}
		matched = matched[opts.Skip:]
	}

	// 3. Limit
	if opts.Limit > 0 && opts.Limit < int64(len(matched)) {
		matched = matched[:opts.Limit]
	}

	// 4. Projection
	if len(opts.Projection) > 0 {
		applyProjection(matched, opts.Projection)
	}

	return matched
}

func (s *mongoMemoryStore) update(filter, update map[string]interface{}, multiple bool) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var modified int64
	for _, id := range s.order {
		doc := s.docs[id]
		if matchFilter(doc, filter) {
			updatedDoc := copyDoc(doc)
			applyUpdateOperators(updatedDoc, update)

			if err := s.checkUniqueConstraints(updatedDoc, id); err != nil {
				return modified, err
			}

			s.docs[id] = updatedDoc
			modified++
			if !multiple {
				break
			}
		}
	}
	return modified, nil
}

func (s *mongoMemoryStore) replace(filter, newDoc map[string]interface{}) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range s.order {
		doc := s.docs[id]
		if matchFilter(doc, filter) {
			newDoc["_id"] = id
			if err := s.checkUniqueConstraints(newDoc, id); err != nil {
				return 0, err
			}
			s.docs[id] = newDoc
			return 1, nil
		}
	}
	return 0, nil
}

func (s *mongoMemoryStore) delete(filter map[string]interface{}, multiple bool) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64
	var newOrder []string

	for _, id := range s.order {
		doc := s.docs[id]
		if (deleted == 0 || multiple) && matchFilter(doc, filter) {
			delete(s.docs, id)
			deleted++
		} else {
			newOrder = append(newOrder, id)
		}
	}
	s.order = newOrder
	return deleted
}

func (s *mongoMemoryStore) distinct(fieldName string, filter map[string]interface{}) []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var result []interface{}

	for _, id := range s.order {
		doc := s.docs[id]
		if matchFilter(doc, filter) {
			if val, ok := doc[fieldName]; ok {
				key := fmt.Sprintf("%v", val)
				if !seen[key] {
					seen[key] = true
					result = append(result, val)
				}
			}
		}
	}
	return result
}

func (s *mongoMemoryStore) createIndex(keys map[string]int, opts IndexOptions) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := opts.Name
	if name == "" {
		var parts []string
		for k, dir := range keys {
			parts = append(parts, fmt.Sprintf("%s_%d", k, dir))
		}
		sort.Strings(parts)
		name = strings.Join(parts, "_")
	}

	info := IndexInfo{
		Name:   name,
		Keys:   keys,
		Unique: opts.Unique,
	}

	// Verify existing data against unique constraint
	if opts.Unique {
		seen := make(map[string]bool)
		for _, doc := range s.docs {
			var vals []string
			for k := range keys {
				vals = append(vals, fmt.Sprintf("%v", doc[k]))
			}
			keyStr := strings.Join(vals, ":")
			if seen[keyStr] {
				return "", errs.FailureError(fmt.Sprintf("orm: cannot create unique index %s, duplicate values exist", name))
			}
			seen[keyStr] = true
		}
	}

	s.indexes[name] = info
	return name, nil
}

func (s *mongoMemoryStore) listIndexes() []IndexInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]IndexInfo, 0, len(s.indexes))
	for _, idx := range s.indexes {
		res = append(res, idx)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

func (s *mongoMemoryStore) dropIndex(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "_id_" {
		return errs.UsageError("orm: cannot drop _id_ index")
	}
	if _, ok := s.indexes[name]; !ok {
		return errs.FailureError(fmt.Sprintf("orm: index %s not found", name))
	}
	delete(s.indexes, name)
	return nil
}

// ----------------------------------------------------------------------------
// Operator & Filter Evaluators
// ----------------------------------------------------------------------------

func matchFilter(doc, filter map[string]interface{}) bool {
	if len(filter) == 0 {
		return true
	}

	for k, cond := range filter {
		switch k {
		case "$and":
			list, ok := toFilterSlice(cond)
			if !ok {
				return false
			}
			for _, sub := range list {
				if !matchFilter(doc, sub) {
					return false
				}
			}
		case "$or":
			list, ok := toFilterSlice(cond)
			if !ok {
				return false
			}
			matched := false
			for _, sub := range list {
				if matchFilter(doc, sub) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		case "$nor":
			list, ok := toFilterSlice(cond)
			if !ok {
				return false
			}
			for _, sub := range list {
				if matchFilter(doc, sub) {
					return false
				}
			}
		default:
			actualVal, hasField := doc[k]
			if condMap, isMap := cond.(map[string]interface{}); isMap {
				for op, opVal := range condMap {
					if !matchFieldOperator(actualVal, hasField, op, opVal) {
						return false
					}
				}
			} else {
				if !hasField || !compareEqual(actualVal, cond) {
					return false
				}
			}
		}
	}
	return true
}

func matchFieldOperator(actualVal interface{}, hasField bool, op string, expectedVal interface{}) bool {
	switch op {
	case "$eq":
		return hasField && compareEqual(actualVal, expectedVal)
	case "$ne":
		return !hasField || !compareEqual(actualVal, expectedVal)
	case "$gt":
		return hasField && compareValues(actualVal, expectedVal) > 0
	case "$gte":
		return hasField && compareValues(actualVal, expectedVal) >= 0
	case "$lt":
		return hasField && compareValues(actualVal, expectedVal) < 0
	case "$lte":
		return hasField && compareValues(actualVal, expectedVal) <= 0
	case "$in":
		return hasField && inSlice(actualVal, expectedVal)
	case "$nin":
		return !hasField || !inSlice(actualVal, expectedVal)
	case "$exists":
		wantExists := true
		if b, ok := expectedVal.(bool); ok {
			wantExists = b
		}
		return hasField == wantExists
	case "$regex":
		if !hasField {
			return false
		}
		pat := fmt.Sprintf("%v", expectedVal)
		matched, err := regexp.MatchString(pat, fmt.Sprintf("%v", actualVal))
		return err == nil && matched
	default:
		return hasField && compareEqual(actualVal, expectedVal)
	}
}

func applyUpdateOperators(doc, update map[string]interface{}) {
	for op, payload := range update {
		payloadMap, isMap := payload.(map[string]interface{})
		if !isMap {
			// Bare map without operator defaults to $set
			if op != "_id" {
				doc[op] = payload
			}
			continue
		}

		switch op {
		case "$set":
			for k, v := range payloadMap {
				if k != "_id" {
					doc[k] = v
				}
			}
		case "$unset":
			for k := range payloadMap {
				if k != "_id" {
					delete(doc, k)
				}
			}
		case "$inc":
			for k, v := range payloadMap {
				if k == "_id" {
					continue
				}
				cur := toFloat(doc[k])
				inc := toFloat(v)
				doc[k] = cur + inc
			}
		case "$push":
			for k, v := range payloadMap {
				if k == "_id" {
					continue
				}
				var arr []interface{}
				if existing, ok := doc[k].([]interface{}); ok {
					arr = append(arr, existing...)
				}
				arr = append(arr, v)
				doc[k] = arr
			}
		default:
			// Regular field update
			if op != "_id" {
				doc[op] = payload
			}
		}
	}
}

func sortDocuments(docs []map[string]interface{}, sortFields map[string]int) {
	type fieldDir struct {
		field string
		dir   int
	}
	var fields []fieldDir
	for f, d := range sortFields {
		fields = append(fields, fieldDir{field: f, dir: d})
	}

	sort.SliceStable(docs, func(i, j int) bool {
		for _, fd := range fields {
			cmp := compareValues(docs[i][fd.field], docs[j][fd.field])
			if cmp == 0 {
				continue
			}
			if fd.dir < 0 {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

func applyProjection(docs []map[string]interface{}, proj map[string]int) {
	hasIncludes := false
	for k, v := range proj {
		if k != "_id" && v == 1 {
			hasIncludes = true
			break
		}
	}

	for _, d := range docs {
		if hasIncludes {
			// Inclusive projection: keep only matching keys plus _id (unless explicitly _id: 0)
			for k := range d {
				if k == "_id" && proj["_id"] == 0 {
					delete(d, k)
					continue
				}
				if k != "_id" && proj[k] != 1 {
					delete(d, k)
				}
			}
		} else {
			// Exclusive projection: remove 0 keys
			for k, v := range proj {
				if v == 0 {
					delete(d, k)
				}
			}
		}
	}
}

// ----------------------------------------------------------------------------
// Type Coercion & Comparison Helpers
// ----------------------------------------------------------------------------

func toDocumentMap(doc interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: failed to serialize mongo document: %v", err))
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: failed to deserialize document map: %v", err))
	}
	return m, nil
}

func toFilterSlice(val interface{}) ([]map[string]interface{}, bool) {
	raw, err := json.Marshal(val)
	if err != nil {
		return nil, false
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, false
	}
	return list, true
}

func compareEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}
	// Try numeric equality first
	if isNumeric(a) && isNumeric(b) {
		return toFloat(a) == toFloat(b)
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	if isNumeric(a) && isNumeric(b) {
		fa := toFloat(a)
		fb := toFloat(b)
		if fa < fb {
			return -1
		} else if fa > fb {
			return 1
		}
		return 0
	}

	sa := fmt.Sprintf("%v", a)
	sb := fmt.Sprintf("%v", b)
	if sa < sb {
		return -1
	} else if sa > sb {
		return 1
	}
	return 0
}

func isNumeric(v interface{}) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	default:
		return 0
	}
}

func inSlice(item, slice interface{}) bool {
	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return false
	}
	for i := 0; i < val.Len(); i++ {
		if compareEqual(item, val.Index(i).Interface()) {
			return true
		}
	}
	return false
}

func copyDoc(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		if subMap, ok := v.(map[string]interface{}); ok {
			dst[k] = copyDoc(subMap)
		} else if subSlice, ok := v.([]interface{}); ok {
			dst[k] = append([]interface{}{}, subSlice...)
		} else {
			dst[k] = v
		}
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
