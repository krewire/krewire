package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/krewire/krewire/pkg/errs"
)

// WithTrashed includes soft-deleted records in query results.
func (q *Query) WithTrashed() *Query {
	q.withTrashed = true
	return q
}

// OnlyTrashed filters query results to include exclusively soft-deleted records.
func (q *Query) OnlyTrashed() *Query {
	q.onlyTrashed = true
	return q
}

func (q *Query) applySoftDeleteFilter(tm *TableModel) {
	if tm == nil || !tm.IsSoftDeletable || tm.DeletedAtCol == nil {
		return
	}
	if q.onlyTrashed {
		q.builder.WhereNotNull(tm.DeletedAtCol.Name)
	} else if !q.withTrashed {
		q.builder.WhereNull(tm.DeletedAtCol.Name)
	}
}

// Count returns the number of matching records.
func (q *Query) Count() (int64, error) {
	if q.tm != nil {
		q.applySoftDeleteFilter(q.tm)
	}
	q.builder.columns = []string{"COUNT(*)"}
	query, args := q.builder.BuildSelect()
	if err := q.builder.Error(); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: count failed: %v", err))
	}

	row := q.queryRow(query, args...)
	if row == nil {
		return 0, errs.FailureError("orm: no database connection for count")
	}
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: count failed: %v", err))
	}
	return count, nil
}

// Find retrieves a single record by its primary key or by existing where conditions.
func (q *Query) Find(dest interface{}, id ...interface{}) error {
	tm, err := ParseModel(dest)
	if err != nil {
		return err
	}

	if q.builder.table == "" {
		q.builder.Table(tm.TableName)
	}

	q.applySoftDeleteFilter(tm)

	if len(id) > 0 {
		if tm.PrimaryKey == nil {
			return errs.UsageError("orm: cannot find by ID on model without primary key")
		}
		q.builder.Where(tm.PrimaryKey.Name, "=", id[0])
	}

	q.builder.Limit(1)
	query, args := q.builder.BuildSelect()
	if err := q.builder.Error(); err != nil {
		return errs.FailureError(fmt.Sprintf("orm: find query failed: %v", err))
	}

	rows, err := q.query(query, args...)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: find query failed: %v", err))
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	return scanRow(rows, dest, tm)
}

// First retrieves the first matching record.
func (q *Query) First(dest interface{}) error {
	return q.Find(dest)
}

// FirstOrFail retrieves the first matching record or returns an error when not found.
func (q *Query) FirstOrFail(dest interface{}) error {
	err := q.Find(dest)
	if errors.Is(err, sql.ErrNoRows) {
		return errs.FailureError("orm: record not found")
	}
	return err
}

// FindOrFail retrieves a record by primary key or returns an error when not found.
func (q *Query) FindOrFail(dest interface{}, id interface{}) error {
	err := q.Find(dest, id)
	if errors.Is(err, sql.ErrNoRows) {
		return errs.FailureError(fmt.Sprintf("orm: record not found with id %v", id))
	}
	return err
}

// Exists reports whether any records match the query.
func (q *Query) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DoesntExist reports whether no records match the query.
func (q *Query) DoesntExist() (bool, error) {
	ex, err := q.Exists()
	if err != nil {
		return false, err
	}
	return !ex, nil
}

// Pluck extracts a single column's values into a destination slice pointer.
func (q *Query) Pluck(column string, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Slice {
		return errs.UsageError("orm: Pluck requires a pointer to a slice")
	}
	sliceVal := destVal.Elem()
	elemType := sliceVal.Type().Elem()

	q.builder.columns = []string{column}
	if q.tm != nil {
		q.applySoftDeleteFilter(q.tm)
	}

	query, args := q.builder.BuildSelect()
	if err := q.builder.Error(); err != nil {
		return errs.FailureError(fmt.Sprintf("orm: pluck failed: %v", err))
	}

	rows, err := q.query(query, args...)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: pluck failed: %v", err))
	}
	defer rows.Close()

	for rows.Next() {
		newElem := reflect.New(elemType)
		if err := rows.Scan(newElem.Interface()); err != nil {
			return err
		}
		sliceVal.Set(reflect.Append(sliceVal, newElem.Elem()))
	}
	return rows.Err()
}

// Chunk breaks query results into batches and invokes callback fn for each chunk.
func (q *Query) Chunk(size int, destSlice interface{}, fn func() error) error {
	if size <= 0 {
		return errs.UsageError("orm: chunk size must be greater than zero")
	}

	page := 1
	for {
		q.builder.Limit(size).Offset((page - 1) * size)
		if err := q.FindAll(destSlice); err != nil {
			return err
		}
		sliceVal := reflect.ValueOf(destSlice).Elem()
		if sliceVal.Len() == 0 {
			break
		}
		if err := fn(); err != nil {
			return err
		}
		if sliceVal.Len() < size {
			break
		}
		page++
	}
	return nil
}

// FindAll retrieves all matching records into a slice pointer.
func (q *Query) FindAll(dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Slice {
		return errs.UsageError("orm: FindAll requires a pointer to a slice")
	}

	sliceVal := destVal.Elem()
	elemType := sliceVal.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	baseType := elemType
	if isPtr {
		baseType = elemType.Elem()
	}

	tm, err := ParseModel(reflect.New(baseType).Interface())
	if err != nil {
		return err
	}

	if q.builder.table == "" {
		q.builder.Table(tm.TableName)
	}

	q.applySoftDeleteFilter(tm)

	query, args := q.builder.BuildSelect()
	if err := q.builder.Error(); err != nil {
		return errs.FailureError(fmt.Sprintf("orm: findAll query failed: %v", err))
	}
	rows, err := q.query(query, args...)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: findAll query failed: %v", err))
	}
	defer rows.Close()

	for rows.Next() {
		newElem := reflect.New(baseType)
		if err := scanRow(rows, newElem.Interface(), tm); err != nil {
			return err
		}

		if isPtr {
			sliceVal.Set(reflect.Append(sliceVal, newElem))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, newElem.Elem()))
		}
	}

	return rows.Err()
}
