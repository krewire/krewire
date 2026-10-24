package orm

import (
	"context"
	"fmt"
	"strings"

	"github.com/krewire/krewire/packages/errs"
)

// GenerateCreateTableSQL generates the CREATE TABLE DDL statement for a model.
func GenerateCreateTableSQL(model interface{}, dialect Dialect) (string, error) {
	tm, err := ParseModel(model)
	if err != nil {
		return "", err
	}
	if dialect == nil {
		dialect = &SQLiteDialect{}
	}

	var colsDef []string

	for _, col := range tm.Columns {
		var def strings.Builder
		def.WriteString(dialect.QuoteIdentifier(col.Name))
		def.WriteString(" ")

		if col.IsPrimaryKey && col.IsAutoInc {
			def.WriteString(dialect.AutoIncrementPrimaryKeyClause())
		} else {
			def.WriteString(dialect.DataTypeOf(col.Type))

			if col.IsPrimaryKey {
				def.WriteString(" PRIMARY KEY")
			} else if !col.IsNullable {
				def.WriteString(" NOT NULL")
			}

			if col.Unique {
				def.WriteString(" UNIQUE")
			}

			if col.DefaultValue != "" {
				def.WriteString(fmt.Sprintf(" DEFAULT %s", col.DefaultValue))
			}
		}

		colsDef = append(colsDef, def.String())
	}

	sql := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n  %s\n);",
		dialect.QuoteIdentifier(tm.TableName),
		strings.Join(colsDef, ",\n  "),
	)

	return sql, nil
}

// AutoMigrate creates database tables for each provided model struct if they do not exist.
func (db *DB) AutoMigrate(ctx context.Context, models ...interface{}) error {
	for _, m := range models {
		ddl, err := GenerateCreateTableSQL(m, db.dialect)
		if err != nil {
			return err
		}

		if _, err := db.Exec(ctx, ddl); err != nil {
			return errs.FailureError(fmt.Sprintf("auto-migrate failed for model %T: %v", m, err))
		}
	}
	return nil
}
