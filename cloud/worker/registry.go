package worker

import (
	"context"
	"fmt"
	"sync"
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
