package projctx

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/version"
)

var goVersionRe = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)

// goMod reads module identity, language/toolchain versions, and requirements.
func (g *gen) goMod() {
	data, ok := g.read("go.mod")
	if !ok {
		g.diag("KWC1001", "go.mod not found at the repository root", prov("go.mod", "", 0, "go.mod"),
			"run `go mod init <module>`")
		return
	}
	inRequire := false
	for i, raw := range lines(data) {
		n := i + 1
		line, comment, _ := strings.Cut(raw, "//")
		f := strings.Fields(line)
		switch {
		case len(f) == 0:
			continue
		case inRequire && f[0] == ")":
			inRequire = false
		case inRequire || f[0] == "require":
			if f[0] == "require" {
				if len(f) == 2 && f[1] == "(" {
					inRequire = true
					continue
				}
				f = f[1:]
			}
			if len(f) != 2 {
				g.diag("KWC4001", "cannot parse require line: "+strings.TrimSpace(raw), prov("go.mod", "require", n, "go.mod"),
					"use `module/path vX.Y.Z`")
				continue
			}
			g.ctx.Dependencies = append(g.ctx.Dependencies, contract.Dependency{Path: f[0], Version: f[1],
				Indirect: strings.TrimSpace(comment) == "indirect", Provenance: prov("go.mod", "require "+f[0], n, "go.mod")})
		case f[0] == "module" && len(f) == 2:
			g.ctx.Module.Path = strings.Trim(f[1], `"`)
			g.ctx.Module.Provenance = prov("go.mod", "module", n, "go.mod")
		case f[0] == "go" && len(f) == 2:
			if !goVersionRe.MatchString(f[1]) {
				g.diag("KWC4001", "go directive "+f[1]+" is not a Go version", prov("go.mod", "go", n, "go.mod"), "use e.g. `go 1.27.1`")
				continue
			}
			g.ctx.Compatibility.GoVersion = f[1]
			g.ctx.Compatibility.Provenance = prov("go.mod", "go", n, "go.mod")
		case f[0] == "toolchain" && len(f) == 2:
			g.ctx.Compatibility.Toolchain = f[1]
		}
	}
	if g.ctx.Module.Path == "" || g.ctx.Compatibility.GoVersion == "" {
		g.diag("KWC1002", "go.mod lacks a module path or go directive", prov("go.mod", "", 0, "go.mod"),
			"declare both `module` and `go`")
	}
}

// versionDecl reads `Version` and `Tier` from the root package's version.go.
func (g *gen) versionDecl() {
	const rel = "version.go"
	data, ok := g.read(rel)
	if !ok {
		g.diag("KWC1005", "no version.go at the repository root", prov(rel, "Version", 0, "go-source"),
			"declare `var Version = version.MustParseVersion(\"x.y.z\")`")
		return
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, data, 0)
	if err != nil {
		g.diag("KWC4004", "cannot parse version.go: "+err.Error(), prov(rel, "", 0, "go-source"), "fix the syntax error")
		return
	}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				continue
			}
			line := fset.Position(vs.Pos()).Line
			switch vs.Names[0].Name {
			case "Version":
				if s, ok := stringArg(vs.Values[0]); ok {
					if _, err := version.ParseVersion(s); err != nil {
						g.diag("KWC4003", "Version "+strconv.Quote(s)+" is not SemVer", prov(rel, "Version", line, "go-source"),
							"use MAJOR.MINOR.PATCH")
						continue
					}
					g.ctx.Module.Version = s
					g.ctx.Module.Provenance = prov(rel, "Version", line, "go-source")
				}
			case "Tier":
				if sel, ok := vs.Values[0].(*ast.SelectorExpr); ok {
					g.ctx.Module.Tier = strings.ToLower(strings.TrimPrefix(sel.Sel.Name, "Tier"))
				}
			}
		}
	}
	if g.ctx.Module.Version == "" && !g.has("KWC4003") {
		g.diag("KWC1005", "version.go declares no Version", prov(rel, "Version", 0, "go-source"),
			"declare `var Version = version.MustParseVersion(\"x.y.z\")`")
	}
}

// stringArg extracts a string literal from `"x"` or `f("x")`.
func stringArg(e ast.Expr) (string, bool) {
	if call, ok := e.(*ast.CallExpr); ok && len(call.Args) == 1 {
		e = call.Args[0]
	}
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}

func (g *gen) has(code string) bool {
	for _, d := range g.ctx.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

// secretKey matches configuration keys whose values are never emitted.
var secretKey = regexp.MustCompile(`(?i)(secret|passw(or)?d|token|api[_-]?key|private[_-]?key|credential|dsn|auth|cookie|salt)`)

// krewireYAML flattens krewire.yaml scalar leaves into dotted keys.
func (g *gen) krewireYAML() {
	const rel = "krewire.yaml"
	data, ok := g.read(rel)
	if !ok {
		return
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		g.diag("KWC4002", "krewire.yaml: "+err.Error(), prov(rel, "", 0, "krewire.yaml"), "fix the YAML syntax")
		return
	}
	var walk func(n *yaml.Node, key string)
	walk = func(n *yaml.Node, key string) {
		switch n.Kind {
		case yaml.DocumentNode:
			for _, c := range n.Content {
				walk(c, key)
			}
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				k := n.Content[i].Value
				if key != "" {
					k = key + "." + k
				}
				walk(n.Content[i+1], k)
			}
		case yaml.SequenceNode:
			for i, c := range n.Content {
				walk(c, key+"."+strconv.Itoa(i))
			}
		case yaml.ScalarNode:
			cv := contract.ConfigValue{Key: key, Provenance: prov(rel, key, n.Line, "krewire.yaml")}
			if secretKey.MatchString(key) || looksSecret(n.Value) {
				cv.Redacted = true
			} else {
				cv.Value = n.Value
			}
			g.ctx.Config = append(g.ctx.Config, cv)
		}
	}
	walk(&doc, "")
}
