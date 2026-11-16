package orm

import (
	"fmt"
	"reflect"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

// Update updates a record in the database using its primary key.
func (q *Query) Update(record interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}

	if tm.PrimaryKey == nil {
		return 0, errs.UsageError("orm: update requires a primary key on model")
	}

	now := time.Now()
	val := reflect.ValueOf(record)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	applyUpdateTimestamps(val, tm, now)

	// Before hooks
	if err := runBeforeUpdateHooks(q.ctx, record); err != nil {
		return 0, err
	}

	pkVal := val.FieldByName(tm.PrimaryKey.FieldName).Interface()

	var cols []string
	var vals []interface{}

	for _, col := range tm.Columns {
		if col.IsPrimaryKey {
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

	q.builder.Where(tm.PrimaryKey.Name, "=", pkVal)
	query, args := q.builder.BuildUpdate(cols, vals)
	if err := q.builder.Error(); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: update failed: %v", err))
	}

	res, err := q.exec(query, args...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: update failed: %v", err))
	}

	rowsAffected, _ := res.RowsAffected()

	// After hooks
	if err := runAfterUpdateHooks(q.ctx, record); err != nil {
		return rowsAffected, err
	}

	return rowsAffected, nil
}
