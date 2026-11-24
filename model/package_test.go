package model

import (
	"testing"
)

func TestStabilityTierValidity(t *testing.T) {
	validTiers := []StabilityTier{
		StabilityStable,
		StabilityBeta,
		StabilityExperimental,
		StabilityDeprecated,
		StabilityUnsupported,
	}

	for _, tier := range validTiers {
		if !tier.IsValid() {
			t.Errorf("expected tier %q to be valid", tier)
		}
	}

	if StabilityTier("bogus_tier").IsValid() {
		t.Error("expected bogus tier to be invalid")
	}
}
