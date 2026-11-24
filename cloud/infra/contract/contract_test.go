package contract

import (
	"strings"
	"testing"
)

func TestSnapshotPutFindRemove(t *testing.T) {
	var s Snapshot
	s.Put(ResourceState{Resource: Resource{Kind: "file", Name: "b"}})
	s.Put(ResourceState{Resource: Resource{Kind: "file", Name: "a"}})
	s.Put(ResourceState{Resource: Resource{Kind: "file", Name: "a"}, ID: "x"})
	if len(s.Resources) != 2 || s.Resources[0].Name != "a" || s.Resources[0].ID != "x" {
		t.Fatalf("unexpected resources: %+v", s.Resources)
	}
	c := s.Clone()
	c.Remove("file/a")
	if _, ok := c.Find("file/a"); ok {
		t.Fatal("removed resource still found")
	}
	if _, ok := s.Find("file/a"); !ok {
		t.Fatal("clone aliased original")
	}
}

func TestCheckContract(t *testing.T) {
	if err := CheckContract(Capabilities{Contract: Version}); err != nil {
		t.Fatal(err)
	}
	if err := CheckContract(Capabilities{Contract: "infra.krewire.dev/v0", Provider: "p"}); err == nil {
		t.Fatal("mismatched contract accepted")
	}
}

func TestValidateResource(t *testing.T) {
	s := KindSchema{
		Kind: "db", SchemaVersion: "v1", Required: []string{"engine"},
		Optional: []string{"dbPassword"}, SecretProps: []string{"password"},
	}
	ok := Resource{Kind: "db", Name: "a", SchemaVersion: "v1",
		Properties: map[string]any{"engine": "pg"},
		Secrets:    map[string]SecretRef{"password": {Source: "env", Key: "PW"}}}
	if p := s.ValidateResource(ok); len(p) != 0 {
		t.Fatalf("valid resource rejected: %v", p)
	}
	bad := Resource{Kind: "db", Name: "a", SchemaVersion: "v2",
		Properties: map[string]any{"dbPassword": "hunter2", "typo": 1},
		Secrets:    map[string]SecretRef{"other": {}}}
	got := strings.Join(s.ValidateResource(bad), "\n")
	for _, want := range []string{"schemaVersion", `"engine" is required`, `"typo" is not declared`, "looks sensitive", `secret "other"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "hunter2") {
		t.Fatal("validation leaked a sensitive value")
	}
}

func TestIsSensitiveName(t *testing.T) {
	for _, n := range []string{"DB_PASSWORD", "apiKey", "authToken", "private_key"} {
		if !IsSensitiveName(n) {
			t.Errorf("%s should be sensitive", n)
		}
	}
	if IsSensitiveName("path") {
		t.Error("path is not sensitive")
	}
	if (SecretRef{Source: "env", Key: "K"}).String() != "env:K" {
		t.Error("secret ref string")
	}
}
