package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/krewire/krewire/sec"
)

// ErrPathEscape is returned when a path traversal escape attempt is detected.
var ErrPathEscape = sec.ErrPathTraversal

// SafePath ensures the path resolves within root without directory traversal or symlink escapes.
// It delegates to sec.SafePath to provide unified, fail-closed, symlink-safe path validation.
func SafePath(root, rel string) (string, error) {
	target, err := sec.SafePath(root, rel, sec.WithRejectAbsolute())
	if err != nil {
		if errors.Is(err, sec.ErrPathTraversal) || errors.Is(err, sec.ErrInvalidPath) {
			return "", fmt.Errorf("%w: %v", ErrPathEscape, err)
		}
		return "", err
	}
	return target, nil
}

// --- ReadFileTool ---

type ReadFileTool struct {
	Workspace string
}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string {
	return "Read the contents of a text file from the workspace."
}
func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative or absolute path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	ws, err := sec.OpenSafeWorkspace(t.Workspace)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
	}
	defer ws.Close()

	b, err := ws.ReadFile(input.Path)
	if err != nil {
		if errors.Is(err, sec.ErrPathTraversal) || errors.Is(err, sec.ErrInvalidPath) {
			return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
		}
		return nil, fmt.Errorf("read failed: %w", err)
	}
	return string(b), nil
}

// ParallelSafe marks ReadFileTool as safe for concurrent read operations.
func (t *ReadFileTool) ParallelSafe() bool { return true }

// --- WriteFileTool ---

type WriteFileTool struct {
	Workspace string
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Create or overwrite a file with given content."
}
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Full file content",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	ws, err := sec.OpenSafeWorkspace(t.Workspace)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
	}
	defer ws.Close()

	if err := ws.WriteFile(input.Path, []byte(input.Content), 0o644); err != nil {
		if errors.Is(err, sec.ErrPathTraversal) || errors.Is(err, sec.ErrInvalidPath) {
			return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
		}
		return nil, err
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(input.Content), input.Path), nil
}

// --- EditFileTool ---

type EditFileTool struct {
	Workspace string
}

func (t *EditFileTool) Name() string { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "Perform exact target string replacement in an existing file."
}
func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path of the file to edit",
			},
			"target": map[string]any{
				"type":        "string",
				"description": "Exact text chunk to replace",
			},
			"replacement": map[string]any{
				"type":        "string",
				"description": "New replacement content",
			},
		},
		"required": []string{"path", "target", "replacement"},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input struct {
		Path        string `json:"path"`
		Target      string `json:"target"`
		Replacement string `json:"replacement"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	ws, err := sec.OpenSafeWorkspace(t.Workspace)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
	}
	defer ws.Close()

	if err := ws.EditFile(input.Path, input.Target, input.Replacement); err != nil {
		if errors.Is(err, sec.ErrPathTraversal) || errors.Is(err, sec.ErrInvalidPath) {
			return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
		}
		return nil, err
	}
	return "File edited successfully", nil
}

// --- ListDirTool ---

type ListDirTool struct {
	Workspace string
}

func (t *ListDirTool) Name() string { return "list_dir" }
func (t *ListDirTool) Description() string {
	return "List directory contents with file names and directories."
}
func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory path (defaults to root workspace)",
			},
		},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(args, &input)

	ws, err := sec.OpenSafeWorkspace(t.Workspace)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
	}
	defer ws.Close()

	entries, err := ws.ReadDir(input.Path)
	if err != nil {
		if errors.Is(err, sec.ErrPathTraversal) || errors.Is(err, sec.ErrInvalidPath) {
			return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
		}
		return nil, err
	}
	var res []string
	for _, e := range entries {
		suffix := ""
		if e.IsDir() {
			suffix = "/"
		}
		res = append(res, e.Name()+suffix)
	}
	return res, nil
}

// ParallelSafe marks ListDirTool as safe for concurrent directory listing operations.
func (t *ListDirTool) ParallelSafe() bool { return true }

// --- DeleteFileTool ---

type DeleteFileTool struct {
	Workspace string
}

func (t *DeleteFileTool) Name() string { return "delete_file" }
func (t *DeleteFileTool) Description() string {
	return "Delete a file or empty directory from the workspace."
}
func (t *DeleteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file or directory to delete",
			},
		},
		"required": []string{"path"},
	}
}

func (t *DeleteFileTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	ws, err := sec.OpenSafeWorkspace(t.Workspace)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
	}
	defer ws.Close()

	if err := ws.Remove(input.Path); err != nil {
		if errors.Is(err, sec.ErrPathTraversal) || errors.Is(err, sec.ErrInvalidPath) {
			return nil, fmt.Errorf("%w: %v", ErrPathEscape, err)
		}
		return nil, err
	}
	return fmt.Sprintf("Successfully deleted %s", input.Path), nil
}
