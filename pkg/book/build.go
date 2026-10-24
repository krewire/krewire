package book

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultInput  = "."
	defaultOutput = ".krewire/build"
)

// Config configures a build.
// StylesheetName is the output path of the stylesheet this package emits for
// every book page. A host that renders a book into a directory it shares with a
// site (see Config.ExtraCSS) must not also link this file from the site's own
// asset plan, or the page would load it twice.
const StylesheetName = "assets/krewire.css"

type Config struct {
	// Title is the book title.
	Title string
	// Author is the book author.
	Author string
	// Input is the content directory.
	Input string
	// Output is the destination directory for the exported site.
	Output string
	// BasePath is the URL base the site will be served under, e.g. "/guide/".
	// Generated links and asset references are prefixed accordingly. Defaults
	// to "/".
	BasePath string
	// MountPath places the whole book under a sub-path of the output and the
	// URL space, e.g. "/docs/" exports docs/index.html plus docs/<slug>.html
	// and prefixes every link with /docs/. Unlike BasePath it relocates the
	// exported files, letting a book co-exist with an ssg site in one output
	// (progressive enhancement). Empty mounts at the root.
	MountPath string
	// NoRootTOC suppresses the generated root table-of-contents page so the
	// book does not write index.html at its mount — used when an ssg landing
	// page already owns "/" in a shared output. Chapter pages are unaffected.
	NoRootTOC bool
	// Include holds glob patterns (slash-separated, ** crosses segments)
	// selecting which content files build; nil defaults to "**/*.md".
	// An empty non-nil slice disables inclusion (builds nothing).
	Include []string
	// Exclude holds glob patterns removed after include filtering; nil
	// defaults to README/readme developer notes ("**/README.md",
	// "**/readme.md"). An empty non-nil slice excludes nothing.
	Exclude []string
	// NavLinks are optional extra links rendered in the navbar.
	NavLinks []Link
	// HomeURL is the optional URL the brand logo links to; defaults to "/"
	// when MountPath is set, or BasePath otherwise.
	HomeURL string
	// FooterText is optional text shown in the footer; when empty the footer
	// shows the author copyright instead.
	FooterText string
	// Version is the product version rendered in the page credit line; empty
	// omits it. Sourced from krewire.yaml `version:` by kiw build.
	Version string
	// Theme, when non-nil, enables the light/dark theme switcher on every page.
	Theme *Theme
	// ExtraCSS are additional stylesheet URLs linked on every
	// page, in the given order. It lets a book rendered into a shared output
	// adopt the parent site's stylesheets (e.g. tailwind.css, theme.css) so a
	// hybrid site looks identical whether the visitor lands on a landing page
	// or a docs page. Entries are used verbatim, so callers pass a full URL.
	ExtraCSS []string
	// ExtraJS are additional script URLs loaded at the end of <body> on every
	// page, in the given order. Used verbatim like ExtraCSS.
	ExtraJS []string
}

// Build renders the content in cfg.Input into a static website in
// cfg.Output and returns the list of created paths, sorted for determinism.
func Build(cfg Config) ([]string, error) {
	if cfg.Input == "" {
		cfg.Input = defaultInput
	}
	if cfg.Output == "" {
		cfg.Output = defaultOutput
	}
	mount := normalizeBase(cfg.MountPath)
	base := cfg.BasePath
	if mount != "/" {
		if base == "" || base == "/" {
			base = mount
		}
	}
	b, err := loadWithRules(cfg.Input, cfg.Title, cfg.Author, base, cfg.Include, cfg.Exclude)
	if err != nil {
		return nil, err
	}
	b.mount = strings.TrimSuffix(mount, "/")
	homeURL := cfg.HomeURL
	if homeURL == "" {
		if b.mount != "" {
			homeURL = "/"
		} else {
			homeURL = b.base
		}
	}
	b.homeURL = homeURL
	b.noRootTOC = cfg.NoRootTOC
	b.extraCSS = append([]string(nil), cfg.ExtraCSS...)
	b.extraJS = append([]string(nil), cfg.ExtraJS...)
	b.version = cfg.Version
	b.WithChrome(cfg.NavLinks, cfg.FooterText)
	if cfg.Theme != nil {
		b.WithTheme(cfg.Theme)
	}
	pages, err := b.Pages()
	if err != nil {
		return nil, err
	}
	if err := exportPages(cfg.Output, pages, b.mount); err != nil {
		return nil, err
	}
	if len(cfg.ExtraCSS) == 0 {
		if err := exportMountedAssets(cfg.Output, b.mount, Assets()); err != nil {
			return nil, err
		}
	}
	searchIdx := b.BuildSearchIndex()
	navManifest := b.BuildNavManifest()
	if err := exportManifests(cfg.Output, b.mount, searchIdx, navManifest); err != nil {
		return nil, err
	}
	return b.createdPaths(cfg.Output), nil
}

func exportAssets(outDir string, assets map[string]string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(assets))
	for name := range assets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		out := filepath.Join(outDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, []byte(assets[name]), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// exportMountedAssets writes assets under outDir/mount when the book is
// mounted at a sub-path so links like /docs/assets/krewire.css resolve.
func exportMountedAssets(outDir, mount string, assets map[string]string) error {
	if mount == "" {
		return exportAssets(outDir, assets)
	}
	return exportAssets(filepath.Join(outDir, filepath.FromSlash(mount)), assets)
}

// WithChrome sets the book's navigation chrome: optional navbar links and
// footer text. Returns the Book for chaining.
func (b *Book) WithChrome(nav []Link, footer string) *Book {
	b.navLinks = append([]Link(nil), nav...)
	b.footerText = footer
	return b
}

// WithTheme enables the light/dark theme switcher. Returns the Book for
// chaining.
func (b *Book) WithTheme(t *Theme) *Book {
	if t != nil {
		b.theme = t
	}
	return b
}
