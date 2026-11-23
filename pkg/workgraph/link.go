package workgraph

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// inTree reports whether file lies in (or matches) the tree pattern key.
func inTree(key, file string) bool {
	return matchGlob(key, file) || matchGlob(key+"/**", file)
}

func (ws *Workspace) addFiles(id string, files ...string) {
	n, _ := ws.Graph.Node(id)
	n.Files = sortedUnique(append(n.Files, files...))
}

func (ws *Workspace) filesIn(key string) []string {
	var out []string
	for _, f := range ws.Files {
		if inTree(key, f) {
			out = append(out, f)
		}
	}
	return out
}

func (ws *Workspace) packagesByPath() map[string]string {
	idx := map[string]string{}
	for _, n := range ws.Graph.Nodes() {
		if n.Kind == KindPackage {
			idx[n.Name] = n.ID
		}
	}
	return idx
}

func (ws *Workspace) linkImports(imports map[string][]goImport) error {
	ws.Files = sortedUnique(ws.Files)
	for key, ids := range ws.trees { // testdata and embed trees
		for _, id := range ids {
			ws.addFiles(id, ws.filesIn(key)...)
		}
	}
	pkgs := ws.packagesByPath()
	var exts []string
	for _, n := range ws.Graph.Nodes() {
		if n.Kind == KindExternal {
			exts = append(exts, n.Name)
		}
	}
	sort.Slice(exts, func(i, j int) bool { return len(exts[i]) > len(exts[j]) })
	for _, from := range sortedKeysOf(imports) {
		for _, imp := range imports[from] {
			kind := EdgeImport
			if imp.test {
				kind = EdgeTestImport
			}
			p := Provenance{File: imp.file, Line: imp.line, Detail: imp.path}
			to, ok := pkgs[imp.path]
			if !ok {
				for _, e := range exts {
					if imp.path == e || strings.HasPrefix(imp.path, e+"/") {
						to, ok = NodeID(KindExternal, e), true
						break
					}
				}
			}
			if !ok {
				continue // standard library or unresolved; the compiler judges those
			}
			if err := ws.Graph.AddEdge(from, to, kind, p); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedKeysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// templateDirs returns every directory (of any listed file) matching pattern.
func (ws *Workspace) templateDirs(pattern string) []string {
	seen := map[string]bool{}
	for _, f := range ws.Files {
		for d := path.Dir(f); d != "." && d != "/" && d != ".."; d = path.Dir(d) {
			if !seen[d] && matchGlob(pattern, d) {
				seen[d] = true
			}
		}
	}
	return sortedKeysOf(seen)
}

func (ws *Workspace) addTemplatesAndWorkloads() error {
	pkgs := ws.packagesByPath()
	for _, pat := range ws.Manifest.Templates {
		for _, dir := range ws.templateDirs(path.Clean(pat)) {
			id := ws.Graph.AddNode(Node{Kind: KindTemplate, Name: dir, Dir: dir, Files: ws.filesIn(dir)}).ID
			ws.trees[dir] = append(ws.trees[dir], id)
			if err := ws.linkTemplate(id, pkgs); err != nil {
				return err
			}
		}
	}
	for _, w := range ws.Manifest.Workloads {
		n := Node{Kind: KindWorkload, Name: w.Name}
		for _, fx := range w.Fixtures {
			fx = path.Clean(fx)
			if n.Dir == "" {
				n.Dir = fx
			}
			n.Files = append(n.Files, ws.filesIn(fx)...)
		}
		id := ws.Graph.AddNode(n).ID
		for _, fx := range w.Fixtures {
			ws.trees[path.Clean(fx)] = append(ws.trees[path.Clean(fx)], id)
		}
		src := Provenance{File: ManifestFile, Detail: "workload " + w.Name}
		if w.Template != "" {
			tid := NodeID(KindTemplate, path.Clean(w.Template))
			if _, ok := ws.Graph.Node(tid); !ok {
				return fmt.Errorf("workgraph: workload %q: template %q is not a discovered template", w.Name, w.Template)
			}
			if err := ws.Graph.AddEdge(id, tid, EdgeTemplate, src); err != nil {
				return err
			}
		}
		for _, p := range w.Packages {
			pid, ok := pkgs[p]
			if !ok && len(ws.dirOwners[path.Clean(p)]) > 0 {
				pid, ok = ws.dirOwners[path.Clean(p)][0], true
			}
			if !ok {
				return fmt.Errorf("workgraph: workload %q: unknown package %q", w.Name, p)
			}
			if err := ws.Graph.AddEdge(id, pid, EdgeUses, src); err != nil {
				return err
			}
		}
	}
	return nil
}

// linkTemplate adds a uses edge for every workspace import path quoted in the
// template's Go sources (.go and .go.tmpl).
func (ws *Workspace) linkTemplate(id string, pkgs map[string]string) error {
	n, _ := ws.Graph.Node(id)
	for _, f := range n.Files {
		if !strings.HasSuffix(f, ".go") && !strings.HasSuffix(f, ".go.tmpl") {
			continue
		}
		data, err := readRel(ws.Root, f)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			parts := strings.Split(line, `"`)
			for j := 1; j < len(parts); j += 2 {
				if pid, ok := pkgs[parts[j]]; ok {
					p := Provenance{File: f, Line: i + 1, Detail: parts[j]}
					if err := ws.Graph.AddEdge(id, pid, EdgeUses, p); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (ws *Workspace) addExtraInputs() error {
	for _, dir := range sortedKeysOf(ws.Manifest.Inputs) {
		owners := ws.dirOwners[path.Clean(dir)]
		if len(owners) == 0 {
			owners = ws.trees[path.Clean(dir)]
		}
		if len(owners) == 0 {
			return fmt.Errorf("workgraph: inputs: %q is not a package, template, or fixture directory", dir)
		}
		var files []string
		for _, f := range ws.Files {
			if matchAny(ws.Manifest.Inputs[dir], f) {
				files = append(files, f)
			}
		}
		for _, id := range owners {
			ws.addFiles(id, files...)
		}
	}
	return nil
}

// Owners returns the IDs of the nodes that own file (a slash path relative to
// the root). Files that no longer exist (deletions) are attributed through
// their package directory or enclosing template/fixture tree.
func (ws *Workspace) Owners(file string) []string {
	file = path.Clean(file)
	if ids, ok := ws.owners[file]; ok {
		return sortedUnique(ids)
	}
	var ids []string
	if strings.HasSuffix(file, ".go") {
		ids = append(ids, ws.dirOwners[path.Dir(file)]...)
	}
	for key, owners := range ws.trees {
		if inTree(key, file) {
			ids = append(ids, owners...)
		}
	}
	return sortedUnique(ids)
}
