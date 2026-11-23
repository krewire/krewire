package observe

import (
	"context"
	"net/http"
	"os"
	"sort"
	"strings"
)

// TraceHTTP is server middleware that continues the caller's trace (or starts
// a root), activates a server span, adopts the caller's correlation IDs, and
// echoes traceparent and X-Request-ID on the response. A missing request ID is
// filled from the trace ID so every request is addressable in logs.
func TraceHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := Extract(r.Context(), HeaderCarrier(r.Header))
		ctx, sc := StartSpan(ctx)
		if IDsFrom(ctx).RequestID == "" {
			ctx = WithIDs(ctx, IDs{RequestID: sc.TraceID.String()})
		}
		w.Header().Set(HeaderTraceparent, sc.Traceparent())
		w.Header().Set(HeaderRequestID, IDsFrom(ctx).RequestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Transport is an http.RoundTripper that starts a client span per outgoing
// request and injects it, so gateways and service-to-service clients continue
// the trace. A nil Base uses http.DefaultTransport.
type Transport struct {
	Base http.RoundTripper
}

// RoundTrip injects trace context into a clone of req.
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	ctx, _ := StartSpan(req.Context())
	out := req.Clone(ctx)
	Inject(ctx, HeaderCarrier(out.Header))
	return base.RoundTrip(out)
}

// Runner is the structural shape of a background job (worker.Job satisfies
// it), letting observe wrap jobs without importing the worker package.
type Runner interface {
	Run(ctx context.Context) error
}

// TracedJob carries the trace context captured at enqueue time. Carrier is
// exported so serializing queues can persist it beside the payload.
type TracedJob struct {
	Job     Runner
	Carrier MapCarrier
}

// WrapJob captures the active trace and correlation IDs from the enqueuing
// ctx. The returned job satisfies worker.Job.
func WrapJob(ctx context.Context, job Runner) *TracedJob {
	c := MapCarrier{}
	Inject(ctx, c)
	return &TracedJob{Job: job, Carrier: c}
}

// Run restores the enqueuer's trace as the remote parent, starts a consumer
// span, and runs the wrapped job under it. The dequeuing ctx keeps its
// cancellation; only trace values are taken from the carrier.
func (j *TracedJob) Run(ctx context.Context) error {
	ctx, _ = Extract(ctx, j.Carrier)
	ctx, _ = StartSpan(ctx)
	return j.Job.Run(ctx)
}

// EnvCarrier adapts a process environment for child-process propagation:
// key "traceparent" maps to TRACEPARENT, "X-Request-ID" to X_REQUEST_ID.
type EnvCarrier map[string]string

func envKey(k string) string { return strings.ToUpper(strings.ReplaceAll(k, "-", "_")) }

// Get returns the environment value for key.
func (e EnvCarrier) Get(key string) string { return e[envKey(key)] }

// Set stores value under the environment form of key.
func (e EnvCarrier) Set(key, value string) { e[envKey(key)] = value }

// Environ renders the carrier as KEY=VALUE pairs for exec.Cmd.Env.
func (e EnvCarrier) Environ() []string {
	out := make([]string, 0, len(e))
	for k, v := range e {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

// ChildEnv returns the variables a child process needs to continue the trace
// of ctx; append them to cmd.Env.
func ChildEnv(ctx context.Context) []string {
	c := EnvCarrier{}
	ctx, _ = StartSpan(ctx)
	Inject(ctx, c)
	return c.Environ()
}

// FromEnviron continues a trace handed down by a parent process through
// os.Environ-style pairs, activating a child span for the process. It returns
// ctx without a span when none was handed down.
func FromEnviron(ctx context.Context, environ []string) context.Context {
	c := EnvCarrier{}
	for _, kv := range environ {
		if k, v, ok := strings.Cut(kv, "="); ok {
			c[k] = v
		}
	}
	ctx, ok := Extract(ctx, c)
	if ok {
		ctx, _ = StartSpan(ctx)
	}
	return ctx
}

// FromProcessEnv is FromEnviron(ctx, os.Environ()).
func FromProcessEnv(ctx context.Context) context.Context {
	return FromEnviron(ctx, os.Environ())
}
