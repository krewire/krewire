package store

import (
	"context"
	"sync"

	"github.com/krewire/krewire/cloud/infra/contract"
)

// Memory is an in-process Store for tests and ephemeral environments.
type Memory struct {
	mu     sync.Mutex
	snap   contract.Snapshot
	holder *LockInfo
	// Writes counts successful writes; tests use it to assert no mutation.
	Writes int
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory { return &Memory{snap: Empty()} }

// Read returns the current snapshot.
func (m *Memory) Read(context.Context) (contract.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snap.Clone(), nil
}

// Lock acquires the exclusive lock.
func (m *Memory) Lock(_ context.Context, info LockInfo) (Lock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holder != nil {
		return nil, locked(*m.holder)
	}
	if info.ID == "" {
		info.ID = NewLockID()
	}
	m.holder = &info
	return &memLock{m: m, info: info}, nil
}

// Write stores snap when lock is held and the serial is next.
func (m *Memory) Write(_ context.Context, lock Lock, snap contract.Snapshot) (contract.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if lock == nil || m.holder == nil || m.holder.ID != lock.Info().ID {
		return snap, ErrLockNotHeld
	}
	if err := checkNext(m.snap, snap); err != nil {
		return snap, err
	}
	sealed, err := Seal(snap)
	if err != nil {
		return snap, err
	}
	m.snap = sealed.Clone()
	m.Writes++
	return sealed, nil
}

// ForceUnlock releases the lock with lockID regardless of holder.
func (m *Memory) ForceUnlock(_ context.Context, lockID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holder == nil || m.holder.ID != lockID {
		return ErrLockNotHeld
	}
	m.holder = nil
	return nil
}

type memLock struct {
	m    *Memory
	info LockInfo
}

func (l *memLock) Info() LockInfo { return l.info }

func (l *memLock) Unlock() error {
	l.m.mu.Lock()
	defer l.m.mu.Unlock()
	if l.m.holder != nil && l.m.holder.ID == l.info.ID {
		l.m.holder = nil
	}
	return nil
}
