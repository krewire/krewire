package orm

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/krewire/krewire/errs"
)

// InsertMany inserts multiple records into the database table in a single batch statement.
func (q *Query) InsertMany(slice interface{}) (int64, error) {
	sliceVal := reflect.ValueOf(slice)
	for sliceVal.Kind() == reflect.Ptr {
		sliceVal = sliceVal.Elem()
	}
	if sliceVal.Kind() != reflect.Slice {
		return 0, errs.UsageError("orm: InsertMany requires a slice or pointer to a slice")
	}

	numRows := sliceVal.Len()
	if numRows == 0 {
		return 0, nil
	}

	firstElem := sliceVal.Index(0).Interface()
	tm, err := ParseModel(firstElem)
	if err != nil {
		return 0, err
	}

	if q.builder.table == "" {
		q.builder.Table(tm.TableName)
	}

	var cols []string
	for _, col := range tm.Columns {
		if col.IsPrimaryKey && col.IsAutoInc {
			continue
		}
		cols = append(cols, col.Name)
	}

	now := time.Now()
	var allVals []interface{}
	argIndex := 1
	var rowClauses []string

	for i := 0; i < numRows; i++ {
		elemVal := sliceVal.Index(i)
		for elemVal.Kind() == reflect.Ptr {
			elemVal = elemVal.Elem()
		}
		applyInsertTimestamps(elemVal, tm, now)

		if err := runBeforeCreateHooks(q.ctx, sliceVal.Index(i).Interface()); err != nil {
			return 0, err
		}

		var rowPlaceholders []string
		for _, col := range tm.Columns {
			if col.IsPrimaryKey && col.IsAutoInc {
				continue
			}
			fVal := elemVal.FieldByName(col.FieldName)
			if fVal.IsValid() {
				allVals = append(allVals, fVal.Interface())
			} else {
				allVals = append(allVals, nil)
			}
			rowPlaceholders = append(rowPlaceholders, q.builder.dialect.Placeholder(argIndex))
			argIndex++
		}
		rowClauses = append(rowClauses, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	var quotedCols []string
	for _, c := range cols {
		quotedCols = append(quotedCols, q.builder.quote(c))
	}

	querySQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		q.builder.quote(q.builder.table),
		strings.Join(quotedCols, ", "),
		strings.Join(rowClauses, ", "),
	)

	res, err := q.exec(querySQL, allVals...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: batch insert failed: %v", err))
	}
	return res.RowsAffected()
}

// Upsert inserts a record or updates its columns if conflict occurs on conflictCols.
func (q *Query) Upsert(record interface{}, conflictCols []string, updateCols []string) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	val := reflect.ValueOf(record)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	applyInsertTimestamps(val, tm, now)
	applyUpdateTimestamps(val, tm, now)

	var insertCols []string
	var insertVals []interface{}

	for _, col := range tm.Columns {
		fieldVal := val.FieldByName(col.FieldName)
		if !fieldVal.IsValid() {
			continue
		}
		if col.IsPrimaryKey && col.IsAutoInc && isZeroVal(fieldVal.Interface()) {
			continue
		}
		insertCols = append(insertCols, col.Name)
		insertVals = append(insertVals, fieldVal.Interface())
	}

	if q.builder.table == "" {
		q.builder.Table(tm.TableName)
	}

	baseInsert := q.builder.BuildInsert(insertCols)
	var upsertSQL string

	dialectName := strings.ToLower(q.builder.dialect.Name())
	switch dialectName {
	case "mysql", "mariadb":
		var setParts []string
		for _, col := range updateCols {
			qCol := q.builder.quote(col)
			setParts = append(setParts, fmt.Sprintf("%s = VALUES(%s)", qCol, qCol))
		}
		upsertSQL = fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s", baseInsert, strings.Join(setParts, ", "))
	default: // SQLite and PostgreSQL
		var qConflict []string
		for _, c := range conflictCols {
			qConflict = append(qConflict, q.builder.quote(c))
		}
		var setParts []string
		for _, col := range updateCols {
			qCol := q.builder.quote(col)
			setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", qCol, qCol))
		}
		upsertSQL = fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s",
			baseInsert,
			strings.Join(qConflict, ", "),
			strings.Join(setParts, ", "),
		)
	}

	res, err := q.exec(upsertSQL, insertVals...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: upsert failed: %v", err))
	}
	return res.RowsAffected()
}
