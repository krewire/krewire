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
//     also covers transactions that read before they write.
//
// The driver is opt-in: pkg/orm itself never imports it, so programs that use
// another database do not link SQLite.
package sqlite

import (
	"strings"

	"github.com/krewire/krewire/pkg/orm"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// DriverName is the database/sql driver name registered by this package.
const DriverName = "sqlite"

// DefaultTxLock is the BEGIN mode Config applies when TxLock is empty.
const DefaultTxLock = "immediate"

// Config returns the orm.SQLiteConfig for path with this package's defaults:
// the bundled driver, WAL, foreign keys on, a 5s busy timeout, and immediate
// transactions. Adjust the returned value before passing it to OpenConfig.
// A path of "" or ":memory:" selects a private in-memory database.
func Config(path string) orm.SQLiteConfig {
	return orm.SQLiteConfig{
		Path:        path,
		JournalMode: "WAL",
		ForeignKeys: true,
		DriverName:  DriverName,
		TxLock:      DefaultTxLock,
	}
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
