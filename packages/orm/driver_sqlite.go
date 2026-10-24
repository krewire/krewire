package orm

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// SQLiteConfig contains connection and pragma options for SQLite databases.
type SQLiteConfig struct {
	Path            string        // File path or ":memory:"
	InMemory        bool          // If true, forces ":memory:"
	JournalMode     string        // e.g. "WAL", "DELETE", "TRUNCATE"
	ForeignKeys     bool          // Enables PRAGMA foreign_keys = ON (default: true)
	BusyTimeout     time.Duration // Lock timeout before returning busy (default: 5s)
	DriverName      string        // Driver to use (default: "sqlite3" or "sqlite")
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// FormatDSN generates a standard SQLite connection string with query pragmas.
func (c SQLiteConfig) FormatDSN() string {
	if c.InMemory || strings.TrimSpace(c.Path) == "" || c.Path == ":memory:" {
		return ":memory:"
	}

	params := url.Values{}

	journal := strings.ToUpper(strings.TrimSpace(c.JournalMode))
	if journal == "" {
		journal = "WAL"
	}
	params.Set("_journal_mode", journal)

	if c.ForeignKeys || c.JournalMode == "" {
		params.Set("_foreign_keys", "on")
	}

	timeoutMs := 5000
	if c.BusyTimeout > 0 {
		timeoutMs = int(c.BusyTimeout.Milliseconds())
	}
	params.Set("_busy_timeout", fmt.Sprintf("%d", timeoutMs))

	if strings.Contains(c.Path, "?") {
		return fmt.Sprintf("%s&%s", c.Path, params.Encode())
	}
	return fmt.Sprintf("%s?%s", c.Path, params.Encode())
}

// NewSQLite creates and opens a DB connection pool configured for SQLite.
func NewSQLite(cfg SQLiteConfig) (*DB, error) {
	driver := strings.TrimSpace(cfg.DriverName)
	if driver == "" {
		driver = "sqlite3"
	}

	dsn := cfg.FormatDSN()

	dbCfg := Config{
		DriverName:      driver,
		DSN:             dsn,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	// SQLite typically performs best with serialized or single-writer concurrency in memory
	if cfg.InMemory && dbCfg.MaxOpenConns == 0 {
		dbCfg.MaxOpenConns = 1
	}

	return OpenWithConfig(dbCfg)
}
