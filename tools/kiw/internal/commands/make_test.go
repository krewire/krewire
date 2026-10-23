package commands

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/errs"
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

func TestMake_Usage(t *testing.T) {
	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)

	// No arguments
	code := RunMake(fs)
	if code != errs.ExitCodeUsage {
		t.Errorf("RunMake() with no args = %v, want ExitCodeUsage", code)
	}

	// Unknown kind
	fsUnknown := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fsUnknown)
	_ = fsUnknown.Parse([]string{"unknown_kind", "sample"})
	code = RunMake(fsUnknown)
	if code != errs.ExitCodeUsage {
		t.Errorf("RunMake() with unknown kind = %v, want ExitCodeUsage", code)
	}
}

func TestMake_Component(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"component", "card-item"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake component failed with code %v", code)
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
	fsDup := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fsDup)
	_ = fsDup.Parse([]string{"component", "CardItem"})
	if code := RunMake(fsDup); code != errs.ExitCodeFailure {
		t.Errorf("duplicate component generation should fail, got code %v", code)
	}

	// Colon syntax: make:component Button
	fsColon := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fsColon)
	_ = fsColon.Parse([]string{"make:component", "button"})
	if code := RunMake(fsColon); code != errs.ExitCodeSuccess {
		t.Errorf("make:component button failed, got code %v", code)
	}
	if _, err := os.Stat(filepath.Join(root, "components", "Button.kiw")); err != nil {
		t.Errorf("expected components/Button.kiw to exist: %v", err)
	}
}

func TestMake_Page(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"page", "pricing"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake page failed with code %v", code)
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

func TestMake_Handler(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"handler", "create-user"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake handler failed with code %v", code)
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

func TestMake_Migration(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"--kind", "migration", "--name", "create_teams"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake migration failed with code %v", code)
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

func TestMake_Model(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"model", "account"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake model failed with code %v", code)
	}

	target := filepath.Join(root, "internal", "models", "account.go")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected internal/models/account.go to exist: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "type Account struct") || !strings.Contains(content, `json:"created_at"`) {
		t.Errorf("expected Account struct with timestamps, got: %s", content)
	}
}

func TestMake_Middleware(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"middleware", "auth"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake middleware failed with code %v", code)
	}

	target := filepath.Join(root, "internal", "middleware", "auth.go")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected internal/middleware/auth.go to exist: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "func Auth(next http.Handler) http.Handler") {
		t.Errorf("expected Auth middleware function, got: %s", content)
	}
}

func TestMake_Service(t *testing.T) {
	root := setupTestRoot(t)
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(root)

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	RegisterMake(fs)
	_ = fs.Parse([]string{"service", "billing"})

	code := RunMake(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("RunMake service failed with code %v", code)
	}

	target := filepath.Join(root, "internal", "services", "billing.go")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected internal/services/billing.go to exist: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "type BillingService interface") || !strings.Contains(content, "NewBillingService()") {
		t.Errorf("expected BillingService interface and constructor, got: %s", content)
	}
}
