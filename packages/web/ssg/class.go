package ssg

import (
	"fmt"
	"strings"
)

// ClassHelper composes HTML class strings from a base class, conditional modifiers,
// and optional trailing raw tokens into one space-joined string (KWL-DSL-CLASS, FRK-DSL-501..504).
//
// Arguments are processed as:
//   - args[0]: base class token (e.g. "btn", "card", "alert"). If non-empty, it is emitted
//     first and establishes the default modifier prefix (e.g. "btn-").
//   - Repeating groups of modifiers:
//   - Explicit triple: (prefix, value, default) where prefix ends with "-", "_", or ":",
//     or base is empty. Emits prefix+value when value is non-empty, else prefix+default.
//   - Implicit pair: (value, default) using base+"-" as prefix. Emits prefix+value when
//     value is non-empty, else prefix+default.
//     Empty modifier groups contribute nothing (FRK-DSL-502).
//   - Trailing raw tokens (e.g. .Class) are emitted verbatim.
//
// Returns a plain string (NOT template.HTML) so html/template preserves attribute-context escaping (FRK-DSL-504).
// Output collapses leading, trailing, and duplicated spaces (FRK-DSL-503).
func ClassHelper(args ...any) string {
	if len(args) == 0 {
		return ""
	}

	var tokens []string
	addToken := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			tokens = append(tokens, s)
		}
	}

	baseStr := ""
	if args[0] != nil && args[0] != false {
		baseStr = strings.TrimSpace(fmt.Sprint(args[0]))
	}
	if baseStr != "" {
		addToken(baseStr)
	}

	defPrefix := ""
	if baseStr != "" {
		if strings.HasSuffix(baseStr, "-") {
			defPrefix = baseStr
		} else {
			defPrefix = baseStr + "-"
		}
	}

	rem := args[1:]
	i := 0
	usedExplicitPrefix := false

	for i < len(rem) {
		// Explicit triple: (prefix, value, default)
		if i+2 < len(rem) {
			pfxStr, isStr := rem[i].(string)
			if (isStr && (strings.HasSuffix(pfxStr, "-") || strings.HasSuffix(pfxStr, "_") || strings.HasSuffix(pfxStr, ":"))) || (baseStr == "" && isStr) {
				usedExplicitPrefix = true
				val := rem[i+1]
				def := rem[i+2]
				valStr := ""
				if val != nil && val != false {
					valStr = strings.TrimSpace(fmt.Sprint(val))
				}
				defStr := ""
				if def != nil && def != false {
					defStr = strings.TrimSpace(fmt.Sprint(def))
				}

				if valStr != "" {
					addToken(pfxStr + valStr)
				} else if defStr != "" {
					addToken(pfxStr + defStr)
				}
				i += 3
				continue
			}
		}

		// If explicit prefix mode was used, subsequent non-triple args are trailing tokens.
		if usedExplicitPrefix {
			if rem[i] != nil && rem[i] != false {
				tStr := strings.TrimSpace(fmt.Sprint(rem[i]))
				if tStr != "" {
					for _, part := range strings.Fields(tStr) {
						addToken(part)
					}
				}
			}
			i++
			continue
		}

		// Implicit pair: (value, default) with defPrefix
		if i+1 < len(rem) && defPrefix != "" {
			val := rem[i]
			def := rem[i+1]
			valStr := ""
			if val != nil && val != false {
				valStr = strings.TrimSpace(fmt.Sprint(val))
			}
			defStr := ""
			if def != nil && def != false {
				defStr = strings.TrimSpace(fmt.Sprint(def))
			}

			if valStr != "" {
				addToken(defPrefix + valStr)
			} else if defStr != "" {
				addToken(defPrefix + defStr)
			}
			i += 2
			continue
		}

		// Trailing token
		if rem[i] != nil && rem[i] != false {
			tStr := strings.TrimSpace(fmt.Sprint(rem[i]))
			if tStr != "" {
				for _, part := range strings.Fields(tStr) {
					addToken(part)
				}
			}
		}
		i++
	}

	return strings.Join(tokens, " ")
}
