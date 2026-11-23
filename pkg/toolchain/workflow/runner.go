package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/policy"
)

// GateRunner executes one deterministic quality gate. It never needs an LLM.
type GateRunner interface {
	RunGate(ctx context.Context, root string, g contract.Gate) model.VerificationEvidence
}

// ExecRunner runs gates as argv processes (no shell) in the repository.
type ExecRunner struct {
	// Timeout bounds each gate; 0 means ten minutes.
	Timeout time.Duration
	// MaxOutput truncates recorded output; 0 means 64 KiB.
	MaxOutput int
	// Secrets are literal values scrubbed from output in addition to the
	// credential patterns pkg/sec recognizes.
	Secrets []string
	// Commit is the source revision the evidence hash binds to.
	Commit string
}

// RunGate implements GateRunner.
func (r ExecRunner) RunGate(ctx context.Context, root string, g contract.Gate) model.VerificationEvidence {
	timeout, max := r.Timeout, r.MaxOutput
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if max <= 0 {
		max = 64 << 10
	}
	ev := model.VerificationEvidence{GateID: g.ID, Type: g.Type, Command: strings.Join(append(append([]string(nil), g.Env...), g.Argv...), " "),
		Description: "deterministic gate " + g.ID, Required: g.Required}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(ctx, g.Argv[0], g.Argv[1:]...)
	cmd.Dir, cmd.Env = root, append(os.Environ(), g.Env...)
	out, err := cmd.CombinedOutput()
	ev.DurationMs = time.Since(start).Milliseconds()
	ev.Output = policy.Redact(string(out), max, r.Secrets...)
	var exit *exec.ExitError
	switch {
	case err == nil:
		ev.Status = model.GatePassed
		if g.FailOnOutput && strings.TrimSpace(string(out)) != "" {
			ev.Status, ev.ExitCode = model.GateFailed, 1
		}
	case errors.As(err, &exit):
		ev.Status, ev.ExitCode = model.GateFailed, exit.ExitCode()
	default:
		ev.Status, ev.ExitCode = model.GateBlocked, -1
		ev.Output = policy.Redact(err.Error(), max, r.Secrets...)
	}
	ev.EvidenceHash = ev.ComputeHash(r.Commit)
	return ev
}

// Verify runs the task's required gates and evaluates them with
// model.EvaluateTaskVerification. It is the whole of `krew verify` and is
// what the agentic flow calls: verification is never delegated to an engine.
func Verify(ctx context.Context, root string, pctx *contract.ProjectContext, gateIDs []string, run GateRunner, ledger *model.VerificationLedger, commit string) contract.VerificationSection {
	vs := contract.VerificationSection{Gates: []model.VerificationEvidence{}, GitCommit: commit}
	for _, id := range gateIDs {
		g, ok := pctx.Gate(id)
		if !ok {
			vs.Gates = append(vs.Gates, model.VerificationEvidence{GateID: id, Required: true, Status: model.GateBlocked,
				Type: model.GateTypeCustom, Output: "gate is not declared in the project context"})
			continue
		}
		if ctx.Err() != nil {
			vs.Gates = append(vs.Gates, model.VerificationEvidence{GateID: id, Type: g.Type, Required: g.Required,
				Command: strings.Join(g.Argv, " "), Status: model.GateBlocked, Output: ctx.Err().Error()})
			continue
		}
		vs.Gates = append(vs.Gates, run.RunGate(ctx, root, g))
	}
	vs.Verified, vs.Summary = model.EvaluateTaskVerification(vs.Gates)
	if ledger != nil {
		if b, err := ledger.AppendBlock(commit, vs.Gates); err == nil {
			vs.LedgerBlockHash = b.BlockHash
		}
	}
	return vs
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}

// applyFile performs an authorized write_file or delete_file.
func applyFile(root string, call contract.ToolCall) (*model.FileMutation, error) {
	target := filepath.Join(root, filepath.FromSlash(call.Path))
	old, readErr := os.ReadFile(target)
	m := &model.FileMutation{Path: call.Path, LinesRemoved: countLines(string(old))}
	if call.Tool == contract.ToolDeleteFile {
		m.Action = "deleted"
		return m, os.Remove(target)
	}
	m.Action, m.LinesAdded = "modified", countLines(call.Content)
	if readErr != nil {
		m.Action = "created"
	}
	sum := sha256.Sum256([]byte(call.Content))
	m.ContentHash = hex.EncodeToString(sum[:])
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}
	return m, os.WriteFile(target, []byte(call.Content), 0o644)
}
