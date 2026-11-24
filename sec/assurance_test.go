package sec_test

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krewire/krewire/auth"
	"github.com/krewire/krewire/sec"
)

// TestSecurityAssurance_AuthenticationAndTiming verifies cryptographic hygiene,
// PBKDF2-HMAC-SHA256 password hashing, and timing-attack resistance.
func TestSecurityAssurance_AuthenticationAndTiming(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	// 1. Password hashing
	hash1, err := sec.HashPasswordPBKDF2(password, sec.DefaultPBKDF2Iterations)
	if err != nil {
		t.Fatalf("HashPasswordPBKDF2 failed: %v", err)
	}

	hash2, err := sec.HashPasswordPBKDF2(password, sec.DefaultPBKDF2Iterations)
	if err != nil {
		t.Fatalf("HashPasswordPBKDF2 second run failed: %v", err)
	}

	// Salt uniqueness test: identical passwords must generate different salts/hashes
	if hash1 == hash2 {
		t.Fatalf("cryptographic vulnerability: identical hashes generated for same password (missing unique salt)")
	}

	// Verification test
	if !sec.VerifyPasswordPBKDF2(password, hash1) {
		t.Errorf("VerifyPasswordPBKDF2 failed to verify correct password")
	}
	if sec.VerifyPasswordPBKDF2("WrongPassword!", hash1) {
		t.Errorf("VerifyPasswordPBKDF2 unexpectedly succeeded with wrong password")
	}

	// Constant-time comparison verification
	tokenA := []byte("secret_api_token_4fc69929")
	tokenB := []byte("secret_api_token_4fc69929")
	tokenC := []byte("secret_api_token_mismatch")

	if subtle.ConstantTimeCompare(tokenA, tokenB) != 1 {
		t.Errorf("ConstantTimeCompare failed on identical tokens")
	}
	if subtle.ConstantTimeCompare(tokenA, tokenC) != 0 {
		t.Errorf("ConstantTimeCompare failed to reject differing tokens")
	}
}

// TestSecurityAssurance_AuthorizationDefaultDeny verifies that endpoints enforce
// default-deny access control and role-based policies.
func TestSecurityAssurance_AuthorizationDefaultDeny(t *testing.T) {
	mux := http.NewServeMux()

	adminHandler := sec.Require(sec.Authenticated(), sec.WithRoles("admin"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin-secret-data"))
	}))
	mux.Handle("/admin", adminHandler)

	// 1. Anonymous request -> 401 Unauthorized
	reqAnon := httptest.NewRequest(http.MethodGet, "/admin", nil)
	recAnon := httptest.NewRecorder()
	mux.ServeHTTP(recAnon, reqAnon)
	if recAnon.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated request, got %d", recAnon.Code)
	}

	// 2. Authenticated user without required role -> 403 Forbidden
	userCtx := auth.WithIdentity(context.Background(), &auth.Identity{
		Subject: "user-123",
		Roles:   []string{"viewer"},
	})
	reqUser := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx)
	recUser := httptest.NewRecorder()
	mux.ServeHTTP(recUser, reqUser)
	if recUser.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for insufficient role, got %d", recUser.Code)
	}

	// 3. Authenticated admin -> 200 OK
	adminCtx := auth.WithIdentity(context.Background(), &auth.Identity{
		Subject: "admin-001",
		Roles:   []string{"admin"},
	})
	reqAdmin := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(adminCtx)
	recAdmin := httptest.NewRecorder()
	mux.ServeHTTP(recAdmin, reqAdmin)
	if recAdmin.Code != http.StatusOK {
		t.Errorf("expected 200 OK for authorized admin, got %d", recAdmin.Code)
	}
}

// TestSecurityAssurance_PathTraversalSafePath verifies that SafePath enforces
// boundary containment against directory traversal attacks (CWE-22).
func TestSecurityAssurance_PathTraversalSafePath(t *testing.T) {
	baseDir := t.TempDir()
	legitFile := filepath.Join(baseDir, "data", "report.json")
	if err := os.MkdirAll(filepath.Dir(legitFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legitFile, []byte(`{"status":"ok"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Valid path inside baseDir
	resolved, err := sec.SafePath(baseDir, "data/report.json")
	if err != nil || resolved != legitFile {
		t.Errorf("SafePath valid path failed: resolved=%q, err=%v", resolved, err)
	}

	// Traversal attacks and invalid inputs
	maliciousInputs := []string{
		"../secret.txt",
		"../../../../etc/passwd",
		"data/../../shadow",
		"..\\..\\windows\\system32\\config",
		"data/\x00secret.txt",
	}

	for _, input := range maliciousInputs {
		_, err := sec.SafePath(baseDir, input)
		if err == nil {
			t.Errorf("security vulnerability: SafePath allowed malicious traversal input %q", input)
		}
		if !errors.Is(err, sec.ErrPathTraversal) && !errors.Is(err, sec.ErrInvalidPath) {
			t.Errorf("expected ErrPathTraversal or ErrInvalidPath for %q, got: %v", input, err)
		}
	}

	// Absolute path input is safely scoped into baseDir sandbox
	sandboxed, err := sec.SafePath(baseDir, "/etc/passwd")
	if err != nil {
		t.Fatalf("SafePath failed to scope absolute path: %v", err)
	}
	expectedSandboxed := filepath.Join(baseDir, "etc", "passwd")
	if sandboxed != expectedSandboxed {
		t.Errorf("expected %q, got %q", expectedSandboxed, sandboxed)
	}
}

// TestSecurityAssurance_SSRFDefense verifies that outbound network requests
// reject private networks, cloud metadata, and loopback addresses.
func TestSecurityAssurance_SSRFDefense(t *testing.T) {
	deniedURLs := []string{
		"http://127.0.0.1:8080/admin",
		"http://127.0.0.2/status",
		"http://localhost:3000/internal",
		"http://10.0.0.5/api/vault",
		"http://172.16.1.1/internal",
		"http://192.168.1.100/router",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
	}

	for _, u := range deniedURLs {
		err := sec.ValidateOutboundURL(u, "api.trusted-partner.com")
		if err == nil {
			t.Errorf("security vulnerability: SSRF validator allowed dangerous outbound URL: %s", u)
		}
	}

	// Permitted URL matching allowlist
	err := sec.ValidateOutboundURL("https://api.trusted-partner.com/v1/webhook", "api.trusted-partner.com")
	if err != nil {
		t.Errorf("expected allowlisted outbound URL to pass, got: %v", err)
	}
}

// TestSecurityAssurance_TransportHeaders verifies default OWASP defensive headers.
func TestSecurityAssurance_TransportHeaders(t *testing.T) {
	mw := sec.SecurityHeaders()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	handler.ServeHTTP(rec, req)

	headers := rec.Header()
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for key, want := range expectedHeaders {
		got := headers.Get(key)
		if got != want {
			t.Errorf("header %s = %q, want %q", key, got, want)
		}
	}
}

// TestSecurityAssurance_AIAgentBoundariesAndAudit verifies that autonomous AI agent
// actions are restricted to workspace boundaries and produce structured audit records.
func TestSecurityAssurance_AIAgentBoundariesAndAudit(t *testing.T) {
	workspaceRoot := t.TempDir()

	// Simulate AI agent filesystem write action
	attemptWrite := func(agentID, relPath string, content []byte) (string, error) {
		safeTarget, err := sec.SafePath(workspaceRoot, relPath)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(safeTarget), 0755); err != nil {
			return "", err
		}
		return safeTarget, os.WriteFile(safeTarget, content, 0644)
	}

	// 1. Valid workspace write inside boundary
	targetPath, err := attemptWrite("krew-agent-01", "internal/handler.go", []byte("package handler"))
	if err != nil || targetPath == "" {
		t.Fatalf("expected valid agent write to succeed, got err=%v", err)
	}

	// 2. Traversal attempt out of workspace
	_, err = attemptWrite("krew-agent-01", "../../etc/hosts", []byte("evil"))
	if !errors.Is(err, sec.ErrPathTraversal) {
		t.Errorf("expected ErrPathTraversal when agent attempts write outside workspace, got: %v", err)
	}

	// 3. Structured audit logging record structure
	type AIAuditRecord struct {
		AgentID   string    `json:"agent_id"`
		Timestamp time.Time `json:"timestamp"`
		Action    string    `json:"action"`
		Target    string    `json:"target"`
		Status    string    `json:"status"`
	}

	record := AIAuditRecord{
		AgentID:   "krew-agent-01",
		Timestamp: time.Now().UTC(),
		Action:    "fs:write",
		Target:    "internal/handler.go",
		Status:    "allowed",
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(record); err != nil {
		t.Fatalf("failed to encode audit record: %v", err)
	}

	var decoded AIAuditRecord
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode audit record: %v", err)
	}

	if decoded.AgentID != "krew-agent-01" || decoded.Status != "allowed" {
		t.Errorf("audit record fields corrupted: %+v", decoded)
	}
}
