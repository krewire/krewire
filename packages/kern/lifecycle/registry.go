package lifecycle

import (
	"fmt"
	"sort"
	"sync"

	"github.com/krewire/krewire/packages/kern/errs"
)

// Module is a kernel module. Modules are registered by name and initialized
// against the kernel. Optionally, a module may implement DependsOn() []string
// to declare ordering.
type Module interface {
	Name() string
	Init(*Kernel) error
}

// Registry holds modules by name.
type Registry struct {
	mu   sync.RWMutex
	mods map[string]Module
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{mods: make(map[string]Module)}
}

// Register adds a module. Duplicate names return UsageError.
func (r *Registry) Register(m Module) error {
	if m == nil {
		return errs.UsageError("module is nil")
	}
	name := m.Name()
	if name == "" {
		return errs.UsageError("module name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.mods[name]; exists {
		return errs.UsageError(fmt.Sprintf("duplicate module %q", name))
	}
	r.mods[name] = m
	return nil
}

// Resolve returns the module with the given name.
func (r *Registry) Resolve(name string) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.mods[name]
	return m, ok
}

// Depender is implemented by a Module that declares its own ordering
// dependencies by module name.
type Depender interface {
	DependsOn() []string
}

// Ordered returns modules topologically sorted by DependsOn() when the module
// implements Depender, otherwise in registration order (sorted by name for
// determinism). Circular dependencies are resolved safely without infinite
// loops: any module left unvisited by the sort is appended in name order.
func (r *Registry) Ordered() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := newDependencyGraph(r.mods).topologicalOrder()
	out := make([]Module, 0, len(sorted))
	for _, n := range sorted {
		out = append(out, r.mods[n])
	}
	return out
}

// dependencyGraph builds and orders the module dependency graph. It owns graph
// construction, topological ordering, and cycle handling, leaving Registry
// responsible only for storage and locking.
type dependencyGraph struct {
	// names lists every known module name, sorted, for deterministic ordering.
	names []string
	// dependents maps a module name to the modules that depend on it.
	dependents map[string][]string
	// inDegree counts unresolved dependencies per module.
	inDegree map[string]int
}

// newDependencyGraph indexes mods by name and wires each module to the modules
// that declared a dependency on it. Dependencies naming an unregistered module
// are ignored, since they cannot be ordered against.
func newDependencyGraph(mods map[string]Module) *dependencyGraph {
	g := &dependencyGraph{
		dependents: make(map[string][]string, len(mods)),
		inDegree:   make(map[string]int, len(mods)),
	}
	for n := range mods {
		g.names = append(g.names, n)
		g.inDegree[n] = 0
	}
	sort.Strings(g.names)

	for _, n := range g.names {
		for _, dep := range declaredDependencies(mods[n]) {
			if _, exists := mods[dep]; !exists {
				continue
			}
			g.inDegree[n]++
			g.dependents[dep] = append(g.dependents[dep], n)
		}
	}
	return g
}

// declaredDependencies returns the module's declared dependencies, or nil when
// it declares none.
func declaredDependencies(m Module) []string {
	d, ok := m.(Depender)
	if !ok {
		return nil
	}
	return d.DependsOn()
}

// topologicalOrder returns module names in dependency order. Ties are broken
// alphabetically at every level so the result is stable across runs. Modules
// caught in a dependency cycle are appended last, also in name order, so a cycle
// degrades to a deterministic ordering instead of looping forever.
func (g *dependencyGraph) topologicalOrder() []string {
	var queue []string
	for _, n := range g.names {
		if g.inDegree[n] == 0 {
			queue = append(queue, n)
		}
	}

	var ordered []string
	visited := make(map[string]bool, len(g.names))

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if visited[curr] {
			continue
		}
		visited[curr] = true
		ordered = append(ordered, curr)

		var ready []string
		for _, dep := range g.dependents[curr] {
			g.inDegree[dep]--
			if g.inDegree[dep] == 0 && !visited[dep] {
				ready = append(ready, dep)
			}
		}
		sort.Strings(ready)
		queue = append(queue, ready...)
	}

	if len(ordered) < len(g.names) {
		for _, n := range g.names {
			if !visited[n] {
				ordered = append(ordered, n)
			}
		}
	}
	return ordered
}
