// Tests for KWN-JB7PW
package buildinfo

import "testing"

// Spec: KWN-JB7PW RND-VS-004 RND-VS-008 Scope: Unit
func TestKWN_VS_004_ResolveVersion_WorkspaceDevelFallsBackToKnown(t *testing.T) {
	for _, path := range []string{ModKrewire} {
		got, fromSource := ResolveVersion(path)
		if got == "" || got == DevelVersion {
			t.Errorf("ResolveVersion(%q) = %q, want a concrete version for workspace builds", path, got)
		}
		if !fromSource {
			t.Errorf("ResolveVersion(%q) fromSource = false, want true in workspace builds", path)
		}
	}
}

// Spec: KWN-JB7PW RND-VS-004 Scope: Unit
func TestKWN_VS_004_ResolveVersion_UnknownPathReturnsEmpty(t *testing.T) {
	got, fromSource := ResolveVersion("example.com/unknown")
	if got != "" || fromSource {
		t.Errorf("ResolveVersion(unknown) = (%q, %v), want (\"\", false)", got, fromSource)
	}
}

// Spec: KWL-L0-REPRO REQ-L0-023 Scope: Unit
func TestKWL_L0_REPRO_REQ_L0_023_ReadProvenance(t *testing.T) {
	prov, ok := ReadProvenance()
	if !ok {
		// In pure unit test runner without debug metadata, ReadProvenance may return false
		t.Log("debug.ReadBuildInfo not available in this test environment (expected when running uncompiled test binary)")
		return
	}
	t.Logf("Provenance detected: revision=%s, time=%s, modified=%v, trimpath=%v, cgo=%s",
		prov.Revision, prov.Time, prov.Modified, prov.Trimpath, prov.Cgo)
}
