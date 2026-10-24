package book

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
)

// splitOrder splits a manuscript filename into a numeric ordering prefix and a
// name. Files without a numeric prefix sort after all numbered files.
func splitOrder(name string) (int, string) {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	rest := strings.TrimLeft(stem, "0123456789")
	if rest != stem {
		if n, err := strconv.Atoi(stem[:len(stem)-len(rest)]); err == nil {
			return n, rest
		}
	}
	return int(^uint(0) >> 1), stem
}

// slugFor derives a URL-safe slug from a manuscript filename.
func slugFor(name string) string {
	_, stem := splitOrder(name)
	var b strings.Builder
	var lastDash bool
	for _, r := range strings.ToLower(stem) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// titleFor extracts the title from a Markdown body: the first H1 heading, or
// the slug when no such heading exists.
func titleFor(body []byte, slug string) string {
	for _, line := range bytes.Split(body, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("#")) && !bytes.HasPrefix(trimmed, []byte("##")) {
			t := strings.TrimSpace(string(trimmed[1:]))
			if t != "" {
				return t
			}
		}
	}
	return slug
}
