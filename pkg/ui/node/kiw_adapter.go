package node

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"sync"
)

// KiwRendererFunc represents a function capable of rendering a .kiw DSL component.
type KiwRendererFunc func(ctx context.Context, props map[string]any, children []Node, w io.Writer) error

var (
	kiwRegistryMu sync.RWMutex
	kiwRegistry   = make(map[string]KiwRendererFunc)
)

// RegisterKiwComponent registers a .kiw component renderer under a given component name.
func RegisterKiwComponent(name string, fn KiwRendererFunc) {
	kiwRegistryMu.Lock()
	defer kiwRegistryMu.Unlock()
	kiwRegistry[name] = fn
}

// kiwComponentNode invokes a registered .kiw component template within a Go Node tree.
type kiwComponentNode struct {
	name     string
	props    map[string]any
	children []Node
}

// Kiw embeds a .kiw DSL component into a Go Node tree by component name.
// This implements the Go ↔ .kiw interoperability contract defined in ADR-K7P9L.
func Kiw(name string, props map[string]any, children ...Node) Node {
	filtered := make([]Node, 0, len(children))
	for _, c := range children {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	return &kiwComponentNode{
		name:     name,
		props:    props,
		children: filtered,
	}
}

func (k *kiwComponentNode) Render(ctx context.Context, w io.Writer) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	kiwRegistryMu.RLock()
	fn, found := kiwRegistry[k.name]
	kiwRegistryMu.RUnlock()

	if found && fn != nil {
		return fn(ctx, k.props, k.children, w)
	}

	// Graceful fallback for unresolved .kiw components during development / SSG
	fallbackAttr := Attrs{
		"data-kiw-component": k.name,
		"class":              "kiw-component-unresolved",
	}
	for pk, pv := range k.props {
		fallbackAttr["data-prop-"+pk] = fmt.Sprintf("%v", pv)
	}
	fallback := Element("div", fallbackAttr, k.children...)
	return fallback.Render(ctx, w)
}

func (k *kiwComponentNode) RenderHTML(ctx context.Context) (template.HTML, error) {
	var buf bytes.Buffer
	if err := k.Render(ctx, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
