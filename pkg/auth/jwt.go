package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Claims is a JWT claim set.
type Claims map[string]any

// ErrInvalidToken classifies any rejected JWT.
var ErrInvalidToken = errors.New("auth: invalid token")

const jwtAlgHS256 = "HS256"

// SignJWT produces a compact HS256 JWS.
func SignJWT(secret []byte, claims Claims) (string, error) {
	if len(secret) < RecommendedMinSecretLength {
		return "", fmt.Errorf("auth: jwt secret must be at least %d bytes", RecommendedMinSecretLength)
	}
	header := map[string]any{"alg": jwtAlgHS256, "typ": "JWT"}
	h, err := b64JSON(header)
	if err != nil {
		return "", err
	}
	p, err := b64JSON(claims)
	if err != nil {
		return "", err
	}
	signingInput := h + "." + p
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig, nil
}

// DefaultClaims seeds standard claims: sub, iat now, exp now+ttl.
func DefaultClaims(sub string, ttl time.Duration) Claims {
	now := time.Now().Unix()
	return Claims{"sub": sub, "iat": now, "exp": now + int64(ttl.Seconds())}
}

// RecommendedMinSecretLength is 32 bytes (256 bits) per RFC 7518 §3.2 for HS256.
const RecommendedMinSecretLength = 32

// ParseOptions tunes ParseJWT verification.
type ParseOptions struct {
	Leeway          time.Duration
	RequireExp      bool
	MinSecretLength int
}

// ParseOption configures ParseOptions.
type ParseOption func(*ParseOptions)

// WithLeeway sets clock skew tolerance when checking exp, nbf, and iat.
func WithLeeway(d time.Duration) ParseOption {
	return func(o *ParseOptions) { o.Leeway = d }
}

// WithRequireExp requires the exp claim to be present.
func WithRequireExp(req bool) ParseOption {
	return func(o *ParseOptions) { o.RequireExp = req }
}

// WithMinSecretLength sets the minimum allowed secret key length in bytes.
func WithMinSecretLength(n int) ParseOption {
	return func(o *ParseOptions) { o.MinSecretLength = n }
}

// ParseJWT verifies signature (alg pinned to HS256), expiry, nbf, and iat.
func ParseJWT(secret []byte, token string, opts ...ParseOption) (Claims, error) {
	if len(secret) == 0 {
		return nil, errors.New("auth: empty jwt secret")
	}
	po := &ParseOptions{MinSecretLength: RecommendedMinSecretLength}
	for _, f := range opts {
		f(po)
	}
	if po.MinSecretLength > 0 && len(secret) < po.MinSecretLength {
		return nil, fmt.Errorf("%w: secret must be at least %d bytes", ErrInvalidToken, po.MinSecretLength)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: wrong segment count", ErrInvalidToken)
	}
	var hdr struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	rawHdr, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: header", ErrInvalidToken)
	}
	if err := json.Unmarshal(rawHdr, &hdr); err != nil || hdr.Alg != jwtAlgHS256 {
		return nil, fmt.Errorf("%w: alg", ErrInvalidToken)
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(want, got) {
		return nil, fmt.Errorf("%w: signature", ErrInvalidToken)
	}
	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload", ErrInvalidToken)
	}
	var claims Claims
	if err := json.Unmarshal(rawPayload, &claims); err != nil {
		return nil, fmt.Errorf("%w: payload json", ErrInvalidToken)
	}

	now := time.Now().Unix()
	leewaySec := int64(po.Leeway.Seconds())

	if expv, ok := claims["exp"]; ok {
		if exp, ok := toInt64(expv); !ok || now-leewaySec >= exp {
			return nil, fmt.Errorf("%w: expired", ErrInvalidToken)
		}
	} else if po.RequireExp {
		return nil, fmt.Errorf("%w: missing required exp claim", ErrInvalidToken)
	}

	if nbfv, ok := claims["nbf"]; ok {
		if nbf, ok := toInt64(nbfv); !ok || now+leewaySec < nbf {
			return nil, fmt.Errorf("%w: token not yet valid (nbf)", ErrInvalidToken)
		}
	}

	if iatv, ok := claims["iat"]; ok {
		if iat, ok := toInt64(iatv); !ok || now+leewaySec < iat {
			return nil, fmt.Errorf("%w: token issued in the future (iat)", ErrInvalidToken)
		}
	}

	return claims, nil
}

// JWTOptions tunes the JWTAuth middleware.
//
// The three verification knobs are inherited from ParseOptions rather than
// redeclared, so a policy expressed once (WithLeeway, WithRequireExp,
// WithMinSecretLength) means the same thing to the codec and to the middleware.
// The JWT-prefixed option constructors below are adapters over the ParseOption
// set and add no behaviour of their own.
type JWTOptions struct {
	ParseOptions
	CookieName        string
	ContinueOnMissing bool
	Required          []ClaimCheck
}

// verifyOptions projects the middleware options onto the ParseOption set the
// codec consumes, so ParseJWT applies exactly the policy JWTAuth was given.
func (o *JWTOptions) verifyOptions() []ParseOption {
	return []ParseOption{
		func(po *ParseOptions) { *po = o.ParseOptions },
	}
}

// WithJWTLeeway configures clock skew tolerance for JWTAuth middleware.
//
// Deprecated: use WithLeeway, which JWTAuth now shares with ParseJWT.
func WithJWTLeeway(d time.Duration) func(*JWTOptions) {
	return func(o *JWTOptions) { WithLeeway(d)(&o.ParseOptions) }
}

// WithJWTRequireExp configures whether exp claim is required.
//
// Deprecated: use WithRequireExp, which JWTAuth now shares with ParseJWT.
func WithJWTRequireExp(req bool) func(*JWTOptions) {
	return func(o *JWTOptions) { WithRequireExp(req)(&o.ParseOptions) }
}

// WithJWTMinSecretLength configures minimum secret length for JWTAuth middleware.
//
// Deprecated: use WithMinSecretLength, which JWTAuth now shares with ParseJWT.
func WithJWTMinSecretLength(n int) func(*JWTOptions) {
	return func(o *JWTOptions) { WithMinSecretLength(n)(&o.ParseOptions) }
}

// identityFromClaims maps a verified claim set onto an Identity: the subject,
// the roles, and the raw claims. Role extraction accepts the three shapes a
// token may carry — a "roles" array, a single "roles" string, or a singular
// "role" string — so tokens minted by different issuers all authenticate.
//
// It reports false when the claim set violates the claim-check policy, which
// the caller turns into a 401.
func identityFromClaims(claims Claims, required []ClaimCheck) (*Identity, bool) {
	for _, rc := range required {
		if v, ok := claims[rc.Key]; !ok || fmt.Sprint(v) != fmt.Sprint(rc.Value) {
			return nil, false
		}
	}
	id := &Identity{Method: "jwt", Claims: claims}
	if sub, ok := claims["sub"]; ok {
		id.Subject = fmt.Sprint(sub)
	}
	switch rv := claims["roles"].(type) {
	case []any:
		for _, x := range rv {
			id.Roles = append(id.Roles, fmt.Sprint(x))
		}
	case string:
		id.Roles = append(id.Roles, rv)
	}
	if role, ok := claims["role"].(string); ok && len(id.Roles) == 0 {
		id.Roles = append(id.Roles, role)
	}
	return id, true
}

// ClaimCheck asserts one claim equality.
type ClaimCheck struct {
	Key   string
	Value any
}

// JWTAuth verifies bearer credentials and stores the identity (Method "jwt").
//
// SECURITY: the exp claim is required by default. A token without an expiry is
// a permanent credential, so accepting one turns a short-lived session into an
// unrevocable one. Callers that deliberately issue non-expiring tokens must opt
// out explicitly with WithJWTRequireExp(false).
func JWTAuth(secret []byte, opts ...func(*JWTOptions)) Middleware {
	o := &JWTOptions{ParseOptions: ParseOptions{RequireExp: true, MinSecretLength: RecommendedMinSecretLength}}
	for _, f := range opts {
		f(o)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			scheme, param, ok := authParam(r.Header.Get("Authorization"))
			if ok && StrEqFold(scheme, "Bearer") {
				token = param
			} else if o.CookieName != "" {
				token = CookieValue(r, o.CookieName)
			}
			if token == "" {
				if o.ContinueOnMissing {
					next.ServeHTTP(w, r)
					return
				}
				Error(w, Unauthorized("missing bearer token"))
				return
			}
			claims, err := ParseJWT(secret, token, o.verifyOptions()...)
			if err != nil {
				Error(w, Unauthorized("invalid token"))
				return
			}
			id, ok := identityFromClaims(claims, o.Required)
			if !ok {
				Error(w, Unauthorized("claim check failed"))
				return
			}
			next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), id)))
		})
	}
}

// B64JSON encodes v as base64url JSON. Exported for test compatibility.
func B64JSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// b64JSON is the internal alias used by SignJWT.
var b64JSON = B64JSON

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}
