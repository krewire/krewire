// Package bind holds the request-field location rules shared by the contract
// compiler, the server binding, and the typed client, so the three can never
// disagree about where a field travels on the wire.
package bind

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Location names where a request field travels.
const (
	InPath   = "path"
	InQuery  = "query"
	InHeader = "header"
)

// Field is one request struct field bound to a path, query, or header slot.
type Field struct {
	// Index is the reflect field index path from the request struct.
	Index []int
	// Name is the wire name (path segment, query key, or header name).
	Name string
	// In is InPath, InQuery, or InHeader.
	In string
	// Struct is the Go struct field.
	Struct reflect.StructField
}

// Fields returns the location-tagged fields of t (a struct or pointer to a
// struct), in declaration order. Embedded structs are flattened.
func Fields(t reflect.Type) []Field {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	var out []Field
	collect(t, nil, &out)
	return out
}

func collect(t reflect.Type, prefix []int, out *[]Field) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		idx := append(append([]int(nil), prefix...), i)
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct && sf.Tag.Get("json") == "" {
			collect(sf.Type, idx, out)
			continue
		}
		if !sf.IsExported() {
			continue
		}
		for _, in := range []string{InPath, InQuery, InHeader} {
			if name, ok := sf.Tag.Lookup(in); ok && name != "" && name != "-" {
				*out = append(*out, Field{Index: idx, Name: name, In: in, Struct: sf})
				break
			}
		}
	}
}

// IsLocated reports whether sf carries a path, query, or header tag and so
// never travels in the JSON body.
func IsLocated(sf reflect.StructField) bool {
	for _, in := range []string{InPath, InQuery, InHeader} {
		if name, ok := sf.Tag.Lookup(in); ok && name != "" && name != "-" {
			return true
		}
	}
	return false
}

// Set parses the raw wire values into v. Slices take every value; scalars
// take the first.
func Set(v reflect.Value, raw []string) error {
	if len(raw) == 0 {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return Set(v.Elem(), raw)
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() != reflect.Uint8 {
		out := reflect.MakeSlice(v.Type(), len(raw), len(raw))
		for i, s := range raw {
			if err := setScalar(out.Index(i), s); err != nil {
				return err
			}
		}
		v.Set(out)
		return nil
	}
	return setScalar(v, raw[0])
}

func setScalar(v reflect.Value, s string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("invalid boolean %q", s)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid integer %q", s)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid unsigned integer %q", s)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid number %q", s)
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported parameter kind %s", v.Kind())
	}
	return nil
}

// Format renders v as wire values, the inverse of Set. A nil pointer or an
// empty slice yields no values.
func Format(v reflect.Value) []string {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return Format(v.Elem())
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() != reflect.Uint8 {
		out := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			out = append(out, Format(v.Index(i))...)
		}
		return out
	}
	switch v.Kind() {
	case reflect.String:
		return []string{v.String()}
	case reflect.Bool:
		return []string{strconv.FormatBool(v.Bool())}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{strconv.FormatInt(v.Int(), 10)}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []string{strconv.FormatUint(v.Uint(), 10)}
	case reflect.Float32, reflect.Float64:
		return []string{strconv.FormatFloat(v.Float(), 'g', -1, v.Type().Bits())}
	}
	return nil
}

// PathParams returns the {name} parameters of a route pattern in order.
func PathParams(path string) []string {
	var out []string
	for _, part := range strings.Split(path, "/") {
		if len(part) > 2 && strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			out = append(out, part[1:len(part)-1])
		}
	}
	return out
}
