package rules

import "reflect"

// EvalRequired evaluates the 'required' rule: field must not be zero value.
func EvalRequired(arg string, v reflect.Value) (bool, error) {
	return isZero(v), nil
}
