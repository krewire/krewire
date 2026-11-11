package seeder

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
)

// Seeder defines the interface for database seeders (mirroring Laravel Seeders).
type Seeder interface {
	Run(ctx context.Context, db *orm.DB) error
}

// FuncSeeder adapts a plain function into a Seeder.
type FuncSeeder func(ctx context.Context, db *orm.DB) error

// Run executes the underlying seeder function.
func (f FuncSeeder) Run(ctx context.Context, db *orm.DB) error {
	return f(ctx, db)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Seeder)
)

// Register registers a named seeder into the global registry.
func Register(name string, s Seeder) {
	if s == nil || name == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = s
}

// Get retrieves a registered seeder by name.
func Get(name string) (Seeder, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := registry[name]
	return s, ok
}

// GetRegistered returns the sorted names of all registered seeders.
func GetRegistered() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// List is an alias for GetRegistered.
func List() []string {
	return GetRegistered()
}

// ClearRegistry resets all registered seeders (useful for testing).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]Seeder)
}

// Call runs one or more seeders sequentially against the database.
// This is the idiomatic equivalent of Laravel's $this->call([UserSeeder::class]).
func Call(ctx context.Context, db *orm.DB, seeders ...Seeder) error {
	if db == nil {
		return errs.UsageError("seeder: database is nil")
	}
	for _, s := range seeders {
		if s == nil {
			continue
		}
		if err := s.Run(ctx, db); err != nil {
			return errs.FailureError(fmt.Sprintf("seeder: failed executing seeder: %v", err))
		}
	}
	return nil
}

// Run executes registered seeders by name. If no names are passed, it executes
// the default "DatabaseSeeder" if registered.
func Run(ctx context.Context, db *orm.DB, names ...string) error {
	if db == nil {
		return errs.UsageError("seeder: database is nil")
	}

	targets := names
	if len(targets) == 0 {
		targets = []string{"DatabaseSeeder"}
	}

	for _, name := range targets {
		s, ok := Get(name)
		if !ok {
			return errs.UsageError(fmt.Sprintf("seeder: seeder %q not found in registry", name))
		}
		if err := s.Run(ctx, db); err != nil {
			return errs.FailureError(fmt.Sprintf("seeder: execution of %q failed: %v", name, err))
		}
	}
	return nil
}
