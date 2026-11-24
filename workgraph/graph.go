package workgraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// NodeKind classifies a graph node.
type NodeKind string

const (
	// KindModule is a Go module in the workspace (one repository root).
	KindModule NodeKind = "module"
	// KindPackage is a Go package inside a workspace module.
	KindPackage NodeKind = "package"
	// KindExternal is a third-party module required by a workspace module.
	KindExternal NodeKind = "external"
	// KindTemplate is a scaffolding template directory.
	KindTemplate NodeKind = "template"
	// KindWorkload is a workload (app, cli, book, ...) and its fixtures.
	KindWorkload NodeKind = "workload"
)

// EdgeKind classifies why one node depends on another.
type EdgeKind string

const (
	// EdgeImport is a non-test Go import.
	EdgeImport EdgeKind = "import"
	// EdgeTestImport is an import that appears only in _test.go files. Such
	// edges may legally form cycles (external test packages), so cycle
	// validation ignores them.
	EdgeTestImport EdgeKind = "test-import"
	// EdgeMember links a package to its owning module (lock state).
	EdgeMember EdgeKind = "member"
	// EdgeRequire is a go.mod require between modules (version contract).
	EdgeRequire EdgeKind = "require"
	// EdgeUses links a workload or template to a package it relies on.
	EdgeUses EdgeKind = "uses"
	// EdgeTemplate links a workload to the template it is instantiated from.
	EdgeTemplate EdgeKind = "template"
)

// acyclic reports whether the architecture forbids cycles through edges of
// this kind.
func (k EdgeKind) acyclic() bool { return k != EdgeTestImport }

// Node is one vertex of the workspace graph. IDs are "<kind>:<name>".
type Node struct {
	ID      string   `json:"id"`
	Kind    NodeKind `json:"kind"`
	Name    string   `json:"name"`
	Module  string   `json:"module,omitempty"`
	Dir     string   `json:"dir,omitempty"`
	Version string   `json:"version,omitempty"`
	// Files are the slash-separated paths, relative to the workspace root,
	// whose content belongs to this node. They are the node's hash inputs.
	Files []string `json:"files,omitempty"`
}

// Provenance records where an edge was declared.
type Provenance struct {
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Edge states that From depends on To.
type Edge struct {
	From       string       `json:"from"`
	To         string       `json:"to"`
	Kind       EdgeKind     `json:"kind"`
	Provenance []Provenance `json:"provenance,omitempty"`
}

// NodeID builds the canonical node identifier.
func NodeID(kind NodeKind, name string) string { return string(kind) + ":" + name }

type edgeKey struct {
	from, to string
	kind     EdgeKind
}

// Graph is a directed dependency graph. The zero value is not usable; call
// NewGraph. A Graph is not safe for concurrent mutation.
type Graph struct {
	nodes map[string]*Node
	edges map[edgeKey]*Edge
	out   map[string]map[string]bool
	in    map[string]map[string]bool
}

// NewGraph returns an empty graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: map[string]*Node{},
		edges: map[edgeKey]*Edge{},
		out:   map[string]map[string]bool{},
		in:    map[string]map[string]bool{},
	}
}

// AddNode inserts n, or merges its files into an existing node with the same
// ID. It returns the stored node.
func (g *Graph) AddNode(n Node) *Node {
	if n.ID == "" {
		n.ID = NodeID(n.Kind, n.Name)
	}
	if cur, ok := g.nodes[n.ID]; ok {
		cur.Files = sortedUnique(append(cur.Files, n.Files...))
		return cur
	}
	n.Files = sortedUnique(n.Files)
	g.nodes[n.ID] = &n
	return &n
}

// AddEdge records that from depends on to. Both nodes must exist. Repeated
// edges merge their provenance.
func (g *Graph) AddEdge(from, to string, kind EdgeKind, p ...Provenance) error {
	if _, ok := g.nodes[from]; !ok {
		return fmt.Errorf("workgraph: edge from unknown node %q", from)
	}
	if _, ok := g.nodes[to]; !ok {
		return fmt.Errorf("workgraph: edge to unknown node %q", to)
	}
	if from == to {
		return nil
	}
	k := edgeKey{from, to, kind}
	e, ok := g.edges[k]
	if !ok {
		e = &Edge{From: from, To: to, Kind: kind}
		g.edges[k] = e
		addSet(g.out, from, to)
		addSet(g.in, to, from)
	}
	e.Provenance = append(e.Provenance, p...)
	sort.Slice(e.Provenance, func(i, j int) bool {
		a, b := e.Provenance[i], e.Provenance[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return nil
}

// Node returns the node with the given ID.
func (g *Graph) Node(id string) (*Node, bool) {
	n, ok := g.nodes[id]
	return n, ok
}

// Nodes returns all nodes sorted by ID.
func (g *Graph) Nodes() []*Node {
	out := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Edges returns all edges sorted by (from, to, kind).
func (g *Graph) Edges() []*Edge {
	out := make([]*Edge, 0, len(g.edges))
	for _, e := range g.edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return edgeLess(out[i], out[j]) })
	return out
}

func edgeLess(a, b *Edge) bool {
	if a.From != b.From {
		return a.From < b.From
	}
	if a.To != b.To {
		return a.To < b.To
	}
	return a.Kind < b.Kind
}

// EdgesBetween returns the edges from -> to (all kinds), sorted.
func (g *Graph) EdgesBetween(from, to string) []*Edge {
	var out []*Edge
	for _, e := range g.edges {
		if e.From == from && e.To == to {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}

// Dependencies returns the IDs id depends on, directly or transitively,
// sorted.
func (g *Graph) Dependencies(id string, transitive bool) []string {
	return g.walk(id, g.out, transitive)
}

// Dependents returns the IDs that depend on id, directly or transitively,
// sorted.
func (g *Graph) Dependents(id string, transitive bool) []string {
	return g.walk(id, g.in, transitive)
}

func (g *Graph) walk(id string, adj map[string]map[string]bool, transitive bool) []string {
	seen := map[string]bool{id: true}
	queue := []string{id}
	var out []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for next := range adj[cur] {
			if seen[next] {
				continue
			}
			seen[next] = true
			out = append(out, next)
			if transitive {
				queue = append(queue, next)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Path returns a shortest dependency chain from -> ... -> to as edges, or nil
// when to is not reachable. Ties break lexically, so the answer is stable.
func (g *Graph) Path(from, to string) []*Edge {
	prev := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 && prev[to] == "" && from != to {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range sortedKeys(g.out[cur]) {
			if _, ok := prev[next]; ok {
				continue
			}
			prev[next] = cur
			queue = append(queue, next)
		}
	}
	if _, ok := prev[to]; !ok || from == to {
		return nil
	}
	var chain []*Edge
	for cur := to; cur != from; cur = prev[cur] {
		chain = append([]*Edge{g.EdgesBetween(prev[cur], cur)[0]}, chain...)
	}
	return chain
}

// Digest is a stable SHA-256 over the canonical node and edge sets.
func (g *Graph) Digest() string {
	data, _ := json.Marshal(struct {
		Nodes []*Node `json:"nodes"`
		Edges []*Edge `json:"edges"`
	}{g.Nodes(), g.Edges()})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Ref summarizes the graph. Its JSON shape matches the toolchain contract's
// GraphRef (nodes, edges, digest, source) so a ProjectContext can pin it.
type Ref struct {
	Nodes  int    `json:"nodes"`
	Edges  int    `json:"edges"`
	Digest string `json:"digest"`
	Source string `json:"source"`
}

// RefSource names this engine as a GraphRef producer.
const RefSource = "workgraph"

// Ref returns the graph summary.
func (g *Graph) Ref() Ref {
	return Ref{Nodes: len(g.nodes), Edges: len(g.edges), Digest: g.Digest(), Source: RefSource}
}

func addSet(m map[string]map[string]bool, k, v string) {
	if m[k] == nil {
		m[k] = map[string]bool{}
	}
	m[k][v] = true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedUnique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	s := append([]string(nil), in...)
	sort.Strings(s)
	out := s[:1]
	for _, v := range s[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}

// shortID trims the kind prefix for human output.
func shortID(id string) string {
	if _, after, ok := strings.Cut(id, ":"); ok {
		return after
	}
	return id
}
