package model

import (
	"testing"
	"time"
)

func TestCanonicalWorkloadOperationalBaselines(t *testing.T) {
	baselines := CanonicalWorkloadOperationalBaselines()

	if len(baselines) != len(AllKinds) {
		t.Fatalf("expected %d operational baselines, got %d", len(AllKinds), len(baselines))
	}

	for _, kind := range AllKinds {
		b, ok := baselines[kind]
		if !ok {
			t.Errorf("missing operational baseline for kind: %s", kind)
			continue
		}

		if b.Kind != kind {
			t.Errorf("[%s] kind mismatch, got: %s", kind, b.Kind)
		}
		if b.LivenessProbe == "" {
			t.Errorf("[%s] missing liveness probe definition", kind)
		}
		if b.ReadinessProbe == "" {
			t.Errorf("[%s] missing readiness probe definition", kind)
		}
		if b.LoggingFormat == "" {
			t.Errorf("[%s] missing logging format definition", kind)
		}
		if b.DeploymentTarget == "" {
			t.Errorf("[%s] missing deployment target definition", kind)
		}
		if b.RollbackSupport == "" {
			t.Errorf("[%s] missing rollback support definition", kind)
		}
		if b.BackupRequirement == "" {
			t.Errorf("[%s] missing backup requirement definition", kind)
		}

		// Long-running workloads must support tested graceful shutdown
		switch kind {
		case KindApp, KindService, KindGateway, KindWorker:
			if !b.GracefulShutdown {
				t.Errorf("[%s] long-running workload must declare GracefulShutdown=true", kind)
			}
			if b.ShutdownTimeout <= 0 {
				t.Errorf("[%s] long-running workload must have ShutdownTimeout > 0", kind)
			}
			if !b.CorrelationSupported {
				t.Errorf("[%s] long-running service workload must support correlation IDs", kind)
			}
		}
	}
}

func TestGetWorkloadOperationalBaseline(t *testing.T) {
	b, ok := GetWorkloadOperationalBaseline(KindApp)
	if !ok || b.Kind != KindApp {
		t.Fatalf("GetWorkloadOperationalBaseline(KindApp) failed: ok=%v, b=%+v", ok, b)
	}
	if b.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected 10s shutdown timeout, got %v", b.ShutdownTimeout)
	}

	_, ok = GetWorkloadOperationalBaseline("non_existent_kind")
	if ok {
		t.Fatalf("expected false for unknown workload kind, got true")
	}
}
