// Package localfile is the reference infrastructure provider (issue #138): it
// manages directories and files under one root directory on the local host.
//
// It is deliberately small but real — every mutation touches the file system —
// so the full lifecycle (plan, apply, inspect, reconcile, destroy, partial
// failure) is exercised end to end without cloud credentials. Paths are always
// relative to Root and may not escape it.
package localfile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/krewire/krewire/cloud/infra/contract"
)

// Name is the provider name.
const Name = "localfile"

// Kinds served by the provider.
const (
	KindDirectory = "directory"
	KindFile      = "file"
)

// Provider manages files under Root.
type Provider struct {
	Root string
}

// New returns a provider rooted at root.
func New(root string) *Provider { return &Provider{Root: root} }

// Capabilities implements contract.Provider.
func (p *Provider) Capabilities() contract.Capabilities {
	return contract.Capabilities{
		Contract: contract.Version, Provider: Name, ProviderVersion: "1.0.0",
		Kinds: []contract.KindSchema{
			{Kind: KindDirectory, SchemaVersion: "v1", Description: "directory under the provider root",
				Required: []string{"path"}, Optional: []string{"mode"}, ForceNew: []string{"path"},
				Updatable: true, Validate: validate},
			{Kind: KindFile, SchemaVersion: "v1", Description: "regular file under the provider root",
				Required: []string{"path"}, Optional: []string{"content", "mode"}, SecretProps: []string{"content"},
				ForceNew: []string{"path"}, Updatable: true, Validate: validate},
		},
	}
}

func validate(props map[string]any) error {
	path, _ := props["path"].(string)
	if path == "" || !filepath.IsLocal(path) {
		return fmt.Errorf("path %q must be a non-empty relative path inside the provider root", path)
	}
	if m, ok := props["mode"]; ok {
		if _, err := parseMode(m); err != nil {
			return err
		}
	}
	if c, ok := props["content"]; ok {
		if _, isStr := c.(string); !isStr {
			return errors.New("content must be a string")
		}
	}
	return nil
}

func parseMode(v any) (fs.FileMode, error) {
	s, ok := v.(string)
	n, err := strconv.ParseUint(s, 8, 32)
	if !ok || err != nil || n > 0o777 {
		return 0, fmt.Errorf("mode %v must be an octal string such as \"0644\"", v)
	}
	return fs.FileMode(n), nil
}

func (p *Provider) abs(props map[string]any) (string, error) {
	path, _ := props["path"].(string)
	if path == "" || !filepath.IsLocal(path) {
		return "", fmt.Errorf("localfile: unsafe path %q", path)
	}
	return filepath.Join(p.Root, path), nil
}

// Read implements contract.Provider.
func (p *Provider) Read(_ context.Context, rec contract.ResourceState) (contract.Observed, error) {
	path, err := p.abs(rec.Properties)
	if err != nil {
		return contract.Observed{}, err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return contract.Observed{}, contract.ErrNotFound
	}
	if err != nil {
		return contract.Observed{}, err
	}
	wantDir := rec.Kind == KindDirectory
	if info.IsDir() != wantDir || (!wantDir && !info.Mode().IsRegular()) {
		return contract.Observed{}, contract.ErrNotFound
	}
	props := map[string]any{
		"path": rec.Properties["path"],
		"mode": fmt.Sprintf("%04o", info.Mode().Perm()),
	}
	if !wantDir {
		if _, secret := rec.Secrets["content"]; !secret {
			data, err := os.ReadFile(path)
			if err != nil {
				return contract.Observed{}, err
			}
			props["content"] = string(data)
		}
	}
	return contract.Observed{ID: rec.Properties["path"].(string), Properties: props,
		Outputs: map[string]any{"size": info.Size()}}, nil
}

// Create implements contract.Provider; an existing object is converged.
func (p *Provider) Create(ctx context.Context, r contract.Resource, sr contract.SecretResolver) (contract.Observed, error) {
	path, err := p.abs(r.Properties)
	if err != nil {
		return contract.Observed{}, err
	}
	mode := fs.FileMode(0o644)
	if r.Kind == KindDirectory {
		mode = 0o755
	}
	if m, ok := r.Properties["mode"]; ok {
		if mode, err = parseMode(m); err != nil {
			return contract.Observed{}, err
		}
	}
	switch r.Kind {
	case KindDirectory:
		if err := os.MkdirAll(path, mode); err != nil {
			return contract.Observed{}, err
		}
	case KindFile:
		content, _ := r.Properties["content"].(string)
		if ref, ok := r.Secrets["content"]; ok {
			if sr == nil {
				return contract.Observed{}, errors.New("localfile: secret content without resolver")
			}
			if content, err = sr.Resolve(ctx, ref); err != nil {
				return contract.Observed{}, fmt.Errorf("localfile: resolve content: %w", err)
			}
		}
		if err := writeFile(path, []byte(content), mode); err != nil {
			return contract.Observed{}, err
		}
	default:
		return contract.Observed{}, contract.ErrUnsupportedKind
	}
	if err := os.Chmod(path, mode); err != nil {
		return contract.Observed{}, err
	}
	return p.Read(ctx, contract.ResourceState{Resource: r})
}

// Update implements contract.Provider by converging in place.
func (p *Provider) Update(ctx context.Context, r contract.Resource, _ contract.ResourceState, sr contract.SecretResolver) (contract.Observed, error) {
	return p.Create(ctx, r, sr)
}

// Delete implements contract.Provider; an absent object is not an error. A
// non-empty directory is not removed: deleting it would destroy objects the
// engine does not manage.
func (p *Provider) Delete(_ context.Context, rec contract.ResourceState) error {
	path, err := p.abs(rec.Properties)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("localfile: delete %s: %w", rec.Properties["path"], err)
	}
	return nil
}

func writeFile(path string, data []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".krewire-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr, os.Chmod(tmp.Name(), mode)); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
