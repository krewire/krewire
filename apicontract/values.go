package apicontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Violation is one place where a JSON value breaks its schema.
type Violation struct {
	// Path is a JSON-pointer-like location, e.g. "/items/0/title".
	Path    string `json:"path"`
	Message string `json:"message"`
}

// SchemaError lists every violation found in a JSON value.
type SchemaError struct {
	Violations []Violation `json:"violations"`
}

// Error implements error.
func (e *SchemaError) Error() string {
	parts := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		parts[i] = v.Path + ": " + v.Message
	}
	return "apicontract: schema violation: " + strings.Join(parts, "; ")
}

// ValidateJSON checks raw JSON against a schema of this document. It lets any
// consumer — a contract test, a gateway, a non-Go client — verify wire data
// against the canonical contract without the Go types. JSON null is accepted
// wherever a value may be absent (Go encodes nil pointers, slices, and maps as
// null).
func (d *Document) ValidateJSON(s *Schema, data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return &SchemaError{Violations: []Violation{{Path: "/", Message: "invalid JSON: " + err.Error()}}}
	}
	var out []Violation
	d.check(s, v, "", &out, 0)
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return &SchemaError{Violations: out}
}

func (d *Document) check(s *Schema, v any, path string, out *[]Violation, depth int) {
	s = d.Resolve(s)
	if s == nil || v == nil || depth > 64 {
		return
	}
	fail := func(format string, args ...any) {
		p := path
		if p == "" {
			p = "/"
		}
		*out = append(*out, Violation{Path: p, Message: fmt.Sprintf(format, args...)})
	}
	switch s.Type {
	case "object":
		obj, ok := v.(map[string]any)
		if !ok {
			fail("want object")
			return
		}
		for _, r := range s.Required {
			if _, ok := obj[r]; !ok {
				*out = append(*out, Violation{Path: path + "/" + r, Message: "required property is missing"})
			}
		}
		for k, val := range obj {
			if ps, ok := s.Properties[k]; ok {
				d.check(ps, val, path+"/"+k, out, depth+1)
			} else if s.AdditionalProperties != nil {
				d.check(s.AdditionalProperties, val, path+"/"+k, out, depth+1)
			}
		}
	case "array":
		arr, ok := v.([]any)
		if !ok {
			fail("want array")
			return
		}
		checkLen(len(arr), s.MinItems, s.MaxItems, "items", fail)
		for i, el := range arr {
			d.check(s.Items, el, fmt.Sprintf("%s/%d", path, i), out, depth+1)
		}
	case "string":
		str, ok := v.(string)
		if !ok {
			fail("want string")
			return
		}
		checkLen(utf8.RuneCountInString(str), s.MinLength, s.MaxLength, "characters", fail)
		checkFormat(s.Format, str, fail)
		if s.Pattern != "" {
			if re, err := regexp.Compile(s.Pattern); err == nil && !re.MatchString(str) {
				fail("does not match pattern %q", s.Pattern)
			}
		}
	case "integer", "number":
		n, ok := v.(json.Number)
		if !ok {
			fail("want %s", s.Type)
			return
		}
		f, err := n.Float64()
		if err != nil || (s.Type == "integer" && f != math.Trunc(f)) {
			fail("want %s, got %s", s.Type, n)
			return
		}
		if s.Minimum != nil && f < *s.Minimum {
			fail("%s is below minimum %v", n, *s.Minimum)
		}
		if s.Maximum != nil && f > *s.Maximum {
			fail("%s is above maximum %v", n, *s.Maximum)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			fail("want boolean")
		}
	}
	if len(s.Enum) > 0 && !inEnum(s.Enum, v) {
		fail("value is not one of the allowed enum values")
	}
}

func checkLen(n int, lo, hi *int, unit string, fail func(string, ...any)) {
	if lo != nil && n < *lo {
		fail("has %d %s, want at least %d", n, unit, *lo)
	}
	if hi != nil && n > *hi {
		fail("has %d %s, want at most %d", n, unit, *hi)
	}
}

func checkFormat(format, s string, fail func(string, ...any)) {
	switch format {
	case "date-time":
		if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
			fail("want RFC 3339 date-time")
		}
	case "email":
		if _, err := mail.ParseAddress(s); err != nil {
			fail("want email address")
		}
	}
}

func inEnum(enum []any, v any) bool {
	got := canonical(v)
	for _, e := range enum {
		if canonical(e) == got {
			return true
		}
	}
	return false
}

// canonical renders a JSON value so 3, 3.0 and json.Number("3") compare equal.
func canonical(v any) string {
	switch n := v.(type) {
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return fmt.Sprint(f)
		}
	case float64:
		return fmt.Sprint(n)
	case int64:
		return fmt.Sprint(float64(n))
	case int:
		return fmt.Sprint(float64(n))
	}
	b, _ := json.Marshal(v)
	return string(b)
}
