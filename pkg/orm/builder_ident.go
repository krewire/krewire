package orm

import (
	"fmt"
	"regexp"
	"strings"
)

var allowedOperators = map[string]bool{
	"=":           true,
	"!=":          true,
	"<>":          true,
	"<":           true,
	"<=":          true,
	">":           true,
	">=":          true,
	"LIKE":        true,
	"ILIKE":       true,
	"NOT LIKE":    true,
	"NOT ILIKE":   true,
	"IN":          true,
	"NOT IN":      true,
	"IS":          true,
	"IS NOT":      true,
	"BETWEEN":     true,
	"NOT BETWEEN": true,
}

var (
	identRegex  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	dottedRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\.(?:[a-zA-Z_][a-zA-Z0-9_]*|\*)$`)
	funcRegex   = regexp.MustCompile(`^(?i:[a-zA-Z_][a-zA-Z0-9_]*)\((?:[a-zA-Z_][a-zA-Z0-9_.]*|\*)\)$`)
	aliasRegex  = regexp.MustCompile(`^(?i)([a-zA-Z_][a-zA-Z0-9_.]*)\s+AS\s+([a-zA-Z_][a-zA-Z0-9_]*)$`)
)

func isValidIdentifier(ident string) bool {
	ident = strings.TrimSpace(ident)
	if ident == "*" {
		return true
	}
	if identRegex.MatchString(ident) {
		return true
	}
	if dottedRegex.MatchString(ident) {
		return true
	}
	if funcRegex.MatchString(ident) {
		return true
	}
	if aliasRegex.MatchString(ident) {
		return true
	}
	return false
}

func (b *QueryBuilder) quote(ident string) string {
	ident = strings.TrimSpace(ident)
	if ident == "" {
		if b.err == nil {
			b.err = fmt.Errorf("orm: empty identifier")
		}
		return ""
	}
	if !isValidIdentifier(ident) {
		if b.err == nil {
			b.err = fmt.Errorf("orm: invalid identifier %q", ident)
		}
		return ident
	}
	if ident == "*" {
		return "*"
	}
	if m := aliasRegex.FindStringSubmatch(ident); len(m) == 3 {
		return b.quote(m[1]) + " AS " + b.dialect.QuoteIdentifier(m[2])
	}
	if m := funcRegex.FindStringSubmatch(ident); len(m) > 0 {
		return ident
	}
	if strings.Contains(ident, ".") {
		parts := strings.Split(ident, ".")
		quoted := make([]string, len(parts))
		for i, p := range parts {
			if p == "*" {
				quoted[i] = "*"
			} else {
				quoted[i] = b.dialect.QuoteIdentifier(p)
			}
		}
		return strings.Join(quoted, ".")
	}
	return b.dialect.QuoteIdentifier(ident)
}
