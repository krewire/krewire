package repository

import (
	"context"
	"fmt"
	"math"
	"reflect"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
)

// Pagination holds paginated results and navigation metadata.
type Pagination[T any] struct {
	Items      []*T  `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// Repository defines the generic type-safe Data Access Object (DAO) contract.
type Repository[T any] interface {
	// Retrieval
	FindByID(ctx context.Context, id any) (*T, error)
	FindAll(ctx context.Context) ([]*T, error)
	FindWhere(ctx context.Context, fn func(q *orm.Query) *orm.Query) ([]*T, error)
	FirstWhere(ctx context.Context, fn func(q *orm.Query) *orm.Query) (*T, error)
	Count(ctx context.Context, fn ...func(q *orm.Query) *orm.Query) (int64, error)
	Exists(ctx context.Context, fn ...func(q *orm.Query) *orm.Query) (bool, error)
	Paginate(ctx context.Context, page, perPage int, fn ...func(q *orm.Query) *orm.Query) (*Pagination[T], error)

	// Persistence
	Create(ctx context.Context, entity *T) error
	CreateMany(ctx context.Context, entities []*T) error
	Update(ctx context.Context, entity *T) error
	Save(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id any) error
	DeleteEntity(ctx context.Context, entity *T) error

	// Infrastructure & Querying
	Query(ctx context.Context) *orm.Query
	Table() string
	DB() *orm.DB
	Transaction(ctx context.Context, fn func(repo Repository[T]) error) error
}

// BaseRepository provides the standard generic implementation of Repository[T].
type BaseRepository[T any] struct {
	db        *orm.DB
	tx        *orm.Tx
	tableName string
	tm        *orm.TableModel
}

// New creates a new BaseRepository for entity type T. If table name is omitted,
// it is automatically derived from the model struct or TableName() method.
func New[T any](db *orm.DB, table ...string) *BaseRepository[T] {
	var sample T
	tm, _ := orm.ParseModel(&sample)

	resolvedTable := ""
	if len(table) > 0 && table[0] != "" {
		resolvedTable = table[0]
	} else if tm != nil {
		resolvedTable = tm.TableName
	}

	return &BaseRepository[T]{
		db:        db,
		tableName: resolvedTable,
		tm:        tm,
	}
}

// DB returns the underlying orm.DB instance.
func (r *BaseRepository[T]) DB() *orm.DB {
	return r.db
}

// Tx returns the current transaction if the repository is within a transaction.
func (r *BaseRepository[T]) Tx() *orm.Tx {
	return r.tx
}

// Table returns the target database table name.
func (r *BaseRepository[T]) Table() string {
	return r.tableName
}

// Query creates a new *orm.Query bound to the entity and context.
func (r *BaseRepository[T]) Query(ctx context.Context) *orm.Query {
	var sample T
	var q *orm.Query
	if r.tx != nil {
		q = r.tx.Model(&sample).WithContext(ctx)
	} else if r.db != nil {
		q = r.db.Model(&sample).WithContext(ctx)
	} else {
		return nil
	}
	if r.tableName != "" {
		q.Table(r.tableName)
	}
	return q
}

// FindByID retrieves an entity by its primary key.
func (r *BaseRepository[T]) FindByID(ctx context.Context, id any) (*T, error) {
	var entity T
	if err := r.Query(ctx).Find(&entity, id); err != nil {
		return nil, err
	}
	return &entity, nil
}

// FindAll retrieves all active entities from the table.
func (r *BaseRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	var items []*T
	if err := r.Query(ctx).FindAll(&items); err != nil {
		return nil, err
	}
	return items, nil
}

// FindWhere retrieves matching entities using a query customization callback.
func (r *BaseRepository[T]) FindWhere(ctx context.Context, fn func(q *orm.Query) *orm.Query) ([]*T, error) {
	q := r.Query(ctx)
	if fn != nil {
		q = fn(q)
	}
	var items []*T
	if err := q.FindAll(&items); err != nil {
		return nil, err
	}
	return items, nil
}

// FirstWhere retrieves the first matching entity using a query customization callback.
func (r *BaseRepository[T]) FirstWhere(ctx context.Context, fn func(q *orm.Query) *orm.Query) (*T, error) {
	q := r.Query(ctx)
	if fn != nil {
		q = fn(q)
	}
	var entity T
	if err := q.First(&entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// Count returns the number of entities matching optional criteria.
func (r *BaseRepository[T]) Count(ctx context.Context, fn ...func(q *orm.Query) *orm.Query) (int64, error) {
	q := r.Query(ctx)
	for _, f := range fn {
		if f != nil {
			q = f(q)
		}
	}
	return q.Count()
}

// Exists reports whether any entity matches the criteria.
func (r *BaseRepository[T]) Exists(ctx context.Context, fn ...func(q *orm.Query) *orm.Query) (bool, error) {
	count, err := r.Count(ctx, fn...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Paginate retrieves a paginated subset of records along with total count and page metadata.
func (r *BaseRepository[T]) Paginate(ctx context.Context, page, perPage int, fn ...func(q *orm.Query) *orm.Query) (*Pagination[T], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}

	total, err := r.Count(ctx, fn...)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	offset := (page - 1) * perPage
	q := r.Query(ctx).Limit(perPage).Offset(offset)
	for _, f := range fn {
		if f != nil {
			q = f(q)
		}
	}

	var items []*T
	if err := q.FindAll(&items); err != nil {
		return nil, err
	}

	return &Pagination[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}, nil
}

// Create inserts a new entity into the database and populates its auto-generated ID.
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	if entity == nil {
		return errs.UsageError("repository: entity is nil")
	}
	_, err := r.Query(ctx).Insert(entity)
	return err
}

// CreateMany inserts multiple entities into the database sequentially.
func (r *BaseRepository[T]) CreateMany(ctx context.Context, entities []*T) error {
	for i, entity := range entities {
		if err := r.Create(ctx, entity); err != nil {
			return errs.FailureError(fmt.Sprintf("repository: create many failed at index [%d]: %v", i, err))
		}
	}
	return nil
}

// Update updates an existing entity by its primary key.
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	if entity == nil {
		return errs.UsageError("repository: entity is nil")
	}
	_, err := r.Query(ctx).Update(entity)
	return err
}

// Save creates or updates the entity based on whether its primary key is already set.
func (r *BaseRepository[T]) Save(ctx context.Context, entity *T) error {
	if entity == nil {
		return errs.UsageError("repository: entity is nil")
	}
	if r.tm != nil && r.tm.PrimaryKey != nil {
		val := reflect.ValueOf(entity)
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		pkField := val.FieldByName(r.tm.PrimaryKey.FieldName)
		if pkField.IsValid() {
			switch pkField.Kind() {
			case reflect.Int, reflect.Int64, reflect.Int32:
				if pkField.Int() > 0 {
					return r.Update(ctx, entity)
				}
			case reflect.Uint, reflect.Uint64, reflect.Uint32:
				if pkField.Uint() > 0 {
					return r.Update(ctx, entity)
				}
			case reflect.String:
				if pkField.String() != "" {
					return r.Update(ctx, entity)
				}
			}
		}
	}
	return r.Create(ctx, entity)
}

// Delete removes an entity by primary key.
func (r *BaseRepository[T]) Delete(ctx context.Context, id any) error {
	var sample T
	_, err := r.Query(ctx).Delete(&sample, id)
	return err
}

// DeleteEntity removes or soft-deletes a specific entity.
func (r *BaseRepository[T]) DeleteEntity(ctx context.Context, entity *T) error {
	if entity == nil {
		return errs.UsageError("repository: entity is nil")
	}
	_, err := r.Query(ctx).Delete(entity)
	return err
}

// Transaction executes a function within a database transaction.
func (r *BaseRepository[T]) Transaction(ctx context.Context, fn func(repo Repository[T]) error) error {
	if r.db == nil {
		return errs.UsageError("repository: database is nil")
	}
	return r.db.Transaction(ctx, func(tx *orm.Tx) error {
		txRepo := &BaseRepository[T]{
			db:        r.db,
			tx:        tx,
			tableName: r.tableName,
			tm:        r.tm,
		}
		return fn(txRepo)
	})
}
