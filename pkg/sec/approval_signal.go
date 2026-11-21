package sec

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
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

	// ErrInvalidApprovalTTL is returned when an approval signal TTL is negative.
	ErrInvalidApprovalTTL = errors.New("sec: approval signal TTL must be positive")
)

// ApprovalSignal represents an explicit human or control surface authorization
// signal (e.g. from Krewire Dash) approving a privileged write or mutating operation.
type ApprovalSignal struct {
	ApproverID string    `json:"approver_id"`
	ApprovedAt time.Time `json:"approved_at"`
	Token      string    `json:"token"`
	Scope      string    `json:"scope"`
	ValidUntil time.Time `json:"valid_until"`
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

// computeApprovalPayload forms the canonical message payload for HMAC signing.
func computeApprovalPayload(approverID, scope string, approvedAt, validUntil time.Time) string {
	return fmt.Sprintf("%s:%s:%d:%d", approverID, scope, approvedAt.Unix(), validUntil.Unix())
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

	now := time.Now().UTC()
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	validUntil := now.Add(ttl)

	payload := computeApprovalPayload(approverID, scope, now, validUntil)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(payload))
	token := hex.EncodeToString(h.Sum(nil))

	return &ApprovalSignal{
		ApproverID: approverID,
		ApprovedAt: now,
		Token:      token,
		Scope:      scope,
		ValidUntil: validUntil,
	}, nil
}

// VerifyToken cryptographically verifies the authenticity and integrity of the approval token.
func (s *ApprovalSignal) VerifyToken(secretKey string) bool {
	if !s.IsValid() || secretKey == "" {
		return false
	}

	expectedPayload := computeApprovalPayload(s.ApproverID, s.Scope, s.ApprovedAt, s.ValidUntil)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(expectedPayload))
	expectedMAC := h.Sum(nil)

	actualMAC, err := hex.DecodeString(s.Token)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(actualMAC, expectedMAC) == 1
}
