package orm

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// SetLogger attaches a structured slog.Logger for query execution telemetry.
func (db *DB) SetLogger(logger *slog.Logger) {
	db.logger = logger
}

// SetSlowQueryThreshold sets the latency threshold above which queries are logged as warnings.
func (db *DB) SetSlowQueryThreshold(d time.Duration) {
	db.slowQueryThreshold = d
}

func (db *DB) logQuery(ctx context.Context, sqlStr string, args []interface{}, start time.Time, err error, rowsAffected int64) {
	if db == nil || db.logger == nil {
		return
	}
	duration := time.Since(start)
	attrs := []any{
		"query", sqlStr,
		"args_count", len(args),
		"duration_ms", float64(duration.Microseconds()) / 1000.0,
	}
	if rowsAffected >= 0 {
		attrs = append(attrs, "rows_affected", rowsAffected)
	}
	if err != nil && err != sql.ErrNoRows {
		attrs = append(attrs, "error", err.Error())
		db.logger.ErrorContext(ctx, "orm: query failed", attrs...)
		return
	}

	if db.slowQueryThreshold > 0 && duration >= db.slowQueryThreshold {
		attrs = append(attrs, "slow_query", true)
		db.logger.WarnContext(ctx, "orm: slow query detected", attrs...)
		return
	}

	db.logger.DebugContext(ctx, "orm: query executed", attrs...)
}
