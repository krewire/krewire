package node

import (
	"context"
	"fmt"
	"html/template"
	"io"
)

// ComponentFunc defines the signature of a functional Go UI component.
type ComponentFunc[P any] func(props P, children ...Node) Node

// HTMLRenderer represents any legacy or custom component that produces template.HTML.
type HTMLRenderer interface {
	Render() template.HTML
}

// widgetAdapterNode wraps an HTMLRenderer (e.g. pkg/ui/widget.Widget) to implement Node.
type widgetAdapterNode struct {
	renderer HTMLRenderer
}

// FromWidget adapts an HTMLRenderer (such as any pkg/ui/widget.Widget) into a canonical Node.
func FromWidget(w HTMLRenderer) Node {
	if w == nil {
		return nil
	}
	return &widgetAdapterNode{renderer: w}
}

func (wa *widgetAdapterNode) Render(ctx context.Context, w io.Writer) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	html := wa.renderer.Render()
	_, err := io.WriteString(w, string(html))
	return err
}

func (wa *widgetAdapterNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	if ctx != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}
	return wa.renderer.Render(), nil
}

// Textf creates a formatted, contextually escaped text node.
func Textf(format string, a ...any) Node {
	return Text(fmt.Sprintf(format, a...))
}

// Strings converts a slice of string literals into a slice of safe Text nodes.
func Strings(strs ...string) []Node {
	nodes := make([]Node, len(strs))
	for i, s := range strs {
		nodes[i] = Text(s)
	}
	return nodes
}
