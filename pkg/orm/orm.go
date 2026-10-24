package orm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

// Config defines connection pool parameters and driver configuration.
type Config struct {
	DriverName      string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DB represents a database connection pool with multi-driver dialect support.
type DB struct {
	sqlDB   *sql.DB
	dialect Dialect
	driver  string
}

// Open creates a new DB connection pool using a registered sql driver name and DSN.
func Open(driverName, dsn string) (*DB, error) {
	return OpenWithConfig(Config{
		DriverName: driverName,
		DSN:        dsn,
	})
}

// OpenWithConfig creates a DB instance with custom pool configuration.
func OpenWithConfig(cfg Config) (*DB, error) {
	driverName := strings.TrimSpace(cfg.DriverName)
	if driverName == "" {
		return nil, errs.UsageError("orm: driver name is required")
	}

	sqlDB, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: failed to open driver %s: %v", driverName, err))
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	dialect := GetDialect(driverName)

	return &DB{
		sqlDB:   sqlDB,
		dialect: dialect,
		driver:  driverName,
	}, nil
}

// New wraps an existing standard *sql.DB instance with a dialect.
func New(sqlDB *sql.DB, driverName string) *DB {
	return &DB{
		sqlDB:   sqlDB,
		dialect: GetDialect(driverName),
		driver:  driverName,
	}
}

// Close releases connection pool resources.
func (db *DB) Close() error {
	if db.sqlDB == nil {
		return nil
	}
	return db.sqlDB.Close()
}

// SQLDB returns the underlying *sql.DB.
func (db *DB) SQLDB() *sql.DB {
	return db.sqlDB
}

// Dialect returns the active database dialect.
func (db *DB) Dialect() Dialect {
	return db.dialect
}

// Ping verifies a connection to the database is still alive.
func (db *DB) Ping(ctx context.Context) error {
	if db.sqlDB == nil {
		return errs.FailureError("orm: database connection is nil")
	}
	return db.sqlDB.PingContext(ctx)
}

// Exec executes a query without returning rows.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if db.sqlDB == nil {
		return nil, errs.FailureError("orm: database connection is nil")
	}
	return db.sqlDB.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if db.sqlDB == nil {
		return nil, errs.FailureError("orm: database connection is nil")
	}
	return db.sqlDB.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if db.sqlDB == nil {
		return nil
	}
	return db.sqlDB.QueryRowContext(ctx, query, args...)
}

// Tx represents an active database transaction.
type Tx struct {
	sqlTx   *sql.Tx
	dialect Dialect
}

// Begin starts a transaction.
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	if db.sqlDB == nil {
		return nil, errs.FailureError("orm: database connection is nil")
	}
	tx, err := db.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, errs.FailureError(fmt.Sprintf("orm: failed to begin transaction: %v", err))
	}
	return &Tx{sqlTx: tx, dialect: db.dialect}, nil
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	if tx.sqlTx == nil {
		return errs.FailureError("orm: transaction is nil")
	}
	return tx.sqlTx.Commit()
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback() error {
	if tx.sqlTx == nil {
		return errs.FailureError("orm: transaction is nil")
	}
	return tx.sqlTx.Rollback()
}

// Transaction runs a closure within a transaction, committing on success or rolling back on error/panic.
func (db *DB) Transaction(ctx context.Context, fn func(tx *Tx) error) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// Table creates a Query scoped to a table name.
func (db *DB) Table(name string) *Query {
	return newQuery(db, name, nil)
}

// Model creates a Query scoped to a Go model struct.
func (db *DB) Model(v interface{}) *Query {
	tm, _ := ParseModel(v)
	tableName := ""
	if tm != nil {
		tableName = tm.TableName
	}
	return newQuery(db, tableName, v)
}
