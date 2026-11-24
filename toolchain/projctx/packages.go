package projctx

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/krewire/krewire/toolchain/contract"
)

// skipDir reports directories that never hold first-party packages.
func skipDir(name string) bool {
	switch name {
	case "testdata", "vendor", "node_modules", "dist":
		return true
	}
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// walk visits repository files in lexical order, skipping nested modules.
func (g *gen) walk(fn func(rel string, d fs.DirEntry)) {
	_ = filepath.WalkDir(g.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(g.root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() && rel != "." {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir
			}
		}
		fn(rel, d)
		return nil
	})
}

// packages inventories Go packages and their module-internal imports.
func (g *gen) packages() {
	mod := g.ctx.Module.Path
	type pkgAcc struct {
		pkg     contract.Package
		imports map[string]bool
	}
	byDir := map[string]*pkgAcc{}
	g.walk(func(rel string, d fs.DirEntry) {
		if d.IsDir() || !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return
		}
		data, _ := g.read(rel)
		f, err := parser.ParseFile(token.NewFileSet(), rel, data, parser.ImportsOnly)
		if err != nil {
			g.diag("KWC4004", "cannot parse "+rel+": "+err.Error(), prov(rel, "", 0, "go-source"), "fix the syntax error")
			return
		}
		dir := path.Dir(rel)
		acc := byDir[dir]
		if acc == nil {
			ip := mod
			if dir != "." {
				ip = mod + "/" + dir
			}
			acc = &pkgAcc{pkg: contract.Package{ImportPath: ip, Dir: dir, Name: f.Name.Name,
				Provenance: prov(rel, "package "+f.Name.Name, 0, "go-source")}, imports: map[string]bool{}}
			byDir[dir] = acc
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if mod != "" && (p == mod || strings.HasPrefix(p, mod+"/")) {
				acc.imports[p] = true
			}
		}
	})
	var edges []string
	for _, acc := range byDir {
		for imp := range acc.imports {
			acc.pkg.Imports = append(acc.pkg.Imports, imp)
			edges = append(edges, acc.pkg.ImportPath+" -> "+imp)
		}
		sort.Strings(acc.pkg.Imports)
		g.ctx.Packages = append(g.ctx.Packages, acc.pkg)
	}
	sort.Strings(edges)
	sum := sha256.Sum256([]byte(strings.Join(edges, "\n")))
	g.ctx.Graph = contract.GraphRef{Nodes: len(byDir), Edges: len(edges), Digest: hex.EncodeToString(sum[:]), Source: "go-imports"}
}

// guardRules turns documented Test functions in root *_test.go files — the
// executable architecture guards — into rules enforced by `go test -run`.
func (g *gen) guardRules() {
	g.walk(func(rel string, d fs.DirEntry) {
		if d.IsDir() || strings.Contains(rel, "/") || !strings.HasSuffix(rel, "_test.go") {
			return
		}
		data, _ := g.read(rel)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, rel, data, parser.ParseComments)
		if err != nil {
			g.diag("KWC4004", "cannot parse "+rel+": "+err.Error(), prov(rel, "", 0, "go-source"), "fix the syntax error")
			return
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Doc == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			g.ctx.Rules = append(g.ctx.Rules, contract.Rule{
				ID: fn.Name.Name, Category: "architecture", Statement: firstSentence(fn.Doc.Text()),
				EnforcedBy: "go test -run ^" + fn.Name.Name + "$ .",
				Provenance: prov(rel, fn.Name.Name, fset.Position(fn.Pos()).Line, "go-source")})
		}
	})
}

// firstSentence returns the first sentence of a doc comment on one line.
func firstSentence(doc string) string {
	doc = strings.Join(strings.Fields(doc), " ")
	if i := strings.Index(doc, ". "); i >= 0 {
		return doc[:i+1]
	}
	return doc
}

var adrName = regexp.MustCompile(`^ADR-[A-Z0-9]+`)

// adrs records architecture decisions in docs/adr as decision rules.
func (g *gen) adrs() {
	g.walk(func(rel string, d fs.DirEntry) {
		base := path.Base(rel)
		if d.IsDir() || path.Dir(rel) != "docs/adr" || !adrName.MatchString(base) || !strings.HasSuffix(base, ".md") {
			return
		}
		data, _ := g.read(rel)
		title := strings.TrimSuffix(base, ".md")
		line := 0
		for i, l := range lines(data) {
			if strings.HasPrefix(l, "# ") {
				title, line = strings.TrimSpace(strings.TrimPrefix(l, "# ")), i+1
				break
			}
		}
		g.ctx.Rules = append(g.ctx.Rules, contract.Rule{ID: adrName.FindString(base), Category: "decision",
			Statement: title, Provenance: prov(rel, "title", line, "markdown")})
	})
}

// workload classifies the repository: an explicit krewire.yaml
// `project.kind` wins; otherwise a root `package main` means app, else
// library. Maturity follows the module version.
func (g *gen) workload() {
	w := contract.Workload{Kind: "library", Maturity: "unknown", Provenance: prov("go.mod", "module", 0, "go.mod")}
	for _, p := range g.ctx.Packages {
		if p.Dir == "." && p.Name == "main" {
			w.Kind, w.Provenance = "app", p.Provenance
		}
	}
	for _, c := range g.ctx.Config {
		if c.Key == "project.kind" && c.Value != "" {
			w.Kind, w.Provenance = c.Value, c.Provenance
		}
	}
	if v := g.ctx.Module.Version; v != "" {
		w.Maturity = "preview"
		if !strings.HasPrefix(v, "0.") {
			w.Maturity = "stable"
		}
	}
	g.ctx.Workload = w
}

// kiwCommand matches a command-line use of the retired `kiw` executable,
// not the `.kiw` component file extension.
var kiwCommand = regexp.MustCompile("(^|[\\s`'\"(])kiw\\s+[a-z]")

// staleDocs flags maintained Markdown that still drives the retired `kiw`.
func (g *gen) staleDocs() {
	g.walk(func(rel string, d fs.DirEntry) {
		top := strings.SplitN(rel, "/", 2)[0]
		if d.IsDir() || !strings.HasSuffix(rel, ".md") || (top != "docs" && strings.Contains(rel, "/")) {
			return
		}
		data, _ := g.read(rel)
		first, count := 0, 0
		for i, l := range lines(data) {
			if kiwCommand.MatchString(l) {
				count++
				if first == 0 {
					first = i + 1
				}
			}
		}
		if count > 0 {
			g.diag("KWC3001", strconv.Itoa(count)+" line(s) invoke the retired `kiw` executable", prov(rel, "kiw", first, "markdown"),
				"replace `kiw <cmd>` with `krew <cmd>` (ADR-T1C4A)")
		}
	})
}
