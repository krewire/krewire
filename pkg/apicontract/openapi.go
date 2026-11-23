package apicontract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// OpenAPIVersion is the OpenAPI specification version emitted by OpenAPI.
const OpenAPIVersion = "3.1.0"

// OpenAPI renders the document as an OpenAPI 3.1 description. The output is
// deterministic (sorted keys, stable ordering), so it can be committed and
// compared in golden tests. Krewire-specific metadata that OpenAPI has no
// field for (roles, contract format, error codes) is carried in x-krewire-*
// extensions.
func (d *Document) OpenAPI() ([]byte, error) {
	paths := map[string]map[string]any{}
	usedSchemes := map[string]bool{}
	for _, ep := range d.Endpoints {
		p := d.BasePath + ep.Path
		if paths[p] == nil {
			paths[p] = map[string]any{}
		}
		paths[p][strings.ToLower(ep.Method)] = d.operation(ep, usedSchemes)
	}
	components := map[string]any{"schemas": d.Schemas}
	if len(usedSchemes) > 0 {
		sec := map[string]any{}
		for name := range usedSchemes {
			sec[name] = securityScheme(name)
		}
		components["securitySchemes"] = sec
	}
	info := map[string]any{"title": firstNonEmpty(d.Title, d.Service), "version": d.Version}
	if d.Description != "" {
		info["description"] = d.Description
	}
	root := map[string]any{
		"openapi":                OpenAPIVersion,
		"jsonSchemaDialect":      "https://spec.openapis.org/oas/3.1/dialect/base",
		"info":                   info,
		"paths":                  paths,
		"components":             components,
		"x-krewire-contract":     d.Contract,
		"x-krewire-service":      d.Service,
		"x-krewire-error-schema": d.ErrorSchema,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (d *Document) operation(ep EndpointDoc, usedSchemes map[string]bool) map[string]any {
	op := map[string]any{"operationId": ep.Name}
	setIf := func(k string, v any, ok bool) {
		if ok {
			op[k] = v
		}
	}
	setIf("summary", ep.Summary, ep.Summary != "")
	setIf("description", ep.Description, ep.Description != "")
	setIf("tags", ep.Tags, len(ep.Tags) > 0)
	setIf("deprecated", true, ep.Deprecated)
	if len(ep.Params) > 0 {
		params := make([]map[string]any, 0, len(ep.Params))
		for _, p := range ep.Params {
			m := map[string]any{"name": p.Name, "in": p.In, "schema": p.Schema}
			if p.Required {
				m["required"] = true
			}
			if p.Description != "" {
				m["description"] = p.Description
			}
			params = append(params, m)
		}
		op["parameters"] = params
	}
	if ep.Body != nil {
		op["requestBody"] = map[string]any{"required": true, "content": jsonContent(ep.Body)}
	}
	responses := map[string]any{}
	ok := map[string]any{"description": firstNonEmpty(http.StatusText(ep.Status), "Success")}
	if ep.Response != nil {
		ok["content"] = jsonContent(ep.Response)
	}
	responses[strconv.Itoa(ep.Status)] = ok
	byStatus := map[int][]ErrorSpec{}
	for _, e := range ep.Errors {
		byStatus[e.Status] = append(byStatus[e.Status], e)
	}
	errRef := &Schema{Ref: RefPrefix + d.ErrorSchema}
	for status, specs := range byStatus {
		codes := make([]string, 0, len(specs))
		descs := make([]string, 0, len(specs))
		for _, s := range specs {
			codes = append(codes, s.Code)
			descs = append(descs, "`"+s.Code+"`"+suffix(": ", s.Description))
		}
		sort.Strings(codes)
		sort.Strings(descs)
		responses[strconv.Itoa(status)] = map[string]any{
			"description":           strings.Join(descs, "\n"),
			"content":               jsonContent(errRef),
			"x-krewire-error-codes": codes,
		}
	}
	op["responses"] = responses
	if ep.Auth.Required {
		schemes := []string{"bearer", "basic"}
		if ep.Auth.Scheme != "" {
			schemes = []string{ep.Auth.Scheme}
		}
		var sec []map[string][]string
		for _, s := range schemes {
			name := s + "Auth"
			usedSchemes[name] = true
			sec = append(sec, map[string][]string{name: {}})
		}
		op["security"] = sec
		setIf("x-krewire-roles", ep.Auth.Roles, len(ep.Auth.Roles) > 0)
	}
	return op
}

func jsonContent(s *Schema) map[string]any {
	return map[string]any{"application/json": map[string]any{"schema": s}}
}

func securityScheme(name string) map[string]any {
	scheme := strings.TrimSuffix(name, "Auth")
	m := map[string]any{"type": "http", "scheme": scheme}
	if scheme == "bearer" {
		m["bearerFormat"] = "JWT"
	}
	return m
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func suffix(sep, s string) string {
	if s == "" {
		return ""
	}
	return sep + s
}
