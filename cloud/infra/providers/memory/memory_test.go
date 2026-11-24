package memory_test

import (
	"slices"
	"testing"

	"github.com/krewire/krewire/cloud/infra/conformance"
	"github.com/krewire/krewire/cloud/infra/contract"
	"github.com/krewire/krewire/cloud/infra/engine"
	"github.com/krewire/krewire/cloud/infra/providers/memory"
)

// Sample returns the desired state used by the memory provider tests.
func sample() []contract.Resource {
	return []contract.Resource{
		{Kind: memory.KindBucket, Name: "assets", SchemaVersion: "v1", Properties: map[string]any{"region": "ap-southeast-1"}},
		{Kind: memory.KindService, Name: "api", SchemaVersion: "v1", Properties: map[string]any{"image": "app:1", "replicas": 2},
			Secrets: map[string]contract.SecretRef{"token": {Source: "env", Key: "API_TOKEN"}}, DependsOn: []string{"bucket/assets"}},
		{Kind: memory.KindDNS, Name: "www", SchemaVersion: "v1", Properties: map[string]any{"target": "api"}, DependsOn: []string{"service/api"}},
	}
}

func TestConformance(t *testing.T) {
	conformance.Run(t, func(t *testing.T) conformance.Harness {
		p := memory.New()
		return conformance.Harness{
			Provider:  p,
			Resources: sample(),
			Secrets:   engine.MapSecrets{"env:API_TOKEN": "s3cr3t-value"},
			Mutate: func(rs []contract.Resource) []contract.Resource {
				out := slices.Clone(rs)
				out[1].Properties = map[string]any{"image": "app:2", "replicas": 2}
				return out
			},
			TamperAddress: "service/api",
			Tamper:        func(t *testing.T, addr string) { p.Tamper(addr, "image", "evil:latest") },
			RemoveAddress: "dns/www",
			Remove:        func(t *testing.T, addr string) { p.Remove(addr) },
		}
	})
}
