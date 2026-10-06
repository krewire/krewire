package app

import (
	"context"
	"testing"

	"github.com/krewire/krewire/packages/cloud/storage"
)

// Spec: KWF-AST-K7Q2M FRK-AST-013 Scope: Unit
//
// The container binding lives here rather than in the storage package because
// the adapter depends on the app container; keeping it in storage would make
// `cloud` import `app` while `app` imports `cloud/storage`.
func TestFRK_AST_013_Provider_BindsIntoContainer(t *testing.T) {
	c, err := NewApp(StorageProvider(storage.NewMemory())).Build()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resolve[*storage.KV](c)
	if err != nil {
		t.Fatal(err)
	}
	kv := *got
	if err := kv.Put(context.Background(), "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	val, ok, err := kv.Get(context.Background(), "k")
	if err != nil || !ok || string(val) != "v" {
		t.Errorf("resolved kv broken: %q %v %v", val, ok, err)
	}
}
