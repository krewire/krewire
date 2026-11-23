package contract

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite published schemas and golden files")

// TestPublishedSchemasMatchGoTypes pins schemas/*.json to the Go types: a
// field added, removed, or retagged without regenerating fails here. Run
// `go test ./pkg/toolchain/contract -update` to regenerate.
func TestPublishedSchemasMatchGoTypes(t *testing.T) {
	for _, k := range Kinds {
		want, err := GenerateSchema(k)
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		path := filepath.Join("schemas", string(k)+".json")
		if *update {
			if err := os.WriteFile(path, want, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: published schema missing: %v (run with -update)", k, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: published schema drifted from Go types; run go test ./pkg/toolchain/contract -update", k)
		}
	}
	entries, _ := os.ReadDir("schemas")
	if len(entries) != len(Kinds) {
		t.Errorf("schemas/ holds %d files, want exactly one per kind (%d)", len(entries), len(Kinds))
	}
}

func TestSchemasAreWellFormed(t *testing.T) {
	for _, k := range Kinds {
		raw, err := Schema(k)
		if err != nil {
			t.Fatal(err)
		}
		var s map[string]any
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if s["$schema"] != SchemaDialect || s["$id"] != SchemaBaseURL+string(k)+".json" {
			t.Errorf("%s: wrong $schema/$id", k)
		}
	}
	if _, err := Schema("Nope"); err == nil {
		t.Error("unknown kind must fail")
	}
	if _, err := GenerateSchema("Nope"); err == nil {
		t.Error("unknown kind must fail")
	}
}

// TestGoldenDocuments decodes every testdata/golden/*.json example through
// the schema and the semantic validators, and checks it round-trips.
func TestGoldenDocuments(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("testdata", "golden", "*.json"))
	if len(files) < len(Kinds) {
		t.Fatalf("want at least one golden document per kind, have %d", len(files))
	}
	seen := map[Kind]bool{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := Decode(data)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		var meta TypeMeta
		_ = json.Unmarshal(data, &meta)
		seen[meta.Kind] = true
		again, _ := json.Marshal(doc)
		if err := ValidateJSON(meta.Kind, again); err != nil {
			t.Errorf("%s: re-encoded document no longer validates: %v", f, err)
		}
	}
	for _, k := range Kinds {
		if !seen[k] {
			t.Errorf("no golden document for kind %s", k)
		}
	}
}

func TestInvalidDocumentsAreRejected(t *testing.T) {
	cases := map[string]string{
		"bad json":       `{`,
		"unknown kind":   `{"api_version":"toolchain.krewire.dev/v1","kind":"Nope"}`,
		"old version":    `{"api_version":"toolchain.krewire.dev/v0","kind":"Plan"}`,
		"missing fields": `{"api_version":"toolchain.krewire.dev/v1","kind":"Plan"}`,
		"unknown field":  `{"api_version":"toolchain.krewire.dev/v1","kind":"Plan","task_id":"t","summary":"s","steps":[],"extra":1}`,
		"bad enum": `{"api_version":"toolchain.krewire.dev/v1","kind":"Plan","task_id":"t","summary":"s",
			"steps":[{"id":"a","description":"d","target_files":[],"risk":"extreme"}]}`,
		"wrong type": `{"api_version":"toolchain.krewire.dev/v1","kind":"Plan","task_id":"t","summary":"s","steps":"x"}`,
		"semantic":   `{"api_version":"toolchain.krewire.dev/v1","kind":"Plan","task_id":"t","summary":"s","steps":[]}`,
	}
	for name, doc := range cases {
		if _, err := Decode([]byte(doc)); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

// TestCommandCatalogGolden pins the published command hierarchy.
func TestCommandCatalogGolden(t *testing.T) {
	want, _ := json.MarshalIndent(Catalog(), "", "  ")
	want = append(want, '\n')
	path := filepath.Join("testdata", "golden", "command-catalog.json")
	if *update {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, want) {
		t.Error("command catalog drifted from testdata/golden/command-catalog.json; run with -update")
	}
}
