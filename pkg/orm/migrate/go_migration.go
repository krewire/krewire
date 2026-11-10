package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/schema"
)

// GoMigration represents a programmatic migration written in Go (mirroring Laravel migrations).
type GoMigration struct {
	// Version is the timestamp or numeric identifier, e.g. "20261110224500".
	Version string
	// Name is the descriptive slug, e.g. "create_users_table".
	Name string
	// Up executes the forward schema modifications.
	Up func(ctx context.Context, s *schema.Builder) error
	// Down rolls back the schema modifications.
	Down func(ctx context.Context, s *schema.Builder) error
}

// String returns the migration identifier, e.g. "20261110224500_create_users_table".
func (m *GoMigration) String() string {
	return m.Version + "_" + m.Name
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]*GoMigration)
)

// Register registers a GoMigration into the global registry (typically called in init()).
func Register(m *GoMigration) {
	if m == nil || m.Version == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[m.Version] = m
}

// GetRegistered returns all registered Go migrations sorted by Version ascending.
func GetRegistered() []*GoMigration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]*GoMigration, 0, len(registry))
	for _, m := range registry {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		return versionLess(out[i].Version, out[j].Version)
	})
	return out
}

// ClearRegistry resets the global registry (useful in test suites).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]*GoMigration)
}

// GoMigrator executes registered Go migrations against an *orm.DB connection.
type GoMigrator struct {
	m          *Migrator
	migrations []*GoMigration
}

// NewGoMigrator initializes a migrator for registered Go migrations.
func NewGoMigrator(db *orm.DB, opts Options) (*GoMigrator, error) {
	if db == nil {
		return nil, errs.UsageError("migrate: database is nil")
	}
	table := opts.Table
	if table == "" {
		table = DefaultTable
	}
	if !identRe.MatchString(table) {
		return nil, errs.UsageError(fmt.Sprintf("migrate: invalid table name %q", table))
	}
	d, err := dialectFor(db.Dialect(), table)
	if err != nil {
		return nil, err
	}
	timeout := opts.LockTimeout
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}

	m := &Migrator{
		db:          db,
		table:       table,
		lockTimeout: timeout,
		dialect:     d,
	}

	return &GoMigrator{
		m:          m,
		migrations: GetRegistered(),
	}, nil
}

// Up applies pending Go migrations up to n steps (all when n <= 0).
func (g *GoMigrator) Up(ctx context.Context, n int) ([]Result, error) {
	if err := g.m.ensureTable(ctx); err != nil {
		return nil, err
	}

	var done []Result
	for n <= 0 || len(done) < n {
		var res *Result
		err := g.m.locked(ctx, func(q querier) error {
			applied, err := g.m.applied(ctx, q)
			if err != nil {
				return err
			}
			for _, mig := range g.migrations {
				if _, ok := applied[normVersion(mig.Version)]; !ok {
					var applyErr error
					res, applyErr = g.apply(ctx, q, mig)
					return applyErr
				}
			}
			return nil
		})
		if err != nil {
			return done, err
		}
		if res == nil {
			break
		}
		done = append(done, *res)
	}
	return done, nil
}

// Down rolls back the n most recently applied Go migrations (n >= 1).
func (g *GoMigrator) Down(ctx context.Context, n int) ([]Result, error) {
	if n < 1 {
		return nil, errs.UsageError("migrate: down needs a step count of at least 1")
	}
	if err := g.m.ensureTable(ctx); err != nil {
		return nil, err
	}

	var done []Result
	for len(done) < n {
		var res *Result
		err := g.m.locked(ctx, func(q querier) error {
			var err error
			res, err = g.rollbackLatest(ctx, q)
			return err
		})
		if err != nil {
			return done, err
		}
		if res == nil {
			break
		}
		done = append(done, *res)
	}
	return done, nil
}

// Redo rolls back the newest migration and applies it again in one lock session.
func (g *GoMigrator) Redo(ctx context.Context) ([]Result, error) {
	if err := g.m.ensureTable(ctx); err != nil {
		return nil, err
	}
	var done []Result
	err := g.m.locked(ctx, func(q querier) error {
		down, err := g.rollbackLatest(ctx, q)
		if err != nil || down == nil {
			return err
		}
		var target *GoMigration
		for _, m := range g.migrations {
			if normVersion(m.Version) == normVersion(down.Migration.Version) {
				target = m
				break
			}
		}
		if target == nil {
			return fmt.Errorf("migrate: cannot redo missing migration %s", down.Migration.Version)
		}
		up, err := g.apply(ctx, q, target)
		if err != nil {
			return err
		}
		done = []Result{*down, *up}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return done, nil
}

// Status returns the migration state of all registered Go migrations.
func (g *GoMigrator) Status(ctx context.Context) ([]Status, error) {
	db := g.m.db.SQLDB()
	var n int
	if err := db.QueryRowContext(ctx, g.m.dialect.tableExists, g.m.table).Scan(&n); err != nil {
		return nil, fmt.Errorf("migrate: status: %w", err)
	}
	applied := map[string]record{}
	if n > 0 {
		var err error
		if applied, err = g.m.applied(ctx, db); err != nil {
			return nil, err
		}
	}
	var out []Status
	registered := map[string]bool{}
	for _, mig := range g.migrations {
		key := normVersion(mig.Version)
		registered[key] = true
		st := Status{Version: mig.Version, Name: mig.Name, State: StatePending}
		if rec, ok := applied[key]; ok {
			st.State, st.AppliedAt = StateApplied, rec.appliedAt
		}
		out = append(out, st)
	}
	for key, rec := range applied {
		if !registered[key] {
			out = append(out, Status{Version: rec.version, Name: rec.name, State: StateMissing, AppliedAt: rec.appliedAt})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return versionLess(out[i].Version, out[j].Version) })
	return out, nil
}

func (g *GoMigrator) apply(ctx context.Context, q querier, mig *GoMigration) (*Result, error) {
	start := time.Now()
	if mig.Up != nil {
		s := schema.NewExecutorBuilder(ctx, q, g.m.db.Dialect())
		if err := mig.Up(ctx, s); err != nil {
			return nil, fmt.Errorf("migrate: %s up: %w", mig.String(), err)
		}
	}

	checksum := computeGoChecksum(mig)
	if _, err := q.ExecContext(ctx, g.m.dialect.insert, mig.Version, mig.Name, checksum, time.Now().UTC()); err != nil {
		return nil, fmt.Errorf("migrate: record %s: %w", mig.String(), err)
	}

	return &Result{
		Migration: Migration{
			Version:  mig.Version,
			Name:     mig.Name,
			File:     mig.String() + ".go",
			Checksum: checksum,
		},
		Direction: DirectionUp,
		Duration:  time.Since(start),
	}, nil
}

func (g *GoMigrator) rollbackLatest(ctx context.Context, q querier) (*Result, error) {
	applied, err := g.m.applied(ctx, q)
	if err != nil || len(applied) == 0 {
		return nil, err
	}
	var latest record
	for _, rec := range applied {
		if latest.version == "" || versionLess(latest.version, rec.version) {
			latest = rec
		}
	}
	if latest.version == "" {
		return nil, nil
	}

	var target *GoMigration
	for _, m := range g.migrations {
		if normVersion(m.Version) == normVersion(latest.version) {
			target = m
			break
		}
	}

	start := time.Now()
	if target != nil && target.Down != nil {
		s := schema.NewExecutorBuilder(ctx, q, g.m.db.Dialect())
		if err := target.Down(ctx, s); err != nil {
			return nil, fmt.Errorf("migrate: %s down: %w", latest.name, err)
		}
	}

	if _, err := q.ExecContext(ctx, g.m.dialect.delete, latest.version); err != nil {
		return nil, fmt.Errorf("migrate: delete row %s: %w", latest.version, err)
	}

	name := latest.name
	if target != nil {
		name = target.Name
	}

	return &Result{
		Migration: Migration{
			Version: latest.version,
			Name:    name,
			File:    latest.version + "_" + name + ".go",
		},
		Direction: DirectionDown,
		Duration:  time.Since(start),
	}, nil
}

func computeGoChecksum(m *GoMigration) string {
	h := sha256.New()
	h.Write([]byte(m.Version + ":" + m.Name))
	return hex.EncodeToString(h.Sum(nil))
}

// RunGo applies or rolls back registered Go migrations.
// action is "up", "down", or "redo"; steps is 0 for all (up) or n steps.
func RunGo(ctx context.Context, db *orm.DB, action string, steps int) ([]Result, error) {
	gm, err := NewGoMigrator(db, Options{})
	if err != nil {
		return nil, err
	}
	switch action {
	case "up":
		return gm.Up(ctx, steps)
	case "down":
		if steps <= 0 {
			steps = 1
		}
		return gm.Down(ctx, steps)
	case "redo":
		return gm.Redo(ctx)
	default:
		return nil, errs.UsageError(fmt.Sprintf("migrate: unknown action %q", action))
	}
}

// UpGo applies all registered Go migrations.
func UpGo(ctx context.Context, db *orm.DB) error {
	_, err := RunGo(ctx, db, "up", 0)
	return err
}
