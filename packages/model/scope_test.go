package model

import (
	"testing"

	"github.com/krewire/krewire/packages/errs"
)

func TestParseScope_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want Scope
	}{
		{"Workspace", ScopeWorkspace},
		{"Module", ScopeModule},
		{"Domain", ScopeDomain},
		{"Service", ScopeService},
		{"Unit", ScopeUnit},
		{"workspace", ScopeWorkspace},
		{"DOMAIN", ScopeDomain},
		{"  Unit  ", ScopeUnit},
		{"SERVICE", ScopeService},
	}
	for _, c := range cases {
		got, err := ParseScope(c.in)
		if err != nil {
			t.Errorf("ParseScope(%q) error = %v, want nil", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseScope(%q) = %v, want %v", c.in, got, c.want)
		}
		if !got.IsValid() {
			t.Errorf("Scope(%q).IsValid() = false, want true", c.in)
		}
	}
}

func TestParseScope_Invalid(t *testing.T) {
	cases := []string{"", "unknown", "app", "WS", "Project", "Package", "Func", "ProjectX"}
	for _, in := range cases {
		if _, err := ParseScope(in); err == nil {
			t.Errorf("ParseScope(%q) succeeded, want error", in)
		} else {
			if e, ok := err.(*errs.Error); !ok || e.Code != errs.ExitCodeUsage {
				t.Errorf("ParseScope(%q) error = %v, want UsageError (errs.ExitCodeUsage)", in, err)
			}
		}
	}
}

func TestScope_Ordering(t *testing.T) {
	// Ordering: Workspace < Module < Domain < Service < Unit
	ordered := []Scope{ScopeWorkspace, ScopeModule, ScopeDomain, ScopeService, ScopeUnit}
	for i := 0; i < len(ordered)-1; i++ {
		if !ordered[i].Less(ordered[i+1]) {
			t.Errorf("%v.Less(%v) = false, want true", ordered[i], ordered[i+1])
		}
		if ordered[i].Level() >= ordered[i+1].Level() {
			t.Errorf("Level(%v)=%d should be < Level(%v)=%d", ordered[i], ordered[i].Level(), ordered[i+1], ordered[i+1].Level())
		}
	}
	if ScopeUnit.Less(ScopeWorkspace) {
		t.Error("Unit.Less(Workspace) = true, want false")
	}
}

func TestScope_Level_Invalid(t *testing.T) {
	if got := Scope("Unknown").Level(); got != -1 {
		t.Errorf("Scope(Unknown).Level() = %d, want -1", got)
	}
	if Scope("Unknown").IsValid() {
		t.Error("Scope(Unknown).IsValid() = true, want false")
	}
}

func TestScope_Containment(t *testing.T) {
	if len(AllScopes) != 5 {
		t.Fatalf("len(AllScopes) = %d, want 5", len(AllScopes))
	}
	for _, s := range AllScopes {
		if !s.IsValid() {
			t.Errorf("AllScopes contains invalid scope %q", s)
		}
	}
}

func TestScope_Identity(t *testing.T) {
	for _, s := range AllScopes {
		got, err := ParseScope(string(s))
		if err != nil || got != s {
			t.Errorf("ParseScope(%q) = %v, %v; want %v, nil", string(s), got, err, s)
		}
	}
}
