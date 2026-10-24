package commerce_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestCommerceDependencies asserts that packages/commerce adheres strictly to
// the pyramid rule: it may only depend on standard library packages and Level 1
// foundational packages (packages/errs, packages/model).
func TestCommerceDependencies(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		// exclude test files from package dependency checking
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}

	for pkgName, pkg := range pkgs {
		for filename, file := range pkg.Files {
			for _, imp := range file.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", imp.Path.Value, err)
				}

				if strings.HasPrefix(path, "github.com/krewire/krewire/") {
					sub := strings.TrimPrefix(path, "github.com/krewire/krewire/")
					// Allowed: packages/errs, packages/model
					if sub != "pkg/errs" && sub != "pkg/model" {
						t.Errorf("%s (%s): illegal upward dependency on %s (commerce may only depend on errs and model)",
							filepath.Base(filename), pkgName, path)
					}
				}
			}
		}
	}
}
