package boost

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateEmbedded(t *testing.T) {
	for _, want := range []string{
		"AGENTS.md",
		"opencode.json",
		".agents/agents/build.md",
		".agents/commands/kickoff.md",
		".agents/skills/project-init/SKILL.md",
	} {
		if _, err := Template.Open(want); err != nil {
			t.Errorf("Template missing %s: %v", want, err)
		}
	}
}

func TestManaged(t *testing.T) {
	got := Managed()
	for _, want := range managed() {
		if !contains(got, want) {
			t.Errorf("Managed missing %q", want)
		}
	}
}

func TestInstallFresh(t *testing.T) {
	target := t.TempDir()
	got, err := Install(target)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Install returned no files")
	}
	for _, p := range got {
		if !hasPrefixAlias(p+"/", target+"/") {
			t.Errorf("created path %q outside target", p)
		}
	}
	for _, want := range []string{"AGENTS.md", "opencode.json"} {
		dest := filepath.Join(target, want)
		if _, err := os.Stat(dest); err != nil {
			t.Errorf("expected %s: %v", dest, err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(target, "AGENTS.md")); err != nil || !strings.Contains(string(data), "Agent Constitution") {
		t.Errorf("AGENTS.md not written correctly: %v %q", err, data)
	}
}

func TestInstallConflictRefuses(t *testing.T) {
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Install(target)
	if !errors.Is(err, ErrConflicts) {
		t.Fatalf("want ErrConflicts, got %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(target, "AGENTS.md")); err != nil || string(data) != "mine" {
		t.Errorf("conflict must not overwrite: %v %q", err, data)
	}
	if _, err := os.Stat(filepath.Join(target, "opencode.json")); !os.IsNotExist(err) {
		t.Errorf("refused install must write nothing (opencode.json created): %v", err)
	}
}

func TestInstallWithForceOverwrites(t *testing.T) {
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Install(target, WithForce())
	if err != nil {
		t.Fatalf("Install(WithForce): %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(target, "AGENTS.md")); err != nil || !strings.Contains(string(data), "Agent Constitution") {
		t.Errorf("WithForce must overwrite: %v %q", err, data)
	}
	if len(got) == 0 {
		t.Error("expected created paths")
	}
}

func TestInstallDryRunWritesNothing(t *testing.T) {
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Install(target, WithDryRun())
	if err != nil {
		t.Fatalf("Install(WithDryRun): %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(target, "AGENTS.md")); err != nil || string(data) != "mine" {
		t.Errorf("dry-run must not touch existing file: %v %q", err, data)
	}
	if _, err := os.Stat(filepath.Join(target, "opencode.json")); !os.IsNotExist(err) {
		t.Errorf("dry-run must not create files: %v", err)
	}
	if len(got) == 0 {
		t.Error("dry-run should still report paths")
	}
}

func TestInstallMissingTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nope")
	_, err := Install(target)
	if !errors.Is(err, ErrTargetMissing) {
		t.Fatalf("want ErrTargetMissing, got %v", err)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func hasPrefixAlias(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
