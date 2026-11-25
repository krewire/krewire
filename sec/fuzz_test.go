package sec_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/sec"
)

func FuzzSafePath(f *testing.F) {
	// Seed corpus with representative traversal vectors and valid paths
	seeds := []string{
		"valid/sub/file.txt",
		"normal.go",
		"../../etc/passwd",
		"../parent",
		"sub/../../escape",
		"file\x00nullbyte",
		"/absolute/path/attempt",
		"./././dot/slash",
		"....//....//traversal",
		"~/.ssh/id_rsa",
		"\\\\host\\share\\path",
		"sub/nested/deep/file.md",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	tempBase := f.TempDir()
	canonicalBase, err := filepath.EvalSymlinks(tempBase)
	if err != nil {
		canonicalBase = tempBase
	}

	f.Fuzz(func(t *testing.T, untrusted string) {
		res, err := sec.SafePath(tempBase, untrusted)
		if err == nil {
			// Invariant 1: If SafePath succeeds, res MUST reside within canonicalBase
			canonicalRes, evalErr := filepath.EvalSymlinks(res)
			if evalErr != nil {
				// If target does not exist yet, its directory or cleaned path must be inside base
				cleanRes := filepath.Clean(res)
				if !strings.HasPrefix(cleanRes, canonicalBase) && !strings.HasPrefix(cleanRes, tempBase) {
					t.Fatalf("SafePath returned path escaping boundary: base=%q, res=%q", tempBase, res)
				}
			} else {
				if !strings.HasPrefix(canonicalRes, canonicalBase) {
					t.Fatalf("SafePath resolved symlink escaping boundary: base=%q, resolved=%q", canonicalBase, canonicalRes)
				}
			}

			// Invariant 2: Result must not contain null bytes
			if strings.ContainsRune(res, '\x00') {
				t.Fatalf("SafePath returned path containing null byte: %q", res)
			}
		}
	})
}
