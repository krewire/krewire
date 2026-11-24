package orm

import (
	"context"
	"database/sql"
	"time"

	"github.com/krewire/krewire/errs"
)

// Stats returns the connection pool statistics of the underlying database.
func (db *DB) Stats() sql.DBStats {
	if db.sqlDB == nil {
		return sql.DBStats{}
	}
	return db.sqlDB.Stats()
}

// Health checks the health and connectivity of the database connection within a timeout.
func (db *DB) Health(ctx context.Context) error {
	if db.sqlDB == nil {
		return errs.FailureError("orm: database connection is nil")
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return db.sqlDB.PingContext(timeoutCtx)
}
