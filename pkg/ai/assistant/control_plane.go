package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/pkg/ai/sdlc"
	"github.com/krewire/krewire/pkg/sec"
)

var (
	ErrTaskNotFound = errors.New("control plane: task not found")
	ErrTaskHalted   = errors.New("control plane: task has already completed or halted")
)

// ActivityRecord represents an auditable execution, decision, or check event.
type ActivityRecord struct {
	ID         string         `json:"id"`
	Timestamp  time.Time      `json:"timestamp"`
	Type       string         `json:"type"` // "stage_transition", "plan_created", "plan_decided", "tool_call", "verification_check", "review_created", "review_decided", "task_status"
	Actor      string         `json:"actor"`
	IsVerified bool           `json:"is_verified"` // true for verified system checks and human approvals; false for unverified model suggestions
	Label      string         `json:"label"`       // "Verified System Evidence" vs "AI Model Suggestion"
	Summary    string         `json:"summary"`
	Details    map[string]any `json:"details,omitempty"`
}

// EngineeringTask is a supervised SDLC engineering workflow within a workspace.
//
// The workflow mutates a task from its own goroutine, so the control plane
// never hands out the live record: CreateTask, GetTask and ListTasks return
// point-in-time snapshots (copy-on-read) whose exported fields are safe to
// read without synchronization. Methods called on a snapshot (for example
// SubscribeActivities) are forwarded to the live task.
type EngineeringTask struct {
	mu          sync.RWMutex
	ID          string                    `json:"id"`
	WorkspaceID string                    `json:"workspace_id"`
	Title       string                    `json:"title"`
	Brief       sdlc.TaskBrief            `json:"brief"`
	Stage       sdlc.Stage                `json:"stage"`
	Status      sdlc.Status               `json:"status"`
	Plan        *sdlc.Plan                `json:"plan,omitempty"`
	Diffs       []sdlc.FileDiff           `json:"diffs,omitempty"`
	Evidence    []sdlc.VerificationResult `json:"evidence,omitempty"`
	Review      *sdlc.ReviewSummary       `json:"review,omitempty"`
	Provenance  sdlc.Provenance           `json:"provenance"`
	Activities  []ActivityRecord          `json:"activities"`
	Error       string                    `json:"error,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`

	workflow       *sdlc.Workflow
	planDecision   chan decision
	reviewDecision chan decision
	listeners      map[chan ActivityRecord]struct{}

	// live points at the authoritative task when this value is a snapshot.
	live *EngineeringTask
}

// TaskSnapshot is an immutable, concurrency-safe read model of an EngineeringTask.
// All nested slices, maps, and structures are deep-copied under the task read lock,
// guaranteeing that subsequent mutations by workflow goroutines cannot cause data races
// during reads or JSON serialization.
type TaskSnapshot struct {
	ID          string                    `json:"id"`
	WorkspaceID string                    `json:"workspace_id"`
	Title       string                    `json:"title"`
	Brief       sdlc.TaskBrief            `json:"brief"`
	Stage       sdlc.Stage                `json:"stage"`
	Status      sdlc.Status               `json:"status"`
	Plan        *sdlc.Plan                `json:"plan,omitempty"`
	Diffs       []sdlc.FileDiff           `json:"diffs,omitempty"`
	Evidence    []sdlc.VerificationResult `json:"evidence,omitempty"`
	Review      *sdlc.ReviewSummary       `json:"review,omitempty"`
	Provenance  sdlc.Provenance           `json:"provenance"`
	Activities  []ActivityRecord          `json:"activities"`
	Error       string                    `json:"error,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

func cloneBrief(b sdlc.TaskBrief) sdlc.TaskBrief {
	cp := b
	if b.Scope != nil {
		cp.Scope = append([]string(nil), b.Scope...)
	}
	if b.Constraints != nil {
		cp.Constraints = append([]string(nil), b.Constraints...)
	}
	if b.AcceptanceCriteria != nil {
		cp.AcceptanceCriteria = append([]string(nil), b.AcceptanceCriteria...)
	}
	if b.VerificationCommands != nil {
		cp.VerificationCommands = append([]string(nil), b.VerificationCommands...)
	}
	return cp
}

func clonePlan(p *sdlc.Plan) *sdlc.Plan {
	if p == nil {
		return nil
	}
	cp := *p
	if p.Steps != nil {
		cp.Steps = make([]sdlc.PlanStep, len(p.Steps))
		for i, step := range p.Steps {
			stepCopy := step
			if step.TargetFiles != nil {
				stepCopy.TargetFiles = append([]string(nil), step.TargetFiles...)
			}
			cp.Steps[i] = stepCopy
		}
	}
	return &cp
}

func cloneReview(r *sdlc.ReviewSummary) *sdlc.ReviewSummary {
	if r == nil {
		return nil
	}
	cp := *r
	if r.VerificationEvidence != nil {
		cp.VerificationEvidence = append([]sdlc.VerificationResult(nil), r.VerificationEvidence...)
	}
	if r.Risks != nil {
		cp.Risks = append([]string(nil), r.Risks...)
	}
	if r.UnresolvedQuestions != nil {
		cp.UnresolvedQuestions = append([]string(nil), r.UnresolvedQuestions...)
	}
	return &cp
}

func cloneProvenance(p sdlc.Provenance) sdlc.Provenance {
	cp := p
	if p.ToolsUsed != nil {
		cp.ToolsUsed = append([]string(nil), p.ToolsUsed...)
	}
	if p.VerificationEvidence != nil {
		cp.VerificationEvidence = append([]sdlc.VerificationResult(nil), p.VerificationEvidence...)
	}
	if p.CreatedFiles != nil {
		cp.CreatedFiles = append([]string(nil), p.CreatedFiles...)
	}
	if p.ModifiedFiles != nil {
		cp.ModifiedFiles = append([]string(nil), p.ModifiedFiles...)
	}
	return cp
}

func cloneActivities(acts []ActivityRecord) []ActivityRecord {
	if acts == nil {
		return nil
	}
	cp := make([]ActivityRecord, len(acts))
	for i, act := range acts {
		actCopy := act
		if act.Details != nil {
			actCopy.Details = cloneDetails(act.Details)
		}
		cp[i] = actCopy
	}
	return cp
}

func cloneDetails(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = cloneValue(v)
	}
	return cp
}

func cloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return cloneDetails(val)
	case []string:
		return append([]string(nil), val...)
	case []sdlc.PlanStep:
		steps := make([]sdlc.PlanStep, len(val))
		for i, s := range val {
			sCopy := s
			if s.TargetFiles != nil {
				sCopy.TargetFiles = append([]string(nil), s.TargetFiles...)
			}
			steps[i] = sCopy
		}
		return steps
	case []sdlc.VerificationResult:
		return append([]sdlc.VerificationResult(nil), val...)
	case []sdlc.FileDiff:
		return append([]sdlc.FileDiff(nil), val...)
	case []any:
		slice := make([]any, len(val))
		for i, item := range val {
			slice[i] = cloneValue(item)
		}
		return slice
	case *sdlc.Plan:
		return clonePlan(val)
	case sdlc.Plan:
		p := clonePlan(&val)
		if p != nil {
			return *p
		}
		return val
	case *sdlc.ReviewSummary:
		return cloneReview(val)
	case sdlc.ReviewSummary:
		r := cloneReview(&val)
		if r != nil {
			return *r
		}
		return val
	default:
		return val
	}
}

// target resolves a snapshot to its live task.
func (task *EngineeringTask) target() *EngineeringTask {
	if task.live != nil {
		return task.live
	}
	return task
}

// Snapshot returns an immutable point-in-time snapshot of the task taken
// under its read lock. All nested slices, maps, and pointer types are
// deep-copied so subsequent mutations by workflow goroutines cannot affect
// the snapshot or cause data races during concurrent reads/JSON marshaling.
func (task *EngineeringTask) Snapshot() *TaskSnapshot {
	target := task.target()
	target.mu.RLock()
	defer target.mu.RUnlock()

	var diffs []sdlc.FileDiff
	if target.Diffs != nil {
		diffs = append([]sdlc.FileDiff(nil), target.Diffs...)
	}
	var evidence []sdlc.VerificationResult
	if target.Evidence != nil {
		evidence = append([]sdlc.VerificationResult(nil), target.Evidence...)
	}

	return &TaskSnapshot{
		ID:          target.ID,
		WorkspaceID: target.WorkspaceID,
		Title:       target.Title,
		Brief:       cloneBrief(target.Brief),
		Stage:       target.Stage,
		Status:      target.Status,
		Plan:        clonePlan(target.Plan),
		Diffs:       diffs,
		Evidence:    evidence,
		Review:      cloneReview(target.Review),
		Provenance:  cloneProvenance(target.Provenance),
		Activities:  cloneActivities(target.Activities),
		Error:       target.Error,
		CreatedAt:   target.CreatedAt,
		UpdatedAt:   target.UpdatedAt,
	}
}

// snapshot returns a copy of the task's exported state taken under its read
// lock. Slices and the plan/review values are copied so later mutations by
// the workflow goroutine are not observable through the snapshot.
func (task *EngineeringTask) snapshot() *EngineeringTask {
	snap := task.Snapshot()
	return &EngineeringTask{
		ID:          snap.ID,
		WorkspaceID: snap.WorkspaceID,
		Title:       snap.Title,
		Brief:       snap.Brief,
		Stage:       snap.Stage,
		Status:      snap.Status,
		Plan:        snap.Plan,
		Diffs:       snap.Diffs,
		Evidence:    snap.Evidence,
		Review:      snap.Review,
		Provenance:  snap.Provenance,
		Activities:  snap.Activities,
		Error:       snap.Error,
		CreatedAt:   snap.CreatedAt,
		UpdatedAt:   snap.UpdatedAt,
		live:        task.target(),
	}
}

// MarshalJSON ensures that even if an EngineeringTask is serialized directly,
// an immutable, race-safe snapshot is serialized under read locks rather than
// traversing live internal fields without synchronization.
func (task *EngineeringTask) MarshalJSON() ([]byte, error) {
	return json.Marshal(task.Snapshot())
}

type decision struct {
	approve  bool
	reviewer string
}

// ControlPlane coordinates human-in-the-loop AI workflows across workspaces.
type ControlPlane struct {
	mu     sync.RWMutex
	tasks  map[string]*EngineeringTask // key: wsID + ":" + taskID
	runner CommandRunner
}

// NewControlPlane initializes a human-in-the-loop engineering control plane.
func NewControlPlane(runner CommandRunner) *ControlPlane {
	if runner == nil {
		runner = HostCommandRunner{}
	}
	return &ControlPlane{
		tasks:  make(map[string]*EngineeringTask),
		runner: runner,
	}
}

func taskKey(wsID, taskID string) string {
	return wsID + ":" + taskID
}

// CreateTask registers and initiates a new supervised SDLC task for a workspace.
func (cp *ControlPlane) CreateTask(ctx context.Context, wsID, wsPath string, brief sdlc.TaskBrief, actor string, configure ...func(*sdlc.WorkflowOptions)) (*EngineeringTask, error) {
	if err := brief.Validate(); err != nil {
		return nil, err
	}

	key := taskKey(wsID, brief.ID)
	cp.mu.Lock()
	if _, exists := cp.tasks[key]; exists {
		cp.mu.Unlock()
		return nil, fmt.Errorf("task %q already exists in workspace %q", brief.ID, wsID)
	}

	task := &EngineeringTask{
		ID:             brief.ID,
		WorkspaceID:    wsID,
		Title:          brief.Title,
		Brief:          brief,
		Stage:          sdlc.StageUnderstand,
		Status:         sdlc.StatusInProgress,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		planDecision:   make(chan decision, 1),
		reviewDecision: make(chan decision, 1),
		listeners:      make(map[chan ActivityRecord]struct{}),
		Activities:     make([]ActivityRecord, 0),
	}
	cp.tasks[key] = task
	cp.mu.Unlock()

	// Initial activity record
	task.recordActivity(ActivityRecord{
		ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       "task_status",
		Actor:      actor,
		IsVerified: true,
		Label:      "Verified System Evidence",
		Summary:    fmt.Sprintf("Task %q initiated by %s", brief.ID, actor),
		Details:    map[string]any{"objective": brief.Objective, "scope": brief.Scope},
	})

	executor := func(c context.Context, dir, cmdStr string) (int, string, error) {
		if cp.runner != nil {
			res, err := cp.runner.RunCommand(c, wsID, wsPath, []string{"/bin/sh", "-c", cmdStr}, 60*time.Second)
			exitCode := 0
			output := ""
			if res != nil {
				exitCode = res.ExitCode
				output = res.Stdout
				if res.Stderr != "" {
					output += "\n" + res.Stderr
				}
			}
			if err != nil && exitCode == 0 {
				exitCode = 1
			}
			return exitCode, output, err
		}
		return sdlc.DefaultCommandExecutor(c, dir, cmdStr)
	}

	wfOpts := sdlc.WorkflowOptions{
		Workspace: wsPath,
		Brief:     brief,
		Executor:  executor,
		OnEvent: func(ev sdlc.WorkflowEvent) {
			task.mu.Lock()
			task.Stage = ev.Stage
			task.Status = ev.Status
			task.UpdatedAt = ev.Timestamp
			task.mu.Unlock()

			isVerified := true
			label := "Verified System Evidence"
			summary := fmt.Sprintf("Stage changed to %s (%s)", ev.Stage, ev.Status)
			if ev.Message != "" {
				summary = ev.Message
			}

			task.recordActivity(ActivityRecord{
				ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
				Timestamp:  ev.Timestamp,
				Type:       "stage_transition",
				Actor:      "sdlc_engine",
				IsVerified: isVerified,
				Label:      label,
				Summary:    summary,
				Details:    map[string]any{"stage": ev.Stage, "status": ev.Status},
			})
		},
		PlanApprover: func(ctx context.Context, plan *sdlc.Plan) (bool, string, error) {
			// Keep a private copy: the workflow goroutine later stamps the
			// approval fields on its own plan value.
			planCopy := *plan
			task.mu.Lock()
			task.Plan = &planCopy
			task.Stage = sdlc.StagePlan
			task.Status = sdlc.StatusAwaitingPlanApproval
			task.UpdatedAt = time.Now()
			task.mu.Unlock()

			summary := fmt.Sprintf("Proposed execution plan with %d steps", len(plan.Steps))
			task.recordActivity(ActivityRecord{
				ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
				Timestamp:  time.Now(),
				Type:       "plan_created",
				Actor:      "planner_agent",
				IsVerified: false,
				Label:      "AI Model Suggestion",
				Summary:    summary,
				Details:    map[string]any{"steps": plan.Steps, "summary": plan.Summary},
			})

			select {
			case dec := <-task.planDecision:
				label := "Human Decision Gate"
				actionSummary := fmt.Sprintf("Plan approved by %s", dec.reviewer)
				if !dec.approve {
					actionSummary = fmt.Sprintf("Plan rejected by %s", dec.reviewer)
				}
				task.recordActivity(ActivityRecord{
					ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
					Timestamp:  time.Now(),
					Type:       "plan_decided",
					Actor:      dec.reviewer,
					IsVerified: true,
					Label:      label,
					Summary:    actionSummary,
					Details:    map[string]any{"approved": dec.approve},
				})
				if dec.approve {
					task.mu.Lock()
					approved := *task.Plan
					approved.Approved = true
					approved.ApprovedBy = dec.reviewer
					approved.ApprovedAt = time.Now()
					task.Plan = &approved
					task.mu.Unlock()
				}
				return dec.approve, dec.reviewer, nil
			case <-ctx.Done():
				return false, "", ctx.Err()
			}
		},
		ReviewApprover: func(ctx context.Context, summary *sdlc.ReviewSummary) (bool, string, error) {
			// Private copy: the workflow stamps approval fields on its own value.
			reviewCopy := *summary
			task.mu.Lock()
			task.Review = &reviewCopy
			task.Stage = sdlc.StageReview
			task.Status = sdlc.StatusAwaitingReview
			task.UpdatedAt = time.Now()
			if task.workflow != nil {
				res := task.workflow.Result()
				task.Diffs = res.Diffs
				task.Evidence = res.Evidence
			}
			task.mu.Unlock()

			task.recordActivity(ActivityRecord{
				ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
				Timestamp:  time.Now(),
				Type:       "review_created",
				Actor:      "sdlc_verifier",
				IsVerified: true,
				Label:      "Verified System Evidence",
				Summary:    fmt.Sprintf("Verification complete: %s", summary.ChangeSummary),
				Details:    map[string]any{"evidence": summary.VerificationEvidence, "risks": summary.Risks},
			})

			select {
			case dec := <-task.reviewDecision:
				label := "Human Decision Gate"
				actionSummary := fmt.Sprintf("Changes accepted by %s", dec.reviewer)
				if !dec.approve {
					actionSummary = fmt.Sprintf("Changes rejected by %s", dec.reviewer)
				}
				task.recordActivity(ActivityRecord{
					ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
					Timestamp:  time.Now(),
					Type:       "review_decided",
					Actor:      dec.reviewer,
					IsVerified: true,
					Label:      label,
					Summary:    actionSummary,
					Details:    map[string]any{"approved": dec.approve},
				})
				return dec.approve, dec.reviewer, nil
			case <-ctx.Done():
				return false, "", ctx.Err()
			}
		},
	}

	for _, cfg := range configure {
		cfg(&wfOpts)
	}

	wf, err := sdlc.NewWorkflow(wfOpts)
	if err != nil {
		cp.mu.Lock()
		delete(cp.tasks, key)
		cp.mu.Unlock()
		return nil, fmt.Errorf("failed to initialize workflow: %w", err)
	}

	task.mu.Lock()
	task.workflow = wf
	task.mu.Unlock()

	// Run workflow asynchronously
	go func() {
		res, runErr := wf.Run(context.Background())
		task.mu.Lock()
		task.UpdatedAt = time.Now()
		if res != nil {
			task.Diffs = res.Diffs
			task.Evidence = res.Evidence
			task.Review = res.Review
			task.Provenance = res.Provenance
		}
		if runErr != nil {
			task.Status = sdlc.StatusFailed
			task.Error = runErr.Error()
		} else {
			task.Status = sdlc.StatusCompleted
		}
		finalStatus, finalErr := task.Status, task.Error
		task.mu.Unlock()

		statusSummary := fmt.Sprintf("Workflow ended with status: %s", finalStatus)
		if finalErr != "" {
			statusSummary = fmt.Sprintf("Workflow failed: %s", finalErr)
		}
		task.recordActivity(ActivityRecord{
			ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
			Timestamp:  time.Now(),
			Type:       "task_status",
			Actor:      "sdlc_engine",
			IsVerified: true,
			Label:      "Verified System Evidence",
			Summary:    statusSummary,
			Details:    map[string]any{"status": finalStatus, "error": finalErr},
		})
	}()

	return task.snapshot(), nil
}

// liveTask returns the authoritative task record (internal use only).
func (cp *ControlPlane) liveTask(wsID, taskID string) (*EngineeringTask, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	task, exists := cp.tasks[taskKey(wsID, taskID)]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// GetTask returns a point-in-time snapshot of an existing supervised task.
func (cp *ControlPlane) GetTask(wsID, taskID string) (*EngineeringTask, error) {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return nil, err
	}
	return task.snapshot(), nil
}

// GetTaskSnapshot returns an immutable, concurrency-safe snapshot of an existing supervised task.
func (cp *ControlPlane) GetTaskSnapshot(wsID, taskID string) (*TaskSnapshot, error) {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return nil, err
	}
	return task.Snapshot(), nil
}

// ListTasks lists snapshots of all tasks for a workspace.
func (cp *ControlPlane) ListTasks(wsID string) []*EngineeringTask {
	cp.mu.RLock()
	live := make([]*EngineeringTask, 0)
	for key, task := range cp.tasks {
		if strings.HasPrefix(key, wsID+":") {
			live = append(live, task)
		}
	}
	cp.mu.RUnlock()
	list := make([]*EngineeringTask, 0, len(live))
	for _, task := range live {
		list = append(list, task.snapshot())
	}
	return list
}

// ListTaskSnapshots lists immutable, concurrency-safe snapshots of all tasks for a workspace.
func (cp *ControlPlane) ListTaskSnapshots(wsID string) []*TaskSnapshot {
	cp.mu.RLock()
	live := make([]*EngineeringTask, 0)
	for key, task := range cp.tasks {
		if strings.HasPrefix(key, wsID+":") {
			live = append(live, task)
		}
	}
	cp.mu.RUnlock()
	list := make([]*TaskSnapshot, 0, len(live))
	for _, task := range live {
		list = append(list, task.Snapshot())
	}
	return list
}

// DecidePlan allows a human reviewer to approve or reject an execution plan.
func (cp *ControlPlane) DecidePlan(wsID, taskID string, approve bool, reviewer string) error {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return err
	}
	task.mu.RLock()
	status := task.Status
	task.mu.RUnlock()

	if status != sdlc.StatusAwaitingPlanApproval {
		return fmt.Errorf("task is not awaiting plan approval (status: %s)", status)
	}

	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		reviewer = "human_operator"
	}

	select {
	case task.planDecision <- decision{approve: approve, reviewer: reviewer}:
		return nil
	default:
		return errors.New("plan decision already delivered or channel full")
	}
}

// DecideReview allows a human reviewer to accept or reject the final code review.
func (cp *ControlPlane) DecideReview(wsID, taskID string, approve bool, reviewer string) error {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return err
	}
	task.mu.RLock()
	status := task.Status
	task.mu.RUnlock()

	if status != sdlc.StatusAwaitingReview {
		return fmt.Errorf("task is not awaiting review approval (status: %s)", status)
	}

	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		reviewer = "human_operator"
	}

	select {
	case task.reviewDecision <- decision{approve: approve, reviewer: reviewer}:
		return nil
	default:
		return errors.New("review decision already delivered or channel full")
	}
}

// PauseTask pauses active execution of an engineering task.
func (cp *ControlPlane) PauseTask(wsID, taskID, actor string) error {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return err
	}
	task.mu.Lock()
	if task.workflow == nil {
		task.mu.Unlock()
		return ErrTaskHalted
	}
	task.workflow.Pause()
	task.Status = sdlc.StatusPaused
	task.UpdatedAt = time.Now()
	task.mu.Unlock()

	task.recordActivity(ActivityRecord{
		ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       "task_status",
		Actor:      actor,
		IsVerified: true,
		Label:      "Verified System Evidence",
		Summary:    fmt.Sprintf("Task paused by %s", actor),
	})
	return nil
}

// CancelTask cancels an active engineering task.
func (cp *ControlPlane) CancelTask(wsID, taskID, actor string) error {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return err
	}
	task.mu.Lock()
	if task.workflow == nil {
		task.mu.Unlock()
		return ErrTaskHalted
	}
	task.workflow.Cancel()
	task.Status = sdlc.StatusCancelled
	task.UpdatedAt = time.Now()
	task.mu.Unlock()

	task.recordActivity(ActivityRecord{
		ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       "task_status",
		Actor:      actor,
		IsVerified: true,
		Label:      "Verified System Evidence",
		Summary:    fmt.Sprintf("Task cancelled by %s", actor),
	})
	return nil
}

// DecideDiff sets approval status ("accepted" or "rejected") on a specific file diff.
func (cp *ControlPlane) DecideDiff(wsID, taskID, filePath string, accept bool, reviewer string) error {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return err
	}

	task.mu.Lock()
	found := false
	status := "rejected"
	if accept {
		status = "accepted"
	}
	for i := range task.Diffs {
		if task.Diffs[i].Path == filePath {
			task.Diffs[i].Status = status
			found = true
			break
		}
	}
	task.mu.Unlock()

	if !found {
		return fmt.Errorf("diff not found for file %q", filePath)
	}

	task.recordActivity(ActivityRecord{
		ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       "diff_decided",
		Actor:      reviewer,
		IsVerified: true,
		Label:      "Verified System Decision",
		Summary:    fmt.Sprintf("File diff %q %s by %s", filePath, status, reviewer),
		Details:    map[string]any{"path": filePath, "status": status, "reviewer": reviewer},
	})
	return nil
}

// GenerateApprovalSignal issues a cryptographically authenticated approval signal
// for a privileged task operation within a workspace.
func (cp *ControlPlane) GenerateApprovalSignal(wsID, taskID, approverID, secretKey string, ttl time.Duration) (*sec.ApprovalSignal, error) {
	task, err := cp.liveTask(wsID, taskID)
	if err != nil {
		return nil, err
	}

	scope := fmt.Sprintf("dash:task:%s:%s", wsID, taskID)
	sig, err := sec.NewApprovalSignal(approverID, scope, secretKey, ttl)
	if err != nil {
		return nil, err
	}

	task.recordActivity(ActivityRecord{
		ID:         fmt.Sprintf("act-%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Type:       "approval_signal_issued",
		Actor:      approverID,
		IsVerified: true,
		Label:      "Verified System Decision",
		Summary:    fmt.Sprintf("Cryptographic approval signal issued by %s for scope %q", approverID, scope),
		Details:    map[string]any{"scope": scope, "valid_until": sig.ValidUntil},
	})

	return sig, nil
}

// Subscribe opens a real-time event channel for task execution events.
func (task *EngineeringTask) Subscribe() (<-chan ActivityRecord, func()) {
	return task.SubscribeActivities()
}

// SubscribeActivities registers a channel to receive live activity updates for a task.
func (task *EngineeringTask) SubscribeActivities() (<-chan ActivityRecord, func()) {
	task = task.target()
	task.mu.Lock()
	defer task.mu.Unlock()
	// Size the buffer so replaying history under the lock can never block.
	ch := make(chan ActivityRecord, len(task.Activities)+32)
	task.listeners[ch] = struct{}{}

	// Replay existing activities
	for _, act := range task.Activities {
		ch <- act
	}

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			task.mu.Lock()
			defer task.mu.Unlock()
			delete(task.listeners, ch)
			close(ch)
		})
	}
	return ch, unsubscribe
}

func (task *EngineeringTask) recordActivity(act ActivityRecord) {
	task.mu.Lock()
	defer task.mu.Unlock()
	task.Activities = append(task.Activities, act)
	// Deliver while holding the lock so an unsubscribe cannot close a channel
	// mid-send. Sends are non-blocking, so the lock is never held for long.
	for ch := range task.listeners {
		select {
		case ch <- act:
		default:
			// Non-blocking write to avoid stalling the control plane
		}
	}
}
