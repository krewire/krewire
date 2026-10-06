package rules

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// EvalOneof evaluates the 'oneof' rule: value must be one of space-delimited options.
func EvalOneof(arg string, v reflect.Value) (bool, error) {
	tokens := strings.Fields(arg)
	if len(tokens) == 0 {
		return true, nil
	}
	switch v.Kind() {
	case reflect.String:
		for _, tok := range tokens {
			if v.String() == tok {
				return false, nil
			}
		}
		return true, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := v.Int()
		for _, tok := range tokens {
			parsed, err := strconv.ParseInt(tok, 10, 64)
			if err == nil && parsed == val {
				return false, nil
			}
		}
		return true, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := v.Uint()
		for _, tok := range tokens {
			parsed, err := strconv.ParseUint(tok, 10, 64)
			if err == nil && parsed == val {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("oneof requires string, int, or uint, got %s", v.Kind())
	}
}
