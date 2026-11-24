package router

import (
	"strings"
)

// Route represents a registered HTTP route pattern, handler, and path segments.
type Route struct {
	method      string
	pattern     string
	name        string
	handle      HandlerFunc
	segments    []segment
	constraints map[string]string
}

// Method returns the HTTP method of the route.
func (rt *Route) Method() string { return rt.method }

// Pattern returns the registered path pattern.
func (rt *Route) Pattern() string { return rt.pattern }

// Name returns the route name.
func (rt *Route) Name() string { return rt.name }

type segment struct {
	text     string
	param    string
	hasParam bool
}

// joinPattern mounts pattern under base, preserving a leading slash when the
// base has one (e.g. base "/api/v1" + pattern "/users" -> "/api/v1/users").
func joinPattern(base, pattern string) string {
	trimmed := strings.Trim(pattern, "/")
	if base == "" {
		return "/" + trimmed
	}
	return strings.TrimRight(base, "/") + "/" + trimmed
}

// match reports whether path matches the route pattern and returns extracted parameters.
func (rt *Route) match(path string) (Params, bool) {
	parts := splitPath(path)
	if len(parts) != len(rt.segments) {
		return nil, false
	}
	params := make(Params, len(rt.segments))
	for i, seg := range rt.segments {
		if seg.hasParam {
			if parts[i] == "" {
				return nil, false
			}
			params[seg.param] = parts[i]
			continue
		}
		if seg.text != parts[i] {
			return nil, false
		}
	}
	return params, true
}

func parsePattern(pattern string) []segment {
	segs := make([]segment, 0, 8)
	for _, part := range splitPath(pattern) {
		if len(part) > 2 && strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			segs = append(segs, segment{param: part[1 : len(part)-1], hasParam: true})
			continue
		}
		segs = append(segs, segment{text: part})
	}
	return segs
}

// splitPath splits a URL path into non-empty segments.
func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
}
