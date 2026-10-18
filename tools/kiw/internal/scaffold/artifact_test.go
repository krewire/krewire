package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeArtifact(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		kind     string
		name     string
		expected string
	}{
		{"component", "Button", filepath.Join("components", "Button.kiw")},
		{"page", "about-us", filepath.Join("pages", "about-us.kiw")},
		{"handler", "User", filepath.Join("internal", "handlers", "user.go")},
		{"model", "Account", filepath.Join("internal", "models", "account.go")},
		{"middleware", "Auth", filepath.Join("internal", "middleware", "auth.go")},
		{"service", "Payment", filepath.Join("internal", "services", "payment.go")},
	}

	for _, tc := range tests {
		t.Run(tc.kind+"_"+tc.name, func(t *testing.T) {
			if err := MakeArtifact(tmpDir, tc.kind, tc.name); err != nil {
				t.Fatalf("MakeArtifact failed for %s %s: %v", tc.kind, tc.name, err)
			}
			target := filepath.Join(tmpDir, tc.expected)
			if _, err := os.Stat(target); os.IsNotExist(err) {
				t.Fatalf("expected file was not created: %s", target)
			}

			// Duplicate test
			if err := MakeArtifact(tmpDir, tc.kind, tc.name); err == nil {
				t.Fatalf("expected error on duplicate creation for %s, got nil", tc.kind)
			}
		})
	}

	// Test migration
	t.Run("migration", func(t *testing.T) {
		if err := MakeArtifact(tmpDir, "migration", "create_users"); err != nil {
			t.Fatalf("MakeArtifact migration failed: %v", err)
		}
		files, err := os.ReadDir(filepath.Join(tmpDir, "migrations"))
		if err != nil || len(files) == 0 {
			t.Fatalf("expected migration file in migrations dir, got %v", files)
		}
		if !strings.HasSuffix(files[0].Name(), "_create_users.sql") {
			t.Errorf("migration filename mismatch: %s", files[0].Name())
		}
	})

	// Test invalid kind
	t.Run("invalid_kind", func(t *testing.T) {
		if err := MakeArtifact(tmpDir, "unknown_kind", "Test"); err == nil {
			t.Fatal("expected error for unknown kind, got nil")
		}
	})
}
