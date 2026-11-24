// Package store persists versioned infrastructure snapshots with locking and
// integrity checks (issue #138).
//
// Every store implements the same contract:
//
//   - Read returns the latest snapshot, or an empty one, and verifies its
//     checksum so a corrupted or hand-edited file is reported, not trusted.
//   - Lock grants one exclusive mutation lock per environment. A competing
//     Lock fails with *LockedError naming the holder.
//   - Write requires the held lock and the next serial (compare-and-swap), so
//     a writer can never overwrite a snapshot it did not read.
//   - ForceUnlock is the explicit operator escape hatch for a stale lock.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/krewire/krewire/cloud/infra/contract"
	"github.com/krewire/krewire/errs"
)

// LockInfo identifies the holder of a state lock.
type LockInfo struct {
	ID         string    `json:"id"`
	Who        string    `json:"who"`
	Operation  string    `json:"operation"`
	PlanDigest string    `json:"planDigest,omitempty"`
	Created    time.Time `json:"created"`
}

// Lock is a held state lock.
type Lock interface {
	Info() LockInfo
	Unlock() error
}

// Store is the state persistence contract.
type Store interface {
	Read(ctx context.Context) (contract.Snapshot, error)
	Lock(ctx context.Context, info LockInfo) (Lock, error)
	Write(ctx context.Context, lock Lock, snap contract.Snapshot) (contract.Snapshot, error)
	ForceUnlock(ctx context.Context, lockID string) error
}

// LockedError reports a lock held by another operation.
type LockedError struct {
	Holder LockInfo
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("infra state is locked by %s for %s since %s (lock %s)",
		e.Holder.Who, e.Holder.Operation, e.Holder.Created.Format(time.RFC3339), e.Holder.ID)
}

// Errors returned by stores.
var (
	ErrIntegrity      = errors.New("infra state integrity check failed")
	ErrSerialConflict = errors.New("infra state serial conflict")
	ErrLockNotHeld    = errors.New("infra state lock not held")
	ErrFormat         = errors.New("infra state format not supported")
)

// locked wraps a LockedError with operator guidance.
func locked(holder LockInfo) error {
	return errs.WithHint(&LockedError{Holder: holder},
		"wait for the other operation to finish; if it crashed, verify nothing is running and release the lock with ForceUnlock (krew infra unlock "+holder.ID+")")
}

// NewLockID returns a random lock identifier.
func NewLockID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("lock-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// Checksum computes the integrity digest of snap, ignoring snap.Checksum.
func Checksum(snap contract.Snapshot) (string, error) {
	snap.Checksum = ""
	data, err := json.Marshal(snap)
	if err != nil {
		return "", fmt.Errorf("infra state: encode: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Seal stamps the format version and checksum onto snap.
func Seal(snap contract.Snapshot) (contract.Snapshot, error) {
	snap.FormatVersion = contract.StateFormatVersion
	sum, err := Checksum(snap)
	if err != nil {
		return snap, err
	}
	snap.Checksum = sum
	return snap, nil
}

// Verify checks the format version and checksum of a stored snapshot.
func Verify(snap contract.Snapshot) error {
	if snap.FormatVersion != contract.StateFormatVersion {
		return fmt.Errorf("%w: version %d (supported: %d)", ErrFormat, snap.FormatVersion, contract.StateFormatVersion)
	}
	sum, err := Checksum(snap)
	if err != nil {
		return err
	}
	if sum != snap.Checksum {
		return errs.WithHint(fmt.Errorf("%w: recorded %s, computed %s", ErrIntegrity, snap.Checksum, sum),
			"the state file was modified outside the engine; restore the last good copy from the state history directory")
	}
	return nil
}

// Empty returns the snapshot a store reports before its first write.
func Empty() contract.Snapshot {
	return contract.Snapshot{FormatVersion: contract.StateFormatVersion}
}

// checkNext validates the compare-and-swap rule for a write.
func checkNext(current, next contract.Snapshot) error {
	if next.Serial != current.Serial+1 {
		return errs.WithHint(fmt.Errorf("%w: stored serial %d, write serial %d", ErrSerialConflict, current.Serial, next.Serial),
			"another writer changed the state; re-read the state and re-plan")
	}
	if current.Lineage != "" && next.Lineage != current.Lineage {
		return fmt.Errorf("%w: lineage %q does not match stored %q", ErrSerialConflict, next.Lineage, current.Lineage)
	}
	return nil
}
