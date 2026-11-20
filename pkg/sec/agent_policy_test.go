package sec_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/sec"
)

func TestAuthorizeAgentAction_WorkspaceBoundary(t *testing.T) {
	workspaceDir := t.TempDir()
	policy := sec.DefaultAgentPolicy(workspaceDir)
	// Temporarily remove delete approval requirement to test boundary check
	delete(policy.RequireHumanApproval, sec.ActionFileDelete)

	// 1. Valid write inside workspace
	if err := sec.AuthorizeAgentAction(policy, sec.ActionFileWrite, "pkg/handler.go"); err != nil {
		t.Errorf("expected valid file write to succeed, got: %v", err)
	}

	// 2. Traversal attempt out of workspace
	err := sec.AuthorizeAgentAction(policy, sec.ActionFileWrite, "../../etc/passwd")
	if err == nil {
		t.Fatalf("security violation: agent policy allowed write outside workspace")
	}
	if !errors.Is(err, sec.ErrAgentPolicyDenied) {
		t.Errorf("expected ErrAgentPolicyDenied, got: %v", err)
	}
}

func TestAuthorizeAgentAction_DestructiveCommandsBlocked(t *testing.T) {
	workspaceDir := t.TempDir()
	policy := sec.DefaultAgentPolicy(workspaceDir)

	destructiveCommands := []string{
		"rm -rf /",
		"rm -fr /home/user",
		"git push origin main --force",
		"DROP DATABASE production;",
		"kill -9 1",
	}

	for _, cmd := range destructiveCommands {
		err := sec.AuthorizeAgentAction(policy, sec.ActionRunCommand, cmd)
		if err == nil {
			t.Errorf("security violation: agent policy allowed destructive command: %q", cmd)
		}
		if !errors.Is(err, sec.ErrAgentPolicyDenied) {
			t.Errorf("expected ErrAgentPolicyDenied for %q, got: %v", cmd, err)
		}
	}

	// Non-destructive commands allowed
	if err := sec.AuthorizeAgentAction(policy, sec.ActionRunCommand, "go test ./..."); err != nil {
		t.Errorf("expected go test to be allowed, got: %v", err)
	}
}

func TestAuthorizeAgentAction_HumanApprovalRequired(t *testing.T) {
	workspaceDir := t.TempDir()
	policy := sec.DefaultAgentPolicy(workspaceDir)

	// High-impact actions require approval
	for _, act := range []sec.AgentAction{sec.ActionGitPush, sec.ActionDeploy} {
		err := sec.AuthorizeAgentAction(policy, act, "production")
		if err == nil {
			t.Errorf("security violation: high-impact action %s did not require approval", act)
		}
		if !errors.Is(err, sec.ErrApprovalRequired) {
			t.Errorf("expected ErrApprovalRequired for %s, got: %v", act, err)
		}
	}
}

func TestRedactSensitiveOutput(t *testing.T) {
	input := "Connected with api_key='sk-1234567890abcdef' and Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test"
	redacted := sec.RedactSensitiveOutput(input)

	if strings.Contains(redacted, "sk-1234567890abcdef") {
		t.Errorf("API key was not redacted: %s", redacted)
	}
	if strings.Contains(redacted, "eyJhbGciOiJIUzI1NiJ9") {
		t.Errorf("Bearer token was not redacted: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_SECRET]") {
		t.Errorf("expected redaction marker in output: %s", redacted)
	}
}
