package conformance

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/krewire/krewire/pkg/model"
)

// maxOutput bounds the command output kept per gate.
const maxOutput = 4 << 10

// run is the execution state of one workload fixture.
type run struct {
	env     *Env
	ctx     context.Context
	dir     string // scaffolded project root (clean temp dir)
	res     *WorkloadResult
	blocked bool // a required gate failed; later gates are BLOCKED
}

// gate executes fn as one evidence gate. A failed required gate blocks every
// later gate of the workload, so nothing downstream of a failure can pass.
func (r *run) gate(step string, typ model.QualityGateType, desc, command string, fn func() (string, error)) bool {
	ev := model.VerificationEvidence{
		GateID:      r.res.Workload + "." + step,
		Type:        typ,
		Description: desc,
		Command:     command,
		Required:    true,
	}
	start := time.Now()
	switch {
	case r.blocked:
		ev.Status, ev.ExitCode, ev.Output = model.GateBlocked, -1, "blocked by an earlier failed gate"
	default:
		out, err := fn()
		ev.Output = truncate(out)
		ev.Status = model.GatePassed
		if err != nil {
			ev.Status, ev.ExitCode = model.GateFailed, exitCode(err)
			ev.Output = truncate(strings.TrimSpace(out + "\n" + err.Error()))
			r.blocked = true
		}
	}
	ev.DurationMs = time.Since(start).Milliseconds()
	ev.EvidenceHash = ev.ComputeHash(r.env.Commit)
	r.res.Gates = append(r.res.Gates, ev)
	return ev.Status == model.GatePassed
}

// assert is a non-blocking gate: a failed behavioural probe is recorded but
// does not prevent shutdown and diagnostics from being exercised.
func (r *run) assert(step string, typ model.QualityGateType, desc, command string, fn func() (string, error)) bool {
	prev := r.blocked
	ok := r.gate(step, typ, desc, command, fn)
	r.blocked = prev
	return ok
}

// exclude records a lifecycle step the core repository cannot exercise.
func (r *run) exclude(step, reason, tracking string) {
	r.res.Exclusions = append(r.res.Exclusions, Exclusion{Step: step, Reason: reason, Tracking: tracking})
}

// scaffold renders a starter template directory from templates/ and copies
// the maintained fixture tree on top, then writes a go.mod that resolves the
// framework from this checkout.
func (r *run) scaffold(template string) error {
	data := struct{ Name, Module string }{r.res.Workload + "-fixture", "example.com/kwconf/" + r.res.Workload}
	src := filepath.Join(r.env.Root, "templates", template)
	if err := renderTree(src, r.dir, data); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(r.env.Root, "internal", "conformance", "testdata", r.res.Workload), r.dir); err != nil {
		return err
	}
	gomod := fmt.Sprintf("module %s\n\ngo %s\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
		data.Module, r.env.GoDirective, r.env.Module, r.env.Module, filepath.ToSlash(r.env.Root))
	if err := os.WriteFile(filepath.Join(r.dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return err
	}
	sum, err := os.ReadFile(filepath.Join(r.env.Root, "go.sum"))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.dir, "go.sum"), sum, 0o644)
}

// goCmd runs the go tool inside the fixture with a hermetic, pure-Go env.
func (r *run) goCmd(timeout time.Duration, args ...string) (string, error) {
	return r.exec(timeout, nil, r.env.GoBin, args...)
}

// exec runs a command in the fixture directory and returns combined output.
func (r *run) exec(timeout time.Duration, extraEnv []string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(r.ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.dir
	cmd.Env = append(r.env.buildEnv(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s", timeout)
	}
	return string(out), err
}

// build compiles the fixture and records the binary as an artifact.
func (r *run) build(out string) (string, error) {
	o, err := r.goCmd(3*time.Minute, "build", "-trimpath", "-o", out, ".")
	if err != nil {
		return o, err
	}
	a, err := fingerprint(r.dir, out)
	if err != nil {
		return o, err
	}
	r.res.Artifacts = append(r.res.Artifacts, a)
	return o + "sha256 " + a.SHA256, nil
}

// proc is a started fixture process.
type proc struct {
	cmd    *exec.Cmd
	stderr *lockedBuffer
	mu     sync.Mutex
	lines  []string
	ready  chan string
	done   chan error
}

// start launches the binary with the given env and waits for its
// "READY <detail>" line.
func (r *run) start(bin string, env []string, args ...string) (*proc, string, error) {
	cmd := exec.Command(filepath.Join(r.dir, bin), args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), env...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	stderr := &lockedBuffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	p := &proc{cmd: cmd, stderr: stderr, ready: make(chan string, 1), done: make(chan error, 1)}
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			p.mu.Lock()
			p.lines = append(p.lines, line)
			p.mu.Unlock()
			if rest, ok := strings.CutPrefix(line, "READY"); ok {
				select {
				case p.ready <- strings.TrimSpace(rest):
				default:
				}
			}
		}
		p.done <- cmd.Wait()
	}()
	select {
	case d := <-p.ready:
		return p, d, nil
	case err := <-p.done:
		return nil, "", fmt.Errorf("exited before READY: %v\n%s", err, stderr.String())
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		return nil, "", errors.New("no READY line within 30s")
	}
}

// waitLine waits until the process printed a line with the given prefix.
func (p *proc) waitLine(prefix string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		for _, l := range p.lines {
			if strings.HasPrefix(l, prefix) {
				p.mu.Unlock()
				return l, nil
			}
		}
		p.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	return "", fmt.Errorf("no %q line within %s", prefix, timeout)
}

// stop sends an interrupt and requires a clean exit with a STOPPED line.
func (p *proc) stop(timeout time.Duration) (string, error) {
	if err := interrupt(p.cmd.Process); err != nil {
		return p.stderr.String(), err
	}
	select {
	case err := <-p.done:
		if err != nil {
			return p.stderr.String(), fmt.Errorf("unclean exit: %w", err)
		}
	case <-time.After(timeout):
		_ = p.cmd.Process.Kill()
		return "", fmt.Errorf("did not exit within %s of interrupt", timeout)
	}
	return p.waitLine("STOPPED", time.Second)
}

// lockedBuffer is a goroutine-safe stderr sink.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// kill is the cleanup path for a process whose stop gate never ran.
func (p *proc) kill() {
	if p != nil && p.cmd.ProcessState == nil {
		_ = p.cmd.Process.Kill()
	}
}

// get performs an HTTP GET and returns status and body.
func get(url string) (int, string, error) {
	c := http.Client{Timeout: 10 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(b), err
}

// expect asserts an HTTP probe result.
func expect(url string, status int, contains string) (string, error) {
	code, body, err := get(url)
	if err != nil {
		return "", err
	}
	got := fmt.Sprintf("GET %s -> %d %q", url, code, truncateTo(body, 200))
	if code != status || !strings.Contains(body, contains) {
		return got, fmt.Errorf("want %d containing %q", status, contains)
	}
	return got, nil
}

// freeAddr reserves a loopback port for a fixture.
func freeAddr() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

func renderTree(src, dst string, data any) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		t, err := template.New(rel).Option("missingkey=error").Parse(string(raw))
		if err != nil {
			return fmt.Errorf("template %s: %w", rel, err)
		}
		var b bytes.Buffer
		if err := t.Execute(&b, data); err != nil {
			return fmt.Errorf("template %s: %w", rel, err)
		}
		return writeFile(filepath.Join(dst, strings.TrimSuffix(rel, ".tmpl")), b.Bytes())
	})
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if rel == "krewire.yaml" {
			// A fixture krewire.yaml extends the rendered starter config with
			// declarative battery keys (worker:, infra:, ...).
			prev, err := os.ReadFile(target)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			b = append(append(prev, '\n'), b...)
		}
		return writeFile(target, b)
	})
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func fingerprint(dir, rel string) (Artifact, error) {
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return Artifact{}, err
	}
	sum := sha256.Sum256(b)
	return Artifact{Path: filepath.ToSlash(rel), SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(b))}, nil
}

func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func truncate(s string) string { return truncateTo(ansi.ReplaceAllString(s, ""), maxOutput) }

func truncateTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…[truncated]"
}
