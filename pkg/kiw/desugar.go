package kiw

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	identPathRe       = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(?:\.[a-zA-Z0-9_$]+)*$`)
	eachOpenRe        = regexp.MustCompile(`\{#each\s+([a-zA-Z0-9_$.]+)\s+as\s+([a-zA-Z0-9_$]+)(?:\s*,\s*([a-zA-Z0-9_$]+))?\s*\}`)
	ifTagRe           = regexp.MustCompile(`\{#if\s+([^}]+)\}`)
	elseIfTagRe       = regexp.MustCompile(`\{:else\s+if\s+([^}]+)\}`)
	elseTagRe         = regexp.MustCompile(`\{:else\}`)
	endIfTagRe        = regexp.MustCompile(`\{/if\}`)
	classDirRe        = regexp.MustCompile(`class:([a-zA-Z0-9_-]+)(?:=\{([^}]*)\}|="([^"]*)"|='([^']*)')?`)
	classAttrRe       = regexp.MustCompile(`\bclass=(?:"([^"]*)"|'([^']*)')`)
	tagWithClassDirRe = regexp.MustCompile(`(?s)<([a-zA-Z0-9_-]+)([^>]*?\sclass:[^>]*?)>`)
	punnedPropRe      = regexp.MustCompile(`(?:^|[\s/])\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
)

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toGoField(path string, loopVars map[string]bool) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, ".")
	root := parts[0]

	// Preserve explicit $ variables
	if strings.HasPrefix(root, "$") {
		var res []string
		res = append(res, root)
		for _, p := range parts[1:] {
			res = append(res, capitalize(p))
		}
		return strings.Join(res, ".")
	}

	// Loop variable reference (e.g. item -> $item)
	if loopVars != nil && loopVars[root] {
		var res []string
		res = append(res, "$"+root)
		for _, p := range parts[1:] {
			res = append(res, capitalize(p))
		}
		return strings.Join(res, ".")
	}

	// Preserve leading dot if already present
	if strings.HasPrefix(root, ".") {
		var res []string
		res = append(res, root)
		for _, p := range parts[1:] {
			res = append(res, capitalize(p))
		}
		return strings.Join(res, ".")
	}

	// In loop context, outer variables need $. prefix to escape shifting scope
	prefix := "."
	if len(loopVars) > 0 {
		prefix = "$."
	}

	var res []string
	res = append(res, prefix+capitalize(root))
	for _, p := range parts[1:] {
		res = append(res, capitalize(p))
	}
	return strings.Join(res, ".")
}

// desugarInterpolation converts single-brace {var} and {path.field} to Go template {{.Path.Field}}.
func desugarInterpolation(src string, loopVars map[string]bool) string {
	buf := GetBuffer()
	defer PutBuffer(buf)
	n := len(src)
	i := 0

	for i < n {
		// Preserve double-brace {{...}} verbatim
		if i+1 < n && src[i] == '{' && src[i+1] == '{' {
			end := strings.Index(src[i+2:], "}}")
			if end != -1 {
				buf.WriteString(src[i : i+2+end+2])
				i += 2 + end + 2
				continue
			}
		}

		// Check for single brace {expr}
		if src[i] == '{' && (i+1 < n && src[i+1] != '{') {
			closeIdx := -1
			depth := 1
			for j := i + 1; j < n; j++ {
				if src[j] == '{' {
					depth++
				} else if src[j] == '}' {
					depth--
					if depth == 0 {
						closeIdx = j
						break
					}
				}
			}

			if closeIdx != -1 {
				content := src[i+1 : closeIdx]
				trimmed := strings.TrimSpace(content)
				if identPathRe.MatchString(trimmed) && trimmed != "true" && trimmed != "false" && trimmed != "nil" && trimmed != "null" {
					goField := toGoField(trimmed, loopVars)
					buf.WriteString("{{")
					buf.WriteString(goField)
					buf.WriteString("}}")
					i = closeIdx + 1
					continue
				}
			}
		}

		buf.WriteByte(src[i])
		i++
	}

	return buf.String()
}

// desugarClassDirectives handles class:name={cond} on HTML tags.
func desugarClassDirectives(src string, pad bool) string {
	return tagWithClassDirRe.ReplaceAllStringFunc(src, func(tagMatch string) string {
		sub := tagWithClassDirRe.FindStringSubmatch(tagMatch)
		if len(sub) < 3 {
			return tagMatch
		}
		tagName := sub[1]
		attrContent := sub[2]

		cleanAttr := classDirRe.ReplaceAllString(attrContent, "")
		cleanAttr = strings.TrimRight(cleanAttr, " \t\r\n")

		hasClassAttr := classAttrRe.MatchString(cleanAttr)
		var condClasses []string
		classDirs := classDirRe.FindAllStringSubmatch(attrContent, -1)
		for i, m := range classDirs {
			className := m[1]
			expr := ""
			if len(m) > 2 && m[2] != "" {
				expr = m[2]
			} else if len(m) > 3 && m[3] != "" {
				expr = m[3]
			} else if len(m) > 4 && m[4] != "" {
				expr = m[4]
			}
			expr = strings.TrimSpace(expr)
			if expr == "" {
				expr = className
			}
			goCond := desugarInfixExpr(expr, nil)
			space := " "
			if i == 0 && !hasClassAttr {
				space = ""
			}
			condClasses = append(condClasses, fmt.Sprintf("{{if %s}}%s%s{{end}}", goCond, space, className))
		}

		joinedConds := strings.Join(condClasses, "")
		if classMatch := classAttrRe.FindStringSubmatchIndex(cleanAttr); classMatch != nil {
			valStart := classMatch[2]
			valEnd := classMatch[3]
			quote := `"`
			if valStart == -1 {
				valStart = classMatch[4]
				valEnd = classMatch[5]
				quote = `'`
			}
			origVal := cleanAttr[valStart:valEnd]
			newClassVal := fmt.Sprintf(`class=%s%s%s%s`, quote, origVal, joinedConds, quote)
			cleanAttr = cleanAttr[:classMatch[0]] + newClassVal + cleanAttr[classMatch[1]:]
		} else {
			cleanAttr += fmt.Sprintf(` class="%s"`, joinedConds)
		}

		return keepLines(tagMatch, fmt.Sprintf("<%s%s>", tagName, cleanAttr), pad)
	})
}

// eachClose terminates an {#each} block.
const eachClose = "{/each}"

// matchingEachClose returns the index in src of the {/each} that closes a
// block whose body starts at from, skipping nested {#each} blocks.
func matchingEachClose(src string, from int) int {
	depth := 1
	for i := from; i < len(src); {
		open := eachOpenRe.FindStringIndex(src[i:])
		closeIdx := strings.Index(src[i:], eachClose)
		if closeIdx < 0 {
			return -1
		}
		if open != nil && open[0] < closeIdx {
			depth++
			i += open[1]
			continue
		}
		depth--
		if depth == 0 {
			return i + closeIdx
		}
		i += closeIdx + len(eachClose)
	}
	return -1
}

// desugarLoops translates {#each items as item, index}...{/each} blocks.
// Blocks nest: an inner block sees the variables of every enclosing loop, so
// {#each rows as row}{#each row as cell}...{/each}{/each} ranges over $row.
// A block without a matching {/each} is left untouched (Lint reports it).
func desugarLoops(src string, scope map[string]bool, pad bool) string {
	var b strings.Builder
	rest := src
	for {
		m := eachOpenRe.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		end := matchingEachClose(rest, m[1])
		if end < 0 {
			break
		}
		itemsExpr := strings.TrimSpace(rest[m[2]:m[3]])
		itemVar := strings.TrimSpace(rest[m[4]:m[5]])
		indexVar := ""
		if m[6] != -1 && m[7] != -1 {
			indexVar = strings.TrimSpace(rest[m[6]:m[7]])
		}

		inner := make(map[string]bool, len(scope)+2)
		for k, v := range scope {
			inner[k] = v
		}
		inner[itemVar] = true
		if indexVar != "" {
			inner[indexVar] = true
		}

		goItems := toGoField(itemsExpr, scope)
		header := fmt.Sprintf("{{range $%s := %s}}", itemVar, goItems)
		if indexVar != "" {
			header = fmt.Sprintf("{{range $%s, $%s := %s}}", indexVar, itemVar, goItems)
		}
		b.WriteString(rest[:m[0]])
		b.WriteString(keepLines(rest[m[0]:m[1]], header, pad))
		b.WriteString(desugar(rest[m[1]:end], inner, pad))
		b.WriteString("{{end}}")
		rest = rest[end+len(eachClose):]
	}
	b.WriteString(rest)
	return b.String()
}

// keepLines returns repl followed by the newlines it dropped from orig when
// pad is set. Line-preserving desugaring (used only to locate errors) keeps
// every template line at its .kiw source line this way.
func keepLines(orig, repl string, pad bool) string {
	if !pad {
		return repl
	}
	if d := strings.Count(orig, "\n") - strings.Count(repl, "\n"); d > 0 {
		return repl + strings.Repeat("\n", d)
	}
	return repl
}

// desugarConditionals converts {#if expr}, {:else if expr}, {:else}, {/if}.
func desugarConditionals(src string, loopVars map[string]bool, pad bool) string {
	res := ifTagRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := ifTagRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		expr := strings.TrimSpace(sub[1])
		return keepLines(match, fmt.Sprintf("{{if %s}}", desugarInfixExpr(expr, loopVars)), pad)
	})

	res = elseIfTagRe.ReplaceAllStringFunc(res, func(match string) string {
		sub := elseIfTagRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		expr := strings.TrimSpace(sub[1])
		return keepLines(match, fmt.Sprintf("{{else if %s}}", desugarInfixExpr(expr, loopVars)), pad)
	})

	res = elseTagRe.ReplaceAllString(res, "{{else}}")
	res = endIfTagRe.ReplaceAllString(res, "{{end}}")
	return res
}

var (
	slotFallbackRe = regexp.MustCompile(`(?s)<slot(?:\s+name=["']([a-zA-Z0-9_-]+)["'])?\s*>(.*?)</slot>`)
	slotEmptyRe    = regexp.MustCompile(`<slot(?:\s+name=["']([a-zA-Z0-9_-]+)["'])?\s*/>`)
	snippetDefRe   = regexp.MustCompile(`(?s)\{#snippet\s+([a-zA-Z0-9_$]+)\s*(?:\(([^)]*)\))?\s*\}([\s\S]*?)\{/snippet\}`)
	renderCallRe   = regexp.MustCompile(`\{@render\s+([a-zA-Z0-9_$]+)\s*(?:\(([^)]*)\))?\s*\}`)
)

func desugarSlots(src string, pad bool) string {
	res := slotFallbackRe.ReplaceAllStringFunc(src, func(m string) string {
		sub := slotFallbackRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		slotName := sub[1]
		fallback := sub[2]
		targetField := ".Body"
		if slotName != "" {
			targetField = "." + capitalize(slotName)
			if strings.ContainsAny(slotName, "-_") {
				parts := strings.FieldsFunc(slotName, func(r rune) bool { return r == '-' || r == '_' })
				for i := range parts {
					parts[i] = capitalize(parts[i])
				}
				targetField = "." + strings.Join(parts, "")
			}
		}

		trimmedFallback := strings.TrimSpace(fallback)
		if trimmedFallback == "" {
			return keepLines(m, fmt.Sprintf("{{%s}}", targetField), pad)
		}
		return keepLines(m, fmt.Sprintf("{{if %s}}{{%s}}{{else}}%s{{end}}", targetField, targetField, fallback), pad)
	})

	res = slotEmptyRe.ReplaceAllStringFunc(res, func(m string) string {
		sub := slotEmptyRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		slotName := sub[1]
		targetField := ".Body"
		if slotName != "" {
			targetField = "." + capitalize(slotName)
			if strings.ContainsAny(slotName, "-_") {
				parts := strings.FieldsFunc(slotName, func(r rune) bool { return r == '-' || r == '_' })
				for i := range parts {
					parts[i] = capitalize(parts[i])
				}
				targetField = "." + strings.Join(parts, "")
			}
		}
		return keepLines(m, fmt.Sprintf("{{%s}}", targetField), pad)
	})

	return res
}

func desugarSnippets(src string, loopVars map[string]bool, pad bool) string {
	snippetParams := make(map[string][]string)

	res := snippetDefRe.ReplaceAllStringFunc(src, func(m string) string {
		sub := snippetDefRe.FindStringSubmatch(m)
		if len(sub) < 4 {
			return m
		}
		name := sub[1]
		paramsRaw := sub[2]
		body := sub[3]

		var paramList []string
		scope := make(map[string]bool, len(loopVars))
		for k, v := range loopVars {
			scope[k] = v
		}

		if strings.TrimSpace(paramsRaw) != "" {
			for _, p := range strings.Split(paramsRaw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					paramList = append(paramList, p)
					scope[p] = true
				}
			}
		}
		snippetParams[name] = paramList

		// Desugar snippet body
		desugaredBody := desugar(body, scope, pad)
		replacement := fmt.Sprintf(`{{define %q}}%s{{end}}`, name, desugaredBody)
		return keepLines(m, replacement, pad)
	})

	res = renderCallRe.ReplaceAllStringFunc(res, func(m string) string {
		sub := renderCallRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		name := sub[1]
		argsRaw := ""
		if len(sub) > 2 {
			argsRaw = strings.TrimSpace(sub[2])
		}

		if argsRaw == "" {
			return keepLines(m, fmt.Sprintf(`{{template %q .}}`, name), pad)
		}

		params := snippetParams[name]
		args := splitArgs(argsRaw)
		var pairs []string

		for i, arg := range args {
			arg = strings.TrimSpace(arg)
			if arg == "" {
				continue
			}
			key := ""
			val := arg
			if strings.Contains(arg, "=") {
				kv := strings.SplitN(arg, "=", 2)
				key = capitalize(strings.TrimSpace(kv[0]))
				val = strings.TrimSpace(kv[1])
			} else if i < len(params) {
				key = capitalize(params[i])
			} else {
				key = fmt.Sprintf("Arg%d", i)
			}

			// Format val for dict
			if identPathRe.MatchString(val) && val != "true" && val != "false" && !strings.HasPrefix(val, ".") && !strings.HasPrefix(val, "$") && !isNumeric(val) {
				val = toGoField(val, loopVars)
			} else if !strings.HasPrefix(val, `"`) && !strings.HasPrefix(val, `'`) && !isNumeric(val) && val != "true" && val != "false" && !strings.HasPrefix(val, ".") && !strings.HasPrefix(val, "$") {
				val = fmt.Sprintf("%q", val)
			}
			pairs = append(pairs, fmt.Sprintf("%q %s", key, val))
		}

		if len(pairs) == 0 {
			return keepLines(m, fmt.Sprintf(`{{template %q .}}`, name), pad)
		}
		return keepLines(m, fmt.Sprintf(`{{template %q (dict %s)}}`, name, strings.Join(pairs, " ")), pad)
	})

	return res
}

func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	depth := 0
	inQuote := rune(0)

	for _, r := range s {
		switch {
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			}
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			inQuote = r
			cur.WriteRune(r)
		case r == '(' || r == '{' || r == '[':
			depth++
			cur.WriteRune(r)
		case r == ')' || r == '}' || r == ']':
			if depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case r == ',' && depth == 0:
			args = append(args, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		args = append(args, strings.TrimSpace(cur.String()))
	}
	return args
}
