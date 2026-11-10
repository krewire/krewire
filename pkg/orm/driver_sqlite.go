package orm

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultSQLiteInMemory is the sentinel DSN for an in-memory SQLite database.
	DefaultSQLiteInMemory = ":memory:"

	// DefaultSQLiteJournalMode is the default high-concurrency journal mode.
	DefaultSQLiteJournalMode = "WAL"

	// DefaultSQLiteBusyTimeout is the default wait timeout when database is locked.
	DefaultSQLiteBusyTimeout = 5 * time.Second

	// DefaultSQLiteDriverName is the fallback driver name for sqlite3.
	DefaultSQLiteDriverName = "sqlite3"
)

// SQLiteConfig contains connection and pragma options for SQLite databases.
type SQLiteConfig struct {
	Path            string        // File path or DefaultSQLiteInMemory (":memory:")
	InMemory        bool          // If true, forces DefaultSQLiteInMemory
	JournalMode     string        // e.g. DefaultSQLiteJournalMode ("WAL"), "DELETE", "TRUNCATE"
	ForeignKeys     bool          // Enables PRAGMA foreign_keys = ON (default: true)
	BusyTimeout     time.Duration // Lock timeout before returning busy (default: DefaultSQLiteBusyTimeout)
	DriverName      string        // Driver to use (default: DefaultSQLiteDriverName)
	TxLock          string        // BEGIN mode: "deferred", "immediate", or "exclusive" (default: driver default, deferred)
	Synchronous     string        // e.g. "NORMAL", "FULL", "OFF" (default: "NORMAL" in WAL mode)
	CacheSize       int           // Cache size in KiB (negative, e.g. -64000 = 64MB) or pages (positive)
	TempStore       string        // e.g. "MEMORY", "FILE"
	MmapSize        int64         // Maximum memory-mapped I/O address space size in bytes
	AutoVacuum      string        // e.g. "NONE", "FULL", "INCREMENTAL"
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// FormatDSN generates a standard SQLite connection string with query pragmas.
func (c SQLiteConfig) FormatDSN() string {
	if c.InMemory || strings.TrimSpace(c.Path) == "" || c.Path == DefaultSQLiteInMemory {
		return DefaultSQLiteInMemory
	}

	params := url.Values{}

	journal := strings.ToUpper(strings.TrimSpace(c.JournalMode))
	if journal == "" {
		journal = DefaultSQLiteJournalMode
	}
	params.Set("_journal_mode", journal)

	if c.ForeignKeys || c.JournalMode == "" {
		params.Set("_foreign_keys", "on")
	}

	timeoutMs := int(DefaultSQLiteBusyTimeout.Milliseconds())
	if c.BusyTimeout > 0 {
		timeoutMs = int(c.BusyTimeout.Milliseconds())
	}
	params.Set("_busy_timeout", fmt.Sprintf("%d", timeoutMs))

	if lock := strings.ToLower(strings.TrimSpace(c.TxLock)); lock != "" {
		params.Set("_txlock", lock)
	}

	if syncMode := strings.ToUpper(strings.TrimSpace(c.Synchronous)); syncMode != "" {
		params.Set("_synchronous", syncMode)
	}

	if c.CacheSize != 0 {
		params.Set("_cache_size", fmt.Sprintf("%d", c.CacheSize))
	}

	if temp := strings.ToUpper(strings.TrimSpace(c.TempStore)); temp != "" {
		params.Set("_temp_store", temp)
	}

	if c.MmapSize > 0 {
		params.Set("_mmap_size", fmt.Sprintf("%d", c.MmapSize))
	}

	if autoVac := strings.ToUpper(strings.TrimSpace(c.AutoVacuum)); autoVac != "" {
		params.Set("_auto_vacuum", autoVac)
	}

	if strings.Contains(c.Path, "?") {
		return fmt.Sprintf("%s&%s", c.Path, params.Encode())
	}
	return fmt.Sprintf("%s?%s", c.Path, params.Encode())
}

// NewSQLite creates and opens a DB connection pool configured for SQLite.
func NewSQLite(cfg SQLiteConfig) (*DB, error) {
	driver := strings.TrimSpace(cfg.DriverName)
	if driver == "" {
		driver = DefaultSQLiteDriverName
	}

	dsn := cfg.FormatDSN()

	dbCfg := Config{
		DriverName:      driver,
		DSN:             dsn,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	}

	// SQLite typically performs best with serialized or single-writer concurrency in memory
	if cfg.InMemory && dbCfg.MaxOpenConns == 0 {
		dbCfg.MaxOpenConns = 1
	}

	return OpenWithConfig(dbCfg)
}
