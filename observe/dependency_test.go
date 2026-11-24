package observe_test

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestImportBoundary keeps observe stdlib-first: besides the standard library
// it may import only pkg/log, so every workload can afford to depend on it.
func TestImportBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(p, "/")
			if !strings.Contains(first, ".") || p == "github.com/krewire/krewire/log" {
				continue
			}
			t.Errorf("%s imports %q: observe may import only the standard library and pkg/log", name, p)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no files checked")
	}
}
