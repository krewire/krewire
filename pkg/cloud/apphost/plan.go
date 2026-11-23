package apphost

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// knownDeps returns the dependencies of r that exist in t.
func (t *Topology) knownDeps(r *Resource) []string {
	var out []string
	for _, d := range r.Dependencies() {
		if _, ok := t.Resources[d]; ok && d != r.Name {
			out = append(out, d)
		}
	}
	return out
}

// cycle returns one dependency cycle (first node repeated last), or nil.
func (t *Topology) cycle() []string {
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var stack []string
	var found []string
	var visit func(string) bool
	visit = func(n string) bool {
		color[n] = grey
		stack = append(stack, n)
		for _, d := range t.knownDeps(t.Resources[n]) {
			switch color[d] {
			case grey:
				i := len(stack) - 1
				for stack[i] != d {
					i--
				}
				found = append(append([]string{}, stack[i:]...), d)
				return true
			case white:
				if visit(d) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
		return false
	}
	for _, n := range t.Names() {
		if color[n] == white && visit(n) {
			return found
		}
	}
	return nil
}

// Levels groups resources into startup waves: every resource in wave i
// depends only on resources in earlier waves. Names inside a wave are sorted,
// so the order is deterministic. The topology must be valid (acyclic).
func (t *Topology) Levels() [][]string {
	level := map[string]int{}
	var depth func(string) int
	depth = func(n string) int {
		if l, ok := level[n]; ok {
			return l
		}
		level[n] = 0 // guards against cycles on invalid input
		l := 0
		for _, d := range t.knownDeps(t.Resources[n]) {
			if dl := depth(d) + 1; dl > l {
				l = dl
			}
		}
		level[n] = l
		return l
	}
	var out [][]string
	for _, n := range t.Names() {
		l := depth(n)
		for len(out) <= l {
			out = append(out, nil)
		}
		out[l] = append(out[l], n)
	}
	return out
}

// Order is the flattened startup order; shutdown uses its reverse.
func (t *Topology) Order() []string {
	var out []string
	for _, l := range t.Levels() {
		out = append(out, l...)
	}
	return out
}

// Dependents returns the resources that directly depend on name.
func (t *Topology) Dependents(name string) []string {
	var out []string
	for _, n := range t.Names() {
		for _, d := range t.knownDeps(t.Resources[n]) {
			if d == name {
				out = append(out, n)
			}
		}
	}
	return out
}

// Step is one planned operation.
type Step struct {
	Wave      int      `json:"wave"`
	Resource  string   `json:"resource"`
	Kind      Kind     `json:"kind"`
	Runner    string   `json:"runner"`
	DependsOn []string `json:"dependsOn,omitempty"`
	Readiness string   `json:"readiness,omitempty"`
}

// Plan is the machine-readable execution plan of a validated topology. It is
// what `krew app plan` prints and what an AI workflow receives as context.
type Plan struct {
	Start []Step   `json:"start"`
	Stop  []string `json:"stop"`
}

// NewPlan validates t and returns its start/stop plan.
func NewPlan(t *Topology) (*Plan, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	p := &Plan{}
	for w, names := range t.Levels() {
		for _, n := range names {
			r := t.Resources[n]
			p.Start = append(p.Start, Step{Wave: w, Resource: n, Kind: r.Kind, Runner: r.Runner(), DependsOn: t.knownDeps(r), Readiness: describeProbe(r.Readiness)})
		}
	}
	for i := len(p.Start) - 1; i >= 0; i-- {
		p.Stop = append(p.Stop, p.Start[i].Resource)
	}
	return p, nil
}

// Explain renders the plan as short human-readable text.
func (p *Plan) Explain() string {
	var b strings.Builder
	for _, s := range p.Start {
		fmt.Fprintf(&b, "wave %d: start %s (%s via %s)", s.Wave, s.Resource, s.Kind, s.Runner)
		if len(s.DependsOn) > 0 {
			fmt.Fprintf(&b, " after %s", strings.Join(s.DependsOn, ", "))
		}
		if s.Readiness != "" {
			fmt.Fprintf(&b, ", ready when %s", s.Readiness)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "stop order: %s\n", strings.Join(p.Stop, ", "))
	return b.String()
}

func describeProbe(p *Probe) string {
	switch {
	case p == nil:
		return ""
	case p.TCP != "":
		return "tcp " + p.TCP + " accepts"
	case p.HTTP != nil:
		return "GET " + p.HTTP.Endpoint + p.HTTP.Path + " succeeds"
	case p.Exec != nil:
		return "exec " + p.Exec.Command + " exits 0"
	}
	return ""
}

// Diff lists resource-level changes between two topologies.
type Diff struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

// Empty reports whether nothing changed.
func (d Diff) Empty() bool { return len(d.Added)+len(d.Removed)+len(d.Changed) == 0 }

// Proposal is the validated outcome of ValidateProposal.
type Proposal struct {
	Diff Diff  `json:"diff"`
	Plan *Plan `json:"plan"`
}

// ValidateProposal checks an agent- or human-proposed topology against the
// current one before anything executes. The proposal is normalized on a copy;
// an invalid proposal returns the *ValidationError and no plan.
func ValidateProposal(current, proposed *Topology) (*Proposal, error) {
	next := proposed.Clone()
	next.Normalize()
	plan, err := NewPlan(next)
	if err != nil {
		return nil, err
	}
	var d Diff
	if current != nil {
		current = current.Clone()
		current.Normalize()
		for _, n := range next.Names() {
			old, ok := current.Resources[n]
			switch {
			case !ok:
				d.Added = append(d.Added, n)
			case !reflect.DeepEqual(old, next.Resources[n]):
				d.Changed = append(d.Changed, n)
			}
		}
		for _, n := range current.Names() {
			if _, ok := next.Resources[n]; !ok {
				d.Removed = append(d.Removed, n)
			}
		}
	} else {
		d.Added = next.Names()
	}
	sort.Strings(d.Changed)
	return &Proposal{Diff: d, Plan: plan}, nil
}
