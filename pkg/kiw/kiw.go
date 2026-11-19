// Package kiw parses the `.kiw` module format: YAML frontmatter, <style> and
// <script> blocks, <markdown>, and JSX-like component tags.
//
// It lives in libs because the parser is a leaf concern: pkg/web/ssg needs
// it to load a file-based site, and it must never be forced to reach upward into
// the devtool for it (see the strict downward layering rules). Holding the parser here keeps the dependency arrow pointing down and
// lets every layer above — framework, kiw, ui — parse a .kiw file without a
// cycle. kiw/dsl remains as a thin alias of this package.
package kiw

import (
	"bytes"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/krewire/krewire/pkg/markdown"
	"gopkg.in/yaml.v3"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// GetBuffer acquires a buffer from the package sync.Pool.
func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the package sync.Pool.
func PutBuffer(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bufferPool.Put(buf)
	}
}

var (
	styleRe    = regexp.MustCompile(`(?is)<style([^>]*)>(.*?)</style>`)
	scriptRe   = regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	markdownRe = regexp.MustCompile(`(?is)<markdown[^>]*>(.*?)</markdown>`)
	attrRe     = regexp.MustCompile("([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|([^\\s\"'=<>`]+)))?")
)

// StyleBlock holds a scoped style block with its attributes.
type StyleBlock struct {
	Lang    string `json:"lang"`
	Scoped  bool   `json:"scoped"`
	Content string `json:"content"`
}

// ScriptBlock holds a script block with tier attributes (FRK-DSL-030/031).
// Lang is js|ts|go|rust, Hydrate is load|idle|visible, Server/Compute flags.
type ScriptBlock struct {
	Lang    string `json:"lang"`
	Hydrate string `json:"hydrate"`
	Server  bool   `json:"server"`
	Compute bool   `json:"compute"`
	Content string `json:"content"`
}

// ComponentCall records a parsed component invocation in a .kiw template,
// whether written as {{component "Name" ...}}, {{mount "Name" "mode" ...}},
// or <Name ... /> with optional hydration directives.
type ComponentCall struct {
	Name      string            `json:"name"`
	Hydrate   string            `json:"hydrate,omitempty"`
	Framework string            `json:"framework,omitempty"`
	Props     map[string]string `json:"props,omitempty"`
	Raw       string            `json:"raw"`
}

// KiwModule is the parsed result of a .kiw file.
// It is JSON-serializable and intentionally mirrors the JS parser output
// so the same .kiw file can be consumed from Go (html/template) and from
// JS/TS (string templates) without a custom toolchain.
type KiwModule struct {
	Frontmatter    map[string]any      `yaml:",inline" json:"frontmatter"`
	Body           string              `json:"body"`
	Styles         []string            `json:"styles"`
	Scripts        []string            `json:"scripts"`
	StyleBlocks    []StyleBlock        `json:"styleBlocks"`
	ScriptBlocks   []ScriptBlock       `json:"scriptBlocks"`
	Markdown       []string            `json:"markdown"`
	Components     []ComponentCall     `json:"components,omitempty"`
	ComponentNames []string            `json:"componentNames,omitempty"`
	Imports        []ImportDecl        `json:"imports,omitempty"`
	Exports        []ExportDecl        `json:"exports,omitempty"`
	Props          map[string]PropSpec `json:"props,omitempty"`
	Defaults       map[string]any      `json:"defaults,omitempty"`
	Slots          []SlotSpec          `json:"slots,omitempty"`
	Raw            string              `json:"-"`
}

func parseAttrs(tag string) map[string]string {
	m := map[string]string{}
	for _, sm := range attrRe.FindAllStringSubmatch(tag, -1) {
		if len(sm) < 2 {
			continue
		}
		key := strings.ToLower(sm[1])
		val := ""
		if len(sm) > 2 && sm[2] != "" {
			val = sm[2]
		} else if len(sm) > 3 && sm[3] != "" {
			val = sm[3]
		} else if len(sm) > 4 && sm[4] != "" {
			val = sm[4]
		}
		m[key] = val
	}
	return m
}

// ParseKiw parses a .kiw file content into a KiwModule.
//
// Format (Astro-like, but YAML frontmatter for Go/JS native):
//
//	---
//	title: Landing
//	layout: Base
//	---
//	<h1>{{.Title}}</h1>
//	<Navbar />
//	<style>h1{color:red}</style>
//	<script>console.log(1)</script>
//
// Frontmatter is optional YAML between leading ---\n ... ---\n.
// Body is html/template source with zero or more top-level <style> and <script>
// blocks extracted as scoped CSS / client JS.
// Both {{component "Name"}} and <ComponentName /> are recognized and desugared
// to uniform component calls.
func ParseKiw(src string) (*KiwModule, error) {
	return parseKiw(src, false), nil
}

// parseKiw implements ParseKiw. With pad set it produces a line-preserving
// Body for diagnostics: the frontmatter and removed blocks become blank lines
// and the body is not trimmed, so body line N is source line N.
func parseKiw(src string, pad bool) *KiwModule {
	m := &KiwModule{
		Frontmatter: map[string]any{},
		Raw:         src,
	}
	body := src

	if strings.HasPrefix(strings.TrimSpace(src), "---") {
		trimmed := strings.TrimLeft(src, "\r\n\t ")
		if strings.HasPrefix(trimmed, "---") {
			rest := trimmed[3:]
			// find closing ---\n
			idx := strings.Index(rest, "\n---")
			if idx >= 0 {
				fmRaw := rest[:idx]
				body = rest[idx+4:]
				// strip leading newline after closing ---
				body = strings.TrimLeft(body, "\r\n")
				var fm map[string]any
				if err := yaml.Unmarshal([]byte(fmRaw), &fm); err == nil && fm != nil {
					m.Frontmatter = fm
				}
				if pad {
					body = strings.Repeat("\n", strings.Count(src[:len(src)-len(body)], "\n")) + body
				}
			}
		}
	}

	styles := styleRe.FindAllStringSubmatch(body, -1)
	for _, sm := range styles {
		if len(sm) < 3 {
			continue
		}
		attrs := parseAttrs(sm[1])
		content := strings.TrimSpace(sm[2])
		m.Styles = append(m.Styles, content)
		lang := attrs["lang"]
		if lang == "" {
			lang = "css"
		}
		m.StyleBlocks = append(m.StyleBlocks, StyleBlock{
			Lang:    strings.ToLower(lang),
			Scoped:  false,
			Content: content,
		})
		if _, ok := attrs["scoped"]; ok {
			m.StyleBlocks[len(m.StyleBlocks)-1].Scoped = true
		}
	}
	body = styleRe.ReplaceAllStringFunc(body, func(match string) string { return keepLines(match, "", pad) })

	scripts := scriptRe.FindAllStringSubmatch(body, -1)
	for _, sm := range scripts {
		if len(sm) < 3 {
			continue
		}
		attrs := parseAttrs(sm[1])
		content := strings.TrimSpace(sm[2])
		// External HTML script tags (e.g. <script src="/assets/app.js"></script>)
		// without inline content should remain in the template markup.
		if attrs["src"] != "" && content == "" {
			continue
		}
		m.Scripts = append(m.Scripts, content)
		lang := strings.ToLower(attrs["lang"])
		hydrate := strings.ToLower(attrs["hydrate"])
		_, hasServer := attrs["server"]
		_, hasCompute := attrs["compute"]
		if lang == "" {
			lang = "js"
		}
		if lang == "js" || lang == "ts" || lang == "go" {
			if !hasServer && !hasCompute && hydrate == "" {
				hydrate = "load"
			}
		}
		m.ScriptBlocks = append(m.ScriptBlocks, ScriptBlock{
			Lang:    lang,
			Hydrate: hydrate,
			Server:  hasServer,
			Compute: hasCompute,
			Content: content,
		})
	}
	body = scriptRe.ReplaceAllStringFunc(body, func(match string) string {
		sm := scriptRe.FindStringSubmatch(match)
		if len(sm) >= 3 {
			attrs := parseAttrs(sm[1])
			content := strings.TrimSpace(sm[2])
			if attrs["src"] != "" && content == "" {
				return match
			}
		}
		return keepLines(match, "", pad)
	})

	markdowns := markdownRe.FindAllStringSubmatch(body, -1)
	for _, mm := range markdowns {
		if len(mm) > 1 {
			m.Markdown = append(m.Markdown, strings.TrimSpace(mm[1]))
		}
	}
	body = markdownRe.ReplaceAllStringFunc(body, func(match string) string {
		sub := markdownRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		inner := strings.TrimSpace(sub[1])
		if inner == "" {
			return keepLines(match, "", pad)
		}
		rendered, _ := markdown.Render([]byte(inner))
		if pad {
			return keepLines(match, strings.ReplaceAll(rendered, "\n", " "), pad)
		}
		return "\n" + strings.TrimSpace(rendered) + "\n"
	})

	// Extract and parse imports, exports, props, defaults, and slots
	imports, exports, props, defaults, slots := parseImportsAndExports(src, m.Frontmatter, m.ScriptBlocks)
	m.Imports = imports
	m.Exports = exports
	m.Props = props
	m.Defaults = defaults
	m.Slots = slots

	// Strip top-level @import directives from body
	body = directiveDefaultImportRe.ReplaceAllString(body, "")
	body = directiveNamespaceImport.ReplaceAllString(body, "")
	body = directiveNamedImport.ReplaceAllString(body, "")

	// Desugar JSX-like component tags (<ComponentName ... />) to {{component "ComponentName" ...}}
	body = desugar(body, nil, pad)
	if pad {
		m.Body = body
		return m
	}

	// Extract all component invocations (both original {{component ...}}, {{mount ...}}, and desugared <Tag />)
	nameSet := make(map[string]bool)
	for _, match := range mustacheCompRe.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		compName := match[1]
		m.Components = append(m.Components, ComponentCall{
			Name: compName,
			Raw:  match[0],
		})
		nameSet[compName] = true
	}
	for _, match := range mustacheMountRe.FindAllStringSubmatch(body, -1) {
		if len(match) < 4 {
			continue
		}
		framework := strings.ToLower(match[1])
		compName := match[2]
		hydrate := match[3]
		m.Components = append(m.Components, ComponentCall{
			Name:      compName,
			Hydrate:   hydrate,
			Framework: framework,
			Raw:       match[0],
		})
		nameSet[compName] = true
	}

	for name := range nameSet {
		m.ComponentNames = append(m.ComponentNames, name)
	}
	sort.Strings(m.ComponentNames)

	m.Body = strings.TrimSpace(body)
	return m
}

// ParseKiwFile reads and parses a .kiw file at path.
func ParseKiwFile(path string) (*KiwModule, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseKiw(string(b))
}
