// Package engine is the canonical, deterministic infrastructure lifecycle
// engine (issue #138). It owns plan, apply, inspect (drift), reconcile, and
// destroy for any provider implementing pkg/cloud/infra/contract.
//
// The engine — never an agent and never a provider — computes the
// authoritative plan. Apply re-plans under the state lock and refuses to run
// when the result differs from the reviewed plan, so an approval is always
// bound to the exact actions that execute.
package engine

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/cloud/infra/contract"
	"github.com/krewire/krewire/pkg/cloud/infra/store"
)

// Engine binds one provider and one state store to one environment.
type Engine struct {
	Provider    contract.Provider
	Store       store.Store
	Environment string
	// Secrets resolves secret references during apply; nil means no secrets.
	Secrets contract.SecretResolver
	// Policy decides which plans need explicit approval.
	Policy Policy
	// Principal is who runs the engine (a human or an agent).
	Principal Principal
	// Now is the clock; time.Now when nil.
	Now func() time.Time
}

// New returns an engine for environment env.
func New(p contract.Provider, s store.Store, env string) *Engine {
	return &Engine{Provider: p, Store: s, Environment: env}
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Problem is one configuration validation finding.
type Problem struct {
	Address string `json:"address"`
	Message string `json:"message"`
}

// ValidationError reports invalid desired state. It is distinct from drift:
// validation concerns configuration, drift concerns actual infrastructure.
type ValidationError struct {
	Problems []Problem `json:"problems"`
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.Address + ": " + p.Message
	}
	return "infra: invalid desired state: " + strings.Join(parts, "; ")
}

// validate checks desired state against the provider capabilities and
// returns the canonical resources keyed by address.
func validate(caps contract.Capabilities, desired []contract.Resource) (map[string]contract.Resource, error) {
	if err := contract.CheckContract(caps); err != nil {
		return nil, err
	}
	var problems []Problem
	add := func(addr, msg string) { problems = append(problems, Problem{Address: addr, Message: msg}) }
	byAddr := make(map[string]contract.Resource, len(desired))
	for _, r := range desired {
		addr := r.Address()
		if r.Kind == "" || r.Name == "" || strings.ContainsAny(r.Kind+r.Name, "/ \t\n") {
			add(addr, "kind and name must be non-empty and contain no '/' or whitespace")
			continue
		}
		if _, dup := byAddr[addr]; dup {
			add(addr, "declared more than once")
			continue
		}
		schema, ok := caps.Kind(r.Kind)
		if !ok {
			add(addr, fmt.Sprintf("%v: provider %s does not declare kind %q", contract.ErrUnsupportedKind, caps.Provider, r.Kind))
			continue
		}
		cr, err := canonical(r)
		if err != nil {
			add(addr, err.Error())
			continue
		}
		for _, msg := range schema.ValidateResource(cr) {
			add(addr, msg)
		}
		byAddr[addr] = cr
	}
	for _, addr := range sortedAddrs(byAddr) {
		for _, dep := range byAddr[addr].DependsOn {
			if dep == addr {
				add(addr, "depends on itself")
			} else if _, ok := byAddr[dep]; !ok {
				add(addr, fmt.Sprintf("depends on undeclared resource %q", dep))
			}
		}
	}
	if len(problems) == 0 {
		if _, err := order(deps(byAddr)); err != nil {
			add("", err.Error())
		}
	}
	if len(problems) > 0 {
		return nil, &ValidationError{Problems: problems}
	}
	return byAddr, nil
}

// canonical normalises a resource so that equal configuration always
// compares and serialises identically: properties take their JSON form
// (numbers become float64) and dependencies are sorted and de-duplicated.
func canonical(r contract.Resource) (contract.Resource, error) {
	props, err := canonicalMap(r.Properties)
	if err != nil {
		return r, fmt.Errorf("properties are not JSON-serialisable: %w", err)
	}
	r.Properties = props
	if len(r.Secrets) == 0 {
		r.Secrets = nil
	}
	d := slices.Clone(r.DependsOn)
	slices.Sort(d)
	r.DependsOn = slices.Compact(d)
	if len(r.DependsOn) == 0 {
		r.DependsOn = nil
	}
	return r, nil
}

func canonicalMap(m map[string]any) (map[string]any, error) {
	if len(m) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = json.Unmarshal(data, &out)
	return out, err
}

func deps(byAddr map[string]contract.Resource) map[string][]string {
	g := make(map[string][]string, len(byAddr))
	for a, r := range byAddr {
		g[a] = r.DependsOn
	}
	return g
}

// order returns the addresses of g in dependency order (dependencies first),
// breaking ties lexically so the result is deterministic. Edges to nodes
// outside g are ignored.
func order(g map[string][]string) ([]string, error) {
	indeg := make(map[string]int, len(g))
	users := make(map[string][]string, len(g))
	for a, ds := range g {
		indeg[a] += 0
		for _, d := range ds {
			if _, ok := g[d]; ok {
				indeg[a]++
				users[d] = append(users[d], a)
			}
		}
	}
	var ready, out []string
	for a, n := range indeg {
		if n == 0 {
			ready = append(ready, a)
		}
	}
	for len(ready) > 0 {
		sort.Strings(ready)
		a := ready[0]
		ready = ready[1:]
		out = append(out, a)
		for _, u := range users[a] {
			if indeg[u]--; indeg[u] == 0 {
				ready = append(ready, u)
			}
		}
	}
	if len(out) != len(g) {
		var cyc []string
		for a, n := range indeg {
			if n > 0 {
				cyc = append(cyc, a)
			}
		}
		sort.Strings(cyc)
		return nil, fmt.Errorf("dependency cycle among %s", strings.Join(cyc, ", "))
	}
	return out, nil
}

func sortedAddrs[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
