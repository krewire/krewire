package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/ui/node"
)

// Component renders markup from arbitrary props. Implementations decode a
// declarative data value (typically map[string]any from YAML) into their typed
// props and return the rendered HTML. A Component is a pure, idempotent
// function of its props: the same props always produce the same output.
type Component interface {
	Render(props any) (template.HTML, error)
}

// ComponentMeta describes static metadata for a UI component.
// It can be inspected by devtools, build pipelines, and language servers
// without executing arbitrary component code.
type ComponentMeta struct {
	Name         string                `json:"name"`
	Namespace    string                `json:"namespace,omitempty"`
	Package      string                `json:"package,omitempty"`
	Version      string                `json:"version,omitempty"`
	SourceOrigin string                `json:"source_origin,omitempty"` // "go" or "kiw"
	SourceFile   string                `json:"source_file,omitempty"`
	Description  string                `json:"description,omitempty"`
	Props        map[string]PropSchema `json:"props,omitempty"`
	Slots        []string              `json:"slots,omitempty"`
	Styles       []string              `json:"styles,omitempty"`
	Scripts      []string              `json:"scripts,omitempty"`
	Interactive  bool                  `json:"interactive,omitempty"`
}

// PropSchema describes a single component property.
type PropSchema struct {
	Type        string `json:"type"` // "string", "int", "bool", "any", "node"
	Required    bool   `json:"required,omitempty"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// Registry maps component names to renderable Components. Pages reference
// components by name; the registry resolves them. Register with the same name
// twice to override a built-in or existing component.
type Registry struct {
	components map[string]Component
	meta       map[string]ComponentMeta
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		components: map[string]Component{},
		meta:       map[string]ComponentMeta{},
	}
}

// Default returns the registry pre-loaded with all standard UI components.
// The component set is defined in builtinComponents (components.go) so adding
// a new built-in component only requires editing that slice — not this function.
func Default() *Registry {
	r := NewRegistry()
	for _, e := range builtinComponents {
		r.components[e.Name] = ComponentFunc(e.render)
		r.meta[e.Name] = ComponentMeta{
			Name:         e.Name,
			SourceOrigin: "go",
			Description:  fmt.Sprintf("Built-in Krewire UI %s component", e.Name),
		}
	}
	return r
}

// ComponentFunc adapts a plain render function into the Component interface.
type ComponentFunc func(any) (template.HTML, error)

// Render implements Component.
func (f ComponentFunc) Render(props any) (template.HTML, error) { return f(props) }

// Register binds a name to a Component. Re-registering an existing name
// replaces the previous component.
func (r *Registry) Register(name string, c Component) *Registry {
	r.components[name] = c
	return r
}

// RegisterWithMeta binds a name to a Component alongside inspectable metadata.
func (r *Registry) RegisterWithMeta(name string, c Component, meta ComponentMeta) *Registry {
	if meta.Name == "" {
		meta.Name = name
	}
	r.components[name] = c
	r.meta[name] = meta
	return r
}

// Get returns the Component registered under name, if any.
func (r *Registry) Get(name string) (Component, bool) {
	c, ok := r.components[name]
	return c, ok
}

// Has reports whether a component is registered under name.
func (r *Registry) Has(name string) bool {
	_, ok := r.components[name]
	return ok
}

// Meta returns the metadata for a registered component.
func (r *Registry) Meta(name string) (ComponentMeta, bool) {
	m, ok := r.meta[name]
	return m, ok
}

// AllMeta returns all component metadata sorted by name.
func (r *Registry) AllMeta() []ComponentMeta {
	names := make([]string, 0, len(r.meta))
	for name := range r.meta {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]ComponentMeta, 0, len(names))
	for _, name := range names {
		result = append(result, r.meta[name])
	}
	return result
}

// Names returns a sorted list of all registered component names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Render renders the component registered under name with the given props.
func (r *Registry) Render(name string, props any) (template.HTML, error) {
	c, ok := r.components[name]
	if !ok {
		return "", fmt.Errorf("ui: undefined component %q", name)
	}
	return c.Render(props)
}

// typed adapts a value-based renderer (a component reading typed props) into
// the Component interface, so components can be registered by name.
func typed[P any](render func(P) template.HTML) Component {
	return typedComponent[P]{render: render}
}

// typedComponent binds a render function to a props type P.
type typedComponent[P any] struct {
	render func(P) template.HTML
}

// Render decodes declarative props into the typed value P and renders it.
// Already-typed values pass through unchanged.
func (c typedComponent[P]) Render(props any) (template.HTML, error) {
	var p P
	if v, ok := props.(P); ok {
		p = v
	} else if props != nil {
		if err := decodeProps(props, &p); err != nil {
			return "", err
		}
	}
	return c.render(p), nil
}

// RegisterNode binds a typed node.Node component into the registry.
// This enables Go-authored components implementing node.Node to be invoked
// seamlessly from .kiw DSL templates or registry Lookups.
func RegisterNode[P any](r *Registry, name string, fn func(props P, children ...node.Node) node.Node, meta ...ComponentMeta) *Registry {
	var m ComponentMeta
	if len(meta) > 0 {
		m = meta[0]
	}
	if m.Name == "" {
		m.Name = name
	}
	if m.SourceOrigin == "" {
		m.SourceOrigin = "go"
	}

	comp := typedNodeComponent[P]{
		render: fn,
		meta:   m,
	}
	r.RegisterWithMeta(name, comp, m)
	return r
}

// typedNodeComponent bridges a typed node.Node component function to the Component interface.
type typedNodeComponent[P any] struct {
	render func(P, ...node.Node) node.Node
	meta   ComponentMeta
}

func (c typedNodeComponent[P]) Render(props any) (template.HTML, error) {
	var p P
	var children []node.Node
	slots := node.SlotMap{}

	if v, ok := props.(P); ok {
		p = v
	} else if m, ok := props.(map[string]any); ok {
		propCopy := make(map[string]any, len(m))
		for k, v := range m {
			switch strings.ToLower(k) {
			case "body", "children":
				if str, ok := v.(string); ok && str != "" {
					children = append(children, node.RawHTML(str))
				} else if htmlVal, ok := v.(template.HTML); ok && htmlVal != "" {
					children = append(children, node.RawHTML(string(htmlVal)))
				} else if n, ok := v.(node.Node); ok && n != nil {
					children = append(children, n)
				}
			default:
				if strings.HasPrefix(k, "slot:") || strings.HasPrefix(k, "Slot:") {
					slotName := strings.TrimPrefix(strings.TrimPrefix(k, "slot:"), "Slot:")
					if str, ok := v.(string); ok {
						slots[slotName] = node.RawHTML(str)
					}
				} else {
					propCopy[k] = v
				}
			}
		}
		if err := decodeProps(propCopy, &p); err != nil {
			return "", err
		}
	} else if props != nil {
		if err := decodeProps(props, &p); err != nil {
			return "", err
		}
	}

	n := c.render(p, children...)
	if len(slots) > 0 {
		n = node.WithSlots(n, slots)
	}
	return node.RenderToHTML(context.Background(), n)
}

// BridgeToNode bridges all components in the registry to pkg/ui/node.RegisterKiwComponent,
// allowing any registered component (Go or .kiw) to be instantiated within Go widget trees
// via node.Kiw(name, props, children...).
func (r *Registry) BridgeToNode() {
	for name, comp := range r.components {
		c := comp
		node.RegisterKiwComponent(name, func(ctx context.Context, props map[string]any, children []node.Node, w io.Writer) error {
			var bodyHTML string
			if len(children) > 0 {
				var buf bytes.Buffer
				for _, ch := range children {
					if err := ch.Render(ctx, &buf); err != nil {
						return err
					}
				}
				bodyHTML = buf.String()
			}
			p := make(map[string]any, len(props)+1)
			for k, v := range props {
				p[k] = v
			}
			if bodyHTML != "" {
				p["Body"] = bodyHTML
			}
			html, err := c.Render(p)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, string(html))
			return err
		})
	}
}

// decodeProps converts a declarative props value (a map[string]any decoded
// from YAML, or any JSON-serializable value) into the target props value,
// rejecting unknown keys loudly so config typos fail the build.
func decodeProps(src any, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("ui: encode props: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("ui: decode props: %w", err)
	}
	return nil
}
