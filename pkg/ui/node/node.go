package node

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"
)

// Node represents any renderable unit in the Krewire UI tree.
// Every native Go component, layout, primitive, and template adapter
// implements this canonical contract.
type Node interface {
	// Render writes the HTML representation of this node to w.
	Render(ctx context.Context, w io.Writer) error
	// RenderHTML renders the node and returns safe template.HTML.
	RenderHTML(ctx context.Context) (template.HTML, error)
}

// NodeFunc is an adapter function allowing a plain Go function
// to satisfy the Node interface.
type NodeFunc func(ctx context.Context, w io.Writer) error

// Render implements Node.
func (f NodeFunc) Render(ctx context.Context, w io.Writer) error {
	if f == nil {
		return nil
	}
	return f(ctx, w)
}

// RenderHTML implements Node.
func (f NodeFunc) RenderHTML(ctx context.Context) (template.HTML, error) {
	var buf bytes.Buffer
	if err := f.Render(ctx, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// RenderToString renders a Node tree to a string.
func RenderToString(ctx context.Context, n Node) (string, error) {
	if n == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := n.Render(ctx, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderToHTML renders a Node tree to safe template.HTML.
func RenderToHTML(ctx context.Context, n Node) (template.HTML, error) {
	if n == nil {
		return "", nil
	}
	return n.RenderHTML(ctx)
}

// textNode represents an HTML-escaped text node.
type textNode struct {
	text string
}

// Text creates a new text node with contextual HTML escaping.
func Text(s string) Node {
	return &textNode{text: s}
}

func (t *textNode) Render(ctx context.Context, w io.Writer) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	escaped := template.HTMLEscapeString(t.text)
	_, err := io.WriteString(w, escaped)
	return err
}

func (t *textNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	return template.HTML(template.HTMLEscapeString(t.text)), nil
}

// rawHTMLNode represents raw, pre-escaped or trusted HTML content.
type rawHTMLNode struct {
	html string
}

// RawHTML creates a node that outputs raw HTML content without additional escaping.
// Use for trusted SVG icons or pre-sanitized markup.
func RawHTML(h string) Node {
	return &rawHTMLNode{html: h}
}

func (r *rawHTMLNode) Render(ctx context.Context, w io.Writer) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	_, err := io.WriteString(w, r.html)
	return err
}

func (r *rawHTMLNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	return template.HTML(r.html), nil
}

// fragmentNode groups multiple nodes without adding a surrounding HTML tag.
type fragmentNode struct {
	children []Node
}

// Fragment groups multiple nodes into a single composite node.
func Fragment(children ...Node) Node {
	filtered := make([]Node, 0, len(children))
	for _, c := range children {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	return &fragmentNode{children: filtered}
}

func (f *fragmentNode) Render(ctx context.Context, w io.Writer) error {
	for _, child := range f.children {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if child != nil {
			if err := child.Render(ctx, w); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *fragmentNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	var buf bytes.Buffer
	if err := f.Render(ctx, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// Attrs maps HTML attribute names to their values.
// Supports strings, numbers, booleans, and custom values.
type Attrs map[string]any

// voidElements is the set of HTML void tags that cannot have child nodes or end tags.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// elementNode represents a standard HTML element with attributes and children.
type elementNode struct {
	tag      string
	attrs    Attrs
	children []Node
}

// Element creates an HTML element node.
func Element(tag string, attrs Attrs, children ...Node) Node {
	cleanedTag := strings.ToLower(strings.TrimSpace(tag))
	if cleanedTag == "" {
		cleanedTag = "div"
	}
	filtered := make([]Node, 0, len(children))
	for _, c := range children {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	return &elementNode{
		tag:      cleanedTag,
		attrs:    attrs,
		children: filtered,
	}
}

func (e *elementNode) Render(ctx context.Context, w io.Writer) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	if _, err := io.WriteString(w, "<"+e.tag); err != nil {
		return err
	}

	if len(e.attrs) > 0 {
		if err := writeAttrs(w, e.attrs); err != nil {
			return err
		}
	}

	if voidElements[e.tag] {
		_, err := io.WriteString(w, " />")
		return err
	}

	if _, err := io.WriteString(w, ">"); err != nil {
		return err
	}

	for _, child := range e.children {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if child != nil {
			if err := child.Render(ctx, w); err != nil {
				return err
			}
		}
	}

	_, err := io.WriteString(w, "</"+e.tag+">")
	return err
}

func (e *elementNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	var buf bytes.Buffer
	if err := e.Render(ctx, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// writeAttrs writes sorted attributes to w with strict escaping and URL validation.
func writeAttrs(w io.Writer, attrs Attrs) error {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := attrs[k]
		if val == nil {
			continue
		}

		cleanKey := sanitizeAttrKey(k)
		if cleanKey == "" {
			continue
		}

		switch v := val.(type) {
		case bool:
			if v {
				if _, err := io.WriteString(w, " "+cleanKey); err != nil {
					return err
				}
			}
		case string:
			if isURLAttr(cleanKey) {
				v = SafeURL(v)
			} else {
				v = template.HTMLEscapeString(v)
			}
			if _, err := fmt.Fprintf(w, ` %s="%s"`, cleanKey, v); err != nil {
				return err
			}
		default:
			strVal := fmt.Sprintf("%v", v)
			if isURLAttr(cleanKey) {
				strVal = SafeURL(strVal)
			} else {
				strVal = template.HTMLEscapeString(strVal)
			}
			if _, err := fmt.Fprintf(w, ` %s="%s"`, cleanKey, strVal); err != nil {
				return err
			}
		}
	}
	return nil
}

// sanitizeAttrKey validates and normalizes attribute names.
func sanitizeAttrKey(k string) string {
	trimmed := strings.TrimSpace(k)
	if trimmed == "" {
		return ""
	}
	var sb strings.Builder
	for i, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == ':' || r == '@' || r == '.' {
			sb.WriteRune(r)
		} else if i == 0 {
			return ""
		}
	}
	return sb.String()
}

// isURLAttr reports whether an attribute typically contains a URI or resource locator.
func isURLAttr(name string) bool {
	switch strings.ToLower(name) {
	case "href", "src", "action", "formaction", "poster", "data":
		return true
	default:
		return false
	}
}

// safeURLSchemes are the allowed URI protocols.
var safeURLSchemes = map[string]bool{
	"http": true, "https": true, "mailto": true, "tel": true, "ftp": true,
}

// SafeURL sanitizes a URL string against JavaScript injection attacks (XSS).
// Relative URLs and safe schemes are preserved and HTML-escaped.
// Executable schemes (javascript:, vbscript:, data:) are replaced with "#".
func SafeURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	cleaned := strings.Map(func(r rune) rune {
		if r <= 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, trimmed)

	idx := strings.Index(cleaned, ":")
	if idx <= 0 {
		return template.HTMLEscapeString(trimmed)
	}
	prefix := cleaned[:idx]
	if strings.ContainsAny(prefix, "/?#") {
		return template.HTMLEscapeString(trimmed)
	}
	if !safeURLSchemes[strings.ToLower(prefix)] {
		return "#"
	}
	return template.HTMLEscapeString(trimmed)
}
