package observe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func probe(t *testing.T, mux *http.ServeMux, path string) (int, Report) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	var rep Report
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return rec.Code, rep
}

func TestHealthLifecycle(t *testing.T) {
	h := NewHealth()
	reg := NewRegistry()
	h.SetMetrics(reg)
	dbDown := true
	h.Add(
		Check{Name: "db", Probes: []Probe{ProbeReadiness, ProbeStartup}, Func: func(context.Context) error {
			if dbDown {
				return errors.New("dial postgres://app:s3cret@db/x: refused")
			}
			return nil
		}},
		Check{Name: "cache", Optional: true, Func: func(context.Context) error { return errors.New("miss") }},
		Check{Name: "slow", Timeout: 10 * time.Millisecond, Func: func(ctx context.Context) error {
			<-ctx.Done()
			time.Sleep(5 * time.Millisecond)
			return nil
		}, Optional: true},
		Check{Name: "boom", Optional: true, Func: func(context.Context) error { panic("x") }},
		Check{Name: "nofunc"},
	)
	mux := http.NewServeMux()
	h.Mount(mux, DefaultPaths)

	code, rep := probe(t, mux, PathLiveness)
	if code != 200 || rep.Status != StatusPass || len(rep.Checks) != 0 {
		t.Fatalf("liveness must ignore dependencies: %d %+v", code, rep)
	}
	code, rep = probe(t, mux, PathStartup)
	if code != 503 || rep.Reason != "starting" {
		t.Fatalf("startup before MarkStarted: %d %+v", code, rep)
	}
	h.MarkStarted()
	code, rep = probe(t, mux, PathReadiness)
	if code != 503 || rep.Status != StatusFail || strings.Contains(rep.Checks[0].Error, "s3cret") {
		t.Fatalf("readiness with db down: %d %+v", code, rep)
	}
	if rep.Checks[2].Status != StatusWarn || !strings.Contains(rep.Checks[2].Error, "timed out") || !strings.Contains(rep.Checks[3].Error, "panicked") {
		t.Fatalf("optional checks: %+v", rep.Checks)
	}
	dbDown = false
	code, rep = probe(t, mux, PathReadiness)
	if code != 200 || rep.Status != StatusWarn {
		t.Fatalf("degraded but serving: %d %+v", code, rep)
	}
	h.MarkDraining()
	if code, rep = probe(t, mux, PathReadiness); code != 503 || rep.Reason != "draining" {
		t.Fatalf("draining: %d %+v", code, rep)
	}
	if code, _ = probe(t, mux, PathLiveness); code != 200 {
		t.Fatal("draining must not fail liveness")
	}
	var b strings.Builder
	_ = reg.WritePrometheus(&b)
	if !strings.Contains(b.String(), `krewire_health_check_status{probe="readiness",check="db"} 1`) ||
		!strings.Contains(b.String(), `krewire_health_check_status{probe="readiness",check="cache"} 0.5`) {
		t.Fatalf("health metrics:\n%s", b.String())
	}
}

func TestHeartbeatAndNil(t *testing.T) {
	h := NewHealth()
	beat := h.Heartbeat("loop", 20*time.Millisecond)
	if h.Evaluate(context.Background(), ProbeLiveness).Status != StatusPass {
		t.Fatal("fresh heartbeat")
	}
	time.Sleep(30 * time.Millisecond)
	if rep := h.Evaluate(context.Background(), ProbeLiveness); rep.Status != StatusFail || !strings.Contains(rep.Checks[0].Error, "no heartbeat") {
		t.Fatalf("stale heartbeat %+v", rep)
	}
	beat()
	if h.Evaluate(context.Background(), ProbeLiveness).Status != StatusPass {
		t.Fatal("beat must restore liveness")
	}

	var nilH *Health
	nilH.Add(Check{})
	nilH.MarkStarted()
	nilH.MarkDraining()
	nilH.SetMetrics(nil)
	nilH.Heartbeat("x", time.Second)()
	mux := http.NewServeMux()
	nilH.Mount(mux, Paths{Readiness: "/ready"})
	if code, rep := probe(t, mux, "/ready"); code != 200 || rep.Status != StatusPass {
		t.Fatal("nil health must pass")
	}
}
