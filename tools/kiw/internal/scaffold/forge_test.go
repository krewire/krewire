package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldForgeCasing(t *testing.T) {
	if got := ToSlug("UserProfile"); got != "user-profile" {
		t.Errorf("ToSlug(UserProfile) = %q, want user-profile", got)
	}
	if got := ToPascal("user_profile"); got != "UserProfile" {
		t.Errorf("ToPascal(user_profile) = %q, want UserProfile", got)
	}
	if got := ToTitle("user-profile"); got != "User Profile" {
		t.Errorf("ToTitle(user-profile) = %q, want User Profile", got)
	}
}

func TestScaffoldForgePanel(t *testing.T) {
	tmpDir := t.TempDir()

	err := ForgePanel(tmpDir, "admin", false)
	if err != nil {
		t.Fatalf("ForgePanel failed: %v", err)
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

func TestScaffoldForgeResource(t *testing.T) {
	tmpDir := t.TempDir()

	err := ForgeResource(tmpDir, "product", "Product", false)
	if err != nil {
		t.Fatalf("ForgeResource failed: %v", err)
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

func TestScaffoldForgeForm(t *testing.T) {
	tmpDir := t.TempDir()

	err := ForgeForm(tmpDir, "contact", "name:string,email:string,message:textarea", false)
	if err != nil {
		t.Fatalf("ForgeForm failed: %v", err)
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
