package localfile_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/krewire/krewire/cloud/infra/conformance"
	"github.com/krewire/krewire/cloud/infra/contract"
	"github.com/krewire/krewire/cloud/infra/engine"
	"github.com/krewire/krewire/cloud/infra/providers/localfile"
)

func sample() []contract.Resource {
	return []contract.Resource{
		{Kind: localfile.KindDirectory, Name: "conf", SchemaVersion: "v1", Properties: map[string]any{"path": "conf"}},
		{Kind: localfile.KindFile, Name: "app", SchemaVersion: "v1",
			Properties: map[string]any{"path": "conf/app.env", "content": "A=1\n", "mode": "0640"}, DependsOn: []string{"directory/conf"}},
		{Kind: localfile.KindFile, Name: "token", SchemaVersion: "v1", Properties: map[string]any{"path": "conf/token"},
			Secrets: map[string]contract.SecretRef{"content": {Source: "env", Key: "TOKEN"}}, DependsOn: []string{"directory/conf"}},
	}
}

func TestConformance(t *testing.T) {
	conformance.Run(t, func(t *testing.T) conformance.Harness {
		root := t.TempDir()
		return conformance.Harness{
			Provider:  localfile.New(root),
			Resources: sample(),
			Secrets:   engine.MapSecrets{"env:TOKEN": "tok-123"},
			Mutate: func(rs []contract.Resource) []contract.Resource {
				out := slices.Clone(rs)
				out[1].Properties = map[string]any{"path": "conf/app.env", "content": "A=2\n", "mode": "0640"}
				return out
			},
			TamperAddress: "file/app",
			Tamper: func(t *testing.T, _ string) {
				if err := os.WriteFile(filepath.Join(root, "conf/app.env"), []byte("HACKED"), 0o640); err != nil {
					t.Fatal(err)
				}
			},
			RemoveAddress: "file/token",
			Remove: func(t *testing.T, _ string) {
				if err := os.Remove(filepath.Join(root, "conf/token")); err != nil {
					t.Fatal(err)
				}
			},
		}
	})
}

func TestValidationRejectsUnsafePaths(t *testing.T) {
	p := localfile.New(t.TempDir())
	schema, _ := p.Capabilities().Kind(localfile.KindFile)
	for _, props := range []map[string]any{
		{"path": "../escape"}, {"path": "/etc/passwd"}, {"path": "ok", "mode": "999"}, {"path": "ok", "content": 1},
	} {
		r := contract.Resource{Kind: localfile.KindFile, Name: "x", SchemaVersion: "v1", Properties: props}
		if len(schema.ValidateResource(r)) == 0 {
			t.Errorf("accepted %v", props)
		}
	}
	if _, err := p.Read(context.Background(), contract.ResourceState{Resource: contract.Resource{
		Kind: localfile.KindFile, Properties: map[string]any{"path": "../x"}}}); err == nil {
		t.Error("read outside root accepted")
	}
}
