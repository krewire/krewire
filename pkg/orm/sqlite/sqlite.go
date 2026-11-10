// Package sqlite bundles a pure-Go SQLite driver for pkg/orm.
//
// Importing this package registers modernc.org/sqlite under the driver name
// "sqlite" (no cgo, so CGO_ENABLED=0 cross-compiles keep working) and offers
// Open and DSN helpers with production-friendly defaults:
//
//   - journal_mode=WAL: readers do not block the writer;
//   - foreign_keys=on: REFERENCES clauses are enforced;
//   - busy_timeout=5000: a locked database is retried for 5s instead of
//     failing at once with "database is locked";
//   - _txlock=immediate: BEGIN takes the write lock up front, so busy_timeout
//     also covers transactions that read before they write;
//   - synchronous=NORMAL: zero data loss in WAL mode while maximizing disk IOPS;
//   - cache_size=-64000: 64 MB thread-safe in-memory page cache;
//   - temp_store=MEMORY: transient sorting and indices reside in memory.
//
// The driver is opt-in: pkg/orm itself never imports it, so programs that use
// another database do not link SQLite.
package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// DriverName is the database/sql driver name registered by this package.
const DriverName = "sqlite"

// DefaultTxLock is the BEGIN mode Config applies when TxLock is empty.
const DefaultTxLock = "immediate"

// CheckpointMode defines the WAL checkpoint mode.
type CheckpointMode string

const (
	// CheckpointPassive checkpoints as many frames as possible without blocking.
	CheckpointPassive CheckpointMode = "PASSIVE"

	// CheckpointFull blocks until all current readers finish and checkpoints all frames.
	CheckpointFull CheckpointMode = "FULL"

	// CheckpointRestart blocks like FULL and ensures subsequent writers start at frame 1.
	CheckpointRestart CheckpointMode = "RESTART"

	// CheckpointTruncate blocks, checkpoints everything, and truncates the WAL file to 0 bytes.
	CheckpointTruncate CheckpointMode = "TRUNCATE"
)

// Config returns the orm.SQLiteConfig for path with standard safe defaults:
// the bundled driver, WAL, foreign keys on, a 5s busy timeout, and immediate
// transactions. Adjust the returned value before passing it to OpenConfig.
// A path of "" or ":memory:" selects a private in-memory database.
func Config(path string) orm.SQLiteConfig {
	return orm.SQLiteConfig{
		Path:            path,
		JournalMode:     "WAL",
		ForeignKeys:     true,
		DriverName:      DriverName,
		TxLock:          DefaultTxLock,
		BusyTimeout:     5 * time.Second,
		Synchronous:     "NORMAL",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// ProductionConfig returns an enterprise-tuned orm.SQLiteConfig:
// WAL mode, synchronous NORMAL, 64MB memory page cache, in-memory temp tables,
// 256MB memory-mapped I/O, and incremental auto-vacuum.
func ProductionConfig(path string) orm.SQLiteConfig {
	cfg := Config(path)
	cfg.CacheSize = -64000           // 64 MB page cache
	cfg.TempStore = "MEMORY"         // memory temp store
	cfg.MmapSize = 256 * 1024 * 1024 // 256 MB mmap
	cfg.AutoVacuum = "INCREMENTAL"
	return cfg
}

// DSN returns the connection string for path with the defaults of Config,
// for use with orm.Open(sqlite.DriverName, dsn) or database/sql directly.
func DSN(path string) string {
	return Config(path).FormatDSN()
}

// Open opens (creating it when missing) the SQLite database at path with the
// defaults of Config.
func Open(path string) (*orm.DB, error) {
	return OpenConfig(Config(path))
}

// OpenProduction opens a SQLite database optimized for heavy production workloads.
func OpenProduction(path string) (*orm.DB, error) {
	return OpenConfig(ProductionConfig(path))
}

// OpenConfig opens a SQLite database from cfg. An empty DriverName selects the
// bundled driver; an empty TxLock selects DefaultTxLock.
func OpenConfig(cfg orm.SQLiteConfig) (*orm.DB, error) {
	if strings.TrimSpace(cfg.DriverName) == "" {
		cfg.DriverName = DriverName
	}
	if strings.TrimSpace(cfg.TxLock) == "" {
		cfg.TxLock = DefaultTxLock
	}
	return orm.NewSQLite(cfg)
}

// Checkpoint executes a PRAGMA wal_checkpoint command to flush WAL frames back to the main DB.
func Checkpoint(ctx context.Context, db *orm.DB, mode CheckpointMode) error {
	if db == nil {
		return errs.UsageError("sqlite: db connection must not be nil")
	}
	if mode == "" {
		mode = CheckpointPassive
	}
	query := fmt.Sprintf("PRAGMA wal_checkpoint(%s)", mode)
	_, err := db.Exec(ctx, query)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("sqlite checkpoint (%s) failed: %v", mode, err))
	}
	return nil
}

// Vacuum executes the VACUUM command to defragment and optimize the database file.
func Vacuum(ctx context.Context, db *orm.DB) error {
	if db == nil {
		return errs.UsageError("sqlite: db connection must not be nil")
	}
	_, err := db.Exec(ctx, "VACUUM")
	if err != nil {
		return errs.FailureError(fmt.Sprintf("sqlite vacuum failed: %v", err))
	}
	return nil
}

// IncrementalVacuum frees unused database pages back to the filesystem.
func IncrementalVacuum(ctx context.Context, db *orm.DB, pages int) error {
	if db == nil {
		return errs.UsageError("sqlite: db connection must not be nil")
	}
	var query string
	if pages > 0 {
		query = fmt.Sprintf("PRAGMA incremental_vacuum(%d)", pages)
	} else {
		query = "PRAGMA incremental_vacuum"
	}
	_, err := db.Exec(ctx, query)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("sqlite incremental vacuum failed: %v", err))
	}
	return nil
}

// Optimize executes PRAGMA optimize to refresh query planner statistics.
func Optimize(ctx context.Context, db *orm.DB) error {
	if db == nil {
		return errs.UsageError("sqlite: db connection must not be nil")
	}
	_, err := db.Exec(ctx, "PRAGMA optimize")
	if err != nil {
		return errs.FailureError(fmt.Sprintf("sqlite optimize failed: %v", err))
	}
	return nil
}

// IntegrityCheck runs PRAGMA integrity_check and returns any diagnostics or errors.
func IntegrityCheck(ctx context.Context, db *orm.DB) ([]string, error) {
	if db == nil {
		return nil, errs.UsageError("sqlite: db connection must not be nil")
	}
	rows, err := db.Query(ctx, "PRAGMA integrity_check")
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("sqlite integrity check query failed: %v", err))
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var msg string
		if err := rows.Scan(&msg); err != nil {
			return nil, errs.FailureError(fmt.Sprintf("sqlite scan integrity error: %v", err))
		}
		messages = append(messages, msg)
	}

	if len(messages) == 1 && messages[0] == "ok" {
		return messages, nil
	}
	return messages, errs.FailureError(fmt.Sprintf("sqlite integrity issues detected: %s", strings.Join(messages, "; ")))
}

// QuickCheck runs a faster integrity scan (skipping index consistency verify).
func QuickCheck(ctx context.Context, db *orm.DB) ([]string, error) {
	if db == nil {
		return nil, errs.UsageError("sqlite: db connection must not be nil")
	}
	rows, err := db.Query(ctx, "PRAGMA quick_check")
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("sqlite quick check query failed: %v", err))
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var msg string
		if err := rows.Scan(&msg); err != nil {
			return nil, errs.FailureError(fmt.Sprintf("sqlite scan quick check error: %v", err))
		}
		messages = append(messages, msg)
	}

	if len(messages) == 1 && messages[0] == "ok" {
		return messages, nil
	}
	return messages, errs.FailureError(fmt.Sprintf("sqlite quick check issues detected: %s", strings.Join(messages, "; ")))
}
