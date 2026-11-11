package seeder

import (
	"context"
	"fmt"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/faker"
)

// GeneratorFunc defines how to generate a new model instance using fake data.
type GeneratorFunc[T any] func(f *faker.Faker) *T

// StateFunc applies mutations or specific attribute states to a model instance.
type StateFunc[T any] func(item *T)

// Factory provides fluent model generation and database seeding (mirroring Laravel Model Factories).
type Factory[T any] struct {
	generator GeneratorFunc[T]
	states    []StateFunc[T]
	count     int
	faker     *faker.Faker
}

// NewFactory creates a new model factory given a generator function.
func NewFactory[T any](generator GeneratorFunc[T]) *Factory[T] {
	return &Factory[T]{
		generator: generator,
		count:     1,
		faker:     faker.Default(),
	}
}

// Count specifies the number of models to produce when MakeMany or CreateMany is invoked.
func (f *Factory[T]) Count(n int) *Factory[T] {
	clone := f.clone()
	if n < 1 {
		n = 1
	}
	clone.count = n
	return clone
}

// State registers one or more mutation callbacks (e.g. for creating admin users or verified emails).
func (f *Factory[T]) State(states ...StateFunc[T]) *Factory[T] {
	clone := f.clone()
	clone.states = append(clone.states, states...)
	return clone
}

// WithFaker customizes the Faker instance used for generation (e.g. for deterministic seeding).
func (f *Factory[T]) WithFaker(fk *faker.Faker) *Factory[T] {
	clone := f.clone()
	if fk != nil {
		clone.faker = fk
	}
	return clone
}

func (f *Factory[T]) clone() *Factory[T] {
	statesCopy := make([]StateFunc[T], len(f.states))
	copy(statesCopy, f.states)
	return &Factory[T]{
		generator: f.generator,
		states:    statesCopy,
		count:     f.count,
		faker:     f.faker,
	}
}

// Make generates a single in-memory model instance with all states applied.
func (f *Factory[T]) Make() *T {
	if f.generator == nil {
		var zero T
		return &zero
	}
	item := f.generator(f.faker)
	for _, state := range f.states {
		if state != nil {
			state(item)
		}
	}
	return item
}

// MakeMany generates a slice of in-memory model instances of length Count.
func (f *Factory[T]) MakeMany() []*T {
	items := make([]*T, f.count)
	for i := 0; i < f.count; i++ {
		items[i] = f.Make()
	}
	return items
}

// Create generates a single model instance and persists it into the database via orm.DB.
func (f *Factory[T]) Create(ctx context.Context, db *orm.DB) (*T, error) {
	if db == nil {
		return nil, errs.UsageError("factory: database is nil")
	}
	item := f.Make()
	if _, err := db.Model(item).WithContext(ctx).Insert(item); err != nil {
		return nil, errs.FailureError(fmt.Sprintf("factory: create failed: %v", err))
	}
	return item, nil
}

// CreateMany generates Count model instances and persists each into the database.
func (f *Factory[T]) CreateMany(ctx context.Context, db *orm.DB) ([]*T, error) {
	if db == nil {
		return nil, errs.UsageError("factory: database is nil")
	}
	items := f.MakeMany()
	for i, item := range items {
		if _, err := db.Model(item).WithContext(ctx).Insert(item); err != nil {
			return nil, errs.FailureError(fmt.Sprintf("factory: create item [%d] failed: %v", i, err))
		}
	}
	return items, nil
}
