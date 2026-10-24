package commands

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
)

func TestAuditSecurityPosture(t *testing.T) {
	tmp := t.TempDir()
	cleanYAML := `project:
  name: safe-app
  kind: app
auth:
  secret: "${env:APP_SECRET:-default-secret}"
`
	if err := os.WriteFile(filepath.Join(tmp, "krewire.yaml"), []byte(cleanYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	issues := auditSecurityPosture(tmp)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for clean config, got: %v", issues)
	}

	leakyYAML := `project:
  name: leaky-app
auth:
  password: supersecretpassword
`
	if err := os.WriteFile(filepath.Join(tmp, "krewire.yaml"), []byte(leakyYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	issues = auditSecurityPosture(tmp)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for leaky config, got: %v", issues)
	}
}

func TestAuditEnvReferences(t *testing.T) {
	tmp := t.TempDir()
	yamlWithVars := `project:
  name: test-app
database:
  url: "${env:DB_URL}"
  port: "${DB_PORT:-5432}"
`
	if err := os.WriteFile(filepath.Join(tmp, "krewire.yaml"), []byte(yamlWithVars), 0o600); err != nil {
		t.Fatal(err)
	}

	// Case 1: DB_URL is missing
	issues := auditEnvReferences(tmp)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for missing DB_URL, got: %v", issues)
	}

	// Case 2: DB_URL is defined in .env
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte("DB_URL=postgres://localhost/db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	issues = auditEnvReferences(tmp)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues after adding .env, got: %v", issues)
	}
}

func TestRunVetVuln(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()

	goMod := "module example.com/testvuln\n\ngo 1.27.1\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}

	mainCode := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(mainCode), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	fs := flag.NewFlagSet("vet", flag.ContinueOnError)
	RegisterVet(fs)
	_ = fs.Set("vuln", "true")

	code := RunVet(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("got exit code %v, want ExitCodeSuccess (0)", code)
	}
}
