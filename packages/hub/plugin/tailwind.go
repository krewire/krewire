package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/krewire/krewire/packages/kern"
)

func init() {
	Register(&Tailwind{})
}

// Tailwind implements Plugin for Tailwind CSS via the Tailwind CLI.
// Detection: presence of tailwind.config.js at the project root.
// Build: runs `npx tailwindcss -i assets/tailwind.css -o <outDir>/assets/tailwind.css --minify`
// (or ./node_modules/.bin/tailwindcss if npx is not available). Failures are
// logged as warnings and do not fail the overall site build — Tailwind is
// an optional plugin, not a core requirement.
// Tailwind plugin constants. The config filenames are checked in three places
// (Detect, Remove, ensureTailwindConfig), so they live here as one list rather
// than being repeated inline.
const (
	// tailwindConfigFile is the config file created by Add when none exists.
	tailwindConfigFile = "tailwind.config.js"
	// tailwindNpmPackage is the npm package installed by Add and removed by Remove.
	tailwindNpmPackage = "tailwindcss"
	// tailwindAssetPath is the logical asset path Tailwind emits, relative to the
	// project root; the build output uses the same path under outDir.
	tailwindAssetPath = "assets/tailwind.css"
	// projectDirPerm is the mode for directories Add and Build create.
	projectDirPerm = 0o755
	// projectFilePerm is the mode for the config and input files Add creates.
	projectFilePerm = 0o644
)

// tailwindConfigFiles lists every config filename Detect recognises, in the order
// Add prefers to keep an existing one.
var tailwindConfigFiles = []string{
	tailwindConfigFile,
	"tailwind.config.cjs",
	"tailwind.config.ts",
}

type Tailwind struct{}

func (t *Tailwind) Name() string { return "tailwind" }

func (t *Tailwind) Aliases() []string { return []string{"twcss", "tailwindcss"} }

func (t *Tailwind) Version() kern.Version {
	return kern.MustParseVersion("3.4.0") // Tailwind CSS v3.4.0
}

func (t *Tailwind) Detect(root string) bool {
	for _, name := range tailwindConfigFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func (t *Tailwind) Build(root, outDir string) error {
	input := filepath.Join(root, filepath.FromSlash(tailwindAssetPath))
	if _, err := os.Stat(input); err != nil {
		slog.Warn("tailwind: "+tailwindAssetPath+" not found, skipping", "root", root)
		return nil
	}
	output := filepath.Join(outDir, filepath.FromSlash(tailwindAssetPath))
	if err := os.MkdirAll(filepath.Dir(output), projectDirPerm); err != nil {
		return fmt.Errorf("tailwind: mkdir: %w", err)
	}

	// Prefer local binary, fall back to npx
	candidates := [][]string{
		{"./node_modules/.bin/" + tailwindNpmPackage, "-i", input, "-o", output, "--minify"},
		{"npx", tailwindNpmPackage, "-i", input, "-o", output, "--minify"},
	}
	var lastErr error
	for _, args := range candidates {
		bin := args[0]
		if _, err := exec.LookPath(bin); err != nil && bin != "npx" {
			continue
		}
		slog.Info("tailwind: building", "input", input, "output", output, "cmd", args)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("tailwind: %v: %s", err, string(out))
			slog.Warn("tailwind: build failed, skipping", "err", lastErr)
			continue
		}
		slog.Info("tailwind: built", "output", output)
		return nil
	}
	if lastErr != nil {
		slog.Warn("tailwind: CLI not available, skipping", "err", lastErr, "hint", "npm install -D "+tailwindNpmPackage)
	}
	return nil
}

// Assets implements plugin.AssetProvider: Tailwind always emits
// assets/tailwind.css, so `kiw build` links it without a manual <link> tag.
func (t *Tailwind) Assets(root string) []string {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(tailwindAssetPath))); err != nil {
		return nil
	}
	return []string{tailwindAssetPath}
}

const tailwindConfigTemplate = `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pages/**/*.kiw","components/**/*.kiw","layouts/**/*.kiw","content/**/*.md"],
  theme: {
    extend: {
      colors: {
        primary: "var(--color-primary)",
      },
    },
  },
  plugins: [],
}
`

const tailwindInputTemplate = `@tailwind base;
@tailwind components;
@tailwind utilities;
`

// Add installs tailwind at version ("" or "latest" means latest) — part of scalable plugin system.
func (t *Tailwind) Add(root, version string) error {
	if version == "" || version == "latest" {
		version = "latest"
	}
	pkg := tailwindNpmPackage + "@" + version
	if version == "latest" {
		pkg = tailwindNpmPackage + "@latest"
	}
	slog.Info("tailwind: installing", "package", pkg, "root", root)
	if err := ensureTailwindConfig(root); err != nil {
		return err
	}
	if err := ensureTailwindInput(root); err != nil {
		return err
	}
	if err := npmInstall(root, pkg, true); err != nil {
		slog.Warn("tailwind: npm install failed, config files still created", "err", err, "hint", "run npm install -D "+pkg+" manually")
		return err
	}
	slog.Info("tailwind: installed", "package", pkg)
	return nil
}

// Remove uninstalls tailwind.
func (t *Tailwind) Remove(root string) error {
	slog.Info("tailwind: removing", "root", root)
	removed := 0
	for _, name := range tailwindConfigFiles {
		p := filepath.Join(root, name)
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("tailwind: remove %s: %w", name, err)
			}
			removed++
			slog.Info("tailwind: removed", "file", name)
		}
	}
	// do not delete assets/tailwind.css if user customized — only if it matches template
	input := filepath.Join(root, filepath.FromSlash(tailwindAssetPath))
	if data, err := os.ReadFile(input); err == nil {
		if string(data) == tailwindInputTemplate {
			_ = os.Remove(input)
			slog.Info("tailwind: removed", "file", tailwindAssetPath)
		} else {
			slog.Info("tailwind: kept customized " + tailwindAssetPath)
		}
	}
	if err := npmUninstall(root, tailwindNpmPackage); err != nil {
		slog.Warn("tailwind: npm uninstall failed", "err", err)
		return err
	}
	if removed == 0 {
		slog.Info("tailwind: already not installed")
	}
	return nil
}

func ensureTailwindConfig(root string) error {
	for _, name := range tailwindConfigFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return nil
		}
	}
	path := filepath.Join(root, tailwindConfigFile)
	if err := os.WriteFile(path, []byte(tailwindConfigTemplate), projectFilePerm); err != nil {
		return fmt.Errorf("tailwind: write config: %w", err)
	}
	slog.Info("tailwind: created", "file", tailwindConfigFile)
	return nil
}

func ensureTailwindInput(root string) error {
	path := filepath.Join(root, filepath.FromSlash(tailwindAssetPath))
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), projectDirPerm); err != nil {
		return fmt.Errorf("tailwind: mkdir assets: %w", err)
	}
	if err := os.WriteFile(path, []byte(tailwindInputTemplate), projectFilePerm); err != nil {
		return fmt.Errorf("tailwind: write input: %w", err)
	}
	slog.Info("tailwind: created", "file", tailwindAssetPath)
	return nil
}

func npmInstall(root, pkg string, dev bool) error {
	args := []string{"install", pkg}
	if dev {
		args = []string{"install", "-D", pkg}
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH")
	}
	cmd := exec.Command("npm", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install %s: %v: %s", pkg, err, string(out))
	}
	return nil
}

func npmUninstall(root, pkg string) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH")
	}
	cmd := exec.Command("npm", "uninstall", pkg)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm uninstall %s: %v: %s", pkg, err, string(out))
	}
	return nil
}
