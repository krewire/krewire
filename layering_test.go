// Tests for the ecosystem layering guards.
//
// krewire is a single Go module, so the layering can no longer be derived from
// go.work module entries: there is exactly one module. The rules that still
// carry weight are therefore enforced directly against the packages on disk,
// and every guard reads the module path from go.mod instead of hardcoding it.
//
// The invariants themselves are documented in
// docs/foundation/architecture.md §3.3 and AGENTS.md.
package krewire_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// foundationRank states the position of each foundation package in the
// dependency ladder: a foundation package may import the standard library and
// any foundation package ranked below it, never one ranked at or above it.
// This mirrors the Layer 1 / Layer 2 rows of the architecture table; `log`
// sits above `env` because it is the only foundation package that imports a
// sibling foundation package.
var foundationRank = map[string]int{
	"errs":      0,
	"model":     1,
	"version":   1,
	"env":       2,
	"lifecycle": 2,
	"log":       3,
}

// repoRoot walks up from the current package to the directory holding go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above %s: the guards need the module root", dir)
		}
		dir = parent
	}
}

// modulePath reads the module path from the repository's go.mod, which is the
// single source of truth for the import prefix every guard matches against.
func modulePath(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, raw := range strings.Split(string(data), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(raw), "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	t.Fatal("go.mod declares no module path")
	return ""
}

// goFiles returns the non-test Go files under dir, skipping generated and
// vendored trees.
func goFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "testdata", ".krewire", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	sort.Strings(out)
	return out
}

// parseImports parses one file and returns its import paths.
func parseImports(t *testing.T, path string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("unquote import in %s: %v", path, err)
		}
		out = append(out, p)
	}
	return out
}

// isStdlib reports whether an import path belongs to the standard library: the
// first path element of a stdlib import never contains a dot.
func isStdlib(path string) bool {
	first := path
	if i := strings.Index(path, "/"); i >= 0 {
		first = path[:i]
	}
	return !strings.Contains(first, ".")
}

// packageName maps an import path under the module to its package name, and
// reports whether the path is inside pkg/.
func packageName(mod, path string) (string, bool) {
	rest, ok := strings.CutPrefix(path, mod+"/pkg/")
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(rest, "/")
	return name, true
}

// TestFoundationLayering asserts that every foundation package imports only
// the standard library and foundation packages ranked below it. The rank comes
// from foundationRank; the import edges come from the files themselves, so a
// new dependency cannot slip in unnoticed.
func TestFoundationLayering(t *testing.T) {
	root := repoRoot(t)
	mod := modulePath(t, root)

	checked := 0
	for name, rank := range foundationRank {
		dir := filepath.Join(root, "pkg", name)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("foundation package %q has no directory: update foundationRank or restore the package", name)
		}
		for _, file := range goFiles(t, dir) {
			rel, _ := filepath.Rel(root, file)
			for _, imp := range parseImports(t, file) {
				if isStdlib(imp) {
					continue
				}
				dep, inPkg := packageName(mod, imp)
				if !inPkg {
					t.Errorf("%s imports %q: foundation packages may import only the standard library and lower foundation packages",
						rel, imp)
					continue
				}
				depRank, isFoundation := foundationRank[dep]
				if !isFoundation {
					t.Errorf("%s imports %q: %s is not a foundation package, so the dependency points upward",
						rel, imp, dep)
					continue
				}
				if depRank >= rank {
					t.Errorf("%s imports %q: %s sits at rank %d, not below %s at rank %d",
						rel, imp, dep, depRank, name, rank)
				}
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no foundation files checked; the layering guard did not run")
	}
}

// TestFoundationNamesNoHigherPackage asserts that foundation packages name no
// package above them, not even in a string literal. Naming a higher package in
// a registry, a default, or a diagnostic would couple the floor of the stack to
// the domain layer just as surely as an import would.
func TestFoundationNamesNoHigherPackage(t *testing.T) {
	root := repoRoot(t)
	mod := modulePath(t, root)
	prefix := mod + "/pkg/"

	checked := 0
	for name, rank := range foundationRank {
		for _, file := range goFiles(t, filepath.Join(root, "pkg", name)) {
			rel, _ := filepath.Rel(root, file)
			f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", rel, err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil || !strings.HasPrefix(value, prefix) {
					return true
				}
				dep := strings.TrimPrefix(value, prefix)
				if i := strings.Index(dep, "/"); i >= 0 {
					dep = dep[:i]
				}
				depRank, isFoundation := foundationRank[dep]
				if isFoundation && depRank < rank {
					return true
				}
				t.Errorf("%s: foundation package %q names %q — package %q sits at or above it in the layering",
					rel, name, value, dep)
				return true
			})
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no foundation files checked; the naming guard did not run")
	}
}

// TestPackagesNeverImportKiw asserts that no package under pkg/ imports the
// developer tool. Libraries are consumed by the CLI, never the other way
// around; logic both need belongs in a package, not in tools/kiw.
func TestPackagesNeverImportKiw(t *testing.T) {
	root := repoRoot(t)
	mod := modulePath(t, root)

	checked := 0
	for _, file := range goFiles(t, filepath.Join(root, "pkg")) {
		rel, _ := filepath.Rel(root, file)
		for _, imp := range parseImports(t, file) {
			if strings.HasPrefix(imp, mod+"/tools/") || strings.HasPrefix(imp, "github.com/krewire/kiw") {
				t.Errorf("%s imports %q: pkg/ may never import tools/kiw", rel, imp)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no pkg files checked; the tooling-isolation guard did not run")
	}
}

// TestUIDoesNotImportWeb asserts the one sibling direction the architecture
// pins explicitly: pkg/web may use pkg/ui, but pkg/ui must never reach for
// pkg/web.
func TestUIDoesNotImportWeb(t *testing.T) {
	root := repoRoot(t)
	mod := modulePath(t, root)
	web := mod + "/pkg/web"

	checked := 0
	for _, file := range goFiles(t, filepath.Join(root, "pkg", "ui")) {
		rel, _ := filepath.Rel(root, file)
		for _, imp := range parseImports(t, file) {
			if imp == web || strings.HasPrefix(imp, web+"/") {
				t.Errorf("%s imports %q: pkg/ui must not depend on pkg/web", rel, imp)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no pkg/ui files checked; the sibling-direction guard did not run")
	}
}
