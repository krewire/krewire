package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ValidateJSON validates a raw document against the published schema of its
// kind. It implements the JSON Schema subset the published schemas use
// ($ref, type, enum, const, properties, required, additionalProperties,
// items), so Go consumers need no schema library; any 2020-12 validator
// gives the same verdict.
func ValidateJSON(k Kind, data []byte) error {
	raw, err := Schema(k)
	if err != nil {
		return err
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return Errorf(CodeInternal, "published schema %s is malformed: %v", k, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return Errorf(CodeParseError, "%s: %v", k, err)
	}
	v := validator{defs: asMap(schema["$defs"])}
	if err := v.check(schema, doc, "$"); err != nil {
		return Errorf(CodeInvalidDocument, "%s: %v", k, err)
	}
	return nil
}

// Decode validates a raw document of any kind against its schema, decodes it
// into its Go type (rejecting unknown fields), and runs the kind's semantic
// validation where it needs no peer document. It returns a pointer to the
// typed document.
func Decode(data []byte) (any, error) {
	var meta TypeMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, Errorf(CodeParseError, "%v", err)
	}
	t, ok := kindTypes[meta.Kind]
	if !ok {
		return nil, Errorf(CodeInvalidDocument, "unknown kind %q", meta.Kind)
	}
	if err := meta.Check(meta.Kind); err != nil {
		return nil, err
	}
	if err := ValidateJSON(meta.Kind, data); err != nil {
		return nil, err
	}
	ptr := reflect.New(t)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(ptr.Interface()); err != nil {
		return nil, Errorf(CodeInvalidDocument, "%s: %v", meta.Kind, err)
	}
	var verr error
	switch d := ptr.Interface().(type) {
	case *ProjectContext:
		verr = d.Validate()
	case *TaskBrief:
		verr = d.Validate(nil)
	case *Plan:
		verr = d.Validate(nil)
	case *EvidenceReport:
		verr = d.Validate()
	case *CommandCatalog:
		verr = d.Validate()
	}
	if verr != nil {
		return nil, verr
	}
	return ptr.Interface(), nil
}

type validator struct{ defs map[string]any }

func asMap(v any) map[string]any { m, _ := v.(map[string]any); return m }

func jsonType(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case json.Number:
		if _, err := x.Int64(); err == nil {
			return "integer"
		}
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", v)
}

func (v validator) check(s map[string]any, doc any, at string) error {
	if ref, ok := s["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/$defs/")
		def := asMap(v.defs[name])
		if def == nil {
			return fmt.Errorf("%s: unresolved $ref %q", at, ref)
		}
		return v.check(def, doc, at)
	}
	if c, ok := s["const"]; ok && fmt.Sprint(c) != fmt.Sprint(doc) {
		return fmt.Errorf("%s: must equal %v", at, c)
	}
	if want, ok := s["type"].(string); ok {
		got := jsonType(doc)
		if got != want && !(want == "number" && got == "integer") {
			return fmt.Errorf("%s: expected %s, got %s", at, want, got)
		}
	}
	if enum, ok := s["enum"].([]any); ok {
		found := false
		for _, e := range enum {
			if e == doc {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s: %v is not one of %v", at, doc, enum)
		}
	}
	switch d := doc.(type) {
	case []any:
		if items := asMap(s["items"]); items != nil {
			for i, e := range d {
				if err := v.check(items, e, fmt.Sprintf("%s[%d]", at, i)); err != nil {
					return err
				}
			}
		}
	case map[string]any:
		return v.object(s, d, at)
	}
	return nil
}

func (v validator) object(s map[string]any, d map[string]any, at string) error {
	props := asMap(s["properties"])
	if req, ok := s["required"].([]any); ok {
		for _, r := range req {
			if _, ok := d[r.(string)]; !ok {
				return fmt.Errorf("%s: missing required property %q", at, r)
			}
		}
	}
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if ps := asMap(props[k]); ps != nil {
			if err := v.check(ps, d[k], at+"."+k); err != nil {
				return err
			}
			continue
		}
		switch ap := s["additionalProperties"].(type) {
		case bool:
			if !ap {
				return fmt.Errorf("%s: unknown property %q", at, k)
			}
		case map[string]any:
			if err := v.check(ap, d[k], at+"."+k); err != nil {
				return err
			}
		}
	}
	return nil
}
