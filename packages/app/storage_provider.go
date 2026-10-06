package app

import (
	"github.com/krewire/krewire/packages/cloud/storage"
)

// StorageProvider returns an app.Provider registering kv as the container-wide
// KV singleton so application modules resolve storage.KV during assembly.
//
// The adapter lives in app rather than in the storage package itself: the
// storage package is a runtime primitive and must not depend on the container,
// or `cloud` would end up importing `app` while `app` imports `cloud`.
func StorageProvider(kv storage.KV) Provider {
	return ProviderFunc(func(c *Container) error {
		return Singleton[*storage.KV](c, func() *storage.KV { return &kv })
	})
}
