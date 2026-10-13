// Tests for KWN-PKG-7X9K2
package commands

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/packages/hub/plugin"
	"github.com/krewire/krewire/packages/hub/resolver"
	"github.com/krewire/krewire/packages/kern"
)

// Spec: KWN-PKG-7X9K2 KWN-PKG-001 Scope: Module
func TestKWN_PKG_001_RunAdd_EmptyArgsReturnsUsage(t *testing.T) {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	RegisterAdd(fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("fs.Parse: %v", err)
	}
	code := RunAdd(fs)
	if code != kern.ExitCodeUsage {
		t.Errorf("RunAdd(empty) = %v, want ExitCodeUsage", code)
	}
}

// Spec: KWN-PKG-7X9K2 KWN-PKG-001 Scope: Module
func TestKWN_PKG_001_RunRemove_EmptyArgsReturnsUsage(t *testing.T) {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	RegisterRemove(fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("fs.Parse: %v", err)
	}
	code := RunRemove(fs)
	if code != kern.ExitCodeUsage {
		t.Errorf("RunRemove(empty) = %v, want ExitCodeUsage", code)
	}
}

// Spec: KWN-PKG-7X9K2 KWN-PKG-001 Scope: Module
func TestKWN_PKG_001_RunAdd_UnsafePackageReturnsUsage(t *testing.T) {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	RegisterAdd(fs)
	if err := fs.Parse([]string{"file://../malicious"}); err != nil {
		t.Fatalf("fs.Parse: %v", err)
	}
	code := RunAdd(fs)
	if code != kern.ExitCodeUsage {
		t.Errorf("RunAdd(unsafe) = %v, want ExitCodeUsage", code)
	}
}

// Spec: KWN-PKG-7X9K2 KWN-PKG-002 Scope: Module
// Spec: KWN-PKG-7X9K2 KWN-PKG-006 Scope: Module
func TestKWN_PKG_006_TailwindPlugin_AddAndRemove(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	tw := &plugin.Tailwind{}
	if err := tw.Add(dir, "latest"); err != nil {
		t.Logf("tw.Add note: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tailwind.config.js")); err != nil {
		t.Errorf("expected tailwind.config.js to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "tailwind.css")); err != nil {
		t.Errorf("expected assets/tailwind.css to exist: %v", err)
	}

	if err := tw.Remove(dir); err != nil {
		t.Logf("tw.Remove note: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tailwind.config.js")); !os.IsNotExist(err) {
		t.Errorf("expected tailwind.config.js to be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "tailwind.css")); !os.IsNotExist(err) {
		t.Errorf("expected unmodified assets/tailwind.css to be removed")
	}
}

// Spec: KWN-PKG-7X9K2 KWN-PKG-004 Scope: Module
// Spec: KWN-PKG-7X9K2 KWN-PKG-005 Scope: Module
func TestKWN_PKG_004_ResolverChainOrder(t *testing.T) {
	chain := resolver.DefaultChain()

	// 1. Plugin alias matches plugin
	res, err := chain.Resolve(resolver.Spec{Name: "twcss", Raw: "twcss"})
	if err != nil {
		t.Fatalf("resolve twcss: %v", err)
	}
	if res.Kind != "plugin" {
		t.Errorf("twcss kind = %q, want plugin", res.Kind)
	}

	// 2. Go module path matches gomod
	resGo, err := chain.Resolve(resolver.Spec{Name: "github.com/foo/bar", Raw: "github.com/foo/bar"})
	if err != nil {
		t.Fatalf("resolve go module: %v", err)
	}
	if resGo.Kind != "gomod" {
		t.Errorf("go module kind = %q, want gomod", resGo.Kind)
	}

	// 3. Npm package fallback
	resNpm, err := chain.Resolve(resolver.Spec{Name: "lodash", Raw: "lodash"})
	if err != nil {
		t.Fatalf("resolve npm: %v", err)
	}
	if resNpm.Kind != "npm" {
		t.Errorf("npm package kind = %q, want npm", resNpm.Kind)
	}
}
