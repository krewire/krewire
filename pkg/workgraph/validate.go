package workgraph

import (
	"fmt"
	"sort"
	"strings"
)

// CycleError reports dependency cycles the architecture forbids.
type CycleError struct {
	// Cycles lists each strongly connected component, sorted.
	Cycles [][]string
}

func (e *CycleError) Error() string {
	parts := make([]string, len(e.Cycles))
	for i, c := range e.Cycles {
		parts[i] = strings.Join(c, " <-> ")
	}
	return "workgraph: forbidden dependency cycle(s): " + strings.Join(parts, "; ")
}

// Cycles returns every strongly connected component of size > 1 formed by
// edges whose kind forbids cycles (all kinds except test-import). Output is
// sorted for determinism.
func (g *Graph) Cycles() [][]string {
	adj := map[string][]string{}
	for _, e := range g.Edges() {
		if e.Kind.acyclic() {
			adj[e.From] = append(adj[e.From], e.To)
		}
	}
	// Tarjan's algorithm.
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	next := 0
	var strong func(v string)
	strong = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, seen := index[w]; !seen {
				strong(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			if len(comp) > 1 {
				sort.Strings(comp)
				out = append(out, comp)
			}
		}
	}
	for _, n := range g.Nodes() {
		if _, seen := index[n.ID]; !seen {
			strong(n.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// Validate checks structural invariants: every forbidden cycle is reported
// as a *CycleError.
func (g *Graph) Validate() error {
	if c := g.Cycles(); len(c) > 0 {
		return &CycleError{Cycles: c}
	}
	return nil
}

// TopoOrder returns node IDs so that every dependency precedes its
// dependents (ignoring test-import edges). Ties break lexically.
func (g *Graph) TopoOrder() ([]string, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	deps := map[string][]string{}
	for _, e := range g.Edges() {
		if e.Kind.acyclic() {
			deps[e.From] = append(deps[e.From], e.To)
		}
	}
	ids := make([]string, 0, len(g.nodes))
	for _, n := range g.Nodes() {
		ids = append(ids, n.ID)
	}
	return topoSort(ids, deps)
}

// topoSort orders ids so that each id follows its deps (Kahn's algorithm with
// a sorted ready set). It fails if a cycle remains.
func topoSort(ids []string, deps map[string][]string) ([]string, error) {
	indeg := map[string]int{}
	rev := map[string][]string{}
	for _, id := range ids {
		indeg[id] += 0
		for _, d := range deps[id] {
			indeg[id]++
			rev[d] = append(rev[d], id)
		}
	}
	var ready []string
	for _, id := range ids {
		if indeg[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	out := make([]string, 0, len(ids))
	for len(ready) > 0 {
		cur := ready[0]
		ready = ready[1:]
		out = append(out, cur)
		for _, d := range rev[cur] {
			indeg[d]--
			if indeg[d] == 0 {
				ready = insertSorted(ready, d)
			}
		}
	}
	if len(out) != len(ids) {
		return nil, fmt.Errorf("workgraph: cycle among %d unordered item(s)", len(ids)-len(out))
	}
	return out, nil
}

func insertSorted(s []string, v string) []string {
	i := sort.SearchStrings(s, v)
	s = append(s, "")
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}
