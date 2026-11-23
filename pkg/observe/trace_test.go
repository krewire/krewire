package observe

import (
	"context"
	"net/http"
	"strings"
	"testing"

	klog "github.com/krewire/krewire/pkg/log"
)

const validTP = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func TestParseTraceparent(t *testing.T) {
	sc, err := ParseTraceparent(validTP)
	if err != nil {
		t.Fatal(err)
	}
	if sc.TraceID.String() != "4bf92f3577b34da6a3ce929d0e0e4736" || sc.SpanID.String() != "00f067aa0ba902b7" || !sc.Sampled() {
		t.Fatalf("parsed %+v", sc)
	}
	if sc.Traceparent() != validTP {
		t.Fatalf("round trip %q", sc.Traceparent())
	}
	if _, err := ParseTraceparent("01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-future"); err != nil {
		t.Fatalf("future version must parse: %v", err)
	}
	bad := []string{
		"", "garbage",
		"ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra",
		"00-zzf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902zz-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-zz",
		"zz-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}
	for _, v := range bad {
		if _, err := ParseTraceparent(v); err == nil {
			t.Errorf("%q: expected error", v)
		}
	}
	if (SpanContext{}).Traceparent() != "" || (TraceID{}).String() != "" || (SpanID{}).String() != "" {
		t.Fatal("zero values must render empty")
	}
}

func TestChildAndStartSpan(t *testing.T) {
	root := NewRoot()
	child := root.Child()
	if child.TraceID != root.TraceID || child.Parent != root.SpanID || child.SpanID == root.SpanID {
		t.Fatalf("child %+v of %+v", child, root)
	}
	if !(SpanContext{}).Child().IsValid() {
		t.Fatal("child of invalid must be a new root")
	}
	ctx, sc := StartSpan(ContextWithSpan(context.Background(), root))
	if SpanFromContext(ctx) != sc || sc.Parent != root.SpanID {
		t.Fatal("StartSpan must activate a child")
	}
	if _, trace := klog.CorrelationFrom(ctx); trace != root.TraceID.String() {
		t.Fatalf("pkg/log correlation trace_id = %q", trace)
	}
	if SpanFromContext(nil).IsValid() { //nolint:staticcheck // nil ctx is tolerated
		t.Fatal("nil ctx")
	}
}

func TestInjectExtract(t *testing.T) {
	ctx := ContextWithSpan(context.Background(), SpanContext{TraceID: NewRoot().TraceID, SpanID: NewRoot().SpanID, Flags: 1, State: "vendor=x"})
	ctx = WithIDs(ctx, IDs{RequestID: "r1", TaskID: "task 7", AgentRunID: "run-1"})
	ctx = WithIDs(ctx, IDs{ToolCallID: "tool-3"})
	h := http.Header{}
	Inject(ctx, HeaderCarrier(h))
	if h.Get(HeaderTraceparent) == "" || h.Get(HeaderTracestate) != "vendor=x" || h.Get(HeaderRequestID) != "r1" {
		t.Fatalf("headers %v", h)
	}
	if !strings.Contains(h.Get(HeaderBaggage), "krewire.task_id=task+7") {
		t.Fatalf("baggage %q", h.Get(HeaderBaggage))
	}
	h.Set(HeaderBaggage, h.Get(HeaderBaggage)+",other=1;prop,broken")
	got, ok := Extract(context.Background(), HeaderCarrier(h))
	if !ok || SpanFromContext(got).TraceID != SpanFromContext(ctx).TraceID || SpanFromContext(got).State != "vendor=x" {
		t.Fatal("extract lost the span")
	}
	ids := IDsFrom(got)
	if ids != (IDs{RequestID: "r1", TaskID: "task 7", AgentRunID: "run-1", ToolCallID: "tool-3"}) {
		t.Fatalf("ids %+v", ids)
	}
	if _, ok := Extract(context.Background(), MapCarrier{}); ok {
		t.Fatal("empty carrier must not extract")
	}
	if _, ok := Extract(context.Background(), nil); ok {
		t.Fatal("nil carrier")
	}
	m := MapCarrier{}
	Inject(context.Background(), m)
	Inject(ctx, nil)
	if len(m) != 0 {
		t.Fatal("inject without span must be a no-op")
	}
}
