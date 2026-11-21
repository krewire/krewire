package model

import (
	"testing"
)

func TestVerificationLedger_ChainingAndIntegrity(t *testing.T) {
	ledger := NewVerificationLedger("TASK-100")

	// Initially empty
	if _, err := ledger.VerifyIntegrity(); err != ErrEmptyLedger {
		t.Fatalf("expected ErrEmptyLedger on empty ledger, got %v", err)
	}

	// Append block 1 (passing)
	gates1 := []VerificationEvidence{
		{
			GateID:       "fmt",
			Type:         GateTypeFormat,
			Command:      "gofmt -l .",
			Required:     true,
			Status:       GatePassed,
			ExitCode:     0,
			EvidenceHash: "hash1",
		},
		{
			GateID:       "vet",
			Type:         GateTypeStatic,
			Command:      "go vet ./...",
			Required:     true,
			Status:       GatePassed,
			ExitCode:     0,
			EvidenceHash: "hash2",
		},
	}

	b1, err := ledger.AppendBlock("commit-aaa", gates1)
	if err != nil {
		t.Fatalf("failed to append block 1: %v", err)
	}
	if b1.Index != 0 {
		t.Errorf("expected block 1 index 0, got %d", b1.Index)
	}
	if !b1.AllPassed {
		t.Errorf("expected block 1 AllPassed to be true")
	}

	// Append block 2 (failing test)
	gates2 := []VerificationEvidence{
		{
			GateID:       "test",
			Type:         GateTypeTest,
			Command:      "go test ./...",
			Required:     true,
			Status:       GateFailed,
			ExitCode:     1,
			EvidenceHash: "hash3",
		},
	}
	b2, err := ledger.AppendBlock("commit-bbb", gates2)
	if err != nil {
		t.Fatalf("failed to append block 2: %v", err)
	}
	if b2.Index != 1 {
		t.Errorf("expected block 2 index 1, got %d", b2.Index)
	}
	if b2.AllPassed {
		t.Errorf("expected block 2 AllPassed to be false")
	}
	if b2.PrevHash != b1.BlockHash {
		t.Errorf("expected block 2 PrevHash %s == block 1 BlockHash %s", b2.PrevHash, b1.BlockHash)
	}

	// Verify integrity
	ok, err := ledger.VerifyIntegrity()
	if !ok || err != nil {
		t.Fatalf("expected ledger integrity to pass, got ok=%v, err=%v", ok, err)
	}

	// Tampering detection: alter b1 exit code
	ledger.Blocks[0].Gates[0].ExitCode = 999
	ok, err = ledger.VerifyIntegrity()
	if ok || err == nil {
		t.Fatalf("expected tampering to be detected, got ok=%v, err=%v", ok, err)
	}

	// Restore and alter prev hash in b2
	ledger.Blocks[0].Gates[0].ExitCode = 0
	ledger.Blocks[1].PrevHash = "tampered-prev-hash"
	ok, err = ledger.VerifyIntegrity()
	if ok || err == nil {
		t.Fatalf("expected broken chain to be detected, got ok=%v, err=%v", ok, err)
	}
}

func TestVerificationLedger_LatestBlock(t *testing.T) {
	ledger := NewVerificationLedger("TASK-200")
	if _, err := ledger.LatestBlock(); err != ErrEmptyLedger {
		t.Fatalf("expected ErrEmptyLedger, got %v", err)
	}

	ledger.AppendBlock("sha-1", []VerificationEvidence{
		{GateID: "fmt", Status: GatePassed, Required: true},
	})
	b, err := ledger.LatestBlock()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.CommitSHA != "sha-1" {
		t.Errorf("expected CommitSHA sha-1, got %s", b.CommitSHA)
	}
}
