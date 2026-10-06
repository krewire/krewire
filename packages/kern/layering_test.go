// Tests for the workspace layering guards.
//
// The dependency rule is that an import may only ever point downward. Nothing
// enforced that for years — which is how web/ssg ended up importing
// github.com/krewire/kiw/dsl. The web's go.mod never declared kiw at all; the
// build only succeeded because go.work papered over the missing requirement.
//
// These guards derive everything they need from disk: go.work says which
// modules exist, and each go.mod declares the edges between them. Nothing here
// lists a module by name, so adding a module to the ecosystem never requires
// editing this file — or the kernel at all.
package kern

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

// krewirePrefix is the import prefix every ecosystem module shares.
const krewirePrefix = "github.com/krewire/"

// workspaceRoot walks up from this package to the directory holding go.work,
// which is the root of the multi-module workspace.
func workspaceRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.work found above %s: the guards need the workspace root", dir)
		}
		dir = parent
	}
}

// moduleDirs reads the workspace layout from go.work and maps each module name
// (the last path segment of its module path) to its workspace-relative
// directory. go.work is the single source of truth for where modules live.
func moduleDirs(t *testing.T) map[string]string {
	t.Helper()
	root := workspaceRoot(t)
	work, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("read go.work: %v", err)
	}
	dirs := map[string]string{}
	for _, line := range strings.Split(string(work), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "./") {
			continue
		}
		rel := strings.TrimSuffix(strings.TrimPrefix(line, "./"), "/")
		gomod, gerr := os.ReadFile(filepath.Join(root, rel, "go.mod"))
		if gerr != nil {
			continue
		}
		for _, mod := range strings.Split(string(gomod), "\n") {
			mod = strings.TrimSpace(mod)
			if after, ok := strings.CutPrefix(mod, "module "); ok {
				name := strings.TrimPrefix(strings.TrimSpace(after), krewirePrefix)
				dirs[name] = rel
				break
			}
		}
	}
	if len(dirs) == 0 {
		t.Skip("no multi-module entries found in go.work: skipping multi-repo layering guards in monorepo layout")
	}
	return dirs
}

// krewireRequires returns the ecosystem modules a go.mod requires, parsed from
// the require directives themselves. This is what makes the layering derived:
// a module's position in the stack comes from the edges it declares, not from a
// list the kernel maintains.
func krewireRequires(t *testing.T, gomodPath string) []string {
	t.Helper()
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		t.Fatalf("read %s: %v", gomodPath, err)
	}
	var out []string
	inBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		switch {
		case line == "require (":
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "require "):
			line = strings.TrimPrefix(line, "require ")
		case !inBlock:
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if name := strings.TrimPrefix(fields[0], krewirePrefix); name != fields[0] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// layers derives each module's position from the dependency graph: a module
// with no ecosystem dependencies sits at 0, and every other module sits one
// level above the deepest thing it requires. Only the relative comparison
// matters, never an exact index, so ties are harmless.
func layers(t *testing.T, dirs map[string]string) map[string]int {
	t.Helper()
	root := workspaceRoot(t)
	deps := make(map[string][]string, len(dirs))
	for name, rel := range dirs {
		deps[name] = krewireRequires(t, filepath.Join(root, rel, "go.mod"))
	}

	depth := map[string]int{}
	visiting := map[string]bool{}
	var depthOf func(string) int
	depthOf = func(name string) int {
		if d, ok := depth[name]; ok {
			return d
		}
		if visiting[name] {
			t.Fatalf("dependency cycle through %q: the workspace layering cannot be derived", name)
		}
		visiting[name] = true
		best := 0
		for _, dep := range deps[name] {
			if _, known := dirs[dep]; !known {
				continue
			}
			if d := depthOf(dep) + 1; d > best {
				best = d
			}
		}
		delete(visiting, name)
		depth[name] = best
		return best
	}
	for name := range dirs {
		depthOf(name)
	}
	return depth
}

// goFiles walks a module directory and returns its non-test Go files.
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

// TestKWL_LAYER_001_DownwardImportsOnly asserts that no module imports a module
// above it in the derived layering. The layering comes from each go.mod's own
// require directives, so a new module is placed automatically and this guard
// needs no edit to keep working.
func TestKWL_LAYER_001_DownwardImportsOnly(t *testing.T) {
	root := workspaceRoot(t)
	fset := token.NewFileSet()
	dirs := moduleDirs(t)
	depth := layers(t, dirs)

	names := make([]string, 0, len(dirs))
	for name := range dirs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, owner := range names {
		for _, path := range goFiles(t, filepath.Join(root, dirs[owner])) {
			f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if perr != nil {
				t.Errorf("parse %s: %v", path, perr)
				continue
			}
			rel, _ := filepath.Rel(root, path)
			for _, imp := range f.Imports {
				p, uerr := strconv.Unquote(imp.Path.Value)
				if uerr != nil || !strings.HasPrefix(p, krewirePrefix) {
					continue
				}
				target := strings.SplitN(strings.TrimPrefix(p, krewirePrefix), "/", 2)[0]
				if _, known := dirs[target]; !known || target == owner {
					continue
				}
				if depth[target] > depth[owner] {
					t.Errorf("%s imports %q: %s sits above %s (layer %d > %d), and imports may only point downward (KWL-LAYER-001)",
						rel, p, target, owner, depth[target], depth[owner])
				}
			}
		}
	}
}

// TestKWL_LAYER_002_KernIsTheFloor asserts the kernel really is the bottom of
// the stack: it requires no ecosystem module, so nothing can sit beneath it.
func TestKWL_LAYER_002_KernIsTheFloor(t *testing.T) {
	dirs := moduleDirs(t)
	depth := layers(t, dirs)

	if _, known := dirs["kern"]; !known {
		t.Fatal("go.work does not include the kern module: the guards need the workspace layout")
	}
	if got := depth["kern"]; got != 0 {
		t.Errorf("kern sits at layer %d, want 0: the kernel must require no ecosystem module", got)
	}
}

// TestKERN_LAYER_001_NoEcosystemNames is the guard that keeps the kernel
// scalable.
//
// The rule it enforces: anything that requires editing the kernel when a module
// is added to the ecosystem is a sign the kernel is not scalable. So the kernel
// must name no module at all — not in a constant, not in a table, not in a
// lookup.
//
// The roster it checks against is derived from go.work at run time rather than
// written in source, so the ecosystem can gain a module without anyone editing
// a line of kernel code.
func TestKERN_LAYER_001_NoEcosystemNames(t *testing.T) {
	root := workspaceRoot(t)
	fset := token.NewFileSet()
	dirs := moduleDirs(t)

	self, known := dirs["kern"]
	if !known {
		t.Fatal("go.work does not include the kern module: the guard needs the workspace layout")
	}

	// The invariant is stated in terms of import paths, not bare names. In Go a
	// bare name like "app", "testing", or "runtime" is legitimately something
	// else here — a Kind value, an environment, the standard library — so
	// matching names would flag the kernel's own vocabulary. An import path is
	// unambiguous: nothing in the kernel may reach for another module.
	bannedPaths := map[string]bool{}
	for name := range dirs {
		if name == "kern" {
			continue
		}
		bannedPaths[krewirePrefix+name] = true
	}

	for _, path := range goFiles(t, filepath.Join(root, self)) {
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Errorf("parse %s: %v", path, perr)
			continue
		}
		rel, _ := filepath.Rel(root, path)
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}
			for banned := range bannedPaths {
				if value == banned || strings.HasPrefix(value, banned+"/") {
					t.Errorf("%s: the kernel references %q — module %q. The kernel must depend on no "+
						"ecosystem module, or adding one would force an edit here (KERN-LAYER-001)",
						rel, value, strings.TrimPrefix(banned, krewirePrefix))
				}
			}
			return true
		})
	}
}
