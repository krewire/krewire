package router

import (
	"fmt"
	"strings"
)

// Register mounts c's routes onto the router. It accepts both Controller
// and Registrar via type switch so Module and func-based registrars work
// without separate call sites.
func (r *Router) Register(c any) {
	if reg, ok := c.(Registrar); ok {
		reg.RegisterRoutes(r)
		return
	}
	if ctrl, ok := c.(Controller); ok {
		ctrl.Register(r)
		return
	}
	panic("router: Register expects Controller or Registrar")
}

// RouteBuilder fluently configures a single route before registration.
// Obtain one with Router.Route; finish with Handle.
type RouteBuilder struct {
	router  *Router
	method  string
	pattern string
	name    string
	mws     []Middleware
	where   map[string]string
}

// Route starts a fluent route declaration:
//
//	r.Route("GET", "/users/{id}").Name("users.show").Handle(h)
func (r *Router) Route(method, pattern string) *RouteBuilder {
	return &RouteBuilder{router: r, method: method, pattern: pattern}
}

// GET starts a fluent GET route declaration.
func (r *Router) GET(pattern string) *RouteBuilder {
	return r.Route("GET", pattern)
}

// POST starts a fluent POST route declaration.
func (r *Router) POST(pattern string) *RouteBuilder {
	return r.Route("POST", pattern)
}

// PUT starts a fluent PUT route declaration.
func (r *Router) PUT(pattern string) *RouteBuilder {
	return r.Route("PUT", pattern)
}

// PATCH starts a fluent PATCH route declaration.
func (r *Router) PATCH(pattern string) *RouteBuilder {
	return r.Route("PATCH", pattern)
}

// DELETE starts a fluent DELETE route declaration.
func (r *Router) DELETE(pattern string) *RouteBuilder {
	return r.Route("DELETE", pattern)
}

// Name tags the route for reverse lookup via Router.URL. Reusing a name
// overwrites the previous mapping (last wins). The router's namePrefix is prepended.
func (rb *RouteBuilder) Name(name string) *RouteBuilder {
	if rb.router.namePrefix != "" {
		name = rb.router.namePrefix + name
	}
	rb.name = name
	return rb
}

// Where adds a param constraint for this route (introspection only).
func (rb *RouteBuilder) Where(param, constraint string) *RouteBuilder {
	if rb.where == nil {
		rb.where = map[string]string{}
	}
	rb.where[param] = constraint
	return rb
}

// Use attaches middleware scoped to this single route, applied after any
// group middleware and before the handler.
func (rb *RouteBuilder) Use(mws ...Middleware) *RouteBuilder {
	rb.mws = append(rb.mws, mws...)
	return rb
}

// Handle registers the route with its final handler.
func (rb *RouteBuilder) Handle(h HandlerFunc) {
	rb.router.Handle(rb.method, rb.pattern, applyChain(rb.mws, h))
	root := rb.router.root()
	if len(root.routes) > 0 {
		last := root.routes[len(root.routes)-1]
		if len(rb.where) > 0 {
			if last.constraints == nil {
				last.constraints = cloneWhere(rb.where)
			} else {
				for k, v := range rb.where {
					last.constraints[k] = v
				}
			}
		}
		if rb.name != "" {
			if root.named == nil {
				root.named = map[string][]segment{}
			}
			root.named[rb.name] = parsePattern(joinPattern(rb.router.base, rb.pattern))
			last.name = rb.name
		}
	}
	if rb.name != "" || len(rb.where) > 0 {
		rb.router.root().dirty = true
	}
}

// URL rebuilds the path of a named route, substituting params.
func (r *Router) URL(name string, params PathParams) (string, error) {
	segs, ok := r.root().named[name]
	if !ok {
		return "", fmt.Errorf("router: unknown route name %q", name)
	}
	var b strings.Builder
	for _, seg := range segs {
		b.WriteByte('/')
		if !seg.hasParam {
			b.WriteString(seg.text)
			continue
		}
		v, ok := params[seg.param]
		if !ok || v == "" || strings.ContainsAny(v, "/?#") {
			return "", fmt.Errorf("router: route %q missing valid value for param %q", name, seg.param)
		}
		b.WriteString(v)
	}
	out := b.String()
	if out == "" {
		out = "/"
	}
	return out, nil
}
