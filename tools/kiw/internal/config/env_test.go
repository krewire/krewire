// Tests for KWL-K4T7W
package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/packages/kern"
)

func writeConfig(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Spec: KWL-K4T7W KWL-ENVV-003 Scope: Domain
func TestKWL_ENVV_003_LoadParsesEnvAndDebug(t *testing.T) {
	c := writeConfig(t, "env: production\ndebug: true\n")
	if c.Env != "production" || !c.Debug {
		t.Errorf("config = (env %q, debug %v), want (production, true)", c.Env, c.Debug)
	}
	env, err := c.ResolveEnv("")
	if err != nil || env != kern.EnvProduction {
		t.Errorf("ResolveEnv = (%q, %v), want (production, nil)", env, err)
	}
}

// Spec: KWL-K4T7W KWL-ENVV-003 Scope: Domain
func TestKWL_ENVV_003_EmptyConfigResolvesDefaults(t *testing.T) {
	c := writeConfig(t, "title: x\n")
	env, err := c.ResolveEnv("")
	if err != nil || env != kern.DefaultEnv {
		t.Errorf("ResolveEnv = (%q, %v), want (%q, nil)", env, err, kern.DefaultEnv)
	}
	if c.ResolveDebug("", false) {
		t.Error("ResolveDebug = true, want default false")
	}
}

// Spec: KWL-K4T7W KWL-ENVV-004 Scope: Domain
func TestKWL_ENVV_004_ResolveEnv_Precedence(t *testing.T) {
	yamlTesting := writeConfig(t, "env: testing\n")

	t.Setenv("KIW_ENV", "production")
	if env, _ := yamlTesting.ResolveEnv(""); env != kern.EnvProduction {
		t.Errorf("KIW_ENV should beat krewire.yaml, got %q", env)
	}
	if env, _ := yamlTesting.ResolveEnv("local"); env != kern.EnvLocal {
		t.Errorf("flag should beat KIW_ENV and yaml, got %q", env)
	}

	t.Setenv("KIW_ENV", "")
	if env, _ := yamlTesting.ResolveEnv(""); env != kern.EnvTesting {
		t.Errorf("yaml should win when no overrides, got %q", env)
	}
}

// Spec: KWL-K4T7W KWL-ENVV-002 KWL-ENVV-007 Scope: Domain
func TestKWL_ENVV_007_ResolveEnv_InvalidYamlValueIsUsageError(t *testing.T) {
	c := writeConfig(t, "env: staging\n")
	_, err := c.ResolveEnv("")
	if err == nil {
		t.Fatal("ResolveEnv(staging) = nil error, want usage error")
	}
	var ce interface{ ExitCode() kern.ExitCode }
	if !errors.As(err, &ce) || ce.ExitCode() != kern.ExitCodeUsage {
		t.Errorf("ResolveEnv(staging) error = %v, want usage exit code", err)
	}
}

// Spec: KWL-K4T7W KWL-ENVV-004 Scope: Domain
func TestKWL_ENVV_004_ResolveDebug_Precedence(t *testing.T) {
	yamlOn := writeConfig(t, "debug: true\n")
	yamlOff := writeConfig(t, "title: x\n")

	if !yamlOn.ResolveDebug("", false) {
		t.Error("yaml debug: true should resolve true")
	}
	if yamlOff.ResolveDebug("", false) {
		t.Error("default should resolve false")
	}

	t.Setenv("KIW_DEBUG", "1")
	if !yamlOff.ResolveDebug("", false) {
		t.Error("KIW_DEBUG=1 should beat yaml default false")
	}
	if yamlOn.ResolveDebug("false", true) {
		t.Error("explicit --debug=false should beat KIW_DEBUG and krewire.yaml")
	}
}
