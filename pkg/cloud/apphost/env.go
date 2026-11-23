package apphost

import (
	"fmt"
	"net"
	"sort"
	"strconv"
)

// discoveryPrefix starts every variable the orchestrator injects.
const discoveryPrefix = "KREWIRE_"

// Injected variable names. For each referenced resource REF (upper-cased,
// '-' → '_') and each of its endpoints EP:
//
//	KREWIRE_REF_URL / _HOST / _PORT        primary endpoint (or external.url)
//	KREWIRE_REF_EP_URL / _HOST / _PORT     every named endpoint
//
// and for the resource itself KREWIRE_RESOURCE, KREWIRE_KIND and
// KREWIRE_SELF_EP_PORT for each own endpoint.
const (
	EnvResource = discoveryPrefix + "RESOURCE"
	EnvKind     = discoveryPrefix + "KIND"
)

// DiscoveryEnv returns the service discovery variables for res, given the
// allocated topology. It is deterministic (sorted by name).
func DiscoveryEnv(t *Topology, res *Resource) map[string]string {
	out := map[string]string{
		EnvResource: res.Name,
		EnvKind:     string(res.Kind),
	}
	for _, e := range res.Endpoints {
		out[discoveryPrefix+"SELF_"+envName(e.Name)+"_PORT"] = strconv.Itoa(e.Port)
	}
	for _, name := range res.References {
		ref, ok := t.Resources[name]
		if !ok {
			continue
		}
		base := discoveryPrefix + envName(name)
		if ref.External != nil && ref.External.URL != "" {
			out[base+"_URL"] = ref.External.URL
		}
		for i, e := range ref.Endpoints {
			put := func(prefix string) {
				out[prefix+"_URL"] = endpointURL(e)
				out[prefix+"_HOST"] = e.Host
				out[prefix+"_PORT"] = strconv.Itoa(e.Port)
			}
			if i == 0 {
				if _, set := out[base+"_URL"]; !set {
					put(base)
				} else {
					out[base+"_HOST"] = e.Host
					out[base+"_PORT"] = strconv.Itoa(e.Port)
				}
			}
			put(base + "_" + envName(e.Name))
		}
	}
	return out
}

// buildEnv composes base + discovery + expanded explicit env (explicit wins).
func buildEnv(t *Topology, res *Resource, base []string) ([]string, error) {
	vars := DiscoveryEnv(t, res)
	find := func(n string) (*Resource, bool) { r, ok := t.Resources[n]; return r, ok }
	for k, v := range res.Env {
		x, err := expand(v, res, find)
		if err != nil {
			return nil, fmt.Errorf("env %s: %w", k, err)
		}
		vars[k] = x
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	env := append([]string{}, base...)
	for _, k := range keys {
		env = append(env, k+"="+vars[k])
	}
	return env, nil
}

// allocatePorts assigns a free port to every dynamic (port 0) endpoint of a
// managed resource and checks that every fixed port is free. It returns the
// issues found so nothing starts on a conflict.
func allocatePorts(t *Topology) error {
	var issues []Issue
	// Hold every listener until all ports are chosen so no two dynamic
	// endpoints receive the same port.
	var held []net.Listener
	defer func() {
		for _, l := range held {
			_ = l.Close()
		}
	}()
	for _, name := range t.Names() {
		r := t.Resources[name]
		if r.External != nil {
			continue
		}
		for i := range r.Endpoints {
			e := &r.Endpoints[i]
			l, err := net.Listen("tcp", net.JoinHostPort(e.Host, strconv.Itoa(e.Port)))
			if err != nil {
				issues = append(issues, Issue{
					Code: CodePortInUse, Resource: name, Field: "endpoints." + e.Name,
					Message: fmt.Sprintf("cannot bind %s:%d: %v", e.Host, e.Port, err),
					Hint:    "stop the process holding the port or set port: 0",
				})
				continue
			}
			e.Port = l.Addr().(*net.TCPAddr).Port
			held = append(held, l)
		}
	}
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// CodePortInUse is reported by Start when a fixed port is already bound.
const CodePortInUse = "port-in-use"
