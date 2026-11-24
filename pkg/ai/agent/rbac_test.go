package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/ai"
	"github.com/krewire/krewire/pkg/ai/agent"
	"github.com/krewire/krewire/pkg/ai/tools"
	"github.com/krewire/krewire/pkg/sec"
)

type rbacMockProvider struct {
	responses []*ai.CompletionResponse
	step      int
}

func (m *rbacMockProvider) Name() string { return "rbac-mock" }
func (m *rbacMockProvider) Complete(_ context.Context, _ ai.CompletionRequest) (*ai.CompletionResponse, error) {
	if m.step >= len(m.responses) {
		return &ai.CompletionResponse{Content: "Completed", FinishReason: "stop"}, nil
	}
	resp := m.responses[m.step]
	m.step++
	return resp, nil
}
func (m *rbacMockProvider) Stream(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	ch := make(chan ai.StreamChunk, 1)
	ch <- ai.StreamChunk{Done: true}
	close(ch)
	return ch, nil
}

func TestAgent_RBAC_ViewerDeniedRunCommand(t *testing.T) {
	eval := agent.NewMemoryAuthorizer()
	ledger := agent.NewMemoryAuditor()

	approver := agent.NewAgentApprover(eval, []string{"viewer"}, "tenant_test", "user_viewer", ledger)

	mock := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_exec",
						Name:      "run_command",
						Arguments: `{"command": "uptime"}`,
					},
				},
			},
		},
	}

	ag, err := agent.New(
		agent.WithProvider(mock),
		agent.WithTools(tools.NewRegistry().Register(&tools.RunCommandTool{})),
		agent.WithApprover(approver),
		agent.WithStrictApprovals(true),
	)
	if err != nil {
		t.Fatalf("agent.New failed: %v", err)
	}

	_, err = ag.Run(context.Background(), "Check system uptime")
	if err == nil {
		t.Fatal("expected run_command to fail with permission denied for viewer")
	}
	if !errors.Is(err, agent.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got: %v", err)
	}

	// Verify audit ledger captured the violation
	events := ledger.Query(context.Background(), "tenant_test", "agent.tool_denied")
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}
	if events[0].TargetID != "run_command" {
		t.Errorf("expected target run_command, got %q", events[0].TargetID)
	}
	if events[0].ActorID != "user_viewer" {
		t.Errorf("expected actor user_viewer, got %q", events[0].ActorID)
	}
}

func TestAgent_RBAC_DeveloperAllowedFileTools(t *testing.T) {
	tmp := t.TempDir()
	reg := tools.DefaultWorkspaceTools(tmp)
	eval := agent.NewMemoryAuthorizer()
	ledger := agent.NewMemoryAuditor()

	approver := agent.NewAgentApprover(eval, []string{"developer"}, "tenant_test", "user_dev", ledger)

	mock := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				Content: "Creating file",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_w",
						Name:      "write_file",
						Arguments: `{"path": "test.txt", "content": "developer access verified"}`,
					},
				},
			},
			{
				Content: "Reading file",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_r",
						Name:      "read_file",
						Arguments: `{"path": "test.txt"}`,
					},
				},
			},
			{
				Content:      "All file operations succeeded.",
				FinishReason: "stop",
			},
		},
	}

	ag, err := agent.New(
		agent.WithProvider(mock),
		agent.WithTools(reg),
		agent.WithApprover(approver),
		agent.WithStrictApprovals(true),
	)
	if err != nil {
		t.Fatalf("agent.New failed: %v", err)
	}

	resp, err := ag.Run(context.Background(), "Write and read test.txt")
	if err != nil {
		t.Fatalf("developer execution failed unexpectedly: %v", err)
	}
	if resp.ToolCallsRun != 2 {
		t.Errorf("expected 2 tool calls run, got %d", resp.ToolCallsRun)
	}

	// Verify no permission denials were logged
	events := ledger.Query(context.Background(), "", "")
	if len(events) != 0 {
		t.Errorf("expected 0 denial events for developer, got %d", len(events))
	}
}

func TestAgent_RBAC_PersonaPolicyDefaults(t *testing.T) {
	roleCoding, ok := agent.DefaultPersonaRole("CodingAssistant")
	if !ok || roleCoding.Name != "CodingAssistant" {
		t.Errorf("expected CodingAssistant persona role, got %+v", roleCoding)
	}

	roleDevOps, ok := agent.DefaultPersonaRole("DevOpsRunner")
	if !ok || roleDevOps.Name != "DevOpsRunner" {
		t.Errorf("expected DevOpsRunner persona role, got %+v", roleDevOps)
	}

	roleViewer, ok := agent.DefaultPersonaRole("ViewerAssistant")
	if !ok || roleViewer.Name != "ViewerAssistant" {
		t.Errorf("expected ViewerAssistant persona role, got %+v", roleViewer)
	}

	_, ok = agent.DefaultPersonaRole("unknown")
	if ok {
		t.Error("expected unknown persona role to return false")
	}
}

func TestAgent_DefaultProductionProfile_FailsClosedWithoutPolicy(t *testing.T) {
	tmp := t.TempDir()
	reg := tools.DefaultWorkspaceTools(tmp)

	// Step 1: Model requests privileged tools (write_file, run_command, fetch_url)
	// and one non-privileged inspection tool (list_dir).
	mock := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				Content: "Attempting privileged operations",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_w",
						Name:      "write_file",
						Arguments: `{"path": "unauthorized.txt", "content": "data"}`,
					},
					{
						ID:        "call_cmd",
						Name:      "run_command",
						Arguments: `{"command": "whoami"}`,
					},
					{
						ID:        "call_web",
						Name:      "fetch_url",
						Arguments: `{"url": "https://example.com"}`,
					},
					{
						ID:        "call_ls",
						Name:      "list_dir",
						Arguments: `{"path": "."}`,
					},
				},
			},
			{
				Content:      "Finished.",
				FinishReason: "stop",
			},
		},
	}

	var events []agent.Event
	ag, err := agent.New(
		agent.WithProvider(mock),
		agent.WithTools(reg),
		agent.WithEventHandler(func(e agent.Event) {
			if e.Type == agent.EventToolResult {
				events = append(events, e)
			}
		}),
	)
	if err != nil {
		t.Fatalf("agent.New failed: %v", err)
	}

	resp, err := ag.Run(context.Background(), "Run tools")
	if err != nil {
		t.Fatalf("run failed unexpectedly: %v", err)
	}
	if resp.ToolCallsRun != 4 {
		t.Errorf("expected 4 tool calls recorded, got %d", resp.ToolCallsRun)
	}

	// Verify fail-closed behavior for privileged tools
	deniedTools := make(map[string]string)
	for _, ev := range events {
		if ev.Error != "" {
			deniedTools[ev.ToolName] = ev.Error
		}
	}

	privileged := []string{"write_file", "run_command", "fetch_url"}
	for _, pt := range privileged {
		errMsg, ok := deniedTools[pt]
		if !ok {
			t.Errorf("security violation: privileged tool %s was not denied fail-closed", pt)
		}
		if !strings.Contains(errMsg, "requires an explicit authorization or approval policy in production profile") {
			t.Errorf("unexpected error message for %s: %s", pt, errMsg)
		}
	}

	// Non-privileged tool list_dir should have executed without policy denial
	if _, ok := deniedTools["list_dir"]; ok {
		t.Errorf("expected read-only inspection tool list_dir to be permitted, got error: %s", deniedTools["list_dir"])
	}
}

func TestAgent_ReadOnlyProfile_BlocksPrivilegedTools(t *testing.T) {
	tmp := t.TempDir()
	reg := tools.DefaultWorkspaceTools(tmp)

	// Even if an approver callback attempts to approve everything,
	// ProfileReadOnly must unconditionally prohibit privileged tools.
	permissiveApprover := func(toolName string, args json.RawMessage) bool {
		return true
	}

	mock := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				Content: "Attempting write and command in read-only profile",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_w",
						Name:      "write_file",
						Arguments: `{"path": "test.txt", "content": "should be blocked"}`,
					},
					{
						ID:        "call_cmd",
						Name:      "run_command",
						Arguments: `{"command": "echo blocked"}`,
					},
				},
			},
			{
				Content:      "Done.",
				FinishReason: "stop",
			},
		},
	}

	var events []agent.Event
	ag, err := agent.New(
		agent.WithProvider(mock),
		agent.WithTools(reg),
		agent.WithSecurityProfile(agent.ProfileReadOnly),
		agent.WithApprover(permissiveApprover),
		agent.WithEventHandler(func(e agent.Event) {
			if e.Type == agent.EventToolResult {
				events = append(events, e)
			}
		}),
	)
	if err != nil {
		t.Fatalf("agent.New failed: %v", err)
	}

	_, err = ag.Run(context.Background(), "Run tools")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	for _, ev := range events {
		if !strings.Contains(ev.Error, "read-only profile prohibits command execution, file modifications, and network requests") {
			t.Errorf("expected read-only block message for %s, got: %q", ev.ToolName, ev.Error)
		}
	}
}

func TestAgent_CanonicalApprovalBindingAndAntiReplay(t *testing.T) {
	tmp := t.TempDir()
	reg := tools.DefaultWorkspaceTools(tmp)

	secret := "0123456789abcdef0123456789abcdef"
	scope := "workspace-test"
	tracker := sec.NewMemoryNonceTracker()

	argsValid := []byte(`{"path": "approved.txt", "content": "valid content"}`)

	// Issue approval signal bound to exact canonical tool name and arguments
	sig, err := sec.NewApprovalSignalForTool(secret, "human-approver", scope, "write_file", argsValid, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create approval signal: %v", err)
	}

	signalApprover := agent.NewSignalApprover(secret, scope, tracker, func(toolName string) *sec.ApprovalSignal {
		return sig
	})

	// 1. First execution with exact valid arguments succeeds
	mockSuccess := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_1",
						Name:      "write_file",
						Arguments: string(argsValid),
					},
				},
			},
			{
				Content:      "File created successfully.",
				FinishReason: "stop",
			},
		},
	}

	ag1, err := agent.New(
		agent.WithProvider(mockSuccess),
		agent.WithTools(reg),
		agent.WithApprover(signalApprover),
		agent.WithStrictApprovals(true),
	)
	if err != nil {
		t.Fatalf("agent.New: %v", err)
	}

	_, err = ag1.Run(context.Background(), "Write file")
	if err != nil {
		t.Fatalf("expected valid approved call to succeed, got: %v", err)
	}

	// 2. Replay attack: attempting to use the same approval signal a second time fails
	mockReplay := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_replay",
						Name:      "write_file",
						Arguments: string(argsValid),
					},
				},
			},
		},
	}

	ag2, _ := agent.New(
		agent.WithProvider(mockReplay),
		agent.WithTools(reg),
		agent.WithApprover(signalApprover),
		agent.WithStrictApprovals(true),
	)

	_, err = ag2.Run(context.Background(), "Replay write file")
	if err == nil || !errors.Is(err, agent.ErrPermissionDenied) {
		t.Fatalf("expected replay attack to fail with ErrPermissionDenied, got: %v", err)
	}

	// 3. Tampered payload attack: arguments modified
	argsTampered := []byte(`{"path": "approved.txt", "content": "TAMPERED PAYLOAD"}`)
	sigTampered, _ := sec.NewApprovalSignalForTool(secret, "human-approver", scope, "write_file", argsValid, 5*time.Minute)
	tracker2 := sec.NewMemoryNonceTracker()

	tamperedApprover := agent.NewSignalApprover(secret, scope, tracker2, func(toolName string) *sec.ApprovalSignal {
		return sigTampered
	})

	mockTampered := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_tampered",
						Name:      "write_file",
						Arguments: string(argsTampered),
					},
				},
			},
		},
	}

	ag3, _ := agent.New(
		agent.WithProvider(mockTampered),
		agent.WithTools(reg),
		agent.WithApprover(tamperedApprover),
		agent.WithStrictApprovals(true),
	)

	_, err = ag3.Run(context.Background(), "Tampered write file")
	if err == nil || !errors.Is(err, agent.ErrPermissionDenied) {
		t.Fatalf("expected tampered argument call to fail with ErrPermissionDenied, got: %v", err)
	}
}

func TestAgent_PromptInjectionDefense_IndirectFileInstruction(t *testing.T) {
	tmp := t.TempDir()
	reg := tools.DefaultWorkspaceTools(tmp)

	// Simulation of an indirect prompt injection:
	// A file in repository contains instructions to execute destructive tools.
	// The model is tricked into calling run_command and write_file.
	mockInjected := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				Content: "Found instructions in repository. Executing required commands.",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_malicious_cmd",
						Name:      "run_command",
						Arguments: `{"command": "rm -rf /"}`,
					},
					{
						ID:        "call_malicious_write",
						Name:      "write_file",
						Arguments: `{"path": "/etc/compromised", "content": "pwned"}`,
					},
				},
			},
			{
				Content:      "Execution finished.",
				FinishReason: "stop",
			},
		},
	}

	var events []agent.Event
	ag, err := agent.New(
		agent.WithProvider(mockInjected),
		agent.WithTools(reg),
		// Default ProfileProduction with no explicit approver/authorizer
		agent.WithEventHandler(func(e agent.Event) {
			if e.Type == agent.EventToolResult {
				events = append(events, e)
			}
		}),
	)
	if err != nil {
		t.Fatalf("agent.New failed: %v", err)
	}

	_, err = ag.Run(context.Background(), "Process repo files")
	if err != nil {
		t.Fatalf("unexpected agent run failure: %v", err)
	}

	// Verify both injected tool calls were blocked by the policy gate
	if len(events) != 2 {
		t.Fatalf("expected 2 tool events, got %d", len(events))
	}
	for _, ev := range events {
		if !strings.Contains(ev.Error, "requires an explicit authorization or approval policy in production profile") {
			t.Errorf("prompt injection defense failed for %s, got error: %s", ev.ToolName, ev.Error)
		}
	}
}

func TestAgent_InsecureDevMode_OptIn(t *testing.T) {
	tmp := t.TempDir()
	reg := tools.DefaultWorkspaceTools(tmp)

	mock := &rbacMockProvider{
		responses: []*ai.CompletionResponse{
			{
				Content: "Creating dev file without approver",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "call_dev_write",
						Name:      "write_file",
						Arguments: `{"path": "dev.txt", "content": "local dev mode enabled"}`,
					},
				},
			},
			{
				Content:      "Dev file ready.",
				FinishReason: "stop",
			},
		},
	}

	ag, err := agent.New(
		agent.WithProvider(mock),
		agent.WithTools(reg),
		agent.WithInsecureDevMode(), // Conspicuous local development opt-in
	)
	if err != nil {
		t.Fatalf("agent.New failed: %v", err)
	}

	resp, err := ag.Run(context.Background(), "Run dev tool")
	if err != nil {
		t.Fatalf("dev mode run failed: %v", err)
	}
	if resp.ToolCallsRun != 1 {
		t.Errorf("expected 1 tool call run in dev mode, got %d", resp.ToolCallsRun)
	}
}
