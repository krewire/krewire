package markdown

import (
	"fmt"
	"strings"
	"unicode"
)

// slugTracker ensures deterministic, unique heading IDs matching Goldmark/GFM.
type slugTracker struct {
	counts map[string]int
}

func newSlugTracker() *slugTracker {
	return &slugTracker{counts: make(map[string]int)}
}

// Slug generates a kebab-case ID from the heading raw text.
func (st *slugTracker) Slug(text string) string {
	raw := stripFormatting(text)
	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(unicode.ToLower(r))
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "-") {
				sb.WriteRune('-')
			}
		}
	}
	slug := strings.Trim(sb.String(), "-")
	if slug == "" {
		slug = "heading"
	}
	st.counts[slug]++
	count := st.counts[slug]
	if count > 1 {
		return fmt.Sprintf("%s-%d", slug, count-1)
	}
	return slug
}

// stripFormatting removes markdown markers like `*`, `_`, `~`, “ ` “ and raw HTML tags.
func stripFormatting(s string) string {
	var sb strings.Builder
	inTag := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' && inTag {
			inTag = false
			continue
		}
		if inTag {
			continue
		}
		// Skip markdown markup chars from slug calculation
		if ch == '*' || ch == '_' || ch == '~' || ch == '`' {
			continue
		}
		sb.WriteByte(ch)
	}
	return sb.String()
}
