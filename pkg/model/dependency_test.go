package model_test

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// packages/model must strictly depend only on standard library packages
// and its internal errs sibling. It must never import any other
// Krewire packages or external third-party dependencies.
func TestModel_StdlibBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read model dir: %v", err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", name, err)
			}
			if path == "github.com/krewire/krewire/pkg/errs" {
				continue
			}
			if !isStdlibImport(path) {
				t.Errorf("%s imports non-stdlib package %q: packages/model may depend only on standard library and packages/errs (REQ-L0-004)", name, path)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no non-test Go files checked; the stdlib boundary guard did not run")
	}
}

func isStdlibImport(path string) bool {
	first := path
	if i := strings.Index(path, "/"); i >= 0 {
		first = path[:i]
	}
	return !strings.Contains(first, ".")
}
