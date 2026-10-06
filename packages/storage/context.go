package storage

import (
	"context"
	"fmt"
	"io"

	kfs "github.com/krewire/krewire/packages/fs"
)

// ContextStore is the context-aware extension of Store. Store remains the
// stable minimal contract for existing backends.
type ContextStore interface {
	Store
	PutContext(context.Context, string, []byte) error
	GetContext(context.Context, string) ([]byte, error)
	DeleteContext(context.Context, string) error
	StatContext(context.Context, string) (ObjectInfo, error)
}

// NewMemory returns an in-memory local store for tests and ephemeral data.
func NewMemory() (*Local, error) {
	return NewLocal("memory", kfs.NewMemory())
}

// PutReaderContext stores all bytes read from r, honoring context before and
// after the read and write operations.
func (s *Local) PutReaderContext(ctx context.Context, key string, r io.Reader) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("storage: reader must not be nil")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("storage: read object %q: %w", key, err)
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	return s.PutContext(ctx, key, data)
}

// PutContext stores an object while checking context cancellation.
func (s *Local) PutContext(ctx context.Context, key string, data []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.Put(key, data); err != nil {
		return err
	}
	return contextError(ctx)
}

// GetContext retrieves an object while checking context cancellation.
func (s *Local) GetContext(ctx context.Context, key string) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	data, err := s.Get(key)
	if err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

// DeleteContext deletes an object while checking context cancellation.
func (s *Local) DeleteContext(ctx context.Context, key string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.Delete(key); err != nil {
		return err
	}
	return contextError(ctx)
}

// StatContext retrieves object metadata while checking context cancellation.
func (s *Local) StatContext(ctx context.Context, key string) (ObjectInfo, error) {
	if err := contextError(ctx); err != nil {
		return ObjectInfo{}, err
	}
	info, err := s.Stat(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	if err := contextError(ctx); err != nil {
		return ObjectInfo{}, err
	}
	return info, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("storage: nil context")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

var _ ContextStore = (*Local)(nil)
