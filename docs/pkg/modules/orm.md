# `orm`

Import: `github.com/krewire/krewire/pkg/orm`

## Purpose

A lightweight, pure-Go Object-Relational Mapping (ORM) and SQL Query Builder library with unified multi-driver support. Designed for SQLite, PostgreSQL, and MySQL, `packages/orm` eliminates external CGO dependencies and abstracts dialect differences (quoting, parameter placeholders `$1` vs `?`, data types, DDL statements, and auto-increment conventions) behind a single fluent API.

## Main API

- **Adapter Abstraction**:
  - `Adapter` interface (`Name() string`, `Type() AdapterType`, `Ping(ctx context.Context) error`, `Close() error`)
  - `AdapterType`: `AdapterTypeSQL`, `AdapterTypeDocument`, `AdapterTypeSearch`
- **Supported Drivers & Adapters**:
  - **SQLite**: `NewSQLite(cfg SQLiteConfig) (*DB, error)`, `cfg.FormatDSN()`
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
  - `DB.Begin(ctx context.Context) (*Tx, error)`
  - `DB.Transaction(ctx context.Context, fn func(tx *Tx) error) error`
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
  - `DB.AutoMigrate(ctx context.Context, models ...interface{}) error`

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

## Security & Architecture Boundaries

- **Parameterized Queries**: All dynamic criteria pass through dialect-specific placeholders (`$1, $2` for PostgreSQL; `?` for SQLite/MySQL) preventing SQL injection vulnerabilities.
- **Pure-Go Architecture**: No CGO dependencies required. Fully cross-compilable across Linux, macOS, and Windows.
- **Downward Dependency Pyramid**: Depends only on Level 1 (`packages/errs`, `packages/model`) and standard library `database/sql`.
