package apicontract

import (
	"fmt"
	"strings"
)

// direction tells schema rules whether a value flows from the consumer to the
// service (input) or back (output); the same edit can break one and not the
// other.
type direction int

const (
	input direction = iota
	output
)

func (c *differ) endpoint(o, n EndpointDoc) {
	ep := o.Name
	c.ep = ep
	if o.Method != n.Method {
		c.add(Breaking, "endpoint.method_changed", ep, "", "method changed from %s to %s", o.Method, n.Method)
	}
	if normalizeRoute(o.Path) != normalizeRoute(n.Path) {
		c.add(Breaking, "endpoint.path_changed", ep, "", "path changed from %s to %s", o.Path, n.Path)
	}
	if o.Status != n.Status {
		c.add(Breaking, "endpoint.status_changed", ep, "", "success status changed from %d to %d", o.Status, n.Status)
	}
	if !o.Deprecated && n.Deprecated {
		c.add(NonBreaking, "endpoint.deprecated", ep, "", "endpoint deprecated")
	}
	c.auth(o.Auth, n.Auth)
	c.params(o.Params, n.Params)
	switch {
	case o.Body == nil && n.Body != nil:
		req := len(c.new.Resolve(n.Body).Required) > 0
		c.add(pick(req), "request.body_added", ep, "body", "request body added")
	case o.Body != nil && n.Body == nil:
		c.add(Breaking, "request.body_removed", ep, "body", "request body removed; strict decoding rejects bodies existing consumers send")
	case o.Body != nil:
		c.schema("body", o.Body, n.Body, input)
	}
	switch {
	case o.Response == nil && n.Response != nil:
		c.add(NonBreaking, "response.body_added", ep, "response", "response body added")
	case o.Response != nil && n.Response == nil:
		c.add(Breaking, "response.body_removed", ep, "response", "response body removed")
	case o.Response != nil:
		c.schema("response", o.Response, n.Response, output)
	}
	oldErrs := map[string]ErrorSpec{}
	for _, e := range o.Errors {
		oldErrs[e.Code] = e
	}
	for _, e := range n.Errors {
		prev, ok := oldErrs[e.Code]
		switch {
		case !ok:
			c.add(NonBreaking, "error.added", ep, "errors."+e.Code, "error %s (%d) declared", e.Code, e.Status)
		case prev.Status != e.Status:
			c.add(Breaking, "error.status_changed", ep, "errors."+e.Code, "error %s status changed from %d to %d", e.Code, prev.Status, e.Status)
		}
		delete(oldErrs, e.Code)
	}
	for code, e := range oldErrs {
		c.add(NonBreaking, "error.removed", ep, "errors."+code, "error %s (%d) no longer declared", code, e.Status)
	}
}

func (c *differ) auth(o, n Auth) {
	ep := c.ep
	if !o.Required && n.Required {
		c.add(Breaking, "auth.required", ep, "auth", "authentication became required")
	}
	if o.Required && !n.Required {
		c.add(NonBreaking, "auth.relaxed", ep, "auth", "authentication no longer required")
	}
	if o.Required && n.Required && o.Scheme != n.Scheme && n.Scheme != "" {
		c.add(Breaking, "auth.scheme_changed", ep, "auth", "auth scheme changed from %q to %q", o.Scheme, n.Scheme)
	}
	added, removed := setDiff(o.Roles, n.Roles)
	if len(o.Roles) == 0 && len(n.Roles) > 0 {
		c.add(Breaking, "auth.roles_restricted", ep, "auth.roles", "access restricted to roles %v", n.Roles)
	} else if len(n.Roles) == 0 && len(o.Roles) > 0 {
		c.add(NonBreaking, "auth.roles_relaxed", ep, "auth.roles", "role restriction removed")
	} else {
		// Roles are any-of: removing an allowed role locks its holders out.
		if len(removed) > 0 {
			c.add(Breaking, "auth.roles_removed", ep, "auth.roles", "roles %v no longer admitted", removed)
		}
		if len(added) > 0 {
			c.add(NonBreaking, "auth.roles_added", ep, "auth.roles", "roles %v now admitted", added)
		}
	}
}

func (c *differ) params(o, n []Param) {
	ep := c.ep
	key := func(p Param) string { return p.In + "." + p.Name }
	old := map[string]Param{}
	for _, p := range o {
		old[key(p)] = p
	}
	for _, np := range n {
		k := key(np)
		op, ok := old[k]
		delete(old, k)
		switch {
		case !ok && np.In == "path":
			// A new path parameter shows up as a path change already.
		case !ok:
			c.add(pick(np.Required), "param.added", ep, k, "%s parameter %q added (required=%t)", np.In, np.Name, np.Required)
		default:
			if !op.Required && np.Required {
				c.add(Breaking, "param.required", ep, k, "parameter became required")
			}
			c.schema(k, op.Schema, np.Schema, input)
		}
	}
	for k, op := range old {
		if op.In != "path" {
			c.add(NonBreaking, "param.removed", ep, k, "%s parameter %q removed; the service ignores it", op.In, op.Name)
		}
	}
}

func (c *differ) schema(loc string, o, n *Schema, dir direction) {
	ep := c.ep
	o, n = c.old.Resolve(o), c.new.Resolve(n)
	if o == nil || n == nil || c.seen[[2]*Schema{o, n}] {
		return
	}
	c.seen[[2]*Schema{o, n}] = true
	defer delete(c.seen, [2]*Schema{o, n})
	if o.Type != n.Type {
		c.add(Breaking, "schema.type_changed", ep, loc, "type changed from %q to %q", o.Type, n.Type)
		return
	}
	if o.Format != n.Format {
		c.add(Breaking, "schema.format_changed", ep, loc, "format changed from %q to %q", o.Format, n.Format)
	}
	c.enum(loc, o.Enum, n.Enum, dir)
	if dir == input {
		c.constraints(loc, o, n)
	}
	switch o.Type {
	case "array":
		c.schema(loc+"[]", o.Items, n.Items, dir)
	case "object":
		if o.AdditionalProperties != nil || n.AdditionalProperties != nil {
			c.schema(loc+"{}", o.AdditionalProperties, n.AdditionalProperties, dir)
		}
		c.properties(loc, o, n, dir)
	}
}

func (c *differ) properties(loc string, o, n *Schema, dir direction) {
	ep := c.ep
	oReq, nReq := toSet(o.Required), toSet(n.Required)
	for name, ps := range o.Properties {
		p := loc + "." + name
		nps, ok := n.Properties[name]
		if !ok {
			if dir == input {
				c.add(Breaking, "property.removed", ep, p, "request property removed; strict decoding rejects it")
			} else {
				c.add(Breaking, "property.removed", ep, p, "response property removed")
			}
			continue
		}
		switch {
		case !oReq[name] && nReq[name]:
			c.add(pick(dir == input), "property.required", ep, p, "property became required")
		case oReq[name] && !nReq[name]:
			c.add(pick(dir == output), "property.optional", ep, p, "property became optional")
		}
		c.schema(p, ps, nps, dir)
	}
	for name := range n.Properties {
		if _, ok := o.Properties[name]; !ok {
			c.add(pick(dir == input && nReq[name]), "property.added", ep, loc+"."+name, "property added (required=%t)", nReq[name])
		}
	}
}

func (c *differ) enum(loc string, o, n []any, dir direction) {
	ep := c.ep
	switch {
	case len(o) == 0 && len(n) == 0:
		return
	case len(o) == 0:
		c.add(pick(dir == input), "enum.added", ep, loc, "values restricted to %v", n)
		return
	case len(n) == 0:
		c.add(pick(dir == output), "enum.removed", ep, loc, "value restriction removed")
		return
	}
	added, removed := setDiff(canonicalAll(o), canonicalAll(n))
	if len(removed) > 0 {
		c.add(pick(dir == input), "enum.value_removed", ep, loc, "enum values %v removed", removed)
	}
	if len(added) > 0 {
		c.add(pick(dir == output), "enum.value_added", ep, loc, "enum values %v added", added)
	}
}

// constraints classifies bound changes on input values: tightening breaks
// consumers that sent previously valid values, loosening does not.
func (c *differ) constraints(loc string, o, n *Schema) {
	ep := c.ep
	lower := func(name string, a, b *float64) {
		if tight, changed := tighter(a, b, false); changed {
			c.add(pick(tight), "constraint."+name, ep, loc, "%s changed from %s to %s", name, fmtBound(a), fmtBound(b))
		}
	}
	upper := func(name string, a, b *float64) {
		if tight, changed := tighter(a, b, true); changed {
			c.add(pick(tight), "constraint."+name, ep, loc, "%s changed from %s to %s", name, fmtBound(a), fmtBound(b))
		}
	}
	lower("minimum", o.Minimum, n.Minimum)
	upper("maximum", o.Maximum, n.Maximum)
	lower("minLength", f64(o.MinLength), f64(n.MinLength))
	upper("maxLength", f64(o.MaxLength), f64(n.MaxLength))
	lower("minItems", f64(o.MinItems), f64(n.MinItems))
	upper("maxItems", f64(o.MaxItems), f64(n.MaxItems))
	if o.Pattern != n.Pattern {
		c.add(pick(n.Pattern != ""), "constraint.pattern", ep, loc, "pattern changed from %q to %q", o.Pattern, n.Pattern)
	}
}

// tighter reports whether moving a bound from a to b narrows the accepted
// range (upper selects maximum semantics), and whether it changed at all.
func tighter(a, b *float64, upper bool) (tight, changed bool) {
	switch {
	case a == nil && b == nil:
		return false, false
	case a == nil:
		return true, true
	case b == nil:
		return false, true
	case *a == *b:
		return false, false
	case upper:
		return *b < *a, true
	default:
		return *b > *a, true
	}
}

func f64(p *int) *float64 {
	if p == nil {
		return nil
	}
	f := float64(*p)
	return &f
}

func fmtBound(p *float64) string {
	if p == nil {
		return "none"
	}
	return fmt.Sprint(*p)
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// setDiff returns the sorted elements only in b (added) and only in a
// (removed).
func setDiff(a, b []string) (added, removed []string) {
	as, bs := toSet(a), toSet(b)
	for _, x := range b {
		if !as[x] {
			added = append(added, x)
		}
	}
	for _, x := range a {
		if !bs[x] {
			removed = append(removed, x)
		}
	}
	return added, removed
}

func canonicalAll(xs []any) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strings.Trim(canonical(x), `"`)
	}
	return out
}
