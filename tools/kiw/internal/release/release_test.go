package release

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/packages/kern"
)

func TestBump(t *testing.T) {
	v := kern.MustParseVersion("0.3.1")
	if got := Bump(v, BumpPatch); got.String() != "0.3.2" {
		t.Errorf("Bump patch = %s, want 0.3.2", got)
	}
	if got := Bump(v, BumpMinor); got.String() != "0.4.0" {
		t.Errorf("Bump minor = %s, want 0.4.0", got)
	}
	if got := Bump(v, BumpMajor); got.String() != "1.0.0" {
		t.Errorf("Bump major = %s, want 1.0.0", got)
	}
}

func TestPlanReleasingKrewirePropagatesToMdbind(t *testing.T) {
	edits, err := Plan([]string{string(ModuleKrewire)}, BumpPatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) == 0 {
		t.Fatal("expected edits when planning krewire release")
	}
}

func TestPlanAllModules(t *testing.T) {
	var all []string
	for _, m := range Modules {
		all = append(all, m.Name)
	}
	edits, err := Plan(all, BumpPatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) < len(Modules) {
		t.Fatalf("got %d edits, want at least %d", len(edits), len(Modules))
	}
}

func TestApplyWritesAndValidates(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v.go")
	orig := `var Version = kern.MustParseVersion("0.1.0")`
	if err := os.WriteFile(p, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}

	edits := []Edit{{
		Module:  "krewire",
		File:    p,
		Summary: "bump",
		From:    orig,
		To:      `var Version = kern.MustParseVersion("0.1.1")`,
	}}
	if _, err := Apply(edits, "", false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `var Version = kern.MustParseVersion("0.1.1")` {
		t.Errorf("got %q, want updated", string(got))
	}
}

func TestCheckTierCompliance(t *testing.T) {
	// Verify current ecosystem modules comply with TierFree and valid open-source licensing
	if err := CheckTierCompliance(""); err != nil {
		t.Fatalf("CheckTierCompliance failed: %v", err)
	}

	for _, m := range Modules {
		tier, err := CurrentTier(m.Name)
		if err != nil {
			t.Fatalf("CurrentTier(%s) err = %v", m.Name, err)
		}
		if tier != kern.TierFree {
			t.Errorf("expected module %s to be free-tier, got %s", m.Name, tier)
		}
	}
}
