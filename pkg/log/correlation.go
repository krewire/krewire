package log

import (
	"context"
	"log/slog"
)

// Standard structured log attribute keys for request and distributed trace correlation.
const (
	AttrRequestID = "request_id"
	AttrTraceID   = "trace_id"
)

type correlationCtxKey struct{}

type correlationData struct {
	requestID string
	traceID   string
}

// WithCorrelation injects requestID and traceID into context for structured telemetry correlation.
func WithCorrelation(ctx context.Context, requestID, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, correlationCtxKey{}, correlationData{
		requestID: requestID,
		traceID:   traceID,
	})
}

// CorrelationFrom extracts requestID and traceID from context, returning empty strings if absent.
func CorrelationFrom(ctx context.Context) (requestID, traceID string) {
	if ctx == nil {
		return "", ""
	}
	data, ok := ctx.Value(correlationCtxKey{}).(correlationData)
	if !ok {
		return "", ""
	}
	return data.requestID, data.traceID
}

// CorrelationAttrs returns slog.Attr slices for active correlation IDs in ctx.
func CorrelationAttrs(ctx context.Context) []slog.Attr {
	reqID, trcID := CorrelationFrom(ctx)
	var attrs []slog.Attr
	if reqID != "" {
		attrs = append(attrs, slog.String(AttrRequestID, reqID))
	}
	if trcID != "" {
		attrs = append(attrs, slog.String(AttrTraceID, trcID))
	}
	return attrs
}

// WithContext decorates logger with request_id and trace_id attributes extracted from ctx.
// If logger is nil, slog.Default() is used.
func WithContext(l *slog.Logger, ctx context.Context) *slog.Logger {
	if l == nil {
		l = slog.Default()
	}
	attrs := CorrelationAttrs(ctx)
	if len(attrs) == 0 {
		return l
	}
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return l.With(args...)
}
