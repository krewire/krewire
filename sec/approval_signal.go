package sec

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// MinSecretKeyLength is the minimum acceptable secret length in bytes (128-bit minimum).
	MinSecretKeyLength = 16

	// BlockedFallbackSecret is the former insecure constant fallback, strictly rejected.
	BlockedFallbackSecret = "dash-ephemeral-approval-secret"
)

var (
	// ErrInvalidApprovalSignal is returned when an approval signal is nil, malformed, or expired.
	ErrInvalidApprovalSignal = errors.New("sec: invalid or expired approval signal")

	// ErrMissingApproverID is returned when an approval signal does not specify an approver.
	ErrMissingApproverID = errors.New("sec: approver ID is required")

	// ErrMissingApprovalScope is returned when an approval signal does not specify a scope.
	ErrMissingApprovalScope = errors.New("sec: approval scope is required")

	// ErrMissingSecretKey is returned when token generation or verification lacks a secret key.
	ErrMissingSecretKey = errors.New("sec: secret key is required for approval signal authentication")

	// ErrWeakSecretKey is returned when a secret key has fewer than MinSecretKeyLength bytes.
	ErrWeakSecretKey = errors.New("sec: secret key must be at least 16 bytes of entropy")

	// ErrInsecureFallbackSecret is returned when attempting to use the former hard-coded fallback.
	ErrInsecureFallbackSecret = errors.New("sec: hard-coded fallback secret is rejected")

	// ErrInvalidApprovalTTL is returned when an approval signal TTL is negative.
	ErrInvalidApprovalTTL = errors.New("sec: approval signal TTL must be positive")

	// ErrApprovalExpired is returned when an approval signal validity window has expired.
	ErrApprovalExpired = errors.New("sec: approval signal has expired")

	// ErrApprovalReplayed is returned when an approval signal token or nonce has already been consumed.
	ErrApprovalReplayed = errors.New("sec: approval signal has already been used (replay detected)")

	// ErrApprovalScopeMismatch is returned when an approval signal scope does not match the target scope.
	ErrApprovalScopeMismatch = errors.New("sec: approval scope mismatch")

	// ErrApprovalActionHashMismatch is returned when an approval signal action hash does not match.
	ErrApprovalActionHashMismatch = errors.New("sec: approval action hash mismatch")

	// ErrUnknownKeyID is returned when a signal's key ID is not found in the key ring.
	ErrUnknownKeyID = errors.New("sec: unknown key id in key ring")

	// ErrApprovalSignatureMismatch is returned when HMAC verification fails.
	ErrApprovalSignatureMismatch = errors.New("sec: approval signature verification failed")
)

// ValidateSecretKey checks that a signing key is present, has sufficient entropy,
// and is not the insecure former fallback secret.
func ValidateSecretKey(key string) error {
	if key == "" {
		return ErrMissingSecretKey
	}
	if subtle.ConstantTimeCompare([]byte(key), []byte(BlockedFallbackSecret)) == 1 {
		return ErrInsecureFallbackSecret
	}
	if len(key) < MinSecretKeyLength {
		return ErrWeakSecretKey
	}
	return nil
}

// ApprovalSignal represents an explicit human or control surface authorization
// signal (e.g. from Krewire Dash) approving a privileged write or mutating operation.
type ApprovalSignal struct {
	ApproverID string    `json:"approver_id"`
	ApprovedAt time.Time `json:"approved_at"`
	Token      string    `json:"token"`
	Scope      string    `json:"scope"`
	ValidUntil time.Time `json:"valid_until"`
	KeyID      string    `json:"key_id,omitempty"`
	Nonce      string    `json:"nonce,omitempty"`
	ActionHash string    `json:"action_hash,omitempty"`
}

// ApprovalOptions defines options for constructing an ApprovalSignal.
type ApprovalOptions struct {
	ApproverID string
	Scope      string
	SecretKey  string
	KeyID      string
	Nonce      string
	ActionHash string
	TTL        time.Duration
}

// IsValid checks whether the approval signal is active, well-formed, and unexpired.
func (s *ApprovalSignal) IsValid() bool {
	if s == nil || s.ApproverID == "" || s.Token == "" {
		return false
	}
	if !s.ValidUntil.IsZero() && time.Now().UTC().After(s.ValidUntil) {
		return false
	}
	return true
}

// payload forms the canonical message payload for HMAC signing.
func (s *ApprovalSignal) payload() string {
	if s.KeyID == "" && s.Nonce == "" && s.ActionHash == "" {
		return fmt.Sprintf("%s:%s:%d:%d", s.ApproverID, s.Scope, s.ApprovedAt.Unix(), s.ValidUntil.Unix())
	}
	return fmt.Sprintf("v2:%s:%s:%s:%s:%s:%d:%d", s.KeyID, s.ApproverID, s.Scope, s.ActionHash, s.Nonce, s.ApprovedAt.Unix(), s.ValidUntil.Unix())
}

// NewApprovalSignalWithOptions creates a cryptographically authenticated approval signal with explicit options.
func NewApprovalSignalWithOptions(opts ApprovalOptions) (*ApprovalSignal, error) {
	if opts.ApproverID == "" {
		return nil, ErrMissingApproverID
	}
	if opts.Scope == "" {
		return nil, ErrMissingApprovalScope
	}
	if opts.SecretKey == "" {
		return nil, ErrMissingSecretKey
	}
	if opts.TTL < 0 {
		return nil, ErrInvalidApprovalTTL
	}
	if err := ValidateSecretKey(opts.SecretKey); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	ttl := opts.TTL
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	validUntil := now.Add(ttl)

	nonce := opts.Nonce
	if nonce == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("sec: failed to generate nonce: %w", err)
		}
		nonce = hex.EncodeToString(b)
	}

	sig := &ApprovalSignal{
		ApproverID: opts.ApproverID,
		ApprovedAt: now,
		Scope:      opts.Scope,
		ValidUntil: validUntil,
		KeyID:      opts.KeyID,
		Nonce:      nonce,
		ActionHash: opts.ActionHash,
	}

	h := hmac.New(sha256.New, []byte(opts.SecretKey))
	h.Write([]byte(sig.payload()))
	sig.Token = hex.EncodeToString(h.Sum(nil))

	return sig, nil
}

// NewApprovalSignal creates a cryptographically authenticated approval signal with an HMAC-SHA256 token.
func NewApprovalSignal(approverID, scope, secretKey string, ttl time.Duration) (*ApprovalSignal, error) {
	if approverID == "" {
		return nil, ErrMissingApproverID
	}
	if scope == "" {
		return nil, ErrMissingApprovalScope
	}
	if secretKey == "" {
		return nil, ErrMissingSecretKey
	}
	if ttl < 0 {
		return nil, ErrInvalidApprovalTTL
	}
	if err := ValidateSecretKey(secretKey); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	validUntil := now.Add(ttl)

	sig := &ApprovalSignal{
		ApproverID: approverID,
		ApprovedAt: now,
		Scope:      scope,
		ValidUntil: validUntil,
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(sig.payload()))
	sig.Token = hex.EncodeToString(h.Sum(nil))

	return sig, nil
}

// VerifyToken cryptographically verifies the authenticity and integrity of the approval token.
func (s *ApprovalSignal) VerifyToken(secretKey string) bool {
	if !s.IsValid() || secretKey == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(secretKey), []byte(BlockedFallbackSecret)) == 1 {
		return false
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(s.payload()))
	expectedMAC := h.Sum(nil)

	actualMAC, err := hex.DecodeString(s.Token)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(actualMAC, expectedMAC) == 1
}

// VerifySignalOptions configures verification criteria for an ApprovalSignal.
type VerifySignalOptions struct {
	SecretKey          string
	KeyRing            *KeyRing
	ExpectedScope      string
	ExpectedActionHash string
	Tracker            NonceTracker
}

// VerifyWithOptions performs strict cryptographic verification against scope, action hash, key ring, and replay tracker.
func (s *ApprovalSignal) VerifyWithOptions(opts VerifySignalOptions) error {
	if s == nil {
		return ErrInvalidApprovalSignal
	}
	if !s.IsValid() {
		return ErrApprovalExpired
	}

	if opts.ExpectedScope != "" {
		if subtle.ConstantTimeCompare([]byte(s.Scope), []byte(opts.ExpectedScope)) != 1 {
			return ErrApprovalScopeMismatch
		}
	}
	if opts.ExpectedActionHash != "" {
		if subtle.ConstantTimeCompare([]byte(s.ActionHash), []byte(opts.ExpectedActionHash)) != 1 {
			return ErrApprovalActionHashMismatch
		}
	}

	var secret string
	if opts.KeyRing != nil {
		keyID := s.KeyID
		if keyID == "" {
			var err error
			_, secret, err = opts.KeyRing.PrimaryKey()
			if err != nil {
				return err
			}
		} else {
			var ok bool
			secret, ok = opts.KeyRing.GetKey(keyID)
			if !ok {
				return ErrUnknownKeyID
			}
		}
	} else {
		secret = opts.SecretKey
	}

	if err := ValidateSecretKey(secret); err != nil {
		return err
	}

	if !s.VerifyToken(secret) {
		return ErrApprovalSignatureMismatch
	}

	if opts.Tracker != nil {
		replayKey := s.Nonce
		if replayKey == "" {
			replayKey = s.Token
		}
		if !opts.Tracker.CheckAndRecord(replayKey, s.ValidUntil) {
			return ErrApprovalReplayed
		}
	}

	return nil
}

// NonceTracker defines the interface for one-time replay resistance.
type NonceTracker interface {
	CheckAndRecord(tokenOrNonce string, expiresAt time.Time) bool
}

// MemoryNonceTracker is a thread-safe, in-memory implementation of NonceTracker.
type MemoryNonceTracker struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

// NewMemoryNonceTracker constructs an in-memory replay tracker.
func NewMemoryNonceTracker() *MemoryNonceTracker {
	return &MemoryNonceTracker{entries: make(map[string]time.Time)}
}

// CheckAndRecord returns true if the token or nonce has not been seen before,
// storing it until expiresAt. If already seen, returns false.
func (m *MemoryNonceTracker) CheckAndRecord(tokenOrNonce string, expiresAt time.Time) bool {
	if tokenOrNonce == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if len(m.entries) > 1000 {
		for k, exp := range m.entries {
			if now.After(exp) {
				delete(m.entries, k)
			}
		}
	}

	if exp, exists := m.entries[tokenOrNonce]; exists {
		if now.Before(exp) {
			return false
		}
		delete(m.entries, tokenOrNonce)
	}

	m.entries[tokenOrNonce] = expiresAt
	return true
}

// KeyRing manages secret keys for approval signal signing and verification,
// supporting key rotation via key identifiers.
type KeyRing struct {
	mu         sync.RWMutex
	primaryKey string
	keys       map[string]string
}

// NewKeyRing creates a key ring with a primary signing key.
func NewKeyRing(primaryID, primarySecret string) (*KeyRing, error) {
	if primaryID == "" {
		return nil, errors.New("sec: primary key ID is required")
	}
	if err := ValidateSecretKey(primarySecret); err != nil {
		return nil, err
	}
	return &KeyRing{
		primaryKey: primaryID,
		keys:       map[string]string{primaryID: primarySecret},
	}, nil
}

// AddKey adds a secondary key to the ring for verification of older signals.
func (kr *KeyRing) AddKey(id, secret string) error {
	if id == "" {
		return errors.New("sec: key ID is required")
	}
	if err := ValidateSecretKey(secret); err != nil {
		return err
	}
	kr.mu.Lock()
	defer kr.mu.Unlock()
	kr.keys[id] = secret
	return nil
}

// RotateKey sets a new primary key for signing while keeping older keys for verification.
func (kr *KeyRing) RotateKey(newID, newSecret string) error {
	if err := kr.AddKey(newID, newSecret); err != nil {
		return err
	}
	kr.mu.Lock()
	defer kr.mu.Unlock()
	kr.primaryKey = newID
	return nil
}

// PrimaryKey returns the current primary signing key ID and secret.
func (kr *KeyRing) PrimaryKey() (string, string, error) {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	sec, ok := kr.keys[kr.primaryKey]
	if !ok {
		return "", "", errors.New("sec: primary key not found in key ring")
	}
	return kr.primaryKey, sec, nil
}

// GetKey returns the secret for the specified key identifier.
func (kr *KeyRing) GetKey(id string) (string, bool) {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	sec, ok := kr.keys[id]
	return sec, ok
}

// CanonicalActionHash computes a deterministic SHA256 hex string binding a tool name and argument JSON.
// Map keys are canonicalized via sorted JSON encoding and leading/trailing whitespace is trimmed.
func CanonicalActionHash(toolName string, rawArgs []byte) string {
	cleanName := strings.ToLower(strings.TrimSpace(toolName))
	var raw any
	if len(rawArgs) > 0 && json.Unmarshal(rawArgs, &raw) == nil {
		canonical, err := json.Marshal(raw)
		if err == nil {
			rawArgs = canonical
		}
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", cleanName, string(rawArgs))))
	return hex.EncodeToString(h[:])
}

// NewApprovalSignalForTool creates an authenticated approval signal specifically bound to a tool and canonical arguments.
func NewApprovalSignalForTool(secretKey, approverID, scope, toolName string, rawArgs []byte, ttl time.Duration) (*ApprovalSignal, error) {
	actionHash := CanonicalActionHash(toolName, rawArgs)
	return NewApprovalSignalWithOptions(ApprovalOptions{
		ApproverID: approverID,
		Scope:      scope,
		SecretKey:  secretKey,
		ActionHash: actionHash,
		TTL:        ttl,
	})
}
