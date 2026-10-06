package auth

// Tests for SEC-A02-001, SEC-A02-002, SEC-A07-001.
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJWTRoundTrip(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := DefaultClaims("user-123", time.Hour)
	claims["email"] = "user@example.com"

	token, err := SignJWT(secret, claims)
	if err != nil {
		t.Fatalf("SignJWT error = %v", err)
	}

	parsed, err := ParseJWT(secret, token)
	if err != nil {
		t.Fatalf("ParseJWT error = %v", err)
	}
	if parsed["sub"] != "user-123" {
		t.Errorf("sub = %v, want user-123", parsed["sub"])
	}
	if parsed["email"] != "user@example.com" {
		t.Errorf("email = %v, want user@example.com", parsed["email"])
	}
}

func TestJWTExpired(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := DefaultClaims("user-123", -time.Minute)

	token, err := SignJWT(secret, claims)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseJWT(secret, token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestJWTTampered(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := DefaultClaims("user-123", time.Hour)
	token, _ := SignJWT(secret, claims)

	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1] + "extra." + parts[2]
	_, err := ParseJWT(secret, tampered)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestB64JSON(t *testing.T) {
	data := map[string]string{"foo": "bar"}
	encoded, err := B64JSON(data)
	if err != nil {
		t.Fatalf("B64JSON error = %v", err)
	}
	if encoded == "" {
		t.Error("B64JSON returned empty string")
	}
}

func TestJWT_NBF_NotYetValid(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := DefaultClaims("user-123", time.Hour)
	claims["nbf"] = time.Now().Add(10 * time.Minute).Unix()

	token, err := SignJWT(secret, claims)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseJWT(secret, token)
	if err == nil {
		t.Error("expected error for token with future nbf")
	}

	// With leeway >= 10m, it should pass
	_, err = ParseJWT(secret, token, WithLeeway(15*time.Minute))
	if err != nil {
		t.Errorf("expected token to pass with sufficient leeway: %v", err)
	}
}

func TestJWT_IAT_Future(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := DefaultClaims("user-123", time.Hour)
	claims["iat"] = time.Now().Add(10 * time.Minute).Unix()

	token, err := SignJWT(secret, claims)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseJWT(secret, token)
	if err == nil {
		t.Error("expected error for token with future iat")
	}
}

func TestJWT_RequireExp(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := Claims{"sub": "user-123"} // no exp

	token, _ := SignJWT(secret, claims)
	_, err := ParseJWT(secret, token, WithRequireExp(true))
	if err == nil {
		t.Error("expected error when exp is required but missing")
	}

	_, err = ParseJWT(secret, token, WithRequireExp(false))
	if err != nil {
		t.Errorf("expected success when exp is optional: %v", err)
	}
}

func TestJWT_MinSecretLength(t *testing.T) {
	shortSecret := []byte("short")
	claims := DefaultClaims("user-123", time.Hour)
	token, _ := SignJWT(shortSecret, claims)

	_, err := ParseJWT(shortSecret, token, WithMinSecretLength(32))
	if err == nil {
		t.Error("expected error for short secret when min length is 32")
	}

	validSecret := []byte("01234567890123456789012345678901") // 32 bytes
	tokenValid, _ := SignJWT(validSecret, claims)
	_, err = ParseJWT(validSecret, tokenValid, WithMinSecretLength(32))
	if err != nil {
		t.Errorf("expected success for 32-byte secret: %v", err)
	}
}

// TestSEC_AUTH_001_JWTAuth_RequiresExp verifies the middleware rejects a token
// with no exp claim by default. A non-expiring token is a permanent credential
// that cannot be revoked, so the safe default is to refuse it and require an
// explicit opt-out.
func TestSEC_AUTH_001_JWTAuth_RequiresExp(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	tok, err := SignJWT(secret, Claims{"sub": "admin", "role": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	h := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a token without exp", rec.Code)
	}
}

// TestSEC_AUTH_002_JWTAuth_ExpiringTokenAccepted is the counterpart: a properly
// expiring token must still authenticate.
func TestSEC_AUTH_002_JWTAuth_ExpiringTokenAccepted(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	tok, err := SignJWT(secret, DefaultClaims("alice", time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	h := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for a valid expiring token", rec.Code)
	}
}

// TestSEC_AUTH_003_JWTAuth_ExpiredTokenRejected verifies expiry is enforced.
func TestSEC_AUTH_003_JWTAuth_ExpiredTokenRejected(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	tok, err := SignJWT(secret, DefaultClaims("alice", -time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	h := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for an expired token", rec.Code)
	}
}

// TestSEC_AUTH_004_JWTAuth_WeakSecretRejected verifies the middleware inherits
// the minimum-secret-length floor, so a 4-byte key cannot be used to mint
// forgable tokens. The token is signed by hand because SignJWT already refuses
// to mint with a short key, and the point here is the verify path.
func TestSEC_AUTH_004_JWTAuth_WeakSecretRejected(t *testing.T) {
	weak := []byte("weak")
	hdr, err := B64JSON(map[string]any{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := B64JSON(DefaultClaims("alice", time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, weak)
	mac.Write([]byte(hdr + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	forged := hdr + "." + payload + "." + sig

	h := JWTAuth(weak)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+forged)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a too-short signing key", rec.Code)
	}
}

// TestSEC_AUTH_005_ParseJWT_RejectsAlgNone verifies algorithm pinning. A token
// whose header claims alg=none must never verify.
func TestSEC_AUTH_005_ParseJWT_RejectsAlgNone(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	hdr, err := B64JSON(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := B64JSON(DefaultClaims("admin", time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	forged := hdr + "." + payload + "."
	if _, err := ParseJWT(secret, forged); err == nil {
		t.Fatal("ParseJWT must reject alg=none")
	}
}
