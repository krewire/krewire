// Package components delegates to the canonical Krewire UI (kui) component
// catalog in github.com/krewire/ui/components, so the rest of the toolchain
// depends on kui through one package.
package components

import (
	"runtime/debug"

	uicomponents "github.com/krewire/ui/components"
)

// FS delegates to the canonical Krewire UI (kui) embedded component filesystem.
var FS = uicomponents.FS

// ComponentMeta is a kui catalog entry (components.json).
type ComponentMeta = uicomponents.ComponentMeta

// Pack is a curated kui starter pack.
type Pack = uicomponents.Pack

// Registry maps each kui component name to its catalog entry.
var Registry = uicomponents.Registry

// Get returns the .kiw source of a kui component.
func Get(name string) ([]byte, error) { return uicomponents.Get(name) }

// List returns the sorted kui component names.
func List() []string { return uicomponents.List() }

// Lookup resolves a component name case-insensitively.
func Lookup(name string) (ComponentMeta, bool) { return uicomponents.Lookup(name) }

// Packs returns the starter packs sorted by name.
func Packs() []Pack { return uicomponents.Packs() }

// Resolve expands names and "pack:<name>" references with dependencies.
func Resolve(refs ...string) ([]string, error) { return uicomponents.Resolve(refs...) }

// ModulePath is the Go module that provides the catalog.
const ModulePath = "github.com/krewire/ui"

// Version returns the kui module version compiled into the running binary,
// or "devel" when it cannot be determined (tests, replaced modules).
func Version() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == ModulePath {
				if d.Replace != nil && d.Replace.Version != "" {
					return d.Replace.Version
				}
				if d.Version != "" {
					return d.Version
				}
			}
		}
	}
	return "devel"
}
