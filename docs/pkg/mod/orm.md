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
- **Database Connection Pool (Pillar S2: Sustainable)**:
  - `Open(driverName, dsn string) (*DB, error)` — Opens with bounded defaults (`MaxOpenConns=25`, `MaxIdleConns=10`, `ConnMaxLifetime=1h`, `ConnMaxIdleTime=15m`).
  - `OpenWithConfig(cfg Config) (*DB, error)` — Configurable connection pool parameters.
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
  - `Query.WhereNull(col string) *Query`, `Query.WhereNotNull(col string) *Query`
  - `Query.WhereRaw(clause string, args ...interface{}) *Query`
  - `Query.Join(joinType, table, on string) *Query`, `Query.LeftJoin`, `Query.InnerJoin`
  - `Query.GroupBy(cols ...string) *Query`, `Query.Having(clause string, args ...interface{}) *Query`
  - `Query.OrderBy(col, dir string) *Query`
  - `Query.Limit(n int) *Query`, `Query.Offset(n int) *Query`
  - `Query.Select(cols ...string) *Query`, `Query.Distinct() *Query`
  - `Query.Scope(fn func(q *Query) *Query) *Query`
  - `Query.WithTrashed() *Query`, `Query.OnlyTrashed() *Query`
  - `Query.Insert(record interface{}) (int64, error)`
  - `Query.Find(dest interface{}, id ...interface{}) error`
  - `Query.FindOrFail(dest interface{}, id interface{}) error`
  - `Query.First(dest interface{}) error`, `Query.FirstOrFail(dest interface{}) error`
  - `Query.FindAll(dest interface{}) error`
  - `Query.Pluck(column string, dest interface{}) error`
  - `Query.Chunk(size int, destSlice interface{}, fn func() error) error`
  - `Query.Exists() (bool, error)`, `Query.DoesntExist() (bool, error)`
  - `Query.Update(record interface{}) (int64, error)`
  - `Query.Delete(record interface{}, id ...interface{}) (int64, error)`
  - `Query.ForceDelete(record interface{}, id ...interface{}) (int64, error)`
  - `Query.Restore(record interface{}, id ...interface{}) (int64, error)`
  - `Query.Count() (int64, error)`
- **Schema & Migrations**:
  - `ParseModel(v interface{}) (*TableModel, error)`
  - `GenerateCreateTableSQL(model interface{}, dialect Dialect) (string, error)`
  - `schema.Blueprint`: Programmatic schema builder (`Increments`, `String`, `Text`, `Integer`, `Float`, `Boolean`, `Timestamps`, `SoftDeletes`, `Morphs`)
  - `DB.AutoMigrate(ctx context.Context, models ...interface{}) error`
  - Versioned SQL & Go migrations: [`pkg/orm/migrate`](#migrations-pkgormmigrate)
- **Model Lifecycle & Traits**:
  - `orm.BaseModel`, `orm.Timestamps`, `orm.SoftDeletes`
  - Hook interfaces: `BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, `AfterDelete`, `BeforeSave`, `AfterSave`
- **Faker & Seeder / Factory Engine**:
  - Pure-Go Faker: [`pkg/orm/faker`](#faker-engine-pkgormfaker)
  - Seeders & Model Factories: [`pkg/orm/seeder`](#seeder--model-factory-engine-pkgormseeder)
- **Generic Repository Pattern**:
  - Type-safe generic DAO: [`pkg/orm/repository`](#generic-repository-pattern-pkgormrepository)

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

## Models, Timestamps & Soft Deletes

Krewire models provide built-in timestamps, soft deletes, and lifecycle hooks via embedded structs:

```go
type Post struct {
    orm.BaseModel // Embeds ID (int64), Timestamps (CreatedAt, UpdatedAt), and SoftDeletes (DeletedAt)

    Title   string `orm:"column:title"`
    Slug    string `orm:"column:slug;unique"`
    Content string `orm:"column:content"`
}

func (Post) TableName() string {
    return "posts"
}

// Lifecycle Hooks: BeforeCreate, AfterCreate, BeforeUpdate, AfterUpdate, BeforeDelete, AfterDelete, BeforeSave, AfterSave
func (p *Post) BeforeCreate(ctx context.Context) error {
    if p.Slug == "" {
        p.Slug = slugify(p.Title)
    }
    return nil
}
```

Queries automatically filter out soft-deleted records (`WHERE deleted_at IS NULL`). Use `.WithTrashed()` to include soft-deleted records or `.OnlyTrashed()` to view only soft-deleted records:

```go
// Normal query excludes deleted posts
db.Model(&Post{}).FindAll(&posts)

// Include soft-deleted
db.Model(&Post{}).WithTrashed().FindAll(&allPosts)

// Restore a soft-deleted record
db.Model(&Post{}).Restore(&post)

// Permanently delete from table
db.Model(&Post{}).ForceDelete(&post)
```

## Programmatic Migrations (`schema.Blueprint`)

In addition to raw SQL migrations, Krewire supports Laravel-like programmatic schema blueprints in Go:

```go
package migrations

import (
    "context"

    "github.com/krewire/krewire/pkg/orm/migrate"
    "github.com/krewire/krewire/pkg/orm/schema"
)

func init() {
    migrate.RegisterGo(migrate.GoMigration{
        Version: "20261111000000",
        Name:    "create_articles_table",
        Up: func(ctx context.Context, s *schema.Builder) error {
            return s.Create("articles", func(t *schema.Blueprint) {
                t.BigIncrements("id")
                t.String("title", 255).NotNull()
                t.String("slug", 255).Unique()
                t.Text("content")
                t.Morphs("taggable") // taggable_id (int64) + taggable_type (varchar 255)
                t.Timestamps()
                t.SoftDeletes()
            })
        },
        Down: func(ctx context.Context, s *schema.Builder) error {
            return s.DropTableIfExists("articles")
        },
    })
}
```

## Faker Engine (`pkg/orm/faker`)

Pure-Go, thread-safe, seedable fake data generator with rich, realistic datasets:

```go
import "github.com/krewire/krewire/pkg/orm/faker"

f := faker.New() // or faker.New(12345) for deterministic seed

name    := f.Name()
email   := f.SafeEmail()
phone   := f.Phone()
address := f.Address()
company := f.Company()
uuid    := f.UUID()
price   := f.Price(10.0, 500.0)
text    := f.Paragraph(3)
past    := f.PastDate(30)
```

## Seeder & Model Factory Engine (`pkg/orm/seeder`)

Declarative, Laravel-style model factories and seeders for database population and automated tests:

### Model Factories

```go
import (
    "github.com/krewire/krewire/pkg/orm"
    "github.com/krewire/krewire/pkg/orm/faker"
    "github.com/krewire/krewire/pkg/orm/seeder"
)

func UserFactory(db *orm.DB) *seeder.Factory[User] {
    return seeder.NewFactory(db, func(f *faker.Faker) *User {
        return &User{
            Name:  f.Name(),
            Email: f.SafeEmail(),
        }
    })
}

// Generate models without saving:
user := UserFactory(db).Make(ctx)
users := UserFactory(db).Count(10).MakeMany(ctx)

// Generate and persist directly to database:
savedUser, err := UserFactory(db).Create(ctx)
savedUsers, err := UserFactory(db).Count(50).CreateMany(ctx)

// Apply state transformations:
adminUsers, err := UserFactory(db).State(func(u *User) {
    u.Role = "admin"
}).Count(5).CreateMany(ctx)
```

### Seeders

```go
package seeds

import (
    "context"

    "github.com/krewire/krewire/pkg/orm"
    "github.com/krewire/krewire/pkg/orm/seeder"
)

func init() {
    seeder.Register("DatabaseSeeder", DatabaseSeeder{})
}

type DatabaseSeeder struct{}

func (DatabaseSeeder) Run(ctx context.Context, db *orm.DB) error {
    // Call other seeders or factories
    _, err := UserFactory(db).Count(20).CreateMany(ctx)
    return err
}
```

Execute seeders via the CLI:
```bash
kiw seed                       # runs DatabaseSeeder
kiw seed --class UserSeeder    # runs specific seeder
kiw db:seed                    # Laravel-like alias
```

## Generic Repository Pattern (`pkg/orm/repository`)

Type-safe Data Access Object (DAO) providing high cohesion, separation of concerns, and full database transaction safety:

```go
import (
    "context"

    "github.com/krewire/krewire/pkg/orm"
    "github.com/krewire/krewire/pkg/orm/repository"
)

type UserRepository interface {
    repository.Repository[User]
}

type userRepositoryImpl struct {
    *repository.BaseRepository[User]
}

func NewUserRepository(db *orm.DB) UserRepository {
    return &userRepositoryImpl{
        BaseRepository: repository.New[User](db),
    }
}
```

### Repository Operations

```go
repo := NewUserRepository(db)
ctx := context.Background()

// CRUD
user, err := repo.FindByID(ctx, 42)
users, err := repo.FindAll(ctx)
err = repo.Create(ctx, &User{Name: "Alice", Email: "alice@example.com"})
err = repo.Update(ctx, user)
err = repo.Save(ctx, user) // Inserts if ID==0, updates if ID>0
err = repo.Delete(ctx, 42)

// Fluent criteria & counts
activeUsers, err := repo.FindWhere(ctx, func(q *orm.Query) *orm.Query {
    return q.Where("is_active", "=", 1).OrderBy("name", "ASC")
})
count, err := repo.Count(ctx)
exists, err := repo.Exists(ctx, func(q *orm.Query) *orm.Query {
    return q.Where("email", "=", "alice@example.com")
})

// Pagination
pager, err := repo.Paginate(ctx, 1, 15)
// pager.Items, pager.Total, pager.Page, pager.PerPage, pager.TotalPages, pager.HasNext, pager.HasPrev

// Safe atomic transactions
err = repo.Transaction(ctx, func(txRepo repository.Repository[User]) error {
    if err := txRepo.Create(ctx, u1); err != nil {
        return err // automatically rolled back
    }
    return txRepo.Create(ctx, u2) // committed on return nil
})
```

## Security & Architecture Boundaries

- **Connection Pool Bounding (Pillar S2: Sustainable)**:
  - Enforces bounded connection pools to prevent file descriptor leaks and memory exhaustion:
    - `MaxOpenConns`: 25 (prevents database connection floods)
    - `MaxIdleConns`: 10 (retains ready connections without excessive idle overhead)
    - `ConnMaxLifetime`: 1 hour (recycles aged connections cleanly)
    - `ConnMaxIdleTime`: 15 minutes (prunes unused idle connections)
    - `BusyTimeout`: 5,000 ms (5 seconds wait for locks in SQLite WAL mode)
  - Custom limits can be specified via `orm.Config{MaxOpenConns, MaxIdleConns, ConnMaxLifetime, ConnMaxIdleTime}` passed to `orm.OpenWithConfig`.
- **Parameterized Queries**: All dynamic criteria pass through dialect-specific placeholders (`$1, $2` for PostgreSQL; `?` for SQLite/MySQL) preventing SQL injection vulnerabilities.
- **Pure-Go Architecture**: No CGO dependencies required. Fully cross-compilable across Linux, macOS, and Windows.
- **Downward Dependency Pyramid**: Depends only on Level 1 (`pkg/errs`, `pkg/model`) and standard library `database/sql`. `pkg/orm/migrate` adds only `pkg/orm`; `pkg/orm/sqlite` is the one package that links a driver.
