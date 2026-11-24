package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/krewire/krewire/sec"
)

const (
	// DefaultCommandTimeout is the default duration for command execution when unspecified.
	DefaultCommandTimeout = 30 * time.Second

	// MaxCommandTimeout is the maximum permitted timeout ceiling for command execution.
	MaxCommandTimeout = 10 * time.Minute

	// MaxCommandOutputBytes is the maximum captured output bytes (stdout/stderr) to prevent memory exhaustion.
	MaxCommandOutputBytes = 512 * 1024
)

// boundedBuffer captures output up to a maximum byte capacity, safely truncating excess bytes.
type boundedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	if limit <= 0 {
		limit = MaxCommandOutputBytes
	}
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if b.buf.Len() >= b.limit {
		b.truncated = true
		return n, nil
	}
	remaining := b.limit - b.buf.Len()
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
	} else {
		b.buf.Write(p)
	}
	return n, nil
}

func (b *boundedBuffer) String() string {
	res := b.buf.String()
	if b.truncated {
		res += "\n... [output truncated: exceeded limit]"
	}
	return res
}

// RunCommandTool allows executing shell commands with strict timeouts, sanitized environments, and scrubbed outputs.
type RunCommandTool struct {
	Workspace      string
	DefaultTimeout time.Duration
	AllowEnv       []string          // Explicit variable keys allowed to be inherited from parent
	Env            map[string]string // Explicit non-secret environment overrides
	Policy         *sec.AgentPolicy  // Optional security policy for permission checks
}

func (t *RunCommandTool) Name() string { return "run_command" }
func (t *RunCommandTool) Description() string {
	return "Execute a shell command in the workspace directory with timeout and sanitized environment."
}
func (t *RunCommandTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The exact shell command to run",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Optional timeout in seconds (default 30s)",
			},
			"env": map[string]any{
				"type":        "object",
				"description": "Optional explicit non-secret environment overrides",
			},
		},
		"required": []string{"command"},
	}
}

func (t *RunCommandTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input struct {
		Command        string            `json:"command"`
		TimeoutSeconds int               `json:"timeout_seconds"`
		Env            map[string]string `json:"env,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(input.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}

	// 1. Evaluate AgentPolicy if configured
	if t.Policy != nil {
		if err := sec.AuthorizeAgentAction(*t.Policy, sec.ActionRunCommand, input.Command); err != nil {
			return nil, err
		}
	}

	timeout := t.DefaultTimeout
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	if input.TimeoutSeconds > 0 {
		timeout = time.Duration(input.TimeoutSeconds) * time.Second
	}
	if timeout > MaxCommandTimeout {
		timeout = MaxCommandTimeout
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 2. Build minimal, sanitized child environment
	mergedOverrides := make(map[string]string)
	for k, v := range t.Env {
		mergedOverrides[k] = v
	}
	for k, v := range input.Env {
		mergedOverrides[k] = v
	}

	safeEnv, err := sec.SanitizeChildEnv(sec.EnvConfig{
		Allowlist: t.AllowEnv,
		Overrides: mergedOverrides,
		Workspace: t.Workspace,
	})
	if err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	// #nosec G204 — explicitly designed for AI Agent tool execution
	cmd := exec.CommandContext(cmdCtx, "sh", "-c", input.Command)
	cmd.Env = safeEnv

	if t.Workspace != "" {
		cleanWorkspace, err := sec.SafePath(t.Workspace, ".")
		if err != nil {
			return nil, fmt.Errorf("invalid workspace: %w", err)
		}
		cmd.Dir = cleanWorkspace
	}

	stdout := newBoundedBuffer(MaxCommandOutputBytes)
	stderr := newBoundedBuffer(MaxCommandOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()

	outStr := sec.RedactSensitiveOutput(stdout.String())
	errStr := sec.RedactSensitiveOutput(stderr.String())

	resp := map[string]any{
		"stdout": outStr,
		"stderr": errStr,
	}

	if err != nil {
		resp["error"] = sec.RedactSensitiveOutput(err.Error())
		if cmdCtx.Err() == context.DeadlineExceeded {
			resp["error"] = fmt.Sprintf("command timed out after %v", timeout)
		}
	} else {
		resp["exit_code"] = 0
	}

	return resp, nil
}
