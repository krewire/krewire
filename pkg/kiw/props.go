package kiw

import (
	"strings"
	"unicode"
)

// initialisms are prop names that Go-style templates spell in capitals
// (.ID, .URL); a lower-case attribute (id="x") also resolves to them.
var initialisms = map[string]bool{"id": true, "url": true, "uri": true, "api": true, "html": true, "css": true, "svg": true}

// PropAliases returns the keys under which a component prop passed as name is
// visible to the component template, implementing the lenient prop-casing
// convention:
//
//   - the name as written: variant, Variant
//   - first letter upper and lower: Variant <-> variant, so {variant} and
//     .Variant read the same prop
//   - kebab-case and snake_case also as camelCase and PascalCase:
//     aria-label -> ariaLabel, AriaLabel
//   - common initialisms in capitals: id -> ID, url -> URL
//
// The name as written is always first. Earlier keys win when two props alias
// each other, so an explicit Variant= is never overwritten by variant=.
func PropAliases(name string) []string {
	if name == "" {
		return nil
	}
	out := []string{name}
	add := func(s string) {
		for _, v := range out {
			if v == s {
				return
			}
		}
		out = append(out, s)
	}
	base := name
	if strings.ContainsAny(name, "-_") {
		parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
		for i := 1; i < len(parts); i++ {
			parts[i] = upperFirst(parts[i])
		}
		base = strings.Join(parts, "")
	}
	add(upperFirst(base))
	add(lowerFirstRune(base))
	if initialisms[strings.ToLower(base)] {
		add(strings.ToUpper(base))
	}
	return out
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func lowerFirstRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
