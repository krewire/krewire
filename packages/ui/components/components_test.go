package components_test

import (
	"strings"
	"testing"

	"github.com/krewire/krewire/packages/ui/components"
)

// Tests for KWF-M4R8T. Scope: Forge — Components — Atomic.
// FRK-FA-001, FRK-FA-004: every built-in component is readable from the
// embedded FS and non-empty.
//
// A blank or unreadable .kiw registers as an empty component and renders
// nothing, with no error from the loader. Parsing is asserted in
// framework/web/ssg (forge_atoms_test.go), which runs the real DSL pipeline —
// forge cannot import libs/kiw here without a layering violation
// (KWL-LAYER-001).
//
// This deliberately does not assert a class-name or style-block convention:
// the library is not uniform there yet (older components ship unprefixed
// classes like `.btn`, `.card`, `.alert`; newer ones use the `kiw-` prefix),
// and enforcing it would fail components that render correctly today. See
// FRK-FA-032 for the rule that governs new components.
func TestEveryEmbeddedComponentIsReadable(t *testing.T) {
	entries, err := components.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kiw") {
			continue
		}
		count++
		body, err := components.FS.ReadFile(e.Name())
		if err != nil {
			t.Errorf("%s: read: %v", e.Name(), err)
			continue
		}
		if strings.TrimSpace(string(body)) == "" {
			t.Errorf("%s: component is empty and would render nothing", e.Name())
		}
	}
	if count == 0 {
		t.Fatal("no components embedded")
	}
}

// FRK-FA-030: the status vocabulary is info / success / warning / error,
// matching the --error theme token and the names framework/ui already uses.
// `danger` and `warn` were removed rather than aliased, so nothing should
// reintroduce them. This is a source-level guard because that is where the
// drift happens — a variant name only becomes user-visible after a build.
func TestNoRetiredStatusVocabulary(t *testing.T) {
	retired := []string{"danger", "warn"}
	entries, err := components.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kiw") {
			continue
		}
		body, err := components.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("%s: read: %v", e.Name(), err)
		}
		src := string(body)
		for _, r := range retired {
			// `warning` legitimately contains "warn", so match whole words only.
			if matchesWord(src, r) {
				t.Errorf("%s still uses the retired status name %q; use %q (FRK-FA-030)",
					e.Name(), r, map[string]string{"danger": "error", "warn": "warning"}[r])
			}
		}
	}
}

// matchesWord reports whether word occurs in src delimited by non-word
// characters, so "warn" matches `.callout.warn` but not "warning".
func matchesWord(src, word string) bool {
	idx := strings.Index(src, word)
	for idx >= 0 {
		beforeOK := idx == 0 || !isWordByte(src[idx-1])
		after := idx + len(word)
		afterOK := after >= len(src) || !isWordByte(src[after])
		if beforeOK && afterOK {
			return true
		}
		next := strings.Index(src[idx+len(word):], word)
		if next < 0 {
			return false
		}
		idx += len(word) + next
	}
	return false
}

// isWordByte reports whether b continues an identifier. Punctuation such as
// `-`, `.` and `:` is deliberately NOT a word byte, so a retired name is still
// caught when it appears as part of a class (`alert-danger`, `.callout.warn`).
// Letters, digits and `_` continue it, which is what keeps `warning` from
// matching a search for `warn`.
func isWordByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// legacyWithoutHook lists components that predate FRK-FA-003 and still ship no
// data-kiw-component hook.
//
// They are enumerated rather than simply tolerated so the gap cannot quietly
// widen: a new component missing the hook fails this test, and the only way to
// add one here is to state that it is a legacy exception. Adding the hook to a
// component changes what its scoped CSS matches, so it is a deliberate change
// to the rendered page, not a silent cleanup.
var legacyWithoutHook = map[string]bool{
	"Alert": true, "Badge": true, "Brand": true, "Button": true, "Callout": true,
	"Card": true, "Footer": true, "Head": true, "Input": true, "Modal": true,
	"NavMenu": true, "Navbar": true, "Section": true, "Select": true,
	"SideMenu": true, "Sidebar": true, "Textarea": true, "ThemeSwitch": true,
	"Toggle": true,
}

// FRK-FA-003: a component carries a data-kiw-component hook so scoped CSS and
// tests can target it. New components must; the ones listed above are tracked
// legacy exceptions. Adding a hook is not a silent cleanup — it changes what
// that component's scoped CSS matches — so each addition is a deliberate,
// reviewed change rather than a drive-by fix.
func TestComponentsCarryScopeHookUnlessLegacy(t *testing.T) {
	entries, err := components.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kiw") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".kiw")
		body, err := components.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("%s: read: %v", name, err)
		}
		has := strings.Contains(string(body), `data-kiw-component="`+name+`"`)
		switch {
		case has:
			legacyWithoutHook[name] = false // no longer an exception
		case !legacyWithoutHook[name]:
			t.Errorf("%s: no data-kiw-component hook; add one, or list it as a known FRK-FA-003 exception", name)
		}
	}
}
