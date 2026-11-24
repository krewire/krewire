package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/krewire/krewire/cloud/infra/contract"
)

// File names used inside a File store directory.
const (
	stateFile  = "state.json"
	lockFile   = "state.lock"
	historyDir = "history"
)

// DefaultHistory is the number of prior snapshots a File store keeps.
const DefaultHistory = 10

// File stores snapshots in a directory:
//
//	<Dir>/state.json             latest sealed snapshot (0600)
//	<Dir>/state.lock             exclusive lock (created with O_EXCL)
//	<Dir>/history/<serial>.json  recent snapshots for recovery
//
// The lock file makes the lock visible across processes on one host. It is
// not a distributed lock; remote backends implement Store themselves.
type File struct {
	Dir string
	// History is the number of snapshots kept in history/ (DefaultHistory
	// when zero, none when negative).
	History int
}

// NewFile returns a File store rooted at dir.
func NewFile(dir string) *File { return &File{Dir: dir} }

func (f *File) path(name string) string { return filepath.Join(f.Dir, name) }

// Read loads and verifies the latest snapshot.
func (f *File) Read(context.Context) (contract.Snapshot, error) {
	return readSnapshot(f.path(stateFile))
}

func readSnapshot(path string) (contract.Snapshot, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Empty(), nil
	}
	if err != nil {
		return contract.Snapshot{}, fmt.Errorf("infra state: read: %w", err)
	}
	var snap contract.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return contract.Snapshot{}, fmt.Errorf("%w: decode %s: %v", ErrIntegrity, path, err)
	}
	if err := Verify(snap); err != nil {
		return contract.Snapshot{}, err
	}
	return snap, nil
}

// Lock creates the lock file exclusively.
func (f *File) Lock(_ context.Context, info LockInfo) (Lock, error) {
	if err := os.MkdirAll(f.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("infra state: mkdir: %w", err)
	}
	if info.ID == "" {
		info.ID = NewLockID()
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	fh, err := os.OpenFile(f.path(lockFile), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, fs.ErrExist) {
		holder, rerr := f.holder()
		if rerr != nil {
			return nil, fmt.Errorf("infra state: lock file present but unreadable: %w", rerr)
		}
		return nil, locked(holder)
	}
	if err != nil {
		return nil, fmt.Errorf("infra state: lock: %w", err)
	}
	_, werr := fh.Write(data)
	cerr := fh.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(f.path(lockFile))
		return nil, fmt.Errorf("infra state: write lock: %w", errors.Join(werr, cerr))
	}
	return &fileLock{f: f, info: info}, nil
}

func (f *File) holder() (LockInfo, error) {
	var info LockInfo
	data, err := os.ReadFile(f.path(lockFile))
	if err != nil {
		return info, err
	}
	err = json.Unmarshal(data, &info)
	return info, err
}

// Write atomically replaces state.json and records the previous snapshot in
// history/.
func (f *File) Write(ctx context.Context, lock Lock, snap contract.Snapshot) (contract.Snapshot, error) {
	if lock == nil {
		return snap, ErrLockNotHeld
	}
	if h, err := f.holder(); err != nil || h.ID != lock.Info().ID {
		return snap, ErrLockNotHeld
	}
	current, err := f.Read(ctx)
	if err != nil {
		return snap, err
	}
	if err := checkNext(current, snap); err != nil {
		return snap, err
	}
	sealed, err := Seal(snap)
	if err != nil {
		return snap, err
	}
	data, err := json.MarshalIndent(sealed, "", "  ")
	if err != nil {
		return snap, err
	}
	if err := writeAtomic(f.path(stateFile), append(data, '\n')); err != nil {
		return snap, err
	}
	if err := f.keepHistory(sealed.Serial, data); err != nil {
		return sealed, fmt.Errorf("infra state: history (state was written): %w", err)
	}
	return sealed, nil
}

func (f *File) keepHistory(serial uint64, data []byte) error {
	keep := f.History
	if keep == 0 {
		keep = DefaultHistory
	}
	if keep < 0 {
		return nil
	}
	dir := f.path(historyDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(dir, fmt.Sprintf("%020d.json", serial)), data); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > keep {
		if err := os.Remove(filepath.Join(dir, names[0])); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}

// ForceUnlock removes the lock file when it carries lockID.
func (f *File) ForceUnlock(_ context.Context, lockID string) error {
	h, err := f.holder()
	if err != nil || h.ID != lockID {
		return ErrLockNotHeld
	}
	return os.Remove(f.path(lockFile))
}

type fileLock struct {
	f    *File
	info LockInfo
}

func (l *fileLock) Info() LockInfo { return l.info }

func (l *fileLock) Unlock() error {
	h, err := l.f.holder()
	if errors.Is(err, fs.ErrNotExist) || (err == nil && h.ID != l.info.ID) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.Remove(l.f.path(lockFile))
}

// writeAtomic writes data to a temp file, syncs it, and renames it over path.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("infra state: temp: %w", err)
	}
	name := tmp.Name()
	_, werr := tmp.Write(data)
	serr := tmp.Sync()
	cerr := tmp.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("infra state: write: %w", err)
	}
	if err := os.Chmod(name, 0o600); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("infra state: rename: %w", err)
	}
	return nil
}
