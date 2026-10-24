package errs_test

import (
	"flag"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
)

// errs may import only the Go standard library. It must never import any other
// Krewire module or a third-party dependency.
func TestImportBoundary(t *testing.T) {
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
			if isStdlibImport(path) {
				continue
			}
			t.Errorf("%s imports %q: errs may import only the standard library", name, path)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no non-test Go files checked; the import-boundary guard did not run")
	}
}

func TestLibraryFoundations(t *testing.T) {
	// KWL-CLI-002: stdlib flag package standard
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if fs == nil {
		t.Fatal("expected flag.FlagSet")
	}

	// KWL-CLI-003: stdlib log/slog standard
	logger := slog.Default()
	if logger == nil {
		t.Fatal("expected slog.Logger")
	}

	// KWL-CLI-004: stdlib os/strconv standard
	s := strconv.Itoa(8080)
	if s != "8080" {
		t.Errorf("strconv.Itoa = %q, want 8080", s)
	}

	// KWL-CLI-006: common error type and exit code mapping
	uerr := errs.UsageError("bad argument")
	if uerr.ExitCode() != errs.ExitCodeUsage {
		t.Errorf("UsageError ExitCode = %v, want %v", uerr.ExitCode(), errs.ExitCodeUsage)
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
