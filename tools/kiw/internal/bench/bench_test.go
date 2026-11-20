package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/pkg/kiw"
	"github.com/krewire/krewire/pkg/markdown"
	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/tools/kiw/internal/config"
)

// ── 1. Unit Tests for Inventory & Regression ────────────────────────────────

func TestCanonicalInventory_Valid(t *testing.T) {
	if len(CanonicalInventory) == 0 {
		t.Fatal("CanonicalInventory must not be empty")
	}

	seen := make(map[string]bool)
	for _, item := range CanonicalInventory {
		if item.ID == "" {
			t.Errorf("empty benchmark item ID: %+v", item)
		}
		if seen[item.ID] {
			t.Errorf("duplicate benchmark ID: %s", item.ID)
		}
		seen[item.ID] = true

		if item.Name == "" || item.Fixture == "" || item.Rationale == "" {
			t.Errorf("item %s has incomplete inventory metadata: %+v", item.ID, item)
		}
	}
}

func TestRegressionPolicy_Detection(t *testing.T) {
	policy := DefaultPolicy()

	base := &BenchmarkReport{
		Results: []BenchmarkResult{
			{ID: "CORE-CFG-001", Name: "Config Load", NsPerOp: 100_000, BytesPerOp: 10_000},
			{ID: "CORE-KIW-001", Name: "Kiw Parse", NsPerOp: 200_000, BytesPerOp: 20_000},
		},
	}

	// 1. Unchanged: no alerts
	currUnchanged := &BenchmarkReport{
		Results: []BenchmarkResult{
			{ID: "CORE-CFG-001", Name: "Config Load", NsPerOp: 105_000, BytesPerOp: 10_500},
			{ID: "CORE-KIW-001", Name: "Kiw Parse", NsPerOp: 202_000, BytesPerOp: 20_100},
		},
	}
	alerts := CompareReports(base, currUnchanged, policy)
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts on normal variance, got %d: %+v", len(alerts), alerts)
	}

	// 2. 20% latency increase: warning alert
	currWarn := &BenchmarkReport{
		Results: []BenchmarkResult{
			{ID: "CORE-CFG-001", Name: "Config Load", NsPerOp: 120_000, BytesPerOp: 10_000}, // +20%
			{ID: "CORE-KIW-001", Name: "Kiw Parse", NsPerOp: 200_000, BytesPerOp: 20_000},
		},
	}
	alertsWarn := CompareReports(base, currWarn, policy)
	if len(alertsWarn) != 1 || alertsWarn[0].Severity != SeverityWarning {
		t.Errorf("expected 1 warning alert, got: %+v", alertsWarn)
	}

	// 3. 50% latency increase: critical alert
	currCrit := &BenchmarkReport{
		Results: []BenchmarkResult{
			{ID: "CORE-CFG-001", Name: "Config Load", NsPerOp: 160_000, BytesPerOp: 10_000}, // +60%
		},
	}
	alertsCrit := CompareReports(base, currCrit, policy)
	if len(alertsCrit) != 1 || alertsCrit[0].Severity != SeverityCritical {
		t.Errorf("expected 1 critical alert, got: %+v", alertsCrit)
	}
}

// ── 2. Standard Go Benchmark Primitives (go test -bench=.) ──────────────────

func BenchmarkCore_ConfigLoad(b *testing.B) {
	dir := b.TempDir()
	cfgData := `project:
  name: benchmark-site
  kind: site
  module: example.com/site
plugins:
  - name: test-plugin
    version: "1.0.0"
`
	if err := os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte(cfgData), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = config.Load(dir)
	}
}

func BenchmarkCore_ModelResolve(b *testing.B) {
	p := model.Project{
		Name:       "benchmark-app",
		Kind:       model.KindSite,
		ModulePath: "github.com/krewire/krewire",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = p.Validate()
		_, _ = model.ParseKind("site")
	}
}

func BenchmarkCore_KiwParse(b *testing.B) {
	fixture := `---
title: Benchmark Component
layout: Base
props:
  headline: { type: string, default: "Benchmark" }
  count: { type: number, default: 100 }
---
<script lang="ts">export let count: number = 100;</script>
<style scoped>.metric { font-family: monospace; }</style>
<div class="metric">
  <Header Title={headline} />
  {#if count > 50}<span>High</span>{:else}<span>Low</span>{/if}
  {#each items as item}<div>{item.title}</div>{/each}
</div>`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = kiw.ParseKiw(fixture)
	}
}

func BenchmarkCore_KiwDesugar(b *testing.B) {
	fixture := `
<section>
  {#if active}
    {#each items as item, index}
      <div class:current={item.isCurrent}>
        <span>{index}: {item.title}</span>
      </div>
    {/each}
  {:else}
    <p>Inactive</p>
  {/if}
</section>`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = kiw.DesugarTemplate(fixture)
	}
}

func BenchmarkCore_MarkdownRender(b *testing.B) {
	doc := []byte(`# Benchmark Documentation
Technical documentation benchmark testing rendering speed and allocations.
- First item
- Second item with **bold text** and ` + "`code inline`" + `
` + "```go\nfunc hello() {\n\tprintln(\"benchmark\")\n}\n```")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = markdown.Render(doc)
	}
}
