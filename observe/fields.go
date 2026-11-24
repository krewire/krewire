package observe

import (
	"context"
	"log/slog"
	"net/url"
	"sort"
	"strings"

	klog "github.com/krewire/krewire/log"
)

// Standard log/telemetry field keys. Every Krewire workload emits these names
// and no synonyms, so one query works across app, service, gateway, worker,
// cli, and toolchain logs. trace_id and request_id are shared with pkg/log.
const (
	FieldService      = "service"
	FieldWorkload     = "workload"
	FieldResource     = "resource"
	FieldInstance     = "instance"
	FieldVersion      = "version"
	FieldEnvironment  = "environment"
	FieldTraceID      = klog.AttrTraceID
	FieldSpanID       = "span_id"
	FieldParentSpanID = "parent_span_id"
	FieldRequestID    = klog.AttrRequestID
	FieldSessionID    = "session_id"
	FieldTaskID       = "task_id"
	FieldAgentRunID   = "agent_run_id"
	FieldToolCallID   = "tool_call_id"
	FieldProcessID    = "process_id"
	FieldJobID        = "job_id"
	FieldQueue        = "queue"
	FieldOrigin       = "origin"
	FieldFailureClass = "failure_class"
)

// Identity is the stable service/resource identity stamped on every record.
// All fields are low-cardinality by contract: never put user input here.
type Identity struct {
	Service     string `json:"service,omitempty"`
	Workload    string `json:"workload,omitempty"`
	Resource    string `json:"resource,omitempty"`
	Instance    string `json:"instance,omitempty"`
	Version     string `json:"version,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// Attrs returns the non-empty identity fields as slog attributes.
func (id Identity) Attrs() []slog.Attr {
	var out []slog.Attr
	add := func(k, v string) {
		if v != "" {
			out = append(out, slog.String(k, v))
		}
	}
	add(FieldService, id.Service)
	add(FieldWorkload, id.Workload)
	add(FieldResource, id.Resource)
	add(FieldInstance, id.Instance)
	add(FieldVersion, id.Version)
	add(FieldEnvironment, id.Environment)
	return out
}

// Origin distinguishes who produced a telemetry event. AI workflow reports
// must keep agent narrative apart from measured evidence.
type Origin string

const (
	// OriginAgent marks events an agent reported about its own work.
	OriginAgent Origin = "agent"
	// OriginCommand marks measured outcomes of deterministic commands.
	OriginCommand Origin = "command"
	// OriginProvider marks responses from external model or cloud providers.
	OriginProvider Origin = "provider"
	// OriginRuntime marks telemetry measured by the workload runtime itself.
	OriginRuntime Origin = "runtime"
)

// FailureClass attributes a failure to the layer that owns it.
type FailureClass string

const (
	FailureModel     FailureClass = "model"     // model/agent error or refusal
	FailureToolchain FailureClass = "toolchain" // deterministic build/test/lint failure
	FailureWorkload  FailureClass = "workload"  // the running workload or its resources
	FailureProvider  FailureClass = "provider"  // external API, cloud, or model provider
)

// IDs are the correlation identifiers that tie one unit of work together
// across the toolchain, the agent engine, and workloads. All are opaque
// strings; none may carry personal data (use a session ID, never a user email).
type IDs struct {
	RequestID  string `json:"request_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
	AgentRunID string `json:"agent_run_id,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ProcessID  string `json:"process_id,omitempty"`
}

func (ids IDs) fields() [][2]string {
	return [][2]string{
		{FieldRequestID, ids.RequestID}, {FieldSessionID, ids.SessionID},
		{FieldTaskID, ids.TaskID}, {FieldAgentRunID, ids.AgentRunID},
		{FieldToolCallID, ids.ToolCallID}, {FieldProcessID, ids.ProcessID},
	}
}

func (ids *IDs) set(key, v string) {
	switch key {
	case FieldRequestID:
		ids.RequestID = v
	case FieldSessionID:
		ids.SessionID = v
	case FieldTaskID:
		ids.TaskID = v
	case FieldAgentRunID:
		ids.AgentRunID = v
	case FieldToolCallID:
		ids.ToolCallID = v
	case FieldProcessID:
		ids.ProcessID = v
	}
}

// merge overlays the non-empty fields of o onto ids.
func (ids IDs) merge(o IDs) IDs {
	for _, f := range o.fields() {
		if f[1] != "" {
			ids.set(f[0], f[1])
		}
	}
	return ids
}

type idsCtxKey struct{}

// WithIDs merges ids into the correlation IDs carried by ctx; empty fields
// keep their inherited value, so a tool call inherits its task and run.
func WithIDs(ctx context.Context, ids IDs) context.Context {
	merged := IDsFrom(ctx).merge(ids)
	ctx = context.WithValue(ctx, idsCtxKey{}, merged)
	return withCorrelation(ctx, SpanFromContext(ctx))
}

// IDsFrom returns the correlation IDs carried by ctx. A request ID stored only
// through pkg/log.WithCorrelation is reported too.
func IDsFrom(ctx context.Context) IDs {
	if ctx == nil {
		return IDs{}
	}
	ids, _ := ctx.Value(idsCtxKey{}).(IDs)
	if ids.RequestID == "" {
		ids.RequestID, _ = klog.CorrelationFrom(ctx)
	}
	return ids
}

// withCorrelation keeps pkg/log's correlation in step with the active span so
// existing log.WithContext callers see the same trace_id.
func withCorrelation(ctx context.Context, sc SpanContext) context.Context {
	reqID, traceID := klog.CorrelationFrom(ctx)
	if ids, ok := ctx.Value(idsCtxKey{}).(IDs); ok && ids.RequestID != "" {
		reqID = ids.RequestID
	}
	if sc.IsValid() {
		traceID = sc.TraceID.String()
	}
	return klog.WithCorrelation(ctx, reqID, traceID)
}

// ContextAttrs returns the trace and correlation attributes active in ctx.
func ContextAttrs(ctx context.Context) []slog.Attr {
	var out []slog.Attr
	if sc := SpanFromContext(ctx); sc.IsValid() {
		out = append(out, slog.String(FieldTraceID, sc.TraceID.String()), slog.String(FieldSpanID, sc.SpanID.String()))
		if sc.Parent.IsValid() {
			out = append(out, slog.String(FieldParentSpanID, sc.Parent.String()))
		}
	} else if _, traceID := klog.CorrelationFrom(ctx); traceID != "" {
		out = append(out, slog.String(FieldTraceID, traceID))
	}
	for _, f := range IDsFrom(ctx).fields() {
		if f[1] != "" {
			out = append(out, slog.String(f[0], f[1]))
		}
	}
	return out
}

// HeaderBaggage is the W3C Baggage header that carries correlation IDs.
const HeaderBaggage = "baggage"

// baggagePrefix namespaces Krewire keys inside W3C baggage.
const baggagePrefix = "krewire."

// HeaderRequestID mirrors pkg/web's request-ID header for cross-hop continuity.
const HeaderRequestID = "X-Request-ID"

func injectIDs(ctx context.Context, c Carrier) {
	ids := IDsFrom(ctx)
	var members []string
	for _, f := range ids.fields() {
		if f[1] != "" && f[0] != FieldRequestID {
			members = append(members, baggagePrefix+f[0]+"="+url.QueryEscape(f[1]))
		}
	}
	sort.Strings(members)
	if len(members) > 0 {
		c.Set(HeaderBaggage, strings.Join(members, ","))
	}
	if ids.RequestID != "" {
		c.Set(HeaderRequestID, ids.RequestID)
	}
}

func extractIDs(ctx context.Context, c Carrier) context.Context {
	var ids IDs
	for _, member := range strings.Split(c.Get(HeaderBaggage), ",") {
		kv, _, _ := strings.Cut(member, ";")
		k, v, ok := strings.Cut(strings.TrimSpace(kv), "=")
		if !ok {
			continue
		}
		key, ours := strings.CutPrefix(strings.TrimSpace(k), baggagePrefix)
		if !ours {
			continue
		}
		if dv, err := url.QueryUnescape(strings.TrimSpace(v)); err == nil {
			ids.set(key, dv)
		}
	}
	ids.RequestID = c.Get(HeaderRequestID)
	if ids == (IDs{}) {
		return ctx
	}
	return WithIDs(ctx, ids)
}
