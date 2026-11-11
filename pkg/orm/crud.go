package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
)

// Query provides fluent database querying and struct mapping with Eloquent-like ergonomics.
type Query struct {
	db          *DB
	tx          *Tx
	builder     *QueryBuilder
	ctx         context.Context
	model       interface{}
	tm          *TableModel
	withTrashed bool
	onlyTrashed bool
	forceDelete bool
}

func (q *Query) queryRow(query string, args ...interface{}) *sql.Row {
	if q.tx != nil {
		return q.tx.QueryRow(q.ctx, query, args...)
	}
	if q.db != nil {
		return q.db.QueryRow(q.ctx, query, args...)
	}
	return nil
}

func (q *Query) query(query string, args ...interface{}) (*sql.Rows, error) {
	if q.tx != nil {
		return q.tx.Query(q.ctx, query, args...)
	}
	if q.db != nil {
		return q.db.Query(q.ctx, query, args...)
	}
	return nil, errs.FailureError("orm: no database or transaction attached to query")
}

func (q *Query) exec(query string, args ...interface{}) (sql.Result, error) {
	if q.tx != nil {
		return q.tx.Exec(q.ctx, query, args...)
	}
	if q.db != nil {
		return q.db.Exec(q.ctx, query, args...)
	}
	return nil, errs.FailureError("orm: no database or transaction attached to query")
}

// WithTx associates an active transaction with the query.
func (q *Query) WithTx(tx *Tx) *Query {
	q.tx = tx
	return q
}

func newQuery(db *DB, table string, model interface{}) *Query {
	var dialect Dialect = &SQLiteDialect{}
	if db != nil && db.dialect != nil {
		dialect = db.dialect
	}

	builder := NewQueryBuilder(dialect).Table(table)

	var tm *TableModel
	if model != nil {
		tm, _ = ParseModel(model)
		if table == "" && tm != nil {
			builder.Table(tm.TableName)
		}
	}

	return &Query{
		db:      db,
		builder: builder,
		ctx:     context.Background(),
		model:   model,
		tm:      tm,
	}
}

// WithContext associates a context.Context with the query execution.
func (q *Query) WithContext(ctx context.Context) *Query {
	q.ctx = ctx
	return q
}

// Table sets or overrides the target table name for the query.
func (q *Query) Table(name string) *Query {
	q.builder.Table(name)
	return q
}

// Select sets specific columns to retrieve.
func (q *Query) Select(cols ...string) *Query {
	q.builder.Select(cols...)
	return q
}

// Distinct ensures unique rows are returned.
func (q *Query) Distinct() *Query {
	q.builder.Distinct()
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

// WhereNull adds an IS NULL condition.
func (q *Query) WhereNull(col string) *Query {
	q.builder.WhereNull(col)
	return q
}

// WhereNotNull adds an IS NOT NULL condition.
func (q *Query) WhereNotNull(col string) *Query {
	q.builder.WhereNotNull(col)
	return q
}

// WhereRaw adds a raw WHERE clause with optional bound arguments.
func (q *Query) WhereRaw(clause string, args ...interface{}) *Query {
	q.builder.WhereRaw(clause, args...)
	return q
}

// Join adds a JOIN clause.
func (q *Query) Join(joinType, table, on string) *Query {
	q.builder.Join(joinType, table, on)
	return q
}

// LeftJoin adds a LEFT JOIN clause.
func (q *Query) LeftJoin(table, on string) *Query {
	q.builder.LeftJoin(table, on)
	return q
}

// InnerJoin adds an INNER JOIN clause.
func (q *Query) InnerJoin(table, on string) *Query {
	q.builder.InnerJoin(table, on)
	return q
}

// GroupBy adds GROUP BY columns.
func (q *Query) GroupBy(cols ...string) *Query {
	q.builder.GroupBy(cols...)
	return q
}

// Having adds a HAVING clause.
func (q *Query) Having(clause string, args ...interface{}) *Query {
	q.builder.Having(clause, args...)
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

// Scope applies a query scope callback to this query.
func (q *Query) Scope(fn func(q *Query) *Query) *Query {
	if fn != nil {
		return fn(q)
	}
	return q
}

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

	// Auto-populate timestamps if zero
	if tm.CreatedAtCol != nil {
		field := val.FieldByName(tm.CreatedAtCol.FieldName)
		if field.IsValid() && field.CanSet() {
			if t, ok := field.Interface().(time.Time); ok && t.IsZero() {
				field.Set(reflect.ValueOf(now))
			}
		}
	}
	if tm.UpdatedAtCol != nil {
		field := val.FieldByName(tm.UpdatedAtCol.FieldName)
		if field.IsValid() && field.CanSet() {
			if t, ok := field.Interface().(time.Time); ok && t.IsZero() {
				field.Set(reflect.ValueOf(now))
			}
		}
	}

	// Before hooks
	if hook, ok := record.(BeforeSaver); ok {
		if err := hook.BeforeSave(q.ctx); err != nil {
			return 0, err
		}
	}
	if hook, ok := record.(BeforeCreator); ok {
		if err := hook.BeforeCreate(q.ctx); err != nil {
			return 0, err
		}
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
	if hook, ok := record.(AfterCreator); ok {
		if err := hook.AfterCreate(q.ctx); err != nil {
			return rowsAffected, err
		}
	}
	if hook, ok := record.(AfterSaver); ok {
		if err := hook.AfterSave(q.ctx); err != nil {
			return rowsAffected, err
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

	// Auto-populate UpdatedAt
	if tm.UpdatedAtCol != nil {
		field := val.FieldByName(tm.UpdatedAtCol.FieldName)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(now))
		}
	}

	// Before hooks
	if hook, ok := record.(BeforeSaver); ok {
		if err := hook.BeforeSave(q.ctx); err != nil {
			return 0, err
		}
	}
	if hook, ok := record.(BeforeUpdater); ok {
		if err := hook.BeforeUpdate(q.ctx); err != nil {
			return 0, err
		}
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
	if hook, ok := record.(AfterUpdater); ok {
		if err := hook.AfterUpdate(q.ctx); err != nil {
			return rowsAffected, err
		}
	}
	if hook, ok := record.(AfterSaver); ok {
		if err := hook.AfterSave(q.ctx); err != nil {
			return rowsAffected, err
		}
	}

	return rowsAffected, nil
}

// Delete removes records matching where conditions or by primary key.
// If the model supports soft deletes, it updates deleted_at instead, unless ForceDelete is used.
func (q *Query) Delete(record interface{}, id ...interface{}) (int64, error) {
	tm, err := ParseModel(record)
	if err != nil {
		return 0, err
	}

	// Before hook
	if hook, ok := record.(BeforeDeleter); ok {
		if err := hook.BeforeDelete(q.ctx); err != nil {
			return 0, err
		}
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
	if hook, ok := record.(AfterDeleter); ok {
		if err := hook.AfterDelete(q.ctx); err != nil {
			return rowsAffected, err
		}
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
