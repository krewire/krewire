package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

var errMalformedBasic = errors.New("auth: malformed basic credentials")

// BasicVerifier validates an identifier/password pair.
type BasicVerifier func(identifier, password string) (*Identity, error)

// BasicAuth implements RFC 7617 over the verifier.
func BasicAuth(realm string, verify BasicVerifier) Middleware {
	challenge := "Basic realm=\"" + sanitizeRealm(realm) + "\""

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, param, ok := authParam(r.Header.Get("Authorization"))
			if !ok || !StrEqFold(scheme, "Basic") {
				challenge401(w, challenge)
				return
			}
			id, pass, err := decodeBasicPair(param)
			if err != nil {
				challenge401(w, challenge)
				return
			}
			identity, verr := verify(id, pass)
			if verr != nil || identity == nil {
				if verr != nil {
					Error(w, verr)
					return
				}
				challenge401(w, challenge)
				return
			}
			identity.Method = "basic"
			next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), identity)))
		})
	}
}

func challenge401(w http.ResponseWriter, challenge string) {
	w.Header().Set("WWW-Authenticate", challenge)
	Error(w, Unauthorized("authentication required"))
}

func decodeBasicPair(param string) (identifier, password string, err error) {
	raw, err := base64.StdEncoding.DecodeString(param)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(param)
		if err != nil {
			return "", "", err
		}
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] == ':' {
			return string(raw[:i]), string(raw[i+1:]), nil
		}
	}
	return "", "", errMalformedBasic
}

func subtleCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func sanitizeRealm(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	if strings.TrimSpace(s) == "" {
		return "Restricted"
	}
	return s
}
