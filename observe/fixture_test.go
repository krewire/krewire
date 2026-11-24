package observe_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/krewire/krewire/cloud/worker"
	"github.com/krewire/krewire/observe"
)

// logSink collects JSON log lines from several workloads in one buffer.
type logSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *logSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *logSink) records(t *testing.T) []map[string]any {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(s.buf.String()), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("bad log line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

type jobFunc func(ctx context.Context) error

func (f jobFunc) Run(ctx context.Context) error { return f(ctx) }

// TestMultiWorkloadTraceContinuity is the integration fixture for #139: a
// toolchain task drives a gateway, which proxies to a service, which enqueues
// a job on the real pkg/cloud/worker queue, which spawns a child process. One trace ID and one task ID
// must connect every log record, and each hop must be a child span.
func TestMultiWorkloadTraceContinuity(t *testing.T) {
	sink := &logSink{}
	logger := func(workload, resource string) *slog.Logger {
		return observe.NewLogger(slog.New(slog.NewJSONHandler(sink, nil)),
			observe.Identity{Service: "shop", Workload: workload, Resource: resource})
	}
	queue := worker.NewInMemoryQueue()

	// service: logs, then enqueues a job carrying its trace.
	svcLog := logger("service", "orders-api")
	service := httptest.NewServer(observe.TraceHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svcLog.InfoContext(r.Context(), "order accepted", "authorization", "Bearer leak")
		job := observe.WrapJob(r.Context(), jobFunc(func(ctx context.Context) error {
			logger("worker", "orders-queue").InfoContext(ctx, "job run")
			childCtx := observe.FromEnviron(context.Background(), observe.ChildEnv(ctx))
			logger("cli", "migrate").InfoContext(childCtx, "child process")
			return nil
		}))
		if _, err := queue.Enqueue(r.Context(), job, worker.Options{}); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusAccepted)
	})))
	defer service.Close()

	// gateway: reverse proxy whose outbound transport continues the trace.
	target, _ := url.Parse(service.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = observe.Transport{}
	gwLog := logger("gateway", "edge")
	gateway := httptest.NewServer(observe.TraceHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gwLog.InfoContext(r.Context(), "proxy")
		proxy.ServeHTTP(w, r)
	})))
	defer gateway.Close()

	// toolchain: a task + agent tool call issues the request.
	ctx, root := observe.StartSpan(observe.WithIDs(context.Background(),
		observe.IDs{SessionID: "sess-1", TaskID: "task-42", AgentRunID: "run-7", ToolCallID: "call-3"}))
	logger("cli", "krew").InfoContext(ctx, "tool call", "origin", string(observe.OriginCommand))
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, gateway.URL+"/orders", nil)
	resp, err := (&http.Client{Transport: observe.Transport{}}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted || resp.Header.Get(observe.HeaderTraceparent) == "" {
		t.Fatalf("status %d headers %v", resp.StatusCode, resp.Header)
	}

	// worker: dequeue on an unrelated context, as a real consumer would.
	task, err := queue.Dequeue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	_ = queue.Ack(context.Background(), task)

	recs := sink.records(t)
	if len(recs) != 5 {
		t.Fatalf("want 5 records, got %d: %v", len(recs), recs)
	}
	spans := map[string]bool{}
	var requestID string
	for _, rec := range recs {
		if rec[observe.FieldTraceID] != root.TraceID.String() {
			t.Errorf("%s/%s: trace_id %v broke continuity", rec["workload"], rec["msg"], rec[observe.FieldTraceID])
		}
		if rec[observe.FieldTaskID] != "task-42" || rec[observe.FieldAgentRunID] != "run-7" || rec[observe.FieldSessionID] != "sess-1" {
			t.Errorf("%s: correlation IDs lost: %v", rec["msg"], rec)
		}
		if rec[observe.FieldService] != "shop" || rec[observe.FieldWorkload] == nil || rec[observe.FieldResource] == nil {
			t.Errorf("%s: identity missing: %v", rec["msg"], rec)
		}
		span, _ := rec[observe.FieldSpanID].(string)
		if spans[span] {
			t.Errorf("span %s reused across hops", span)
		}
		spans[span] = true
		if rec["msg"] != "tool call" {
			if rec[observe.FieldParentSpanID] == nil {
				t.Errorf("%s: hop is not a child span", rec["msg"])
			}
			if id, _ := rec[observe.FieldRequestID].(string); requestID == "" {
				requestID = id
			} else if id != requestID {
				t.Errorf("%s: request_id %q != %q", rec["msg"], id, requestID)
			}
		}
	}
	if strings.Contains(sink.buf.String(), "leak") {
		t.Error("credential leaked into logs")
	}
}

func TestEnvCarrierAndProcessEnv(t *testing.T) {
	ctx, sc := observe.StartSpan(context.Background())
	env := observe.ChildEnv(ctx)
	if !strings.Contains(strings.Join(env, " "), "TRACEPARENT=00-"+sc.TraceID.String()) {
		t.Fatalf("env %v", env)
	}
	t.Setenv("TRACEPARENT", sc.Traceparent())
	if got := observe.SpanFromContext(observe.FromProcessEnv(context.Background())); got.TraceID != sc.TraceID {
		t.Fatalf("FromProcessEnv %+v", got)
	}
	if observe.SpanFromContext(observe.FromEnviron(context.Background(), []string{"NOEQUALS"})).IsValid() {
		t.Fatal("garbage env must not yield a span")
	}
}
