package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/krewire/krewire/cloud/infra/contract"
	"github.com/krewire/krewire/errs"
)

func stores(t *testing.T) map[string]Store {
	return map[string]Store{
		"memory": NewMemory(),
		"file":   &File{Dir: filepath.Join(t.TempDir(), "state"), History: 2},
	}
}

func next(s contract.Snapshot) contract.Snapshot {
	s.Serial++
	s.Lineage = "L1"
	s.Environment = "dev"
	return s
}

func TestStoreContract(t *testing.T) {
	ctx := context.Background()
	for name, st := range stores(t) {
		t.Run(name, func(t *testing.T) {
			snap, err := st.Read(ctx)
			if err != nil || snap.Serial != 0 || len(snap.Resources) != 0 {
				t.Fatalf("empty read: %+v %v", snap, err)
			}
			if _, err := st.Write(ctx, nil, next(snap)); !errors.Is(err, ErrLockNotHeld) {
				t.Fatalf("write without lock: %v", err)
			}
			lk, err := st.Lock(ctx, LockInfo{Who: "a", Operation: "apply"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = st.Lock(ctx, LockInfo{Who: "b", Operation: "apply"})
			var le *LockedError
			if !errors.As(err, &le) || le.Holder.Who != "a" || errs.HintOf(err) == "" {
				t.Fatalf("competing lock: %v", err)
			}
			for i := 0; i < 3; i++ {
				n := next(snap)
				n.Put(contract.ResourceState{Resource: contract.Resource{Kind: "file", Name: "x"}, Status: contract.StatusReady})
				if snap, err = st.Write(ctx, lk, n); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := st.Write(ctx, lk, snap); !errors.Is(err, ErrSerialConflict) {
				t.Fatalf("stale serial accepted: %v", err)
			}
			bad := next(snap)
			bad.Lineage = "other"
			if _, err := st.Write(ctx, lk, bad); !errors.Is(err, ErrSerialConflict) {
				t.Fatalf("foreign lineage accepted: %v", err)
			}
			got, err := st.Read(ctx)
			if err != nil || got.Serial != 3 || got.Checksum == "" || len(got.Resources) != 1 {
				t.Fatalf("read back: %+v %v", got, err)
			}
			if err := st.ForceUnlock(ctx, "wrong"); !errors.Is(err, ErrLockNotHeld) {
				t.Fatalf("force unlock wrong id: %v", err)
			}
			if err := st.ForceUnlock(ctx, lk.Info().ID); err != nil {
				t.Fatal(err)
			}
			if _, err := st.Write(ctx, lk, next(got)); !errors.Is(err, ErrLockNotHeld) {
				t.Fatalf("write after force unlock: %v", err)
			}
			lk2, err := st.Lock(ctx, LockInfo{Who: "b"})
			if err != nil {
				t.Fatal(err)
			}
			if err := lk.Unlock(); err != nil { // stale handle must not release lk2
				t.Fatal(err)
			}
			if _, err := st.Lock(ctx, LockInfo{Who: "c"}); err == nil {
				t.Fatal("stale unlock released a newer lock")
			}
			if err := lk2.Unlock(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConcurrentLock(t *testing.T) {
	ctx := context.Background()
	for name, st := range stores(t) {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			var mu sync.Mutex
			wins := 0
			for i := 0; i < 16; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if _, err := st.Lock(ctx, LockInfo{Who: "w"}); err == nil {
						mu.Lock()
						wins++
						mu.Unlock()
					}
				}()
			}
			wg.Wait()
			if wins != 1 {
				t.Fatalf("expected exactly one lock winner, got %d", wins)
			}
		})
	}
}

func TestFileIntegrityAndHistory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st := &File{Dir: dir, History: 2}
	lk, err := st.Lock(ctx, LockInfo{Who: "a"})
	if err != nil {
		t.Fatal(err)
	}
	snap := Empty()
	for i := 0; i < 4; i++ {
		if snap, err = st.Write(ctx, lk, next(snap)); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, historyDir))
	if len(entries) != 2 {
		t.Fatalf("history not pruned: %d entries", len(entries))
	}
	info, _ := os.Stat(filepath.Join(dir, stateFile))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state file mode %v", info.Mode().Perm())
	}
	path := filepath.Join(dir, stateFile)
	data, _ := os.ReadFile(path)
	tampered := strings.Replace(string(data), `"environment": "dev"`, `"environment": "prod"`, 1)
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Read(ctx); !errors.Is(err, ErrIntegrity) || errs.HintOf(err) == "" {
		t.Fatalf("tampered state accepted: %v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Read(ctx); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("corrupt state accepted: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"formatVersion":99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Read(ctx); !errors.Is(err, ErrFormat) {
		t.Fatalf("future format accepted: %v", err)
	}
}
