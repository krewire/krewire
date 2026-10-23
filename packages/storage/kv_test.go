package storage_test

import (
	"context"
	"testing"

	"github.com/krewire/krewire/packages/storage"
)

func TestMemoryKV(t *testing.T) {
	ctx := context.Background()
	kv := storage.NewMemoryKV()

	// Put
	if err := kv.Put(ctx, "users/1", []byte("alice")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get
	val, ok, err := kv.Get(ctx, "users/1")
	if err != nil || !ok || string(val) != "alice" {
		t.Fatalf("Get: val=%s, ok=%t, err=%v", string(val), ok, err)
	}

	// List
	_ = kv.Put(ctx, "users/2", []byte("bob"))
	keys, err := kv.List(ctx, "users")
	if err != nil || len(keys) != 2 {
		t.Fatalf("List: keys=%v, err=%v", keys, err)
	}

	// Delete
	if err := kv.Delete(ctx, "users/1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, ok, _ = kv.Get(ctx, "users/1")
	if ok {
		t.Fatal("expected users/1 to be deleted")
	}
}
