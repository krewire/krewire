package orm

import (
	"context"
	"database/sql"
	"reflect"
	"time"

	"github.com/krewire/krewire/errs"
)

// RawQuery provides raw SQL execution with structured scanning.
type RawQuery struct {
	db   *DB
	tx   *Tx
	ctx  context.Context
	sql  string
	args []interface{}
}

// WithContext associates a context.Context with the raw query execution.
func (r *RawQuery) WithContext(ctx context.Context) *RawQuery {
	r.ctx = ctx
	return r
}

// Exec executes the raw query statement without returning rows.
func (r *RawQuery) Exec() (sql.Result, error) {
	start := time.Now()
	var res sql.Result
	var err error
	if r.tx != nil {
		res, err = r.tx.Exec(r.ctx, r.sql, r.args...)
	} else if r.db != nil {
		res, err = r.db.Exec(r.ctx, r.sql, r.args...)
	} else {
		err = errs.FailureError("orm: raw query has no active database connection")
	}
	var rowsAffected int64 = -1
	if res != nil {
		if ra, raErr := res.RowsAffected(); raErr == nil {
			rowsAffected = ra
		}
	}
	if r.db != nil {
		r.db.logQuery(r.ctx, r.sql, r.args, start, err, rowsAffected)
	}
	return res, err
}

func (r *RawQuery) query() (*sql.Rows, error) {
	start := time.Now()
	var rows *sql.Rows
	var err error
	if r.tx != nil {
		rows, err = r.tx.Query(r.ctx, r.sql, r.args...)
	} else if r.db != nil {
		rows, err = r.db.Query(r.ctx, r.sql, r.args...)
	} else {
		err = errs.FailureError("orm: raw query has no active database connection")
	}
	if r.db != nil {
		r.db.logQuery(r.ctx, r.sql, r.args, start, err, -1)
	}
	return rows, err
}

func (r *RawQuery) queryRow() *sql.Row {
	start := time.Now()
	var row *sql.Row
	if r.tx != nil {
		row = r.tx.QueryRow(r.ctx, r.sql, r.args...)
	} else if r.db != nil {
		row = r.db.QueryRow(r.ctx, r.sql, r.args...)
	}
	if r.db != nil {
		r.db.logQuery(r.ctx, r.sql, r.args, start, nil, -1)
	}
	return row
}

// Scan scans the results of the raw query into dest.
// dest can be a pointer to a primitive, a pointer to a struct,
// or a pointer to a slice of structs or primitives.
func (r *RawQuery) Scan(dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.IsNil() {
		return errs.UsageError("orm: Scan requires a non-nil pointer")
	}

	elem := destVal.Elem()

	// Case 1: Slice (multiple rows)
	if elem.Kind() == reflect.Slice {
		rows, err := r.query()
		if err != nil {
			return err
		}
		defer rows.Close()

		sliceElemType := elem.Type().Elem()
		isPtr := sliceElemType.Kind() == reflect.Ptr
		baseType := sliceElemType
		if isPtr {
			baseType = sliceElemType.Elem()
		}

		if baseType.Kind() == reflect.Struct {
			tm, err := ParseModel(reflect.New(baseType).Interface())
			if err != nil {
				return err
			}
			for rows.Next() {
				newElem := reflect.New(baseType)
				if err := scanRow(rows, newElem.Interface(), tm); err != nil {
					return err
				}
				if isPtr {
					elem.Set(reflect.Append(elem, newElem))
				} else {
					elem.Set(reflect.Append(elem, newElem.Elem()))
				}
			}
			return rows.Err()
		}

		// Slice of primitives
		for rows.Next() {
			newElem := reflect.New(sliceElemType)
			if err := rows.Scan(newElem.Interface()); err != nil {
				return err
			}
			elem.Set(reflect.Append(elem, newElem.Elem()))
		}
		return rows.Err()
	}

	// Case 2: Struct (single row)
	if elem.Kind() == reflect.Struct {
		rows, err := r.query()
		if err != nil {
			return err
		}
		defer rows.Close()

		if !rows.Next() {
			return sql.ErrNoRows
		}
		tm, err := ParseModel(dest)
		if err != nil {
			return err
		}
		return scanRow(rows, dest, tm)
	}

	// Case 3: Primitive scalar value
	row := r.queryRow()
	if row == nil {
		return errs.FailureError("orm: raw query has no active database connection")
	}
	return row.Scan(dest)
}

// Pluck extracts a single column's values into dest slice pointer.
func (r *RawQuery) Pluck(dest interface{}) error {
	return r.Scan(dest)
}
