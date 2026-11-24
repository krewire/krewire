package config

import (
	"fmt"
	"reflect"
)

func overrideFields(kiw reflect.Value, baseKey, path, prefix string, lookup func(string) (string, bool)) error {
	t := kiw.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := kiw.Field(i)
		name := fieldName(sf)
		fullPath := name
		if path != "" {
			fullPath = path + "." + name
		}

		key := envKey(sf, baseKey, prefix)
		val, ok := "", false
		if key != "" {
			val, ok = lookup(key)
		}

		if isStructKind(fv) {
			child := derefPtr(fv)
			if child.IsValid() {
				if err := overrideFields(child, key, fullPath, prefix, lookup); err != nil {
					return err
				}
			}
			continue
		}

		if !ok || val == "" {
			continue
		}
		if fv.Kind() == reflect.Slice || fv.Kind() == reflect.Map {
			continue
		}
		if err := setScalar(fv, val); err != nil {
			return fmt.Errorf("config: field %s (env %s): %w", fullPath, key, err)
		}
	}
	return nil
}
