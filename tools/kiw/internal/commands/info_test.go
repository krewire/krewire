package commands

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/kern"
)

func TestRunInfo_NodeJSEcosystemAndColocated(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Host Node.js project
	pkgJSON := `{
		"name": "host-frontend",
		"dependencies": {
			"@sveltejs/kit": "^2.0.0"
		},
		"scripts": {
			"build": "vite build"
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Co-located Krewire service
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module host/api\n\ngo 1.27.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, code := captureStdout(t, func() kern.ExitCode {
		return RunInfo(&flag.FlagSet{})
	})

	if code != kern.ExitCodeSuccess {
		t.Fatalf("expected exit code success, got %d", code)
	}

	if !strings.Contains(output, "Node.js") {
		t.Errorf("expected output to mention Node.js, got:\n%s", output)
	}
	if !strings.Contains(output, "SvelteKit") {
		t.Errorf("expected output to mention SvelteKit, got:\n%s", output)
	}
	if !strings.Contains(output, "api/") {
		t.Errorf("expected output to mention co-located api/, got:\n%s", output)
	}
}

func TestRunInfo_LaravelEcosystemAndColocated(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Host Laravel project
	compJSON := `{
		"name": "laravel/laravel",
		"require": {
			"php": "^8.2",
			"laravel/framework": "^11.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(compJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "artisan"), []byte("#!/usr/bin/env php\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Co-located Krewire worker
	workerDir := filepath.Join(dir, "worker")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	krewireYaml := `project:
  name: queue-worker
  kind: worker
`
	if err := os.WriteFile(filepath.Join(workerDir, "krewire.yaml"), []byte(krewireYaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "go.mod"), []byte("module host/worker\n\ngo 1.27.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, code := captureStdout(t, func() kern.ExitCode {
		return RunInfo(&flag.FlagSet{})
	})

	if code != kern.ExitCodeSuccess {
		t.Fatalf("expected exit code success, got %d", code)
	}

	if !strings.Contains(output, "Laravel") {
		t.Errorf("expected output to mention Laravel, got:\n%s", output)
	}
	if !strings.Contains(output, "composer") {
		t.Errorf("expected output to mention composer, got:\n%s", output)
	}
	if !strings.Contains(output, "worker/") {
		t.Errorf("expected output to mention co-located worker/, got:\n%s", output)
	}
}
