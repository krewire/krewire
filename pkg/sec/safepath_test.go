package sec_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/pkg/sec"
)

func TestSafePath_ValidCases(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "app")
	if err := os.MkdirAll(filepath.Join(baseDir, "uploads", "avatars"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple relative filename",
			input:    "profile.png",
			expected: filepath.Join(baseDir, "profile.png"),
		},
		{
			name:     "nested relative path",
			input:    "uploads/avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
		{
			name:     "path with self-references",
			input:    "./uploads/./avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
		{
			name:     "path with benign backward resolution inside base",
			input:    "uploads/sub/../avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
		{
			name:     "leading slash treated as relative to base",
			input:    "/uploads/avatars/user_1.png",
			expected: filepath.Join(baseDir, "uploads", "avatars", "user_1.png"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sec.SafePath(baseDir, tc.input)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestSafePath_TraversalAttacks(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "app")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	attacks := []struct {
		name  string
		input string
	}{
		{name: "parent step-out", input: "../secret.txt"},
		{name: "multi parent step-out", input: "../../../etc/passwd"},
		{name: "nested step-out", input: "uploads/../../secret.txt"},
		{name: "deeply nested step-out", input: "a/b/c/../../../../etc/shadow"},
		{name: "backslashes on windows/cross-platform", input: "..\\..\\etc\\passwd"},
	}

	for _, tc := range attacks {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sec.SafePath(baseDir, tc.input)
			if err == nil {
				t.Fatalf("expected error for traversal attack %q, got path %q", tc.input, got)
			}
			if !errors.Is(err, sec.ErrPathTraversal) {
				t.Errorf("expected ErrPathTraversal, got: %v", err)
			}
		})
	}
}

func TestSafePath_InvalidParameters(t *testing.T) {
	_, err := sec.SafePath("", "file.txt")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for empty base, got: %v", err)
	}

	_, err = sec.SafePath("/some/dir", "")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for empty target, got: %v", err)
	}

	_, err = sec.SafePath("   ", "file.txt")
	if !errors.Is(err, sec.ErrInvalidPath) {
		t.Errorf("expected ErrInvalidPath for whitespace base, got: %v", err)
	}
}
