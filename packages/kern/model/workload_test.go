// Tests for KWL-CORE-K1N2Q
package model

import (
	"testing"
)

// Spec: KWL-CORE-K1N2Q KWL-CORE-001 Scope: Unit
func TestParseKind(t *testing.T) {
	cases := []struct {
		in   string
		want Kind
		ok   bool
	}{
		{"app", KindApp, true},
		{"cli", KindCLI, true},
		{"worker", KindWorker, true},
		{"service", KindService, true},
		{"infra", KindInfra, true},
		{"kernel", KindKernel, true},
		{"unknown", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := ParseKind(c.in)
		if c.ok && err != nil {
			t.Errorf("ParseKind(%q) error = %v, want nil", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ParseKind(%q) = %v, want error", c.in, got)
		}
		if c.ok && got != c.want {
			t.Errorf("ParseKind(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-001 Scope: Unit
func TestKindIsValid(t *testing.T) {
	if !KindApp.IsValid() {
		t.Error("KindApp should be valid")
	}
	if Kind("nope").IsValid() {
		t.Error("unknown kind should be invalid")
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-001 Scope: Unit
// IsValid is derived from AllKinds: every listed kind is valid and nothing else.
func TestKWL_CORE_001_IsValidDerivedFromAllKinds(t *testing.T) {
	for _, k := range AllKinds {
		if !k.IsValid() {
			t.Errorf("AllKinds contains %q but IsValid() is false", k)
		}
	}
	if Kind("bogus").IsValid() {
		t.Error(`Kind("bogus").IsValid() = true, want false`)
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-002 Scope: Unit
// Matrix is generic over the owning package type and carries no roster of its
// own: the ecosystem declares the cells, the kernel only holds them.
func TestMatrixIsGenericAndCallerOwned(t *testing.T) {
	type moduleID int

	m := NewMatrix(
		Workload[string]{Kind: KindCLI, Package: "tui", Title: "CLI tools", Status: StatusShipped},
		Workload[string]{Kind: KindSite, Package: "web/ssg", Title: "Static sites (SSG)", Status: StatusShipped},
	)
	if m.Len() != 2 {
		t.Fatalf("Len = %d, want 2", m.Len())
	}

	w, ok := m.For(KindCLI)
	if !ok {
		t.Fatal("For(cli) not found")
	}
	if w.Package != "tui" || w.Title != "CLI tools" {
		t.Errorf("For(cli) = %+v, want the tui cell", w)
	}
	if _, ok := m.For(Kind("unknown")); ok {
		t.Error("For(unknown) should not be found")
	}

	// A second, differently typed matrix proves the shape is not bound to one
	// package representation.
	ids := NewMatrix(Workload[moduleID]{Kind: KindApp, Package: 7})
	if got, ok := ids.For(KindApp); !ok || got.Package != 7 {
		t.Errorf("matrix over moduleID: For(app) = %+v, %v; want package 7, true", got, ok)
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-002 Scope: Unit
// The first cell registered for a Kind wins, and iteration follows insertion
// order, so a report built from a matrix is deterministic.
func TestMatrixFirstWinsAndIsOrdered(t *testing.T) {
	m := NewMatrix(
		Workload[string]{Kind: KindApp, Package: "web", Title: "Backend / API"},
		Workload[string]{Kind: KindSite, Package: "web/ssg", Title: "Static sites (SSG)"},
	)
	m.Add(Workload[string]{Kind: KindApp, Package: "app", Title: "Fullstack / Monolith"})

	if m.Len() != 2 {
		t.Fatalf("Len = %d, want 2: a duplicate Kind must not be appended", m.Len())
	}
	w, _ := m.For(KindApp)
	if w.Package != "web" {
		t.Errorf("For(app) = %q, want the first registered cell %q", w.Package, "web")
	}

	wantTitles := []string{"Backend / API", "Static sites (SSG)"}
	got := m.All()
	for i := range wantTitles {
		if got[i].Title != wantTitles[i] {
			t.Errorf("All()[%d] = %q, want %q", i, got[i].Title, wantTitles[i])
		}
	}
	pkgs := m.Packages()
	if len(pkgs) != 2 || pkgs[0] != "web" || pkgs[1] != "web/ssg" {
		t.Errorf("Packages() = %v, want [web web/ssg]", pkgs)
	}
}

// Spec: KWL-CORE-K1N2Q KWL-CORE-003 Scope: Unit
func TestKWL_CORE_003_StatusValues(t *testing.T) {
	for _, s := range []Status{StatusShipped, StatusPlanned} {
		if s == "" {
			t.Error("a declared Status must not be empty")
		}
	}
}
