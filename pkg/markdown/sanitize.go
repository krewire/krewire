package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// atomBody is the fragment context Sanitize parses against.
var atomBody = atom.Body

// urlAttrRe matches href/src/srcset/cite attributes with a single- or
// double-quoted value. Only quoted attributes are rewritten: an unquoted value
// cannot start with an executable scheme without already being treated as an
// attribute name by any conformant HTML parser.
var urlAttrRe = regexp.MustCompile(`(?i)\b(href|src|srcset|cite)\s*=\s*(?:"([^"]*)"|'([^']*)')`)

// StripScriptURLs returns in with attributes whose value is an executable URL
// scheme (javascript:, vbscript:, data:) rewritten to an empty value. Every
// other byte is preserved verbatim, so trusted raw HTML keeps rendering exactly
// as authored while script-bearing links can no longer execute.
//
// It is applied by Render. Use RenderSafe when the surrounding markup itself is
// untrusted.
func StripScriptURLs(in string) (string, error) {
	if !strings.ContainsRune(in, '=') {
		return in, nil
	}
	out := urlAttrRe.ReplaceAllStringFunc(in, func(m string) string {
		sub := urlAttrRe.FindStringSubmatch(m)
		val := sub[2]
		quote := `"`
		if val == "" && sub[3] != "" {
			val = sub[3]
			quote = "'"
		}
		if isSafeURL(val) {
			return m
		}
		return sub[1] + "=" + quote + quote
	})
	return out, nil
}

// safeTags is the allowlist of elements kept by Sanitize. Anything not listed is
// dropped. Void and structural elements needed by GitHub-flavored Markdown are
// included; script-bearing and embedding elements are deliberately absent.
var safeTags = map[string]bool{
	"a": true, "abbr": true, "b": true, "blockquote": true, "br": true,
	"caption": true, "cite": true, "code": true, "col": true, "colgroup": true,
	"dd": true, "del": true, "details": true, "div": true, "dl": true, "dt": true,
	"em": true, "figcaption": true, "figure": true, "h1": true, "h2": true,
	"h3": true, "h4": true, "h5": true, "h6": true, "hr": true, "i": true,
	"img": true, "ins": true, "kbd": true, "li": true, "mark": true, "ol": true,
	"p": true, "picture": true, "pre": true, "q": true, "rp": true, "rt": true,
	"ruby": true, "s": true, "samp": true, "small": true, "source": true,
	"span": true, "strong": true, "sub": true, "summary": true, "sup": true,
	"table": true, "tbody": true, "td": true, "tfoot": true, "th": true,
	"thead": true, "tr": true, "u": true, "ul": true, "var": true, "wbr": true,
}

// voidTags never have children; they are written self-closing.
var voidTags = map[string]bool{
	"br": true, "col": true, "hr": true, "img": true, "source": true, "wbr": true,
}

// safeAttrs is the allowlist of attributes kept by Sanitize, keyed by element.
// The empty-string key holds attributes allowed on any allowed element.
var safeAttrs = map[string]map[string]bool{
	"":           {"class": true, "id": true, "title": true, "dir": true, "lang": true},
	"a":          {"href": true, "rel": true, "target": true, "name": true},
	"img":        {"src": true, "alt": true, "width": true, "height": true, "loading": true, "srcset": true},
	"source":     {"src": true, "srcset": true, "type": true, "media": true},
	"td":         {"colspan": true, "rowspan": true, "align": true},
	"th":         {"colspan": true, "rowspan": true, "scope": true, "align": true},
	"col":        {"span": true},
	"colgroup":   {"span": true},
	"ol":         {"start": true, "type": true},
	"details":    {"open": true},
	"blockquote": {"cite": true},
	"q":          {"cite": true},
	"del":        {"cite": true, "datetime": true},
	"ins":        {"cite": true, "datetime": true},
}

// urlAttrs are the attributes whose value is a URL and therefore needs scheme
// validation to block javascript:, data:, and vbscript: payloads.
var urlAttrs = map[string]bool{"href": true, "src": true, "srcset": true, "cite": true}

// safeSchemes are the URL schemes permitted in rendered output. Everything
// else — javascript, vbscript, data, file — is rejected. Scheme-relative and
// root-relative URLs carry no scheme and remain allowed.
var safeSchemes = map[string]bool{
	"http": true, "https": true, "mailto": true, "tel": true, "ftp": true,
}

// Sanitize removes unsafe elements and attributes from an HTML fragment,
// keeping only allowlisted structural markup and text. It is safe to call on
// untrusted output and is applied by RenderSafe.
func Sanitize(in string) (string, error) {
	context := &sanitizeContext{}
	nodes, err := html.ParseFragment(strings.NewReader(in), &html.Node{
		Type: html.ElementNode, Data: "body", DataAtom: atomBody,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSanitize, err)
	}
	var sb strings.Builder
	for _, n := range nodes {
		context.render(&sb, n)
	}
	return sb.String(), nil
}

type sanitizeContext struct{}

func (c *sanitizeContext) render(sb *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		sb.WriteString(html.EscapeString(n.Data))
	case html.CommentNode, html.DoctypeNode:
		// Dropped.
	case html.ElementNode:
		c.renderElement(sb, n)
	default:
		c.renderChildren(sb, n)
	}
}

// dropContentTags are elements whose text content is code or fallback markup
// rather than prose. When such an element is disallowed, its subtree is removed
// entirely instead of being flattened to text, so a stripped <script> does not
// leak its body into the rendered output.
var dropContentTags = map[string]bool{
	"script": true, "style": true, "template": true, "noscript": true,
	"iframe": true, "object": true, "embed": true, "applet": true,
	"frame": true, "frameset": true, "noframes": true, "xmp": true,
	"svg": true, "math": true, "form": true, "select": true, "option": true,
	"textarea": true, "title": true, "head": true, "audio": true, "video": true,
}

func (c *sanitizeContext) renderElement(sb *strings.Builder, n *html.Node) {
	tag := strings.ToLower(n.Data)
	if !safeTags[tag] {
		if !dropContentTags[tag] {
			// Keep the text of a disallowed element so prose survives.
			c.renderTextOnly(sb, n)
		}
		return
	}
	sb.WriteString("<" + tag)
	for _, attr := range n.Attr {
		key := strings.ToLower(attr.Key)
		if !allowedAttr(tag, key) {
			continue
		}
		if urlAttrs[key] && !isSafeURL(attr.Val) {
			continue
		}
		sb.WriteString(" " + attr.Key + `="` + html.EscapeString(attr.Val) + `"`)
	}
	if tag == "a" {
		sb.WriteString(` rel="nofollow noopener noreferrer"`)
	}
	if voidTags[tag] {
		sb.WriteString(" />")
		return
	}
	sb.WriteString(">")
	c.renderChildren(sb, n)
	sb.WriteString("</" + tag + ">")
}

func (c *sanitizeContext) renderChildren(sb *strings.Builder, n *html.Node) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.render(sb, child)
	}
}

func (c *sanitizeContext) renderTextOnly(sb *strings.Builder, n *html.Node) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			sb.WriteString(html.EscapeString(child.Data))
		}
	}
}

func allowedAttr(tag, key string) bool {
	if strings.HasPrefix(key, "on") {
		return false
	}
	switch key {
	case "style", "srcdoc", "formaction", "xlink:href", "http-equiv":
		return false
	}
	if perTag := safeAttrs[tag]; perTag != nil && perTag[key] {
		return true
	}
	return safeAttrs[""][key]
}

// isSafeURL reports whether a URL value uses an allowed scheme. Values without
// a scheme (relative, absolute-path, protocol-relative, fragment, query) are
// allowed; control characters are stripped first so "java\tscript:" cannot slip
// past the scheme check.
func isSafeURL(raw string) bool {
	cleaned := strings.Map(func(r rune) rune {
		if r <= 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, raw)
	if cleaned == "" {
		return true
	}
	idx := strings.Index(cleaned, ":")
	if idx <= 0 {
		return true
	}
	prefix := cleaned[:idx]
	if strings.ContainsAny(prefix, "/?#") {
		// A separator before ':' means it is a relative path, not a scheme.
		return true
	}
	return safeSchemes[strings.ToLower(prefix)]
}
