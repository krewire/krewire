// Package markdown provides the shared Goldmark-based Markdown renderer for
// the Krewire ecosystem. Both mdbind (book) and framework/web/ssg + dsl use it
// so a docs site can start as a lightweight manuscript and progressively
// enhance to a full ssg site without re-parsing or duplicated dependencies.
// It is the leaf Package that allows mdbind and framework to be depended on
// together (KWF-M8K2Q, KWM-FX9H2).
package markdown

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
)

// ErrSanitize reports a malformed HTML fragment. It never occurs for
// well-formed Goldmark output; it exists so callers can distinguish an
// internal failure from a policy decision to strip content.
var ErrSanitize = errors.New("markdown: sanitize")

var gold = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(htmlrenderer.WithUnsafe()),
)

// Render converts Markdown src to HTML using Goldmark GFM + AutoHeadingID.
//
// SECURITY: raw HTML is preserved because the Krewire ecosystem relies on it
// (`.kiw` components, framework/dsl, mdbind books). Render therefore trusts its
// input and must only receive Markdown authored by the site owner. It still
// strips script-bearing URLs, so a Markdown document from an untrusted author
// cannot smuggle `javascript:` payloads through link syntax. To render
// user-generated or CMS-supplied Markdown, use RenderSafe instead, which
// sanitizes the HTML output against an allowlist.
func Render(src []byte) (string, error) {
	var buf bytes.Buffer
	if err := gold.Convert(src, &buf); err != nil {
		return "", err
	}
	out := buf.String()
	return StripScriptURLs(out)
}

// RenderSafe converts Markdown src to HTML and sanitizes the result, allowing
// only known-safe tags and attributes. Use this for any Markdown that originates
// outside the repository owner: comments, CMS content, pull-request bodies, or
// user profiles. Raw HTML such as <script>, <iframe>, and event handlers
// (onerror, onload, ...) is removed while structural markup survives.
func RenderSafe(src []byte) (string, error) {
	var buf bytes.Buffer
	if err := gold.Convert(src, &buf); err != nil {
		return "", err
	}
	return Sanitize(buf.String())
}

var absLinkRe = regexp.MustCompile(`(href|src)="/([^"]*)"`)

// RenderWithBase converts Markdown src to HTML and rewrites absolute
// href/src="/*" links to be resolved under base (e.g. "/guide/").
// Page links keep their extensionless form; only the site root keeps the
// trailing slash. When base is "/" or empty no rewriting occurs.
func RenderWithBase(src []byte, base string) (string, error) {
	html, err := Render(src)
	if err != nil {
		return "", err
	}
	return prefixLinks(html, base), nil
}

// RenderSafeWithBase behaves like RenderSafe and then rewrites absolute links
// under base.
func RenderSafeWithBase(src []byte, base string) (string, error) {
	html, err := RenderSafe(src)
	if err != nil {
		return "", err
	}
	return prefixLinks(html, base), nil
}

// PrefixLinks rewrites absolute href/src links so they resolve under base.
// Exported for book tests and progressive callers; internal prefixLinks
// delegates to it.
func PrefixLinks(html, base string) string {
	return prefixLinks(html, base)
}

var codeBlockRe = regexp.MustCompile(`(?is)(<pre\b[^>]*>.*?</pre>|<code\b[^>]*>.*?</code>)`)

// prefixLinks rewrites absolute href/src links so they resolve under base,
// skipping <pre> and <code> blocks to prevent modifying code examples.
func prefixLinks(html, base string) string {
	prefix := ""
	if base != "" && base != "/" {
		prefix = strings.Trim(base, "/")
	}
	if prefix == "" {
		return html
	}

	matches := codeBlockRe.FindAllStringIndex(html, -1)
	if len(matches) == 0 {
		return rewriteLinks(html, prefix)
	}

	var sb strings.Builder
	lastIdx := 0
	for _, m := range matches {
		if m[0] > lastIdx {
			sb.WriteString(rewriteLinks(html[lastIdx:m[0]], prefix))
		}
		sb.WriteString(html[m[0]:m[1]])
		lastIdx = m[1]
	}
	if lastIdx < len(html) {
		sb.WriteString(rewriteLinks(html[lastIdx:], prefix))
	}
	return sb.String()
}

func rewriteLinks(html, prefix string) string {
	return absLinkRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := absLinkRe.FindStringSubmatch(m)
		attr, rest := sub[1], sub[2]
		if strings.HasPrefix(rest, "/") || (prefix != "" && (strings.HasPrefix(rest, prefix+"/") || rest == prefix)) {
			return m
		}
		out := attr + `="/`
		if prefix != "" {
			out += prefix + "/"
		}
		out += rest
		return out + `"`
	})
}
