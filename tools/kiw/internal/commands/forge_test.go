package commands

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
)

func TestForgeCasingHelpers(t *testing.T) {
	if got := forgeToSlug("UserProfile"); got != "user-profile" {
		t.Errorf("forgeToSlug(UserProfile) = %q, want user-profile", got)
	}
	if got := forgeToPascal("user_profile"); got != "UserProfile" {
		t.Errorf("forgeToPascal(user_profile) = %q, want UserProfile", got)
	}
	if got := forgeToTitle("user-profile"); got != "User Profile" {
		t.Errorf("forgeToTitle(user-profile) = %q, want User Profile", got)
	}
}

func TestForgeUsageOnEmpty(t *testing.T) {
	fs := flag.NewFlagSet("forge", flag.ContinueOnError)
	RegisterForge(fs)
	code := RunForge(fs)
	if code != errs.ExitCodeUsage {
		t.Errorf("expected ExitCodeUsage, got %v", code)
	}
}

func TestForgePanel(t *testing.T) {
	tmpDir := t.TempDir()

	err := forgePanel(tmpDir, "admin", false)
	if err != nil {
		t.Fatalf("forgePanel failed: %v", err)
	}

	goFile := filepath.Join(tmpDir, "panels", "admin", "panel.go")
	kiwFile := filepath.Join(tmpDir, "panels", "admin", "Dashboard.kiw")

	goBytes, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("failed to read panel.go: %v", err)
	}
	if !strings.Contains(string(goBytes), "package admin") {
		t.Errorf("panel.go does not contain package admin")
	}

	kiwBytes, err := os.ReadFile(kiwFile)
	if err != nil {
		t.Fatalf("failed to read Dashboard.kiw: %v", err)
	}
	if !strings.Contains(string(kiwBytes), "Admin Panel") {
		t.Errorf("Dashboard.kiw does not contain Admin Panel")
	}
}

func TestForgeResource(t *testing.T) {
	tmpDir := t.TempDir()

	err := forgeResource(tmpDir, "product", "Product", false)
	if err != nil {
		t.Fatalf("forgeResource failed: %v", err)
	}

	goFile := filepath.Join(tmpDir, "resources", "product", "resource.go")
	indexFile := filepath.Join(tmpDir, "resources", "product", "Index.kiw")
	formFile := filepath.Join(tmpDir, "resources", "product", "Form.kiw")

	for _, f := range []string{goFile, indexFile, formFile} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
}

func TestForgeForm(t *testing.T) {
	tmpDir := t.TempDir()

	err := forgeForm(tmpDir, "contact", "name:string,email:string,message:textarea", false)
	if err != nil {
		t.Fatalf("forgeForm failed: %v", err)
	}

	goFile := filepath.Join(tmpDir, "forms", "contact_form.go")
	kiwFile := filepath.Join(tmpDir, "components", "ContactForm.kiw")

	goBytes, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("failed to read form go file: %v", err)
	}
	if !strings.Contains(string(goBytes), "type ContactForm struct") {
		t.Errorf("expected ContactForm struct definition in %s", string(goBytes))
	}

	if _, err := os.Stat(kiwFile); err != nil {
		t.Errorf("expected ContactForm.kiw to exist: %v", err)
	}
}
