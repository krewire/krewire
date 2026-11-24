package orm

import (
	"fmt"
	"reflect"
	"time"

	"github.com/krewire/krewire/errs"
)

// Delete removes records matching where conditions or by primary key.
// If the model supports soft deletes, it updates deleted_at instead, unless ForceDelete is used.
func (q *Query) Delete(record interface{}, id ...interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}

	// Before hook
	if err := runBeforeDeleteHooks(q.ctx, record); err != nil {
		return 0, err
	}

	var rowsAffected int64

	if tm.IsSoftDeletable && !q.forceDelete {
		// Soft delete: UPDATE table SET deleted_at = NOW() WHERE ...
		now := time.Now()
		if reflect.ValueOf(record).Kind() == reflect.Ptr {
			val := reflect.ValueOf(record).Elem()
			f := val.FieldByName(tm.DeletedAtCol.FieldName)
			if f.IsValid() && f.CanSet() {
				f.Set(reflect.ValueOf(&now))
			}
		}

		if q.builder.table == "" {
			q.builder.Table(tm.TableName)
		}

		if len(id) > 0 {
			if tm.PrimaryKey == nil {
				return 0, errs.UsageError("orm: cannot delete by ID without primary key")
			}
			q.builder.Where(tm.PrimaryKey.Name, "=", id[0])
		} else if tm.PrimaryKey != nil {
			val := reflect.ValueOf(record)
			for val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			pkVal := val.FieldByName(tm.PrimaryKey.FieldName).Interface()
			q.builder.Where(tm.PrimaryKey.Name, "=", pkVal)
		}

		query, args := q.builder.BuildUpdate([]string{tm.DeletedAtCol.Name}, []interface{}{now})
		if err := q.builder.Error(); err != nil {
			return 0, errs.FailureError(fmt.Sprintf("orm: soft delete failed: %v", err))
		}
		res, err := q.exec(query, args...)
		if err != nil {
			return 0, errs.FailureError(fmt.Sprintf("orm: soft delete failed: %v", err))
		}
		rowsAffected, _ = res.RowsAffected()
	} else {
		// Hard delete: DELETE FROM table WHERE ...
		if q.builder.table == "" {
			q.builder.Table(tm.TableName)
		}

		if len(id) > 0 {
			if tm.PrimaryKey == nil {
				return 0, errs.UsageError("orm: cannot delete by ID without primary key")
			}
			q.builder.Where(tm.PrimaryKey.Name, "=", id[0])
		} else if tm.PrimaryKey != nil {
			val := reflect.ValueOf(record)
			for val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			pkVal := val.FieldByName(tm.PrimaryKey.FieldName).Interface()
			q.builder.Where(tm.PrimaryKey.Name, "=", pkVal)
		}

		query, args := q.builder.BuildDelete()
		if err := q.builder.Error(); err != nil {
			return 0, errs.FailureError(fmt.Sprintf("orm: delete failed: %v", err))
		}
		res, err := q.exec(query, args...)
		if err != nil {
			return 0, errs.FailureError(fmt.Sprintf("orm: delete failed: %v", err))
		}
		rowsAffected, _ = res.RowsAffected()
	}

	// After hook
	if err := runAfterDeleteHooks(q.ctx, record); err != nil {
		return rowsAffected, err
	}

	return rowsAffected, nil
}

// ForceDelete permanently deletes a soft-deletable record from the database.
func (q *Query) ForceDelete(record interface{}, id ...interface{}) (int64, error) {
	q.forceDelete = true
	return q.Delete(record, id...)
}

// Restore restores a soft-deleted record by resetting its deleted_at timestamp to NULL.
func (q *Query) Restore(record interface{}, id ...interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}
	if !tm.IsSoftDeletable {
		return 0, errs.UsageError("orm: model does not support soft deletes")
	}

	if reflect.ValueOf(record).Kind() == reflect.Ptr {
		val := reflect.ValueOf(record).Elem()
		f := val.FieldByName(tm.DeletedAtCol.FieldName)
		if f.IsValid() && f.CanSet() {
			var nilTime *time.Time
			f.Set(reflect.ValueOf(nilTime))
		}
	}

	if q.builder.table == "" {
		q.builder.Table(tm.TableName)
	}

	if len(id) > 0 {
		if tm.PrimaryKey == nil {
			return 0, errs.UsageError("orm: cannot restore by ID without primary key")
		}
		q.builder.Where(tm.PrimaryKey.Name, "=", id[0])
	} else if tm.PrimaryKey != nil {
		val := reflect.ValueOf(record)
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		pkVal := val.FieldByName(tm.PrimaryKey.FieldName).Interface()
		q.builder.Where(tm.PrimaryKey.Name, "=", pkVal)
	}

	query, args := q.builder.BuildUpdate([]string{tm.DeletedAtCol.Name}, []interface{}{nil})
	if err := q.builder.Error(); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: restore failed: %v", err))
	}
	res, err := q.exec(query, args...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: restore failed: %v", err))
	}
	return res.RowsAffected()
}
