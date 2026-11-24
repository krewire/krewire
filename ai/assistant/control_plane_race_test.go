package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krewire/krewire/ai/assistant"
	"github.com/krewire/krewire/ai/sdlc"
	"github.com/krewire/krewire/sec"
)

// TestControlPlane_ConcurrentReadersRace is a regression test for the data
// race between the workflow goroutine started by CreateTask and callers that
// read task state (krewire/krewire#130). Run it with -race: readers poll
// snapshots, subscribe/unsubscribe and drive the decision gates while the
// workflow mutates the task.
func TestControlPlane_ConcurrentReadersRace(t *testing.T) {
	cp := assistant.NewControlPlane(assistant.HostCommandRunner{})
	wsID := "ws-race"
	brief := sdlc.TaskBrief{
		ID:            "TASK-RACE-1",
		Title:         "race regression",
		Objective:     "exercise concurrent reads",
		Scope:         []string{"."},
		RequirePlan:   true,
		RequireReview: true,
	}

	created, err := cp.CreateTask(context.Background(), wsID, t.TempDir(), brief, "dev", func(opts *sdlc.WorkflowOptions) {
		opts.PlanFn = func(ctx context.Context, b sdlc.TaskBrief) (*sdlc.Plan, error) {
			return &sdlc.Plan{Summary: "noop", Steps: []sdlc.PlanStep{{ID: "s1", Description: "noop"}}}, nil
		}
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	// Reading the returned value must be safe while the workflow runs.
	_ = created.Status

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				if task, err := cp.GetTask(wsID, brief.ID); err == nil {
					_, _, _ = task.Status, task.Stage, len(task.Activities)
					if task.Plan != nil {
						_ = len(task.Plan.Steps)
					}
					// Verify direct task serialization is safe and invokes snapshotting.
					if data, err := json.Marshal(task); err == nil {
						_ = len(data)
					}
					ch, unsub := task.SubscribeActivities()
					select {
					case <-ch:
					default:
					}
					unsub()
					unsub() // idempotent
				}
				if snap, err := cp.GetTaskSnapshot(wsID, brief.ID); err == nil {
					_, _, _ = snap.Status, snap.Stage, len(snap.Activities)
					if snap.Plan != nil {
						_ = len(snap.Plan.Steps)
					}
					if data, err := json.Marshal(snap); err == nil {
						_ = len(data)
					}
				}
				for _, task := range cp.ListTasks(wsID) {
					_ = task.UpdatedAt
				}
				for _, snap := range cp.ListTaskSnapshots(wsID) {
					_ = snap.UpdatedAt
					if data, err := json.Marshal(snap); err == nil {
						_ = len(data)
					}
				}
			}
		}()
	}

	waitStatus := func(want sdlc.Status) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if task, err := cp.GetTask(wsID, brief.ID); err == nil && task.Status == want {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for status %s", want)
	}

	waitStatus(sdlc.StatusAwaitingPlanApproval)
	if err := cp.DecidePlan(wsID, brief.ID, true, "lead"); err != nil {
		t.Fatalf("DecidePlan: %v", err)
	}
	waitStatus(sdlc.StatusAwaitingReview)
	if err := cp.DecideReview(wsID, brief.ID, true, "lead"); err != nil {
		t.Fatalf("DecideReview: %v", err)
	}
	waitStatus(sdlc.StatusCompleted)

	cancel()
	wg.Wait()

	// Snapshots are isolated from later mutation.
	snap, err := cp.GetTask(wsID, brief.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	before := len(snap.Activities)
	if _, err := cp.GenerateApprovalSignal(wsID, brief.ID, "lead", "0123456789abcdef0123456789abcdef", time.Minute); err != nil {
		t.Fatalf("GenerateApprovalSignal: %v", err)
	}
	if len(snap.Activities) != before {
		t.Fatalf("snapshot mutated after the fact: %d -> %d", before, len(snap.Activities))
	}
	if fresh, _ := cp.GetTask(wsID, brief.ID); len(fresh.Activities) != before+1 {
		t.Fatalf("expected a new activity in a fresh snapshot, got %d (was %d)", len(fresh.Activities), before)
	}
}

func TestControlPlane_TaskSnapshotDeepCopyIsolation(t *testing.T) {
	cp := assistant.NewControlPlane(assistant.HostCommandRunner{})
	wsID := "ws-iso"
	brief := sdlc.TaskBrief{
		ID:                   "TASK-ISO-1",
		Title:                "isolation verification",
		Objective:            "verify deep copy guarantees",
		Scope:                []string{"dir/a"},
		Constraints:          []string{"c1"},
		AcceptanceCriteria:   []string{"a1"},
		VerificationCommands: []string{"echo 1"},
		RequirePlan:          true,
		RequireReview:        true,
	}

	created, err := cp.CreateTask(context.Background(), wsID, t.TempDir(), brief, "dev", func(opts *sdlc.WorkflowOptions) {
		opts.PlanFn = func(ctx context.Context, b sdlc.TaskBrief) (*sdlc.Plan, error) {
			return &sdlc.Plan{
				Summary: "Plan summary",
				Steps: []sdlc.PlanStep{
					{ID: "s1", Description: "Step 1", TargetFiles: []string{"file1.go"}},
				},
			}, nil
		}
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Wait until plan is created
	deadline := time.Now().Add(5 * time.Second)
	var snap *assistant.TaskSnapshot
	for time.Now().Before(deadline) {
		var sErr error
		snap, sErr = cp.GetTaskSnapshot(wsID, brief.ID)
		if sErr == nil && snap != nil && snap.Plan != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if snap == nil || snap.Plan == nil {
		t.Fatalf("timed out waiting for plan snapshot")
	}

	// 1. Mutate returned snapshot fields and verify original in Core is unaffected
	snap.Brief.Scope[0] = "mutated-scope"
	snap.Plan.Steps[0].TargetFiles[0] = "mutated-file.go"
	snap.Plan.Summary = "mutated-summary"

	freshSnap, err := cp.GetTaskSnapshot(wsID, brief.ID)
	if err != nil {
		t.Fatalf("GetTaskSnapshot: %v", err)
	}
	if freshSnap.Brief.Scope[0] == "mutated-scope" {
		t.Errorf("Brief.Scope was mutated through snapshot reference")
	}
	if freshSnap.Plan.Steps[0].TargetFiles[0] == "mutated-file.go" {
		t.Errorf("Plan.Steps.TargetFiles was mutated through snapshot reference")
	}
	if freshSnap.Plan.Summary == "mutated-summary" {
		t.Errorf("Plan.Summary was mutated through snapshot reference")
	}

	// 2. Mutate activities details map if present
	if len(freshSnap.Activities) > 0 && freshSnap.Activities[0].Details != nil {
		freshSnap.Activities[0].Details["mutated_key"] = "bad"
		thirdSnap, _ := cp.GetTaskSnapshot(wsID, brief.ID)
		if _, exists := thirdSnap.Activities[0].Details["mutated_key"]; exists {
			t.Errorf("Activities[0].Details was mutated through snapshot reference")
		}
	}

	// 3. Verify created.Snapshot() returns independent snapshot
	fromCreated := created.Snapshot()
	if fromCreated.ID != brief.ID {
		t.Errorf("expected snapshot ID %s, got %s", brief.ID, fromCreated.ID)
	}
}

func TestControlPlane_ApprovalSignalSecurity(t *testing.T) {
	cp := assistant.NewControlPlane(assistant.HostCommandRunner{})
	wsID := "ws-sig-sec"
	taskID := "TASK-SIG-SEC-1"
	brief := sdlc.TaskBrief{
		ID:        taskID,
		Title:     "approval signal security verification",
		Objective: "verify cryptographic binding and replay protection",
		Scope:     []string{"."},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := cp.CreateTask(ctx, wsID, t.TempDir(), brief, "dev"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	secret := "0123456789abcdef0123456789abcdef"
	diffHash := "sha256:4d6023d8819543e49870f7cfcb7891bb08e7ff9ffba92d0cbb577e1dbab454f2"

	// 1. Generation with full options
	sig, err := cp.GenerateApprovalSignalWithOptions(assistant.ApprovalSignalOptions{
		TenantID:    "ten-alpha",
		WorkspaceID: wsID,
		TaskID:      taskID,
		ApproverID:  "lead-security",
		ActionHash:  diffHash,
		SecretKey:   secret,
		KeyID:       "key-2026-q1",
		TTL:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("GenerateApprovalSignalWithOptions: %v", err)
	}
	if sig.KeyID != "key-2026-q1" || sig.ActionHash != diffHash || sig.Nonce == "" {
		t.Fatalf("unexpected signal fields: %+v", sig)
	}

	// 2. Successful verification
	if err := cp.VerifyApprovalSignalWithOptions(sig, assistant.VerifyApprovalSignalOptions{
		TenantID:           "ten-alpha",
		WorkspaceID:        wsID,
		TaskID:             taskID,
		ExpectedActionHash: diffHash,
		SecretKey:          secret,
	}); err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	// 3. Replay detection: second verification must fail
	if err := cp.VerifyApprovalSignalWithOptions(sig, assistant.VerifyApprovalSignalOptions{
		TenantID:           "ten-alpha",
		WorkspaceID:        wsID,
		TaskID:             taskID,
		ExpectedActionHash: diffHash,
		SecretKey:          secret,
	}); !errors.Is(err, sec.ErrApprovalReplayed) {
		t.Fatalf("expected ErrApprovalReplayed, got %v", err)
	}

	// 4. Insecure former fallback secret rejected
	fallbackSecret := "dash-ephemeral-approval-secret"
	if _, err := cp.GenerateApprovalSignalWithOptions(assistant.ApprovalSignalOptions{
		WorkspaceID: wsID,
		TaskID:      taskID,
		ApproverID:  "attacker",
		SecretKey:   fallbackSecret,
	}); !errors.Is(err, sec.ErrInsecureFallbackSecret) {
		t.Fatalf("expected ErrInsecureFallbackSecret, got %v", err)
	}

	// 5. Verification fails on tampered action hash
	sig2, err := cp.GenerateApprovalSignalWithOptions(assistant.ApprovalSignalOptions{
		WorkspaceID: wsID,
		TaskID:      taskID,
		ApproverID:  "lead-security",
		ActionHash:  diffHash,
		SecretKey:   secret,
		TTL:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("GenerateApprovalSignalWithOptions sig2: %v", err)
	}

	if err := cp.VerifyApprovalSignalWithOptions(sig2, assistant.VerifyApprovalSignalOptions{
		WorkspaceID:        wsID,
		TaskID:             taskID,
		ExpectedActionHash: "sha256:tampered",
		SecretKey:          secret,
	}); !errors.Is(err, sec.ErrApprovalActionHashMismatch) {
		t.Fatalf("expected ErrApprovalActionHashMismatch, got %v", err)
	}

	// 6. Verification with key ring and rotation
	ring, err := sec.NewKeyRing("key-2026-q1", secret)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	sig3, err := cp.GenerateApprovalSignalWithOptions(assistant.ApprovalSignalOptions{
		WorkspaceID: wsID,
		TaskID:      taskID,
		ApproverID:  "lead-security",
		KeyID:       "key-2026-q1",
		SecretKey:   secret,
		TTL:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("sig3 generation: %v", err)
	}

	if err := cp.VerifyApprovalSignalWithOptions(sig3, assistant.VerifyApprovalSignalOptions{
		WorkspaceID: wsID,
		TaskID:      taskID,
		KeyRing:     ring,
	}); err != nil {
		t.Fatalf("verification with key ring failed: %v", err)
	}
}
