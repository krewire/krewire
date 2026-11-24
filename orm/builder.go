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
	err        error
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
	if table != "" && !isValidIdentifier(table) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid table name %q", table)
		}
	}
	return b
}

// Select sets columns to retrieve.
func (b *QueryBuilder) Select(cols ...string) *QueryBuilder {
	for _, col := range cols {
		if !isValidIdentifier(col) {
			if b.err == nil {
				b.err = fmt.Errorf("orm: invalid column name %q", col)
			}
		}
	}
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
	normOp := strings.ToUpper(strings.TrimSpace(op))
	if !allowedOperators[normOp] {
		if b.err == nil {
			b.err = fmt.Errorf("orm: unsupported operator %q", op)
		}
		return b
	}
	if !isValidIdentifier(col) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid column name %q", col)
		}
		return b
	}
	b.wheres = append(b.wheres, whereCondition{
		Column:   col,
		Operator: normOp,
		Value:    val,
		IsOr:     false,
	})
	return b
}

// OrWhere adds an OR condition.
func (b *QueryBuilder) OrWhere(col, op string, val interface{}) *QueryBuilder {
	normOp := strings.ToUpper(strings.TrimSpace(op))
	if !allowedOperators[normOp] {
		if b.err == nil {
			b.err = fmt.Errorf("orm: unsupported operator %q", op)
		}
		return b
	}
	if !isValidIdentifier(col) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid column name %q", col)
		}
		return b
	}
	b.wheres = append(b.wheres, whereCondition{
		Column:   col,
		Operator: normOp,
		Value:    val,
		IsOr:     true,
	})
	return b
}

// WhereIn adds an IN (val1, val2, ...) condition.
func (b *QueryBuilder) WhereIn(col string, vals ...interface{}) *QueryBuilder {
	if !isValidIdentifier(col) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid column name %q", col)
		}
		return b
	}
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
	if !isValidIdentifier(col) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid column name %q", col)
		}
		return b
	}
	return b.WhereRaw(fmt.Sprintf("%s IS NULL", b.quote(col)))
}

// WhereNotNull adds an IS NOT NULL condition.
func (b *QueryBuilder) WhereNotNull(col string) *QueryBuilder {
	if !isValidIdentifier(col) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid column name %q", col)
		}
		return b
	}
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
	if !isValidIdentifier(table) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid join table name %q", table)
		}
	}
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
	for _, col := range cols {
		if !isValidIdentifier(col) {
			if b.err == nil {
				b.err = fmt.Errorf("orm: invalid column name %q", col)
			}
		}
	}
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
	if !isValidIdentifier(col) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid column name %q", col)
		}
		return b
	}
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

// Error returns the first error encountered during query building or compilation.
func (b *QueryBuilder) Error() error {
	return b.err
}
