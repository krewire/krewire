package book

import (
	"path"
	"regexp"
	"strings"
)

// Default include/exclude rules. Every Markdown file is included; README
// notes (developer-internal) are excluded from builds by default.
var (
	defaultInclude = []string{"**/*.md"}
	defaultExclude = []string{"**/README.md", "**/readme.md"}
)

// resolveRules returns the effective rules: nil selects the defaults; an
// empty non-nil slice disables that side of the filter (nil matchers).
func resolveRules(include, exclude []string) (inc, exc *matcher) {
	if include == nil {
		include = defaultInclude
	}
	if exclude == nil {
		exclude = defaultExclude
	}
	if len(include) == 0 {
		inc = nil
	} else {
		inc = newMatcher(include)
	}
	if len(exclude) == 0 {
		exc = nil
	} else {
		exc = newMatcher(exclude)
	}
	return inc, exc
}

// accepts reports whether rel (slash-separated, relative to the content
// root) passes the include set and not the exclude set. nil rules select
// the defaults.
func accepts(include, exclude []string, rel string) bool {
	inc, exc := resolveRules(include, exclude)
	if inc != nil && !inc.matches(rel) {
		return false
	}
	return exc == nil || !exc.matches(rel)
}

// matcher holds compiled glob patterns for full-path matching.
type matcher struct {
	res []*regexp.Regexp
}

// newMatcher compiles each pattern against slash-separated paths.
func newMatcher(patterns []string) *matcher {
	m := &matcher{res: make([]*regexp.Regexp, 0, len(patterns))}
	for _, p := range patterns {
		m.res = append(m.res, compileGlob(p))
	}
	return m
}

// matches reports whether name matches any compiled pattern.
func (m *matcher) matches(name string) bool {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	for _, re := range m.res {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// compileGlob translates a glob pattern into an anchored regexp. Patterns
// support ** (any number of path segments), * (within one segment),
// ? (one character); all other characters are literal.
func compileGlob(pattern string) *regexp.Regexp {
	pattern = strings.TrimPrefix(path.Clean("/"+pattern), "/")
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:[^/]*/)*")
					i += 2
				} else {
					b.WriteString(".*")
					i++
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return regexp.MustCompile(`^\z.`)
	}
	return re
}
