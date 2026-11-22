package orm

import (
	"context"
	"database/sql"
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
	preloads    []preloadItem
}

func (q *Query) queryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	var row *sql.Row
	if q.tx != nil {
		row = q.tx.QueryRow(q.ctx, query, args...)
	} else if q.db != nil {
		row = q.db.QueryRow(q.ctx, query, args...)
	}
	if q.db != nil {
		q.db.logQuery(q.ctx, query, args, start, nil, -1)
	}
	return row
}

func (q *Query) query(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	var rows *sql.Rows
	var err error
	if q.tx != nil {
		rows, err = q.tx.Query(q.ctx, query, args...)
	} else if q.db != nil {
		rows, err = q.db.Query(q.ctx, query, args...)
	} else {
		err = errs.FailureError("orm: no database or transaction attached to query")
	}
	if q.db != nil {
		q.db.logQuery(q.ctx, query, args, start, err, -1)
	}
	return rows, err
}

func (q *Query) exec(query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	var res sql.Result
	var err error
	if q.tx != nil {
		res, err = q.tx.Exec(q.ctx, query, args...)
	} else if q.db != nil {
		res, err = q.db.Exec(q.ctx, query, args...)
	} else {
		err = errs.FailureError("orm: no database or transaction attached to query")
	}
	var rowsAffected int64 = -1
	if res != nil {
		if ra, raErr := res.RowsAffected(); raErr == nil {
			rowsAffected = ra
		}
	}
	if q.db != nil {
		q.db.logQuery(q.ctx, query, args, start, err, rowsAffected)
	}
	return res, err
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
