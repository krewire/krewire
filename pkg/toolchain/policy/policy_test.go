package policy

import (
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/toolchain/contract"
)

const secret = "host-secret"

func authorizer(t *testing.T) *Authorizer {
	t.Helper()
	task := &contract.TaskBrief{ID: "131", Scope: contract.TaskScope{Allowed: []string{"pkg/greet"},
		Prohibited: []string{"pkg/greet/vendor"}, Commands: [][]string{{"go", "generate"}}},
		Approval: contract.ApprovalPolicy{Actions: []model.ActionCategory{model.ActionCommand}, Approvers: []string{"alice"}}}
	ctx := &contract.ProjectContext{Gates: []contract.Gate{{ID: "test", Argv: []string{"go", "test", "./..."}}}}
	return &Authorizer{Root: t.TempDir(), Policy: contract.DefaultPolicy(), Task: task, Context: ctx, Secret: secret}
}

func TestAuthorize(t *testing.T) {
	a := authorizer(t)
	cases := []struct {
		call contract.ToolCall
		code string
	}{
		{contract.ToolCall{Tool: contract.ToolWriteFile, Path: "pkg/greet/a.go"}, ""},
		{contract.ToolCall{Tool: contract.ToolWriteFile, Path: "pkg/other/a.go"}, "scope_violation"},
		{contract.ToolCall{Tool: contract.ToolWriteFile, Path: "pkg/greet/vendor/x"}, "scope_violation"},
		{contract.ToolCall{Tool: contract.ToolWriteFile, Path: "pkg/greet/../../etc"}, "scope_violation"},
		{contract.ToolCall{Tool: contract.ToolDeleteFile, Path: "pkg/greet/a.go"}, "approval_required"},
		{contract.ToolCall{Tool: contract.ToolRunCommand, Argv: []string{"go", "test", "./..."}}, "approval_required"},
		{contract.ToolCall{Tool: contract.ToolRunCommand, Argv: []string{"curl", "x"}}, "scope_violation"},
		{contract.ToolCall{Tool: contract.ToolRunCommand}, "policy_denied"},
		{contract.ToolCall{Tool: contract.ToolFetch, URL: "https://x"}, "policy_denied"},
		{contract.ToolCall{Tool: "shell"}, "policy_denied"},
	}
	for _, c := range cases {
		d := a.Authorize(c.call, nil)
		if d.Code != c.code || d.Allowed != (c.code == "") {
			t.Errorf("%s %s%v: got %+v, want %q", c.call.Tool, c.call.Path, c.call.Argv, d, c.code)
		}
	}
	a.Policy.AllowFileWrite = false
	if d := a.Authorize(contract.ToolCall{Tool: contract.ToolWriteFile, Path: "pkg/greet/a.go"}, nil); d.Code != "policy_denied" {
		t.Errorf("writes disabled: %+v", d)
	}
	a.Policy.AllowNetwork, a.Task.Limits.AllowNetwork = true, true
	if d := a.Authorize(contract.ToolCall{Tool: contract.ToolFetch, URL: "https://x"}, nil); !d.Allowed {
		t.Errorf("network enabled by policy and task: %+v", d)
	}
}

func TestDestructiveCommandDenied(t *testing.T) {
	a := authorizer(t)
	a.Task.Scope.Commands = [][]string{{"rm"}}
	if d := a.Authorize(contract.ToolCall{Tool: contract.ToolRunCommand, Argv: []string{"rm", "-rf", "/"}}, nil); d.Code != "policy_denied" {
		t.Errorf("destructive command: %+v", d)
	}
}

func TestApprovals(t *testing.T) {
	a := authorizer(t)
	call := contract.ToolCall{ID: "c1", Tool: contract.ToolRunCommand, Argv: []string{"go", "generate", "./..."}}
	d := a.Authorize(call, nil)
	if d.Code != "approval_required" || d.ApprovalScope != CallScope("131", call) {
		t.Fatalf("want approval_required with scope, got %+v", d)
	}
	good, err := Sign("alice", d.ApprovalScope, secret, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if d := a.Authorize(call, []contract.ApprovalRecord{good}); !d.Allowed {
		t.Errorf("approved call denied: %+v", d)
	}
	forged := good
	forged.Token = strings.Repeat("0", len(good.Token))
	wrongScope, _ := Sign("alice", PlanScope("131", "other"), secret, time.Minute)
	wrongKey, _ := Sign("alice", d.ApprovalScope, "other-secret", time.Minute)
	stranger, _ := Sign("mallory", d.ApprovalScope, secret, time.Minute)
	for name, rec := range map[string]contract.ApprovalRecord{"forged": forged, "scope": wrongScope, "key": wrongKey, "approver": stranger} {
		if d := a.Authorize(call, []contract.ApprovalRecord{rec}); d.Allowed {
			t.Errorf("%s approval must not verify", name)
		}
	}
	changed := call
	changed.Argv = []string{"go", "generate", "./pkg/..."}
	if d := a.Authorize(changed, []contract.ApprovalRecord{good}); d.Allowed {
		t.Error("an approval binds to the exact call")
	}
	a.Secret = ""
	if _, ok := a.FindApproval([]contract.ApprovalRecord{good}, d.ApprovalScope); ok {
		t.Error("without a host secret nothing verifies")
	}
	if _, err := Sign("", "s", secret, time.Minute); err == nil {
		t.Error("approver is required")
	}
}

func TestRedact(t *testing.T) {
	out := Redact("token=abc api_key=XYZ123 my-env-secret-value done", 0, "my-env-secret-value", "ab")
	if strings.Contains(out, "XYZ123") || strings.Contains(out, "my-env-secret-value") || !strings.Contains(out, "done") {
		t.Errorf("redaction failed: %q", out)
	}
	if got := Redact(strings.Repeat("x", 100), 10); !strings.HasSuffix(got, "[truncated]") || len(got) > 30 {
		t.Errorf("truncation failed: %q", got)
	}
}
