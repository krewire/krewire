// Tests for KWN-PKG-7X9K2
package commands

import (
	"flag"
	"testing"

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
