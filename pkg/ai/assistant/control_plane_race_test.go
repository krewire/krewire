package assistant_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/ai/assistant"
	"github.com/krewire/krewire/pkg/ai/sdlc"
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
					ch, unsub := task.SubscribeActivities()
					select {
					case <-ch:
					default:
					}
					unsub()
					unsub() // idempotent
				}
				for _, task := range cp.ListTasks(wsID) {
					_ = task.UpdatedAt
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
