package apicontract

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/apicontract/internal/bind"
)

// RefPrefix prefixes every schema reference. It is valid in both the
// canonical document and the generated OpenAPI document.
const RefPrefix = "#/components/schemas/"

// Schema is the JSON Schema (2020-12 subset, as used by OpenAPI 3.1) for one
// value. Fields are pointers or omitempty so the encoding stays minimal and
// deterministic.
type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	MinItems             *int               `json:"minItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
}

var (
	timeType = reflect.TypeFor[time.Time]()
	rawType  = reflect.TypeFor[json.RawMessage]()
)

// generator converts Go types into schemas, collecting named structs as
// reusable components.
type generator struct {
	defs  map[string]*Schema
	names map[reflect.Type]string
}

func newGenerator() *generator {
	return &generator{defs: map[string]*Schema{}, names: map[reflect.Type]string{}}
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// componentName returns a stable, unique, OpenAPI-safe name for t.
func (g *generator) componentName(t reflect.Type) string {
	if n, ok := g.names[t]; ok {
		return n
	}
	base := strings.Trim(unsafeName.ReplaceAllString(t.Name(), "_"), "_")
	name := base
	for i := 2; ; i++ {
		if _, taken := g.defs[name]; !taken {
			break
		}
		name = base + "_" + strconv.Itoa(i)
	}
	g.names[t] = name
	return name
}

// schema returns the schema for t, referencing named structs.
func (g *generator) schema(t reflect.Type) *Schema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == timeType:
		return &Schema{Type: "string", Format: "date-time"}
	case t == rawType:
		return &Schema{}
	}
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		zero := 0.0
		return &Schema{Type: "integer", Minimum: &zero}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: g.schema(t.Elem())}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: g.schema(t.Elem())}
	case reflect.Struct:
		if t.Name() == "" {
			return g.object(t, false)
		}
		if _, seen := g.names[t]; !seen {
			name := g.componentName(t)
			g.defs[name] = &Schema{} // reserve before recursing (self-reference)
			*g.defs[name] = *g.object(t, false)
		}
		return &Schema{Ref: RefPrefix + g.names[t]}
	}
	return &Schema{} // interfaces and other dynamic values accept any JSON
}

// object builds an inline object schema. When body is true, path, query,
// and header fields are excluded (they travel outside the JSON body).
func (g *generator) object(t reflect.Type, body bool) *Schema {
	s := &Schema{Type: "object", Properties: map[string]*Schema{}}
	g.fields(t, body, s)
	return s
}

func (g *generator) fields(t reflect.Type, body bool, s *Schema) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		name, _, skip := jsonName(sf)
		if skip || (body && bind.IsLocated(sf)) {
			continue
		}
		if sf.Anonymous && name == "" {
			et := sf.Type
			for et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				g.fields(et, body, s)
				continue
			}
		}
		if !sf.IsExported() {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		prop := g.schema(sf.Type)
		applyRules(prop, sf)
		s.Properties[name] = prop
		if fieldRequired(sf) {
			s.Required = append(s.Required, name)
		}
	}
}

// jsonName follows encoding/json tag rules.
func jsonName(sf reflect.StructField) (name string, omitempty, skip bool) {
	tag := sf.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	name, opts, _ := strings.Cut(tag, ",")
	return name, strings.Contains(","+opts+",", ",omitempty,"), false
}

// fieldRequired mirrors pkg/validation: a `required` rule, or a struct
// pointer without `omitempty` (validation rejects it when nil).
func fieldRequired(sf reflect.StructField) bool {
	rules := ruleSet(sf.Tag.Get("validate"))
	if _, ok := rules["required"]; ok {
		return true
	}
	_, omit := rules["omitempty"]
	return !omit && sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct
}

// ruleSet parses a validate tag into rule name → argument. Pattern arguments
// may contain commas, so a "pattern=" rule consumes the rest of the tag.
func ruleSet(tag string) map[string]string {
	out := map[string]string{}
	for tag != "" {
		var rule string
		if strings.HasPrefix(tag, "pattern=") {
			rule, tag = tag, ""
		} else {
			rule, tag, _ = strings.Cut(tag, ",")
		}
		name, arg, _ := strings.Cut(strings.TrimSpace(rule), "=")
		if name != "" {
			out[name] = arg
		}
	}
	return out
}

// applyRules maps pkg/validation rules onto JSON Schema keywords.
func applyRules(s *Schema, sf reflect.StructField) {
	if doc := sf.Tag.Get("doc"); doc != "" && s.Ref == "" {
		s.Description = doc
	}
	if s.Ref != "" {
		return
	}
	rules := ruleSet(sf.Tag.Get("validate"))
	num := func(arg string) (float64, bool) {
		f, err := strconv.ParseFloat(strings.TrimSpace(arg), 64)
		return f, err == nil
	}
	intp := func(f float64) *int { n := int(f); return &n }
	for name, arg := range rules {
		f, ok := num(arg)
		switch {
		case (name == "min" || name == "max" || name == "len") && ok:
			lo, hi := name != "max", name != "min"
			switch s.Type {
			case "string":
				if lo {
					s.MinLength = intp(f)
				}
				if hi {
					s.MaxLength = intp(f)
				}
			case "array":
				if lo {
					s.MinItems = intp(f)
				}
				if hi {
					s.MaxItems = intp(f)
				}
			case "integer", "number":
				if lo && name != "len" {
					s.Minimum = &f
				}
				if hi && name != "len" {
					s.Maximum = &f
				}
			}
		case name == "email":
			s.Format = "email"
		case name == "pattern":
			s.Pattern = arg
		case name == "oneof":
			for _, tok := range strings.Fields(arg) {
				if s.Type == "integer" {
					if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
						s.Enum = append(s.Enum, n)
					}
					continue
				}
				s.Enum = append(s.Enum, tok)
			}
		}
	}
}
