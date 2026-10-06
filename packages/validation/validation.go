// Package validation validates structs using rules declared in `validate`
// struct tags. It is framework-agnostic and stdlib-only, so the web and CLI
// layers can share one rule model.
package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/krewire/krewire/packages/validation/rules"
)

// Struct validates v (a struct or pointer to a struct) against its `validate`
// tags. It returns nil when every rule passes, a *ValidationError collecting
// the first failure per field, or a wrapped error for malformed tags.
func Struct(v any) error {
	if v == nil {
		return &ValidationError{Fields: []FieldError{{Rule: "struct"}}}
	}
	kiw := reflect.ValueOf(v)
	if kiw.Kind() != reflect.Ptr && kiw.Kind() != reflect.Struct {
		return &ValidationError{Fields: []FieldError{{Rule: "struct"}}}
	}
	eff, nilPtr := effective(kiw)
	if nilPtr || eff.Kind() != reflect.Struct {
		return &ValidationError{Fields: []FieldError{{Rule: "struct"}}}
	}
	var errs []FieldError
	if err := walk(eff, "", &errs); err != nil {
		return err
	}
	if len(errs) == 0 {
		return nil
	}
	return &ValidationError{Fields: errs}
}

// Field validates a single value against a rule set, as used for scalar
// checks. It returns nil when the value passes and a *ValidationError
// otherwise.
func Field(value any, tag string) error {
	if strings.TrimSpace(tag) == "" {
		return nil
	}
	var kiw reflect.Value
	if value != nil {
		kiw, _ = effective(reflect.ValueOf(value))
	}
	fe, err := applyRules("", kiw, tag)
	if err != nil {
		return err
	}
	if fe != nil {
		return &ValidationError{Fields: []FieldError{*fe}}
	}
	return nil
}

// walk validates the exported fields of a struct value.
func walk(kiw reflect.Value, path string, errs *[]FieldError) error {
	t := kiw.Type()
	for i := 0; i < kiw.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := kiw.Field(i)
		name := sf.Name
		if path != "" {
			name = path + "." + sf.Name
		}
		tag := sf.Tag.Get("validate")
		eff, nilPtr := effective(fv)
		kind := valueKind(fv)

		if kind == reflect.Struct {
			if nilPtr {
				if hasRule(tag, "omitempty") {
					continue
				}
				*errs = append(*errs, FieldError{Field: name, Rule: "required"})
				continue
			}
			if fe, err := applyRules(name, eff, tag); err != nil {
				return err
			} else if fe != nil {
				*errs = append(*errs, *fe)
			}
			if err := walk(eff, name, errs); err != nil {
				return err
			}
			continue
		}

		fe, err := applyRules(name, eff, tag)
		if err != nil {
			return err
		}
		if fe != nil {
			*errs = append(*errs, *fe)
		}

		if !nilPtr && (eff.Kind() == reflect.Slice || eff.Kind() == reflect.Array || eff.Kind() == reflect.Map) {
			if err := walkElements(eff, name, errs); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkElements(v reflect.Value, path string, errs *[]FieldError) error {
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elem, nilPtr := effective(v.Index(i))
			if nilPtr {
				continue
			}
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			if elem.Kind() == reflect.Struct {
				if err := walk(elem, elemPath, errs); err != nil {
					return err
				}
			} else if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array || elem.Kind() == reflect.Map {
				if err := walkElements(elem, elemPath, errs); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		keys := v.MapKeys()
		sortMapKeys(keys)
		for _, key := range keys {
			val, nilPtr := effective(v.MapIndex(key))
			if nilPtr {
				continue
			}
			elemPath := fmt.Sprintf("%s[%v]", path, key.Interface())
			if val.Kind() == reflect.Struct {
				if err := walk(val, elemPath, errs); err != nil {
					return err
				}
			} else if val.Kind() == reflect.Slice || val.Kind() == reflect.Array || val.Kind() == reflect.Map {
				if err := walkElements(val, elemPath, errs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// applyRules evaluates every rule in tag, reporting the first failing rule.
// A value with `omitempty` that is zero skips all rules. Malformed rules are
// returned as errors, never panics.
func applyRules(field string, v reflect.Value, tag string) (*FieldError, error) {
	if strings.TrimSpace(tag) == "" {
		return nil, nil
	}
	if hasRule(tag, "omitempty") && isZero(v) {
		return nil, nil
	}
	for _, rule := range splitRules(tag) {
		rule = strings.TrimSpace(rule)
		if rule == "" || rule == "omitempty" {
			continue
		}
		name, arg := parseRule(rule)
		fail, err := rules.Evaluate(name, arg, v)
		if err != nil {
			return nil, fmt.Errorf("validate: field %s: %w", field, err)
		}
		if fail {
			return &FieldError{Field: field, Value: interfaceValue(v), Rule: name}, nil
		}
	}
	return nil, nil
}
