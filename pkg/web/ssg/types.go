package ssg

import (
	"html/template"
	"strings"
	"sync"

	"github.com/krewire/krewire/pkg/ui"
)

// Component is a reusable piece of UI. Body is the html/template source;
// Style is CSS scoped to the component's rendered root element.
type Component struct {
	Name  string
	Body  string
	Style string
}

// Layout wraps page content in a shared shell. Body receives a LayoutData
// value (fields Title, Content, Data). Style is scoped to the layout's root
// element.
type Layout struct {
	Name    string
	Body    string
	Style   string
	Scripts []string
}

// LayoutData is the value passed to a layout template.
type LayoutData struct {
	// Title is the page title.
	Title string
	// Content is the rendered page component HTML.
	Content template.HTML
	// Data is the page's data value.
	Data any
	// Version is the site version from krewire.yaml, exposed for layout
	// chrome (badges) as {{.Version}}.
	Version string
	// AssetVersion is a cache-busting token derived from Version (strips "v" prefix).
	// Used with {{assetLinks .AssetVersion}} to auto-inject CSS/JS tags.
	AssetVersion string
	// BasePath is the site's URL base ("/" for root sites, "/guide/" for a
	// site mounted under a sub-path), so hand-written layout links resolve
	// against one source of truth instead of hardcoding a sub-path.
	BasePath string
}

// Page is one output page of the site.
type Page struct {
	// Path is the page route: "/" for the root, extensionless file-based
	// routes like "/about" or "/docs/quickstart", or an explicit filename
	// like "/404.html". Each route emits a sibling .html file: "/" ->
	// "index.html", "/about" -> "about.html".
	Path string
	// Title is passed to the layout.
	Title string
	// Layout names the Layout wrapping this page.
	Layout string
	// Root names the root component of the page.
	Root string
	// Data is passed to the root component.
	Data any
	// Props, when the Site emits props, is serialized to a data-props
	// attribute on the page's root element for client-side mounting.
	Props any
	// Scripts are <script src> URLs injected before the closing body tag,
	// empty by default.
	Scripts []string
}

// Site builds a static website from components, layouts, pages, and assets.
type Site struct {
	funcs     template.FuncMap
	layouts   map[string]*Layout
	comps     map[string]*Component
	pages     []*Page
	assets    map[string]string
	reg       *ui.Registry
	emitProps bool

	// pipeline holds the asset transform rules (KWF-DR5YU).
	pipeline []PipelineRule
	// manifest maps logical asset paths to fingerprinted URLs for the
	// asset() template helper.
	manifest map[string]string
	// auto drives automatic CSS/JS injection into rendered documents.
	auto autoAssets
	// scriptAssets records assets that belong to a single layout or page, so
	// they are never injected site-wide.
	scriptAssets map[string]bool
	// declared records asset paths produced outside the build (plugins), which
	// are injected but never written by Build.
	declared map[string]bool
	// assetMeta holds per-asset ordering metadata (layer, order, kind, scoped).
	// It is the backing store for the resolved asset plan.
	assetMeta map[string]assetMeta
	// autoOrderErr collects configuration errors from AutoAssets, such as an
	// unknown layer name, so a bad entry is reported without failing the build.
	autoOrderErr []error
	// assetSeq assigns each asset a registration sequence, used as the final
	// tie-break so the plan order is deterministic.
	assetSeq map[string]int
	// assetSeqNext is the next registration sequence number.
	assetSeqNext int

	// BasePath is the URL base the site is served under, e.g. "/guide/".
	// Asset URLs emitted by injection and the asset helpers are prefixed
	// with it so a site can be mounted under a sub-path. Empty or "/"
	// keeps the historical root-absolute "/assets/..." URLs.
	BasePath string

	// templateFns is the resolved func map, kept so a component body can be
	// rendered as its own template. html/template exposes no getter for it.
	templateFns template.FuncMap

	set *template.Template
	mu  sync.Mutex
	// used records every component and layout whose styles were referenced
	// during a build.
	used map[string]bool
	// wasm holds client WebAssembly island hydration configuration.
	wasm WasmConfig
}

// WasmConfig configures client WebAssembly island hydration (KWF-T4X9P).
type WasmConfig struct {
	Enabled bool
	WASMURL string
	JSURL   string
}

// Wasm configures WebAssembly island hydration.
func (s *Site) Wasm(cfg WasmConfig) *Site {
	s.wasm = cfg
	return s
}

// New returns an empty Site.
func New() *Site {
	return &Site{
		layouts:      map[string]*Layout{},
		comps:        map[string]*Component{},
		assets:       map[string]string{},
		used:         map[string]bool{},
		manifest:     map[string]string{},
		scriptAssets: map[string]bool{},
		declared:     map[string]bool{},
		assetMeta:    map[string]assetMeta{},
		assetSeq:     map[string]int{},
		// Automatic CSS/JS injection is on by default, with scripts in <head>
		// so first-paint scripts such as theme bootstrapping still run early.
		auto: autoAssets{enabled: true, jsInHead: true},
	}
}

// Pipeline installs asset transform rules applied at build time (KWF-DR5YU).
func (s *Site) Pipeline(rules []PipelineRule) *Site {
	s.pipeline = rules
	return s
}

// WithBasePath pins the URL base the site is served under, e.g. "/guide/".
// Injected asset URLs and the asset helpers are prefixed accordingly, so a
// site can be mounted under a sub-path. Empty or "/" keeps the historical
// root-absolute "/assets/..." URLs unchanged.
func (s *Site) WithBasePath(base string) *Site {
	s.BasePath = normalizeSiteBase(base)
	return s
}

// normalizeSiteBase canonicalizes a URL base into leading-and-trailing-slash
// form ("/", "/guide/", ...). Empty normalizes to "/" (no prefix).
func normalizeSiteBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "/" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/") + "/"
	return base
}

// prefixBaseURL prefixes a root-absolute URL path with the site's base.
// Non-absolute URLs (external, anchors, relative) pass through unchanged,
// and base "/" or "" returns the URL untouched.
func (s *Site) prefixBaseURL(url string) string {
	if s == nil || s.BasePath == "" || s.BasePath == "/" {
		return url
	}
	if !strings.HasPrefix(url, "/") {
		return url
	}
	trimmed := strings.TrimPrefix(s.BasePath, "/")
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed != "" && (url == "/"+trimmed || strings.HasPrefix(url, "/"+trimmed+"/")) {
		return url
	}
	return strings.TrimSuffix(s.BasePath, "/") + url
}
