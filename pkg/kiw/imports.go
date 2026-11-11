package kiw

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ImportDecl represents a component or module import declaration in a .kiw file.
type ImportDecl struct {
	LocalName    string `json:"localName"`
	ImportedName string `json:"importedName,omitempty"`
	Source       string `json:"source"`
	IsDefault    bool   `json:"isDefault,omitempty"`
	IsNamespace  bool   `json:"isNamespace,omitempty"`
}

// ExportDecl represents an exported component, subcomponent, or symbol.
type ExportDecl struct {
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
	Kind   string `json:"kind"` // "prop", "component", "subcomponent"
}

// PropSpec represents a declared component prop contract.
type PropSpec struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// SlotSpec represents a component slot outlet definition.
type SlotSpec struct {
	Name        string `json:"name"`
	HasFallback bool   `json:"hasFallback,omitempty"`
	Fallback    string `json:"fallback,omitempty"`
}

var (
	// Script / DSL import regexes
	defaultImportRe          = regexp.MustCompile(`(?m)^\s*import\s+([A-Z][a-zA-Z0-9_]*)\s+from\s+['"]([^'"]+)['"]\s*;?`)
	namespaceImportRe        = regexp.MustCompile(`(?m)^\s*import\s+\*\s+as\s+([a-zA-Z0-9_]+)\s+from\s+['"]([^'"]+)['"]\s*;?`)
	namedImportRe            = regexp.MustCompile(`(?m)^\s*import\s*\{([^}]+)\}\s*from\s+['"]([^'"]+)['"]\s*;?`)
	directiveDefaultImportRe = regexp.MustCompile(`(?m)^\s*@import\s+([A-Z][a-zA-Z0-9_]*)\s+from\s+['"]([^'"]+)['"]\s*;?`)
	directiveNamespaceImport = regexp.MustCompile(`(?m)^\s*@import\s+\*\s+as\s+([a-zA-Z0-9_]+)\s+from\s+['"]([^'"]+)['"]\s*;?`)
	directiveNamedImport     = regexp.MustCompile(`(?m)^\s*@import\s*\{([^}]+)\}\s*from\s+['"]([^'"]+)['"]\s*;?`)

	// Export prop regex: export let title = "Default" or export let count: number = 0
	exportLetRe = regexp.MustCompile(`(?m)^\s*export\s+let\s+([a-zA-Z0-9_$]+)(?:\s*:\s*([^=;\n]+))?(?:\s*=\s*([^;\n]+))?\s*;?`)
	// Export subcomponent: export { Header, Body } from "./CardParts.kiw"
	exportNamedRe = regexp.MustCompile(`(?m)^\s*export\s*\{([^}]+)\}(?:\s+from\s+['"]([^'"]+)['"])?\s*;?`)
)

func parseImportsAndExports(src string, fm map[string]any, scriptBlocks []ScriptBlock) (
	imports []ImportDecl,
	exports []ExportDecl,
	props map[string]PropSpec,
	defaults map[string]any,
	slots []SlotSpec,
) {
	props = make(map[string]PropSpec)
	defaults = make(map[string]any)
	slotSet := make(map[string]SlotSpec)

	// 1. Parse from frontmatter
	if fm != nil {
		// Imports from frontmatter (map or list)
		if impRaw, ok := fm["imports"]; ok {
			switch v := impRaw.(type) {
			case map[string]any:
				for local, srcVal := range v {
					srcStr := fmt.Sprint(srcVal)
					isNS := false
					if !strings.HasSuffix(srcStr, ".kiw") {
						// Package imports without .kiw (e.g. @krewire/ui or ui) are namespace imports if local is lowercase or matched
						if strings.HasPrefix(srcStr, "@") || !strings.Contains(srcStr, "/") {
							isNS = (local == strings.ToLower(local)) || !strings.Contains(strings.TrimPrefix(srcStr, "@"), "/")
						}
					}
					imports = append(imports, ImportDecl{
						LocalName:   local,
						Source:      srcStr,
						IsDefault:   !isNS,
						IsNamespace: isNS,
					})
				}
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						base := s
						if idx := strings.LastIndex(s, "/"); idx >= 0 {
							base = s[idx+1:]
						}
						base = strings.TrimSuffix(base, ".kiw")
						imports = append(imports, ImportDecl{
							LocalName: base,
							Source:    s,
							IsDefault: true,
						})
					}
				}
			}
		}

		// Props from frontmatter
		if propsRaw, ok := fm["props"]; ok {
			if pm, ok := propsRaw.(map[string]any); ok {
				for name, pDef := range pm {
					spec := PropSpec{Name: name}
					switch def := pDef.(type) {
					case map[string]any:
						if t, ok := def["type"].(string); ok {
							spec.Type = t
						}
						if req, ok := def["required"].(bool); ok {
							spec.Required = req
						}
						if d, ok := def["default"]; ok {
							spec.Default = fmt.Sprint(d)
							defaults[name] = d
						}
					default:
						spec.Default = fmt.Sprint(def)
						defaults[name] = def
					}
					props[name] = spec
				}
			}
		}

		// Slots from frontmatter
		if slotsRaw, ok := fm["slots"]; ok {
			if sl, ok := slotsRaw.([]any); ok {
				for _, item := range sl {
					name := fmt.Sprint(item)
					slotSet[name] = SlotSpec{Name: name}
				}
			}
		}

		// Exports from frontmatter
		if expRaw, ok := fm["exports"]; ok {
			if em, ok := expRaw.(map[string]any); ok {
				for name, srcVal := range em {
					exports = append(exports, ExportDecl{
						Name:   name,
						Source: fmt.Sprint(srcVal),
						Kind:   "subcomponent",
					})
				}
			}
		}
	}

	// 2. Parse from top-level @import directives in body
	for _, m := range directiveDefaultImportRe.FindAllStringSubmatch(src, -1) {
		imports = append(imports, ImportDecl{
			LocalName: m[1],
			Source:    m[2],
			IsDefault: true,
		})
	}
	for _, m := range directiveNamespaceImport.FindAllStringSubmatch(src, -1) {
		imports = append(imports, ImportDecl{
			LocalName:   m[1],
			Source:      m[2],
			IsNamespace: true,
		})
	}
	for _, m := range directiveNamedImport.FindAllStringSubmatch(src, -1) {
		srcPath := m[2]
		for _, part := range strings.Split(m[1], ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			impName, localName := part, part
			if strings.Contains(part, " as ") {
				sub := strings.Split(part, " as ")
				impName = strings.TrimSpace(sub[0])
				localName = strings.TrimSpace(sub[1])
			}
			imports = append(imports, ImportDecl{
				LocalName:    localName,
				ImportedName: impName,
				Source:       srcPath,
			})
		}
	}

	// 3. Parse from <script> blocks
	for _, sb := range scriptBlocks {
		content := sb.Content

		// Default imports
		for _, m := range defaultImportRe.FindAllStringSubmatch(content, -1) {
			imports = append(imports, ImportDecl{
				LocalName: m[1],
				Source:    m[2],
				IsDefault: true,
			})
		}
		// Namespace imports
		for _, m := range namespaceImportRe.FindAllStringSubmatch(content, -1) {
			imports = append(imports, ImportDecl{
				LocalName:   m[1],
				Source:      m[2],
				IsNamespace: true,
			})
		}
		// Named imports
		for _, m := range namedImportRe.FindAllStringSubmatch(content, -1) {
			srcPath := m[2]
			for _, part := range strings.Split(m[1], ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				impName, localName := part, part
				if strings.Contains(part, " as ") {
					sub := strings.Split(part, " as ")
					impName = strings.TrimSpace(sub[0])
					localName = strings.TrimSpace(sub[1])
				}
				imports = append(imports, ImportDecl{
					LocalName:    localName,
					ImportedName: impName,
					Source:       srcPath,
				})
			}
		}

		// export let prop = ...
		for _, m := range exportLetRe.FindAllStringSubmatch(content, -1) {
			propName := m[1]
			propType := strings.TrimSpace(m[2])
			propDef := strings.TrimSpace(m[3])
			propDef = strings.Trim(propDef, `"'`)

			spec := PropSpec{
				Name:    propName,
				Type:    propType,
				Default: propDef,
			}
			props[propName] = spec
			if propDef != "" {
				defaults[propName] = propDef
			}
			exports = append(exports, ExportDecl{
				Name: propName,
				Kind: "prop",
			})
		}

		// export { Header, Body } from ...
		for _, m := range exportNamedRe.FindAllStringSubmatch(content, -1) {
			srcPath := ""
			if len(m) > 2 {
				srcPath = m[2]
			}
			for _, item := range strings.Split(m[1], ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					exports = append(exports, ExportDecl{
						Name:   item,
						Source: srcPath,
						Kind:   "subcomponent",
					})
				}
			}
		}
	}

	// Discover slots defined inside the markup: <slot ...>
	slotMatches := slotTagRe.FindAllStringSubmatch(src, -1)
	for _, sm := range slotMatches {
		name := "default"
		if len(sm) > 1 && sm[1] != "" {
			name = sm[1]
		}
		fallback := ""
		if len(sm) > 2 {
			fallback = strings.TrimSpace(sm[2])
		}
		slotSet[name] = SlotSpec{
			Name:        name,
			HasFallback: fallback != "",
			Fallback:    fallback,
		}
	}

	for _, s := range slotSet {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Name < slots[j].Name })

	return imports, exports, props, defaults, slots
}

var slotTagRe = regexp.MustCompile(`(?s)<slot(?:\s+name=["']([a-zA-Z0-9_-]+)["'])?[^>]*>(.*?)</slot>`)
