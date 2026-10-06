// Tests for KWL-CORE-K1N2Q
package kern

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Spec: KWL-CORE-K1N2Q KWL-CORE-050 Scope: Unit
// core may import only the Go standard library and its own module's sibling
// packages (github.com/krewire/kern/*); it must never import any other Krewire
// module or a third-party dependency. This is parsed from the package's own
// source imports, so the guard is hermetic (no external tooling required).
func TestKWL_CORE_050_ImportBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
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
			if path == "github.com/krewire/krewire/packages/kern" || strings.HasPrefix(path, "github.com/krewire/krewire/packages/kern/") {
				continue
			}
			if isStdlibImport(path) {
				continue
			}
			t.Errorf("%s imports %q: core may import only the standard library and github.com/krewire/kern/* (KWL-CORE-050)", name, path)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no non-test Go files checked; the import-boundary guard did not run")
	}
}

// isStdlibImport reports whether an import path belongs to the standard
// library: the first path element of a stdlib import never contains a dot.
func isStdlibImport(path string) bool {
	first := path
	if i := strings.Index(path, "/"); i >= 0 {
		first = path[:i]
	}
	return !strings.Contains(first, ".")
}
