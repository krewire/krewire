package sec

import (
	"testing"
	"time"
)

func TestApprovalSignal_ValidationAndVerification(t *testing.T) {
	secret := "krewire-control-secret-12345"
	approver := "user-lead-42"
	scope := "workload:deploy"

	// 1. Success case
	sig, err := NewApprovalSignal(approver, scope, secret, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error creating approval signal: %v", err)
	}

	if !sig.IsValid() {
		t.Fatal("expected newly generated signal to be valid")
	}

	if !sig.VerifyToken(secret) {
		t.Fatal("expected token verification to succeed with correct secret")
	}

	// 2. Wrong secret
	if sig.VerifyToken("wrong-secret-key") {
		t.Fatal("expected token verification to fail with wrong secret")
	}

	// 3. Tampered scope
	tampered := *sig
	tampered.Scope = "admin:root"
	if tampered.VerifyToken(secret) {
		t.Fatal("expected token verification to fail for tampered scope")
	}

	// 4. Tampered approver
	tamperedApprover := *sig
	tamperedApprover.ApproverID = "attacker"
	if tamperedApprover.VerifyToken(secret) {
		t.Fatal("expected token verification to fail for tampered approver")
	}

	// 5. Expired signal
	expiredSig := *sig
	expiredSig.ValidUntil = time.Now().UTC().Add(-1 * time.Minute)
	if expiredSig.IsValid() {
		t.Fatal("expected expired signal to be invalid")
	}
	if expiredSig.VerifyToken(secret) {
		t.Fatal("expected expired signal token verification to fail")
	}

	// 6. Nil or empty signal
	var nilSig *ApprovalSignal
	if nilSig.IsValid() {
		t.Fatal("expected nil signal to be invalid")
	}
	if nilSig.VerifyToken(secret) {
		t.Fatal("expected nil signal verify token to return false")
	}

	emptySig := &ApprovalSignal{}
	if emptySig.IsValid() {
		t.Fatal("expected empty signal to be invalid")
	}
}

func TestNewApprovalSignal_ValidationErrors(t *testing.T) {
	secret := "secret-key"

	if _, err := NewApprovalSignal("", "scope", secret, time.Minute); err != ErrMissingApproverID {
		t.Fatalf("expected ErrMissingApproverID, got %v", err)
	}

	if _, err := NewApprovalSignal("approver", "", secret, time.Minute); err != ErrMissingApprovalScope {
		t.Fatalf("expected ErrMissingApprovalScope, got %v", err)
	}

	if _, err := NewApprovalSignal("approver", "scope", "", time.Minute); err != ErrMissingSecretKey {
		t.Fatalf("expected ErrMissingSecretKey, got %v", err)
	}

	if _, err := NewApprovalSignal("approver", "scope", secret, -1*time.Minute); err != ErrInvalidApprovalTTL {
		t.Fatalf("expected ErrInvalidApprovalTTL, got %v", err)
	}
}
