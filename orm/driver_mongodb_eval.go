package orm

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"

	"github.com/krewire/krewire/errs"
)

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
