package workgraph

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func chain(t *testing.T) *Graph {
	t.Helper()
	g := NewGraph()
	for _, n := range []string{"a", "b", "c", "d"} {
		g.AddNode(Node{Kind: KindPackage, Name: n})
	}
	must(t, g.AddEdge("package:a", "package:b", EdgeImport, Provenance{File: "a/a.go", Line: 3}))
	must(t, g.AddEdge("package:b", "package:c", EdgeImport, Provenance{File: "b/b.go", Line: 4}))
	must(t, g.AddEdge("package:d", "package:c", EdgeTestImport, Provenance{File: "d/d_test.go", Line: 5}))
	return g
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGraphDependencies(t *testing.T) {
	g := chain(t)
	if got := g.Dependencies("package:a", false); !reflect.DeepEqual(got, []string{"package:b"}) {
		t.Fatalf("direct deps = %v", got)
	}
	if got := g.Dependencies("package:a", true); !reflect.DeepEqual(got, []string{"package:b", "package:c"}) {
		t.Fatalf("transitive deps = %v", got)
	}
	if got := g.Dependents("package:c", true); !reflect.DeepEqual(got, []string{"package:a", "package:b", "package:d"}) {
		t.Fatalf("transitive dependents = %v", got)
	}
	path := g.Path("package:a", "package:c")
	if len(path) != 2 || path[1].Provenance[0].File != "b/b.go" {
		t.Fatalf("path = %+v", path)
	}
	if g.Path("package:c", "package:a") != nil {
		t.Fatal("unexpected reverse path")
	}
}

func TestGraphUnknownNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{Kind: KindPackage, Name: "a"})
	if err := g.AddEdge("package:a", "package:zz", EdgeImport); err == nil {
		t.Fatal("edge to unknown node accepted")
	}
	if err := g.AddEdge("package:zz", "package:a", EdgeImport); err == nil {
		t.Fatal("edge from unknown node accepted")
	}
}

func TestGraphCycles(t *testing.T) {
	g := chain(t)
	// A test-import back edge is legal (external test packages).
	must(t, g.AddEdge("package:c", "package:a", EdgeTestImport))
	if err := g.Validate(); err != nil {
		t.Fatalf("test-import cycle rejected: %v", err)
	}
	must(t, g.AddEdge("package:c", "package:a", EdgeImport))
	err := g.Validate()
	var ce *CycleError
	if !errors.As(err, &ce) || !reflect.DeepEqual(ce.Cycles, [][]string{{"package:a", "package:b", "package:c"}}) {
		t.Fatalf("cycle not reported: %v", err)
	}
	if _, err := g.TopoOrder(); err == nil {
		t.Fatal("topo order on cyclic graph")
	}
}

func TestGraphTopoAndDeterminism(t *testing.T) {
	g := chain(t)
	order, err := g.TopoOrder()
	must(t, err)
	if !reflect.DeepEqual(order, []string{"package:c", "package:b", "package:a", "package:d"}) {
		t.Fatalf("order = %v", order)
	}
	var j1, j2 bytes.Buffer
	must(t, g.WriteJSON(&j1))
	must(t, chain(t).WriteJSON(&j2))
	if j1.String() != j2.String() || g.Digest() != chain(t).Digest() {
		t.Fatal("graph JSON is not deterministic")
	}
	var doc GraphDocument
	must(t, json.Unmarshal(j1.Bytes(), &doc))
	if doc.APIVersion != APIVersion || doc.Ref.Nodes != 4 || doc.Ref.Edges != 3 || doc.Ref.Source != RefSource {
		t.Fatalf("doc = %+v", doc.Ref)
	}
	var txt bytes.Buffer
	must(t, g.WriteText(&txt))
	if !strings.Contains(txt.String(), "-> import     package:b  [a/a.go:3]") {
		t.Fatalf("text output:\n%s", txt.String())
	}
}
