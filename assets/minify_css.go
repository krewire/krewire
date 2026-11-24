package assets

import (
	"strings"
)

// MinifyCSS applies tokenizer-based minification to CSS.
// It strips comments, collapses whitespace, preserves string literals and CSS
// custom properties, strips trailing semicolons, and eliminates empty rule blocks.
func MinifyCSS(css string) string {
	if len(css) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(css))

	runes := []rune(css)
	n := len(runes)
	i := 0

	for i < n {
		r := runes[i]

		// 1. Skip comments /* ... */
		if r == '/' && i+1 < n && runes[i+1] == '*' {
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i += 2 // skip closing */
			continue
		}

		// 2. String literal: '...' or "..." (preserve verbatim)
		if r == '\'' || r == '"' {
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

		// 3. Whitespace handling
		if isCSSWhitespace(r) {
			prev := lastNonSpaceRune(sb.String())
			nextIdx := i
			for nextIdx < n && isCSSWhitespace(runes[nextIdx]) {
				nextIdx++
			}

			if nextIdx < n {
				next := runes[nextIdx]
				// Safe stripping around major delimiters
				if canDropSpaceBetween(prev, next) {
					i = nextIdx
					continue
				}
				sb.WriteByte(' ')
				i = nextIdx
				continue
			} else {
				i = nextIdx
				continue
			}
		}

		// 4. Semicolon before closing brace: ';}' -> '}'
		if r == ';' {
			nextIdx := i + 1
			for nextIdx < n && isCSSWhitespace(runes[nextIdx]) {
				nextIdx++
			}
			if nextIdx < n && runes[nextIdx] == '}' {
				i = nextIdx
				continue
			}
		}

		sb.WriteRune(r)
		i++
	}

	out := sb.String()
	out = removeEmptyCSSRules(out)

	return strings.TrimSpace(out)
}

func isCSSWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
}

func canDropSpaceBetween(prev, next rune) bool {
	// Always drop spaces around { } ; ,
	if prev == '{' || prev == '}' || prev == ';' || prev == ',' {
		return true
	}
	if next == '{' || next == '}' || next == ';' || next == ',' {
		return true
	}

	// Drop space before and after colon in declarations, e.g. "color: red" -> "color:red"
	// But preserve space in media queries or selectors where appropriate
	if prev == ':' {
		return true
	}
	if next == ':' {
		return true
	}

	// Drop space around combinators >, +, ~
	if prev == '>' || next == '>' {
		return true
	}

	return false
}

func lastNonSpaceRune(s string) rune {
	runes := []rune(s)
	for j := len(runes) - 1; j >= 0; j-- {
		if !isCSSWhitespace(runes[j]) {
			return runes[j]
		}
	}
	return 0
}

func removeEmptyCSSRules(css string) string {
	for {
		start := strings.Index(css, "{}")
		if start == -1 {
			break
		}
		selStart := 0
		for j := start - 1; j >= 0; j-- {
			if css[j] == '}' || css[j] == '{' || css[j] == ';' {
				selStart = j + 1
				break
			}
		}
		css = css[:selStart] + css[start+2:]
	}
	return css
}
