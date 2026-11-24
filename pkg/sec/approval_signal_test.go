package sec

import (
	"errors"
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
	if sig.VerifyToken("wrong-secret-key-999") {
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

func TestApprovalSignal_FormerFallbackRejected(t *testing.T) {
	fallbackSecret := "dash-ephemeral-approval-secret"

	// 1. ValidateSecretKey rejects former fallback
	if err := ValidateSecretKey(fallbackSecret); !errors.Is(err, ErrInsecureFallbackSecret) {
		t.Fatalf("expected ErrInsecureFallbackSecret, got %v", err)
	}

	// 2. NewApprovalSignal rejects former fallback
	if _, err := NewApprovalSignal("approver", "scope", fallbackSecret, time.Minute); !errors.Is(err, ErrInsecureFallbackSecret) {
		t.Fatalf("expected ErrInsecureFallbackSecret on creation, got %v", err)
	}

	// 3. NewApprovalSignalWithOptions rejects former fallback
	if _, err := NewApprovalSignalWithOptions(ApprovalOptions{
		ApproverID: "approver",
		Scope:      "scope",
		SecretKey:  fallbackSecret,
		TTL:        time.Minute,
	}); !errors.Is(err, ErrInsecureFallbackSecret) {
		t.Fatalf("expected ErrInsecureFallbackSecret on options creation, got %v", err)
	}

	// 4. VerifyToken rejects verification with fallback secret
	validSecret := "0123456789abcdef0123456789abcdef"
	sig, err := NewApprovalSignal("approver", "scope", validSecret, time.Minute)
	if err != nil {
		t.Fatalf("failed to create valid signal: %v", err)
	}
	if sig.VerifyToken(fallbackSecret) {
		t.Fatal("VerifyToken must reject verification with the former fallback secret")
	}

	// 5. VerifyWithOptions rejects verification with fallback secret
	if err := sig.VerifyWithOptions(VerifySignalOptions{SecretKey: fallbackSecret}); !errors.Is(err, ErrInsecureFallbackSecret) {
		t.Fatalf("expected ErrInsecureFallbackSecret in VerifyWithOptions, got %v", err)
	}

	// 6. KeyRing rejects fallback secret
	if _, err := NewKeyRing("k1", fallbackSecret); !errors.Is(err, ErrInsecureFallbackSecret) {
		t.Fatalf("expected NewKeyRing to reject fallback secret, got %v", err)
	}
}

func TestApprovalSignal_MinimumEntropyAndWeakKeyRejected(t *testing.T) {
	weakKeys := []string{
		"short",
		"123456789012345", // 15 bytes
	}
	for _, k := range weakKeys {
		if err := ValidateSecretKey(k); !errors.Is(err, ErrWeakSecretKey) {
			t.Errorf("expected ErrWeakSecretKey for key %q, got %v", k, err)
		}
		if _, err := NewApprovalSignal("approver", "scope", k, time.Minute); !errors.Is(err, ErrWeakSecretKey) {
			t.Errorf("expected NewApprovalSignal to reject weak key %q, got %v", k, err)
		}
	}
}

func TestApprovalSignal_ReplayResistance(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	tracker := NewMemoryNonceTracker()

	sig, err := NewApprovalSignalWithOptions(ApprovalOptions{
		ApproverID: "lead",
		Scope:      "dash:ws1:task1",
		SecretKey:  secret,
		ActionHash: "hash-val-123",
		TTL:        5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewApprovalSignalWithOptions: %v", err)
	}
	if sig.Nonce == "" {
		t.Fatal("expected auto-generated nonce")
	}

	// First verification must succeed
	if err := sig.VerifyWithOptions(VerifySignalOptions{
		SecretKey:          secret,
		ExpectedScope:      "dash:ws1:task1",
		ExpectedActionHash: "hash-val-123",
		Tracker:            tracker,
	}); err != nil {
		t.Fatalf("first verification failed: %v", err)
	}

	// Second verification with the same tracker must fail due to replay detection
	if err := sig.VerifyWithOptions(VerifySignalOptions{
		SecretKey:          secret,
		ExpectedScope:      "dash:ws1:task1",
		ExpectedActionHash: "hash-val-123",
		Tracker:            tracker,
	}); !errors.Is(err, ErrApprovalReplayed) {
		t.Fatalf("expected ErrApprovalReplayed, got %v", err)
	}
}

func TestApprovalSignal_ScopeAndActionHashBinding(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	sig, err := NewApprovalSignalWithOptions(ApprovalOptions{
		ApproverID: "reviewer",
		Scope:      "dash:ten1:ws1:task1",
		SecretKey:  secret,
		ActionHash: "sha256:immutable-diff-hash",
		TTL:        5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to create signal: %v", err)
	}

	// 1. Success matching both
	if err := sig.VerifyWithOptions(VerifySignalOptions{
		SecretKey:          secret,
		ExpectedScope:      "dash:ten1:ws1:task1",
		ExpectedActionHash: "sha256:immutable-diff-hash",
	}); err != nil {
		t.Fatalf("verification should succeed: %v", err)
	}

	// 2. Scope mismatch
	if err := sig.VerifyWithOptions(VerifySignalOptions{
		SecretKey:          secret,
		ExpectedScope:      "dash:ten1:ws1:other-task",
		ExpectedActionHash: "sha256:immutable-diff-hash",
	}); !errors.Is(err, ErrApprovalScopeMismatch) {
		t.Fatalf("expected ErrApprovalScopeMismatch, got %v", err)
	}

	// 3. Action hash mismatch
	if err := sig.VerifyWithOptions(VerifySignalOptions{
		SecretKey:          secret,
		ExpectedScope:      "dash:ten1:ws1:task1",
		ExpectedActionHash: "sha256:tampered-diff-hash",
	}); !errors.Is(err, ErrApprovalActionHashMismatch) {
		t.Fatalf("expected ErrApprovalActionHashMismatch, got %v", err)
	}
}

func TestApprovalSignal_KeyRotation(t *testing.T) {
	key1 := "0123456789abcdef0123456789abcdef"
	key2 := "fedcba9876543210fedcba9876543210"

	ring, err := NewKeyRing("key-2026-v1", key1)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}

	// Issue signal under key-2026-v1
	sig1, err := NewApprovalSignalWithOptions(ApprovalOptions{
		ApproverID: "approver-1",
		Scope:      "dash:ws1:task1",
		KeyID:      "key-2026-v1",
		SecretKey:  key1,
		TTL:        10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewApprovalSignalWithOptions sig1: %v", err)
	}

	if err := sig1.VerifyWithOptions(VerifySignalOptions{KeyRing: ring}); err != nil {
		t.Fatalf("verify sig1 with ring: %v", err)
	}

	// Rotate key to key-2026-v2
	if err := ring.RotateKey("key-2026-v2", key2); err != nil {
		t.Fatalf("RotateKey: %v", err)
	}

	primaryID, primarySec, err := ring.PrimaryKey()
	if err != nil || primaryID != "key-2026-v2" || primarySec != key2 {
		t.Fatalf("expected rotated primary key 'key-2026-v2', got id=%q, err=%v", primaryID, err)
	}

	// Verify that sig1 (signed with old key-2026-v1) still validates during the rotation grace window
	if err := sig1.VerifyWithOptions(VerifySignalOptions{KeyRing: ring}); err != nil {
		t.Fatalf("verify sig1 after rotation: %v", err)
	}

	// Issue new signal with key-2026-v2
	sig2, err := NewApprovalSignalWithOptions(ApprovalOptions{
		ApproverID: "approver-1",
		Scope:      "dash:ws1:task1",
		KeyID:      "key-2026-v2",
		SecretKey:  key2,
		TTL:        10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewApprovalSignalWithOptions sig2: %v", err)
	}
	if err := sig2.VerifyWithOptions(VerifySignalOptions{KeyRing: ring}); err != nil {
		t.Fatalf("verify sig2 with ring: %v", err)
	}

	// Signal referencing unknown key ID fails
	sigUnknownKey := *sig2
	sigUnknownKey.KeyID = "key-nonexistent"
	if err := sigUnknownKey.VerifyWithOptions(VerifySignalOptions{KeyRing: ring}); !errors.Is(err, ErrUnknownKeyID) {
		t.Fatalf("expected ErrUnknownKeyID, got %v", err)
	}
}
