package apicontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"

	"github.com/krewire/krewire/pkg/apicontract/internal/bind"
)

// Document is the canonical compiled contract. It is plain data: it can be
// committed as JSON, diffed without the Go types, and fed to generators.
type Document struct {
	Contract    string             `json:"contract"`
	Service     string             `json:"service"`
	Version     string             `json:"version"`
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	BasePath    string             `json:"basePath,omitempty"`
	Endpoints   []EndpointDoc      `json:"endpoints"`
	Schemas     map[string]*Schema `json:"schemas,omitempty"`
	// ErrorSchema names the component describing ErrorBody.
	ErrorSchema string `json:"errorSchema"`
}

// EndpointDoc is the compiled form of one endpoint.
type EndpointDoc struct {
	Name        string      `json:"name"`
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Status      int         `json:"status"`
	Deprecated  bool        `json:"deprecated,omitempty"`
	Auth        Auth        `json:"auth"`
	Params      []Param     `json:"params,omitempty"`
	Body        *Schema     `json:"body,omitempty"`
	Response    *Schema     `json:"response,omitempty"`
	Errors      []ErrorSpec `json:"errors"`
}

// Param is a path, query, or header parameter.
type Param struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

// Compile validates the service and produces its canonical document.
func (s *Service) Compile() (*Document, error) {
	g := newGenerator()
	doc := &Document{
		Contract:    FormatVersion,
		Service:     s.Name,
		Version:     s.Version,
		Title:       s.Title,
		Description: s.Description,
		BasePath:    s.BasePath,
		Endpoints:   []EndpointDoc{},
	}
	doc.ErrorSchema = g.schema(reflect.TypeFor[ErrorBody]()).Ref[len(RefPrefix):]
	for _, ep := range s.endpoints {
		doc.Endpoints = append(doc.Endpoints, compileEndpoint(g, ep, s.Errors))
	}
	doc.Schemas = g.defs
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}

// MustCompile is Compile for package-level contract variables; it panics on
// an invalid contract.
func (s *Service) MustCompile() *Document {
	doc, err := s.Compile()
	if err != nil {
		panic(err)
	}
	return doc
}

func compileEndpoint(g *generator, ep Endpoint, serviceErrs []ErrorSpec) EndpointDoc {
	d := EndpointDoc{
		Name: ep.Name, Method: ep.Method, Path: ep.Path,
		Summary: ep.Summary, Description: ep.Description,
		Tags: ep.Tags, Status: ep.SuccessStatus(), Deprecated: ep.Deprecated,
		Auth: ep.Auth,
	}
	if len(d.Auth.Roles) > 0 {
		d.Auth.Required = true
	}
	errs := append(append([]ErrorSpec(nil), serviceErrs...), ep.Errors...)
	if req := structType(ep.Request); req != nil && req != reflect.TypeFor[Empty]() {
		for _, f := range bind.Fields(req) {
			ps := g.schema(f.Struct.Type)
			applyRules(ps, f.Struct)
			d.Params = append(d.Params, Param{
				Name: f.Name, In: f.In, Schema: ps,
				Required:    f.In == bind.InPath || fieldRequired(f.Struct),
				Description: f.Struct.Tag.Get("doc"),
			})
		}
		if body := g.object(req, true); len(body.Properties) > 0 {
			d.Body = body
		}
		errs = append(errs, ErrBadRequest, ErrValidation)
	}
	if resp := structType(ep.Response); ep.Response != nil && resp != reflect.TypeFor[Empty]() {
		d.Response = g.schema(ep.Response)
	}
	if d.Auth.Required {
		errs = append(errs, ErrUnauthenticated)
	}
	if len(d.Auth.Roles) > 0 {
		errs = append(errs, ErrForbidden)
	}
	errs = append(errs, ErrInternal)
	d.Errors = normalizeErrors(errs)
	return d
}

func structType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	return t
}

// normalizeErrors de-duplicates by (status, code) keeping the first
// description, then sorts by status and code.
func normalizeErrors(in []ErrorSpec) []ErrorSpec {
	seen := map[[2]any]bool{}
	out := []ErrorSpec{}
	for _, e := range in {
		k := [2]any{e.Status, e.Code}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			return out[i].Status < out[j].Status
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// Endpoint returns the endpoint named name, or nil.
func (d *Document) Endpoint(name string) *EndpointDoc {
	for i := range d.Endpoints {
		if d.Endpoints[i].Name == name {
			return &d.Endpoints[i]
		}
	}
	return nil
}

// Resolve follows a $ref to its component; other schemas return unchanged.
func (d *Document) Resolve(s *Schema) *Schema {
	for i := 0; s != nil && s.Ref != "" && i < 64; i++ {
		s = d.Schemas[s.Ref[len(RefPrefix):]]
	}
	return s
}

// JSON returns the canonical encoding: two-space indented, sorted map keys,
// trailing newline. Identical contracts always encode to identical bytes.
func (d *Document) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReadDocument decodes and validates a canonical document.
func ReadDocument(r io.Reader) (*Document, error) {
	var d Document
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("apicontract: decode document: %w", err)
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

// LoadDocument reads a canonical document from a file.
func LoadDocument(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadDocument(f)
}
