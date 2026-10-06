package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalRoundTripAndTraversalProtection(t *testing.T) {
	store, err := NewLocal(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("objects/item.txt", []byte("payload")); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("objects/item.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "payload" {
		t.Fatalf("payload = %q, want payload", got)
	}
	info, err := store.Stat("objects/item.txt")
	if err != nil {
		t.Fatal(err)
	}
	if info.Key != "objects/item.txt" || info.Size != int64(len("payload")) {
		t.Fatalf("info = %+v", info)
	}
	if err := store.Put("../escape.txt", []byte("blocked")); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	if err := store.Delete("objects/item.txt"); err != nil {
		t.Fatal(err)
	}
}

// TestStore_LocalRejectsEscapingKeys verifies a key cannot address a file
// outside the store root on every operation, which would otherwise turn the
// store into an arbitrary-file primitive.
func TestStore_LocalRejectsEscapingKeys(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocal(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape.bin", "../../escape.bin", "a/../../escape.bin"} {
		if err := store.Put(key, []byte("x")); err == nil {
			t.Errorf("Put(%q) must be rejected", key)
		}
		if _, err := store.Get(key); err == nil {
			t.Errorf("Get(%q) must be rejected", key)
		}
		if err := store.Delete(key); err == nil {
			t.Errorf("Delete(%q) must be rejected", key)
		}
		if _, err := store.Stat(key); err == nil {
			t.Errorf("Stat(%q) must be rejected", key)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.bin")); err == nil {
		t.Error("a file was created outside the store root")
	}
}

// TestStore_LocalErrors verifies the failure paths report an error rather than
// returning zero values, including the nil-receiver guard on path().
func TestStore_LocalErrors(t *testing.T) {
	if _, err := NewLocal("", nil); err == nil {
		t.Error("NewLocal must reject an empty root")
	}
	store, err := NewLocal(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("absent.bin"); err == nil {
		t.Error("Delete of a missing object must fail")
	}
	if _, err := store.Stat("absent.bin"); err == nil {
		t.Error("Stat of a missing object must fail")
	}
	var nilStore *Local
	if _, err := nilStore.path("k"); err == nil {
		t.Error("a nil *Local must report an error instead of panicking")
	}
}

// TestStore_MemoryBackend verifies the in-memory convenience constructor
// satisfies the exported Store contract.
func TestStore_MemoryBackend(t *testing.T) {
	store, err := NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	var _ Store = store
	if err := store.Put("k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v" {
		t.Errorf("Get = %q", got)
	}
}
