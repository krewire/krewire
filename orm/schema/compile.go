package schema

import (
	"fmt"
	"strings"

	"github.com/krewire/krewire/errs"
)

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
