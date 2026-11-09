package kiw

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var (
	blockTokRe   = regexp.MustCompile(`\{(#if|#each|:else\s+if|:else|/if|/each)\b[^}]*\}`)
	compStartRe  = regexp.MustCompile(`<[A-Z][a-zA-Z0-9_]*`)
	rawBlockRe   = regexp.MustCompile(`(?is)<(style|script)[^>]*>.*?</(?:style|script)>`)
	yamlLineErrR = regexp.MustCompile(`line (\d+): (.*)`)
)

// Lint reports .kiw mistakes that desugaring cannot recover from, positioned in
// src: invalid YAML frontmatter, control blocks inside component tag
// attributes, and unbalanced {#if}/{#each} blocks. file is used in messages.
func Lint(file, src string) []Diagnostic {
	var out []Diagnostic
	at := func(off int, msg, hint string) {
		line := strings.Count(src[:off], "\n") + 1
		start := strings.LastIndexByte(src[:off], '\n') + 1
		out = append(out, Diagnostic{File: file, Line: line, Col: utf8.RuneCountInString(src[start:off]) + 1, Message: msg, Hint: hint})
	}

	// Frontmatter: ParseKiw ignores invalid YAML, which silently drops page data.
	markup := src
	if trimmed := strings.TrimLeft(src, "\r\n\t "); strings.HasPrefix(trimmed, "---") {
		lead := len(src) - len(trimmed)
		if end := strings.Index(trimmed[3:], "\n---"); end >= 0 {
			var fm map[string]any
			if err := yaml.Unmarshal([]byte(trimmed[3:3+end]), &fm); err != nil {
				line := 1
				msg := err.Error()
				if m := yamlLineErrR.FindStringSubmatch(msg); m != nil {
					fmt.Sscan(m[1], &line)
					msg = m[2]
				}
				off := lead
				for i := 1; i < line && off < len(src); i++ { // yaml line 1 is the rest of the --- line
					if nl := strings.IndexByte(src[off:], '\n'); nl >= 0 {
						off += nl + 1
					}
				}
				at(off, "invalid frontmatter: "+msg, "frontmatter is YAML between the leading --- lines")
			}
			markup = strings.Repeat(" ", lead+3+end) + trimmed[3+end:]
		}
	}
	// Blank out <style>/<script> content, keeping offsets.
	markup = rawBlockRe.ReplaceAllStringFunc(markup, func(m string) string {
		return strings.Map(func(r rune) rune {
			if r == '\n' {
				return r
			}
			return ' '
		}, m)
	})

	// Control blocks inside component attributes.
	for _, loc := range compStartRe.FindAllStringIndex(markup, -1) {
		end := tagEnd(markup, loc[1])
		attrs := markup[loc[1]:end]
		if bm := blockTokRe.FindStringIndex(attrs); bm != nil {
			tag := markup[loc[0]+1 : loc[1]]
			at(loc[1]+bm[0], fmt.Sprintf("%s inside an attribute of <%s>: component props cannot contain control blocks", attrs[bm[0]:bm[1]], tag),
				"pass the value as a prop (label={label}) and branch inside the component, or wrap the whole tag in {#if}…{:else}…{/if}")
		}
	}

	// Balanced blocks.
	type open struct {
		kind string
		off  int
	}
	var stack []open
	for _, loc := range blockTokRe.FindAllStringSubmatchIndex(markup, -1) {
		tok := markup[loc[2]:loc[3]]
		switch {
		case tok == "#if" || tok == "#each":
			stack = append(stack, open{tok[1:], loc[0]})
		case strings.HasPrefix(tok, ":else"):
			isElseIf := tok != ":else"
			if len(stack) == 0 || (isElseIf && stack[len(stack)-1].kind != "if") {
				at(loc[0], fmt.Sprintf("{%s} without an enclosing {#if}", tok), "")
			}
		default: // /if or /each
			want := tok[1:]
			if len(stack) == 0 {
				at(loc[0], fmt.Sprintf("{%s} without a matching {#%s}", tok, want), "")
				continue
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if top.kind != want {
				at(loc[0], fmt.Sprintf("{%s} closes {#%s} opened at line %d", tok, top.kind, strings.Count(src[:top.off], "\n")+1),
					fmt.Sprintf("close {#%s} with {/%s} first", top.kind, top.kind))
			}
		}
	}
	for _, o := range stack {
		at(o.off, fmt.Sprintf("{#%s} is never closed", o.kind), fmt.Sprintf("add {/%s}", o.kind))
	}
	return out
}

// tagEnd returns the offset of the '>' that ends the tag whose attributes start
// at from, skipping quoted values and {…} expressions.
func tagEnd(s string, from int) int {
	var quote byte
	depth := 0
	for i := from; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '{':
			depth++
		case c == '}':
			if depth > 0 {
				depth--
			}
		case c == '>' && depth == 0:
			return i
		}
	}
	return len(s)
}
