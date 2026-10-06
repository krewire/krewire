package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAUTH_001_ToInt64CoversEveryJSONNumberShape verifies the numeric claim
// coercion accepts every shape a decoded JSON number or hand-built claim can
// arrive as, and rejects everything else. A rejected shape must return false so
// the caller fails closed rather than treating a non-numeric claim as a
// timestamp.
func TestAUTH_001_ToInt64CoversEveryJSONNumberShape(t *testing.T) {
	accepted := []struct {
		name string
		in   any
		want int64
	}{
		{"float64 from JSON decoding", float64(1700000000), 1700000000},
		{"int64", int64(42), 42},
		{"int", int(7), 7},
		{"json.Number", json.Number("1234"), 1234},
		{"negative float64", float64(-5), -5},
	}
	for _, c := range accepted {
		t.Run(c.name, func(t *testing.T) {
			got, ok := toInt64(c.in)
			if !ok {
				t.Fatalf("toInt64(%#v) reported failure, want %d", c.in, c.want)
			}
			if got != c.want {
				t.Errorf("toInt64(%#v) = %d, want %d", c.in, got, c.want)
			}
		})
	}

	rejected := []struct {
		name string
		in   any
	}{
		{"string is not a number", "1700000000"},
		{"nil", nil},
		{"bool", true},
		{"float32", float32(1)},
		{"unparseable json.Number", json.Number("not-a-number")},
		{"overflowing json.Number", json.Number("99999999999999999999999")},
	}
	for _, c := range rejected {
		t.Run(c.name, func(t *testing.T) {
			if got, ok := toInt64(c.in); ok {
				t.Errorf("toInt64(%#v) = %d, want rejection", c.in, got)
			}
		})
	}
}

// TestAUTH_002_NonNumericExpIsRejected verifies a token whose exp claim is a
// string rather than a number is refused, rather than the claim being ignored
// and the token treated as non-expiring.
func TestAUTH_002_NonNumericExpIsRejected(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	token, err := SignJWT(secret, Claims{"sub": "u", "exp": "tomorrow"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseJWT(secret, token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken for a non-numeric exp", err)
	}
}

// TestAUTH_003_JWTAuthOptionAdaptersMatchTheSharedPolicy verifies the deprecated
// WithJWT* adapters reach the same ParseOptions the ParseOption set does, since
// JWTAuth now embeds ParseOptions and a mismatch would silently ignore a caller's
// policy.
func TestAUTH_003_JWTAuthOptionAdaptersMatchTheSharedPolicy(t *testing.T) {
	o := &JWTOptions{ParseOptions: ParseOptions{RequireExp: true, MinSecretLength: RecommendedMinSecretLength}}
	WithJWTLeeway(90 * time.Second)(o)
	WithJWTRequireExp(false)(o)
	WithJWTMinSecretLength(16)(o)

	if o.Leeway != 90*time.Second {
		t.Errorf("Leeway = %v, want 90s", o.Leeway)
	}
	if o.RequireExp {
		t.Error("RequireExp = true, want the adapter to disable it")
	}
	if o.MinSecretLength != 16 {
		t.Errorf("MinSecretLength = %d, want 16", o.MinSecretLength)
	}

	// The projection handed to ParseJWT must reproduce those values exactly.
	po := &ParseOptions{}
	for _, f := range o.verifyOptions() {
		f(po)
	}
	if *po != o.ParseOptions {
		t.Errorf("projected options = %+v, want %+v", *po, o.ParseOptions)
	}
}

// TestAUTH_004_JWTAuthHonorsLeewayThroughTheSharedPolicy verifies the leeway set
// on the middleware actually reaches the codec, end to end through a token that is
// inside its nbf window.
func TestAUTH_004_JWTAuthHonorsLeewayThroughTheSharedPolicy(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	claims := DefaultClaims("carol", time.Hour)
	claims["nbf"] = time.Now().Add(5 * time.Minute).Unix()
	token, err := SignJWT(secret, claims)
	if err != nil {
		t.Fatal(err)
	}

	call := func(opts ...func(*JWTOptions)) int {
		h := JWTAuth(secret, opts...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := call(); got != http.StatusUnauthorized {
		t.Errorf("status without leeway = %d, want 401", got)
	}
	if got := call(WithJWTLeeway(10 * time.Minute)); got != http.StatusOK {
		t.Errorf("status with leeway = %d, want 200: the adapter must reach ParseJWT", got)
	}
}

// TestAUTH_005_JWTAuthReadsTokenFromCookie verifies the cookie fallback, which is
// how a browser client that cannot set an Authorization header authenticates.
func TestAUTH_005_JWTAuthReadsTokenFromCookie(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	token, err := SignJWT(secret, DefaultClaims("dave", time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	var got *Identity
	h := JWTAuth(secret, func(o *JWTOptions) { o.CookieName = "session" })(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			got = IdentityFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got == nil || got.Subject != "dave" {
		t.Errorf("identity = %+v, want subject dave", got)
	}
}

// TestAUTH_006_JWTAuthRejectsFailedClaimChecks verifies the claim policy is
// enforced by the middleware, including the numeric-and-string comparison that a
// JSON round trip can change.
func TestAUTH_006_JWTAuthRejectsFailedClaimChecks(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	// The token must carry exp: JWTAuth requires it by default, so a token
	// without one is refused before the claim checks ever run.
	claims := DefaultClaims("eve", time.Hour)
	claims["role"] = "user"
	claims["tenant"] = 42
	token, err := SignJWT(secret, claims)
	if err != nil {
		t.Fatal(err)
	}

	call := func(check ClaimCheck) int {
		h := JWTAuth(secret, func(o *JWTOptions) { o.Required = []ClaimCheck{check} })(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := call(ClaimCheck{Key: "role", Value: "user"}); got != http.StatusOK {
		t.Errorf("matching claim: status = %d, want 200", got)
	}
	if got := call(ClaimCheck{Key: "role", Value: "admin"}); got != http.StatusUnauthorized {
		t.Errorf("mismatched claim: status = %d, want 401", got)
	}
	if got := call(ClaimCheck{Key: "absent", Value: "x"}); got != http.StatusUnauthorized {
		t.Errorf("missing claim: status = %d, want 401", got)
	}
	// A claim that arrives as a float64 must still match its integer form.
	if got := call(ClaimCheck{Key: "tenant", Value: 42}); got != http.StatusOK {
		t.Errorf("numeric claim: status = %d, want 200", got)
	}
}

// TestAUTH_007_IdentityFromClaimsRoleShapes verifies the three claim shapes that
// carry roles, plus the precedence between them.
func TestAUTH_007_IdentityFromClaimsRoleShapes(t *testing.T) {
	cases := []struct {
		name   string
		claims Claims
		want   []string
	}{
		{"roles array", Claims{"roles": []any{"admin", "editor"}}, []string{"admin", "editor"}},
		{"roles string", Claims{"roles": "admin"}, []string{"admin"}},
		{"singular role", Claims{"role": "admin"}, []string{"admin"}},
		{"no roles", Claims{"sub": "u"}, nil},
		{"non-string entries are stringified", Claims{"roles": []any{1, true}}, []string{"1", "true"}},
		{"roles array wins over singular role", Claims{"roles": []any{"a"}, "role": "b"}, []string{"a"}},
		{"numeric roles are ignored", Claims{"roles": 7}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id, ok := identityFromClaims(c.claims, nil)
			if !ok {
				t.Fatal("identityFromClaims reported a policy failure with no required claims")
			}
			if len(id.Roles) != len(c.want) {
				t.Fatalf("roles = %v, want %v", id.Roles, c.want)
			}
			for i := range c.want {
				if id.Roles[i] != c.want[i] {
					t.Errorf("roles[%d] = %q, want %q", i, id.Roles[i], c.want[i])
				}
			}
		})
	}
}

// TestAUTH_008_IdentityFromClaimsSubject verifies sub is stringified when present
// and an absent claim leaves the subject empty rather than panicking.
func TestAUTH_008_IdentityFromClaimsSubject(t *testing.T) {
	id, _ := identityFromClaims(Claims{"sub": 12345}, nil)
	if id.Subject != "12345" {
		t.Errorf("Subject = %q, want \"12345\"", id.Subject)
	}
	if id.Method != "jwt" {
		t.Errorf("Method = %q, want jwt", id.Method)
	}
	id, _ = identityFromClaims(Claims{}, nil)
	if id.Subject != "" {
		t.Errorf("Subject = %q, want empty when sub is absent", id.Subject)
	}
}
