package orm

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/krewire/krewire/errs"
)

// mongoMemoryStore provides a thread-safe in-memory document store with index support.
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
