package kiw

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const compNamePattern = `(?:[a-zA-Z_][a-zA-Z0-9_-]*:)?[A-Z][a-zA-Z0-9_]*(?:\.[A-Z][a-zA-Z0-9_]*)*`

var (
	selfClosingCompRe = regexp.MustCompile(`(?s)<(` + compNamePattern + `)([^>]*?)\s*/>`)
	openCompTagRe     = regexp.MustCompile(`<(` + compNamePattern + `)([^>]*)>`)
	mustacheCompRe    = regexp.MustCompile(`{{\s*component\s+"([^"]+)"(?:\s+([^{}]+))?\s*}}`)
	mustacheMountRe   = regexp.MustCompile(`{{\s*mount\s+"([^"]+)"\s+"([^"]+)"(?:\s+([^{}]+))?\s*}}`)
	compAttrRe        = regexp.MustCompile("([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|\\{([^}]*)\\}|([^\\s\"'=<>`]+)))?")
	namedSlotRe       = regexp.MustCompile(`(?s)<(?:template|div|section|svelte:fragment)\s+slot=["']([a-zA-Z0-9_-]+)["']>([\s\S]*?)</(?:template|div|section|svelte:fragment)>`)
	namedSlotTagRe    = regexp.MustCompile(`(?s)<slot\s+name=["']([a-zA-Z0-9_-]+)["']>([\s\S]*?)</slot>`)
	namedSlotElemRe   = regexp.MustCompile(`(?s)<slot:([a-zA-Z0-9_-]+)>([\s\S]*?)</slot:([a-zA-Z0-9_-]+)>`)
)

func isNumeric(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

func desugarComponent(name, attrStr, bodyContent, raw string) (string, ComponentCall) {
	call := ComponentCall{
		Name:  name,
		Props: make(map[string]string),
		Raw:   raw,
	}

	trimmedAttr := strings.TrimSpace(attrStr)
	trimmedBody := strings.TrimSpace(bodyContent)

	if trimmedAttr == "" && trimmedBody == "" {
		return fmt.Sprintf(`{{component %q}}`, name), call
	}

	if trimmedAttr == "." && trimmedBody == "" {
		call.Props["."] = "."
		return fmt.Sprintf(`{{component %q .}}`, name), call
	}

	if strings.HasPrefix(trimmedAttr, "(dict") && trimmedBody == "" {
		return fmt.Sprintf(`{{component %q %s}}`, name, trimmedAttr), call
	}

	var dictPairs []string
	cleanAttr := attrStr
	var hydrateMode string
	if trimmedAttr != "" {
		punned := punnedPropRe.FindAllStringSubmatch(attrStr, -1)
		for _, pm := range punned {
			propName := pm[1]
			if propName == "true" || propName == "false" || propName == "nil" {
				continue
			}
			call.Props[propName] = "." + propName
			dictPairs = append(dictPairs, fmt.Sprintf("%q .%s", propName, propName))
		}
		cleanAttr = punnedPropRe.ReplaceAllStringFunc(cleanAttr, func(m string) string {
			sub := punnedPropRe.FindStringSubmatch(m)
			if len(sub) > 1 && (sub[1] == "true" || sub[1] == "false" || sub[1] == "nil") {
				return m
			}
			return " "
		})
	}

	if strings.TrimSpace(cleanAttr) != "" {
		matches := compAttrRe.FindAllStringSubmatch(cleanAttr, -1)
		for _, sm := range matches {
			if len(sm) < 2 || strings.TrimSpace(sm[1]) == "" {
				continue
			}
			key := sm[1]

			// Check for hydration directives (e.g. client:load, client:idle, client:visible, client:hydrate="load", hydrate="load")
			switch key {
			case "client:load":
				hydrateMode = "load"
				continue
			case "client:idle":
				hydrateMode = "idle"
				continue
			case "client:visible":
				hydrateMode = "visible"
				continue
			case "client:hydrate", "hydrate":
				v := ""
				if len(sm) > 2 && sm[2] != "" {
					v = sm[2]
				} else if len(sm) > 3 && sm[3] != "" {
					v = sm[3]
				} else if len(sm) > 5 && sm[5] != "" {
					v = sm[5]
				}
				if v == "" {
					v = "load"
				}
				hydrateMode = v
				continue
			}

			// boolean flag without explicit value (e.g. <Navbar ShowSidebarToggle />)
			if (len(sm) < 3 || (sm[2] == "" && sm[3] == "" && sm[4] == "" && sm[5] == "")) && !strings.Contains(sm[0], "=") {
				call.Props[key] = "true"
				dictPairs = append(dictPairs, fmt.Sprintf("%q true", key))
				continue
			}

			val := ""
			isExpr := false
			if len(sm) > 2 && sm[2] != "" {
				val = sm[2]
				if strings.HasPrefix(val, "{{") && strings.HasSuffix(val, "}}") {
					val = strings.TrimSpace(val[2 : len(val)-2])
					isExpr = true
				} else if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
					val = strings.TrimSpace(val[1 : len(val)-1])
					isExpr = true
					if identPathRe.MatchString(val) && !strings.HasPrefix(val, ".") && !strings.HasPrefix(val, "$") && val != "true" && val != "false" {
						val = toGoField(val, nil)
					}
				}
			} else if len(sm) > 3 && sm[3] != "" {
				val = sm[3]
			} else if len(sm) > 4 && sm[4] != "" {
				val = strings.TrimSpace(sm[4])
				isExpr = true
				if identPathRe.MatchString(val) && !strings.HasPrefix(val, ".") && !strings.HasPrefix(val, "$") && val != "true" && val != "false" {
					val = toGoField(val, nil)
				}
			} else if len(sm) > 5 && sm[5] != "" {
				val = sm[5]
				if val == "true" || val == "false" || isNumeric(val) || strings.HasPrefix(val, ".") || strings.HasPrefix(val, "$") {
					isExpr = true
				}
			}

			call.Props[key] = val
			if isExpr {
				dictPairs = append(dictPairs, fmt.Sprintf("%q %s", key, val))
			} else if val == "true" || val == "false" || isNumeric(val) {
				dictPairs = append(dictPairs, fmt.Sprintf("%q %s", key, val))
			} else {
				dictPairs = append(dictPairs, fmt.Sprintf("%q %q", key, val))
			}
		}
	}

	if trimmedBody != "" {
		// Extract named slots (<div slot="xyz">...</div>, <slot name="xyz">...</slot>, <slot:xyz>...</slot:xyz>)
		for _, re := range []*regexp.Regexp{namedSlotRe, namedSlotTagRe, namedSlotElemRe} {
			matches := re.FindAllStringSubmatch(trimmedBody, -1)
			for _, ns := range matches {
				if re == namedSlotElemRe && len(ns) > 3 && ns[1] != ns[3] {
					continue
				}
				slotName := capitalize(ns[1])
				slotContent := strings.TrimSpace(ns[2])
				if desugared, _ := DesugarTemplate(slotContent); desugared != "" {
					slotContent = desugared
				}
				call.Props[slotName] = slotContent
				dictPairs = append(dictPairs, fmt.Sprintf("%q %q", slotName, slotContent))
			}
			trimmedBody = strings.TrimSpace(re.ReplaceAllString(trimmedBody, ""))
		}

		if trimmedBody != "" {
			// A component may contain other components (<Breadcrumb><BreadcrumbItem/></Breadcrumb>).
			// The body is embedded below as a quoted string literal, so a child tag
			// left un-desugared would reach the browser as literal text instead of a
			// component call. Desugar the children first.
			if desugared, _ := DesugarTemplate(trimmedBody); desugared != "" {
				trimmedBody = desugared
			}
			call.Props["Body"] = trimmedBody
			dictPairs = append(dictPairs, fmt.Sprintf("%q %q", "Body", trimmedBody))
		}
	}

	if hydrateMode != "" {
		call.Hydrate = hydrateMode
		if len(dictPairs) == 0 {
			return fmt.Sprintf(`{{mount %q %q (dict)}}`, name, hydrateMode), call
		}
		return fmt.Sprintf(`{{mount %q %q (dict %s)}}`, name, hydrateMode, strings.Join(dictPairs, " ")), call
	}

	if len(dictPairs) == 0 {
		return fmt.Sprintf(`{{component %q}}`, name), call
	}
	return fmt.Sprintf(`{{component %q (dict %s)}}`, name, strings.Join(dictPairs, " ")), call
}

func findMatchingCloseTag(s, tagName string) int {
	closeTag := "</" + tagName + ">"
	openTagPrefix := "<" + tagName
	depth := 1
	idx := 0
	for depth > 0 && idx < len(s) {
		nextClose := strings.Index(s[idx:], closeTag)
		if nextClose == -1 {
			return -1
		}
		nextOpen := strings.Index(s[idx:], openTagPrefix)
		if nextOpen != -1 && nextOpen < nextClose {
			afterOpen := idx + nextOpen + len(openTagPrefix)
			if afterOpen < len(s) {
				c := s[afterOpen]
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' || c == '/' {
					gtIdx := strings.IndexByte(s[afterOpen:], '>')
					if gtIdx != -1 {
						insideTag := strings.TrimSpace(s[afterOpen : afterOpen+gtIdx])
						if !strings.HasSuffix(insideTag, "/") {
							depth++
						}
						idx = afterOpen + gtIdx + 1
						continue
					}
				}
			}
			idx += nextOpen + len(openTagPrefix)
			continue
		}
		depth--
		if depth == 0 {
			return idx + nextClose
		}
		idx += nextClose + len(closeTag)
	}
	return -1
}

func desugarPairedComponents(src string, pad bool) string {
	res := src
	for {
		m := openCompTagRe.FindStringSubmatchIndex(res)
		if m == nil {
			break
		}
		tagName := res[m[2]:m[3]]
		attrStr := res[m[4]:m[5]]

		// Check if it's actually self-closing with slash at end of attrs: <Tag .../>
		trimmedAttr := strings.TrimSpace(attrStr)
		if strings.HasSuffix(trimmedAttr, "/") {
			realAttr := strings.TrimSuffix(trimmedAttr, "/")
			fullTag := res[m[0]:m[1]]
			replacement, _ := desugarComponent(tagName, realAttr, "", fullTag)
			res = res[:m[0]] + keepLines(fullTag, replacement, pad) + res[m[1]:]
			continue
		}

		closeTag := "</" + tagName + ">"
		closeIdx := findMatchingCloseTag(res[m[1]:], tagName)
		if closeIdx == -1 {
			// No matching closing tag, treat as self-closing
			fullTag := res[m[0]:m[1]]
			replacement, _ := desugarComponent(tagName, attrStr, "", fullTag)
			res = res[:m[0]] + keepLines(fullTag, replacement, pad) + res[m[1]:]
			continue
		}
		closeIdx += m[1]
		bodyContent := res[m[1]:closeIdx]
		fullTag := res[m[0] : closeIdx+len(closeTag)]
		replacement, _ := desugarComponent(tagName, attrStr, bodyContent, fullTag)
		res = res[:m[0]] + keepLines(fullTag, replacement, pad) + res[closeIdx+len(closeTag):]
	}
	return res
}
