package workgraph

import (
	"bufio"
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Module is a discovered workspace module.
type Module struct {
	Dir  string   `json:"dir"`
	File *ModFile `json:"go_mod"`
}

// Workspace is a discovered workspace: its modules, manifest, and graph.
type Workspace struct {
	// Root is the absolute workspace root. It never appears in JSON output,
	// so graphs and cache keys are portable across checkouts.
	Root     string
	Manifest *Manifest
	Modules  []*Module
	Graph    *Graph
	// Files lists every regular file seen, relative to Root, sorted.
	Files []string

	owners    map[string][]string // file -> owning node IDs
	dirOwners map[string][]string // package dir -> node ID
	trees     map[string][]string // dir prefix -> template/workload node IDs
}

type goImport struct {
	path string
	file string
	line int
	test bool
}

// Discover builds the workspace graph rooted at root. A nil manifest means
// "load krewire.workspace.yaml if present".
func Discover(root string, m *Manifest) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if m == nil {
		if m, err = LoadManifest(abs); err != nil {
			return nil, err
		}
	}
	ws := &Workspace{Root: abs, Manifest: m, Graph: NewGraph(),
		owners: map[string][]string{}, dirOwners: map[string][]string{}, trees: map[string][]string{}}
	if err := ws.loadModules(); err != nil {
		return nil, err
	}
	imports := map[string][]goImport{}
	for _, mod := range ws.Modules {
		if err := ws.walkModule(mod, imports); err != nil {
			return nil, err
		}
	}
	if err := ws.linkImports(imports); err != nil {
		return nil, err
	}
	if err := ws.addTemplatesAndWorkloads(); err != nil {
		return nil, err
	}
	if err := ws.addExtraInputs(); err != nil {
		return nil, err
	}
	ws.Files = sortedUnique(ws.Files)
	for _, n := range ws.Graph.Nodes() {
		for _, f := range n.Files {
			ws.owners[f] = append(ws.owners[f], n.ID)
		}
	}
	return ws, nil
}

func (ws *Workspace) loadModules() error {
	var dirs []string
	if data, err := os.ReadFile(filepath.Join(ws.Root, "go.work")); err == nil {
		dirs = ParseWorkFile(data)
	}
	dirs = append(dirs, ws.Manifest.Modules...)
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	seen := map[string]bool{}
	for _, d := range dirs {
		d = path.Clean(filepath.ToSlash(d))
		if seen[d] {
			continue
		}
		seen[d] = true
		mf, err := readModFile(filepath.Join(ws.Root, filepath.FromSlash(d)))
		if err != nil {
			return fmt.Errorf("workgraph: module %q: %w", d, err)
		}
		ws.Modules = append(ws.Modules, &Module{Dir: d, File: mf})
		files := []string{join(d, "go.mod")}
		if _, err := os.Stat(filepath.Join(ws.Root, filepath.FromSlash(d), "go.sum")); err == nil {
			files = append(files, join(d, "go.sum"))
		}
		ws.Graph.AddNode(Node{Kind: KindModule, Name: mf.Module, Dir: d, Version: mf.Go, Files: files})
	}
	sort.Slice(ws.Modules, func(i, j int) bool { return ws.Modules[i].File.Module < ws.Modules[j].File.Module })
	for _, mod := range ws.Modules {
		from := NodeID(KindModule, mod.File.Module)
		for _, r := range mod.File.Requires {
			p := Provenance{File: join(mod.Dir, "go.mod"), Line: r.Line, Detail: r.Path + " " + r.Version}
			to := NodeID(KindModule, r.Path)
			if _, ok := ws.Graph.Node(to); !ok {
				to = ws.Graph.AddNode(Node{Kind: KindExternal, Name: r.Path, Version: r.Version}).ID
			}
			if err := ws.Graph.AddEdge(from, to, EdgeRequire, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// join is path.Join that keeps "." roots out of the result.
func join(dir, name string) string {
	if dir == "." || dir == "" {
		return name
	}
	return path.Join(dir, name)
}

// skipDir reports directories the Go tool ignores for packages; their files
// are still listed so they can be owned as testdata, embeds, or inputs.
func (ws *Workspace) skipDir(rel, name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" {
		return true
	}
	return matchAny(ws.Manifest.Exclude, rel)
}

func (ws *Workspace) walkModule(mod *Module, imports map[string][]goImport) error {
	base := filepath.Join(ws.Root, filepath.FromSlash(mod.Dir))
	return filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relMod, _ := filepath.Rel(base, p)
		relMod = filepath.ToSlash(relMod)
		rel := join(mod.Dir, relMod)
		if !d.IsDir() {
			if d.Type().IsRegular() {
				ws.Files = append(ws.Files, rel)
			}
			return nil
		}
		if relMod != "." {
			switch d.Name() {
			case ".git", ".krewire", "node_modules", "vendor":
				return filepath.SkipDir // never inputs, potentially huge
			}
			if ws.skipDir(rel, d.Name()) {
				return ws.collectTree(p, rel)
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir // nested module: discovered only if listed
			}
		}
		return ws.addPackage(mod, p, rel, relMod, imports)
	})
}

// collectTree records the files of a skipped directory so they can still be
// owned (testdata, embeds, extra inputs) and returns SkipDir.
func (ws *Workspace) collectTree(abs, rel string) error {
	_ = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			r, _ := filepath.Rel(abs, p)
			ws.Files = append(ws.Files, join(rel, filepath.ToSlash(r)))
		}
		return nil
	})
	return filepath.SkipDir
}

func (ws *Workspace) addPackage(mod *Module, abs, rel, relMod string, imports map[string][]goImport) error {
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	var files, embeds []string
	var imps []goImport
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f := join(rel, e.Name())
		files = append(files, f)
		got, emb, err := parseGoFile(filepath.Join(abs, e.Name()), f)
		if err != nil {
			return err
		}
		imps = append(imps, got...)
		embeds = append(embeds, emb...)
	}
	if len(files) == 0 {
		return nil
	}
	importPath := mod.File.Module
	if relMod != "." {
		importPath += "/" + relMod
	}
	id := NodeID(KindPackage, importPath)
	ws.Graph.AddNode(Node{Kind: KindPackage, Name: importPath, Module: mod.File.Module, Dir: rel, Files: files})
	ws.dirOwners[rel] = append(ws.dirOwners[rel], id)
	// testdata and //go:embed trees are package inputs; resolve them after
	// the walk has listed every file.
	ws.trees[join(rel, "testdata")] = append(ws.trees[join(rel, "testdata")], id)
	for _, pat := range embeds {
		ws.trees[join(rel, pat)] = append(ws.trees[join(rel, pat)], id)
	}
	imports[id] = imps
	return ws.Graph.AddEdge(id, NodeID(KindModule, mod.File.Module), EdgeMember)
}

// parseGoFile returns the imports and //go:embed patterns of one file. All
// files are read regardless of build constraints: the graph is a superset.
func parseGoFile(abs, rel string) ([]goImport, []string, error) {
	src, err := os.ReadFile(abs)
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, src, parser.ImportsOnly)
	if err != nil {
		return nil, nil, fmt.Errorf("workgraph: parse %s: %w", rel, err)
	}
	test := strings.HasSuffix(rel, "_test.go")
	var out []goImport
	for _, s := range f.Imports {
		p, err := strconv.Unquote(s.Path.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("workgraph: %s: %w", rel, err)
		}
		out = append(out, goImport{path: p, file: rel, line: fset.Position(s.Pos()).Line, test: test})
	}
	var embeds []string
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		if after, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "//go:embed "); ok {
			for _, pat := range strings.Fields(after) {
				embeds = append(embeds, strings.TrimPrefix(strings.Trim(pat, "`\""), "all:"))
			}
		}
	}
	return out, embeds, nil
}

// readRel reads a file addressed relative to the workspace root.
func readRel(root, rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
}
