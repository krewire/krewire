package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/krewire/krewire/errs"
	"github.com/krewire/krewire/orm"
)

// SQLExecutor executes queries during schema operations.
// Satisfied by *sql.DB, *sql.Tx, *sql.Conn, and migration runners.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Builder provides a fluent, programmatic schema definition and migration engine
// identical to Laravel's Schema Facade.
type Builder struct {
	ctx      context.Context
	executor SQLExecutor
	db       *orm.DB
	tx       *orm.Tx
	dialect  orm.Dialect
}

// NewBuilder creates a new schema builder bound to a context and database connection.
func NewBuilder(ctx context.Context, db *orm.DB) *Builder {
	var dialect orm.Dialect = &orm.SQLiteDialect{}
	var executor SQLExecutor
	if db != nil {
		executor = db.SQLDB()
		if db.Dialect() != nil {
			dialect = db.Dialect()
		}
	}
	return &Builder{
		ctx:      ctx,
		executor: executor,
		db:       db,
		dialect:  dialect,
	}
}

// NewTxBuilder creates a schema builder bound to an active transaction.
func NewTxBuilder(ctx context.Context, tx *orm.Tx, dialect orm.Dialect) *Builder {
	if dialect == nil {
		dialect = &orm.SQLiteDialect{}
	}
	var executor SQLExecutor
	if tx != nil {
		executor = tx.SQLTx()
	}
	return &Builder{
		ctx:      ctx,
		executor: executor,
		tx:       tx,
		dialect:  dialect,
	}
}

// NewExecutorBuilder creates a schema builder bound to an arbitrary SQLExecutor.
func NewExecutorBuilder(ctx context.Context, executor SQLExecutor, dialect orm.Dialect) *Builder {
	if dialect == nil {
		dialect = &orm.SQLiteDialect{}
	}
	return &Builder{
		ctx:      ctx,
		executor: executor,
		dialect:  dialect,
	}
}

// Dialect returns the active database dialect.
func (b *Builder) Dialect() orm.Dialect {
	return b.dialect
}

// Create defines and executes a new table creation schema.
func (b *Builder) Create(tableName string, fn func(table *Blueprint)) error {
	bp := NewBlueprint(tableName, true)
	fn(bp)

	createSQL, indexSQLs, err := b.CompileCreate(bp)
	if err != nil {
		return err
	}

	if err := b.exec(createSQL); err != nil {
		return errs.FailureError(fmt.Sprintf("schema: create table %q failed: %v", tableName, err))
	}

	for _, idx := range indexSQLs {
		if err := b.exec(idx); err != nil {
			return errs.FailureError(fmt.Sprintf("schema: create index failed: %v", err))
		}
	}
	return nil
}

// Table alters an existing table structure (adding/dropping columns or indexes).
func (b *Builder) Table(tableName string, fn func(table *Blueprint)) error {
	bp := NewBlueprint(tableName, false)
	fn(bp)

	sqls, err := b.CompileAlter(bp)
	if err != nil {
		return err
	}

	for _, stmt := range sqls {
		if err := b.exec(stmt); err != nil {
			return errs.FailureError(fmt.Sprintf("schema: alter table %q failed: %v", tableName, err))
		}
	}
	return nil
}

// Drop drops a table from the database.
func (b *Builder) Drop(tableName string) error {
	q := fmt.Sprintf("DROP TABLE %s", b.dialect.QuoteIdentifier(tableName))
	if err := b.exec(q); err != nil {
		return errs.FailureError(fmt.Sprintf("schema: drop table %q failed: %v", tableName, err))
	}
	return nil
}

// DropIfExists drops a table if it currently exists.
func (b *Builder) DropIfExists(tableName string) error {
	q := fmt.Sprintf("DROP TABLE IF EXISTS %s", b.dialect.QuoteIdentifier(tableName))
	if err := b.exec(q); err != nil {
		return errs.FailureError(fmt.Sprintf("schema: drop table if exists %q failed: %v", tableName, err))
	}
	return nil
}

// Rename renames a table.
func (b *Builder) Rename(from, to string) error {
	q := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", b.dialect.QuoteIdentifier(from), b.dialect.QuoteIdentifier(to))
	if err := b.exec(q); err != nil {
		return errs.FailureError(fmt.Sprintf("schema: rename table from %q to %q failed: %v", from, to, err))
	}
	return nil
}

// HasTable checks if a table exists in the current database.
func (b *Builder) HasTable(tableName string) (bool, error) {
	dName := strings.ToLower(b.dialect.Name())
	var query string

	switch {
	case strings.Contains(dName, "sqlite"):
		query = "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)"
	case strings.Contains(dName, "mysql"):
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_NAME=?)"
	case strings.Contains(dName, "postgres") || strings.Contains(dName, "pgx"):
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1)"
	default:
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=?)"
	}

	var exists bool
	var err error
	if b.executor != nil {
		err = b.executor.QueryRowContext(b.ctx, query, tableName).Scan(&exists)
	} else if b.tx != nil {
		err = b.tx.QueryRow(b.ctx, query, tableName).Scan(&exists)
	} else if b.db != nil {
		err = b.db.QueryRow(b.ctx, query, tableName).Scan(&exists)
	} else {
		return false, errs.UsageError("schema: no db connection available")
	}

	if err != nil {
		return false, errs.FailureError(fmt.Sprintf("schema: has_table check failed: %v", err))
	}
	return exists, nil
}

// HasColumn checks if a column exists in a given table.
func (b *Builder) HasColumn(tableName, columnName string) (bool, error) {
	dName := strings.ToLower(b.dialect.Name())
	var query string

	switch {
	case strings.Contains(dName, "sqlite"):
		// In SQLite, PRAGMA table_info lists columns
		var rows *sql.Rows
		var err error
		infoSQL := fmt.Sprintf("PRAGMA table_info(%s)", b.dialect.QuoteIdentifier(tableName))
		if b.executor != nil {
			rows, err = b.executor.QueryContext(b.ctx, infoSQL)
		} else if b.tx != nil {
			rows, err = b.tx.Query(b.ctx, infoSQL)
		} else if b.db != nil {
			rows, err = b.db.Query(b.ctx, infoSQL)
		}
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt *string
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				return false, err
			}
			if strings.EqualFold(name, columnName) {
				return true, nil
			}
		}
		return false, nil
	case strings.Contains(dName, "mysql"):
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_NAME=? AND COLUMN_NAME=?)"
	case strings.Contains(dName, "postgres") || strings.Contains(dName, "pgx"):
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name=$1 AND column_name=$2)"
	default:
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name=? AND column_name=?)"
	}

	var exists bool
	var err error
	if b.executor != nil {
		err = b.executor.QueryRowContext(b.ctx, query, tableName, columnName).Scan(&exists)
	} else if b.tx != nil {
		err = b.tx.QueryRow(b.ctx, query, tableName, columnName).Scan(&exists)
	} else if b.db != nil {
		err = b.db.QueryRow(b.ctx, query, tableName, columnName).Scan(&exists)
	} else {
		return false, errs.UsageError("schema: no db connection available")
	}

	if err != nil {
		return false, errs.FailureError(fmt.Sprintf("schema: has_column check failed: %v", err))
	}
	return exists, nil
}

// Raw executes arbitrary raw SQL.
func (b *Builder) Raw(query string, args ...interface{}) error {
	return b.exec(query, args...)
}

func (b *Builder) exec(query string, args ...interface{}) error {
	if b.executor != nil {
		var anyArgs []any
		for _, a := range args {
			anyArgs = append(anyArgs, a)
		}
		_, err := b.executor.ExecContext(b.ctx, query, anyArgs...)
		return err
	}
	if b.tx != nil {
		_, err := b.tx.Exec(b.ctx, query, args...)
		return err
	}
	if b.db != nil {
		_, err := b.db.Exec(b.ctx, query, args...)
		return err
	}
	return errs.UsageError("schema: no database or transaction connection present")
}
