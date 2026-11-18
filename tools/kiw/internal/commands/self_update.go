package commands

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
	"github.com/krewire/krewire/pkg/version"
	kiwver "github.com/krewire/krewire/tools/kiw/internal/version"
)

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
