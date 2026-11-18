package orm

import (
	"fmt"
	"strings"
)

// Compile compiles the SELECT query and reports any compilation error.
func (b *QueryBuilder) Compile() (string, []interface{}, error) {
	sql, args := b.BuildSelect()
	if b.err != nil {
		return "", nil, b.err
	}
	return sql, args, nil
}

// BuildSelect compiles the SELECT query and its bound arguments.
func (b *QueryBuilder) BuildSelect() (string, []interface{}) {
	if b.err != nil {
		return "", nil
	}
	if b.table == "" {
		if b.err == nil {
			b.err = fmt.Errorf("orm: missing table name")
		}
		return "", nil
	}

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
		wSQL, wArgs := b.compileWheres(&argIndex)
		sql.WriteString(wSQL)
		args = append(args, wArgs...)
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

	if b.err != nil {
		return "", nil
	}
	return sql.String(), args
}

// BuildInsert compiles an INSERT INTO statement for the specified columns.
func (b *QueryBuilder) BuildInsert(columns []string) string {
	if b.err != nil {
		return ""
	}
	if b.table == "" {
		if b.err == nil {
			b.err = fmt.Errorf("orm: missing table name")
		}
		return ""
	}

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

	if b.err != nil {
		return ""
	}
	return sql.String()
}

// BuildUpdate compiles an UPDATE statement for the specified columns and conditions.
func (b *QueryBuilder) BuildUpdate(columns []string, values []interface{}) (string, []interface{}) {
	if b.err != nil {
		return "", nil
	}
	if b.table == "" {
		if b.err == nil {
			b.err = fmt.Errorf("orm: missing table name")
		}
		return "", nil
	}

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
		wSQL, wArgs := b.compileWheres(&argIndex)
		sql.WriteString(wSQL)
		args = append(args, wArgs...)
	}

	if b.err != nil {
		return "", nil
	}
	return sql.String(), args
}

// BuildDelete compiles a DELETE FROM statement and bound arguments.
func (b *QueryBuilder) BuildDelete() (string, []interface{}) {
	if b.err != nil {
		return "", nil
	}
	if b.table == "" {
		if b.err == nil {
			b.err = fmt.Errorf("orm: missing table name")
		}
		return "", nil
	}

	var sql strings.Builder
	var args []interface{}
	argIndex := 1

	sql.WriteString("DELETE FROM ")
	sql.WriteString(b.quote(b.table))

	if len(b.wheres) > 0 {
		sql.WriteString(" WHERE ")
		wSQL, wArgs := b.compileWheres(&argIndex)
		sql.WriteString(wSQL)
		args = append(args, wArgs...)
	}

	if b.err != nil {
		return "", nil
	}
	return sql.String(), args
}

func (b *QueryBuilder) compileWheres(argIndex *int) (string, []interface{}) {
	var sql strings.Builder
	var args []interface{}

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
				ph := b.dialect.Placeholder(*argIndex)
				expr = strings.Replace(expr, "?", ph, 1)
				*argIndex++
			}
			sql.WriteString(expr)
			args = append(args, w.RawArgs...)
		} else {
			ph := b.dialect.Placeholder(*argIndex)
			*argIndex++
			sql.WriteString(fmt.Sprintf("%s %s %s", b.quote(w.Column), w.Operator, ph))
			args = append(args, w.Value)
		}
	}
	return sql.String(), args
}
