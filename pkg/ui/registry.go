package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
)

// Component renders markup from arbitrary props. Implementations decode a
// declarative data value (typically map[string]any from YAML) into their typed
// props and return the rendered HTML. A Component is a pure, idempotent
// function of its props: the same props always produce the same output.
type Component interface {
	Render(props any) (template.HTML, error)
}

// Registry maps component names to renderable Components. Pages reference
// components by name; the registry resolves them. Register with the same name
// twice to override a built-in or existing component.
type Registry struct {
	components map[string]Component
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{components: map[string]Component{}}
}

// Default returns the registry pre-loaded with all standard UI components.
// The component set is defined in builtinComponents (components.go) so adding
// a new built-in component only requires editing that slice — not this function.
func Default() *Registry {
	r := NewRegistry()
	for _, e := range builtinComponents {
		r.components[e.Name] = componentFunc(e.render)
	}
	return r
}

// componentFunc adapts a plain render function into the Component interface.
type componentFunc func(any) (template.HTML, error)

// Render implements Component.
func (f componentFunc) Render(props any) (template.HTML, error) { return f(props) }

// Register binds a name to a Component. Re-registering an existing name
// replaces the previous component.
func (r *Registry) Register(name string, c Component) *Registry {
	r.components[name] = c
	return r
}

// Get returns the Component registered under name, if any.
func (r *Registry) Get(name string) (Component, bool) {
	c, ok := r.components[name]
	return c, ok
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
