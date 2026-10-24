package auth

import (
	"net/http"
	"strconv"
)

// CookieValue extracts a cookie value by name, returning "" on error.
func CookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// LowerASCII lower-cases ASCII letters in-place, leaving every other byte —
// including UTF-8 continuation bytes — untouched.
func LowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// StrEqFold compares two strings case-insensitively over ASCII only. It is
// deliberately not strings.EqualFold: comparing ASCII avoids Unicode case
// folding in credential paths, where a Unicode-equivalent match must not be
// accepted as the same credential.
func StrEqFold(a, b string) bool {
	return len(a) == len(b) && LowerASCII(a) == LowerASCII(b)
}

// StrconvItoa converts int to string.
//
// Deprecated: use strconv.Itoa directly. This wrapper exists only for
// backward compatibility with the sec re-export and adds no behaviour.
func StrconvItoa(i int) string {
	return strconv.Itoa(i)
}
