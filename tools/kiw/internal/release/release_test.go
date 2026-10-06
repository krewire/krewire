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

func TestPlanReleasingLibsPropagatesToAllDependents(t *testing.T) {
	edits, err := Plan([]string{string(ModuleLibs)}, BumpPatch)
	if err != nil {
		t.Fatal(err)
	}
	// Own bump + one requirement raise per dependent that actually declares
	// libs in its EcosystemRequires. Dependents that do not require libs
	// contribute no edit, so the count is derived rather than hardcoded.
	want := 1
	for _, d := range dependents[string(ModuleLibs)] {
		if _, ok := RequiredVersion(d, string(ModuleLibs)); ok {
			want++
		}
	}
	if len(edits) != want {
		t.Fatalf("got %d edits, want %d (own bump + %d dependents): %+v",
			len(edits), want, len(dependents[string(ModuleLibs)]), edits)
	}
	nv := Bump(mustCur(string(ModuleLibs)), BumpPatch)
	for _, e := range edits {
		if e.Module == string(ModuleLibs) {
			if e.To != versionDecl(string(ModuleLibs), nv.String()) {
				t.Errorf("libs own edit To = %q, want %q", e.To, versionDecl(string(ModuleLibs), nv.String()))
			}
			continue
		}
		want := reqDecl(string(ModuleLibs), nv.String())
		if e.To != want {
			t.Errorf("dependent %s To = %q, want %q", e.Module, e.To, want)
		}
	}
}

// Every leaf module has exactly one dependent — kiw — so releasing one touches
// two files: its own and kiw's requirement on it.
func TestPlanReleasingLeafTouchesOnlyKiw(t *testing.T) {
	edits, err := Plan([]string{string(ModuleBoost)}, BumpPatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 2 {
		t.Fatalf("got %d edits, want 2 (boost + kiw): %+v", len(edits), edits)
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
	// One own bump per module, plus one requirement raise per declared edge.
	edges := 0
	for r, deps := range dependents {
		for _, d := range deps {
			if _, ok := RequiredVersion(d, r); ok {
				edges++
			}
		}
	}
	want := len(Modules) + edges
	if len(edits) != want {
		t.Fatalf("got %d edits, want %d (%d modules + %d dependency edges): %+v",
			len(edits), want, len(Modules), edges, edits)
	}
}

func TestApplyWritesAndValidates(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v.go")
	orig := `var Version = kern.MustParseVersion("0.1.0")`
	if err := os.WriteFile(p, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	edits := []Edit{{Module: string(ModuleBoost), File: "v.go", From: `var Version = kern.MustParseVersion("0.1.0")`, To: `var Version = kern.MustParseVersion("0.2.0")`}}

	if _, err := Apply(edits, dir, true); err != nil {
		t.Fatalf("dry-run apply error: %v", err)
	}
	if data, _ := os.ReadFile(p); string(data) != orig {
		t.Errorf("dry-run modified the file: %q", string(data))
	}
	if _, err := Apply(edits, dir, false); err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if data, _ := os.ReadFile(p); string(data) != `var Version = kern.MustParseVersion("0.2.0")` {
		t.Errorf("file not updated: %q", string(data))
	}
}

func TestApplyRejectsAmbiguousMatch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v.go")
	_ = os.WriteFile(p, []byte(`x: kern.MustParseVersion("0.1.0")
y: kern.MustParseVersion("0.1.0")`), 0o644)
	edits := []Edit{{Module: string(ModuleBoost), File: "v.go", From: `kern.MustParseVersion("0.1.0")`, To: `kern.MustParseVersion("0.2.0")`}}
	if _, err := Apply(edits, dir, false); err == nil {
		t.Error("expected ambiguous-match error")
	}
}

func mustCur(n string) kern.Version {
	v, err := CurrentVersion(n)
	if err != nil {
		panic(err)
	}
	return v
}
