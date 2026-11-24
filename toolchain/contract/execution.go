package contract

import (
	"time"

	"github.com/krewire/krewire/model"
)

// Policy is the host-side execution policy. Engines never evaluate it; the
// toolchain authorizes every tool call against it before execution.
type Policy struct {
	TypeMeta
	AllowFileWrite  bool `json:"allow_file_write"`
	AllowNetwork    bool `json:"allow_network"`
	DenyDestructive bool `json:"deny_destructive"`
	// ApprovalRequired lists categories that always need approval,
	// regardless of the task brief.
	ApprovalRequired []model.ActionCategory `json:"approval_required,omitempty"`
	MaxToolCalls     int                    `json:"max_tool_calls"`
	ToolTimeoutSec   int                    `json:"tool_timeout_sec"`
	MaxOutputBytes   int                    `json:"max_output_bytes"`
}

// DefaultPolicy is the defensive baseline: writes inside scope, no network,
// destructive commands denied, push/deploy/delete need approval.
func DefaultPolicy() Policy {
	return Policy{
		TypeMeta:         Meta(KindPolicy),
		AllowFileWrite:   true,
		DenyDestructive:  true,
		ApprovalRequired: []model.ActionCategory{model.ActionDelete, model.ActionGitPush, model.ActionDeploy},
		MaxToolCalls:     64,
		ToolTimeoutSec:   300,
		MaxOutputBytes:   64 << 10,
	}
}

// ToolName is a tool an engine may ask the host to execute.
type ToolName string

// Host tools. Engines read the workspace through the context and the files
// themselves; every mutation goes through one of these tools.
const (
	ToolWriteFile  ToolName = "write_file"
	ToolDeleteFile ToolName = "delete_file"
	ToolRunCommand ToolName = "run_command"
	ToolFetch      ToolName = "network_fetch"
)

// ToolCall is an engine's request for the host to perform one action.
type ToolCall struct {
	ID     string   `json:"id"`
	StepID string   `json:"step_id,omitempty"`
	Tool   ToolName `json:"tool"`
	Path   string   `json:"path,omitempty"`
	// Content is the full new file content for write_file. Execution
	// records keep only ContentSHA256.
	Content       string   `json:"content,omitempty"`
	ContentSHA256 string   `json:"content_sha256,omitempty"`
	Argv          []string `json:"argv,omitempty"`
	URL           string   `json:"url,omitempty"`
	Rationale     string   `json:"rationale,omitempty"`
}

// Category maps the tool onto the shared action taxonomy in pkg/model.
func (t ToolCall) Category() model.ActionCategory {
	switch t.Tool {
	case ToolWriteFile:
		return model.ActionWrite
	case ToolDeleteFile:
		return model.ActionDelete
	case ToolRunCommand:
		if len(t.Argv) >= 2 && t.Argv[0] == "git" && t.Argv[1] == "push" {
			return model.ActionGitPush
		}
		return model.ActionCommand
	case ToolFetch:
		return model.ActionNetwork
	}
	return model.ActionCategory("UNKNOWN")
}

// Decision is the host's authorization verdict for one tool call.
type Decision struct {
	Allowed bool   `json:"allowed"`
	Code    string `json:"code,omitempty"` // Code.String() when denied
	Reason  string `json:"reason"`
	// ApprovalScope is the scope string an approval must be bound to.
	ApprovalScope string `json:"approval_scope,omitempty"`
}

// ApprovalRecord is a human (or control-surface) authorization, carried as
// an HMAC-authenticated pkg/sec.ApprovalSignal and bound to a scope string.
// Token is omitted from reports; Verified records the host's check.
type ApprovalRecord struct {
	TypeMeta
	ApproverID string    `json:"approver_id"`
	Scope      string    `json:"scope"`
	ApprovedAt time.Time `json:"approved_at"`
	ValidUntil time.Time `json:"valid_until"`
	Token      string    `json:"token,omitempty"`
	Verified   bool      `json:"verified"`
}

// ExecStatus is the outcome of a host-executed tool call.
type ExecStatus string

// Execution statuses.
const (
	ExecApplied ExecStatus = "applied"
	ExecDenied  ExecStatus = "denied"
	ExecFailed  ExecStatus = "failed"
)

// ExecutionRecord is what the host actually did for one tool call.
type ExecutionRecord struct {
	TypeMeta
	Call       ToolCall            `json:"call"`
	Decision   Decision            `json:"decision"`
	Status     ExecStatus          `json:"status"`
	ExitCode   int                 `json:"exit_code"`
	DurationMs int64               `json:"duration_ms"`
	Output     string              `json:"output,omitempty"` // redacted, truncated
	Mutation   *model.FileMutation `json:"mutation,omitempty"`
}
