package sec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrPathTraversal is returned when a target path escapes its root directory boundary.
	ErrPathTraversal = errors.New("sec: path traversal detected: path escapes root boundary")

	// ErrInvalidPath is returned when root or target path is empty or contains illegal characters.
	ErrInvalidPath = errors.New("sec: root and path must not be empty")
)

// SafePathOption allows configuring SafePath behavior.
type SafePathOption func(*safePathConfig)

type safePathConfig struct {
	rejectAbsolute bool
}

// WithRejectAbsolute configures SafePath to strictly reject absolute target paths
// that do not reside within baseDir (rather than scoping them relative to baseDir).
func WithRejectAbsolute() SafePathOption {
	return func(cfg *safePathConfig) {
		cfg.rejectAbsolute = true
	}
}

// SafePath bounds and validates untrustedPath within baseDir, preventing
// directory traversal attacks (CWE-22) and symlink escapes (CWE-59).
//
// It canonicalizes baseDir and untrustedPath, evaluates existing path components
// against symbolic links to ensure they resolve within baseDir, and returns the
// verified canonical path. If untrustedPath attempts to escape baseDir via lexical
// traversal, null bytes, or symbolic link indirection, SafePath returns ErrPathTraversal.
// An empty baseDir or untrustedPath fails closed and returns ErrInvalidPath.
// Sensitive host paths are excluded from error messages to avoid leaking host structure.
func SafePath(baseDir, untrustedPath string, opts ...SafePathOption) (string, error) {
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(untrustedPath) == "" {
		return "", ErrInvalidPath
	}
	if strings.ContainsRune(untrustedPath, 0) || strings.ContainsRune(baseDir, 0) {
		return "", ErrInvalidPath
	}

	var cfg safePathConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("sec: resolve base path: %w", err)
	}
	realBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		// If baseDir itself does not exist or cannot be resolved, fail-closed
		return "", fmt.Errorf("sec: evaluate base directory symlinks: %w", err)
	}

	normalized := strings.ReplaceAll(untrustedPath, "\\", "/")
	cleanedInput := filepath.Clean(filepath.FromSlash(normalized))

	if cleanedInput == ".." || strings.HasPrefix(cleanedInput, ".."+string(filepath.Separator)) || strings.HasPrefix(cleanedInput, "../") {
		return "", fmt.Errorf("%w: path attempts traversal outside sandbox", ErrPathTraversal)
	}

	isAbs := filepath.IsAbs(cleanedInput) || strings.HasPrefix(normalized, "/")
	if cfg.rejectAbsolute && isAbs {
		// If untrustedPath is absolute, it is only allowed if it physically starts with absBase or realBase
		if !strings.HasPrefix(cleanedInput, absBase) && !strings.HasPrefix(cleanedInput, realBase) {
			return "", fmt.Errorf("%w: absolute path outside sandbox", ErrPathTraversal)
		}
	}

	var target string
	if strings.HasPrefix(cleanedInput, absBase) {
		relFromBase, err := filepath.Rel(absBase, cleanedInput)
		if err == nil && !strings.HasPrefix(relFromBase, "..") {
			target = filepath.Clean(filepath.Join(realBase, relFromBase))
		} else {
			target = cleanedInput
		}
	} else if strings.HasPrefix(cleanedInput, realBase) {
		target = cleanedInput
	} else {
		// Treat leading slash or volume root as relative to realBase
		relInput := strings.TrimPrefix(cleanedInput, string(filepath.Separator))
		if vol := filepath.VolumeName(relInput); vol != "" {
			relInput = strings.TrimPrefix(relInput, vol)
			relInput = strings.TrimPrefix(relInput, string(filepath.Separator))
		}
		target = filepath.Clean(filepath.Join(realBase, relInput))
	}

	rel, err := filepath.Rel(realBase, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path attempts traversal outside sandbox", ErrPathTraversal)
	}

	// Symlink check on all existing path components from realBase down to target
	parts := strings.Split(rel, string(filepath.Separator))
	curr := realBase
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		curr = filepath.Join(curr, part)
		info, lerr := os.Lstat(curr)
		if lerr != nil {
			if os.IsNotExist(lerr) {
				// Component does not exist yet; no more existing symlinks down this path
				break
			}
			return "", fmt.Errorf("sec: inspect path component: %w", lerr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			realCurr, err := filepath.EvalSymlinks(curr)
			if err != nil {
				return "", fmt.Errorf("%w: evaluate symlink: %v", ErrPathTraversal, err)
			}
			cRel, err := filepath.Rel(realBase, realCurr)
			if err != nil || cRel == ".." || strings.HasPrefix(cRel, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("%w: symlink target escapes sandbox boundary", ErrPathTraversal)
			}
			// Rebuild target path using canonical intermediate directory
			remaining := filepath.Join(parts[i+1:]...)
			target = filepath.Join(realCurr, remaining)
			curr = realCurr
		}
	}

	// If target exists, evaluate symlinks on target itself
	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			realTarget, err := filepath.EvalSymlinks(target)
			if err != nil {
				return "", fmt.Errorf("%w: evaluate symlink target: %v", ErrPathTraversal, err)
			}
			tRel, err := filepath.Rel(realBase, realTarget)
			if err != nil || tRel == ".." || strings.HasPrefix(tRel, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("%w: symlink target escapes sandbox boundary", ErrPathTraversal)
			}
			return realTarget, nil
		}
	}

	finalRel, err := filepath.Rel(realBase, target)
	if err != nil || finalRel == ".." || strings.HasPrefix(finalRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path attempts traversal outside sandbox", ErrPathTraversal)
	}

	return target, nil
}

// SafeWorkspace encapsulates a validated, confined directory handle
// that executes file operations using descriptor-relative semantics
// (via os.Root) to prevent symlink-swap TOCTOU race conditions.
type SafeWorkspace struct {
	baseDir string
	root    *os.Root
}

// OpenSafeWorkspace initializes a SafeWorkspace bounded to rootPath.
// If rootPath is empty, relative without a trusted root, or does not exist,
// it fails closed and returns an error.
func OpenSafeWorkspace(rootPath string) (*SafeWorkspace, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, ErrInvalidPath
	}
	if strings.ContainsRune(rootPath, 0) {
		return nil, ErrInvalidPath
	}
	absBase, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("sec: resolve workspace root: %w", err)
	}
	realBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		return nil, fmt.Errorf("%w: workspace directory must exist", ErrInvalidPath)
	}
	r, err := os.OpenRoot(realBase)
	if err != nil {
		return nil, fmt.Errorf("sec: open workspace root: %w", err)
	}
	return &SafeWorkspace{
		baseDir: realBase,
		root:    r,
	}, nil
}

// Close closes the underlying root descriptor handle.
func (w *SafeWorkspace) Close() error {
	if w == nil || w.root == nil {
		return nil
	}
	return w.root.Close()
}

// BaseDir returns the resolved physical root path.
func (w *SafeWorkspace) BaseDir() string {
	if w == nil {
		return ""
	}
	return w.baseDir
}

// CleanRel validates and returns a clean, relative path safe for descriptor operations.
func (w *SafeWorkspace) CleanRel(untrustedPath string) (string, error) {
	if w == nil || w.root == nil || w.baseDir == "" {
		return "", ErrInvalidPath
	}
	if strings.TrimSpace(untrustedPath) == "" || strings.ContainsRune(untrustedPath, 0) {
		return "", ErrInvalidPath
	}
	normalized := strings.ReplaceAll(untrustedPath, "\\", "/")
	clean := filepath.Clean(filepath.FromSlash(normalized))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: path attempts traversal outside workspace", ErrPathTraversal)
	}

	// If absolute path was given:
	if strings.HasPrefix(clean, w.baseDir) {
		relFromBase, err := filepath.Rel(w.baseDir, clean)
		if err == nil && !strings.HasPrefix(relFromBase, "..") {
			return relFromBase, nil
		}
	} else if filepath.IsAbs(clean) || strings.HasPrefix(normalized, "/") {
		// Absolute path outside workspace
		return "", fmt.Errorf("%w: absolute path outside workspace", ErrPathTraversal)
	}

	// Strip leading separator / volume
	clean = strings.TrimPrefix(clean, string(filepath.Separator))
	if vol := filepath.VolumeName(clean); vol != "" {
		clean = strings.TrimPrefix(clean, vol)
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	if clean == "" {
		clean = "."
	}
	return clean, nil
}

// ReadFile reads the named file within the workspace.
func (w *SafeWorkspace) ReadFile(relPath string) ([]byte, error) {
	clean, err := w.CleanRel(relPath)
	if err != nil {
		return nil, err
	}
	data, err := w.root.ReadFile(clean)
	if err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "escapes") {
			return nil, fmt.Errorf("%w: %v", ErrPathTraversal, err)
		}
		return nil, err
	}
	return data, nil
}

// WriteFile writes data to a file within the workspace, creating parent directories if needed.
func (w *SafeWorkspace) WriteFile(relPath string, data []byte, perm os.FileMode) error {
	clean, err := w.CleanRel(relPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(clean)
	if dir != "." && dir != "" {
		if err := w.root.MkdirAll(dir, 0o755); err != nil {
			if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "escapes") {
				return fmt.Errorf("%w: %v", ErrPathTraversal, err)
			}
			return err
		}
	}
	if err := w.root.WriteFile(clean, data, perm); err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "escapes") {
			return fmt.Errorf("%w: %v", ErrPathTraversal, err)
		}
		return err
	}
	return nil
}

// EditFile performs an exact substring replacement in a workspace file.
func (w *SafeWorkspace) EditFile(relPath string, target, replacement string) error {
	data, err := w.ReadFile(relPath)
	if err != nil {
		return err
	}
	content := string(data)
	if !strings.Contains(content, target) {
		return errors.New("target text not found in file")
	}
	updated := strings.Replace(content, target, replacement, 1)
	return w.WriteFile(relPath, []byte(updated), 0o644)
}

// ReadDir reads directory entries within the workspace.
func (w *SafeWorkspace) ReadDir(relPath string) ([]os.DirEntry, error) {
	if relPath == "" || relPath == "." {
		f, err := w.root.Open(".")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return f.ReadDir(-1)
	}
	clean, err := w.CleanRel(relPath)
	if err != nil {
		return nil, err
	}
	f, err := w.root.Open(clean)
	if err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "escapes") {
			return nil, fmt.Errorf("%w: %v", ErrPathTraversal, err)
		}
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}

// Remove removes the named file or empty directory in the workspace.
func (w *SafeWorkspace) Remove(relPath string) error {
	clean, err := w.CleanRel(relPath)
	if err != nil {
		return err
	}
	if err := w.root.Remove(clean); err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "escapes") {
			return fmt.Errorf("%w: %v", ErrPathTraversal, err)
		}
		return err
	}
	return nil
}

// RemoveAll removes relPath and any children it contains.
func (w *SafeWorkspace) RemoveAll(relPath string) error {
	clean, err := w.CleanRel(relPath)
	if err != nil {
		return err
	}
	if clean == "." {
		return fmt.Errorf("%w: cannot remove root workspace", ErrInvalidPath)
	}
	if err := w.root.RemoveAll(clean); err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "escapes") {
			return fmt.Errorf("%w: %v", ErrPathTraversal, err)
		}
		return err
	}
	return nil
}

// SafePath returns the canonical host path for relPath, validating all bounds.
func (w *SafeWorkspace) SafePath(relPath string) (string, error) {
	if w == nil || w.baseDir == "" {
		return "", ErrInvalidPath
	}
	return SafePath(w.baseDir, relPath)
}
