package orm

import (
	"context"
	"time"
)

// Timestamps provides standard created_at and updated_at timestamp columns.
// Embedded in models to support automatic timestamp lifecycle management.
type Timestamps struct {
	CreatedAt time.Time `json:"created_at" orm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" orm:"column:updated_at"`
}

// SoftDeletes provides a deleted_at timestamp column for non-destructive deletes.
type SoftDeletes struct {
	DeletedAt *time.Time `json:"deleted_at,omitempty" orm:"column:deleted_at;soft_delete"`
}

// BaseModel provides a standard auto-incrementing 64-bit ID with Timestamps and SoftDeletes.
type BaseModel struct {
	ID int64 `json:"id" orm:"pk;auto"`
	Timestamps
	SoftDeletes
}

// Model Lifecycle Hook Interfaces (mirroring Laravel Eloquent Model Events)

// BeforeCreator is called immediately before a record is inserted into the database.
type BeforeCreator interface {
	BeforeCreate(ctx context.Context) error
}

// AfterCreator is called immediately after a record has been successfully inserted.
type AfterCreator interface {
	AfterCreate(ctx context.Context) error
}

// BeforeUpdater is called immediately before a record is updated in the database.
type BeforeUpdater interface {
	BeforeUpdate(ctx context.Context) error
}

// AfterUpdater is called immediately after a record has been successfully updated.
type AfterUpdater interface {
	AfterUpdate(ctx context.Context) error
}

// BeforeDeleter is called immediately before a record is deleted (soft or hard).
type BeforeDeleter interface {
	BeforeDelete(ctx context.Context) error
}

// AfterDeleter is called immediately after a record has been deleted.
type AfterDeleter interface {
	AfterDelete(ctx context.Context) error
}

// BeforeSaver is called before both Insert and Update operations.
type BeforeSaver interface {
	BeforeSave(ctx context.Context) error
}

// AfterSaver is called after both Insert and Update operations.
type AfterSaver interface {
	AfterSave(ctx context.Context) error
}
