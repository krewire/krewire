package observe

import (
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

// Redacted replaces any value removed by the redaction rules.
const Redacted = "[REDACTED]"

// sensitiveKeyParts mark a key as carrying a secret, credential, or personal
// datum. Keys are lower-cased with '-', '.', and ' ' normalized to '_'. A part
// containing '_' matches as a substring; a single word must equal one '_'
// segment, so "access_token" is sensitive while "input_tokens" is not.
var sensitiveKeyParts = []string{
	"password", "passwd", "pwd", "secret", "token", "authorization", "cookie",
	"apikey", "api_key", "private_key", "access_key", "session_key",
	"credential", "credentials", "signature", "dsn", "connection_string",
	"email", "phone", "ssn", "card_number", "cvv", "iban", "ip_address", "client_ip",
}

// IsSensitiveKey reports whether a field name must never be emitted verbatim.
func IsSensitiveKey(key string) bool {
	k := strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(strings.ToLower(key))
	segments := strings.Split(k, "_")
	for _, part := range sensitiveKeyParts {
		if strings.Contains(part, "_") {
			if strings.Contains(k, part) {
				return true
			}
			continue
		}
		for _, seg := range segments {
			if seg == part {
				return true
			}
		}
	}
	return false
}

var (
	bearerRe   = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=-]+`)
	userinfoRe = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^/\s:@]+:[^/\s@]+@`)
	kvSecretRe = regexp.MustCompile(`(?i)\b(password|passwd|secret|token|api[_-]?key|access[_-]?key)=([^\s&;,]+)`)
	emailRe    = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

// RedactString scrubs credentials embedded in free text: URL userinfo
// (postgres://u:p@host), Authorization schemes, key=value secrets, and email
// addresses. It is applied to error messages before they leave the process.
func RedactString(s string) string {
	if s == "" {
		return s
	}
	s = userinfoRe.ReplaceAllString(s, "${1}"+Redacted+"@")
	s = bearerRe.ReplaceAllString(s, "${1} "+Redacted)
	s = kvSecretRe.ReplaceAllString(s, "${1}="+Redacted)
	s = emailRe.ReplaceAllString(s, Redacted)
	return s
}

// RedactURL returns u without userinfo and with every query value removed,
// keeping only the path shape. Query strings carry user input and must not
// become log fields or metric labels.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return RedactString(raw)
	}
	u.User = nil
	if u.RawQuery != "" {
		q := u.Query()
		for k := range q {
			q[k] = []string{Redacted}
		}
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// RedactAttr applies the key and value rules to one slog attribute, descending
// into groups. It matches the slog.HandlerOptions.ReplaceAttr signature.
func RedactAttr(groups []string, a slog.Attr) slog.Attr {
	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, Redacted)
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return slog.String(a.Key, RedactString(v.String()))
	case slog.KindGroup:
		in := v.Group()
		out := make([]slog.Attr, len(in))
		for i, ga := range in {
			out[i] = RedactAttr(append(groups[:len(groups):len(groups)], a.Key), ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(out...)}
	case slog.KindAny:
		if err, ok := v.Any().(error); ok {
			return slog.String(a.Key, RedactString(err.Error()))
		}
	}
	return slog.Attr{Key: a.Key, Value: v}
}

// RedactMap returns a copy of m with sensitive keys and embedded credentials
// removed. Diagnostics bundles pass environment and config through it.
func RedactMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if IsSensitiveKey(k) {
			out[k] = Redacted
			continue
		}
		out[k] = RedactString(v)
	}
	return out
}
