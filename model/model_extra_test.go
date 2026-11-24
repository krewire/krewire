package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrecedenceTier_String(t *testing.T) {
	tiers := []PrecedenceTier{
		PrecedenceToolchain,
		PrecedenceCoreKernel,
		PrecedenceWorkspaceRoot,
		PrecedenceProjectConfig,
		PrecedenceArchitecture,
		PrecedenceActiveTask,
		PrecedenceTier(99),
	}
	for _, pt := range tiers {
		str := pt.String()
		if str == "" {
			t.Errorf("expected non-empty string for PrecedenceTier %d", pt)
		}
	}
}

func TestDomainEvent_New(t *testing.T) {
	evt := NewDomainEvent("user.created", map[string]string{"id": "u1"})
	if evt.Type != "user.created" {
		t.Errorf("type = %s, want user.created", evt.Type)
	}
	if evt.Payload == nil {
		t.Errorf("payload should not be nil")
	}
	if evt.At.IsZero() {
		t.Errorf("timestamp should not be zero")
	}
}

func TestTier_ExtraCoverage(t *testing.T) {
	if !TierFree.Less(TierPro) {
		t.Errorf("TierFree should be Less than TierPro")
	}
	if TierEnterprise.Less(TierFree) {
		t.Errorf("TierEnterprise should not be Less than TierFree")
	}
	if TierPro.String() != "pro" {
		t.Errorf("TierPro.String() = %s, want pro", TierPro.String())
	}
	invalid := Tier("invalid")
	if invalid.Rank() != -1 {
		t.Errorf("invalid tier rank = %d, want -1", invalid.Rank())
	}
}

func TestEvidence_ExtraCoverage(t *testing.T) {
	// Empty gates
	ok, msg := EvaluateTaskVerification(nil)
	if ok || !strings.Contains(msg, "no verification gates") {
		t.Errorf("expected failure for nil gates, got %v: %s", ok, msg)
	}

	// GateBlocked required
	gatesBlocked := []VerificationEvidence{
		{GateID: "sec", Command: "sec-check", Required: true, Status: GateBlocked},
	}
	ok, msg = EvaluateTaskVerification(gatesBlocked)
	if ok || !strings.Contains(msg, "was blocked") {
		t.Errorf("expected failure for blocked gate, got %v: %s", ok, msg)
	}

	// GatePlanned required
	gatesPlanned := []VerificationEvidence{
		{GateID: "fmt", Command: "fmt-check", Required: true, Status: GatePlanned},
	}
	ok, msg = EvaluateTaskVerification(gatesPlanned)
	if ok || !strings.Contains(msg, "was not completed") {
		t.Errorf("expected failure for planned gate, got %v: %s", ok, msg)
	}

	// GateAttempted required
	gatesAttempted := []VerificationEvidence{
		{GateID: "test", Command: "test-run", Required: true, Status: GateAttempted},
	}
	ok, msg = EvaluateTaskVerification(gatesAttempted)
	if ok || !strings.Contains(msg, "was not completed") {
		t.Errorf("expected failure for attempted gate, got %v: %s", ok, msg)
	}

	// NewTaskVerificationRecord
	rec := NewTaskVerificationRecord("task-1", "agent-1", "commit-1", true, []VerificationEvidence{
		{GateID: "g1", Command: "c1", Required: true, Status: GatePassed},
	})
	if rec.TaskID != "task-1" || !rec.OverallVerified {
		t.Errorf("unexpected record: %+v", rec)
	}
}

func TestLedger_ExtraCoverage(t *testing.T) {
	empty := NewVerificationLedger("TASK-EMPTY")
	if _, err := empty.LatestBlock(); err != ErrEmptyLedger {
		t.Errorf("LatestBlock on empty ledger = %v, want ErrEmptyLedger", err)
	}
	md := empty.ExportMarkdown()
	if !strings.Contains(md, "Empty ledger") {
		t.Errorf("expected Empty ledger in markdown: %s", md)
	}

	// Broken genesis block
	brokenGenesis := NewVerificationLedger("TASK-BROKEN")
	b, _ := brokenGenesis.AppendBlock("c1", []VerificationEvidence{
		{GateID: "g1", Command: "c1", Required: true, Status: GatePassed},
	})
	b.PrevHash = "1111111111111111111111111111111111111111111111111111111111111111"
	b.BlockHash = b.ComputeHash()
	brokenGenesis.Blocks[0] = *b
	if ok, err := brokenGenesis.VerifyIntegrity(); ok || err == nil || !strings.Contains(err.Error(), "genesis block prev_hash invalid") {
		t.Errorf("expected genesis prev_hash invalid, got %v, err: %v", ok, err)
	}

	// Broken block 1 prev_hash
	ledger2 := NewVerificationLedger("TASK-BROKEN2")
	_, _ = ledger2.AppendBlock("c1", []VerificationEvidence{
		{GateID: "g1", Command: "c1", Required: true, Status: GatePassed},
	})
	b2, _ := ledger2.AppendBlock("c2", []VerificationEvidence{
		{GateID: "g2", Command: "c2", Required: true, Status: GatePassed},
	})
	b2.PrevHash = "2222222222222222222222222222222222222222222222222222222222222222"
	b2.BlockHash = b2.ComputeHash()
	ledger2.Blocks[1] = *b2
	if ok, err := ledger2.VerifyIntegrity(); ok || err == nil || !strings.Contains(err.Error(), "prev_hash mismatch") {
		t.Errorf("expected prev_hash mismatch, got %v, err: %v", ok, err)
	}

	// Markdown with failed block
	failedLedger := NewVerificationLedger("TASK-FAILED")
	_, _ = failedLedger.AppendBlock("c1", []VerificationEvidence{
		{GateID: "test", Command: "go test", Required: true, Status: GateFailed, ExitCode: 1},
	})
	mdFailed := failedLedger.ExportMarkdown()
	if !strings.Contains(mdFailed, "[FAILED]") {
		t.Errorf("expected [FAILED] in markdown: %s", mdFailed)
	}

	// SaveFile error
	if err := failedLedger.SaveFile("/proc/forbidden/ledger.json"); err == nil {
		t.Errorf("expected error saving to forbidden path")
	}

	// Load non-existent file
	if _, err := LoadLedgerFile("/nonexistent/ledger.json"); err == nil {
		t.Errorf("expected error loading non-existent ledger")
	}

	// Load corrupt file
	dir := t.TempDir()
	corruptFile := filepath.Join(dir, "corrupt.json")
	_ = os.WriteFile(corruptFile, []byte("{badjson"), 0o644)
	if _, err := LoadLedgerFile(corruptFile); err == nil {
		t.Errorf("expected error loading corrupt JSON")
	}

	// Load broken integrity ledger
	brokenFile := filepath.Join(dir, "broken.json")
	_ = ledger2.SaveFile(brokenFile)
	if _, err := LoadLedgerFile(brokenFile); err == nil {
		t.Errorf("expected error loading ledger with broken integrity")
	}
}

func TestTaskBrief_CheckScope_Extra(t *testing.T) {
	briefEmpty := TaskBrief{}
	if briefEmpty.CheckScope("foo.go") {
		t.Errorf("empty scope must reject all")
	}

	briefWildcard := TaskBrief{Scope: []string{"."}}
	if !briefWildcard.CheckScope("model/project.go") {
		t.Errorf("wildcard scope . should allow all")
	}

	briefStar := TaskBrief{Scope: []string{"*"}}
	if !briefStar.CheckScope("any.go") {
		t.Errorf("wildcard scope * should allow all")
	}

	briefEmptyScope := TaskBrief{Scope: []string{""}}
	if !briefEmptyScope.CheckScope("any.go") {
		t.Errorf("empty string in scope should allow all")
	}

	briefExact := TaskBrief{Scope: []string{"model/project.go"}}
	if !briefExact.CheckScope("model/project.go") {
		t.Errorf("exact match should succeed")
	}
	if briefExact.CheckScope("model/other.go") {
		t.Errorf("different file should fail")
	}

	briefDir := TaskBrief{Scope: []string{"model"}}
	if !briefDir.CheckScope("model/project.go") {
		t.Errorf("subpath match should succeed")
	}
}
