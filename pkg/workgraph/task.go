package workgraph

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// TaskKind classifies a task for reporting and evidence mapping.
type TaskKind string

// Task kinds named by #135.
const (
	TaskFormat   TaskKind = "format"
	TaskLint     TaskKind = "lint"
	TaskTest     TaskKind = "test"
	TaskBuild    TaskKind = "build"
	TaskGenerate TaskKind = "generate"
	TaskPackage  TaskKind = "package"
	TaskRelease  TaskKind = "release"
)

// TaskSpec declares a task. Command arguments may use the placeholders
// {package} (import path), {dir} ("./" + directory relative to the module),
// {module} (module path), and {name} (node name).
type TaskSpec struct {
	Name string   `yaml:"name" json:"name"`
	Kind TaskKind `yaml:"kind" json:"kind"`
	// On selects the node kind that gets one target each (default package).
	On      NodeKind `yaml:"on,omitempty" json:"on,omitempty"`
	Command []string `yaml:"command" json:"command"`
	// DependsOn names tasks that must succeed first: "generate" means the
	// same node's generate target, "^build" means the build target of each
	// direct dependency node (Nx semantics).
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	// Env names environment variables whose values are cache-key inputs, in
	// addition to DefaultEnv.
	Env []string `yaml:"env,omitempty" json:"env,omitempty"`
	// Inputs are extra file globs hashed into the key.
	Inputs []string `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	// Only restricts targets to node names or directories matching a glob.
	Only []string `yaml:"only,omitempty" json:"only,omitempty"`
	// FailOnOutput fails the target when the command prints anything
	// (gofmt -l style gates).
	FailOnOutput bool `yaml:"fail_on_output,omitempty" json:"fail_on_output,omitempty"`
	// NoCache disables restore for tasks with side effects (package, release).
	NoCache bool `yaml:"no_cache,omitempty" json:"no_cache,omitempty"`
}

// DefaultEnv are the environment variables that always enter cache keys.
var DefaultEnv = []string{"GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOEXPERIMENT", "GOTOOLCHAIN"}

// DefaultTasks are the deterministic quality gates of a Go workspace.
func DefaultTasks() []TaskSpec {
	return []TaskSpec{
		{Name: "format", Kind: TaskFormat, On: KindPackage, Command: []string{"gofmt", "-l", "{dir}"}, FailOnOutput: true},
		{Name: "vet", Kind: TaskLint, On: KindPackage, Command: []string{"go", "vet", "{package}"}},
		{Name: "build", Kind: TaskBuild, On: KindPackage, Command: []string{"go", "build", "{package}"}, DependsOn: []string{"^build"}},
		{Name: "test", Kind: TaskTest, On: KindPackage, Command: []string{"go", "test", "{package}"}, DependsOn: []string{"build"}},
	}
}

// Target is one task applied to one node.
type Target struct {
	ID   string   `json:"id"`
	Task string   `json:"task"`
	Node string   `json:"node"`
	Argv []string `json:"argv"`
	// WorkDir is the module directory the command runs in, relative to root.
	WorkDir string   `json:"work_dir"`
	Deps    []string `json:"deps,omitempty"`
	spec    *TaskSpec
}

// TargetID builds the canonical target identifier "<task>#<node id>".
func TargetID(task, node string) string { return task + "#" + node }

// TaskGraph is the target DAG derived from a graph and task specs.
type TaskGraph struct {
	Graph   *Graph
	Specs   []TaskSpec
	targets map[string]*Target
	order   []string
}

// BuildTasks expands specs over the graph. Unknown dependencies, duplicate
// task names, and cycles are rejected.
func BuildTasks(g *Graph, specs []TaskSpec) (*TaskGraph, error) {
	tg := &TaskGraph{Graph: g, Specs: specs, targets: map[string]*Target{}}
	byName := map[string]*TaskSpec{}
	for i := range specs {
		s := &specs[i]
		if s.Name == "" || len(s.Command) == 0 {
			return nil, fmt.Errorf("workgraph: task #%d needs a name and a command", i+1)
		}
		if byName[s.Name] != nil {
			return nil, fmt.Errorf("workgraph: duplicate task %q", s.Name)
		}
		if s.On == "" {
			s.On = KindPackage
		}
		byName[s.Name] = s
	}
	for _, s := range specs {
		for _, d := range s.DependsOn {
			if byName[strings.TrimPrefix(d, "^")] == nil {
				return nil, fmt.Errorf("workgraph: task %q depends on unknown task %q", s.Name, d)
			}
		}
	}
	for i := range specs {
		s := &specs[i]
		for _, n := range g.Nodes() {
			if n.Kind != s.On || (len(s.Only) > 0 && !matchAny(s.Only, n.Name) && !matchAny(s.Only, n.Dir)) {
				continue
			}
			t := &Target{ID: TargetID(s.Name, n.ID), Task: s.Name, Node: n.ID, spec: s}
			t.WorkDir, t.Argv = expand(g, n, s.Command)
			tg.targets[t.ID] = t
		}
	}
	deps := map[string][]string{}
	for _, t := range tg.targets {
		for _, d := range t.spec.DependsOn {
			if name, up := strings.CutPrefix(d, "^"); up {
				for _, dn := range g.Dependencies(t.Node, false) {
					if !onlyTestEdges(g, t.Node, dn) {
						t.Deps = appendIf(t.Deps, tg.targets, TargetID(name, dn))
					}
				}
			} else {
				t.Deps = appendIf(t.Deps, tg.targets, TargetID(d, t.Node))
			}
		}
		t.Deps = sortedUnique(t.Deps)
		deps[t.ID] = t.Deps
	}
	order, err := topoSort(sortedKeysOf(tg.targets), deps)
	if err != nil {
		return nil, fmt.Errorf("workgraph: task graph: %w", err)
	}
	tg.order = order
	return tg, nil
}

func appendIf(list []string, targets map[string]*Target, id string) []string {
	if _, ok := targets[id]; ok {
		return append(list, id)
	}
	return list
}

func onlyTestEdges(g *Graph, from, to string) bool {
	for _, e := range g.EdgesBetween(from, to) {
		if e.Kind != EdgeTestImport {
			return false
		}
	}
	return true
}

func expand(g *Graph, n *Node, argv []string) (string, []string) {
	workDir, rel := ".", n.Dir
	if mod, ok := g.Node(NodeID(KindModule, n.Module)); ok {
		workDir = mod.Dir
		rel = strings.TrimPrefix(strings.TrimPrefix(n.Dir, mod.Dir), "/")
		if mod.Dir == "." {
			rel = n.Dir
		}
	} else if n.Kind == KindModule {
		workDir, rel = n.Dir, "."
	}
	dir := "./" + path.Clean(rel)
	if path.Clean(rel) == "." {
		dir = "."
	}
	r := strings.NewReplacer("{package}", n.Name, "{dir}", dir, "{module}", n.Module, "{name}", n.Name)
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = r.Replace(a)
	}
	return workDir, out
}

// Target returns a target by ID.
func (tg *TaskGraph) Target(id string) (*Target, bool) {
	t, ok := tg.targets[id]
	return t, ok
}

// Order returns all target IDs in dependency order.
func (tg *TaskGraph) Order() []string { return append([]string(nil), tg.order...) }

// PlanStep is an externally proposed target (for example by an AI plan) and
// the dependencies it claims.
type PlanStep struct {
	Target    string   `json:"target"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// ValidatePlan rejects steps naming unknown targets or claiming dependencies
// the target graph does not contain (directly or transitively). Agents must
// reference graph facts, not invent them.
func (tg *TaskGraph) ValidatePlan(steps []PlanStep) error {
	var errs []error
	for _, s := range steps {
		if _, ok := tg.targets[s.Target]; !ok {
			errs = append(errs, fmt.Errorf("unknown target %q", s.Target))
			continue
		}
		reach := tg.closure(s.Target)
		for _, d := range s.DependsOn {
			if !reach[d] {
				errs = append(errs, fmt.Errorf("target %q does not depend on %q", s.Target, d))
			}
		}
	}
	return errors.Join(errs...)
}

func (tg *TaskGraph) closure(id string) map[string]bool {
	seen := map[string]bool{}
	stack := append([]string(nil), tg.targets[id].Deps...)
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !seen[cur] {
			seen[cur] = true
			stack = append(stack, tg.targets[cur].Deps...)
		}
	}
	return seen
}

// Targets returns all targets in dependency order.
func (tg *TaskGraph) Targets() []*Target {
	out := make([]*Target, 0, len(tg.order))
	for _, id := range tg.order {
		out = append(out, tg.targets[id])
	}
	return out
}
