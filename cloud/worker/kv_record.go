package worker

import (
	"encoding/json"
	"fmt"
	"time"
)

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

// deadLetterRecord is the JSON-serializable representation of a dead-lettered job.
type deadLetterRecord struct {
	ID       JobID     `json:"id"`
	Type     string    `json:"type"`
	Payload  []byte    `json:"payload,omitempty"`
	Err      string    `json:"err"`
	Attempts int       `json:"attempts"`
	At       time.Time `json:"at"`
}

// taskLease represents an exclusive, expiring lease held by a worker.
type taskLease struct {
	Owner     string    `json:"owner"`
	ExpiresAt time.Time `json:"expires_at"`
}

func marshalTaskRecord(t *Task) ([]byte, error) {
	jobType := "generic"
	var payload []byte
	if sj, ok := t.Job.(SerializableJob); ok {
		jobType = sj.JobType()
		var err error
		payload, err = sj.Encode()
		if err != nil {
			return nil, fmt.Errorf("worker: encode job: %w", err)
		}
	}

	rec := taskRecord{
		ID:       t.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: t.Options.Priority,
		Delay:    t.Options.Delay,
		Cron:     t.Options.Cron,
		Retry:    t.Options.Retry,
		Attempts: t.Attempts,
		RunAt:    t.runAt,
		Seq:      t.seq,
	}
	return json.Marshal(rec)
}

func marshalDeadLetterRecord(task *Task, at time.Time) ([]byte, error) {
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
		At:       at,
	}
	return json.Marshal(dlqRec)
}
