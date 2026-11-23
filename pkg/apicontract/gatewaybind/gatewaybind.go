// Package gatewaybind derives pkg/web/gateway routes from a compiled
// contract, so the gateway never restates a service's paths (ref #137).
//
// It lives apart from pkg/apicontract because the gateway depends on the
// cloud service registry; contract users who do not run a gateway do not pay
// for that dependency.
package gatewaybind

import (
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/apicontract"
	"github.com/krewire/krewire/pkg/web/gateway"
)

// Options shapes the generated routes.
type Options struct {
	// Service is the registry name requests are proxied to. Defaults to the
	// contract's service name.
	Service string
	// TargetURL, when set, proxies directly instead of through the registry.
	TargetURL string
	// Middleware names gateway middleware applied to every route.
	Middleware []string
	// AuthMiddleware names the gateway middleware added to routes whose
	// endpoint requires authentication.
	AuthMiddleware string
}

// Routes returns one gateway route per distinct (method, path) of the
// contract, exposed under the document's BasePath. The service serves the
// same full paths (mount its router under BasePath); the gateway forwards
// paths unchanged. Exact routes come first, then prefix routes (longest
// first), so reloads are stable.
//
// The gateway matches literal paths or prefixes, so an endpoint path with
// parameters (/todos/{id}) becomes a prefix route on its literal head
// (/todos/); the service router performs the full match.
func Routes(doc *apicontract.Document, opts Options) []gateway.Route {
	service := opts.Service
	if service == "" {
		service = doc.Service
	}
	seen := map[string]int{}
	var routes []gateway.Route
	for _, ep := range doc.Endpoints {
		full := doc.BasePath + ep.Path
		prefix := false
		if i := strings.Index(full, "{"); i >= 0 {
			full, prefix = full[:i], true
		}
		key := ep.Method + " " + full
		if idx, dup := seen[key]; dup {
			// Two parameterised endpoints share a head; the route requires
			// auth when any of them does.
			if ep.Auth.Required {
				routes[idx] = withAuth(routes[idx], opts.AuthMiddleware)
			}
			continue
		}
		r := gateway.Route{
			Path:       full,
			Method:     ep.Method,
			Prefix:     prefix,
			Middleware: append([]string(nil), opts.Middleware...),
		}
		if opts.TargetURL != "" {
			r.TargetURL = opts.TargetURL
		} else {
			r.Service = service
		}
		if ep.Auth.Required {
			r = withAuth(r, opts.AuthMiddleware)
		}
		seen[key] = len(routes)
		routes = append(routes, r)
	}
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i], routes[j]
		if a.Prefix != b.Prefix {
			return !a.Prefix
		}
		if a.Path != b.Path {
			// Longer prefixes first so the most specific head wins.
			if a.Prefix && len(a.Path) != len(b.Path) {
				return len(a.Path) > len(b.Path)
			}
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
	return routes
}

func withAuth(r gateway.Route, mw string) gateway.Route {
	r.Auth = true
	if mw != "" {
		for _, m := range r.Middleware {
			if m == mw {
				return r
			}
		}
		r.Middleware = append(r.Middleware, mw)
	}
	return r
}
