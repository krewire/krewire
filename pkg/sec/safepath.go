package sec

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	// ErrPathTraversal is returned when a target path escapes its root directory boundary.
	ErrPathTraversal = errors.New("sec: path traversal detected: path escapes root boundary")

	// ErrInvalidPath is returned when root or target path is empty.
	ErrInvalidPath = errors.New("sec: root and path must not be empty")
)

// SafePath bounds and validates untrustedPath within baseDir, preventing
// directory traversal attacks (CWE-22).
//
// It cleans the input, resolves relative traversal segments ("." and ".."),
// and verifies that the canonical resolved path is strictly contained within
// baseDir. If untrustedPath attempts to escape baseDir, SafePath returns ErrPathTraversal.
func SafePath(baseDir, untrustedPath string) (string, error) {
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(untrustedPath) == "" {
		return "", ErrInvalidPath
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("sec: resolve base path: %w", err)
	}

	normalized := strings.ReplaceAll(untrustedPath, "\\", "/")
	cleanedInput := filepath.Clean(filepath.FromSlash(normalized))

	var target string
	if strings.HasPrefix(cleanedInput, absBase) {
		target = cleanedInput
	} else {
		// Treat leading slash as relative to baseDir root
		relInput := strings.TrimPrefix(cleanedInput, string(filepath.Separator))
		target = filepath.Clean(filepath.Join(absBase, relInput))
	}

	rel, err := filepath.Rel(absBase, target)
	if err != nil {
		return "", fmt.Errorf("sec: evaluate relative path: %w", err)
	}

	// Reject paths that step out of baseDir
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q escapes base %q", ErrPathTraversal, untrustedPath, baseDir)
	}

	return target, nil
}
