// Package projctx derives the contract.ProjectContext from a repository's
// source-of-truth files — go.mod, the root version declaration, AGENTS.md,
// krewire.yaml, Go package sources, executable architecture guards, and ADRs —
// and lints it with stable diagnostic codes.
//
// It is the single context implementation: `krew context`, every
// deterministic command, and every AI engine consume its output (#140). The
// result is deterministic: identical inputs produce byte-identical JSON.
package projctx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/toolchain/contract"
)

// Options configures context generation.
type Options struct {
	// Root is the repository root (the directory holding go.mod).
	Root string
	// ToolchainVersion is the generating toolchain's version, if known.
	ToolchainVersion string
	// Targets lists supported GOOS/GOARCH targets; nil uses DefaultTargets.
	Targets []contract.Target
}

// DefaultTargets are the targets the core qualifies (see platform_test.go).
var DefaultTargets = []contract.Target{
	{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "js", GOARCH: "wasm"},
	{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "windows", GOARCH: "amd64"},
}

// gen accumulates a context while reading inputs.
type gen struct {
	root   string
	inputs map[string]string // rel path → sha256 of content
	ctx    *contract.ProjectContext
}

// Generate derives the project context for opts.Root. Missing or malformed
// inputs become diagnostics, not errors; an error is returned only when the
// root itself cannot be read.
func Generate(opts Options) (*contract.ProjectContext, error) {
	info, err := os.Stat(opts.Root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("projctx: root %q is not a readable directory: %v", opts.Root, err)
	}
	targets := opts.Targets
	if targets == nil {
		targets = DefaultTargets
	}
	g := &gen{root: opts.Root, inputs: map[string]string{}, ctx: &contract.ProjectContext{
		TypeMeta: contract.Meta(contract.KindProjectContext),
		Toolchain: contract.ToolchainIdentity{Executable: contract.Executable,
			Version: opts.ToolchainVersion, ContractVersion: contract.ContractVersion},
		Packages: []contract.Package{}, Dependencies: []contract.Dependency{},
		Pillars: append([]string(nil), contract.Pillars...), Rules: []contract.Rule{},
		Config: []contract.ConfigValue{}, Gates: []contract.Gate{}, Diagnostics: []contract.Diagnostic{},
	}}
	g.ctx.Compatibility.Targets = append([]contract.Target(nil), targets...)

	g.goMod()
	g.versionDecl()
	g.krewireYAML()
	g.agentsMD()
	g.packages()
	g.guardRules()
	g.adrs()
	g.workload()
	g.lint()
	g.finish()
	return g.ctx, nil
}

// read returns a repository file and records it as an input.
func (g *gen) read(rel string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(g.root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256(data)
	g.inputs[rel] = hex.EncodeToString(sum[:])
	return data, true
}

func (g *gen) diag(code, msg string, loc contract.Provenance, hint string) {
	spec := Codes[code]
	g.ctx.Diagnostics = append(g.ctx.Diagnostics, contract.Diagnostic{Code: code, Class: spec.Class,
		Severity: spec.Severity, Message: msg, Location: loc, Hint: hint})
}

// finish sorts every list and computes the fingerprint.
func (g *gen) finish() {
	c := g.ctx
	sort.Slice(c.Packages, func(i, j int) bool { return c.Packages[i].ImportPath < c.Packages[j].ImportPath })
	sort.Slice(c.Dependencies, func(i, j int) bool { return c.Dependencies[i].Path < c.Dependencies[j].Path })
	sort.SliceStable(c.Rules, func(i, j int) bool {
		if c.Rules[i].Category != c.Rules[j].Category {
			return c.Rules[i].Category < c.Rules[j].Category
		}
		return c.Rules[i].ID < c.Rules[j].ID
	})
	sort.Slice(c.Config, func(i, j int) bool { return c.Config[i].Key < c.Config[j].Key })
	sort.SliceStable(c.Diagnostics, func(i, j int) bool {
		a, b := c.Diagnostics[i], c.Diagnostics[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Location.Path != b.Location.Path {
			return a.Location.Path < b.Location.Path
		}
		return a.Location.Line < b.Location.Line
	})
	paths := make([]string, 0, len(g.inputs))
	for p := range g.inputs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	h := sha256.New()
	fmt.Fprintf(h, "%s\n", contract.ContractVersion)
	for _, p := range paths {
		fmt.Fprintf(h, "%s\x00%s\n", p, g.inputs[p])
	}
	c.Fingerprint = hex.EncodeToString(h.Sum(nil))
}

// prov builds a provenance record.
func prov(path, key string, line int, schema string) contract.Provenance {
	return contract.Provenance{Path: path, Key: key, Line: line, Schema: schema}
}

// lines splits content into lines without trailing carriage returns.
func lines(data []byte) []string {
	out := strings.Split(string(data), "\n")
	for i := range out {
		out[i] = strings.TrimRight(out[i], "\r")
	}
	return out
}
