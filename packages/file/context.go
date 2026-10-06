package file

import (
	"context"
	"fmt"
	"os"

	"github.com/krewire/krewire/packages/fs"
)

// ReadContext reads a complete file unless ctx is canceled before the
// operation starts or after the underlying read completes.
func ReadContext(ctx context.Context, fsys fs.FileSystem, name string) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	data, err := Read(fsys, name)
	if err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

// WriteContext writes a file unless ctx is canceled before or after the
// underlying write.
func WriteContext(ctx context.Context, fsys fs.FileSystem, name string, data []byte, perm os.FileMode) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := Write(fsys, name, data, perm); err != nil {
		return err
	}
	return contextError(ctx)
}

// WriteAtomicContext atomically replaces a file unless ctx is canceled before
// or after the operation.
func WriteAtomicContext(ctx context.Context, fsys fs.FileSystem, name string, data []byte, perm os.FileMode) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := WriteAtomic(fsys, name, data, perm); err != nil {
		return err
	}
	return contextError(ctx)
}

// CopyContext copies a file unless ctx is canceled before or after the copy.
func CopyContext(ctx context.Context, fsys fs.FileSystem, dst, src string, perm os.FileMode) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := Copy(fsys, dst, src, perm); err != nil {
		return err
	}
	return contextError(ctx)
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("file: nil context")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
