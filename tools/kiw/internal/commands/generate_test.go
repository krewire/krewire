package commands

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/kern"
)

func setupTestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Create minimal krewire.yaml so findRoot() discovers it
	cfgPath := filepath.Join(dir, "krewire.yaml")
	if err := os.WriteFile(cfgPath, []byte("project:\n  kind: site\n"), 0o644); err != nil {
		t.Fatalf("failed to create krewire.yaml: %v", err)
	}
	return dir
}

func TestGenerate_Usage(t *testing.T) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fs)

	// No arguments
	code := RunGenerate(fs)
	if code != kern.ExitCodeUsage {
		t.Errorf("RunGenerate() with no args = %v, want ExitCodeUsage", code)
	}

	// Unknown kind
	fsUnknown := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fsUnknown)
	_ = fsUnknown.Parse([]string{"unknown_kind", "sample"})
	code = RunGenerate(fsUnknown)
	if code != kern.ExitCodeUsage {
		t.Errorf("RunGenerate() with unknown kind = %v, want ExitCodeUsage", code)
	}
}

func TestGenerate_Component(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fs)
	_ = fs.Parse([]string{"component", "card-item"})

	code := RunGenerate(fs)
	if code != kern.ExitCodeSuccess {
		t.Fatalf("RunGenerate component failed with code %v", code)
	}

	target := filepath.Join(root, "components", "CardItem.kiw")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected components/CardItem.kiw to exist: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, `<div class="card-item">`) {
		t.Errorf("expected component class card-item, got: %s", content)
	}

	// Duplicate component should fail
	fsDup := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fsDup)
	_ = fsDup.Parse([]string{"component", "CardItem"})
	if code := RunGenerate(fsDup); code != kern.ExitCodeFailure {
		t.Errorf("duplicate component generation should fail, got code %v", code)
	}
}

func TestGenerate_Page(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fs)
	_ = fs.Parse([]string{"page", "pricing"})

	code := RunGenerate(fs)
	if code != kern.ExitCodeSuccess {
		t.Fatalf("RunGenerate page failed with code %v", code)
	}

	target := filepath.Join(root, "pages", "pricing.kiw")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected pages/pricing.kiw to exist: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, `title: "Pricing"`) || !strings.Contains(content, `<div class="pricing-page">`) {
		t.Errorf("unexpected page content: %s", content)
	}
}

func TestGenerate_Handler(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fs)
	_ = fs.Parse([]string{"handler", "create-user"})

	code := RunGenerate(fs)
	if code != kern.ExitCodeSuccess {
		t.Fatalf("RunGenerate handler failed with code %v", code)
	}

	target := filepath.Join(root, "internal", "handlers", "create_user.go")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected internal/handlers/create_user.go to exist: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, `func CreateUserHandler`) {
		t.Errorf("expected CreateUserHandler in content: %s", content)
	}
}

func TestGenerate_Migration(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	RegisterGenerate(fs)
	_ = fs.Parse([]string{"--kind", "migration", "--name", "create_teams"})

	code := RunGenerate(fs)
	if code != kern.ExitCodeSuccess {
		t.Fatalf("RunGenerate migration failed with code %v", code)
	}

	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected migrations/ directory with created file: %v", err)
	}
	name := entries[0].Name()
	if !strings.HasSuffix(name, "_create_teams.sql") {
		t.Errorf("expected migration filename suffix _create_teams.sql, got: %s", name)
	}
	b, _ := os.ReadFile(filepath.Join(root, "migrations", name))
	content := string(b)
	if !strings.Contains(content, "-- +up") || !strings.Contains(content, "-- +down") {
		t.Errorf("expected SQL migration markers in: %s", content)
	}
}
