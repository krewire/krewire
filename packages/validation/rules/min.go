package rules

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// EvalMin evaluates the 'min' rule: number >= min, string/slice/map length >= min.
func EvalMin(arg string, v reflect.Value) (bool, error) {
	want, err := strconv.ParseFloat(strings.TrimSpace(arg), 64)
	if err != nil {
		return false, fmt.Errorf("invalid bound %q", arg)
	}
	var got float64
	switch v.Kind() {
	case reflect.String:
		got = float64(len(v.String()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		got = float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		got = float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		got = v.Float()
	case reflect.Slice, reflect.Array, reflect.Map:
		got = float64(v.Len())
	default:
		return false, fmt.Errorf("min/max require a number, string, slice, map, or array, got %s", v.Kind())
	}
	return got < want, nil
}
