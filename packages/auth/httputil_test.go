package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestAUTH_010_HTTPErrors verifies the error constructors and the writer's status
// selection. The fallback path matters: a plain error must not leak a 200, and an
// HTTPError with no message must not write an empty body.
func TestAUTH_010_HTTPErrors(t *testing.T) {
	un := Unauthorized("no token")
	if un.Status != http.StatusUnauthorized || un.Code != "unauthorized" {
		t.Errorf("Unauthorized = %+v, want 401/unauthorized", un)
	}
	if got := un.Error(); got != "no token" {
		t.Errorf("Error() = %q, want the message", got)
	}

	fb := Forbidden("insufficient role")
	if fb.Status != http.StatusForbidden || fb.Code != "forbidden" {
		t.Errorf("Forbidden = %+v, want 403/forbidden", fb)
	}

	// No message: Error() falls back to the standard status text.
	if got := (&HTTPError{Status: http.StatusTeapot}).Error(); got != http.StatusText(http.StatusTeapot) {
		t.Errorf("Error() = %q, want the status text", got)
	}

	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"HTTPError keeps its status", Unauthorized("x"), http.StatusUnauthorized},
		{"generic error becomes 500", contextError{}, http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Error(rec, c.err)
			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
		})
	}
}

// contextError is a plain error used to exercise the non-HTTPError branch.
type contextError struct{}

func (contextError) Error() string { return "boom" }

// TestAUTH_011_ErrorWriterUsesStatusTextFallback verifies the writer emits the
// standard text when an HTTPError carries no message of its own.
func TestAUTH_011_ErrorWriterUsesStatusTextFallback(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, &HTTPError{Status: http.StatusForbidden})
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Error("body is empty, want the status text fallback")
	}
}

// TestAUTH_012_CookieValue verifies the cookie helper returns the value when
// present and an empty string when absent, so a caller can treat both the same
// way instead of branching on an error.
func TestAUTH_012_CookieValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	if got := CookieValue(req, "session"); got != "abc123" {
		t.Errorf("CookieValue = %q, want abc123", got)
	}
	if got := CookieValue(req, "absent"); got != "" {
		t.Errorf("CookieValue(absent) = %q, want empty", got)
	}
}

// TestAUTH_013_StrEqFoldIsASCIICaseInsensitive verifies the credential-safe
// comparison: ASCII case is ignored, but a Unicode case folding that
// strings.EqualFold would accept must not match, and length differences must
// never match.
func TestAUTH_013_StrEqFoldIsASCIICaseInsensitive(t *testing.T) {
	equal := [][2]string{
		{"Bearer", "bearer"},
		{"BEARER", "Bearer"},
		{"basic", "BASIC"},
		{"", ""},
		{"MiXeD123", "mixed123"},
	}
	for _, c := range equal {
		if !StrEqFold(c[0], c[1]) {
			t.Errorf("StrEqFold(%q, %q) = false, want true", c[0], c[1])
		}
	}

	notEqual := [][2]string{
		{"Bearer", "Bearers"},
		{"Bearer", "Bear"},
		{"", "x"},
		{"user", "pass"},
		// Unicode folding: the long s folds to 's' under EqualFold but must
		// not be accepted as the same credential here.
		{"pass", "paſs"},
	}
	for _, c := range notEqual {
		if StrEqFold(c[0], c[1]) {
			t.Errorf("StrEqFold(%q, %q) = true, want false", c[0], c[1])
		}
	}
}

// TestAUTH_015_DecodeBasicPair verifies RFC 7617 credential decoding for both
// padded and unpadded base64, a password containing colons, and the two failure
// modes: undecodable input and a pair with no colon at all.
func TestAUTH_015_DecodeBasicPair(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	if id, pass, err := decodeBasicPair(enc("alice:s3cret")); err != nil || id != "alice" || pass != "s3cret" {
		t.Errorf("decode = (%q, %q, %v), want (alice, s3cret, nil)", id, pass, err)
	}
	// An empty password is legal in the format and must not be an error.
	if id, pass, err := decodeBasicPair(enc("alice:")); err != nil || id != "alice" || pass != "" {
		t.Errorf("empty password: (%q, %q, %v), want (alice, empty, nil)", id, pass, err)
	}
	// A colon inside the password belongs to the password, not a new field.
	if id, pass, err := decodeBasicPair(enc("alice:pa:ss")); err != nil || id != "alice" || pass != "pa:ss" {
		t.Errorf("colon in password: (%q, %q, %v), want (alice, pa:ss, nil)", id, pass, err)
	}
	// Unpadded base64 is accepted as a fallback.
	if id, _, err := decodeBasicPair(base64.RawStdEncoding.EncodeToString([]byte("bob:pw"))); err != nil || id != "bob" {
		t.Errorf("raw base64: (%q, %v), want (bob, nil)", id, err)
	}

	if _, _, err := decodeBasicPair("!!!not base64!!!"); err == nil {
		t.Error("undecodable input must be an error")
	}
	if _, _, err := decodeBasicPair(enc("nocolonhere")); err == nil {
		t.Error("a credential with no colon must be an error")
	}
}

// TestAUTH_016_SanitizeRealmFallback verifies a blank realm becomes the
// documented "Restricted" default rather than an empty Basic challenge, which
// some clients treat as malformed.
func TestAUTH_016_SanitizeRealmFallback(t *testing.T) {
	for _, in := range []string{"", "   ", "\r\n", "\t"} {
		if got := sanitizeRealm(in); got != "Restricted" {
			t.Errorf("sanitizeRealm(%q) = %q, want Restricted", in, got)
		}
	}
	// A realm of only quotes escapes to a non-empty string, so it is kept: the
	// blank check runs after escaping and must not discard a realm the caller
	// deliberately set.
	if got := sanitizeRealm(`""`); got != `\"\"` {
		t.Errorf("sanitizeRealm(%q) = %q, want the escaped form", `""`, got)
	}
	if got := sanitizeRealm("krewire"); got != "krewire" {
		t.Errorf("sanitizeRealm = %q, want krewire", got)
	}
}

// TestAUTH_017_SubtleCompareIsLengthSafe verifies the constant-time comparison
// returns false, not a panic or a false positive, for differing lengths.
func TestAUTH_017_SubtleCompareIsLengthSafe(t *testing.T) {
	if !subtleCompare("secret", "secret") {
		t.Error("identical strings must compare equal")
	}
	if subtleCompare("secret", "secre") {
		t.Error("different lengths must not compare equal")
	}
	if subtleCompare("", "x") {
		t.Error("empty vs non-empty must not compare equal")
	}
	if !subtleCompare("", "") {
		t.Error("two empty strings must compare equal")
	}
}

// TestAUTH_018_BasicAuthChallengesOnVerifierError verifies the challenge path: a
// verifier returning a nil identity with no error still produces a 401 challenge
// rather than falling through to the handler.
func TestAUTH_018_BasicAuthChallengesOnVerifierError(t *testing.T) {
	mw := BasicAuth("krewire", func(string, string) (*Identity, error) {
		return nil, nil // no error, but no identity either
	})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the handler must not run without an identity")
	}))

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("alice:pw")))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("missing WWW-Authenticate challenge")
	}
}

// TestAUTH_019_BasicAuthIsSchemeAndCaseInsensitive verifies "Basic" is matched
// case-insensitively per RFC 7235, and that a Bearer token is not accepted.
func TestAUTH_019_BasicAuthIsSchemeAndCaseInsensitive(t *testing.T) {
	creds := base64.StdEncoding.EncodeToString([]byte("alice:pw"))
	cases := []struct {
		header string
		want   int
	}{
		{"Basic " + creds, http.StatusOK},
		{"basic " + creds, http.StatusOK},
		{"BASIC " + creds, http.StatusOK},
		{"Bearer " + creds, http.StatusUnauthorized},
		{"Basic", http.StatusUnauthorized},
		{"", http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.header, func(t *testing.T) {
			h := BasicAuth("krewire", func(id, pass string) (*Identity, error) {
				return &Identity{Subject: id}, nil
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/secure", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("status = %d, want %d", rec.Code, c.want)
			}
		})
	}
}

// TestAUTH_020_StrconvItoaMatchesStdlib pins the deprecated wrapper to
// strconv.Itoa so the sec re-export cannot drift from it.
func TestAUTH_020_StrconvItoaMatchesStdlib(t *testing.T) {
	for _, v := range []int{0, 1, -1, 42, -2147483648} {
		if got, want := StrconvItoa(v), strconv.Itoa(v); got != want {
			t.Errorf("StrconvItoa(%d) = %q, want %q", v, got, want)
		}
	}
}
