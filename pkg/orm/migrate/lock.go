package migrate

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
)

// dialect holds the SQL the runner needs for one database engine.
type dialect struct {
	name        string
	create      string
	selectAll   string
	insert      string
	delete      string
	tableExists string
	lockKey     int64
}

func dialectFor(d orm.Dialect, table string) (dialect, error) {
	if d == nil {
		return dialect{}, fmt.Errorf("%w: database has no dialect", ErrUnsupportedDialect)
	}
	q := d.QuoteIdentifier(table)
	p := d.Placeholder
	out := dialect{
		name:      d.Name(),
		selectAll: "SELECT version, name, checksum, applied_at FROM " + q,
		insert: fmt.Sprintf("INSERT INTO %s (version, name, checksum, applied_at) VALUES (%s, %s, %s, %s)",
			q, p(1), p(2), p(3), p(4)),
		delete:  fmt.Sprintf("DELETE FROM %s WHERE version = %s", q, p(1)),
		lockKey: fnvKey(table),
	}
	columns := "version TEXT PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at %s NOT NULL"
	switch out.name {
	case "sqlite":
		out.create = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ("+columns+")", q, "TIMESTAMP")
		out.tableExists = "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?"
	case "postgres":
		out.create = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ("+columns+")", q, "TIMESTAMPTZ")
		out.tableExists = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1"
	default:
		// MySQL commits DDL implicitly, so a failed migration cannot be rolled
		// back with its bookkeeping row; refuse rather than half-apply.
		return dialect{}, errs.WithHint(fmt.Errorf("%w %q", ErrUnsupportedDialect, out.name),
			"migrate supports sqlite and postgres")
	}
	return out, nil
}

// locked runs fn in one transaction that holds the migration lock, on a
// connection reserved for the call. It commits when fn succeeds and rolls
// back otherwise.
//
//   - SQLite: BEGIN IMMEDIATE takes the database write lock up front; a
//     concurrent runner waits up to LockTimeout (as busy_timeout) for it.
//   - PostgreSQL: a transaction-scoped advisory lock, polled until
//     LockTimeout; it is released by COMMIT or ROLLBACK, so a crashed runner
//     never leaves a stale lock behind.
func (m *Migrator) locked(ctx context.Context, fn func(q querier) error) (err error) {
	conn, err := m.db.SQLDB().Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate: connect: %w", err)
	}
	defer conn.Close()

	if m.dialect.name == "sqlite" {
		return m.lockedSQLite(ctx, conn, fn)
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate: begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err := m.advisoryLock(ctx, tx); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate: commit: %w", err)
	}
	return nil
}

func (m *Migrator) advisoryLock(ctx context.Context, tx *sql.Tx) error {
	deadline := time.Now().Add(m.lockTimeout)
	for {
		var ok bool
		if err := tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", m.dialect.lockKey).Scan(&ok); err != nil {
			return fmt.Errorf("migrate: lock: %w", err)
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return errs.WithHint(fmt.Errorf("%w (waited %s)", ErrLocked, m.lockTimeout),
				"another process is migrating this database; retry once it finishes")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (m *Migrator) lockedSQLite(ctx context.Context, conn *sql.Conn, fn func(q querier) error) (err error) {
	// busy_timeout is per connection: raise it for this call, restore it after.
	var prev int64
	if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&prev); err != nil {
		return fmt.Errorf("migrate: read busy_timeout: %w", err)
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", m.lockTimeout.Milliseconds())); err != nil {
		return fmt.Errorf("migrate: set busy_timeout: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), fmt.Sprintf("PRAGMA busy_timeout = %d", prev))
	}()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		if isBusy(err) {
			return errs.WithHint(fmt.Errorf("%w (waited %s): %v", ErrLocked, m.lockTimeout, err),
				"another process is writing to this database; retry once it finishes")
		}
		return fmt.Errorf("migrate: begin: %w", err)
	}
	end := func(stmt string) error {
		_, e := conn.ExecContext(context.WithoutCancel(ctx), stmt)
		return e
	}
	if err := fn(conn); err != nil {
		if rbErr := end("ROLLBACK"); rbErr != nil {
			discard(conn)
		}
		return err
	}
	if err := end("COMMIT"); err != nil {
		if rbErr := end("ROLLBACK"); rbErr != nil {
			discard(conn)
		}
		return fmt.Errorf("migrate: commit: %w", err)
	}
	return nil
}

// discard keeps a connection whose transaction state is unknown out of the
// pool.
func discard(conn *sql.Conn) {
	_ = conn.Raw(func(any) error { return driver.ErrBadConn })
}

func isBusy(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}

// record is one row of the bookkeeping table.
type record struct {
	version, name, checksum string
	appliedAt               time.Time
}

// applied returns the bookkeeping rows keyed by normalized version.
func (m *Migrator) applied(ctx context.Context, q querier) (map[string]record, error) {
	rows, err := q.QueryContext(ctx, m.dialect.selectAll)
	if err != nil {
		return nil, fmt.Errorf("migrate: read %s: %w", m.table, err)
	}
	defer rows.Close()
	out := map[string]record{}
	for rows.Next() {
		var rec record
		var at any
		if err := rows.Scan(&rec.version, &rec.name, &rec.checksum, &at); err != nil {
			return nil, fmt.Errorf("migrate: read %s: %w", m.table, err)
		}
		rec.appliedAt = toTime(at)
		out[normVersion(rec.version)] = rec
	}
	return out, rows.Err()
}

// toTime converts applied_at as returned by the driver. SQLite drivers may
// hand back a string instead of a time.Time depending on the column type.
func toTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case []byte:
		return toTime(string(t))
	case string:
		for _, layout := range []string{
			"2006-01-02 15:04:05.999999999-07:00",
			time.RFC3339Nano,
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
		} {
			if parsed, err := time.Parse(layout, t); err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}
