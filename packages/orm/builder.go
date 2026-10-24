package orm

import (
	"fmt"
	"strings"
)

type whereCondition struct {
	Column   string
	Operator string
	Value    interface{}
	IsOr     bool
	IsRaw    bool
	RawExpr  string
	RawArgs  []interface{}
}

type joinClause struct {
	Type  string // INNER, LEFT, RIGHT, FULL
	Table string
	On    string
}

type orderClause struct {
	Column    string
	Direction string
}

// QueryBuilder constructs SQL statements safely across dialects.
type QueryBuilder struct {
	dialect    Dialect
	table      string
	columns    []string
	distinct   bool
	joins      []joinClause
	wheres     []whereCondition
	groups     []string
	havings    []string
	havingArgs []interface{}
	orders     []orderClause
	limitVal   int
	offsetVal  int
}

// NewQueryBuilder creates a new QueryBuilder for a given dialect.
func NewQueryBuilder(dialect Dialect) *QueryBuilder {
	if dialect == nil {
		dialect = &SQLiteDialect{}
	}
	return &QueryBuilder{
		dialect: dialect,
	}
}

// Table sets the target table.
func (b *QueryBuilder) Table(table string) *QueryBuilder {
	b.table = table
	return b
}

// Select sets columns to retrieve.
func (b *QueryBuilder) Select(cols ...string) *QueryBuilder {
	b.columns = append(b.columns, cols...)
	return b
}

// Distinct enables SELECT DISTINCT.
func (b *QueryBuilder) Distinct() *QueryBuilder {
	b.distinct = true
	return b
}

// Where adds an AND condition.
func (b *QueryBuilder) Where(col, op string, val interface{}) *QueryBuilder {
	b.wheres = append(b.wheres, whereCondition{
		Column:   col,
		Operator: strings.ToUpper(strings.TrimSpace(op)),
		Value:    val,
		IsOr:     false,
	})
	return b
}

// OrWhere adds an OR condition.
func (b *QueryBuilder) OrWhere(col, op string, val interface{}) *QueryBuilder {
	b.wheres = append(b.wheres, whereCondition{
		Column:   col,
		Operator: strings.ToUpper(strings.TrimSpace(op)),
		Value:    val,
		IsOr:     true,
	})
	return b
}

// WhereIn adds an IN (val1, val2, ...) condition.
func (b *QueryBuilder) WhereIn(col string, vals ...interface{}) *QueryBuilder {
	if len(vals) == 0 {
		return b.WhereRaw("1 = 0")
	}
	placeholders := make([]string, len(vals))
	for i := range vals {
		placeholders[i] = "?"
	}
	raw := fmt.Sprintf("%s IN (%s)", b.quote(col), strings.Join(placeholders, ", "))
	return b.WhereRaw(raw, vals...)
}

// WhereNull adds an IS NULL condition.
func (b *QueryBuilder) WhereNull(col string) *QueryBuilder {
	return b.WhereRaw(fmt.Sprintf("%s IS NULL", b.quote(col)))
}

// WhereNotNull adds an IS NOT NULL condition.
func (b *QueryBuilder) WhereNotNull(col string) *QueryBuilder {
	return b.WhereRaw(fmt.Sprintf("%s IS NOT NULL", b.quote(col)))
}

// WhereRaw adds an arbitrary WHERE condition.
func (b *QueryBuilder) WhereRaw(clause string, args ...interface{}) *QueryBuilder {
	b.wheres = append(b.wheres, whereCondition{
		IsRaw:   true,
		RawExpr: clause,
		RawArgs: args,
		IsOr:    false,
	})
	return b
}

// Join adds a JOIN clause.
func (b *QueryBuilder) Join(joinType, table, on string) *QueryBuilder {
	b.joins = append(b.joins, joinClause{
		Type:  strings.ToUpper(strings.TrimSpace(joinType)),
		Table: table,
		On:    on,
	})
	return b
}

// LeftJoin adds a LEFT JOIN clause.
func (b *QueryBuilder) LeftJoin(table, on string) *QueryBuilder {
	return b.Join("LEFT", table, on)
}

// InnerJoin adds an INNER JOIN clause.
func (b *QueryBuilder) InnerJoin(table, on string) *QueryBuilder {
	return b.Join("INNER", table, on)
}

// GroupBy adds GROUP BY columns.
func (b *QueryBuilder) GroupBy(cols ...string) *QueryBuilder {
	b.groups = append(b.groups, cols...)
	return b
}

// Having adds a HAVING condition.
func (b *QueryBuilder) Having(clause string, args ...interface{}) *QueryBuilder {
	b.havings = append(b.havings, clause)
	b.havingArgs = append(b.havingArgs, args...)
	return b
}

// OrderBy adds an ORDER BY clause.
func (b *QueryBuilder) OrderBy(col, dir string) *QueryBuilder {
	d := strings.ToUpper(strings.TrimSpace(dir))
	if d != "DESC" {
		d = "ASC"
	}
	b.orders = append(b.orders, orderClause{Column: col, Direction: d})
	return b
}

// Limit sets maximum rows to return.
func (b *QueryBuilder) Limit(n int) *QueryBuilder {
	b.limitVal = n
	return b
}

// Offset sets rows to skip.
func (b *QueryBuilder) Offset(n int) *QueryBuilder {
	b.offsetVal = n
	return b
}

// BuildSelect compiles the SELECT query and its bound arguments.
func (b *QueryBuilder) BuildSelect() (string, []interface{}) {
	var sql strings.Builder
	var args []interface{}
	argIndex := 1

	sql.WriteString("SELECT ")
	if b.distinct {
		sql.WriteString("DISTINCT ")
	}

	if len(b.columns) == 0 {
		sql.WriteString("*")
	} else {
		quotedCols := make([]string, len(b.columns))
		for i, col := range b.columns {
			quotedCols[i] = b.quote(col)
		}
		sql.WriteString(strings.Join(quotedCols, ", "))
	}

	sql.WriteString(" FROM ")
	sql.WriteString(b.quote(b.table))

	for _, j := range b.joins {
		sql.WriteString(fmt.Sprintf(" %s JOIN %s ON %s", j.Type, b.quote(j.Table), j.On))
	}

	if len(b.wheres) > 0 {
		sql.WriteString(" WHERE ")
		for i, w := range b.wheres {
			if i > 0 {
				if w.IsOr {
					sql.WriteString(" OR ")
				} else {
					sql.WriteString(" AND ")
				}
			}

			if w.IsRaw {
				expr := w.RawExpr
				// Replace ? with dialect placeholders if necessary
				for range w.RawArgs {
					ph := b.dialect.Placeholder(argIndex)
					expr = strings.Replace(expr, "?", ph, 1)
					argIndex++
				}
				sql.WriteString(expr)
				args = append(args, w.RawArgs...)
			} else {
				ph := b.dialect.Placeholder(argIndex)
				argIndex++
				sql.WriteString(fmt.Sprintf("%s %s %s", b.quote(w.Column), w.Operator, ph))
				args = append(args, w.Value)
			}
		}
	}

	if len(b.groups) > 0 {
		sql.WriteString(" GROUP BY ")
		quoted := make([]string, len(b.groups))
		for i, g := range b.groups {
			quoted[i] = b.quote(g)
		}
		sql.WriteString(strings.Join(quoted, ", "))
	}

	if len(b.havings) > 0 {
		sql.WriteString(" HAVING ")
		sql.WriteString(strings.Join(b.havings, " AND "))
		args = append(args, b.havingArgs...)
	}

	if len(b.orders) > 0 {
		sql.WriteString(" ORDER BY ")
		ordParts := make([]string, len(b.orders))
		for i, o := range b.orders {
			ordParts[i] = fmt.Sprintf("%s %s", b.quote(o.Column), o.Direction)
		}
		sql.WriteString(strings.Join(ordParts, ", "))
	}

	if b.limitVal > 0 {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", b.limitVal))
	}
	if b.offsetVal > 0 {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", b.offsetVal))
	}

	return sql.String(), args
}

// BuildInsert compiles an INSERT INTO statement for the specified columns.
func (b *QueryBuilder) BuildInsert(columns []string) string {
	var sql strings.Builder
	sql.WriteString("INSERT INTO ")
	sql.WriteString(b.quote(b.table))
	sql.WriteString(" (")

	quotedCols := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = b.quote(col)
		placeholders[i] = b.dialect.Placeholder(i + 1)
	}

	sql.WriteString(strings.Join(quotedCols, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	return sql.String()
}

// BuildUpdate compiles an UPDATE statement for the specified columns and conditions.
func (b *QueryBuilder) BuildUpdate(columns []string, values []interface{}) (string, []interface{}) {
	var sql strings.Builder
	var args []interface{}
	argIndex := 1

	sql.WriteString("UPDATE ")
	sql.WriteString(b.quote(b.table))
	sql.WriteString(" SET ")

	setParts := make([]string, len(columns))
	for i, col := range columns {
		setParts[i] = fmt.Sprintf("%s = %s", b.quote(col), b.dialect.Placeholder(argIndex))
		argIndex++
		args = append(args, values[i])
	}
	sql.WriteString(strings.Join(setParts, ", "))

	if len(b.wheres) > 0 {
		sql.WriteString(" WHERE ")
		for i, w := range b.wheres {
			if i > 0 {
				if w.IsOr {
					sql.WriteString(" OR ")
				} else {
					sql.WriteString(" AND ")
				}
			}

			if w.IsRaw {
				expr := w.RawExpr
				for range w.RawArgs {
					ph := b.dialect.Placeholder(argIndex)
					expr = strings.Replace(expr, "?", ph, 1)
					argIndex++
				}
				sql.WriteString(expr)
				args = append(args, w.RawArgs...)
			} else {
				ph := b.dialect.Placeholder(argIndex)
				argIndex++
				sql.WriteString(fmt.Sprintf("%s %s %s", b.quote(w.Column), w.Operator, ph))
				args = append(args, w.Value)
			}
		}
	}

	return sql.String(), args
}

// BuildDelete compiles a DELETE FROM statement and bound arguments.
func (b *QueryBuilder) BuildDelete() (string, []interface{}) {
	var sql strings.Builder
	var args []interface{}
	argIndex := 1

	sql.WriteString("DELETE FROM ")
	sql.WriteString(b.quote(b.table))

	if len(b.wheres) > 0 {
		sql.WriteString(" WHERE ")
		for i, w := range b.wheres {
			if i > 0 {
				if w.IsOr {
					sql.WriteString(" OR ")
				} else {
					sql.WriteString(" AND ")
				}
			}

			if w.IsRaw {
				expr := w.RawExpr
				for range w.RawArgs {
					ph := b.dialect.Placeholder(argIndex)
					expr = strings.Replace(expr, "?", ph, 1)
					argIndex++
				}
				sql.WriteString(expr)
				args = append(args, w.RawArgs...)
			} else {
				ph := b.dialect.Placeholder(argIndex)
				argIndex++
				sql.WriteString(fmt.Sprintf("%s %s %s", b.quote(w.Column), w.Operator, ph))
				args = append(args, w.Value)
			}
		}
	}

	return sql.String(), args
}

func (b *QueryBuilder) quote(ident string) string {
	ident = strings.TrimSpace(ident)
	if ident == "*" || strings.Contains(ident, "(") || strings.Contains(ident, " ") {
		return ident
	}
	if strings.Contains(ident, ".") {
		parts := strings.Split(ident, ".")
		quoted := make([]string, len(parts))
		for i, p := range parts {
			quoted[i] = b.dialect.QuoteIdentifier(p)
		}
		return strings.Join(quoted, ".")
	}
	return b.dialect.QuoteIdentifier(ident)
}
