package orm

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/packages/errs"
)

// Dialect abstracts database engine differences (SQL syntax, placeholders, escaping, types).
type Dialect interface {
	Name() string
	QuoteIdentifier(name string) string
	Placeholder(index int) string
	DataTypeOf(typ reflect.Type) string
	AutoIncrementPrimaryKeyClause() string
}

var (
	dialectMu sync.RWMutex
	dialects  = map[string]Dialect{
		"sqlite":     &SQLiteDialect{},
		"sqlite3":    &SQLiteDialect{},
		"postgres":   &PostgresDialect{},
		"postgresql": &PostgresDialect{},
		"pgx":        &PostgresDialect{},
		"mysql":      &MySQLDialect{},
		"mariadb":    &MySQLDialect{},
	}
)

// RegisterDialect registers a custom Dialect for a driver name.
func RegisterDialect(driverName string, dialect Dialect) {
	dialectMu.Lock()
	defer dialectMu.Unlock()
	dialects[strings.ToLower(driverName)] = dialect
}

// GetDialect retrieves the Dialect for a driver name, defaulting to SQLiteDialect if not found.
func GetDialect(driverName string) Dialect {
	dialectMu.RLock()
	defer dialectMu.RUnlock()
	if d, ok := dialects[strings.ToLower(strings.TrimSpace(driverName))]; ok {
		return d
	}
	return &SQLiteDialect{}
}

// ----------------------------------------------------------------------------
// SQLite Dialect
// ----------------------------------------------------------------------------

// SQLiteDialect implements Dialect for SQLite engines.
type SQLiteDialect struct{}

func (d *SQLiteDialect) Name() string {
	return "sqlite"
}

func (d *SQLiteDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

func (d *SQLiteDialect) Placeholder(index int) string {
	return "?"
}

func (d *SQLiteDialect) AutoIncrementPrimaryKeyClause() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (d *SQLiteDialect) DataTypeOf(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "INTEGER"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.String:
		return "TEXT"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "BLOB"
		}
		return "TEXT"
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return "DATETIME"
		}
		return "TEXT"
	default:
		return "TEXT"
	}
}

// ----------------------------------------------------------------------------
// PostgreSQL Dialect
// ----------------------------------------------------------------------------

// PostgresDialect implements Dialect for PostgreSQL engines.
type PostgresDialect struct{}

func (d *PostgresDialect) Name() string {
	return "postgres"
}

func (d *PostgresDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

func (d *PostgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (d *PostgresDialect) AutoIncrementPrimaryKeyClause() string {
	return "BIGSERIAL PRIMARY KEY"
}

func (d *PostgresDialect) DataTypeOf(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.String:
		return "TEXT"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "BYTEA"
		}
		return "TEXT"
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return "TIMESTAMPTZ"
		}
		return "JSONB"
	default:
		return "TEXT"
	}
}

// ----------------------------------------------------------------------------
// MySQL Dialect
// ----------------------------------------------------------------------------

// MySQLDialect implements Dialect for MySQL and MariaDB engines.
type MySQLDialect struct{}

func (d *MySQLDialect) Name() string {
	return "mysql"
}

func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(name, "`", "``"))
}

func (d *MySQLDialect) Placeholder(index int) string {
	return "?"
}

func (d *MySQLDialect) AutoIncrementPrimaryKeyClause() string {
	return "BIGINT AUTO_INCREMENT PRIMARY KEY"
}

func (d *MySQLDialect) DataTypeOf(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "TINYINT(1)"
	case reflect.Int, reflect.Int32:
		return "INT"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Float32:
		return "FLOAT"
	case reflect.Float64:
		return "DOUBLE"
	case reflect.String:
		return "VARCHAR(255)"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "LONGBLOB"
		}
		return "TEXT"
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return "DATETIME(6)"
		}
		return "JSON"
	default:
		return "TEXT"
	}
}

// ValidateDialect checks if a dialect is known.
func ValidateDialect(name string) error {
	dialectMu.RLock()
	defer dialectMu.RUnlock()
	if _, ok := dialects[strings.ToLower(strings.TrimSpace(name))]; !ok {
		return errs.UsageError(fmt.Sprintf("unsupported or unregistered database dialect %q", name))
	}
	return nil
}
