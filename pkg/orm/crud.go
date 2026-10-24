package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
)

// Query provides fluent database querying and struct mapping.
type Query struct {
	db      *DB
	builder *QueryBuilder
	ctx     context.Context
	model   interface{}
}

func newQuery(db *DB, table string, model interface{}) *Query {
	var dialect Dialect = &SQLiteDialect{}
	if db != nil && db.dialect != nil {
		dialect = db.dialect
	}

	builder := NewQueryBuilder(dialect).Table(table)

	return &Query{
		db:      db,
		builder: builder,
		ctx:     context.Background(),
		model:   model,
	}
}

// WithContext associates a context.Context with the query execution.
func (q *Query) WithContext(ctx context.Context) *Query {
	q.ctx = ctx
	return q
}

// Where adds an AND condition.
func (q *Query) Where(col, op string, val interface{}) *Query {
	q.builder.Where(col, op, val)
	return q
}

// OrWhere adds an OR condition.
func (q *Query) OrWhere(col, op string, val interface{}) *Query {
	q.builder.OrWhere(col, op, val)
	return q
}

// WhereIn adds an IN (...) condition.
func (q *Query) WhereIn(col string, vals ...interface{}) *Query {
	q.builder.WhereIn(col, vals...)
	return q
}

// OrderBy adds ordering.
func (q *Query) OrderBy(col, dir string) *Query {
	q.builder.OrderBy(col, dir)
	return q
}

// Limit sets maximum rows to return.
func (q *Query) Limit(n int) *Query {
	q.builder.Limit(n)
	return q
}

// Offset sets rows to skip.
func (q *Query) Offset(n int) *Query {
	q.builder.Offset(n)
	return q
}

// Count returns the number of matching records.
func (q *Query) Count() (int64, error) {
	q.builder.columns = []string{"COUNT(*)"}
	query, args := q.builder.BuildSelect()

	row := q.db.QueryRow(q.ctx, query, args...)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: count failed: %v", err))
	}
	return count, nil
}

// Insert inserts a record into the table and assigns the auto-generated ID if available.
func (q *Query) Insert(record interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}

	val := reflect.ValueOf(record)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
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
	res, err := q.db.Exec(q.ctx, sqlStr, vals...)
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

	return rowsAffected, nil
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

	if len(id) > 0 {
		if tm.PrimaryKey == nil {
			return errs.UsageError("orm: cannot find by ID on model without primary key")
		}
		q.builder.Where(tm.PrimaryKey.Name, "=", id[0])
	}

	q.builder.Limit(1)
	query, args := q.builder.BuildSelect()

	rows, err := q.db.Query(q.ctx, query, args...)
	if err != nil {
		return errs.FailureError(fmt.Sprintf("orm: find query failed: %v", err))
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	return scanRow(rows, dest, tm)
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

	query, args := q.builder.BuildSelect()
	rows, err := q.db.Query(q.ctx, query, args...)
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

// Update updates a record in the database using its primary key.
func (q *Query) Update(record interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}

	if tm.PrimaryKey == nil {
		return 0, errs.UsageError("orm: update requires a primary key on model")
	}

	val := reflect.ValueOf(record)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
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

	res, err := q.db.Exec(q.ctx, query, args...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: update failed: %v", err))
	}
	return res.RowsAffected()
}

// Delete removes records matching where conditions or by primary key.
func (q *Query) Delete(record interface{}, id ...interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
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

	query, args := q.builder.BuildDelete()
	res, err := q.db.Exec(q.ctx, query, args...)
	if err != nil {
		return 0, errs.FailureError(fmt.Sprintf("orm: delete failed: %v", err))
	}
	return res.RowsAffected()
}

func scanRow(rows *sql.Rows, dest interface{}, tm *TableModel) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	val := reflect.ValueOf(dest)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	scanPointers := make([]interface{}, len(cols))
	for i, colName := range cols {
		col, ok := tm.ColumnMap[strings.ToLower(colName)]
		if ok {
			fieldVal := val.FieldByName(col.FieldName)
			if fieldVal.IsValid() && fieldVal.CanAddr() {
				scanPointers[i] = fieldVal.Addr().Interface()
				continue
			}
		}
		var dummy interface{}
		scanPointers[i] = &dummy
	}

	return rows.Scan(scanPointers...)
}
