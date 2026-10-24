package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Default bundle artifact paths.
const (
	DefaultCSSBundlePath = "assets/krewire.css"
	DefaultJSBundlePath  = "assets/krewire.js"
)

var (
	charsetRe = regexp.MustCompile(`(?i)^\s*@charset\s+["'][^"']+["'];?\s*`)
	importRe  = regexp.MustCompile(`(?m)^\s*@import\s+(?:url\([^)]+\)|["'][^"']+["'])[^;]*;?\s*`)
)

// BundleItem represents an asset file with its path and content to be bundled.
type BundleItem struct {
	Path    string
	Content string
}

// BundlerOptions configures bundling behavior.
type BundlerOptions struct {
	Minify         bool
	SourceComments bool
	Banner         string
	CSSOutputName  string
	JSOutputName   string
}

// BundlerOption is a functional configuration option for Bundler.
type BundlerOption func(*BundlerOptions)

// WithMinify toggles basic whitespace/comment minification.
func WithMinify(minify bool) BundlerOption {
	return func(o *BundlerOptions) {
		o.Minify = minify
	}
}

// WithSourceComments toggles file header comments between bundled items.
func WithSourceComments(comments bool) BundlerOption {
	return func(o *BundlerOptions) {
		o.SourceComments = comments
	}
}

// WithBanner sets a top-of-file header banner for generated bundles.
func WithBanner(banner string) BundlerOption {
	return func(o *BundlerOptions) {
		o.Banner = banner
	}
}

// WithOutputNames customizes relative output paths for CSS and JS bundles.
func WithOutputNames(cssPath, jsPath string) BundlerOption {
	return func(o *BundlerOptions) {
		if cssPath != "" {
			o.CSSOutputName = cssPath
		}
		if jsPath != "" {
			o.JSOutputName = jsPath
		}
	}
}

// Bundler combines, optimizes, and fingerprints CSS and JS asset streams.
type Bundler struct {
	opts BundlerOptions
}

// NewBundler returns an initialized Bundler with applied options.
func NewBundler(opts ...BundlerOption) *Bundler {
	cfg := BundlerOptions{
		SourceComments: true,
		CSSOutputName:  DefaultCSSBundlePath,
		JSOutputName:   DefaultJSBundlePath,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Bundler{opts: cfg}
}

// BundleResult holds the rendered bundle strings and metadata.
type BundleResult struct {
	CSSContent     string
	JSContent      string
	CSSPath        string
	JSPath         string
	CSSHash        string
	JSHash         string
	CSSFingerprint string
	JSFingerprint  string
}

// BundleCSS concatenates CSS items, hoisting @charset and @import rules
// to the head of the file per W3C specification, and applying optional minification.
func (b *Bundler) BundleCSS(items []BundleItem) string {
	var charsets []string
	var imports []string
	var bodies []string

	for _, item := range items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		// Extract @charset
		if loc := charsetRe.FindStringIndex(content); loc != nil {
			cs := strings.TrimSpace(content[loc[0]:loc[1]])
			if len(charsets) == 0 {
				charsets = append(charsets, cs)
			}
			content = content[loc[1]:]
		}

		// Extract all @import statements
		content = importRe.ReplaceAllStringFunc(content, func(m string) string {
			imports = append(imports, strings.TrimSpace(m))
			return ""
		})

		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		var fileBody strings.Builder
		if b.opts.SourceComments && !b.opts.Minify {
			name := item.Path
			if name == "" {
				name = "stylesheet"
			}
			fileBody.WriteString("/* " + name + " */\n")
		}
		fileBody.WriteString(content)
		bodies = append(bodies, fileBody.String())
	}

	var out strings.Builder
	if b.opts.Banner != "" && !b.opts.Minify {
		out.WriteString("/* " + b.opts.Banner + " */\n\n")
	}

	// 1. Hoist single @charset if present
	for _, cs := range charsets {
		out.WriteString(cs + "\n")
	}
	if len(charsets) > 0 {
		out.WriteString("\n")
	}

	// 2. Hoist all @import statements
	for _, imp := range imports {
		out.WriteString(imp + "\n")
	}
	if len(imports) > 0 && len(bodies) > 0 {
		out.WriteString("\n")
	}

	// 3. Body rules in priority order
	for i, body := range bodies {
		if i > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(body)
	}

	res := out.String()
	if b.opts.Minify {
		res = MinifyCSS(res)
	}
	return res
}

// BundleJS concatenates JS items safely, guaranteeing semicolon isolation
// to prevent Automatic Semicolon Insertion (ASI) bugs between adjacent scripts.
func (b *Bundler) BundleJS(items []BundleItem) string {
	var bodies []string

	for _, item := range items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		var fileBody strings.Builder
		if b.opts.SourceComments && !b.opts.Minify {
			name := item.Path
			if name == "" {
				name = "script"
			}
			fileBody.WriteString("/* " + name + " */\n")
		}
		fileBody.WriteString(content)
		bodies = append(bodies, fileBody.String())
	}

	var out strings.Builder
	if b.opts.Banner != "" && !b.opts.Minify {
		out.WriteString("/* " + b.opts.Banner + " */\n\n")
	}

	for i, body := range bodies {
		if i > 0 {
			// Semicolon isolation between distinct scripts
			out.WriteString("\n\n;\n")
		}
		out.WriteString(body)
	}

	res := out.String()
	if b.opts.Minify {
		res = MinifyJS(res)
	}
	return res
}

// Bundle collects the given CSS and JS paths from a Store and produces bundled outputs.
func (b *Bundler) Bundle(store *Store, cssPaths, jsPaths []string) (*BundleResult, error) {
	var cssItems []BundleItem
	for _, p := range cssPaths {
		data, _, _, err := store.Open(p)
		if err != nil {
			return nil, fmt.Errorf("assets bundler: read css %s: %w", p, err)
		}
		cssItems = append(cssItems, BundleItem{Path: p, Content: string(data)})
	}

	var jsItems []BundleItem
	for _, p := range jsPaths {
		data, _, _, err := store.Open(p)
		if err != nil {
			return nil, fmt.Errorf("assets bundler: read js %s: %w", p, err)
		}
		jsItems = append(jsItems, BundleItem{Path: p, Content: string(data)})
	}

	cssContent := b.BundleCSS(cssItems)
	jsContent := b.BundleJS(jsItems)

	cssHash := hash8([]byte(cssContent))
	jsHash := hash8([]byte(jsContent))

	cssPath := b.opts.CSSOutputName
	jsPath := b.opts.JSOutputName

	return &BundleResult{
		CSSContent:     cssContent,
		JSContent:      jsContent,
		CSSPath:        cssPath,
		JSPath:         jsPath,
		CSSHash:        cssHash,
		JSHash:         jsHash,
		CSSFingerprint: FingerprintName(cssPath, []byte(cssContent)),
		JSFingerprint:  FingerprintName(jsPath, []byte(jsContent)),
	}, nil
}

// WriteToDir writes bundled files (and fingerprinted copies) to target output directory.
func (b *Bundler) WriteToDir(res *BundleResult, outDir string) ([]string, error) {
	var created []string

	if res.CSSContent != "" {
		outCSS := filepath.Join(outDir, filepath.FromSlash(res.CSSPath))
		if err := os.MkdirAll(filepath.Dir(outCSS), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(outCSS, []byte(res.CSSContent), 0o644); err != nil {
			return nil, err
		}
		created = append(created, res.CSSPath)
	}

	if res.JSContent != "" {
		outJS := filepath.Join(outDir, filepath.FromSlash(res.JSPath))
		if err := os.MkdirAll(filepath.Dir(outJS), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(outJS, []byte(res.JSContent), 0o644); err != nil {
			return nil, err
		}
		created = append(created, res.JSPath)
	}

	return created, nil
}

// MinifyCSS applies safe lightweight minification to CSS.
func MinifyCSS(css string) string {
	// Strip multi-line comments /* ... */
	reComment := regexp.MustCompile(`(?s)/\*.*?\*/`)
	css = reComment.ReplaceAllString(css, "")

	// Collapse multiple whitespaces
	reSpace := regexp.MustCompile(`\s+`)
	css = reSpace.ReplaceAllString(css, " ")

	// Trim spaces around standard delimiters
	for _, del := range []string{":", ";", "{", "}", ",", ">"} {
		css = strings.ReplaceAll(css, " "+del, del)
		css = strings.ReplaceAll(css, del+" ", del)
	}
	css = strings.ReplaceAll(css, ";}", "}")
	return strings.TrimSpace(css)
}

// MinifyJS applies safe lightweight minification to JS.
func MinifyJS(js string) string {
	// Strip multi-line comments
	reMulti := regexp.MustCompile(`(?s)/\*.*?\*/`)
	js = reMulti.ReplaceAllString(js, "")

	// Strip single-line comments that do not look like URLs (http:// or https://)
	lines := strings.Split(js, "\n")
	var out []string
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "//") {
			continue
		}
		if trim != "" {
			out = append(out, trim)
		}
	}
	return strings.Join(out, "\n")
}

func hash8(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:8]
}
