package workgraph

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// APIVersion versions every JSON document this package emits. Consumers must
// reject documents with an unknown major version.
const APIVersion = "workgraph.krewire.dev/v1"

// Document kinds.
const (
	KindGraphDocument    = "WorkspaceGraph"
	KindAffectedDocument = "AffectedSet"
	KindReportDocument   = "TaskReport"
)

// GraphDocument is the machine-readable graph.
type GraphDocument struct {
	APIVersion string  `json:"api_version"`
	Kind       string  `json:"kind"`
	Ref        Ref     `json:"ref"`
	Nodes      []*Node `json:"nodes"`
	Edges      []*Edge `json:"edges"`
}

// Document returns the canonical JSON-ready view of the graph.
func (g *Graph) Document() GraphDocument {
	return GraphDocument{
		APIVersion: APIVersion, Kind: KindGraphDocument,
		Ref: g.Ref(), Nodes: g.Nodes(), Edges: g.Edges(),
	}
}

// WriteJSON writes the graph document as indented, deterministic JSON.
func (g *Graph) WriteJSON(w io.Writer) error { return writeJSON(w, g.Document()) }

// WriteText writes a human-readable listing: one block per node with its
// direct dependencies and the first provenance of each edge.
func (g *Graph) WriteText(w io.Writer) error {
	ref := g.Ref()
	var b strings.Builder
	fmt.Fprintf(&b, "workspace graph: %d nodes, %d edges, %s\n", ref.Nodes, ref.Edges, ref.Digest)
	for _, n := range g.Nodes() {
		fmt.Fprintf(&b, "\n%s %s", n.Kind, n.Name)
		if n.Version != "" {
			fmt.Fprintf(&b, " %s", n.Version)
		}
		if n.Dir != "" {
			fmt.Fprintf(&b, " (%s)", n.Dir)
		}
		b.WriteString("\n")
		for _, dep := range g.Dependencies(n.ID, false) {
			for _, e := range g.EdgesBetween(n.ID, dep) {
				fmt.Fprintf(&b, "  -> %-10s %s%s\n", e.Kind, dep, provSuffix(e))
			}
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func provSuffix(e *Edge) string {
	if len(e.Provenance) == 0 {
		return ""
	}
	p := e.Provenance[0]
	s := "  [" + p.File
	if p.Line > 0 {
		s += fmt.Sprintf(":%d", p.Line)
	}
	if n := len(e.Provenance) - 1; n > 0 {
		s += fmt.Sprintf(" +%d", n)
	}
	return s + "]"
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
