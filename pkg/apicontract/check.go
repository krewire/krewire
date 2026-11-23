package apicontract

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/apicontract/internal/bind"
	"github.com/krewire/krewire/pkg/version"
)

// Issue is one problem found while validating a contract document.
type Issue struct {
	// Where locates the problem, e.g. "endpoints.CreateTodo.path".
	Where   string `json:"where"`
	Message string `json:"message"`
}

// InvalidError reports every issue of an invalid contract at once.
type InvalidError struct {
	Issues []Issue `json:"issues"`
}

// Error implements error.
func (e *InvalidError) Error() string {
	parts := make([]string, len(e.Issues))
	for i, is := range e.Issues {
		parts[i] = is.Where + ": " + is.Message
	}
	return "apicontract: invalid contract: " + strings.Join(parts, "; ")
}

var (
	identRe   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	serviceRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	methods   = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true}
	noBody    = map[string]bool{"GET": true, "HEAD": true, "DELETE": true, "OPTIONS": true}
	schemes   = map[string]bool{"": true, "bearer": true, "basic": true}
)

// Validate checks the document's structural rules. It is the gate run before
// build or release: a document that fails here must not ship.
func (d *Document) Validate() error {
	var issues []Issue
	add := func(where, format string, args ...any) {
		issues = append(issues, Issue{Where: where, Message: fmt.Sprintf(format, args...)})
	}
	if d.Contract != FormatVersion {
		add("contract", "format %q is not %q", d.Contract, FormatVersion)
	}
	if !serviceRe.MatchString(d.Service) {
		add("service", "name %q must match %s", d.Service, serviceRe)
	}
	if _, err := version.ParseVersion(d.Version); err != nil {
		add("version", "%v", err)
	}
	if d.BasePath != "" && (!strings.HasPrefix(d.BasePath, "/") || strings.HasSuffix(d.BasePath, "/")) {
		add("basePath", "%q must start with / and not end with /", d.BasePath)
	}
	if d.Schemas[d.ErrorSchema] == nil {
		add("errorSchema", "component %q is not defined", d.ErrorSchema)
	}
	names := map[string]bool{}
	routes := map[string]string{}
	for _, ep := range d.Endpoints {
		w := "endpoints." + ep.Name
		if !identRe.MatchString(ep.Name) {
			add(w+".name", "%q must be an identifier", ep.Name)
		}
		if names[ep.Name] {
			add(w+".name", "duplicate endpoint name")
		}
		names[ep.Name] = true
		if !methods[ep.Method] {
			add(w+".method", "unsupported method %q", ep.Method)
		}
		if !strings.HasPrefix(ep.Path, "/") {
			add(w+".path", "%q must start with /", ep.Path)
		}
		route := ep.Method + " " + normalizeRoute(ep.Path)
		if other, dup := routes[route]; dup {
			add(w+".path", "route %s already used by %s", route, other)
		}
		routes[route] = ep.Name
		if ep.Status < 200 || ep.Status > 299 {
			add(w+".status", "success status %d is not 2xx", ep.Status)
		}
		if ep.Body != nil && noBody[ep.Method] {
			add(w+".body", "%s endpoints cannot declare a request body", ep.Method)
		}
		if !schemes[ep.Auth.Scheme] {
			add(w+".auth.scheme", "unsupported scheme %q", ep.Auth.Scheme)
		}
		if len(ep.Auth.Roles) > 0 && !ep.Auth.Required {
			add(w+".auth", "roles require auth.required")
		}
		checkParams(ep, w, add)
		codes := map[string]bool{}
		for _, e := range ep.Errors {
			if e.Status < 400 || e.Status > 599 {
				add(w+".errors."+e.Code, "status %d is not 4xx/5xx", e.Status)
			}
			if !identRe.MatchString(e.Code) {
				add(w+".errors", "code %q must be an identifier", e.Code)
			}
			if codes[e.Code] {
				add(w+".errors."+e.Code, "code declared with more than one status")
			}
			codes[e.Code] = true
		}
		for _, s := range []*Schema{ep.Body, ep.Response} {
			d.checkRefs(s, w, add)
		}
		for _, p := range ep.Params {
			d.checkRefs(p.Schema, w, add)
		}
	}
	for name, s := range d.Schemas {
		d.checkRefs(s, "schemas."+name, add)
	}
	sort.SliceStable(issues, func(i, j int) bool { return issues[i].Where < issues[j].Where })
	if len(issues) > 0 {
		return &InvalidError{Issues: issues}
	}
	return nil
}

func checkParams(ep EndpointDoc, w string, add func(string, string, ...any)) {
	inPath := map[string]bool{}
	for _, n := range bind.PathParams(ep.Path) {
		inPath[n] = true
	}
	seen := map[string]bool{}
	for _, p := range ep.Params {
		key := p.In + ":" + p.Name
		if seen[key] {
			add(w+".params."+p.Name, "duplicate %s parameter", p.In)
		}
		seen[key] = true
		switch p.In {
		case bind.InPath:
			if !inPath[p.Name] {
				add(w+".params."+p.Name, "path parameter is not in %q", ep.Path)
			}
			delete(inPath, p.Name)
		case bind.InQuery, bind.InHeader:
		default:
			add(w+".params."+p.Name, "unsupported location %q", p.In)
		}
		if p.Schema == nil || (p.Schema.Type == "object" || p.Schema.Ref != "") {
			add(w+".params."+p.Name, "parameters must be scalars or arrays of scalars")
		}
	}
	for n := range inPath {
		add(w+".path", "{%s} has no request field tagged path:%q", n, n)
	}
}

// normalizeRoute erases parameter names so /a/{x} and /a/{y} collide.
func normalizeRoute(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			parts[i] = "{}"
		}
	}
	return strings.Join(parts, "/")
}

func (d *Document) checkRefs(s *Schema, w string, add func(string, string, ...any)) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		if !strings.HasPrefix(s.Ref, RefPrefix) || d.Schemas[s.Ref[len(RefPrefix):]] == nil {
			add(w, "unresolved reference %q", s.Ref)
		}
		return
	}
	for _, p := range s.Properties {
		d.checkRefs(p, w, add)
	}
	d.checkRefs(s.Items, w, add)
	d.checkRefs(s.AdditionalProperties, w, add)
}
