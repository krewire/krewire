package model

import (
	"strings"
	"testing"
)

func TestEvaluateTaskVerification_AllPassed(t *testing.T) {
	gates := []VerificationEvidence{
		{GateID: "fmt", Command: "gofmt -l .", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "vet", Command: "go vet ./...", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "test", Command: "go test ./...", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "build", Command: "CGO_ENABLED=0 go build ./...", Required: true, Status: GatePassed, ExitCode: 0},
	}

	verified, summary := EvaluateTaskVerification(gates)
	if !verified {
		t.Fatalf("expected OverallVerified=true when all required gates pass, got false: %s", summary)
	}
	if !strings.Contains(summary, "all 4 required verification gates passed") {
		t.Errorf("unexpected summary: %s", summary)
	}
}

func TestEvaluateTaskVerification_FailedRequiredGateRejects(t *testing.T) {
	gates := []VerificationEvidence{
		{GateID: "fmt", Command: "gofmt -l .", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "vet", Command: "go vet ./...", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "test", Command: "go test ./...", Required: true, Status: GateFailed, ExitCode: 1, Output: "panic: nil pointer"},
		{GateID: "build", Command: "CGO_ENABLED=0 go build ./...", Required: true, Status: GatePassed, ExitCode: 0},
	}

	verified, summary := EvaluateTaskVerification(gates)
	if verified {
		t.Fatalf("security violation: OverallVerified was true despite failed test gate")
	}
	if !strings.Contains(summary, "test (go test ./...) failed with exit code 1") {
		t.Errorf("summary missing failure detail: %s", summary)
	}
}

func TestEvaluateTaskVerification_SkippedRequiredGateRejects(t *testing.T) {
	gates := []VerificationEvidence{
		{GateID: "fmt", Command: "gofmt -l .", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "vet", Command: "go vet ./...", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "test", Command: "go test ./...", Required: true, Status: GateSkipped, ExitCode: 0},
	}

	verified, summary := EvaluateTaskVerification(gates)
	if verified {
		t.Fatalf("security violation: OverallVerified was true despite skipped required gate")
	}
	if !strings.Contains(summary, "was skipped despite being required") {
		t.Errorf("summary missing skipped detail: %s", summary)
	}
}

func TestEvaluateTaskVerification_OptionalGateSkippedAllowed(t *testing.T) {
	gates := []VerificationEvidence{
		{GateID: "fmt", Command: "gofmt -l .", Required: true, Status: GatePassed, ExitCode: 0},
		{GateID: "bench", Command: "go test -bench .", Required: false, Status: GateSkipped},
	}

	verified, summary := EvaluateTaskVerification(gates)
	if !verified {
		t.Fatalf("expected true when only optional gate is skipped, got: %s", summary)
	}
}

func TestVerificationEvidence_ComputeHash(t *testing.T) {
	evidence := VerificationEvidence{
		GateID:   "gate-01",
		Command:  "go test ./...",
		ExitCode: 0,
		Status:   GatePassed,
		Output:   "PASS\nok",
	}

	hash1 := evidence.ComputeHash("commit-1234")
	hash2 := evidence.ComputeHash("commit-1234")
	hashDifferentCommit := evidence.ComputeHash("commit-5678")

	if len(hash1) != 64 {
		t.Fatalf("expected SHA-256 hash length 64, got %d (%s)", len(hash1), hash1)
	}
	if hash1 != hash2 {
		t.Errorf("identical evidence produced differing hashes: %s != %s", hash1, hash2)
	}
	if hash1 == hashDifferentCommit {
		t.Errorf("differing commits must produce differing hashes")
	}
}
