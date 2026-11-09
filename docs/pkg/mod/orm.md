# `orm`

Import: `github.com/krewire/krewire/pkg/orm`

## Purpose

A lightweight, pure-Go Object-Relational Mapping (ORM) and SQL Query Builder library with unified multi-driver support. Designed for SQLite, PostgreSQL, and MySQL, `pkg/orm` eliminates external CGO dependencies and abstracts dialect differences (quoting, parameter placeholders `$1` vs `?`, data types, DDL statements, and auto-increment conventions) behind a single fluent API.

## Main API

- **Adapter Abstraction**:
  - `Adapter` interface (`Name() string`, `Type() AdapterType`, `Ping(ctx context.Context) error`, `Close() error`)
  - `AdapterType`: `AdapterTypeSQL`, `AdapterTypeDocument`, `AdapterTypeSearch`
- **Supported Drivers & Adapters**:
  - **SQLite**: `NewSQLite(cfg SQLiteConfig) (*DB, error)`, `cfg.FormatDSN()` (`SQLiteConfig.TxLock` sets the `BEGIN` mode: `deferred`, `immediate`, `exclusive`); bundled driver in [`pkg/orm/sqlite`](#sqlite-driver-pkgormsqlite)
  - **PostgreSQL**: `NewPostgres(cfg PostgresConfig) (*DB, error)`, `cfg.FormatDSN()`
  - **MySQL**: `NewMySQL(cfg MySQLConfig) (*DB, error)`, `cfg.FormatDSN()`
  - **MongoDB**: `NewMongoDB(cfg MongoConfig) (*MongoAdapter, error)`, collections, document CRUD (`InsertOne`, `FindOne`, `Find`, `UpdateOne`, `DeleteOne`, `CountDocuments`)
  - **Elasticsearch**: `NewElasticsearch(cfg ElasticsearchConfig) (*ElasticsearchAdapter, error)`, index management, search query DSL (`Index`, `Get`, `Delete`, `Search`, `CreateIndex`, `DeleteIndex`)
- **SQL Dialects**:
  - `Dialect` interface (`QuoteIdentifier`, `Placeholder`, `DataTypeOf`, `AutoIncrementPrimaryKeyClause`)
  - `SQLiteDialect`, `PostgresDialect`, `MySQLDialect`
  - `RegisterDialect(driverName string, dialect Dialect)`
  - `GetDialect(driverName string) Dialect`
- **Database Connection Pool**:
  - `Open(driverName, dsn string) (*DB, error)`
  - `OpenWithConfig(cfg Config) (*DB, error)`
  - `New(sqlDB *sql.DB, driverName string) *DB`
  - `DB.Close() error`, `DB.Ping(ctx context.Context) error`
  - `DB.Table(name string) *Query`, `DB.Model(v interface{}) *Query`
- **Transactions**:
  - `DB.Begin(ctx context.Context) (*Tx, error)`, `DB.BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error)`
  - `DB.Transaction(ctx context.Context, fn func(tx *Tx) error) error`
  - `Tx.Exec`, `Tx.Query`, `Tx.QueryRow` (same signatures as on `DB`), `Tx.Dialect()`, `Tx.SQLTx() *sql.Tx`, `Tx.Commit()`, `Tx.Rollback()`
- **Fluent Query Builder & CRUD**:
  - `Query.Where(col, op string, val interface{}) *Query`
  - `Query.OrWhere(col, op string, val interface{}) *Query`
  - `Query.WhereIn(col string, vals ...interface{}) *Query`
  - `Query.OrderBy(col, dir string) *Query`
  - `Query.Limit(n int) *Query`, `Query.Offset(n int) *Query`
  - `Query.Insert(record interface{}) (int64, error)`
  - `Query.Find(dest interface{}, id ...interface{}) error`
  - `Query.FindAll(dest interface{}) error`
  - `Query.Update(record interface{}) (int64, error)`
  - `Query.Delete(record interface{}, id ...interface{}) (int64, error)`
  - `Query.Count() (int64, error)`
- **Schema & Migrations**:
  - `ParseModel(v interface{}) (*TableModel, error)`
  - `GenerateCreateTableSQL(model interface{}, dialect Dialect) (string, error)`
  - `DB.AutoMigrate(ctx context.Context, models ...interface{}) error` (creates missing tables only; never alters them)
  - Versioned SQL migrations: [`pkg/orm/migrate`](#migrations-pkgormmigrate)

## Struct Tags

Models declare mapping metadata using the `orm` or `db` struct tag:

```go
type User struct {
    ID        int64     `orm:"pk;auto"`
    Email     string    `orm:"column:email_address;unique;not_null"`
    FullName  string    `orm:"column:full_name"`
    IsActive  bool      `orm:"default:1"`
    CreatedAt time.Time
}

// Optional table name override
func (User) TableName() string {
    return "app_users"
}
```

## Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/krewire/krewire/pkg/orm"
)

type AppUser struct {
	ID    int64  `orm:"pk;auto"`
	Name  string `orm:"column:name"`
	Email string `orm:"column:email;unique"`
}

func main() {
	ctx := context.Background()

	// Open connection (works with any registered sql driver: "sqlite3", "pgx", "mysql", etc.)
	db, err := orm.Open("sqlite", "file:app.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Auto-migrate table
	if err := db.AutoMigrate(ctx, &AppUser{}); err != nil {
		panic(err)
	}

	// Insert record
	newUser := &AppUser{Name: "Dewi", Email: "dewi@example.com"}
	if _, err := db.Table("app_users").Insert(newUser); err != nil {
		panic(err)
	}

	// Query record
	var user AppUser
	if err := db.Table("app_users").Where("email", "=", "dewi@example.com").Find(&user); err != nil {
		panic(err)
	}

	fmt.Printf("Loaded user: ID=%d, Name=%s\n", user.ID, user.Name)
}
```

## SQLite driver (`pkg/orm/sqlite`)

`pkg/orm` does not link any driver. Importing `github.com/krewire/krewire/pkg/orm/sqlite`
registers the pure-Go `modernc.org/sqlite` driver as `"sqlite"` (no cgo, so
`CGO_ENABLED=0` cross-compiles work) and opens databases with WAL,
`foreign_keys=on`, `busy_timeout=5000`, and immediate transactions:

- `sqlite.Open(path string) (*orm.DB, error)` — creates the file when missing; `":memory:"` for a private in-memory database
- `sqlite.OpenConfig(cfg orm.SQLiteConfig) (*orm.DB, error)` — custom pool or pragmas
- `sqlite.Config(path) orm.SQLiteConfig`, `sqlite.DSN(path) string`, `sqlite.DriverName`

```go
db, err := sqlite.Open("app.db")
```

Only binaries opt in: a guard in `layering_test.go` keeps every other package
under `pkg/` from importing the driver. Switching a brand-new file to WAL does
not wait for locks, so create the database once (for example with
`kiw migrate up`) before starting several processes on it.

## Migrations (`pkg/orm/migrate`)

Applies the files `kiw make migration` (or `kiw migrate create`) generates —
`<version>_<name>.sql` with `-- +up` and `-- +down` sections — and records them
in `schema_migrations` (`version`, `name`, `checksum`, `applied_at`).

- `migrate.Up(ctx, db *orm.DB, fsys fs.FS) error` — apply everything pending; the start-up one-liner
- `migrate.New(db, fsys, migrate.Options{Table, LockTimeout}) (*Migrator, error)`
- `Migrator.Up(ctx, n)` (n <= 0: all), `Migrator.Down(ctx, n)` (n >= 1), `Migrator.Redo(ctx)`, `Migrator.Status(ctx)`, `Migrator.Migrations()`
- `migrate.Load(fsys)`, `migrate.Parse(file, src)` — the file format on its own
- Errors: `ErrChecksumMismatch`, `ErrMissingFile`, `ErrIrreversible`, `ErrLocked`, `ErrUnsupportedDialect`

```go
//go:embed *.sql
var Files embed.FS // in package migrations

db, err := sqlite.Open("app.db")
if err != nil {
	return err
}
if err := migrate.Up(ctx, db, migrations.Files); err != nil {
	return err
}
```

Guarantees:

- **Transactional**: each migration and its bookkeeping row commit together; a failing migration leaves nothing behind. `Redo` rolls back and re-applies in one transaction.
- **Checksums**: an applied file that was edited stops `Up` and `Down` (`ErrChecksumMismatch`); put the change in a new migration instead. `Status` reports `pending`, `applied`, `modified`, or `missing`.
- **Locking**: concurrent runners (replicas, `kiw migrate` beside a running app) apply each migration once. SQLite uses `BEGIN IMMEDIATE` with `busy_timeout` raised to `LockTimeout`; PostgreSQL uses a transaction-scoped advisory lock, so a crashed runner never leaves a stale lock.
- **Dialects**: SQLite and PostgreSQL. MySQL is refused because its DDL commits implicitly, so a failed migration could not be rolled back with its row. Statements that cannot run inside a transaction (such as PostgreSQL `CREATE INDEX CONCURRENTLY`) are not supported.
- **Driver-free**: the package links no driver; pass an `*orm.DB` opened with the driver you import.

## Security & Architecture Boundaries

- **Parameterized Queries**: All dynamic criteria pass through dialect-specific placeholders (`$1, $2` for PostgreSQL; `?` for SQLite/MySQL) preventing SQL injection vulnerabilities.
- **Pure-Go Architecture**: No CGO dependencies required. Fully cross-compilable across Linux, macOS, and Windows.
- **Downward Dependency Pyramid**: Depends only on Level 1 (`pkg/errs`, `pkg/model`) and standard library `database/sql`. `pkg/orm/migrate` adds only `pkg/orm`; `pkg/orm/sqlite` is the one package that links a driver.
