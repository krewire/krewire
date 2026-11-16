package web

import (
	"fmt"
	"sort"
	"strings"
)

// RouteInfo describes one registered route for introspection.
type RouteInfo struct {
	Method      string
	Pattern     string
	Name        string
	Static      bool
	Constraints map[string]string
}

// Routes returns a snapshot of all registered routes and statics, in
// registration order (statics appended after routes).
func (r *Router) Routes() []RouteInfo {
	root := r.root()
	out := make([]RouteInfo, 0, len(root.routes)+len(root.statics))
	for _, rt := range root.routes {
		out = append(out, RouteInfo{Method: rt.method, Pattern: rt.pattern, Name: rt.name, Constraints: cloneWhere(rt.constraints)})
	}
	for _, st := range root.statics {
		out = append(out, RouteInfo{Method: "STATIC", Pattern: st.prefix, Static: true})
	}
	return out
}

// HasRoute reports whether method+pattern is already registered. Pattern is
// normalized via joinPattern against the receiver's base when the receiver is
// a Group; call on the root for absolute checks.
func (r *Router) HasRoute(method, pattern string) bool {
	root := r.root()
	normalized := pattern
	if !strings.HasPrefix(normalized, "/") && normalized != "" {
		normalized = "/" + normalized
	}
	resolved := normalized
	if r.base != "" {
		if strings.HasPrefix(normalized, r.base) && (len(normalized) == len(r.base) || normalized[len(r.base)] == '/') {
			resolved = normalized
		} else {
			resolved = joinPattern(r.base, normalized)
		}
	}
	for _, rt := range root.routes {
		if rt.method == method && rt.pattern == resolved {
			return true
		}
	}
	return false
}

// RouteExists is an alias for HasRoute.
func (r *Router) RouteExists(method, pattern string) bool { return r.HasRoute(method, pattern) }

// CheckDuplicates returns every route occurrence after its first appearance
// (method+pattern). Empty means the table is conflict-free.
func (r *Router) CheckDuplicates() []RouteInfo {
	root := r.root()
	seen := map[string]bool{}
	var dups []RouteInfo
	for _, rt := range root.routes {
		key := rt.method + " " + rt.pattern
		if seen[key] {
			dups = append(dups, RouteInfo{Method: rt.method, Pattern: rt.pattern, Name: rt.name})
			continue
		}
		seen[key] = true
	}
	return dups
}

// Validate reports the first duplicate route as an error, or nil if none.
func (r *Router) Validate() error {
	if dups := r.CheckDuplicates(); len(dups) > 0 {
		d := dups[0]
		return fmt.Errorf("web: duplicate route %s %s", d.Method, d.Pattern)
	}
	return nil
}

// DebugString returns a deterministic, human-readable route manifest.
func (r *Router) DebugString() string {
	infos := r.Routes()
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Method != infos[j].Method {
			return infos[i].Method < infos[j].Method
		}
		return infos[i].Pattern < infos[j].Pattern
	})
	var b strings.Builder
	for _, ri := range infos {
		if ri.Static {
			fmt.Fprintf(&b, "%-7s %s\n", ri.Method, ri.Pattern)
			continue
		}
		if ri.Name != "" {
			if len(ri.Constraints) > 0 {
				fmt.Fprintf(&b, "%-7s %-30s -> %s %v\n", ri.Method, ri.Pattern, ri.Name, ri.Constraints)
			} else {
				fmt.Fprintf(&b, "%-7s %-30s -> %s\n", ri.Method, ri.Pattern, ri.Name)
			}
		} else {
			if len(ri.Constraints) > 0 {
				fmt.Fprintf(&b, "%-7s %s %v\n", ri.Method, ri.Pattern, ri.Constraints)
			} else {
				fmt.Fprintf(&b, "%-7s %s\n", ri.Method, ri.Pattern)
			}
		}
	}
	return b.String()
}
