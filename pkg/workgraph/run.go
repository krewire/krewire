package workgraph

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Status is the decision recorded for one target.
type Status string

// Target statuses reported to CI and AI workflows.
const (
	StatusExecuted Status = "executed" // ran and passed
	StatusFailed   Status = "failed"   // ran and failed
	StatusRestored Status = "restored" // passing result restored from cache
	StatusSkipped  Status = "skipped"  // not selected (not affected / not requested)
	StatusBlocked  Status = "blocked"  // a dependency failed, or the run was cancelled
)

// ExecRequest is one command invocation.
type ExecRequest struct {
	Dir  string // absolute working directory
	Argv []string
}

// ExecResult is the outcome of an invocation. Err reports failures to start.
type ExecResult struct {
	ExitCode int
	Output   string
	Err      error
}

// Executor runs commands. Tests and sandboxes substitute their own.
type Executor interface {
	Exec(ctx context.Context, req ExecRequest) ExecResult
}

// OSExecutor runs commands as local processes with combined output.
type OSExecutor struct{}

// Exec implements Executor.
func (OSExecutor) Exec(ctx context.Context, req ExecRequest) ExecResult {
	cmd := exec.CommandContext(ctx, req.Argv[0], req.Argv[1:]...)
	cmd.Dir = req.Dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return ExecResult{Output: out.String()}
	case errors.As(err, &exitErr):
		return ExecResult{ExitCode: exitErr.ExitCode(), Output: out.String()}
	}
	return ExecResult{ExitCode: -1, Output: out.String(), Err: err}
}

// RunOptions configures a run.
type RunOptions struct {
	// Affected selects targets by node; nil means full-run mode.
	Affected *AffectedSet
	// Tasks restricts the requested task names (their dependencies still run).
	Tasks []string
	// Concurrency bounds parallel targets (default GOMAXPROCS).
	Concurrency int
	// Cache enables restore and store; nil disables caching.
	Cache Cache
	// NoRestore executes every selected target but still stores results.
	NoRestore bool
	Hasher    *Hasher
	Executor  Executor
	// OutputLimit keeps the last N bytes of output per target (default 8 KiB).
	OutputLimit int
	// IncludeInputs embeds each target's KeyInputs in the report.
	IncludeInputs bool
}

// Run executes the selected targets in dependency order with bounded
// parallelism and returns the report. The error is non-nil only for invalid
// options; target failures are reported, see Report.OK.
func (tg *TaskGraph) Run(ctx context.Context, ws *Workspace, opt RunOptions) (*Report, error) {
	if opt.Executor == nil {
		opt.Executor = OSExecutor{}
	}
	if opt.Concurrency <= 0 {
		opt.Concurrency = runtime.GOMAXPROCS(0)
	}
	if opt.OutputLimit <= 0 {
		opt.OutputLimit = 8 << 10
	}
	if opt.Hasher == nil {
		if opt.Cache != nil {
			return nil, errors.New("workgraph: a cache requires a hasher")
		}
	}
	sel := tg.selection(ws, opt)
	r := newReport(ws, opt, sel.mode)
	results := make(map[string]*TargetResult, len(tg.order))
	done := make(map[string]chan struct{}, len(tg.order))
	for _, id := range tg.order {
		done[id] = make(chan struct{})
	}
	for _, id := range tg.order {
		t := tg.targets[id]
		res := &TargetResult{ID: id, Task: t.Task, Node: t.Node, Kind: t.spec.Kind, Argv: t.Argv}
		if c, ok := sel.cause[t.Node]; ok {
			res.Cause = &c
		}
		results[id] = res
	}
	// results is read-only from here on; each result is written only by its
	// own goroutine before done[id] is closed, which orders the reads below.
	sem := make(chan struct{}, opt.Concurrency)
	var wg sync.WaitGroup
	for _, id := range tg.order {
		t, res := tg.targets[id], results[id]
		if !sel.run[id] {
			res.Status, res.Reason = StatusSkipped, sel.why(id, t)
			close(done[id])
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(done[id])
			for _, d := range t.Deps {
				<-done[d]
				if ds := results[d].Status; ds == StatusFailed || ds == StatusBlocked {
					res.Status, res.Reason = StatusBlocked, fmt.Sprintf("dependency %s %s", d, ds)
					return
				}
			}
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
			}
			if ctx.Err() != nil {
				res.Status, res.Reason = StatusBlocked, "cancelled: "+ctx.Err().Error()
				return
			}
			tg.runOne(ctx, ws, opt, t, res)
		}()
	}
	wg.Wait()
	for _, id := range tg.order {
		r.add(*results[id])
	}
	return r, nil
}

func (tg *TaskGraph) runOne(ctx context.Context, ws *Workspace, opt RunOptions, t *Target, res *TargetResult) {
	reason := "cache disabled"
	cacheable := opt.Cache != nil && !t.spec.NoCache
	if opt.Hasher != nil {
		in, err := opt.Hasher.Inputs(t)
		if err != nil {
			cacheable, reason = false, "inputs unhashable ("+err.Error()+")"
		} else {
			res.Key = in.Key()
			if opt.IncludeInputs {
				res.Inputs = &in
			}
		}
	}
	switch {
	case !cacheable:
		if opt.Cache != nil && t.spec.NoCache {
			reason = "task is not cacheable"
		}
	case opt.NoRestore:
		reason = "restore disabled"
	default:
		e, err := opt.Cache.Get(res.Key)
		switch {
		case err == nil:
			res.Status, res.Output = StatusRestored, tail(e.Output, opt.OutputLimit)
			res.Reason = "cache hit " + res.Key[:12] + " (passed with identical inputs)"
			return
		case errors.Is(err, ErrCacheCorrupt):
			reason = "cache entry corrupt; fell back to execution"
		default:
			reason = "cache miss"
		}
	}
	start := time.Now()
	out := opt.Executor.Exec(ctx, ExecRequest{Dir: filepath.Join(ws.Root, filepath.FromSlash(t.WorkDir)), Argv: t.Argv})
	res.DurationMs = time.Since(start).Milliseconds()
	res.ExitCode, res.Output = out.ExitCode, tail(out.Output, opt.OutputLimit)
	switch {
	case out.Err != nil:
		res.Status, res.Reason = StatusFailed, reason+"; could not start: "+out.Err.Error()
	case out.ExitCode != 0:
		res.Status, res.Reason = StatusFailed, fmt.Sprintf("%s; exit code %d", reason, out.ExitCode)
	case t.spec.FailOnOutput && strings.TrimSpace(out.Output) != "":
		res.Status, res.Reason = StatusFailed, reason+"; produced output on a fail-on-output gate"
	default:
		res.Status, res.Reason = StatusExecuted, reason
		if cacheable {
			err := opt.Cache.Put(&Entry{Key: res.Key, Target: t.ID, Output: out.Output, DurationMs: res.DurationMs})
			if err != nil {
				res.Reason += "; cache store failed: " + err.Error()
			}
		}
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
