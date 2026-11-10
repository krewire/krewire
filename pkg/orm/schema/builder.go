package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
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

// ----------------------------------------------------------------------------
// SQL Compilation
// ----------------------------------------------------------------------------

// CompileCreate generates the CREATE TABLE DDL and any secondary index DDLs.
func (b *Builder) CompileCreate(bp *Blueprint) (string, []string, error) {
	if len(bp.Columns) == 0 {
		return "", nil, errs.UsageError(fmt.Sprintf("schema: table %q must declare at least one column", bp.TableName))
	}

	dName := strings.ToLower(b.dialect.Name())
	var defs []string
	var indexSQLs []string

	for _, col := range bp.Columns {
		colSQL := b.compileColumn(col, bp.TableName)
		defs = append(defs, colSQL)

		// Check inline column index
		if col.HasIndex {
			idxName := col.IndexName
			if idxName == "" {
				idxName = fmt.Sprintf("%s_%s_index", bp.TableName, col.Name)
			}
			indexSQLs = append(indexSQLs, fmt.Sprintf(
				"CREATE INDEX %s ON %s (%s);",
				b.dialect.QuoteIdentifier(idxName),
				b.dialect.QuoteIdentifier(bp.TableName),
				b.dialect.QuoteIdentifier(col.Name),
			))
		}
	}

	// Multi-column Primary Keys
	if len(bp.PrimaryKeys) > 0 {
		var pkCols []string
		for _, pk := range bp.PrimaryKeys {
			pkCols = append(pkCols, b.dialect.QuoteIdentifier(pk))
		}
		defs = append(defs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	// Foreign Keys
	for _, fk := range bp.ForeignKeys {
		fkDef := fmt.Sprintf(
			"FOREIGN KEY (%s) REFERENCES %s (%s)",
			b.dialect.QuoteIdentifier(fk.Column),
			b.dialect.QuoteIdentifier(fk.TargetTable),
			b.dialect.QuoteIdentifier(fk.TargetColumn),
		)
		if fk.OnDeleteAction != "" {
			fkDef += " ON DELETE " + fk.OnDeleteAction
		}
		if fk.OnUpdateAction != "" {
			fkDef += " ON UPDATE " + fk.OnUpdateAction
		}
		defs = append(defs, fkDef)
	}

	createSQL := fmt.Sprintf(
		"CREATE TABLE %s (\n  %s\n);",
		b.dialect.QuoteIdentifier(bp.TableName),
		strings.Join(defs, ",\n  "),
	)

	// Table-level Indexes
	for _, idx := range bp.Indexes {
		var cols []string
		for _, c := range idx.Columns {
			cols = append(cols, b.dialect.QuoteIdentifier(c))
		}
		uniqueStr := ""
		if idx.IsUnique {
			uniqueStr = "UNIQUE "
		}
		idxName := idx.Name
		if idxName == "" {
			idxName = fmt.Sprintf("%s_%s_idx", bp.TableName, strings.Join(idx.Columns, "_"))
		}
		indexSQLs = append(indexSQLs, fmt.Sprintf(
			"CREATE %sINDEX %s ON %s (%s);",
			uniqueStr,
			b.dialect.QuoteIdentifier(idxName),
			b.dialect.QuoteIdentifier(bp.TableName),
			strings.Join(cols, ", "),
		))
	}

	_ = dName
	return createSQL, indexSQLs, nil
}

// CompileAlter generates ALTER TABLE statements for blueprint mutations.
func (b *Builder) CompileAlter(bp *Blueprint) ([]string, error) {
	var stmts []string

	// 1. Add Columns
	for _, col := range bp.Columns {
		colDef := b.compileColumn(col, bp.TableName)
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s;",
			b.dialect.QuoteIdentifier(bp.TableName),
			colDef,
		))
	}

	// 2. Drop Columns
	for _, colName := range bp.DropColumns {
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s DROP COLUMN %s;",
			b.dialect.QuoteIdentifier(bp.TableName),
			b.dialect.QuoteIdentifier(colName),
		))
	}

	// 3. Rename Columns
	for from, to := range bp.RenameColumns {
		stmts = append(stmts, fmt.Sprintf(
			"ALTER TABLE %s RENAME COLUMN %s TO %s;",
			b.dialect.QuoteIdentifier(bp.TableName),
			b.dialect.QuoteIdentifier(from),
			b.dialect.QuoteIdentifier(to),
		))
	}

	// 4. Create Indexes
	for _, idx := range bp.Indexes {
		var cols []string
		for _, c := range idx.Columns {
			cols = append(cols, b.dialect.QuoteIdentifier(c))
		}
		uniqueStr := ""
		if idx.IsUnique {
			uniqueStr = "UNIQUE "
		}
		stmts = append(stmts, fmt.Sprintf(
			"CREATE %sINDEX %s ON %s (%s);",
			uniqueStr,
			b.dialect.QuoteIdentifier(idx.Name),
			b.dialect.QuoteIdentifier(bp.TableName),
			strings.Join(cols, ", "),
		))
	}

	// 5. Drop Indexes
	for _, idxName := range bp.DropIndexes {
		stmts = append(stmts, fmt.Sprintf(
			"DROP INDEX %s;",
			b.dialect.QuoteIdentifier(idxName),
		))
	}

	return stmts, nil
}

func (b *Builder) compileColumn(col *ColumnDefinition, tableName string) string {
	dName := strings.ToLower(b.dialect.Name())
	quoted := b.dialect.QuoteIdentifier(col.Name)

	var sb strings.Builder
	sb.WriteString(quoted)
	sb.WriteString(" ")

	// Handle Auto-Increment Primary Key
	if col.IsPrimaryKey && col.IsAutoInc {
		sb.WriteString(b.dialect.AutoIncrementPrimaryKeyClause())
		return sb.String()
	}

	// Data Type
	typeStr := b.mapColumnType(col.Type, col.Length, col.Precision, col.Scale, dName)
	sb.WriteString(typeStr)

	if col.IsUnsigned && strings.Contains(dName, "mysql") {
		sb.WriteString(" UNSIGNED")
	}

	if col.IsPrimaryKey {
		sb.WriteString(" PRIMARY KEY")
	} else if !col.IsNullable {
		sb.WriteString(" NOT NULL")
	}

	if col.IsUnique {
		sb.WriteString(" UNIQUE")
	}

	if col.DefaultVal != nil {
		sb.WriteString(" DEFAULT ")
		sb.WriteString(*col.DefaultVal)
	}

	return sb.String()
}

func (b *Builder) mapColumnType(t ColumnType, length, precision, scale int, dialect string) string {
	isSQLite := strings.Contains(dialect, "sqlite")
	isMySQL := strings.Contains(dialect, "mysql")
	isPG := strings.Contains(dialect, "postgres") || strings.Contains(dialect, "pgx")

	switch t {
	case TypeID, TypeBigID:
		if isSQLite {
			return "INTEGER"
		} else if isMySQL {
			return "BIGINT"
		}
		return "BIGINT"
	case TypeUUID:
		if isPG {
			return "UUID"
		} else if isMySQL {
			return "CHAR(36)"
		}
		return "TEXT"
	case TypeString:
		l := length
		if l <= 0 {
			l = 255
		}
		if isSQLite {
			return "TEXT"
		}
		return fmt.Sprintf("VARCHAR(%d)", l)
	case TypeText:
		return "TEXT"
	case TypeMediumText:
		if isMySQL {
			return "MEDIUMTEXT"
		}
		return "TEXT"
	case TypeLongText:
		if isMySQL {
			return "LONGTEXT"
		}
		return "TEXT"
	case TypeInteger:
		if isMySQL {
			return "INT"
		}
		return "INTEGER"
	case TypeTinyInt:
		if isMySQL {
			return "TINYINT"
		} else if isPG {
			return "SMALLINT"
		}
		return "INTEGER"
	case TypeSmallInt:
		return "SMALLINT"
	case TypeBigInt:
		if isSQLite {
			return "INTEGER"
		}
		return "BIGINT"
	case TypeFloat:
		if isSQLite || isPG {
			return "REAL"
		}
		return "FLOAT"
	case TypeDouble:
		if isSQLite {
			return "REAL"
		} else if isPG {
			return "DOUBLE PRECISION"
		}
		return "DOUBLE"
	case TypeDecimal:
		p, s := precision, scale
		if p <= 0 {
			p = 10
		}
		if s <= 0 {
			s = 2
		}
		if isSQLite {
			return "REAL"
		}
		return fmt.Sprintf("DECIMAL(%d, %d)", p, s)
	case TypeBoolean:
		if isPG {
			return "BOOLEAN"
		} else if isMySQL {
			return "TINYINT(1)"
		}
		return "INTEGER"
	case TypeDate:
		if isSQLite {
			return "TEXT"
		}
		return "DATE"
	case TypeDateTime:
		if isSQLite {
			return "DATETIME"
		} else if isPG {
			return "TIMESTAMP"
		}
		return "DATETIME(6)"
	case TypeTimestamp:
		if isSQLite {
			return "DATETIME"
		} else if isPG {
			return "TIMESTAMPTZ"
		}
		return "TIMESTAMP"
	case TypeJSON:
		if isPG {
			return "JSONB"
		} else if isMySQL {
			return "JSON"
		}
		return "TEXT"
	case TypeBinary:
		if isPG {
			return "BYTEA"
		} else if isMySQL {
			return "LONGBLOB"
		}
		return "BLOB"
	default:
		return "TEXT"
	}
}
