package workgraph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeExec records invocations and fails argv strings listed in fail.
type fakeExec struct {
	mu        sync.Mutex
	calls     []string
	fail      map[string]bool
	output    map[string]string
	running   int
	maxActive int
	finished  map[string]bool
	order     []string
}

func (f *fakeExec) Exec(_ context.Context, req ExecRequest) ExecResult {
	cmd := strings.Join(req.Argv, " ")
	f.mu.Lock()
	f.calls = append(f.calls, cmd)
	f.running++
	f.maxActive = max(f.maxActive, f.running)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.running--
		f.order = append(f.order, cmd)
		f.mu.Unlock()
	}()
	if f.fail[cmd] {
		return ExecResult{ExitCode: 2, Output: "boom\n"}
	}
	return ExecResult{Output: f.output[cmd]}
}

type runEnv struct {
	dir   string
	ws    *Workspace
	tg    *TaskGraph
	cache LocalCache
	env   map[string]string
	exec  *fakeExec
	goV   string
}

func newRunEnv(t *testing.T) *runEnv {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/ws")); err != nil {
		t.Fatal(err)
	}
	e := &runEnv{dir: dir, env: map[string]string{"GOOS": "linux"}, goV: "go1.27.1",
		cache: LocalCache{Dir: filepath.Join(t.TempDir(), "cache")}}
	return e
}

func (e *runEnv) run(t *testing.T, opt RunOptions) *Report {
	t.Helper()
	ws, err := Discover(e.dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.ws, e.tg = ws, fixtureTasks(t, ws)
	if e.exec == nil {
		e.exec = &fakeExec{}
	}
	opt.Executor = e.exec
	opt.Hasher = NewHasher(ws, e.goV, func(k string) string { return e.env[k] })
	if opt.Cache == nil {
		opt.Cache = e.cache
	}
	r, err := e.tg.Run(context.Background(), ws, opt)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (e *runEnv) write(t *testing.T, rel, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(e.dir, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func statuses(r *Report) map[string]Status {
	out := map[string]Status{}
	for _, t := range r.Targets {
		out[t.ID] = t.Status
	}
	return out
}

func wantSummary(t *testing.T, r *Report, want map[Status]int) {
	t.Helper()
	for s, n := range want {
		if r.Summary[s] != n {
			var ids []string
			for _, tr := range r.Targets {
				ids = append(ids, string(tr.Status)+" "+tr.ID)
			}
			t.Fatalf("summary = %v, want %v\n%s", r.Summary, want, strings.Join(ids, "\n"))
		}
	}
}

func TestRunAffectedSkipsWithReasons(t *testing.T) {
	e := newRunEnv(t)
	ws, err := Discover(e.dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := e.run(t, RunOptions{Affected: ws.Affected([]string{"tools/cmd/tool/main.go"}, AffectedOptions{Base: "b", Head: "h"})})
	wantSummary(t, r, map[Status]int{StatusExecuted: 4, StatusSkipped: 12})
	for _, tr := range r.Targets {
		if tr.Node != pTool && tr.Reason != "not affected" {
			t.Errorf("%s reason = %q", tr.ID, tr.Reason)
		}
		if tr.Node == pTool && (tr.Cause == nil || tr.Cause.Kind != CauseChanged) {
			t.Errorf("%s cause = %+v", tr.ID, tr.Cause)
		}
	}
	if r.Mode != ModeAffected || r.Base != "b" || r.GoVersion != "go1.27.1" {
		t.Fatalf("report header = %+v", r)
	}
	r = e.run(t, RunOptions{Tasks: []string{"test"}, NoRestore: true})
	wantSummary(t, r, map[Status]int{StatusExecuted: 8, StatusSkipped: 8})
	if got := statuses(r)[TargetID("vet", pBase)]; got != StatusSkipped {
		t.Fatalf("vet not requested but %s", got)
	}
}

func TestRunCacheInvalidation(t *testing.T) {
	e := newRunEnv(t)
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 16})
	r := e.run(t, RunOptions{})
	wantSummary(t, r, map[Status]int{StatusRestored: 16})
	if !strings.HasPrefix(r.Targets[0].Reason, "cache hit") {
		t.Fatalf("reason = %q", r.Targets[0].Reason)
	}
	e.write(t, "tools/cmd/tool/main.go", "package main\n\nimport \"example.com/core/base\"\n\nfunc main() { _ = base.Name }\n")
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 4, StatusRestored: 12})
	e.write(t, "core/base/base.go", "package base\n\nconst Name = \"b2\"\n")
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 16})
	e.write(t, "core/go.sum", "golang.org/x/text v0.43.0 h1:other\n")
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 16})
	e.env["CGO_ENABLED"] = "0"
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 16})
	e.goV = "go1.27.2"
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 16})
	e.write(t, "core/schemas/config.schema.json", "{}\n")
	wantSummary(t, e.run(t, RunOptions{}), map[Status]int{StatusExecuted: 16})
	wantSummary(t, e.run(t, RunOptions{NoRestore: true}), map[Status]int{StatusExecuted: 16})
	r = e.run(t, RunOptions{IncludeInputs: true})
	in := r.Targets[0].Inputs
	if in == nil || in.GoVersion != "go1.27.2" || in.Lock["core/go.sum"] == "" || in.Env["CGO_ENABLED"] != "0" ||
		in.Files["core/schemas/config.schema.json"] == "" || in.Key() != r.Targets[0].Key {
		t.Fatalf("inputs = %+v", in)
	}
}

func TestRunFailuresBlockAndAreNeverCached(t *testing.T) {
	e := newRunEnv(t)
	e.exec = &fakeExec{fail: map[string]bool{"go build example.com/core/mid": true},
		output: map[string]string{"gofmt -l ./top": "top/top.go\n"}}
	r := e.run(t, RunOptions{})
	st := statuses(r)
	for id, want := range map[string]Status{
		TargetID("build", pMid): StatusFailed, TargetID("test", pMid): StatusBlocked,
		TargetID("build", pTop): StatusBlocked, TargetID("test", pTop): StatusBlocked,
		TargetID("format", pTop): StatusFailed, TargetID("build", pTool): StatusExecuted,
	} {
		if st[id] != want {
			t.Errorf("%s = %s, want %s", id, st[id], want)
		}
	}
	if r.OK() {
		t.Fatal("report OK despite failures")
	}
	ev := r.Evidence("abc")
	if len(ev) != 16 || ev[0].EvidenceHash == "" {
		t.Fatalf("evidence = %d entries", len(ev))
	}
	e.exec.calls = nil
	r = e.run(t, RunOptions{})
	if statuses(r)[TargetID("build", pMid)] != StatusFailed || statuses(r)[TargetID("format", pTop)] != StatusFailed {
		t.Fatal("failed targets were not re-executed")
	}
	if statuses(r)[TargetID("build", pBase)] != StatusRestored {
		t.Fatal("passing targets should restore")
	}
}

func TestRunCorruptCacheFallsBack(t *testing.T) {
	e := newRunEnv(t)
	r := e.run(t, RunOptions{})
	key := r.Targets[0].Key
	p := filepath.Join(e.cache.Dir, key[:2], key+".json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = e.run(t, RunOptions{})
	if r.Targets[0].Status != StatusExecuted || !strings.Contains(r.Targets[0].Reason, "corrupt") {
		t.Fatalf("corrupt entry: %s %q", r.Targets[0].Status, r.Targets[0].Reason)
	}
	data, _ := os.ReadFile(p)
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), `"output":""`, `"output":"forged"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.cache.Get(key); err == nil {
		t.Fatal("tampered entry accepted")
	}
	if _, err := e.cache.Get("../x"); err == nil {
		t.Fatal("path-like key accepted")
	}
	if err := e.cache.Put(&Entry{Key: key, ExitCode: 1}); err == nil {
		t.Fatal("failing entry cached")
	}
}

func TestRunParallelRespectsDependencies(t *testing.T) {
	e := newRunEnv(t)
	e.run(t, RunOptions{Concurrency: 2, NoRestore: true})
	if e.exec.maxActive > 2 {
		t.Fatalf("max concurrency %d > 2", e.exec.maxActive)
	}
	pos := map[string]int{}
	for i, c := range e.exec.order {
		pos[c] = i
	}
	if pos["go build example.com/core/base"] > pos["go build example.com/core/mid"] ||
		pos["go build example.com/core/mid"] > pos["go build example.com/core/top"] ||
		pos["go build example.com/core/top"] > pos["go test example.com/core/top"] {
		t.Fatalf("dependency order violated: %v", e.exec.order)
	}
}

func TestRunWithoutCacheAndCancelled(t *testing.T) {
	e := newRunEnv(t)
	ws, _ := Discover(e.dir, nil)
	tg := fixtureTasks(t, ws)
	if _, err := tg.Run(context.Background(), ws, RunOptions{Cache: e.cache}); err == nil {
		t.Fatal("cache without hasher accepted")
	}
	r, err := tg.Run(context.Background(), ws, RunOptions{Executor: &fakeExec{}})
	if err != nil || r.Summary[StatusExecuted] != 16 || r.Targets[0].Reason != "cache disabled" || r.Mode != ModeFull {
		t.Fatalf("uncached run: %v %+v", err, r.Summary)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, _ = tg.Run(ctx, ws, RunOptions{Executor: &fakeExec{}, Concurrency: 1})
	if r.OK() || r.Summary[StatusBlocked] != 16 {
		t.Fatal("cancelled run reported OK")
	}
}

func TestOSExecutorAndGoVersion(t *testing.T) {
	ctx := context.Background()
	v, err := GoVersion(ctx, ".")
	if err != nil || !strings.HasPrefix(v, "go") {
		t.Fatalf("GoVersion = %q, %v", v, err)
	}
	if r := (OSExecutor{}).Exec(ctx, ExecRequest{Dir: ".", Argv: []string{"go", "env", "GOOS"}}); r.ExitCode != 0 || r.Err != nil {
		t.Fatalf("go env: %+v", r)
	}
	if r := (OSExecutor{}).Exec(ctx, ExecRequest{Dir: ".", Argv: []string{"go", "no-such-subcommand"}}); r.ExitCode == 0 {
		t.Fatalf("bad subcommand passed: %+v", r)
	}
	if r := (OSExecutor{}).Exec(ctx, ExecRequest{Dir: ".", Argv: []string{"/nonexistent/binary"}}); r.Err == nil {
		t.Fatal("missing binary did not error")
	}
}

func TestReportRendering(t *testing.T) {
	e := newRunEnv(t)
	e.exec = &fakeExec{fail: map[string]bool{"go vet example.com/core/base": true}}
	r := e.run(t, RunOptions{})
	var buf strings.Builder
	if err := r.WriteText(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "failed   vet#"+pBase+": cache miss; exit code 2") || !strings.Contains(buf.String(), "| boom") {
		t.Fatalf("text:\n%s", buf.String())
	}
	buf.Reset()
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"kind": "TaskReport"`, `"summary"`, `"cache_key"`, `"reason": "cache miss"`} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("json lacks %s", want)
		}
	}
	if tail("abcdef", 3) != "…def" {
		t.Fatal("tail")
	}
}
