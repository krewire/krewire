package projctx

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/krewire/krewire/model"
	"github.com/krewire/krewire/sec"
	"github.com/krewire/krewire/toolchain/contract"
)

// looksSecret reports values that must not be emitted whatever their key:
// credentials pkg/sec would redact and URLs carrying a password.
func looksSecret(v string) bool {
	if sec.RedactSensitiveOutput(v) != v {
		return true
	}
	if u, err := url.Parse(v); err == nil && u.User != nil {
		if _, has := u.User.Password(); has {
			return true
		}
	}
	return false
}

// DefaultGates are used when AGENTS.md declares none. They mirror the core
// quality-gate contract and never need an LLM or network access.
func DefaultGates() []contract.Gate {
	src := prov("AGENTS.md", "Quality Gates", 0, "contract-default")
	return []contract.Gate{
		{ID: "fmt", Type: model.GateTypeFormat, Argv: []string{"gofmt", "-l", "."}, Required: true, FailOnOutput: true, Provenance: src},
		{ID: "vet", Type: model.GateTypeStatic, Argv: []string{"go", "vet", "./..."}, Required: true, Provenance: src},
		{ID: "test", Type: model.GateTypeTest, Argv: []string{"go", "test", "./..."}, Required: true, Provenance: src},
		{ID: "build", Type: model.GateTypeBuild, Argv: []string{"go", "build", "./..."}, Env: []string{"CGO_ENABLED=0"}, Required: true, Provenance: src},
	}
}

// gateFromLine turns one shell line from the Quality Gates block into a gate.
// Leading NAME=value words become Env; a trailing `# comment` is dropped.
func gateFromLine(line string, src contract.Provenance) (contract.Gate, bool) {
	if i := strings.Index(line, "#"); i >= 0 {
		line = line[:i]
	}
	words := strings.Fields(line)
	var env []string
	for len(words) > 0 && strings.Contains(words[0], "=") && !strings.HasPrefix(words[0], "-") {
		env, words = append(env, words[0]), words[1:]
	}
	if len(words) == 0 {
		return contract.Gate{}, false
	}
	gate := contract.Gate{ID: words[0], Type: model.GateTypeCustom, Argv: words, Env: env, Required: true, Provenance: src}
	switch {
	case words[0] == "gofmt":
		gate.ID, gate.Type, gate.FailOnOutput = "fmt", model.GateTypeFormat, true
	case words[0] == "go" && len(words) > 1:
		gate.ID = words[1]
		switch words[1] {
		case "vet":
			gate.Type = model.GateTypeStatic
		case "test":
			gate.Type = model.GateTypeTest
		case "build":
			gate.Type = model.GateTypeBuild
		}
		gate.ID += targetSuffix(env)
	}
	return gate, true
}

// targetSuffix distinguishes cross-compilation gates (for example
// `GOOS=js GOARCH=wasm go build ./...` next to a host `go build ./...`) so
// they get distinct IDs instead of colliding as duplicates.
func targetSuffix(env []string) string {
	var goos, goarch string
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "GOOS":
			goos = v
		case "GOARCH":
			goarch = v
		}
	}
	suffix := ""
	for _, part := range []string{goos, goarch} {
		if part != "" {
			suffix += "-" + part
		}
	}
	return suffix
}

// agentsMD reads architecture rules and quality gates from AGENTS.md.
func (g *gen) agentsMD() {
	const rel = "AGENTS.md"
	data, ok := g.read(rel)
	if !ok {
		g.diag("KWC1003", "AGENTS.md not found", prov(rel, "", 0, "agents-md"), "add AGENTS.md with Architecture Rules and Quality Gates")
		g.ctx.Gates = DefaultGates()
		return
	}
	section, inFence, rule := "", false, 0
	for i, line := range lines(data) {
		n := i + 1
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if !inFence && strings.HasPrefix(trimmed, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		switch {
		case section == "Quality Gates" && inFence && trimmed != "":
			if gate, ok := gateFromLine(trimmed, prov(rel, section, n, "agents-md")); ok {
				g.ctx.Gates = append(g.ctx.Gates, gate)
			}
		case section == "Architecture Rules" && !inFence && strings.HasPrefix(trimmed, "- "):
			rule++
			g.ctx.Rules = append(g.ctx.Rules, contract.Rule{
				ID: "agents-md/architecture/" + strconv.Itoa(rule), Category: "architecture",
				Statement: strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), Provenance: prov(rel, section, n, "agents-md")})
		}
	}
	if len(g.ctx.Gates) == 0 {
		g.diag("KWC1004", "AGENTS.md has no fenced command block under \"## Quality Gates\"", prov(rel, "Quality Gates", 0, "agents-md"),
			"list one gate command per line in a fenced block")
		g.ctx.Gates = DefaultGates()
	}
}
