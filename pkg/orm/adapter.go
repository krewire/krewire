package orm

import (
	"context"
)

// AdapterType categorizes the fundamental storage model of an adapter.
type AdapterType string

const (
	AdapterTypeSQL      AdapterType = "sql"
	AdapterTypeDocument AdapterType = "document"
	AdapterTypeSearch   AdapterType = "search"
)

// Adapter represents the common lifecycle interface implemented by all database drivers.
type Adapter interface {
	Name() string
	Type() AdapterType
	Ping(ctx context.Context) error
	Close() error
}

// Ensure *DB implements Adapter.
var _ Adapter = (*DB)(nil)

// Name returns the underlying SQL driver name.
func (db *DB) Name() string {
	if db.driver != "" {
		return db.driver
	}
	return "sql"
}

// Type returns AdapterTypeSQL.
func (db *DB) Type() AdapterType {
	return AdapterTypeSQL
}
