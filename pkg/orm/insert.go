package orm

import (
	"fmt"
	"reflect"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

// Insert inserts a record into the table and assigns the auto-generated ID if available.
func (q *Query) Insert(record interface{}) (int64, error) {
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

	// Before hooks
	if err := runBeforeCreateHooks(q.ctx, record); err != nil {
		return 0, err
	}

	var cols []string
	var vals []interface{}

	for _, col := range tm.Columns {
		// Skip auto-increment PK on insert
		if col.IsPrimaryKey && col.IsAutoInc {
			continue
		}

		fieldVal := val.FieldByName(col.FieldName)
		if !fieldVal.IsValid() {
			continue
		}

		cols = append(cols, col.Name)
		vals = append(vals, fieldVal.Interface())
	}

	if q.builder.table == "" {
		q.builder.Table(tm.TableName)
	}

	sqlStr := q.builder.BuildInsert(cols)
	if err := q.builder.Error(); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: insert failed: %v", err))
	}
	res, err := q.exec(sqlStr, vals...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: insert failed: %v", err))
	}

	lastID, _ := res.LastInsertId()
	rowsAffected, _ := res.RowsAffected()

	// Assign last insert ID back to the struct if it's a pointer and has integer auto-inc PK
	if lastID > 0 && tm.PrimaryKey != nil && tm.PrimaryKey.IsAutoInc {
		if reflect.ValueOf(record).Kind() == reflect.Ptr {
			pkField := val.FieldByName(tm.PrimaryKey.FieldName)
			if pkField.CanSet() && isIntegerType(pkField.Type()) {
				pkField.SetInt(lastID)
			}
		}
	}

	// After hooks
	if err := runAfterCreateHooks(q.ctx, record); err != nil {
		return rowsAffected, err
	}

	return rowsAffected, nil
}
