package storage

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMemoryContextStore(t *testing.T) {
	store, err := NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.PutContext(ctx, "item", []byte("data")); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetContext(ctx, "item")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "data" {
		t.Fatalf("GetContext = %q", got)
	}
}

func TestCanceledContextDoesNotWrite(t *testing.T) {
	store, err := NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutContext(ctx, "item", []byte("data")); !errors.Is(err, context.Canceled) {
		t.Fatalf("PutContext error = %v, want context.Canceled", err)
	}
}

// TestContextStore_FullSurface exercises every ContextStore method: the
// cancellation guard, the live path, the nil-context guard, and the nil-reader
// rejection.
func TestContextStore_FullSurface(t *testing.T) {
	store, err := NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	var _ ContextStore = store

	if err := store.Put("item", []byte("data")); err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.PutContext(canceled, "x", []byte("v")); !errors.Is(err, context.Canceled) {
		t.Errorf("PutContext = %v, want canceled", err)
	}
	if _, err := store.GetContext(canceled, "item"); !errors.Is(err, context.Canceled) {
		t.Errorf("GetContext = %v, want canceled", err)
	}
	if err := store.DeleteContext(canceled, "item"); !errors.Is(err, context.Canceled) {
		t.Errorf("DeleteContext = %v, want canceled", err)
	}
	if _, err := store.StatContext(canceled, "item"); !errors.Is(err, context.Canceled) {
		t.Errorf("StatContext = %v, want canceled", err)
	}
	if err := store.PutReaderContext(canceled, "x", strings.NewReader("v")); !errors.Is(err, context.Canceled) {
		t.Errorf("PutReaderContext = %v, want canceled", err)
	}

	ctx := context.Background()
	if err := store.PutReaderContext(ctx, "streamed", strings.NewReader("from reader")); err != nil {
		t.Fatalf("PutReaderContext: %v", err)
	}
	got, err := store.GetContext(ctx, "streamed")
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if string(got) != "from reader" {
		t.Errorf("streamed contents = %q", got)
	}
	if err := store.PutContext(ctx, "k2", []byte("v2")); err != nil {
		t.Fatalf("PutContext: %v", err)
	}
	info, err := store.StatContext(ctx, "k2")
	if err != nil {
		t.Fatalf("StatContext: %v", err)
	}
	if info.Size != 2 || info.Key != "k2" {
		t.Errorf("StatContext info = %+v", info)
	}
	if err := store.DeleteContext(ctx, "k2"); err != nil {
		t.Fatalf("DeleteContext: %v", err)
	}

	//nolint:staticcheck // a nil context is the behaviour under test
	if err := store.PutContext(nil, "x", []byte("v")); err == nil {
		t.Error("PutContext(nil) must return an error rather than panic")
	}
	//nolint:staticcheck // a nil reader is the behaviour under test
	if err := store.PutReaderContext(ctx, "x", nil); err == nil {
		t.Error("PutReaderContext(nil reader) must return an error")
	}
}
