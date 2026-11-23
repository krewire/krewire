package conformance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire"
)

// Env is the checkout and toolchain under qualification.
type Env struct {
	Root        string // module root holding go.mod
	Module      string // module path from go.mod
	GoDirective string // go directive from go.mod
	GoBin       string // go tool used to build fixtures
	GoVersion   string
	Commit      string
	Dirty       bool
	// Maturity is the declared maturity per workload, parsed from
	// docs/workloads/maturity-matrix.md (the single source of truth).
	Maturity map[string]string
}

// Discover resolves the environment from a directory inside the checkout.
func Discover(dir string) (*Env, error) {
	root, err := findRoot(dir)
	if err != nil {
		return nil, err
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	env := &Env{Root: root, Commit: "unknown"}
	env.Module = directive(gomod, "module")
	env.GoDirective = directive(gomod, "go")
	if env.Module == "" || env.GoDirective == "" {
		return nil, fmt.Errorf("conformance: go.mod at %s lacks module/go directive", root)
	}
	if env.GoBin, err = exec.LookPath("go"); err != nil {
		return nil, fmt.Errorf("conformance: go tool not on PATH: %w", err)
	}
	if out, err := env.output(env.GoBin, "env", "GOVERSION"); err == nil {
		env.GoVersion = out
	}
	if out, err := env.output("git", "rev-parse", "HEAD"); err == nil {
		env.Commit = out
		st, _ := env.output("git", "status", "--porcelain")
		env.Dirty = st != ""
	}
	matrix, err := os.ReadFile(filepath.Join(root, "docs", "workloads", "maturity-matrix.md"))
	if err != nil {
		return nil, err
	}
	env.Maturity = ParseMaturity(string(matrix))
	return env, nil
}

var matrixRow = regexp.MustCompile("(?m)^\\| \\*\\*`([a-z]+)`\\*\\* \\| \\*\\*([A-Za-z]+)\\*\\* \\|")

// ParseMaturity extracts workload -> declared maturity from the matrix table.
func ParseMaturity(doc string) map[string]string {
	m := map[string]string{}
	for _, g := range matrixRow.FindAllStringSubmatch(doc, -1) {
		m[g[1]] = g[2]
	}
	return m
}

func (e *Env) output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = e.Root
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// buildEnv is the hermetic fixture build environment: no workspace, pure Go,
// module graph resolved from the module cache/proxy configured by the caller.
func (e *Env) buildEnv() []string {
	return append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0", "GOFLAGS=-mod=mod")
}

func findRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("conformance: no go.mod above %s", dir)
		}
		dir = parent
	}
}

func directive(gomod []byte, key string) string {
	for _, line := range strings.Split(string(gomod), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), key+" "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// Options tunes a qualification run.
type Options struct {
	// Workloads restricts the run; empty means all eight.
	Workloads []string
	// Parallel bounds concurrent fixtures; <= 0 picks min(4, NumCPU).
	Parallel int
	// Timeout bounds each workload fixture.
	Timeout time.Duration
	// TempDir is where clean fixture directories are created.
	TempDir string
}

// Run qualifies the selected workloads and returns the finalized report.
func Run(ctx context.Context, env *Env, opts Options) (*Report, error) {
	names := opts.Workloads
	if len(names) == 0 {
		names = Workloads
	}
	for _, n := range names {
		if _, ok := fixtures[n]; !ok {
			return nil, fmt.Errorf("conformance: unknown workload %q (want one of %s)", n, strings.Join(Workloads, ", "))
		}
	}
	if opts.Parallel <= 0 {
		opts.Parallel = min(4, runtime.NumCPU())
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	start := time.Now()
	rep := &Report{
		APIVersion: APIVersion, Kind: KindReport,
		Commit: env.Commit, Dirty: env.Dirty, Module: env.Module,
		Version:   krewire.VersionString(),
		Toolchain: Toolchain{Go: env.GoVersion, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CGO: "0"},
		StartedAt: start.UTC().Format(time.RFC3339),
	}
	results := make([]WorkloadResult, len(names))
	sem := make(chan struct{}, opts.Parallel)
	var wg sync.WaitGroup
	for i, n := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = qualify(ctx, env, n, opts)
		}()
	}
	wg.Wait()
	rep.Workloads = results
	rep.AIWorkflow = aiWorkflowSection(env.Root)
	rep.DurationMs = time.Since(start).Milliseconds()
	rep.Finalize()
	return rep, nil
}

// qualify runs one workload fixture in its own clean temporary directory.
func qualify(ctx context.Context, env *Env, name string, opts Options) WorkloadResult {
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	start := time.Now()
	res := WorkloadResult{Workload: name, Fixture: "internal/conformance/testdata/" + name, DeclaredMaturity: env.Maturity[name]}
	dir, err := os.MkdirTemp(opts.TempDir, "kwconf-"+name+"-")
	r := &run{env: env, ctx: ctx, dir: dir, res: &res}
	if err != nil {
		r.gate("scaffold", "CUSTOM", "create clean fixture directory", "mkdtemp", func() (string, error) { return "", err })
	} else {
		defer os.RemoveAll(dir)
		fixtures[name](r)
	}
	res.DurationMs = time.Since(start).Milliseconds()
	classify(&res)
	return res
}
