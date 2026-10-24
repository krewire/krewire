package validation

import (
	"errors"
	"fmt"
	"strings"
)

// FieldError describes a single validation failure on a field.
type FieldError struct {
	Field string
	Value any
	Rule  string
}

func (fe FieldError) Error() string {
	return fmt.Sprintf("field %s failed on rule %s", fe.Field, fe.Rule)
}

// ValidationError collects all field failures for a validated struct.
type ValidationError struct {
	Fields []FieldError
}

func (ve *ValidationError) Error() string {
	var msgs []string
	for _, f := range ve.Fields {
		msgs = append(msgs, f.Error())
	}
	return "validation failed: " + strings.Join(msgs, "; ")
}

// Is reports whether err matches a target ValidationError.
func (ve *ValidationError) Is(target error) bool {
	var other *ValidationError
	return errors.As(target, &other)
}
