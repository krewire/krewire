package ui

import "html/template"

// componentEntry binds a component name to its typed renderer. All built-in
// components register themselves in builtinComponents so Default() iterates
// the slice rather than maintaining a manual list. Adding a new component
// only requires appending one entry here.
type componentEntry struct {
	Name   string
	render func(any) (template.HTML, error)
}

// builtinComponents is the ordered registry of standard UI components.
// Order is stable (slice, not map) so the Default() registry is deterministic.
var builtinComponents = []componentEntry{
	{"Header", renderTyped(func(h Header) template.HTML { return h.Render() })},
	{"Footer", renderTyped(func(f Footer) template.HTML { return f.Render() })},
	{"Nav", renderTyped(func(n Nav) template.HTML { return n.Render() })},
	{"Container", renderTyped(func(c Container) template.HTML { return c.Render() })},
	{"Grid", renderTyped(func(g Grid) template.HTML { return g.Render() })},
	{"Card", renderTyped(func(c Card) template.HTML { return c.Render() })},
	{"Section", renderTyped(func(s Section) template.HTML { return s.Render() })},
	{"Hero", renderTyped(func(h Hero) template.HTML { return h.Render() })},
	{"ThemeToggle", renderTyped(func(t ThemeToggle) template.HTML { return t.Render() })},
	{"Badge", renderTyped(func(b Badge) template.HTML { return b.Render() })},
	{"Button", renderTyped(func(b Button) template.HTML { return b.Render() })},
}

// renderTyped wraps a typed render function as a component render func.
// It mirrors the logic in typedComponent.Render but returns a plain func
// so componentEntry stays a simple struct.
func renderTyped[P any](render func(P) template.HTML) func(any) (template.HTML, error) {
	tc := typedComponent[P]{render: render}
	return tc.Render
}

// AllBuiltinNames returns the ordered names of all built-in components.
// Consumers can use this list to verify registration coverage in tests.
func AllBuiltinNames() []string {
	names := make([]string, len(builtinComponents))
	for i, e := range builtinComponents {
		names[i] = e.Name
	}
	return names
}
