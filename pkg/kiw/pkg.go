package kiw

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackageManifestFile is the canonical manifest filename for a standalone .kiw component package.
const PackageManifestFile = "kiw.pkg.json"

// PackageExtension is the standard file extension for distributed .kiw component packages.
const PackageExtension = ".kpkg"

// DefaultPackageDir is the canonical relative directory where standalone component packages are installed.
const DefaultPackageDir = ".krewire/pkg"

// PackageManifest defines the metadata and component export catalog of an independently distributable .kiw component package.
type PackageManifest struct {
	Schema       string            `json:"$schema,omitempty"`
	Name         string            `json:"name"`                   // e.g. "@acme/ui" or "acme-ui"
	Version      string            `json:"version"`                // SemVer, e.g. "1.0.0"
	Description  string            `json:"description,omitempty"`  // Summary
	Author       string            `json:"author,omitempty"`       // Author or organization
	License      string            `json:"license,omitempty"`      // e.g. "MIT"
	Main         string            `json:"main,omitempty"`         // Primary entrypoint or directory, e.g. "./components"
	Components   map[string]string `json:"components"`             // Component name to relative .kiw path
	Styles       string            `json:"styles,omitempty"`       // Optional shared CSS bundle path
	Scripts      string            `json:"scripts,omitempty"`      // Optional shared client JS bundle path
	Dependencies map[string]string `json:"dependencies,omitempty"` // Dependent packages and version ranges
}

// LoadPackageManifest decodes a kiw.pkg.json manifest from path.
func LoadPackageManifest(path string) (*PackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pkg: read %s: %w", path, err)
	}
	var m PackageManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("pkg: decode %s: %w", path, err)
	}
	if err := ValidatePackageManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SavePackageManifest writes a PackageManifest to disk as formatted JSON.
func SavePackageManifest(path string, m *PackageManifest) error {
	if err := ValidatePackageManifest(m); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("pkg: encode manifest: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("pkg: write manifest %s: %w", path, err)
	}
	return nil
}

// ValidatePackageManifest enforces validity constraints on a component package manifest.
func ValidatePackageManifest(m *PackageManifest) error {
	if m == nil {
		return errors.New("pkg: manifest is nil")
	}
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("pkg: manifest 'name' is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("pkg: manifest 'version' is required")
	}
	if len(m.Components) == 0 {
		return errors.New("pkg: manifest must export at least one component in 'components'")
	}
	for compName, compPath := range m.Components {
		if strings.TrimSpace(compName) == "" {
			return errors.New("pkg: component name cannot be empty")
		}
		if strings.TrimSpace(compPath) == "" {
			return fmt.Errorf("pkg: component %q has empty path", compName)
		}
		// SafePath: ensure relative component paths do not escape package root
		clean := filepath.Clean(compPath)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("pkg: component %q path %q must be relative inside package", compName, compPath)
		}
	}
	return nil
}

// PackPackage bundles a component package directory into a standalone .kpkg archive.
// Returns the written archive path and its SHA-256 integrity hash.
func PackPackage(dir, outPath string) (archivePath string, sha256Hex string, err error) {
	manifestPath := filepath.Join(dir, PackageManifestFile)
	manifest, err := LoadPackageManifest(manifestPath)
	if err != nil {
		return "", "", fmt.Errorf("pkg: pack failed to load manifest: %w", err)
	}

	if outPath == "" {
		sanitizedName := strings.ReplaceAll(strings.TrimPrefix(manifest.Name, "@"), "/", "-")
		outPath = fmt.Sprintf("%s-%s%s", sanitizedName, manifest.Version, PackageExtension)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return "", "", fmt.Errorf("pkg: create archive %s: %w", outPath, err)
	}
	defer outFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(outFile, hasher)
	gw := gzip.NewWriter(multiWriter)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Walk directory and archive package files
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." || rel == "" {
			return nil
		}

		// Skip hidden files, node_modules, .git, and existing output archives
		base := info.Name()
		if strings.HasPrefix(base, ".") && base != PackageManifestFile {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if base == "node_modules" || strings.HasSuffix(base, PackageExtension) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		tarPath := filepath.ToSlash(rel)
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = tarPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return "", "", fmt.Errorf("pkg: archive error: %w", err)
	}

	if err := tw.Close(); err != nil {
		return "", "", err
	}
	if err := gw.Close(); err != nil {
		return "", "", err
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))
	return outPath, hashStr, nil
}

// UnpackPackage unpacks a .kpkg component archive into destDir with SafePath traversal guards.
func UnpackPackage(archivePath, destDir string) (*PackageManifest, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("pkg: open archive %s: %w", archivePath, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("pkg: gzip reader %s: %w", archivePath, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("pkg: mkdir %s: %w", destDir, err)
	}

	cleanDest := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("pkg: read archive tar: %w", err)
		}

		// SafePath: prevent directory traversal (ZipSlip / TarSlip)
		target := filepath.Join(cleanDest, filepath.FromSlash(header.Name))
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) && cleanTarget != cleanDest {
			return nil, fmt.Errorf("pkg: security error: illegal path traversal in archive: %q", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
				return nil, err
			}
			outFile, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return nil, err
			}
			outFile.Close()
		}
	}

	manifestPath := filepath.Join(cleanDest, PackageManifestFile)
	manifest, err := LoadPackageManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("pkg: unpacked package missing valid %s: %w", PackageManifestFile, err)
	}
	return manifest, nil
}
