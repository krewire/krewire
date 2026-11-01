package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krewire/krewire/pkg/cloud/storage"
)

// SerializableJob is a Job that can be persisted to and reconstructed from storage.
type SerializableJob interface {
	Job
	// JobType returns the registered type name for the job.
	JobType() string
	// Encode serializes the job's internal payload into bytes.
	Encode() ([]byte, error)
	// Decode restores the job's internal state from bytes.
	Decode([]byte) error
}

// JobFactory instantiates a new instance of a registered job type.
type JobFactory func() SerializableJob

// JobRegistry maps job type names to their constructor factories.
type JobRegistry struct {
	mu        sync.RWMutex
	factories map[string]JobFactory
}

// NewJobRegistry creates an empty JobRegistry.
func NewJobRegistry() *JobRegistry {
	return &JobRegistry{
		factories: make(map[string]JobFactory),
	}
}

// Register registers a named job factory.
func (r *JobRegistry) Register(name string, factory JobFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Resolve creates a new Job instance from a registered type name and payload.
func (r *JobRegistry) Resolve(name string, payload []byte) (Job, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("worker: unregistered job type %q", name)
	}
	job := factory()
	if err := job.Decode(payload); err != nil {
		return nil, fmt.Errorf("worker: decode job %q: %w", name, err)
	}
	return job, nil
}

// PayloadJob is a generic SerializableJob helper for byte or JSON payloads.
type PayloadJob struct {
	Type    string
	Data    []byte
	Handler func(ctx context.Context, payload []byte) error `json:"-"`
}

func (p *PayloadJob) JobType() string         { return p.Type }
func (p *PayloadJob) Encode() ([]byte, error) { return p.Data, nil }
func (p *PayloadJob) Decode(b []byte) error   { p.Data = b; return nil }
func (p *PayloadJob) Run(ctx context.Context) error {
	if p.Handler != nil {
		return p.Handler(ctx, p.Data)
	}
	return nil
}

// taskRecord is the JSON-serializable representation of a Task in KV storage.
type taskRecord struct {
	ID       JobID         `json:"id"`
	Type     string        `json:"type"`
	Payload  []byte        `json:"payload,omitempty"`
	Priority int           `json:"priority"`
	Delay    time.Duration `json:"delay"`
	Cron     string        `json:"cron,omitempty"`
	Retry    *RetryPolicy  `json:"retry,omitempty"`
	Attempts int           `json:"attempts"`
	RunAt    time.Time     `json:"run_at"`
	Seq      uint64        `json:"seq"`
}

type deadLetterRecord struct {
	ID       JobID     `json:"id"`
	Type     string    `json:"type"`
	Payload  []byte    `json:"payload,omitempty"`
	Err      string    `json:"err"`
	Attempts int       `json:"attempts"`
	At       time.Time `json:"at"`
}

type taskLease struct {
	Owner     string    `json:"owner"`
	ExpiresAt time.Time `json:"expires_at"`
}

var globalWorkerCounter atomic.Uint64

func newWorkerID() string {
	return fmt.Sprintf("w-%d-%d", time.Now().UnixNano(), globalWorkerCounter.Add(1))
}

const (
	// DefaultPollInterval is how often Dequeue polls KV for ready tasks from other nodes.
	DefaultPollInterval = 10 * time.Millisecond
	// DefaultLeaseDuration is how long a worker holds a lease on a task before it expires.
	DefaultLeaseDuration = 30 * time.Second
)

// KVQueue is a persistent, durable job queue backed by storage.KV with distributed leasing.
type KVQueue struct {
	kv            storage.KV
	registry      *JobRegistry
	mu            sync.Mutex
	localJobs     map[JobID]Job
	inflight      map[JobID]*Task
	seq           atomic.Uint64
	notify        chan struct{}
	now           func() time.Time
	workerID      string
	pollInterval  time.Duration
	leaseDuration time.Duration
}

var _ Queue = (*KVQueue)(nil)

// KVQueueOption configures a KVQueue.
type KVQueueOption func(*KVQueue)

// WithJobRegistry attaches a JobRegistry to KVQueue for restoring jobs on startup.
func WithJobRegistry(reg *JobRegistry) KVQueueOption {
	return func(q *KVQueue) {
		q.registry = reg
	}
}

// WithPollInterval configures how frequently Dequeue polls KV for ready tasks.
func WithPollInterval(d time.Duration) KVQueueOption {
	return func(q *KVQueue) {
		if d > 0 {
			q.pollInterval = d
		}
	}
}

// WithLeaseDuration configures the expiration duration of claimed task leases.
func WithLeaseDuration(d time.Duration) KVQueueOption {
	return func(q *KVQueue) {
		if d > 0 {
			q.leaseDuration = d
		}
	}
}

// NewKVQueue creates a persistent queue backed by storage.KV and restores any existing tasks.
func NewKVQueue(ctx context.Context, kv storage.KV, opts ...KVQueueOption) (*KVQueue, error) {
	if kv == nil {
		return nil, errors.New("worker: nil storage.KV backend")
	}
	q := &KVQueue{
		kv:            kv,
		registry:      NewJobRegistry(),
		localJobs:     make(map[JobID]Job),
		inflight:      make(map[JobID]*Task),
		notify:        make(chan struct{}, 1),
		now:           time.Now,
		workerID:      newWorkerID(),
		pollInterval:  DefaultPollInterval,
		leaseDuration: DefaultLeaseDuration,
	}
	for _, opt := range opts {
		opt(q)
	}

	if err := q.restore(ctx); err != nil {
		return nil, fmt.Errorf("worker: restore queue from kv: %w", err)
	}
	return q, nil
}

func (q *KVQueue) restore(ctx context.Context) error {
	leases, err := q.kv.List(ctx, "worker/leases/")
	if err != nil {
		return err
	}
	now := q.now()
	for _, lk := range leases {
		val, ok, err := q.kv.Get(ctx, lk)
		if err != nil || !ok {
			continue
		}
		var l taskLease
		if err := json.Unmarshal(val, &l); err == nil && !l.ExpiresAt.After(now) {
			_ = q.kv.Delete(ctx, lk)
		}
	}
	return nil
}

// Enqueue persists a job to KV storage and notifies local workers.
func (q *KVQueue) Enqueue(ctx context.Context, job Job, opts Options) (JobID, error) {
	if job == nil {
		return "", errors.New("worker: nil job")
	}
	n := q.seq.Add(1)
	id := JobID(fmt.Sprintf("job-%d-%s-%d", q.now().UnixNano(), q.workerID, n))
	t := &Task{Job: job, Options: opts, seq: uint64(q.now().UnixNano()) + n, ID: id}

	if opts.Cron != "" {
		sched, err := ParseCron(opts.Cron)
		if err != nil {
			return "", err
		}
		t.cron = sched
	}
	t.runAt = q.now().Add(opts.Delay)

	// Persist task to KV
	jobType := "generic"
	var payload []byte
	if sj, ok := job.(SerializableJob); ok {
		jobType = sj.JobType()
		var err error
		payload, err = sj.Encode()
		if err != nil {
			return "", fmt.Errorf("worker: encode job: %w", err)
		}
	}

	rec := taskRecord{
		ID:       t.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: opts.Priority,
		Delay:    opts.Delay,
		Cron:     opts.Cron,
		Retry:    opts.Retry,
		Attempts: t.Attempts,
		RunAt:    t.runAt,
		Seq:      t.seq,
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := q.kv.Put(ctx, "worker/tasks/"+string(id), b); err != nil {
		return "", fmt.Errorf("worker: persist task: %w", err)
	}

	q.mu.Lock()
	q.localJobs[id] = job
	q.mu.Unlock()

	q.wake()
	return t.ID, nil
}

// claim acquires a distributed lease on a task in KV storage.
func (q *KVQueue) claim(ctx context.Context, taskID JobID) bool {
	leaseKey := "worker/leases/" + string(taskID)
	now := q.now()

	claimToken := fmt.Sprintf("%s-%d", q.workerID, q.seq.Add(1))
	lease := taskLease{
		Owner:     claimToken,
		ExpiresAt: now.Add(q.leaseDuration),
	}
	b, err := json.Marshal(lease)
	if err != nil {
		return false
	}

	if akv, ok := q.kv.(storage.AtomicKV); ok {
		stored, err := akv.PutIfAbsent(ctx, leaseKey, b)
		if err != nil {
			return false
		}
		if !stored {
			val, ok, err := q.kv.Get(ctx, leaseKey)
			if err != nil || !ok {
				return false
			}
			var existing taskLease
			if err := json.Unmarshal(val, &existing); err != nil || existing.ExpiresAt.After(now) {
				return false
			}
			// Lease expired, delete and retry PutIfAbsent
			_ = q.kv.Delete(ctx, leaseKey)
			stored, err = akv.PutIfAbsent(ctx, leaseKey, b)
			if err != nil || !stored {
				return false
			}
		}
	} else {
		// Fallback for non-atomic stores
		val, ok, err := q.kv.Get(ctx, leaseKey)
		if err == nil && ok {
			var existing taskLease
			if err := json.Unmarshal(val, &existing); err == nil && existing.ExpiresAt.After(now) {
				return false
			}
		}
		if err := q.kv.Put(ctx, leaseKey, b); err != nil {
			return false
		}
		val, ok, err = q.kv.Get(ctx, leaseKey)
		if err != nil || !ok {
			return false
		}
		var confirmed taskLease
		if err := json.Unmarshal(val, &confirmed); err != nil || confirmed.Owner != claimToken {
			return false
		}
	}

	// Ensure the task still exists in KV
	_, taskOk, err := q.kv.Get(ctx, "worker/tasks/"+string(taskID))
	if err != nil || !taskOk {
		_ = q.kv.Delete(ctx, leaseKey)
		return false
	}

	return true
}

func (q *KVQueue) claimReadyTask(ctx context.Context) (*Task, time.Duration, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	keys, err := q.kv.List(ctx, "worker/tasks/")
	if err != nil {
		return nil, 0, err
	}

	now := q.now()
	var candidates []taskRecord
	var minFutureDelay time.Duration

	for _, k := range keys {
		val, ok, err := q.kv.Get(ctx, k)
		if err != nil || !ok {
			continue
		}
		var rec taskRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}

		if rec.RunAt.After(now) {
			diff := rec.RunAt.Sub(now)
			if minFutureDelay == 0 || diff < minFutureDelay {
				minFutureDelay = diff
			}
			continue
		}

		// Check if actively leased by someone else
		leaseKey := "worker/leases/" + string(rec.ID)
		lval, lok, _ := q.kv.Get(ctx, leaseKey)
		if lok {
			var lease taskLease
			if err := json.Unmarshal(lval, &lease); err == nil {
				if lease.ExpiresAt.After(now) {
					continue
				}
			}
		}

		candidates = append(candidates, rec)
	}

	if len(candidates) == 0 {
		return nil, minFutureDelay, nil
	}

	// Highest priority first, then oldest seq (FIFO)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].Seq < candidates[j].Seq
	})

	for _, cand := range candidates {
		if !q.claim(ctx, cand.ID) {
			continue
		}

		taskKey := "worker/tasks/" + string(cand.ID)
		tval, tok, err := q.kv.Get(ctx, taskKey)
		if err != nil || !tok {
			_ = q.kv.Delete(ctx, "worker/leases/"+string(cand.ID))
			continue
		}
		var rec taskRecord
		if err := json.Unmarshal(tval, &rec); err != nil {
			_ = q.kv.Delete(ctx, "worker/leases/"+string(cand.ID))
			continue
		}

		rec.Attempts++
		if b, err := json.Marshal(rec); err == nil {
			_ = q.kv.Put(ctx, taskKey, b)
		}

		var job Job
		if lj, ok := q.localJobs[rec.ID]; ok {
			job = lj
		} else {
			var err error
			job, err = q.registry.Resolve(rec.Type, rec.Payload)
			if err != nil {
				job = &PayloadJob{Type: rec.Type, Data: rec.Payload}
			}
		}

		task := &Task{
			ID:       rec.ID,
			Job:      job,
			Attempts: rec.Attempts,
			Options: Options{
				Priority: rec.Priority,
				Delay:    rec.Delay,
				Cron:     rec.Cron,
				Retry:    rec.Retry,
			},
			runAt: rec.RunAt,
			seq:   rec.Seq,
		}
		if rec.Cron != "" {
			if sched, err := ParseCron(rec.Cron); err == nil {
				task.cron = sched
			}
		}

		q.inflight[task.ID] = task
		q.wake()
		return task, 0, nil
	}

	return nil, minFutureDelay, nil
}

// Dequeue delivers the highest priority ready task from persistent queue across distributed workers.
func (q *KVQueue) Dequeue(ctx context.Context) (*Task, error) {
	for {
		task, minWait, err := q.claimReadyTask(ctx)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return task, nil
		}

		pollWait := q.pollInterval
		if minWait > 0 && minWait < pollWait {
			pollWait = minWait
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollWait):
		case <-q.notify:
		}
	}
}

// Ack removes the finished task from KV storage or rearms the next cron iteration.
func (q *KVQueue) Ack(ctx context.Context, task *Task) error {
	if task == nil {
		return ErrUnknownTask
	}
	q.mu.Lock()
	if _, ok := q.inflight[task.ID]; !ok {
		q.mu.Unlock()
		return ErrUnknownTask
	}
	delete(q.inflight, task.ID)
	delete(q.localJobs, task.ID)
	q.mu.Unlock()

	if task.cron != nil {
		err := q.rearm(ctx, task)
		_ = q.kv.Delete(ctx, "worker/leases/"+string(task.ID))
		return err
	}

	// Delete completed task from persistent storage FIRST before releasing lease
	_ = q.kv.Delete(ctx, "worker/tasks/"+string(task.ID))
	_ = q.kv.Delete(ctx, "worker/leases/"+string(task.ID))
	return nil
}

// Nack handles task failure, applying retry backoff or filing to DLQ in persistent storage.
func (q *KVQueue) Nack(ctx context.Context, task *Task) error {
	if task == nil {
		return ErrUnknownTask
	}
	policy := task.Options.Retry.resolved()
	q.mu.Lock()
	if _, ok := q.inflight[task.ID]; !ok {
		q.mu.Unlock()
		return ErrUnknownTask
	}
	delete(q.inflight, task.ID)
	delete(q.localJobs, task.ID)
	q.mu.Unlock()

	if task.Attempts >= policy.MaxAttempts {
		_ = q.kv.Delete(ctx, "worker/tasks/"+string(task.ID))
		jobType := "generic"
		var payload []byte
		if sj, ok := task.Job.(SerializableJob); ok {
			jobType = sj.JobType()
			payload, _ = sj.Encode()
		}
		errStr := ""
		if task.LastErr != nil {
			errStr = task.LastErr.Error()
		}
		dlqRec := deadLetterRecord{
			ID:       task.ID,
			Type:     jobType,
			Payload:  payload,
			Err:      errStr,
			Attempts: task.Attempts,
			At:       q.now(),
		}
		b, _ := json.Marshal(dlqRec)
		_ = q.kv.Put(ctx, "worker/dlq/"+string(task.ID), b)
		_ = q.kv.Delete(ctx, "worker/leases/"+string(task.ID))
		return nil
	}

	task.runAt = q.now().Add(policy.Backoff(task.Attempts))
	task.LastErr = nil

	jobType := "generic"
	var payload []byte
	if sj, ok := task.Job.(SerializableJob); ok {
		jobType = sj.JobType()
		payload, _ = sj.Encode()
	}
	rec := taskRecord{
		ID:       task.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: task.Options.Priority,
		Delay:    task.Options.Delay,
		Cron:     task.Options.Cron,
		Retry:    task.Options.Retry,
		Attempts: task.Attempts,
		RunAt:    task.runAt,
		Seq:      task.seq,
	}
	b, _ := json.Marshal(rec)
	_ = q.kv.Put(ctx, "worker/tasks/"+string(task.ID), b)
	_ = q.kv.Delete(ctx, "worker/leases/"+string(task.ID))

	q.wake()
	return nil
}

// DLQ returns all dead-lettered tasks recorded in persistent storage.
func (q *KVQueue) DLQ() []DeadLetter {
	ctx := context.Background()
	keys, err := q.kv.List(ctx, "worker/dlq/")
	if err != nil {
		return nil
	}
	var out []DeadLetter
	for _, k := range keys {
		val, ok, err := q.kv.Get(ctx, k)
		if err != nil || !ok {
			continue
		}
		var rec deadLetterRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}
		job, err := q.registry.Resolve(rec.Type, rec.Payload)
		if err != nil {
			job = &PayloadJob{Type: rec.Type, Data: rec.Payload}
		}
		var retErr error
		if rec.Err != "" {
			retErr = errors.New(rec.Err)
		}
		out = append(out, DeadLetter{
			ID:       rec.ID,
			Job:      job,
			Err:      retErr,
			Attempts: rec.Attempts,
			At:       rec.At,
		})
	}
	return out
}

// ClearDLQ removes all dead-lettered tasks from persistent storage.
func (q *KVQueue) ClearDLQ(ctx context.Context) error {
	keys, err := q.kv.List(ctx, "worker/dlq/")
	if err != nil {
		return err
	}
	for _, k := range keys {
		_ = q.kv.Delete(ctx, k)
	}
	return nil
}

func (q *KVQueue) rearm(ctx context.Context, task *Task) error {
	next := task.cron.NextFire(task.runAt)
	if next.IsZero() {
		_ = q.kv.Delete(ctx, "worker/tasks/"+string(task.ID))
		return nil
	}
	occurrence := *task
	occurrence.runAt = next
	occurrence.seq = uint64(q.now().UnixNano()) + q.seq.Add(1)
	occurrence.Attempts = 0
	occurrence.LastErr = nil

	jobType := "generic"
	var payload []byte
	if sj, ok := task.Job.(SerializableJob); ok {
		jobType = sj.JobType()
		payload, _ = sj.Encode()
	}
	rec := taskRecord{
		ID:       occurrence.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: occurrence.Options.Priority,
		Delay:    occurrence.Options.Delay,
		Cron:     occurrence.Options.Cron,
		Retry:    occurrence.Options.Retry,
		Attempts: occurrence.Attempts,
		RunAt:    occurrence.runAt,
		Seq:      occurrence.seq,
	}
	b, _ := json.Marshal(rec)
	if err := q.kv.Put(ctx, "worker/tasks/"+string(occurrence.ID), b); err != nil {
		return err
	}

	q.wake()
	return nil
}

func (q *KVQueue) wake() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
