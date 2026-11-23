package observe

import (
	"context"
	"log/slog"
)

// Handler is a slog.Handler decorator that enforces the log contract: every
// record carries the workload Identity, the trace/correlation fields active in
// its context, and has sensitive attributes redacted. It wraps any handler,
// so JSON/text formatting stays the job of pkg/log or the caller.
type Handler struct {
	inner slog.Handler
}

// NewHandler wraps inner, pre-binding the identity attributes once.
func NewHandler(inner slog.Handler, id Identity) *Handler {
	if inner == nil {
		inner = slog.Default().Handler()
	}
	if attrs := id.Attrs(); len(attrs) > 0 {
		inner = inner.WithAttrs(attrs)
	}
	return &Handler{inner: inner}
}

// NewLogger returns a logger that applies the log contract on top of base's
// handler. A nil base uses slog.Default().
func NewLogger(base *slog.Logger, id Identity) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	return slog.New(NewHandler(base.Handler(), id))
}

// Enabled defers to the wrapped handler.
func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool { return h.inner.Enabled(ctx, l) }

// Handle adds context correlation fields and redacts the record's attributes.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time, r.Level, RedactString(r.Message), r.PC)
	out.AddAttrs(ContextAttrs(ctx)...)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(RedactAttr(nil, a))
		return true
	})
	return h.inner.Handle(ctx, out)
}

// WithAttrs redacts pre-bound attributes before handing them on.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	red := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		red[i] = RedactAttr(nil, a)
	}
	return &Handler{inner: h.inner.WithAttrs(red)}
}

// WithGroup defers to the wrapped handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{inner: h.inner.WithGroup(name)}
}
