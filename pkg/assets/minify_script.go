package assets

import (
	"strings"
)

// MinifyJS applies safe lightweight minification to JavaScript source code.
// It removes multi-line and single-line comments, collapses excessive whitespace,
// strips blank lines, and preserves string literals and template literals verbatim.
func MinifyJS(js string) string {
	if len(js) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(js))

	runes := []rune(js)
	n := len(runes)
	i := 0

	for i < n {
		r := runes[i]

		// 1. Single-line comment: // ...
		if r == '/' && i+1 < n && runes[i+1] == '/' {
			// Skip until newline
			i += 2
			for i < n && runes[i] != '\n' && runes[i] != '\r' {
				i++
			}
			continue
		}

		// 2. Multi-line comment: /* ... */
		if r == '/' && i+1 < n && runes[i+1] == '*' {
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}

		// 3. String literals: '...', "...", `...` (template literals)
		if r == '\'' || r == '"' || r == '`' {
			quote := r
			sb.WriteRune(r)
			i++
			for i < n {
				qr := runes[i]
				sb.WriteRune(qr)
				if qr == '\\' && i+1 < n {
					i++
					sb.WriteRune(runes[i])
				} else if qr == quote {
					i++
					break
				}
				i++
			}
			continue
		}

		sb.WriteRune(r)
		i++
	}

	// Clean up lines: strip whitespace and collapse consecutive empty lines
	raw := sb.String()
	lines := strings.Split(raw, "\n")
	var cleaned []string
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if trim != "" {
			cleaned = append(cleaned, trim)
		}
	}

	return strings.Join(cleaned, "\n")
}
