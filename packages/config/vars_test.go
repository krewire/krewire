// Tests for KWL-2X1QZ
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Spec: KWL-2X1QZ CFG-KV-002 Scope: Unit
func TestCFG_KV_002_LoadVars_EmptyPathYieldsEmptyVars(t *testing.T) {
	v, err := LoadVars("")
	if err != nil {
		t.Fatalf("LoadVars empty path error: %v", err)
	}
	if len(v) != 0 {
		t.Errorf("expected empty vars, got %v", v)
	}
}

// Spec: KWL-2X1QZ CFG-KV-002 Scope: Unit
func TestCFG_KV_002_LoadVars_MissingFileYieldsEmptyVars(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "nonexistent.yaml")
	v, err := LoadVars(path)
	if err != nil {
		t.Fatalf("LoadVars non-existent error: %v", err)
	}
	if len(v) != 0 {
		t.Errorf("expected empty vars for non-existent file, got %v", v)
	}
}

// Spec: KWL-2X1QZ CFG-KV-002 CFG-KV-003 Scope: Unit
func TestCFG_KV_003_SaveRoundTripsThroughNestedYAML(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "config.yaml")

	original := Vars{
		"app.name":       "testapp",
		"server.port":    "8080",
		"database.host":  "localhost",
		"features.debug": "true",
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := LoadVars(path)
	if err != nil {
		t.Fatalf("LoadVars error: %v", err)
	}

	expected := map[string]string{
		"app.name":       "testapp",
		"server.port":    "8080",
		"database.host":  "localhost",
		"features.debug": "true",
	}
	for k, v := range expected {
		if loaded[k] != v {
			t.Errorf("key %q: got %q, want %q", k, loaded[k], v)
		}
	}
}

// Spec: KWL-2X1QZ CFG-KV-001 Scope: Unit
func TestCFG_KV_001_GetSetDelete(t *testing.T) {
	v := make(Vars)
	v.Set("key1", "value1")
	if v.Get("key1") != "value1" {
		t.Errorf("Get: got %q, want value1", v.Get("key1"))
	}
	v.Delete("key1")
	if v.Get("key1") != "" {
		t.Errorf("Delete failed: got %q, want empty", v.Get("key1"))
	}
}

// Spec: KWL-2X1QZ CFG-KV-001 Scope: Unit
func TestCFG_KV_001_KeysSorted(t *testing.T) {
	v := Vars{
		"zebra": "1",
		"alpha": "2",
		"beta":  "3",
	}
	keys := v.Keys()
	expected := []string{"alpha", "beta", "zebra"}
	if len(keys) != 3 || keys[0] != "alpha" || keys[1] != "beta" || keys[2] != "zebra" {
		t.Errorf("Keys not sorted: got %v, want %v", keys, expected)
	}
}

// Spec: KWL-2X1QZ CFG-KV-005 Scope: Unit
func TestCFG_KV_005_GetOr_ReturnsFallbackWhenAbsentOrEmpty(t *testing.T) {
	v := Vars{"A": "x", "B": ""}
	if got := v.GetOr("A", "d"); got != "x" {
		t.Errorf("GetOr(A) = %q, want x", got)
	}
	if got := v.GetOr("B", "d"); got != "d" {
		t.Errorf("GetOr(B empty) = %q, want fallback d", got)
	}
	if got := v.GetOr("C", "d"); got != "d" {
		t.Errorf("GetOr(C absent) = %q, want fallback d", got)
	}
}

// Spec: KWL-2X1QZ CFG-KV-004 Scope: Unit
func TestCFG_KV_004_MergeLaterSourceWins(t *testing.T) {
	a := Vars{"a": "1", "b": "2"}
	b := Vars{"b": "new", "c": "3"}
	merged := a.Merge(b)
	expected := map[string]string{"a": "1", "b": "new", "c": "3"}
	for k, v := range expected {
		if merged[k] != v {
			t.Errorf("Merge: key %q got %q, want %q", k, merged[k], v)
		}
	}
}

// Spec: KWL-2X1QZ CFG-KV-004 Scope: Unit
func TestCFG_KV_004_WithDefaultsFillsMissingKeysOnly(t *testing.T) {
	v := Vars{"existing": "value"}
	defaults := Vars{"existing": "default", "new": "default"}
	result := v.WithDefaults(defaults)
	if result["existing"] != "value" {
		t.Errorf("existing key overwritten: got %q, want value", result["existing"])
	}
	if result["new"] != "default" {
		t.Errorf("default not applied: got %q, want default", result["new"])
	}
}

// Spec: KWL-2X1QZ CFG-KV-004 Scope: Unit
func TestCFG_KV_004_EnvOverrideMapsPrefixedVarsToDottedKeys(t *testing.T) {
	os.Setenv("APP_SERVER_PORT", "8080")
	os.Setenv("APP_DB_HOST", "prod-db")
	defer os.Unsetenv("APP_SERVER_PORT")
	defer os.Unsetenv("APP_DB_HOST")

	v := make(Vars)
	v.EnvOverride("APP")
	if v.Get("server.port") != "8080" {
		t.Errorf("EnvOverride: server.port = %q, want 8080", v.Get("server.port"))
	}
	if v.Get("db.host") != "prod-db" {
		t.Errorf("EnvOverride: db.host = %q, want prod-db", v.Get("db.host"))
	}
}

// Spec: KWL-2X1QZ CFG-KV-002 CFG-KV-003 Scope: Unit
func TestCFG_KV_002_FlattenUnflattenRoundTrip(t *testing.T) {
	nested := map[string]any{
		"server": map[string]any{
			"port": 8080,
			"host": "localhost",
		},
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
		"features": []any{
			map[string]any{"name": "auth", "enabled": true},
			map[string]any{"name": "cache", "enabled": false},
		},
	}

	flat := make(Vars)
	flatten(nested, "", flat)

	unflat := unflatten(flat)

	// Values are stored as strings, so we need to handle type conversion
	server := unflat["server"].(map[string]any)
	if server["port"] != "8080" && server["port"] != float64(8080) {
		t.Errorf("unflatten port mismatch: got %v (%T)", server["port"], server["port"])
	}
	if server["host"] != "localhost" {
		t.Errorf("unflatten host mismatch")
	}
}
