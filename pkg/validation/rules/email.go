package rules

import (
	"fmt"
	"net/mail"
	"reflect"
)

// EvalEmail evaluates the 'email' rule: string must parse as valid RFC email.
func EvalEmail(arg string, v reflect.Value) (bool, error) {
	if v.Kind() != reflect.String {
		return false, fmt.Errorf("rule \"email\" requires a string field")
	}
	_, err := mail.ParseAddress(v.String())
	return err != nil, nil
}
