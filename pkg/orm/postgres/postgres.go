// Package postgres bundles a pure-Go PostgreSQL driver for pkg/orm based on pgx.
//
// Importing this package registers github.com/jackc/pgx/v5/stdlib under the
// driver name "pgx" (pure Go, CGO_ENABLED=0 compliant) and provides
// production-ready configuration, connection pooling, and maintenance helpers.
//
// The driver is opt-in: pkg/orm itself never imports it, preserving independent
// modularity (Pillar 2: Sustainable).
package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// DriverName is the database/sql driver name registered by pgx/v5/stdlib.
const DriverName = "pgx"

// DefaultPort is the standard PostgreSQL port.
const DefaultPort = 5432

// Config returns a baseline orm.PostgresConfig with safe defaults:
// SSLMode disabled (for local dev), UTC timezone, public search path.
func Config(host string, port int, user, password, database string) orm.PostgresConfig {
	if port <= 0 {
		port = DefaultPort
	}
	if host == "" {
		host = "localhost"
	}
	if user == "" {
		user = "postgres"
	}
	if database == "" {
		database = "postgres"
	}

	return orm.PostgresConfig{
		Host:                            host,
		Port:                            port,
		Database:                        database,
		User:                            user,
		Password:                        password,
		SSLMode:                         "disable",
		TimeZone:                        "UTC",
		DriverName:                      DriverName,
		SearchPath:                      "public",
		ApplicationName:                 "krewire",
		ConnectTimeout:                  5 * time.Second,
		StatementTimeout:                30 * time.Second,
		IdleInTransactionSessionTimeout: 60 * time.Second,
		MaxOpenConns:                    25,
		MaxIdleConns:                    5,
		ConnMaxLifetime:                 10 * time.Minute,
		ConnMaxIdleTime:                 2 * time.Minute,
	}
}

// ProductionConfig returns an enterprise-tuned PostgreSQL configuration:
// SSLMode require, pool limits, prepared statement caching, and bounded timeouts.
func ProductionConfig(host string, port int, user, password, database string) orm.PostgresConfig {
	cfg := Config(host, port, user, password, database)
	cfg.SSLMode = "require"
	cfg.MaxOpenConns = 50
	cfg.MaxIdleConns = 15
	cfg.ConnMaxLifetime = 15 * time.Minute
	cfg.ConnMaxIdleTime = 3 * time.Minute
	cfg.ConnectTimeout = 10 * time.Second
	cfg.StatementTimeout = 45 * time.Second
	cfg.IdleInTransactionSessionTimeout = 30 * time.Second
	return cfg
}

// DSN returns the formatted PostgreSQL connection URL string from cfg.
func DSN(cfg orm.PostgresConfig) string {
	if strings.TrimSpace(cfg.DriverName) == "" {
		cfg.DriverName = DriverName
	}
	return cfg.FormatDSN()
}

// Open opens a PostgreSQL database connection using a formatted DSN string.
func Open(dsn string) (*orm.DB, error) {
	return orm.OpenWithConfig(orm.Config{
		DriverName: DriverName,
		DSN:        dsn,
	})
}

// OpenConfig opens a PostgreSQL database connection using a structured orm.PostgresConfig.
func OpenConfig(cfg orm.PostgresConfig) (*orm.DB, error) {
	if strings.TrimSpace(cfg.DriverName) == "" {
		cfg.DriverName = DriverName
	}
	return orm.NewPostgres(cfg)
}

// OpenProduction opens an enterprise-grade connection pool for PostgreSQL.
func OpenProduction(host string, port int, user, password, database string) (*orm.DB, error) {
	return OpenConfig(ProductionConfig(host, port, user, password, database))
}

// ServerVersion queries the connected PostgreSQL server version string.
func ServerVersion(ctx context.Context, db *orm.DB) (string, error) {
	if db == nil {
		return "", errs.UsageError("postgres: db connection must not be nil")
	}
	var version string
	if err := db.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		return "", errs.FailureError(fmt.Sprintf("postgres select version failed: %v", err))
	}
	return version, nil
}

// CurrentDatabase queries the active database name.
func CurrentDatabase(ctx context.Context, db *orm.DB) (string, error) {
	if db == nil {
		return "", errs.UsageError("postgres: db connection must not be nil")
	}
	var currentDB string
	if err := db.QueryRow(ctx, "SELECT current_database()").Scan(&currentDB); err != nil {
		return "", errs.FailureError(fmt.Sprintf("postgres current_database failed: %v", err))
	}
	return currentDB, nil
}

// SchemaExists checks if a schema exists in information_schema.schemata.
func SchemaExists(ctx context.Context, db *orm.DB, schemaName string) (bool, error) {
	if db == nil {
		return false, errs.UsageError("postgres: db connection must not be nil")
	}
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`
	var exists bool
	if err := db.QueryRow(ctx, query, schemaName).Scan(&exists); err != nil {
		return false, errs.FailureError(fmt.Sprintf("postgres schema_exists check failed: %v", err))
	}
	return exists, nil
}

// TableSize queries the total size on disk for a table including its indexes.
func TableSize(ctx context.Context, db *orm.DB, tableName string) (int64, error) {
	if db == nil {
		return 0, errs.UsageError("postgres: db connection must not be nil")
	}
	query := `SELECT pg_total_relation_size($1)`
	var size int64
	if err := db.QueryRow(ctx, query, tableName).Scan(&size); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("postgres table_size failed: %v", err))
	}
	return size, nil
}

// VacuumAnalyze executes VACUUM (ANALYZE) on a target table to reclaim storage and update planner statistics.
func VacuumAnalyze(ctx context.Context, db *orm.DB, tableName string) error {
	if db == nil {
		return errs.UsageError("postgres: db connection must not be nil")
	}
	var query string
	if tableName != "" {
		query = fmt.Sprintf(`VACUUM (ANALYZE) "%s"`, strings.ReplaceAll(tableName, `"`, `""`))
	} else {
		query = "VACUUM (ANALYZE)"
	}
	_, err := db.Exec(ctx, query)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("postgres vacuum analyze failed: %v", err))
	}
	return nil
}

// TerminateBackend calls pg_terminate_backend(pid) to terminate a runaway query or hung connection.
func TerminateBackend(ctx context.Context, db *orm.DB, pid int) (bool, error) {
	if db == nil {
		return false, errs.UsageError("postgres: db connection must not be nil")
	}
	query := `SELECT pg_terminate_backend($1)`
	var terminated bool
	if err := db.QueryRow(ctx, query, pid).Scan(&terminated); err != nil {
		return false, errs.FailureError(fmt.Sprintf("postgres terminate_backend (%d) failed: %v", pid, err))
	}
	return terminated, nil
}
