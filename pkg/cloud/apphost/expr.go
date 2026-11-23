package apphost

import (
	"fmt"
	"strconv"
	"strings"
)

// selfRef names the resource that owns the environment variable.
const selfRef = "self"

// Expression properties.
const (
	propURL  = "url"
	propHost = "host"
	propPort = "port"
)

// ref is one parsed ${resource[.endpoint].property} expression.
type ref struct {
	Raw      string
	Resource string
	Endpoint string // "" selects the primary endpoint
	Property string
}

// parseExpressions extracts the ${...} references of an environment value.
// "$$" escapes a literal "$".
func parseExpressions(v string) ([]ref, error) {
	var out []ref
	for i := 0; i < len(v); i++ {
		if v[i] != '$' || i+1 >= len(v) {
			continue
		}
		if v[i+1] == '$' {
			i++
			continue
		}
		if v[i+1] != '{' {
			continue
		}
		end := strings.IndexByte(v[i:], '}')
		if end < 0 {
			return out, fmt.Errorf("unterminated expression in %q", v)
		}
		raw := v[i+2 : i+end]
		r, err := parseRef(raw)
		if err != nil {
			return out, err
		}
		out = append(out, r)
		i += end
	}
	return out, nil
}

func parseRef(raw string) (ref, error) {
	parts := strings.Split(raw, ".")
	r := ref{Raw: raw}
	switch len(parts) {
	case 2:
		r.Resource, r.Property = parts[0], parts[1]
	case 3:
		r.Resource, r.Endpoint, r.Property = parts[0], parts[1], parts[2]
	default:
		return r, fmt.Errorf("expression ${%s}: want ${resource.property} or ${resource.endpoint.property}", raw)
	}
	switch r.Property {
	case propURL, propHost, propPort:
	default:
		return r, fmt.Errorf("expression ${%s}: unknown property %q (want url, host or port)", raw, r.Property)
	}
	for _, p := range parts {
		if p == "" {
			return r, fmt.Errorf("expression ${%s}: empty segment", raw)
		}
	}
	return r, nil
}

// lookup resolves resource names to their (allocated) resources.
type lookup func(name string) (*Resource, bool)

// expand substitutes every expression in v. self names the owning resource.
func expand(v string, self *Resource, find lookup) (string, error) {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c != '$' || i+1 >= len(v) {
			b.WriteByte(c)
			continue
		}
		if v[i+1] == '$' {
			b.WriteByte('$')
			i++
			continue
		}
		if v[i+1] != '{' {
			b.WriteByte(c)
			continue
		}
		end := strings.IndexByte(v[i:], '}')
		if end < 0 {
			return "", fmt.Errorf("unterminated expression in %q", v)
		}
		r, err := parseRef(v[i+2 : i+end])
		if err != nil {
			return "", err
		}
		target := self
		if r.Resource != selfRef {
			t, ok := find(r.Resource)
			if !ok {
				return "", fmt.Errorf("expression ${%s}: unknown resource %q", r.Raw, r.Resource)
			}
			target = t
		}
		val, err := property(target, r)
		if err != nil {
			return "", err
		}
		b.WriteString(val)
		i += end
	}
	return b.String(), nil
}

func property(res *Resource, r ref) (string, error) {
	if r.Endpoint == "" && r.Property == propURL && res.External != nil && res.External.URL != "" {
		return res.External.URL, nil
	}
	ep, ok := res.endpoint(r.Endpoint)
	if !ok {
		return "", fmt.Errorf("expression ${%s}: resource %q has no endpoint %q", r.Raw, res.Name, r.Endpoint)
	}
	switch r.Property {
	case propHost:
		return ep.Host, nil
	case propPort:
		return strconv.Itoa(ep.Port), nil
	default:
		return endpointURL(ep), nil
	}
}

// endpointURL renders scheme://host:port, or the explicit URL when set.
func endpointURL(e Endpoint) string {
	if e.URL != "" {
		return e.URL
	}
	return fmt.Sprintf("%s://%s:%d", e.Scheme, e.Host, e.Port)
}
