package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/model"
)

// Workloads lists the eight canonical workloads in matrix order
// (docs/workloads/index.md).
var Workloads = []string{"site", "book", "app", "cli", "worker", "service", "gateway", "infra"}

func order(name string) int {
	for i, w := range Workloads {
		if w == name {
			return i
		}
	}
	return len(Workloads)
}

// fixtures maps each workload to its lifecycle script.
var fixtures = map[string]func(*run){
	"site": siteFixture, "book": bookFixture, "app": appFixture, "cli": cliFixture,
	"worker": workerFixture, "service": serviceFixture, "gateway": gatewayFixture, "infra": infraFixture,
}

const (
	gateCustom = model.GateTypeCustom
	gateBuild  = model.GateTypeBuild
	gateTest   = model.GateTypeTest
	fixtureBin = "bin/fixture"
)

// prepare runs the steps every workload shares: scaffold from a starter
// template into the clean dir, resolve dependencies, vet, pure-Go build, and
// validate the project configuration through the framework.
func (r *run) prepare(template string) bool {
	r.gate("scaffold", gateCustom, "render templates/"+template+" plus the maintained fixture into a clean temp dir",
		"render templates/"+template, func() (string, error) { return r.dir, r.scaffold(template) })
	r.gate("resolve", gateCustom, "resolve the module graph against this checkout",
		"go mod tidy", func() (string, error) { return r.goCmd(3*time.Minute, "mod", "tidy") })
	r.gate("vet", model.GateTypeStatic, "static analysis of the scaffolded project",
		"go vet ./...", func() (string, error) { return r.goCmd(3*time.Minute, "vet", "./...") })
	r.gate("build", gateBuild, "pure-Go build (CGO_ENABLED=0)",
		"go build -trimpath -o "+fixtureBin+" .", func() (string, error) { return r.build(fixtureBin) })
	r.gate("validate", gateCustom, "load and validate krewire.yaml through the framework",
		fixtureBin+" check", func() (string, error) { return r.fixture("check") })
	return !r.blocked
}

// fixture runs the built fixture binary to completion.
func (r *run) fixture(args ...string) (string, error) {
	return r.exec(time.Minute, nil, filepath.Join(r.dir, fixtureBin), args...)
}

// serve starts the fixture, runs probes against it, and requires a graceful
// interrupt-driven shutdown. env is passed to the process.
func (r *run) serve(env []string, probes func(detail string, p *proc)) {
	var p *proc
	var detail string
	r.gate("start", gateTest, "start the workload and wait for readiness",
		fixtureBin+" serve", func() (string, error) {
			var err error
			p, detail, err = r.start(fixtureBin, env, "serve")
			return "READY " + detail, err
		})
	defer p.kill()
	if p != nil && !r.blocked {
		probes(detail, p)
	}
	r.gate("shutdown", gateTest, "interrupt and require graceful shutdown with exit 0",
		"SIGINT", func() (string, error) { return p.stop(15 * time.Second) })
}

// probe records an HTTP expectation; readiness races are retried briefly.
func (r *run) probe(step, desc, url string, status int, contains string) {
	r.assert(step, gateTest, desc, "GET "+url, func() (string, error) {
		var out string
		var err error
		for i := 0; i < 50; i++ {
			if out, err = expect(url, status, contains); err == nil || out != "" {
				return out, err
			}
			time.Sleep(100 * time.Millisecond)
		}
		return out, err
	})
}

// diagnose mutates the project into a common invalid state and requires the
// framework to fail `check` with a diagnostic containing want.
func (r *run) diagnose(step, desc string, mutate func(dir string) error, want string, args ...string) {
	if len(args) == 0 {
		args = []string{"check"}
	}
	r.assert("diagnose."+step, gateTest, desc, fixtureBin+" "+strings.Join(args, " "), func() (string, error) {
		if err := mutate(r.dir); err != nil {
			return "", err
		}
		out, err := r.fixture(args...)
		if err == nil {
			return out, fmt.Errorf("invalid project was accepted: no diagnostic emitted (want %q)", want)
		}
		if !strings.Contains(out, want) {
			return out, fmt.Errorf("diagnostic lacks %q", want)
		}
		return out, nil
	})
}

// file asserts a built output file exists and contains want.
func (r *run) file(step, rel, want string) {
	r.assert(step, gateTest, "build output "+rel+" contains "+want, "read "+rel, func() (string, error) {
		b, err := os.ReadFile(filepath.Join(r.dir, rel))
		if err != nil {
			return "", err
		}
		if !strings.Contains(string(b), want) {
			return truncateTo(string(b), 300), fmt.Errorf("%s lacks %q", rel, want)
		}
		a, err := fingerprint(r.dir, rel)
		r.res.Artifacts = append(r.res.Artifacts, a)
		return "sha256 " + a.SHA256, err
	})
}

func writeRel(rel, content string) func(string) error {
	return func(dir string) error { return writeFile(filepath.Join(dir, rel), []byte(content)) }
}

func replaceIn(rel, old, new string) func(string) error {
	return func(dir string) error {
		p := filepath.Join(dir, rel)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), old) {
			return fmt.Errorf("%s lacks %q", rel, old)
		}
		return os.WriteFile(p, []byte(strings.Replace(string(b), old, new, 1)), 0o644)
	}
}

// maturityRank orders the documented maturity levels.
var maturityRank = map[string]int{"Unsupported": 0, "Planned": 1, "Experimental": 2, "Usable": 3, "Stable": 4}

// classify derives status, end-to-end flag and evidenced maturity. Stable is
// never evidenced here: it also needs compatibility, reproducible benchmark
// and production-use evidence this harness does not produce.
func classify(w *WorkloadResult) {
	failed := len(w.Gates) == 0
	for _, g := range w.Gates {
		if g.Status != model.GatePassed {
			failed = true
		}
	}
	w.EndToEnd = !failed && len(w.Exclusions) == 0 && len(w.Gates) > 0
	switch {
	case failed:
		w.Status, w.EvidencedMaturity = StatusFailed, "Experimental"
	case len(w.Exclusions) > 0:
		w.Status, w.EvidencedMaturity = StatusPartial, "Experimental"
	default:
		w.Status, w.EvidencedMaturity = StatusPassed, "Usable"
	}
	if maturityRank[w.DeclaredMaturity] > maturityRank[w.EvidencedMaturity] {
		w.Findings = append(w.Findings, fmt.Sprintf(
			"declared maturity %s exceeds automated evidence (%s); Stable also requires compatibility, reproducible benchmark and production-use evidence",
			w.DeclaredMaturity, w.EvidencedMaturity))
	}
}

// aiWorkflowSection reports the integrated AI-native workflow qualification.
// The shared context/execution contracts (pkg/toolchain, #131/#140) are not on
// main yet, so the journey cannot run here; it is excluded, never passed.
func aiWorkflowSection(root string) Section {
	if _, err := os.Stat(filepath.Join(root, "pkg", "toolchain", "workflow")); err == nil {
		return Section{Status: StatusExcluded, Exclusions: []Exclusion{{
			Step:     "ai-workflow",
			Reason:   "pkg/toolchain is present but not yet wired into this harness; run its workflow with the fake engine and attach the contract.EvidenceReport",
			Tracking: "#141",
		}}}
	}
	return Section{Status: StatusExcluded, Exclusions: []Exclusion{{
		Step:     "ai-workflow",
		Reason:   "the AI-native toolchain contracts (ProjectContext, TaskBrief, Plan, Policy, EvidenceReport) and the deterministic fake engine are not on main",
		Tracking: "#131, #140",
	}}}
}
