package rules

import "reflect"

func lenOf(v reflect.Value) (int, bool) {
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len(), true
	}
	return 0, false
}

func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		return v.Len() == 0
	}
	return v.IsZero()
}
