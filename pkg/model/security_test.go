package model

import (
	"testing"
)

func TestCanonicalWorkloadSecurityBaselines(t *testing.T) {
	baselines := CanonicalWorkloadSecurityBaselines()

	expectedKinds := AllKinds

	if len(baselines) != len(expectedKinds) {
		t.Fatalf("expected %d baselines, got %d", len(expectedKinds), len(baselines))
	}

	for _, kind := range expectedKinds {
		b, ok := baselines[kind]
		if !ok {
			t.Errorf("missing security baseline for workload kind %q", kind)
			continue
		}

		if b.Kind != kind {
			t.Errorf("baseline kind mismatch: expected %q, got %q", kind, b.Kind)
		}

		if b.ThreatModel == "" {
			t.Errorf("baseline for %q must have a non-empty ThreatModel", kind)
		}

		if len(b.Controls) == 0 {
			t.Errorf("baseline for %q must define at least one SecurityControl", kind)
		}

		if len(b.ResidualRisks) == 0 {
			t.Errorf("baseline for %q must explicitly document residual risks", kind)
		}

		// Verify individual controls integrity
		seenIDs := make(map[string]bool)
		var hasVerified, hasConfigDependentOrUnsupported bool

		for _, ctrl := range b.Controls {
			if ctrl.ID == "" {
				t.Errorf("[%s] control has empty ID", kind)
			}
			if seenIDs[ctrl.ID] {
				t.Errorf("[%s] duplicate control ID %q", kind, ctrl.ID)
			}
			seenIDs[ctrl.ID] = true

			if ctrl.Name == "" || ctrl.Description == "" {
				t.Errorf("[%s] control %s has empty name or description", kind, ctrl.ID)
			}

			if ctrl.FailureBehavior == "" {
				t.Errorf("[%s] control %s has empty failure behavior", kind, ctrl.ID)
			}

			switch ctrl.Status {
			case ControlVerified:
				hasVerified = true
			case ControlConfigDependent, ControlUnsupported:
				hasConfigDependentOrUnsupported = true
			default:
				t.Errorf("[%s] control %s has invalid status %q", kind, ctrl.ID, ctrl.Status)
			}
		}

		if !hasVerified {
			t.Errorf("[%s] baseline must have at least one VERIFIED control", kind)
		}
		if !hasConfigDependentOrUnsupported {
			t.Errorf("[%s] baseline must distinguish non-verified controls", kind)
		}
	}
}

func TestGetWorkloadSecurityBaseline(t *testing.T) {
	b, ok := GetWorkloadSecurityBaseline(KindApp)
	if !ok || b.Kind != KindApp {
		t.Fatalf("GetWorkloadSecurityBaseline(KindApp) failed: ok=%v, b=%+v", ok, b)
	}

	_, ok = GetWorkloadSecurityBaseline("non_existent_kind")
	if ok {
		t.Fatalf("expected false for unknown workload kind, got true")
	}
}
