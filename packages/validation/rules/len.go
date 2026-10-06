package rules

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// EvalLen evaluates the 'len' rule: exact length of string, slice, map, or array.
func EvalLen(arg string, v reflect.Value) (bool, error) {
	l, ok := lenOf(v)
	if !ok {
		return false, fmt.Errorf("rule \"len\" requires a string, slice, map, or array")
	}
	want, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64)
	if err != nil {
		return false, fmt.Errorf("rule \"len\": %w", err)
	}
	return int64(l) != want, nil
}
