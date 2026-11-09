// Package migrate applies the SQL migrations that `kiw make migration`
// generates, keeping track of them in a schema_migrations table.
//
// Every migration runs in its own transaction together with the bookkeeping
// row, so a failing migration leaves neither half-applied DDL nor a stale row
// behind (on engines with transactional DDL: SQLite and PostgreSQL). Each
// applied migration records the checksum of its file; editing an applied file
// is reported as ErrChecksumMismatch instead of being silently ignored.
// Concurrent runners — several replicas starting at once, or `kiw migrate`
// next to a running app — are serialized with a database-level lock, so each
// migration is applied exactly once.
//
// The package does not link a database driver. Open the database with the
// driver of your choice (for example pkg/orm/sqlite) and pass the *orm.DB.
//
//	db, err := sqlite.Open("app.db")
//	...
//	if err := migrate.Up(ctx, db, migrations.Files); err != nil { ... }
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"regexp"
	"sort"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
)

// DefaultTable is the bookkeeping table used when Options.Table is empty.
const DefaultTable = "schema_migrations"

// DefaultLockTimeout bounds how long a runner waits for another runner's lock.
const DefaultLockTimeout = 30 * time.Second

var (
	// ErrChecksumMismatch reports an applied migration whose file changed.
	ErrChecksumMismatch = errors.New("migrate: applied migration was edited")
	// ErrMissingFile reports an applied migration whose file is gone.
	ErrMissingFile = errors.New("migrate: applied migration has no file")
	// ErrIrreversible reports a rollback of a migration without a down section.
	ErrIrreversible = errors.New("migrate: migration has no -- +down section")
	// ErrLocked reports that another runner held the lock past LockTimeout.
	ErrLocked = errors.New("migrate: another migration runner holds the lock")
	// ErrUnsupportedDialect reports a database this package cannot lock safely.
	ErrUnsupportedDialect = errors.New("migrate: unsupported database dialect")
)

// Options tunes a Migrator. The zero value is ready to use.
type Options struct {
	// Table names the bookkeeping table (default "schema_migrations").
	Table string
	// LockTimeout bounds the wait for a concurrent runner (default 30s).
	LockTimeout time.Duration
}

// Direction tells whether a Result applied or rolled back a migration.
type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// Result describes one migration that a call applied or rolled back.
type Result struct {
	Migration Migration
	Direction Direction
	Duration  time.Duration
}

// State is the status of one migration.
type State string

const (
	StatePending  State = "pending"  // file present, not applied
	StateApplied  State = "applied"  // applied, file unchanged
	StateModified State = "modified" // applied, file edited since (checksum differs)
	StateMissing  State = "missing"  // applied, file deleted
)

// Status is one row of Migrator.Status.
type Status struct {
	Version   string
	Name      string
	State     State
	AppliedAt time.Time // zero when pending
}

// Migrator applies a fixed set of migrations to one database.
type Migrator struct {
	db          *orm.DB
	migrations  []Migration
	table       string
	lockTimeout time.Duration
	dialect     dialect
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)

// New loads the *.sql files at the root of fsys (see Load) for db. Pass an
// embed.FS of the migrations directory, or os.DirFS("migrations").
func New(db *orm.DB, fsys fs.FS, opts Options) (*Migrator, error) {
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
	migrations, err := Load(fsys)
	if err != nil {
		return nil, err
	}
	timeout := opts.LockTimeout
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}
	return &Migrator{db: db, migrations: migrations, table: table, lockTimeout: timeout, dialect: d}, nil
}

// Up applies every pending migration in fsys to db. It is the one-liner for
// running migrations when an application starts.
func Up(ctx context.Context, db *orm.DB, fsys fs.FS) error {
	m, err := New(db, fsys, Options{})
	if err != nil {
		return err
	}
	_, err = m.Up(ctx, 0)
	return err
}

// Migrations returns the loaded migrations in version order.
func (m *Migrator) Migrations() []Migration {
	return append([]Migration(nil), m.migrations...)
}

// Up applies up to n pending migrations in version order (all when n <= 0),
// each in its own transaction. It returns the migrations it applied, also
// when a later one fails.
func (m *Migrator) Up(ctx context.Context, n int) ([]Result, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	var done []Result
	for n <= 0 || len(done) < n {
		var res *Result
		err := m.locked(ctx, func(q querier) error {
			applied, err := m.verified(ctx, q)
			if err != nil {
				return err
			}
			for _, mig := range m.migrations {
				if _, ok := applied[normVersion(mig.Version)]; !ok {
					res, err = m.apply(ctx, q, mig)
					return err
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

// Down rolls back the n most recently applied migrations (n >= 1), newest
// first, each in its own transaction.
func (m *Migrator) Down(ctx context.Context, n int) ([]Result, error) {
	if n < 1 {
		return nil, errs.UsageError("migrate: down needs a step count of at least 1")
	}
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	var done []Result
	for len(done) < n {
		var res *Result
		err := m.locked(ctx, func(q querier) error {
			var err error
			res, err = m.rollbackLatest(ctx, q)
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

// Redo rolls back the most recently applied migration and applies it again,
// in one transaction: if the up section fails, the rollback is undone too.
func (m *Migrator) Redo(ctx context.Context) ([]Result, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	var done []Result
	err := m.locked(ctx, func(q querier) error {
		down, err := m.rollbackLatest(ctx, q)
		if err != nil || down == nil {
			return err
		}
		up, err := m.apply(ctx, q, down.Migration)
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

// Status lists every migration known from the files or the database, in
// version order. It does not create the bookkeeping table.
func (m *Migrator) Status(ctx context.Context) ([]Status, error) {
	db := m.db.SQLDB()
	var n int
	if err := db.QueryRowContext(ctx, m.dialect.tableExists, m.table).Scan(&n); err != nil {
		return nil, fmt.Errorf("migrate: status: %w", err)
	}
	applied := map[string]record{}
	if n > 0 {
		var err error
		if applied, err = m.applied(ctx, db); err != nil {
			return nil, err
		}
	}
	var out []Status
	files := map[string]bool{}
	for _, mig := range m.migrations {
		key := normVersion(mig.Version)
		files[key] = true
		st := Status{Version: mig.Version, Name: mig.Name, State: StatePending}
		if rec, ok := applied[key]; ok {
			st.State, st.AppliedAt = StateApplied, rec.appliedAt
			if rec.checksum != mig.Checksum {
				st.State = StateModified
			}
		}
		out = append(out, st)
	}
	for key, rec := range applied {
		if !files[key] {
			out = append(out, Status{Version: rec.version, Name: rec.name, State: StateMissing, AppliedAt: rec.appliedAt})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return versionLess(out[i].Version, out[j].Version) })
	return out, nil
}

// apply runs mig's up section and records it, inside the caller's transaction.
func (m *Migrator) apply(ctx context.Context, q querier, mig Migration) (*Result, error) {
	start := time.Now()
	if !isBlank(mig.Up) {
		if _, err := q.ExecContext(ctx, mig.Up); err != nil {
			return nil, fmt.Errorf("migrate: %s up: %w", mig.File, err)
		}
	}
	if _, err := q.ExecContext(ctx, m.dialect.insert, mig.Version, mig.Name, mig.Checksum, time.Now().UTC()); err != nil {
		return nil, fmt.Errorf("migrate: record %s: %w", mig.File, err)
	}
	return &Result{Migration: mig, Direction: DirectionUp, Duration: time.Since(start)}, nil
}

// rollbackLatest runs the down section of the newest applied migration and
// deletes its row. It returns nil when nothing is applied.
func (m *Migrator) rollbackLatest(ctx context.Context, q querier) (*Result, error) {
	applied, err := m.verified(ctx, q)
	if err != nil || len(applied) == 0 {
		return nil, err
	}
	var latest record
	for _, rec := range applied {
		if latest.version == "" || versionLess(latest.version, rec.version) {
			latest = rec
		}
	}
	mig, ok := m.find(latest.version)
	if !ok {
		return nil, errs.WithHint(fmt.Errorf("%w: %s_%s", ErrMissingFile, latest.version, latest.name),
			"restore the file from version control before rolling it back")
	}
	if isBlank(mig.Down) {
		return nil, errs.WithHint(fmt.Errorf("%w: %s", ErrIrreversible, mig.File),
			"write the reverting SQL under -- +down, or add a new migration instead of rolling back")
	}
	start := time.Now()
	if _, err := q.ExecContext(ctx, mig.Down); err != nil {
		return nil, fmt.Errorf("migrate: %s down: %w", mig.File, err)
	}
	if _, err := q.ExecContext(ctx, m.dialect.delete, latest.version); err != nil {
		return nil, fmt.Errorf("migrate: unrecord %s: %w", mig.File, err)
	}
	return &Result{Migration: mig, Direction: DirectionDown, Duration: time.Since(start)}, nil
}

// verified loads the applied rows and fails when an applied file was edited.
func (m *Migrator) verified(ctx context.Context, q querier) (map[string]record, error) {
	applied, err := m.applied(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, mig := range m.migrations {
		if rec, ok := applied[normVersion(mig.Version)]; ok && rec.checksum != mig.Checksum {
			return nil, errs.WithHint(fmt.Errorf("%w: %s", ErrChecksumMismatch, mig.File),
				"restore the applied file and put the change in a new migration (kiw make migration <name>)")
		}
	}
	return applied, nil
}

func (m *Migrator) find(version string) (Migration, bool) {
	for _, mig := range m.migrations {
		if normVersion(mig.Version) == normVersion(version) {
			return mig, true
		}
	}
	return Migration{}, false
}

// ensureTable creates the bookkeeping table under the lock, so two runners
// starting on an empty database do not race on CREATE TABLE.
func (m *Migrator) ensureTable(ctx context.Context) error {
	return m.locked(ctx, func(q querier) error {
		if _, err := q.ExecContext(ctx, m.dialect.create); err != nil {
			return fmt.Errorf("migrate: create %s: %w", m.table, err)
		}
		return nil
	})
}

// fnvKey derives a stable advisory-lock key from the table name.
func fnvKey(table string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("krewire/migrate:" + table))
	return int64(h.Sum64())
}

// querier is the subset of *sql.Conn and *sql.Tx the runner uses.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
