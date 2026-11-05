package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	"github.com/krewire/krewire/tools/kiw/internal/shape"
)

const (
	// DefaultTarget is the default build target workload.
	DefaultTarget = "ssg"

	// DefaultThemeMode is the default theme selection mode.
	DefaultThemeMode = "auto"

	// DefaultWASMEntry is the default WASM source directory.
	DefaultWASMEntry = "./wasm"

	// DefaultWASMName is the default WASM module output identifier.
	DefaultWASMName = "runtime"

	// DefaultBinaryName is the default fallback binary name.
	DefaultBinaryName = "app"
)

// RegisterBuild registers flags for the build command.
func RegisterBuild(fs *flag.FlagSet) {
	fs.String("input", "", "content directory (default content)")
	fs.String("output", "", "output directory (default .krewire/build)")
	fs.String("o", "", "output directory (shorthand for --output)")
	fs.String("include", "", "comma-separated content glob patterns to build (default **/*.md)")
	fs.String("exclude", "", "comma-separated content globs to skip (default **/README.md,**/readme.md)")
	fs.String("base", "", "URL base the site will be served under (default /)")
	fs.String("title", "", "site title (defaults to the project name)")
	fs.String("author", "", "site author")
	fs.String("theme", "", "theme mode: auto, light, dark, or off (default auto)")
	fs.String("target", DefaultTarget, "build target: site, book, binary, wasm, or infra (default ssg)")
	fs.Bool("skip-deps", false, "skip executing build.depends_on commands")
	fs.Bool("deps-only", false, "run only build.depends_on commands and skip Krewire build")
	fs.Bool("bundle", true, "bundle all CSS and JS into assets/krewire.css and assets/krewire.js (default true)")
	fs.Bool("minify", false, "minify CSS and JS single-bundle assets")
	fs.Bool("plan", false, "infra: compile and show infrastructure plan without applying")
	fs.String("env", "", "infra: target environment (local, production, testing)")
}

// RunBuild builds the project for production. It dispatches according to the
// detected project workload kind (site, book, app, cli, worker, service, infra)
// or explicit --target flag (KWN-BUILD-001).
//
// Progressive enhancement is preserved: when web pages or content docs coexist
// in an app, worker, or service project, both the Go binary and web assets are built
// into the shared output (.krewire/build).
func RunBuild(fs *flag.FlagSet) errs.ExitCode {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return errs.ExitCodeUsage
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fail(err)
	}

	skipDeps := flagValue(fs, "skip-deps") == "true"
	depsOnly := flagValue(fs, "deps-only") == "true"

	if depsOnly {
		if len(cfg.Build.DependsOn) == 0 {
			return errs.ExitCodeSuccess
		}
		_ = cfg.LoadDotEnv(root)
		return executeBuildDeps(root, cfg.Build.DependsOn, os.Environ())
	}

	if !skipDeps && len(cfg.Build.DependsOn) > 0 {
		_ = cfg.LoadDotEnv(root)
		if code := executeBuildDeps(root, cfg.Build.DependsOn, os.Environ()); code != errs.ExitCodeSuccess {
			return code
		}
	}

	outDir := joinRoot(root, firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output), config.DefaultOutput)
	pruneStale(outDir)

	target := ""
	if flagProvided(fs, "target") {
		target = strings.ToLower(strings.TrimSpace(flagValue(fs, "target")))
	}

	// Explicit target overrides
	switch target {
	case "wasm":
		if code := buildWASM(root, cfg, fs); code != errs.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return errs.ExitCodeSuccess
	case "infra":
		if code := buildInfra(root, cfg, fs); code != errs.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return errs.ExitCodeSuccess
	case "binary":
		if code := buildAppBinary(root, cfg, fs, shape.KindApp); code != errs.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return errs.ExitCodeSuccess
	case "ssg", "site":
		hasPages := hasDir(root, "pages")
		hasSSG := cfg.IsSSG()
		if !hasPages && !hasSSG {
			fmt.Fprintln(os.Stderr, "kiw build: no SSG pages or ssg: config found")
			return errs.ExitCodeUsage
		}
		var code errs.ExitCode
		if hasPages {
			code, _, _ = buildSSGFromFile(root, cfg, fs)
		} else {
			code, _, _ = buildSSGFromConfig(root, cfg, fs)
		}
		if code != errs.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return errs.ExitCodeSuccess
	case "book":
		hasBook := hasDir(root, "content")
		if !hasBook {
			fmt.Fprintln(os.Stderr, "kiw build: no content/ directory found for book")
			return errs.ExitCodeUsage
		}
		if code := buildManuscript(root, cfg, fs, false, nil, nil, ""); code != errs.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return errs.ExitCodeSuccess
	case "":
		// auto-detect below
	default:
		fmt.Fprintf(os.Stderr, "kiw build: unknown target %q — supported: site, book, binary, wasm, infra\n", target)
		return errs.ExitCodeUsage
	}

	// Auto-detect project shape
	res, err := shape.Detect(root, cfg.Kind())
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return errs.ExitCodeUsage
	}

	if res.IsPreInit() {
		fmt.Fprintln(os.Stderr, "kiw build: project is not equipped yet — run 'kiw init' to choose a starter")
		return errs.ExitCodeUsage
	}

	hasPages := hasDir(root, "pages")
	hasSSG := cfg.IsSSG()
	hasBook := hasDir(root, "content")

	built := false
	var firstErr errs.ExitCode = errs.ExitCodeSuccess

	// 1. Infra workload dispatch
	if res.Kind == shape.KindInfra {
		if code := buildInfra(root, cfg, fs); code != errs.ExitCodeSuccess {
			return code
		}
		built = true
	}

	// 2. Binary workload dispatch (app, cli, worker, service, gateway)
	if res.Kind == shape.KindApp || res.Kind == shape.KindCLI || res.Kind == shape.KindWorker || res.Kind == shape.KindService || res.Kind == shape.KindGateway {
		if code := buildAppBinary(root, cfg, fs, res.Kind); code != errs.ExitCodeSuccess {
			return code
		}
		built = true
	}

	// 3. Static web assets / Book documentation
	var planCSS, planJS []string
	if hasPages {
		code, css, js := buildSSGFromFile(root, cfg, fs)
		planCSS, planJS = css, js
		if code != errs.ExitCodeSuccess {
			if firstErr == errs.ExitCodeSuccess {
				firstErr = code
			}
		} else {
			built = true
		}
	} else if hasSSG {
		code, css, js := buildSSGFromConfig(root, cfg, fs)
		planCSS, planJS = css, js
		if code != errs.ExitCodeSuccess {
			if firstErr == errs.ExitCodeSuccess {
				firstErr = code
			}
		} else {
			built = true
		}
	}

	if hasBook {
		hybrid := hasPages || hasSSG

		// I18n: build per-locale book output trees when i18n is configured.
		// content/<locale>/docs/ (or legacy content/docs/<locale>/) → .krewire/build/<locale>/docs/
		if cfg.I18n != nil && len(cfg.I18n.Locales) > 0 {
			for _, locale := range cfg.I18n.Locales {
				localeInput := filepath.Join("content", locale, "docs")
				if !hasDir(root, localeInput) {
					localeInput = filepath.Join("content", "docs", locale)
				}
				if !hasDir(root, localeInput) && cfg.Input != "" {
					localeInput = filepath.Join(cfg.Input, locale)
					if !hasDir(root, localeInput) {
						localeInput = filepath.Join(cfg.Input, locale, "docs")
					}
				}
				if !hasDir(root, localeInput) {
					slog.Warn("i18n: skipping locale, input dir missing", "locale", locale, "input", localeInput)
					continue
				}
				origInput := cfg.Input
				origOutput := cfg.Output
				origBase := cfg.Base
				cfg.Input = localeInput
				if locale == cfg.I18n.DefaultLocale {
					cfg.Output = origOutput
				} else {
					cfg.Output = filepath.Join(origOutput, locale)
					cfg.Base = localeBasePath(origBase, locale, cfg.Book.Mount)
				}
				slog.Info("i18n: building book for locale", "locale", locale, "input", cfg.Input, "output", cfg.Output, "base", cfg.Base)
				if code := buildManuscript(root, cfg, fs, hybrid, planCSS, planJS, locale); code != errs.ExitCodeSuccess {
					if firstErr == errs.ExitCodeSuccess {
						firstErr = code
					}
				} else {
					built = true
					if locale == cfg.I18n.DefaultLocale {
						_ = copyDir(filepath.Join(root, origOutput, "docs"), filepath.Join(root, origOutput, locale, "docs"))
					}
				}
				cfg.Input = origInput
				cfg.Output = origOutput
				cfg.Base = origBase
			}
		} else {
			if code := buildManuscript(root, cfg, fs, hybrid, planCSS, planJS, ""); code != errs.ExitCodeSuccess {
				if firstErr == errs.ExitCodeSuccess {
					firstErr = code
				}
			} else {
				built = true
			}
		}
	}

	// 4. WASM entry point if configured or wasm directory present
	if cfg.Wasm.Entry != "" || hasDir(root, "wasm") {
		if code := buildWASM(root, cfg, fs); code != errs.ExitCodeSuccess {
			return code
		}
		built = true
	}

	if !built {
		if firstErr != errs.ExitCodeSuccess {
			return firstErr
		}
		fmt.Fprintln(os.Stderr, "kiw: no buildable workload found — add main.go, pages/*.kiw, an `ssg:`/`infra:` key to krewire.yaml, or content/")
		return errs.ExitCodeUsage
	}

	writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
	return errs.ExitCodeSuccess
}
