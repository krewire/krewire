package model_test

import (
	"testing"

	"github.com/krewire/krewire/packages/kern/model"
)

func TestTier_Ladder(t *testing.T) {
	tiers := model.AllTiers
	if len(tiers) != 4 {
		t.Fatalf("expected 4 tiers, got %d", len(tiers))
	}
	expected := []model.Tier{
		model.TierFree,
		model.TierPro,
		model.TierTeam,
		model.TierEnterprise,
	}
	for i, exp := range expected {
		if tiers[i] != exp {
			t.Errorf("tiers[%d] = %s, want %s", i, tiers[i], exp)
		}
		if !exp.IsValid() {
			t.Errorf("tier %s should be valid", exp)
		}
		if exp.Rank() != i {
			t.Errorf("tier %s rank = %d, want %d", exp, exp.Rank(), i)
		}
	}
}

func TestTier_AtLeast(t *testing.T) {
	tests := []struct {
		current  model.Tier
		required model.Tier
		want     bool
	}{
		{model.TierFree, model.TierFree, true},
		{model.TierFree, model.TierPro, false},
		{model.TierPro, model.TierFree, true},
		{model.TierPro, model.TierPro, true},
		{model.TierPro, model.TierTeam, false},
		{model.TierTeam, model.TierPro, true},
		{model.TierEnterprise, model.TierTeam, true},
		{model.TierEnterprise, model.TierEnterprise, true},
		{model.Tier("invalid"), model.TierFree, false},
		{model.TierFree, model.Tier("invalid"), false},
	}

	for _, tt := range tests {
		got := tt.current.AtLeast(tt.required)
		if got != tt.want {
			t.Errorf("%s.AtLeast(%s) = %v, want %v", tt.current, tt.required, got, tt.want)
		}
	}
}

func TestTier_ParseTier(t *testing.T) {
	tests := []struct {
		input   string
		want    model.Tier
		wantErr bool
	}{
		{"", model.TierFree, false},
		{"free", model.TierFree, false},
		{"FREE", model.TierFree, false},
		{"pro", model.TierPro, false},
		{"Pro", model.TierPro, false},
		{"team", model.TierTeam, false},
		{"enterprise", model.TierEnterprise, false},
		{"ENTERPRISE", model.TierEnterprise, false},
		{"unknown", "", true},
		{"community", "", true},
	}

	for _, tt := range tests {
		got, err := model.ParseTier(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTier(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseTier(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
