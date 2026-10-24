package deploy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BinaryDeployer stages compiled binaries and produces checksum manifests.
type BinaryDeployer struct{}

// Target returns TargetBinary.
func (d *BinaryDeployer) Target() Target {
	return TargetBinary
}

// Deploy calculates checksums of artifacts in ArtifactDir and writes SHA256SUMS.
func (d *BinaryDeployer) Deploy(ctx context.Context, opts Options) (*Result, error) {
	if opts.ArtifactDir == "" {
		return nil, fmt.Errorf("deploy binary: artifact dir is required")
	}
	if err := os.MkdirAll(opts.ArtifactDir, 0o755); err != nil {
		return nil, fmt.Errorf("deploy binary: %w", err)
	}

	entries, err := os.ReadDir(opts.ArtifactDir)
	if err != nil {
		return nil, fmt.Errorf("deploy binary: read dir: %w", err)
	}

	var checksumLines []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "SHA256SUMS" {
			continue
		}
		path := filepath.Join(opts.ArtifactDir, entry.Name())
		hash, err := fileSHA256(path)
		if err != nil {
			return nil, fmt.Errorf("deploy binary: checksum %s: %w", entry.Name(), err)
		}
		checksumLines = append(checksumLines, fmt.Sprintf("%s  %s", hash, entry.Name()))
	}

	if len(checksumLines) > 0 {
		sort.Strings(checksumLines)
		sumsContent := strings.Join(checksumLines, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(opts.ArtifactDir, "SHA256SUMS"), []byte(sumsContent), 0o644); err != nil {
			return nil, fmt.Errorf("deploy binary: write SHA256SUMS: %w", err)
		}
	}

	return &Result{
		Target:  TargetBinary,
		Success: true,
		Message: fmt.Sprintf("binary deployment ready in %s", opts.ArtifactDir),
	}, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
