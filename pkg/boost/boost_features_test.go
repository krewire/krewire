package boost_test

import (
	"testing"

	"github.com/krewire/krewire/pkg/boost"
	"github.com/krewire/krewire/pkg/model"
)

func TestPersonaRegistry_CanonicalPersonas(t *testing.T) {
	reg, err := boost.NewPersonaRegistry(boost.Template)
	if err != nil {
		t.Fatalf("failed to create persona registry: %v", err)
	}

	if reg.Count() < 21 {
		t.Errorf("expected at least 21 canonical personas, got %d", reg.Count())
	}

	// Verify all canonical names exist
	for _, name := range boost.CanonicalPersonaNames {
		p, ok := reg.Get(name)
		if !ok || p == nil {
			t.Errorf("expected persona %q to exist in registry", name)
		}
	}

	// Verify alias resolution
	aliasTests := []struct {
		alias        string
		expectedName string
	}{
		{"plan", "planner"},
		{"build", "builder"},
		{"coder", "builder"},
		{"qa", "tester"},
		{"test", "tester"},
		{"architect", "designer"},
		{"devops", "deployer"},
		{"security", "auditor"},
		{"refactor", "maintainer"},
		{"docs", "writer"},
	}

	for _, tt := range aliasTests {
		p, ok := reg.Get(tt.alias)
		if !ok || p.Name != tt.expectedName {
			t.Errorf("alias %q resolved to %v (ok=%v), want %q", tt.alias, p, ok, tt.expectedName)
		}
	}

	// Non-existent persona
	if _, ok := reg.Get("nonexistent_persona_xyz"); ok {
		t.Errorf("expected nonexistent persona to return false")
	}
}

func TestCommandRegistry_CanonicalCommands(t *testing.T) {
	reg, err := boost.NewCommandRegistry(boost.Template)
	if err != nil {
		t.Fatalf("failed to create command registry: %v", err)
	}

	// Test plan command retrieval
	planCmd, ok := reg.Get("/plan")
	if !ok || planCmd.Name != "plan" {
		t.Fatalf("failed to retrieve /plan command")
	}

	// Test command rendering with arguments
	rendered, err := reg.Render("/plan", "implement authentication with PKCE")
	if err != nil {
		t.Fatalf("failed to render /plan command: %v", err)
	}
	if !contains(rendered, "implement authentication with PKCE") {
		t.Errorf("expected rendered prompt to contain arguments: %s", rendered)
	}

	// Non-existent command
	if _, err := reg.Render("/unknown", "args"); err == nil {
		t.Errorf("expected error for nonexistent slash command")
	}
}

func TestValidateCommitMessage(t *testing.T) {
	validMessages := []string{
		"feat(auth): implement PKCE authorization code flow",
		"fix(sec): prevent path traversal in SafePath",
		"docs(boost): document 21 autonomous agent personas",
		"test(model): add unit tests for verification ledger",
		"chore(deps): bump dependencies",
		"refactor(ui): extract reusable component tokens",
	}

	for _, msg := range validMessages {
		if err := boost.ValidateCommitMessage(msg); err != nil {
			t.Errorf("expected valid commit message %q to pass, got: %v", msg, err)
		}
	}

	invalidMessages := []string{
		"",
		"updated files",
		"fix:something without space",
		"WIP",
		"feat : wrong space before colon",
	}

	for _, msg := range invalidMessages {
		if err := boost.ValidateCommitMessage(msg); err == nil {
			t.Errorf("expected invalid commit message %q to fail", msg)
		}
	}
}

func TestConstitutionFilter_ValidateFileMutations(t *testing.T) {
	filter := boost.DefaultConstitutionFilter()

	// Under 300 LOC limit
	smallMutations := []model.FileMutation{
		{Path: "pkg/auth/token.go", LinesAdded: 50, LinesRemoved: 10},
		{Path: "pkg/auth/token_test.go", LinesAdded: 80, LinesRemoved: 5},
	}
	if err := filter.ValidateFileMutations(smallMutations); err != nil {
		t.Errorf("expected small mutation to pass, got: %v", err)
	}

	// Over 300 LOC limit
	largeMutations := []model.FileMutation{
		{Path: "big.go", LinesAdded: 250, LinesRemoved: 100},
	}
	if err := filter.ValidateFileMutations(largeMutations); err == nil {
		t.Errorf("expected mutation over 300 LOC to fail")
	}
}

func TestEvaluatePillarHealth(t *testing.T) {
	// Empty pillars triggers diagnostics
	ctx := model.ProjectContext{
		Name: "test-app",
	}
	diags := boost.EvaluatePillarHealth(ctx)
	if len(diags) != 3 {
		t.Errorf("expected 3 diagnostics for 3 missing pillars, got %d", len(diags))
	}

	// Populated pillars
	ctx.Pillars.Secure = []model.ContextRule{{ID: "SEC-1"}}
	ctx.Pillars.Sustainable = []model.ContextRule{{ID: "SUSTAIN-1"}}
	ctx.Pillars.Scalable = []model.ContextRule{{ID: "SCALE-1"}}
	diags = boost.EvaluatePillarHealth(ctx)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for complete pillars, got %d", len(diags))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
