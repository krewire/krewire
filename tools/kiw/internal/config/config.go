// Package config reads project configuration from krewire.yaml so that `kiw`
// commands can drive any project kind without a project-specific cmd.
// A krewire.yaml with an `ssg:` key triggers SSG mode; one with a manuscript/
// directory triggers book mode. All top-level fields are shared across modes.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/krewire/krewire/pkg/cloud/infra/provider"
	"github.com/krewire/krewire/pkg/ui"
	"github.com/krewire/krewire/pkg/web/ssg"
	"gopkg.in/yaml.v3"
)

const DefaultOutput = ".krewire/build"

// DefaultInput is the default content directory consumed by the book
// (mdbind) build, mirroring the ssg content layout.
const DefaultInput = "content"

// Config is the krewire.yaml schema consumed by the build command.
type Config struct {
	// Name holds the project name when declared at top level.
	Name string `yaml:"name"`
	// Project pins the project kind when discovery is ambiguous. Empty means
	// auto-detection (app > site > book precedence).
	Project Project `yaml:"project"`
	// Env pins the target environment: "local", "production", or "testing".
	// Empty resolves to local; KIW_ENV and --env override (KWL-K4T7W).
	Env string `yaml:"env"`
	// Debug enables debug mode. Defaults to false; KIW_DEBUG and --debug
	// override.
	Debug bool `yaml:"debug"`
	// Title is the site title.
	Title string `yaml:"title"`
	// Author is the site author.
	Author string `yaml:"author"`
	// Base is the URL base the site is served under, e.g. "/docs/".
	Base string `yaml:"base"`
	// Version is the product version injected into page data (.Version) for
	// both ssg and book pipelines, so web content never hardcodes it.
	Version string `yaml:"version"`
	// Input is the content directory, relative to the project root.
	// Defaults to "content" when empty; may be overridden by krewire.yaml
	// or the --input flag.
	Input string `yaml:"input"`
	// Output is the output directory, relative to the project root.
	// Defaults to .krewire/build when empty; may be overridden by
	// krewire.yaml or the --output/-o flag.
	Output string `yaml:"output"`
	// Build tunes the build pipeline: include/exclude content path globs.
	Build BuildConfig `yaml:"build"`
	// Nav holds optional navbar links.
	Nav []Link `yaml:"nav"`
	// Footer is optional footer text.
	Footer string `yaml:"footer"`
	// Theme configures the light/dark theme switcher.
	Theme *Theme `yaml:"theme"`
	// Book holds book-specific settings for the manuscript/ (mdbind) shape.
	Book BookConfig `yaml:"book"`
	// SSG holds the declarative SSG config. When non-nil, kiw build uses
	// the web/ssg generator instead of mdbind. Top-level fields (Title,
	// Author, Base, Output, Theme) are injected into the SSG config.
	SSG *SSGConfig `yaml:"ssg"`
	// AutoAssets configures asset injection and bundling at top-level.
	AutoAssets *ssg.AutoAssetConfig `yaml:"auto_assets"`
	// Wasm holds WASM build configuration (KWF-T4X9P).
	Wasm WasmConfig `yaml:"wasm"`
	// Runtime holds multi-runtime coordination settings (KWN-MRCN).
	Runtime RuntimeConfig `yaml:"runtime"`
	// Dev holds development settings (e.g. dev server port).
	Dev DevConfig `yaml:"dev"`
	// Server holds server/service settings (e.g. host and port).
	Server ServerConfig `yaml:"server"`
	// Worker holds background worker settings (KWF-L5H2F).
	Worker *WorkerConfig `yaml:"worker"`
	// Service holds standalone microservice configuration (KWF-L5H2F).
	Service *ServiceConfig `yaml:"service"`
	// Services holds multiple declared services in the project (FRK-CONF-032).
	Services []ServiceConfig `yaml:"services"`
	// Infra holds cloud infrastructure configuration (KWF-B7N3D).
	Infra *InfraConfig `yaml:"infra"`
	// Gateway holds API gateway reverse proxy configuration (KWF-L5H2F).
	Gateway *GatewayConfig `yaml:"gateway"`
	// Scripts holds named shell commands runnable via `kiw run <task>`.
	// Keys are task names, values are shell commands executed with `sh -c`
	// (Unix) or `cmd /C` (Windows). Optional, zero-value nil means no tasks.
	Scripts map[string]string `yaml:"scripts"`
	// I18n configures internationalization for the site/book build.
	// When set, kiw build iterates over supported locales and generates
	// localized output trees (e.g. .krewire/build/en/docs/, .../id/docs/).
	I18n *ssg.I18nConfig `yaml:"i18n"`
}

// GatewayConfig holds API gateway configuration under the `gateway:` key (KWF-L5H2F).
type GatewayConfig struct {
	Port   int            `yaml:"port"`
	Host   string         `yaml:"host"`
	Routes []GatewayRoute `yaml:"routes"`
}

// GatewayRoute maps a path to a service or target URL.
type GatewayRoute struct {
	Path        string   `yaml:"path"`
	Method      string   `yaml:"method"`
	Service     string   `yaml:"service"`
	TargetURL   string   `yaml:"target_url"`
	Prefix      bool     `yaml:"prefix"`
	StripPrefix bool     `yaml:"strip_prefix"`
	Middleware  []string `yaml:"middleware"`
	RateLimit   int      `yaml:"rate_limit"`
	Timeout     string   `yaml:"timeout"`
}

// RuntimeConfig holds multi-runtime coordination settings under the `runtime:` key (KWN-MRCN).
type RuntimeConfig struct {
	// Namespace prefixes Krewire-specific environment variables (e.g. "go", "krewire").
	Namespace string `yaml:"namespace"`
	// EnvFile points to the .env file path (default: ".env").
	EnvFile string `yaml:"env_file"`
	// Port declares the default runtime port.
	Port int `yaml:"port"`
}

// DevConfig holds development server configuration under the `dev:` key.
type DevConfig struct {
	Port int `yaml:"port"`
}

// ServerConfig holds service/server settings under the `server:` key.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ServiceConfig holds service-specific settings under the `service:` or `services:` key (FRK-CONF-032).
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

// WorkerConfig holds queue and background processing settings under the `worker:` key (KWF-L5H2F).
type WorkerConfig struct {
	Concurrency int      `yaml:"concurrency"`
	Queues      []string `yaml:"queues"`
}

// InfraConfig holds cloud infrastructure settings under the `infra:` key (KWF-B7N3D).
type InfraConfig struct {
	Provider  string              `yaml:"provider"`
	Region    string              `yaml:"region"`
	State     string              `yaml:"state"`
	Resources []provider.Resource `yaml:"resources"`
}

// BuildConfig tunes the build pipeline under the `build:` key.
type BuildConfig struct {
	// Base is the URL base the site is served under, e.g. "/docs/".
	// Legacy top-level `base:` stays supported; nested `build.base`
	// wins when both are set. kiw --base wins over both.
	Base string `yaml:"base"`
	// Include holds glob patterns selecting which content files build
	// (slash-separated, ** crosses segments). Unset (nil) uses the mdbind
	// default "**/*.md"; an empty list disables the include filter.
	Include []string `yaml:"include"`
	// Exclude holds glob patterns removed after include filtering. Unset
	// (nil) uses the mdbind defaults excluding README/readme developer
	// notes; an empty list excludes nothing.
	Exclude []string `yaml:"exclude"`
	// DependsOn lists shell commands to run in order before building
	// (e.g. ["npm run build", "composer install --no-dev"]).
	DependsOn []string `yaml:"depends_on"`
	// Outputs maps runtime names to output directories (e.g. nodejs: "dist", laravel: "public/build").
	Outputs map[string]string `yaml:"outputs"`
}

// BookConfig holds book-specific settings under the `book:` key.
type BookConfig struct {
	// Mount places the whole book under a sub-path of the output and URL
	// space, e.g. "/docs/" — letting a manuscript co-exist with an ssg site
	// in one .krewire/build (progressive enhancement). Empty mounts at root.
	Mount string `yaml:"mount"`
	// TOC controls the generated root table-of-contents page. In a hybrid
	// project it defaults to false (suppressed so the ssg landing page owns
	// "/"); set `toc: true` to force it on. Standalone books always emit it.
	TOC *bool `yaml:"toc"`
}

// WasmConfig holds WASM build configuration under the `wasm:` key (KWF-T4X9P).
type WasmConfig struct {
	// Entry is the main package path to build (default "./wasm").
	Entry string `yaml:"entry"`
	// Name is the output module name (default "runtime").
	Name string `yaml:"name"`
}

// IsSSG reports whether this config targets the declarative SSG mode.
func (c *Config) IsSSG() bool {
	return c != nil && c.SSG != nil
}

// Kind returns the pinned project kind from config, or "" when unset.
func (c *Config) Kind() string {
	if c == nil {
		return ""
	}
	return c.Project.Kind
}

// DeclaredPort returns the port declared in krewire.yaml across runtime.port,
// dev.port, server.port, or services (FRK-CONF-003, FRK-CONF-030).
func (c *Config) DeclaredPort() int {
	if c == nil {
		return 0
	}
	if c.Runtime.Port > 0 {
		return c.Runtime.Port
	}
	if c.Dev.Port > 0 {
		return c.Dev.Port
	}
	if c.Server.Port > 0 {
		return c.Server.Port
	}
	if len(c.Services) > 0 && c.Services[0].Port > 0 {
		return c.Services[0].Port
	}
	return 0
}

// EffectiveEnvFile returns the configured env file path or ".env" by default (FRK-CONF-012).
func (c *Config) EffectiveEnvFile() string {
	if c != nil && c.Runtime.EnvFile != "" {
		return c.Runtime.EnvFile
	}
	return ".env"
}

// Project holds per-project settings in krewire.yaml.
type Project struct {
	// Name is the project name.
	Name string `yaml:"name"`
	// Kind pins the project kind: "app", "cli", "site", or "book". Empty
	// lets the CLI detect it from marker files.
	Kind string `yaml:"kind"`
	// Version is the product version declared next to the kind, the way a
	// project writes it:
	//
	//	project:
	//	  kind: site
	//	  version: v0.1.0
	//
	// Load folds it into Config.Version when no top-level `version:` is set,
	// so the version has one effective value regardless of where it is declared.
	Version string `yaml:"version"`
	// Dirs overrides the canonical directory locations (FRK-STR-010).
	Dirs Dirs `yaml:"dirs"`
}

// Dirs holds custom directory locations for the project layout.
type Dirs struct {
	Web      string `yaml:"web"`
	Public   string `yaml:"public"`
	Internal string `yaml:"internal"`
	Cmd      string `yaml:"cmd"`
}

// SSGConfig holds SSG-specific fields that live under the `ssg:` key in
// krewire.yaml. It mirrors the fields of ssg.Config that are not shared with
// the top-level Config.
type SSGConfig struct {
	// Description is injected into every page's template data.
	Description string `yaml:"description"`
	// Layouts are the site's named layouts.
	Layouts []SSGLayoutConfig `yaml:"layouts"`
	// Components are the site's named components.
	Components []SSGComponentConfig `yaml:"components"`
	// Pages are the site's output pages.
	Pages []SSGPageConfig `yaml:"pages"`
	// Assets maps output paths to file contents.
	Assets map[string]string `yaml:"assets"`
	// AutoAssets carries the ssg.auto_assets block into the ssg.Config.
	AutoAssets *ssg.AutoAssetConfig `yaml:"auto_assets"`
}

// SSGLayoutConfig is a layout entry under ssg.layouts.
type SSGLayoutConfig struct {
	Name  string `yaml:"name"`
	Body  string `yaml:"body"`
	Style string `yaml:"style"`
	// UI, when set, builds the layout from a reusable framework/ui shell
	// instead of Body/Style.
	UI *ui.LayoutConfig `yaml:"ui"`
}

// SSGComponentConfig is a component entry under ssg.components.
type SSGComponentConfig struct {
	Name  string `yaml:"name"`
	Body  string `yaml:"body"`
	Style string `yaml:"style"`
}

// SSGPageConfig is a page entry under ssg.pages.
type SSGPageConfig struct {
	Path   string         `yaml:"path"`
	Title  string         `yaml:"title"`
	Layout string         `yaml:"layout"`
	Root   string         `yaml:"root"`
	Data   map[string]any `yaml:"data"`
}

// ToSSGConfig converts this Config into an ssg.Config, merging top-level
// fields (Title, Base, Output, Theme) with the SSG-specific fields under ssg:.
// Callers must check IsSSG() before calling this method.
func (c *Config) ToSSGConfig() *ssg.Config {
	s := c.SSG
	autoAssets := s.AutoAssets
	if autoAssets == nil {
		autoAssets = c.AutoAssets
	}
	cfg := &ssg.Config{
		Title:       c.Title,
		Description: s.Description,
		Base:        c.Base,
		Version:     c.Version,
		Output:      c.Output,
		Assets:      s.Assets,
		AutoAssets:  autoAssets,
	}
	if c.Theme != nil {
		cfg.Theme = &ssg.ThemeConfig{
			Default: c.Theme.Default,
		}
	}
	for _, l := range s.Layouts {
		cfg.Layouts = append(cfg.Layouts, ssg.LayoutConfig{Name: l.Name, Body: l.Body, Style: l.Style, UI: l.UI})
	}
	for _, comp := range s.Components {
		cfg.Components = append(cfg.Components, ssg.ComponentConfig{Name: comp.Name, Body: comp.Body, Style: comp.Style})
	}
	for _, p := range s.Pages {
		cfg.Pages = append(cfg.Pages, ssg.PageConfig{
			Path:   p.Path,
			Title:  p.Title,
			Layout: p.Layout,
			Root:   p.Root,
			Data:   p.Data,
		})
	}
	return cfg
}

// Link is a named navigation target shown in the navbar.
type Link struct {
	Text string `yaml:"text"`
	URL  string `yaml:"url"`
}

// Theme configures the theme switcher and per-mode palette overrides.
type Theme struct {
	// Default is the initial mode: auto, light, or dark.
	Default string `yaml:"default"`
	// Light and Dark hold palette token overrides for each mode.
	Light Palette `yaml:"light"`
	Dark  Palette `yaml:"dark"`
}

// Palette holds color token overrides. Empty tokens fall back to the ui
// defaults.
type Palette struct {
	Base1            string `yaml:"base-1"`
	Base1Content     string `yaml:"base-1-content"`
	Base2            string `yaml:"base-2"`
	Base2Content     string `yaml:"base-2-content"`
	Base3            string `yaml:"base-3"`
	Base3Content     string `yaml:"base-3-content"`
	Primary          string `yaml:"primary"`
	PrimaryContent   string `yaml:"primary-content"`
	Secondary        string `yaml:"secondary"`
	SecondaryContent string `yaml:"secondary-content"`
	Accent           string `yaml:"accent"`
	AccentContent    string `yaml:"accent-content"`
	Ghost            string `yaml:"ghost"`
	GhostContent     string `yaml:"ghost-content"`
	Neutral          string `yaml:"neutral"`
	NeutralContent   string `yaml:"neutral-content"`
	Success          string `yaml:"success"`
	SuccessContent   string `yaml:"success-content"`
	Info             string `yaml:"info"`
	InfoContent      string `yaml:"info-content"`
	Warning          string `yaml:"warning"`
	WarningContent   string `yaml:"warning-content"`
	Error            string `yaml:"error"`
	ErrorContent     string `yaml:"error-content"`
}

// Load reads krewire.yaml from dir. A missing file yields a zero Config without
// an error.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, "krewire.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	// Fold project.name or title into top-level Name
	if c.Name == "" {
		c.Name = c.Project.Name
	}
	if c.Name == "" {
		c.Name = c.Title
	}
	// A project declares its version either at the top level or next to the
	// kind. Fold the latter into the single effective value so both pipelines
	// (ssg page data and the mdbind asset cache-busting query) see the same
	// version instead of one silently losing it.
	if c.Version == "" {
		c.Version = c.Project.Version
	}
	// The URL base lives under `build:` in current krewire.yaml files
	// (e.g. krewire.com `build.base: "/"`, portal `build.base:
	// "/contribute/"). Fold the nested form into the single effective
	// value so tooling never silently ignores what the repo declares.
	// Precedence: kiw --base flag > build.base > legacy top-level base:.
	if c.Build.Base != "" {
		c.Base = c.Build.Base
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// UI merges the palette overrides into the given default palette.
func (p *Palette) UI(defaults ui.Palette) ui.Palette {
	if p == nil {
		return defaults
	}
	if p.Base1 != "" {
		defaults.Base1 = ui.Color(p.Base1)
	}
	if p.Base1Content != "" {
		defaults.Base1Content = ui.Color(p.Base1Content)
	}
	if p.Base2 != "" {
		defaults.Base2 = ui.Color(p.Base2)
	}
	if p.Base2Content != "" {
		defaults.Base2Content = ui.Color(p.Base2Content)
	}
	if p.Base3 != "" {
		defaults.Base3 = ui.Color(p.Base3)
	}
	if p.Base3Content != "" {
		defaults.Base3Content = ui.Color(p.Base3Content)
	}
	if p.Primary != "" {
		defaults.Primary = ui.Color(p.Primary)
	}
	if p.PrimaryContent != "" {
		defaults.PrimaryContent = ui.Color(p.PrimaryContent)
	}
	if p.Secondary != "" {
		defaults.Secondary = ui.Color(p.Secondary)
	}
	if p.SecondaryContent != "" {
		defaults.SecondaryContent = ui.Color(p.SecondaryContent)
	}
	if p.Accent != "" {
		defaults.Accent = ui.Color(p.Accent)
	}
	if p.AccentContent != "" {
		defaults.AccentContent = ui.Color(p.AccentContent)
	}
	if p.Ghost != "" {
		defaults.Ghost = ui.Color(p.Ghost)
	}
	if p.GhostContent != "" {
		defaults.GhostContent = ui.Color(p.GhostContent)
	}
	if p.Neutral != "" {
		defaults.Neutral = ui.Color(p.Neutral)
	}
	if p.NeutralContent != "" {
		defaults.NeutralContent = ui.Color(p.NeutralContent)
	}
	if p.Success != "" {
		defaults.Success = ui.Color(p.Success)
	}
	if p.SuccessContent != "" {
		defaults.SuccessContent = ui.Color(p.SuccessContent)
	}
	if p.Info != "" {
		defaults.Info = ui.Color(p.Info)
	}
	if p.InfoContent != "" {
		defaults.InfoContent = ui.Color(p.InfoContent)
	}
	if p.Warning != "" {
		defaults.Warning = ui.Color(p.Warning)
	}
	if p.WarningContent != "" {
		defaults.WarningContent = ui.Color(p.WarningContent)
	}
	if p.Error != "" {
		defaults.Error = ui.Color(p.Error)
	}
	if p.ErrorContent != "" {
		defaults.ErrorContent = ui.Color(p.ErrorContent)
	}
	return defaults
}
