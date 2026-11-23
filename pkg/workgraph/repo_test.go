package workgraph

import (
	"slices"
	"testing"
)

// TestRepositoryGraph discovers this repository with its own manifest: the
// graph must be acyclic, and core invariants must be visible as graph facts.
func TestRepositoryGraph(t *testing.T) {
	ws, err := Discover("../..", nil)
	if err != nil {
		t.Fatal(err)
	}
	g := ws.Graph
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	const mod = "github.com/krewire/krewire"
	errs := NodeID(KindPackage, mod+"/pkg/errs")
	for _, dep := range g.Dependencies(NodeID(KindPackage, mod+"/pkg/model"), true) {
		if n, _ := g.Node(dep); n.Kind == KindPackage && dep != errs {
			t.Errorf("pkg/model transitively depends on %s", dep)
		}
	}
	a := ws.Affected([]string{"pkg/errs/error.go"}, AffectedOptions{})
	for _, want := range []string{NodeID(KindPackage, mod), NodeID(KindPackage, mod+"/pkg/model"), "workload:cli"} {
		if _, ok := a.Cause(want); !ok {
			t.Errorf("pkg/errs change does not affect %s", want)
		}
	}
	docs := ws.Affected([]string{"docs/index.md"}, AffectedOptions{})
	if len(docs.Nodes) != 0 || !slices.Equal(docs.Unowned, []string{"docs/index.md"}) {
		t.Errorf("docs change affected %v", docs.Nodes)
	}
	if _, err := BuildTasks(g, DefaultTasks()); err != nil {
		t.Fatal(err)
	}
}
