package model

import "testing"

func TestGetSecurityBaseline(t *testing.T) {
	baseline := GetSecurityBaseline()

	if baseline.ThreatModel == "" {
		t.Error("baseline must have a non-empty ThreatModel")
	}

	if len(baseline.Controls) == 0 {
		t.Error("baseline must define at least one SecurityControl")
	}

	if len(baseline.ResidualRisks) == 0 {
		t.Error("baseline must explicitly document residual risks")
	}

	seenIDs := make(map[string]bool)
	for _, ctrl := range baseline.Controls {
		if ctrl.ID == "" {
			t.Error("control has empty ID")
		}
		if seenIDs[ctrl.ID] {
			t.Errorf("duplicate control ID %q", ctrl.ID)
		}
		seenIDs[ctrl.ID] = true
	}
}
