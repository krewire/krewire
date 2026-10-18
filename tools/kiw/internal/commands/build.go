package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/packages/cloud/infra/provider"
	"github.com/krewire/krewire/packages/cloud/infra/schema"
	"github.com/krewire/krewire/packages/hub/plugin"
	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/packages/runtime/build"
	"github.com/krewire/krewire/packages/ui"
	"github.com/krewire/krewire/packages/web/ssg"
	"github.com/krewire/krewire/tools/kiw/internal/buildinfo"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	"github.com/krewire/krewire/tools/kiw/internal/shape"
	"github.com/krewire/mdbind/book"
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
	fs.String("target", "ssg", "build target: site, book, binary, wasm, or infra (default ssg)")
	fs.Bool("skip-deps", false, "skip executing build.depends_on commands")
	fs.Bool("deps-only", false, "run only build.depends_on commands and skip Krewire build")
	fs.Bool("bundle", false, "bundle all CSS and JS into assets/krewire.css and assets/krewire.js")
	fs.Bool("plan", false, "infra: compile and show infrastructure plan without applying")
	fs.String("env", "", "infra: target environment (local, production, testing)")
}

// RunBuild builds the project for production. It dispatches according to the
// detected project workload kind (site, book, app, cli, worker, service, infra)
// or explicit --target flag (KWN-BUILD-001).
//
// Progressive enhancement is preserved: when web pages or manuscript docs coexist
// in an app, worker, or service project, both the Go binary and web assets are built
// into the shared output (.krewire/build).
func RunBuild(fs *flag.FlagSet) kern.ExitCode {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return kern.ExitCodeUsage
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fail(err)
	}

	skipDeps := flagValue(fs, "skip-deps") == "true"
	depsOnly := flagValue(fs, "deps-only") == "true"

	if depsOnly {
		if len(cfg.Build.DependsOn) == 0 {
			return kern.ExitCodeSuccess
		}
		_ = cfg.LoadDotEnv(root)
		return executeBuildDeps(root, cfg.Build.DependsOn, os.Environ())
	}

	if !skipDeps && len(cfg.Build.DependsOn) > 0 {
		_ = cfg.LoadDotEnv(root)
		if code := executeBuildDeps(root, cfg.Build.DependsOn, os.Environ()); code != kern.ExitCodeSuccess {
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
		if code := buildWASM(root, cfg, fs); code != kern.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return kern.ExitCodeSuccess
	case "infra":
		if code := buildInfra(root, cfg, fs); code != kern.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return kern.ExitCodeSuccess
	case "binary":
		if code := buildAppBinary(root, cfg, fs, shape.KindApp); code != kern.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return kern.ExitCodeSuccess
	case "ssg", "site":
		hasPages := hasDir(root, "pages")
		hasSSG := cfg.IsSSG()
		if !hasPages && !hasSSG {
			fmt.Fprintln(os.Stderr, "kiw build: no SSG pages or ssg: config found")
			return kern.ExitCodeUsage
		}
		var code kern.ExitCode
		if hasPages {
			code, _, _ = buildSSGFromFile(root, cfg, fs)
		} else {
			code, _, _ = buildSSGFromConfig(root, cfg, fs)
		}
		if code != kern.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return kern.ExitCodeSuccess
	case "book":
		hasBook := hasDir(root, "content") || hasDir(root, "manuscript")
		if !hasBook {
			fmt.Fprintln(os.Stderr, "kiw build: no content/ or manuscript/ directory found for book")
			return kern.ExitCodeUsage
		}
		if code := buildManuscript(root, cfg, fs, false, nil, nil); code != kern.ExitCodeSuccess {
			return code
		}
		writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
		return kern.ExitCodeSuccess
	case "":
		// auto-detect below
	default:
		fmt.Fprintf(os.Stderr, "kiw build: unknown target %q — supported: site, book, binary, wasm, infra\n", target)
		return kern.ExitCodeUsage
	}

	// Auto-detect project shape
	res, err := shape.Detect(root, cfg.Kind())
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return kern.ExitCodeUsage
	}

	if res.Kind == shape.KindKernel {
		fmt.Fprintln(os.Stderr, "kiw build: kernel project must be initialized first — run 'kiw init' to choose a starter")
		return kern.ExitCodeUsage
	}

	hasPages := hasDir(root, "pages")
	hasSSG := cfg.IsSSG()
	hasBook := hasDir(root, "content") || hasDir(root, "manuscript")

	built := false
	var firstErr kern.ExitCode = kern.ExitCodeSuccess

	// 1. Infra workload dispatch
	if res.Kind == shape.KindInfra {
		if code := buildInfra(root, cfg, fs); code != kern.ExitCodeSuccess {
			return code
		}
		built = true
	}

	// 2. Binary workload dispatch (app, cli, worker, service)
	if res.Kind == shape.KindApp || res.Kind == shape.KindCLI || res.Kind == shape.KindWorker || res.Kind == shape.KindService {
		if code := buildAppBinary(root, cfg, fs, res.Kind); code != kern.ExitCodeSuccess {
			return code
		}
		built = true
	}

	// 3. Static web assets / Book documentation
	var planCSS, planJS []string
	if hasPages {
		code, css, js := buildSSGFromFile(root, cfg, fs)
		planCSS, planJS = css, js
		if code != kern.ExitCodeSuccess {
			if firstErr == kern.ExitCodeSuccess {
				firstErr = code
			}
		} else {
			built = true
		}
	} else if hasSSG {
		code, css, js := buildSSGFromConfig(root, cfg, fs)
		planCSS, planJS = css, js
		if code != kern.ExitCodeSuccess {
			if firstErr == kern.ExitCodeSuccess {
				firstErr = code
			}
		} else {
			built = true
		}
	}

	if hasBook {
		hybrid := hasPages || hasSSG
		if code := buildManuscript(root, cfg, fs, hybrid, planCSS, planJS); code != kern.ExitCodeSuccess {
			if firstErr == kern.ExitCodeSuccess {
				firstErr = code
			}
		} else {
			built = true
		}
	}

	// 4. WASM entry point if configured
	if cfg.Wasm.Entry != "" {
		if code := buildWASM(root, cfg, fs); code != kern.ExitCodeSuccess {
			return code
		}
		built = true
	}

	if !built {
		if firstErr != kern.ExitCodeSuccess {
			return firstErr
		}
		fmt.Fprintln(os.Stderr, "kiw: no buildable workload found — add main.go, pages/*.kiw, an `ssg:`/`infra:` key to krewire.yaml, or content/")
		return kern.ExitCodeUsage
	}

	writeManifest(outDir, collectCreated(outDir), cfg.Build.Outputs)
	return kern.ExitCodeSuccess
}

// buildSSGFromFile builds the project's SSG site from the file-based layout
// (pages/, components/, layouts/, content/, public/) using
// ssg.LoadFromDir. krewire.yaml supplies metadata (title, description, theme)
// and output dir.
//
// Detected plugins (e.g. Tailwind via tailwind.config.js) run first and their
// CSS/JS output paths are declared to the site, so `kiw build` auto-injects the
// matching tags — the layout never names plugin assets by hand.
func buildSSGFromFile(root string, cfg *config.Config, fs *flag.FlagSet) (kern.ExitCode, []string, []string) {
	output := firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output, config.DefaultOutput)
	outDir := joinRoot(root, output, config.DefaultOutput)
	slog.Info("building SSG site from file layout", "root", root, "output", outDir)
	site, err := ssg.LoadFromDir(root)
	if err != nil {
		return fail(err), nil, nil
	}
	// The effective URL base honors kiw --base first, then krewire.yaml
	// (build.base folded into cfg.Base at load). A file-based project has
	// no `ssg:` block, so thread it here; declarative sites receive it
	// through ToSSGConfig instead.
	if base := firstNonEmpty(flagValue(fs, "base"), cfg.Base, ""); base != "" {
		site.WithBasePath(base)
	}
	// Run detected plugins (Tailwind is the first; others follow the same pattern).
	for _, p := range plugin.Registry {
		if !p.Detect(root) {
			continue
		}
		slog.Info("plugin detected", "plugin", p.Name())
		if err := p.Build(root, outDir); err != nil {
			slog.Warn("plugin build failed", "plugin", p.Name(), "err", err)
			continue
		}
		slog.Info("plugin built", "plugin", p.Name())
		if ap, ok := p.(plugin.AssetProvider); ok {
			site.DeclareAsset(ap.Assets(root)...)
		}
	}
	if flagValue(fs, "bundle") == "true" {
		site.WithBundle(true)
	}
	created, err := site.Build(outDir)
	if err != nil {
		return fail(err), nil, nil
	}
	warnAutoAssetErrors(site)
	// Post-render build pass: plugins that scan generated HTML artifacts
	// (such as Tailwind) re-run to capture classes emitted by component expansions.
	for _, p := range plugin.Registry {
		if p.Detect(root) {
			if err := p.Build(root, outDir); err != nil {
				slog.Warn("plugin post-build failed", "plugin", p.Name(), "err", err)
			}
		}
	}
	for _, p := range created {
		fmt.Println("created " + p)
	}
	css, js := site.InjectedAssets()
	return kern.ExitCodeSuccess, css, js
}

// buildSSGFromConfig builds the project's SSG site from the `ssg:` section
// of krewire.yaml. Top-level fields (title, output, theme) are merged into the
// ssg.Config so they don't need to be repeated under ssg:.
func buildSSGFromConfig(root string, cfg *config.Config, fs *flag.FlagSet) (kern.ExitCode, []string, []string) {
	ssgCfg := cfg.ToSSGConfig()
	if flagValue(fs, "bundle") == "true" {
		if ssgCfg.AutoAssets == nil {
			ssgCfg.AutoAssets = &ssg.AutoAssetConfig{}
		}
		trueVal := true
		ssgCfg.AutoAssets.Bundle = &trueVal
	}
	output := firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output, config.DefaultOutput)
	outDir := joinRoot(root, output, config.DefaultOutput)
	slog.Info("building SSG site from krewire.yaml", "output", outDir)
	site, created, err := ssg.BuildFromConfigSite(ssgCfg, outDir)
	if err != nil {
		return fail(err), nil, nil
	}
	warnAutoAssetErrors(site)
	for _, p := range created {
		fmt.Println("created " + p)
	}
	css, js := site.InjectedAssets()
	return kern.ExitCodeSuccess, css, js
}

// buildManuscript renders the project's content/ directory with mdbind.
// Settings come from krewire.yaml in the project root, overridden by flags.
// In a hybrid project (ssg pages present) the book suppresses its generated
// root TOC so the ssg landing page owns "/". Include/exclude path globs
// resolve flag > krewire.yaml `build:` > mdbind defaults (README skipped).
func buildManuscript(root string, cfg *config.Config, fs *flag.FlagSet, hybrid bool, planCSS, planJS []string) kern.ExitCode {
	title := firstNonEmpty(flagValue(fs, "title"), cfg.Title, moduleName(root))
	input := firstNonEmpty(flagValue(fs, "input"), cfg.Input)
	if input == "" {
		input = config.DefaultInput
		if !hasDir(root, input) && hasDir(root, "manuscript") {
			input = "manuscript"
		}
	}
	include := globList(fs, "include", cfg.Build.Include)
	exclude := globList(fs, "exclude", cfg.Build.Exclude)
	bcfg := book.Config{
		Input:      joinRoot(root, input, config.DefaultInput),
		Output:     joinRoot(root, firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output), config.DefaultOutput),
		Title:      title,
		Author:     firstNonEmpty(flagValue(fs, "author"), cfg.Author),
		BasePath:   firstNonEmpty(flagValue(fs, "base"), cfg.Base, "/"),
		Version:    cfg.Version,
		NavLinks:   navFromConfig(cfg.Nav),
		FooterText: cfg.Footer,
		Theme:      bookThemeFrom(fs, cfg.Theme),
		MountPath:  cfg.Book.Mount,
		NoRootTOC:  hybrid && cfg.Book.TOC == nil,
		Include:    include,
		Exclude:    exclude,
	}
	slog.Info("building site with mdbind", "input", bcfg.Input, "output", bcfg.Output)
	if hybrid {
		// The ssg half owns the site's asset order. Reuse the plan it reported
		// instead of repeating the list here, so the book pages and the landing
		// page can never load stylesheets in different orders
		// (AGENTS.md § No Hardcoding).
		//
		// mdbind links its own stylesheet last by itself, so the site's copy of
		// that entry is dropped to avoid a duplicate <link>.
		bcfg.ExtraCSS = versionedAssetURLs(withoutAsset(planCSS, book.StylesheetName), cfg.Version)
		bcfg.ExtraJS = versionedAssetURLs(planJS, cfg.Version)
	}
	created, err := book.Build(bcfg)
	if err != nil {
		return fail(err)
	}
	for _, path := range created {
		fmt.Println("created " + path)
	}
	return kern.ExitCodeSuccess
}

// warnAutoAssetErrors surfaces asset-layer configuration problems (FRK-AS-074).
// The build keeps going with the default order, but a typo such as
// `auto_assets.order: {theme.css: {layer: last-milee}}` must not be silently
// ignored — it would reorder assets invisibly, which is exactly the class of
// drift the shared plan exists to prevent.
func warnAutoAssetErrors(site *ssg.Site) {
	for _, err := range site.AutoAssetErrors() {
		slog.Warn("kiw build: auto_assets config ignored", "err", err)
	}
}

// versionedAssetURLs appends the product version as a cache-busting query,
// matching what the ssg half injects so both halves refresh together. The rule
// itself lives in framework/web/ssg (VersionedURLs) — this only forwards to it,
// so the devtool and the generator cannot disagree on the query format.
func versionedAssetURLs(urls []string, version string) []string {
	return ssg.VersionedURLs(urls, version)
}

// withoutAsset drops the entry naming asset from an asset plan. Plan URLs are
// "/assets/<name>", so the base name is compared.
func withoutAsset(urls []string, asset string) []string {
	base := path.Base(strings.TrimPrefix(asset, "/"))
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if path.Base(u) == base {
			continue
		}
		out = append(out, u)
	}
	return out
}

// buildWASM compiles the project's WASM entry point (KWF-T4X9P FRK-WASM-002).
func buildWASM(root string, cfg *config.Config, fs *flag.FlagSet) kern.ExitCode {
	entry := firstNonEmpty(cfg.Wasm.Entry, "./wasm")
	outDir := joinRoot(root, firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output), config.DefaultOutput)
	name := firstNonEmpty(cfg.Wasm.Name, "runtime")

	slog.Info("building WASM module", "entry", entry, "output", outDir)
	artifacts, err := build.BuildWASM(build.Config{
		Entry:  entry,
		OutDir: outDir,
		Name:   name,
	})
	if err != nil {
		return fail(err)
	}
	fp := build.Fingerprint(artifacts.Digest)
	slog.Info("WASM build complete", "wasm", filepath.Base(artifacts.WASM), "js", filepath.Base(artifacts.JS), "fingerprint", fp)
	fmt.Printf("created %s (fingerprint: %s)\n", artifacts.WASM, fp)
	return kern.ExitCodeSuccess
}

// globList resolves a repeatable/comma-separated glob flag over its
// krewire.yaml fallback. Unset everywhere returns nil (mdbind defaults);
// an explicitly empty value disables that filter side.
func globList(fs *flag.FlagSet, name string, fallback []string) []string {
	raw := flagValue(fs, name)
	if raw == "" {
		return fallback
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// themeFrom resolves the theme from the --theme flag and krewire.yaml,
// defaulting to auto. A mode of off disables the switcher.
func themeFrom(fs *flag.FlagSet, cfg *config.Theme) *ui.Theme {
	mode := strings.ToLower(strings.TrimSpace(flagValue(fs, "theme")))
	if mode == "" && cfg != nil {
		mode = strings.ToLower(strings.TrimSpace(cfg.Default))
	}
	if mode == "" {
		mode = "auto"
	}
	switch mode {
	case "off", "none", "disabled":
		return nil
	}
	t := &ui.Theme{Default: mode}
	if cfg != nil {
		t.Light = cfg.Light.UI(ui.DefaultLightPalette)
		t.Dark = cfg.Dark.UI(ui.DefaultDarkPalette)
	}
	return t
}

// bookThemeFrom resolves the mdbind theme from the same flag/config as
// themeFrom but returns the book's local Theme type so mdbind can stay
// framework-free. The two themes stay in sync so a project can depend on
// both framework and mdbind and keep a single krewire.yaml.
func bookThemeFrom(fs *flag.FlagSet, cfg *config.Theme) *book.Theme {
	mode := strings.ToLower(strings.TrimSpace(flagValue(fs, "theme")))
	if mode == "" && cfg != nil {
		mode = strings.ToLower(strings.TrimSpace(cfg.Default))
	}
	if mode == "" {
		mode = "auto"
	}
	switch mode {
	case "off", "none", "disabled":
		return nil
	}
	t := &book.Theme{Default: mode}
	if cfg != nil {
		t.Light = bookPalette(cfg.Light, book.DefaultLightPalette)
		t.Dark = bookPalette(cfg.Dark, book.DefaultDarkPalette)
	}
	return t
}

func bookPalette(p config.Palette, defaults book.Palette) book.Palette {
	out := defaults
	if p.Base1 != "" {
		out.Base1 = book.Color(p.Base1)
	}
	if p.Base1Content != "" {
		out.Base1Content = book.Color(p.Base1Content)
	}
	if p.Base2 != "" {
		out.Base2 = book.Color(p.Base2)
	}
	if p.Base2Content != "" {
		out.Base2Content = book.Color(p.Base2Content)
	}
	if p.Base3 != "" {
		out.Base3 = book.Color(p.Base3)
	}
	if p.Base3Content != "" {
		out.Base3Content = book.Color(p.Base3Content)
	}
	if p.Primary != "" {
		out.Primary = book.Color(p.Primary)
	}
	if p.PrimaryContent != "" {
		out.PrimaryContent = book.Color(p.PrimaryContent)
	}
	if p.Secondary != "" {
		out.Secondary = book.Color(p.Secondary)
	}
	if p.SecondaryContent != "" {
		out.SecondaryContent = book.Color(p.SecondaryContent)
	}
	if p.Accent != "" {
		out.Accent = book.Color(p.Accent)
	}
	if p.AccentContent != "" {
		out.AccentContent = book.Color(p.AccentContent)
	}
	if p.Ghost != "" {
		out.Ghost = book.Color(p.Ghost)
	}
	if p.GhostContent != "" {
		out.GhostContent = book.Color(p.GhostContent)
	}
	if p.Neutral != "" {
		out.Neutral = book.Color(p.Neutral)
	}
	if p.NeutralContent != "" {
		out.NeutralContent = book.Color(p.NeutralContent)
	}
	if p.Success != "" {
		out.Success = book.Color(p.Success)
	}
	if p.SuccessContent != "" {
		out.SuccessContent = book.Color(p.SuccessContent)
	}
	if p.Info != "" {
		out.Info = book.Color(p.Info)
	}
	if p.InfoContent != "" {
		out.InfoContent = book.Color(p.InfoContent)
	}
	if p.Warning != "" {
		out.Warning = book.Color(p.Warning)
	}
	if p.WarningContent != "" {
		out.WarningContent = book.Color(p.WarningContent)
	}
	if p.Error != "" {
		out.Error = book.Color(p.Error)
	}
	if p.ErrorContent != "" {
		out.ErrorContent = book.Color(p.ErrorContent)
	}
	return out
}

// navFromConfig converts krewire.yaml nav links into book links.
func navFromConfig(links []config.Link) []book.Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]book.Link, 0, len(links))
	for _, l := range links {
		out = append(out, book.Link{Text: l.Text, URL: l.URL})
	}
	return out
}

// buildAppBinary compiles the project Go binary into .krewire/build/bin/<name> (KWN-BUILD-001).
func buildAppBinary(root string, cfg *config.Config, fs *flag.FlagSet, kind shape.Kind) kern.ExitCode {
	name := firstNonEmpty(cfg.Name, moduleName(root), "app")
	outDir := joinRoot(root, firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output), config.DefaultOutput)
	binDir := filepath.Join(outDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fail(err)
	}
	binPath := filepath.Join(binDir, name)

	entry := "."
	if !hasFile(root, "main.go") {
		found := false
		if entries, err := os.ReadDir(filepath.Join(root, "cmd")); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					entry = "./cmd/" + e.Name()
					found = true
					break
				} else if e.Name() == "main.go" {
					entry = "./cmd"
					found = true
					break
				}
			}
		}
		if !found && !hasGoFiles(root) {
			return fail(fmt.Errorf("build %s: no main package found in root or cmd/", kind))
		}
	}

	slog.Info("building binary", "kind", kind, "entry", entry, "output", binPath)
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags="+buildinfo.LDFlags(true), "-o", binPath, entry)
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fail(fmt.Errorf("build %s: %w", kind, err))
	}
	fmt.Println("created " + binPath)
	return kern.ExitCodeSuccess
}

// buildInfra compiles and verifies cloud infrastructure topology into plan.json (KWN-BUILD-001, KWF-B7N3D).
func buildInfra(root string, cfg *config.Config, fs *flag.FlagSet) kern.ExitCode {
	providerName := "krewire-cloud"
	region := "asia-southeast1"
	if cfg.Infra != nil {
		if cfg.Infra.Provider != "" {
			providerName = cfg.Infra.Provider
		}
		if cfg.Infra.Region != "" {
			region = cfg.Infra.Region
		}
	}
	env := firstNonEmpty(flagValue(fs, "env"), cfg.Env, "production")

	outDir := joinRoot(root, firstNonEmpty(flagValue(fs, "output"), flagValue(fs, "o"), cfg.Output), config.DefaultOutput)
	infraDir := filepath.Join(outDir, "infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		return fail(err)
	}
	planFile := filepath.Join(infraDir, "plan.json")

	// 1. Gather desired resources
	var resources []provider.Resource
	if cfg.Infra != nil && len(cfg.Infra.Resources) > 0 {
		resources = append(resources, cfg.Infra.Resources...)
	} else {
		// Synthesize declarative infrastructure resources from project definition
		appName := firstNonEmpty(cfg.Name, moduleName(root), "app")
		resources = append(resources, provider.Resource{
			Kind: schema.KindCompute,
			ID:   appName,
			Properties: map[string]any{
				"runtime":  "fargate",
				"replicas": 2,
			},
		})
		if len(cfg.Services) > 0 {
			for _, svc := range cfg.Services {
				if svc.Name == "" {
					continue
				}
				resources = append(resources, provider.Resource{
					Kind: schema.KindCompute,
					ID:   svc.Name,
					Properties: map[string]any{
						"port": svc.Port,
					},
					DependsOn: []string{appName},
				})
			}
		}
		if cfg.Worker != nil {
			resources = append(resources, provider.Resource{
				Kind: schema.KindCompute,
				ID:   "worker",
				Properties: map[string]any{
					"concurrency": cfg.Worker.Concurrency,
					"queues":      cfg.Worker.Queues,
				},
				DependsOn: []string{appName},
			})
		}
	}

	// 2. Validate topology (no duplicate IDs, all dependsOn exist)
	idMap := make(map[string]bool)
	for _, r := range resources {
		if r.ID == "" {
			return fail(fmt.Errorf("infra topology error: resource missing id (kind: %s)", r.Kind))
		}
		if idMap[r.ID] {
			return fail(fmt.Errorf("infra topology error: duplicate resource id %q", r.ID))
		}
		idMap[r.ID] = true
	}
	for _, r := range resources {
		for _, dep := range r.DependsOn {
			if !idMap[dep] {
				return fail(fmt.Errorf("infra topology error: resource %q depends on non-existent resource %q", r.ID, dep))
			}
		}
	}

	// 3. Assemble execution plan
	plan := provider.Plan{
		Actions: make([]provider.Action, len(resources)),
	}
	for i, r := range resources {
		plan.Actions[i] = provider.Action{
			Op:       provider.OpCreate,
			Resource: r,
			Reason:   "declared in infrastructure blueprint",
		}
	}

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fail(fmt.Errorf("marshal infra plan: %w", err))
	}
	if err := os.WriteFile(planFile, data, 0o644); err != nil {
		return fail(fmt.Errorf("write infra plan: %w", err))
	}
	fmt.Println("created " + planFile)

	if fs != nil && (flagValue(fs, "plan") == "true" || flagProvided(fs, "plan")) {
		fmt.Printf("Infrastructure Plan (%s, env=%s, region=%s):\n", providerName, env, region)
		for _, act := range plan.Actions {
			fmt.Printf("  + [%s] %s (%s)\n", act.Op, act.Resource.Kind, act.Resource.ID)
			if len(act.Resource.DependsOn) > 0 {
				fmt.Printf("      depends on: %s\n", strings.Join(act.Resource.DependsOn, ", "))
			}
		}
		fmt.Printf("Plan: %d to add, 0 to change, 0 to destroy.\n", len(plan.Actions))
	}

	return kern.ExitCodeSuccess
}

// hasGoFiles reports whether dir contains non-test Go source files.
func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}
