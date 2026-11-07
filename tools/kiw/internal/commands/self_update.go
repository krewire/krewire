package commands

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
	"github.com/krewire/krewire/pkg/version"
	kiwver "github.com/krewire/krewire/tools/kiw/internal/version"
)

var (
	// defaultReleaseURL is the GitHub API endpoint for the latest release.
	// Overridden in tests to point to a mock HTTP server.
	defaultReleaseURL = "https://api.github.com/repos/krewire/krewire/releases/latest"
	// defaultHTTPTimeout is the network timeout for checking releases.
	defaultHTTPTimeout = 10 * time.Second
)

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt string        `json:"published_at"`
	HTMLURL     string        `json:"html_url"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// RegisterSelfUpdate registers flags for `kiw self-update`.
func RegisterSelfUpdate(fs *flag.FlagSet) {
	fs.Bool("check", false, "check for available updates without installing")
	fs.Bool("force", false, "reinstall the latest version even if already up to date")
	fs.Bool("dry-run", false, "simulate update without replacing the binary")
	fs.String("version", "", "update to a specific version (default: latest stable)")
}

// RunSelfUpdate checks for and installs the latest stable version of the kiw devtool.
func RunSelfUpdate(fs *flag.FlagSet) errs.ExitCode {
	checkOnly := flagValue(fs, "check") == "true"
	force := flagValue(fs, "force") == "true"
	dryRun := flagValue(fs, "dry-run") == "true"
	targetVersionReq := flagValue(fs, "version")

	tm := term.NewTerminal()
	bold := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
	yellow := func(s string) string { return tm.Paint(s, term.ColorYellow, nil) }

	curVerStr := kiwver.VersionString()
	cleanCur := strings.TrimPrefix(curVerStr, "v")
	curVer, err := version.ParseVersion(cleanCur)
	if err != nil {
		curVer = kiwver.Version
	}

	fmt.Printf("%s Checking for updates (current version: %s) ...\n", bold("→"), green("v"+curVer.String()))

	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	rel, err := fetchLatestRelease(ctx, defaultReleaseURL)
	if err != nil {
		slog.Warn("failed to fetch release from GitHub API", "error", err)
		is404 := strings.Contains(err.Error(), "HTTP 404")
		if checkOnly {
			if is404 {
				fmt.Printf("%s No newer releases found on GitHub; kiw is running current version (v%s)\n", green("✓"), curVer.String())
				return errs.ExitCodeSuccess
			}
			fmt.Fprintf(os.Stderr, "error checking for updates: %v\n", err)
			return errs.ExitCodeFailure
		}
		if dryRun {
			if is404 {
				fmt.Printf("%s [dry-run] No remote GitHub release found for kiw (current version: v%s)\n", yellow("!"), curVer.String())
				return errs.ExitCodeSuccess
			}
			fmt.Fprintf(os.Stderr, "error checking for updates: %v\n", err)
			return errs.ExitCodeFailure
		}
		// If actual update requested, fallback to Go toolchain
		return fallbackGoInstallUpdate(curVer, "latest", dryRun)
	}

	targetTag := rel.TagName
	if targetVersionReq != "" {
		targetTag = targetVersionReq
	}
	cleanTarget := strings.TrimPrefix(targetTag, "v")
	targetVer, err := version.ParseVersion(cleanTarget)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid release tag %q: %v\n", targetTag, err)
		return errs.ExitCodeFailure
	}

	cmp := targetVer.Compare(curVer)
	if cmp <= 0 && !force {
		fmt.Printf("%s %s (v%s)\n", green("✓"), "kiw is already up to date", curVer.String())
		return errs.ExitCodeSuccess
	}

	if checkOnly {
		if cmp > 0 {
			fmt.Printf("%s A new stable release of kiw is available: %s -> %s\n", yellow("!"), yellow("v"+curVer.String()), green("v"+targetVer.String()))
			fmt.Println("  Run 'kiw self-update' to update your installation.")
		} else {
			fmt.Printf("%s kiw is up to date (v%s)\n", green("✓"), curVer.String())
		}
		return errs.ExitCodeSuccess
	}

	if dryRun {
		fmt.Printf("%s Would update kiw from v%s to v%s\n", yellow("[dry-run]"), curVer.String(), targetVer.String())
		return errs.ExitCodeSuccess
	}

	fmt.Printf("%s Updating kiw devtool: %s -> %s ...\n", bold("→"), yellow("v"+curVer.String()), green("v"+targetVer.String()))

	// Locate current executable
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine kiw executable path: %v\n", err)
		return errs.ExitCodeFailure
	}
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		execPath = resolved
	}

	// Try downloading binary release asset first
	asset := findMatchingAsset(rel.Assets, targetVer.String(), runtime.GOOS, runtime.GOARCH)
	if asset != nil {
		slog.Info("downloading release asset", "asset", asset.Name, "url", asset.BrowserDownloadURL)
		fmt.Printf("  Downloading %s (%d bytes) ...\n", asset.Name, asset.Size)
		if err := downloadAndInstallAsset(ctx, asset.BrowserDownloadURL, asset.Name, execPath); err == nil {
			fmt.Printf("%s kiw successfully updated to %s\n", green("✓"), green("v"+targetVer.String()))
			fmt.Printf("  Binary installed at: %s\n", execPath)
			return errs.ExitCodeSuccess
		} else {
			slog.Warn("asset installation failed, falling back to go install", "error", err)
			fmt.Fprintf(os.Stderr, "  Asset installation encountered an issue: %v. Trying Go toolchain fallback...\n", err)
		}
	}

	// Fallback to Go toolchain
	return fallbackGoInstallUpdate(curVer, "v"+targetVer.String(), dryRun)
}

func fetchLatestRelease(ctx context.Context, url string) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "kiw-self-update/"+kiwver.VersionString())

	client := &http.Client{Timeout: defaultHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding release JSON: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release has no tag_name")
	}
	return &rel, nil
}

func findMatchingAsset(assets []githubAsset, ver, targetOS, targetArch string) *githubAsset {
	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if strings.Contains(name, targetOS) && strings.Contains(name, targetArch) {
			if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
				return &assets[i]
			}
		}
	}
	return nil
}

func downloadAndInstallAsset(ctx context.Context, url, assetName, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "kiw-self-update/"+kiwver.VersionString())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tempDir, err := os.MkdirTemp("", "kiw-self-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, assetName)
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()

	extractedBinary := filepath.Join(tempDir, "kiw-extracted")
	if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
		if err := extractTarGzBinary(archivePath, extractedBinary); err != nil {
			return err
		}
	} else if strings.HasSuffix(assetName, ".zip") {
		if err := extractZipBinary(archivePath, extractedBinary); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported archive format: %s", assetName)
	}

	return replaceBinary(extractedBinary, destPath)
}

func extractTarGzBinary(archivePath, destBinary string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		base := filepath.Base(hdr.Name)
		if base == "kiw" || base == "kiw.exe" {
			out, err := os.OpenFile(destBinary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			return nil
		}
	}
	return fmt.Errorf("binary 'kiw' not found in archive")
}

func extractZipBinary(archivePath, destBinary string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, file := range zr.File {
		base := filepath.Base(file.Name)
		if base == "kiw" || base == "kiw.exe" {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.OpenFile(destBinary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				return err
			}
			out.Close()
			return nil
		}
	}
	return fmt.Errorf("binary 'kiw' not found in zip archive")
}

func replaceBinary(newBinary, targetPath string) error {
	if err := os.Chmod(newBinary, 0755); err != nil {
		return err
	}

	// Try atomic rename in same directory
	targetDir := filepath.Dir(targetPath)
	tempTarget := filepath.Join(targetDir, fmt.Sprintf(".kiw-new-%d", time.Now().UnixNano()))

	in, err := os.Open(newBinary)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(tempTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("cannot write to target directory %s (may need sudo): %w", targetDir, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tempTarget)
		return err
	}
	out.Close()

	// On Windows, target may be locked by running process; rename existing to .old
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		_ = os.Remove(oldPath)
		_ = os.Rename(targetPath, oldPath)
	}

	if err := os.Rename(tempTarget, targetPath); err != nil {
		os.Remove(tempTarget)
		return fmt.Errorf("replacing binary at %s: %w", targetPath, err)
	}
	return nil
}

func fallbackGoInstallUpdate(curVer version.Version, targetTag string, dryRun bool) errs.ExitCode {
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintf(os.Stderr, "Go toolchain not found in PATH and GitHub releases unavailable.\n")
		fmt.Fprintf(os.Stderr, "Install or update manually: curl -fsSL https://krewire.com/scripts/install.sh | sh\n")
		return errs.ExitCodeFailure
	}

	pkg := "github.com/krewire/krewire/tools/kiw/cmd/kiw@" + targetTag
	if dryRun {
		fmt.Printf("[dry-run] Would run: go install %s\n", pkg)
		return errs.ExitCodeSuccess
	}

	fmt.Printf("  Running: go install %s ...\n", pkg)
	cmd := exec.Command("go", "install", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "go install failed: %v\n", err)
		return errs.ExitCodeFailure
	}

	tm := term.NewTerminal()
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
	fmt.Printf("%s kiw successfully updated via go install (%s)\n", green("✓"), targetTag)
	return errs.ExitCodeSuccess
}
