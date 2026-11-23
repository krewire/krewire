package observe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

// W3C Trace Context header names (https://www.w3.org/TR/trace-context/).
const (
	HeaderTraceparent = "traceparent"
	HeaderTracestate  = "tracestate"
)

// traceVersion is the only traceparent version this package emits.
const traceVersion = "00"

// FlagSampled is the W3C trace-flags bit recording the sampling decision.
const FlagSampled byte = 0x01

// TraceID is a 16-byte W3C trace identifier. The zero value is invalid.
type TraceID [16]byte

// SpanID is an 8-byte W3C parent/span identifier. The zero value is invalid.
type SpanID [8]byte

// IsValid reports whether the trace ID is non-zero.
func (t TraceID) IsValid() bool { return t != TraceID{} }

// String returns the lowercase hex form, or "" for the zero ID.
func (t TraceID) String() string {
	if !t.IsValid() {
		return ""
	}
	return hex.EncodeToString(t[:])
}

// IsValid reports whether the span ID is non-zero.
func (s SpanID) IsValid() bool { return s != SpanID{} }

// String returns the lowercase hex form, or "" for the zero ID.
func (s SpanID) String() string {
	if !s.IsValid() {
		return ""
	}
	return hex.EncodeToString(s[:])
}

// SpanContext is the propagated part of a span: the identity a downstream
// hop needs to continue the trace. It is a value type and safe to copy.
type SpanContext struct {
	TraceID TraceID
	SpanID  SpanID
	// Parent is the span this one continues, zero for a root span.
	Parent SpanID
	Flags  byte
	// State is the opaque W3C tracestate value, forwarded unchanged.
	State string
}

// IsValid reports whether both identifiers are set.
func (sc SpanContext) IsValid() bool { return sc.TraceID.IsValid() && sc.SpanID.IsValid() }

// Sampled reports whether the sampled flag is set.
func (sc SpanContext) Sampled() bool { return sc.Flags&FlagSampled != 0 }

// Traceparent renders the W3C traceparent header value, or "" when invalid.
func (sc SpanContext) Traceparent() string {
	if !sc.IsValid() {
		return ""
	}
	return traceVersion + "-" + sc.TraceID.String() + "-" + sc.SpanID.String() + "-" + hex.EncodeToString([]byte{sc.Flags})
}

// ErrInvalidTraceparent reports a traceparent value that violates the W3C
// format. Callers should start a new root trace rather than fail the request.
var ErrInvalidTraceparent = errors.New("observe: invalid traceparent")

// ParseTraceparent parses a W3C traceparent value. Unknown future versions are
// accepted when their first four fields are well-formed, as the spec requires;
// version "ff" and all-zero identifiers are rejected.
func ParseTraceparent(v string) (SpanContext, error) {
	v = strings.TrimSpace(v)
	parts := strings.Split(v, "-")
	if len(parts) < 4 || len(parts[0]) != 2 || len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return SpanContext{}, ErrInvalidTraceparent
	}
	if parts[0] == "ff" || (parts[0] == traceVersion && len(parts) != 4) {
		return SpanContext{}, ErrInvalidTraceparent
	}
	for _, p := range parts[:4] {
		if strings.ToLower(p) != p {
			return SpanContext{}, ErrInvalidTraceparent
		}
	}
	var sc SpanContext
	var flags [1]byte
	if _, err := hex.Decode(sc.TraceID[:], []byte(parts[1])); err != nil {
		return SpanContext{}, ErrInvalidTraceparent
	}
	if _, err := hex.Decode(sc.SpanID[:], []byte(parts[2])); err != nil {
		return SpanContext{}, ErrInvalidTraceparent
	}
	if _, err := hex.Decode(flags[:], []byte(parts[3])); err != nil {
		return SpanContext{}, ErrInvalidTraceparent
	}
	if _, err := hex.DecodeString(parts[0]); err != nil {
		return SpanContext{}, ErrInvalidTraceparent
	}
	sc.Flags = flags[0]
	if !sc.IsValid() {
		return SpanContext{}, ErrInvalidTraceparent
	}
	return sc, nil
}

// NewRoot starts a new sampled trace with fresh random identifiers.
func NewRoot() SpanContext {
	var sc SpanContext
	fillRandom(sc.TraceID[:])
	fillRandom(sc.SpanID[:])
	sc.Flags = FlagSampled
	return sc
}

// Child returns a new span in the same trace whose parent is sc. An invalid sc
// yields a new root so callers never propagate an empty trace.
func (sc SpanContext) Child() SpanContext {
	if !sc.IsValid() {
		return NewRoot()
	}
	child := SpanContext{TraceID: sc.TraceID, Parent: sc.SpanID, Flags: sc.Flags, State: sc.State}
	fillRandom(child.SpanID[:])
	return child
}

func fillRandom(b []byte) {
	for {
		if _, err := rand.Read(b); err != nil {
			panic("observe: crypto/rand unavailable: " + err.Error())
		}
		for _, x := range b {
			if x != 0 {
				return
			}
		}
	}
}

type spanCtxKey struct{}

// ContextWithSpan returns ctx carrying sc as the active span.
func ContextWithSpan(ctx context.Context, sc SpanContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, spanCtxKey{}, sc)
}

// SpanFromContext returns the active span, or the zero SpanContext.
func SpanFromContext(ctx context.Context) SpanContext {
	if ctx == nil {
		return SpanContext{}
	}
	sc, _ := ctx.Value(spanCtxKey{}).(SpanContext)
	return sc
}

// StartSpan derives a child of the active span (or a new root) and returns
// the context carrying it. It is the single entry point for crossing an
// in-process boundary such as a job run, tool call, or child command.
func StartSpan(ctx context.Context) (context.Context, SpanContext) {
	sc := SpanFromContext(ctx).Child()
	return withCorrelation(ContextWithSpan(ctx, sc), sc), sc
}

// Carrier is the transport-neutral key/value surface trace context is written
// to and read from: HTTP headers, job metadata, environment variables.
type Carrier interface {
	Get(key string) string
	Set(key, value string)
}

// MapCarrier adapts a plain map, e.g. job metadata serialized with a payload.
type MapCarrier map[string]string

// Get returns the value for key.
func (m MapCarrier) Get(key string) string { return m[key] }

// Set stores value under key.
func (m MapCarrier) Set(key, value string) { m[key] = value }

// HeaderCarrier adapts http.Header.
type HeaderCarrier http.Header

// Get returns the first header value for key.
func (h HeaderCarrier) Get(key string) string { return http.Header(h).Get(key) }

// Set replaces the header value for key.
func (h HeaderCarrier) Set(key, value string) { http.Header(h).Set(key, value) }

// Inject writes the active span and correlation IDs from ctx into c. It is a
// no-op when ctx carries no valid span.
func Inject(ctx context.Context, c Carrier) {
	sc := SpanFromContext(ctx)
	if !sc.IsValid() || c == nil {
		return
	}
	c.Set(HeaderTraceparent, sc.Traceparent())
	if sc.State != "" {
		c.Set(HeaderTracestate, sc.State)
	}
	injectIDs(ctx, c)
}

// Extract reads trace context and correlation IDs from c into ctx. The
// extracted span is the remote parent; call StartSpan to continue it. ok is
// false when c holds no valid traceparent.
func Extract(ctx context.Context, c Carrier) (_ context.Context, ok bool) {
	if c == nil {
		return ctx, false
	}
	ctx = extractIDs(ctx, c)
	sc, err := ParseTraceparent(c.Get(HeaderTraceparent))
	if err != nil {
		return ctx, false
	}
	sc.State = c.Get(HeaderTracestate)
	return ContextWithSpan(ctx, sc), true
}
