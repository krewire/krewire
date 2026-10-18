package ssg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"path"
	"sort"
	"strings"

	"github.com/krewire/krewire/packages/runtime/mount"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// renderComponent renders a component with data, marks it used, and injects
// its scope attribute onto the rendered root element.
func (s *Site) renderComponent(name string, data ...any) (template.HTML, error) {
	if err := s.prepare(); err != nil {
		return "", err
	}
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	if _, ok := s.comps[name]; ok {
		s.markUsed(name)
		var buf bytes.Buffer
		if err := s.set.ExecuteTemplate(&buf, name, d); err != nil {
			return "", err
		}
		return scopeFragment(name, buf.String())
	}
	if s.reg != nil {
		if out, err := s.reg.Render(name, componentProps(d)); err == nil {
			return out, nil
		} else if !strings.HasPrefix(err.Error(), "ui: undefined component") {
			return "", err
		}
	}
	return "", fmt.Errorf("ssg: undefined component %q", name)
}

// componentProps strips framework-injected site-wide control keys from a data
// value before it reaches a typed registry component. Templates need the full
// site data (Title, Theme, Collections, …), but typed components decode props
// strictly, so framework keys must not bubble through as unknown fields.
func componentProps(data any) any {
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "Title", "Description", "Theme", "IncludeDrafts", "IncludeFuture", "Collections":
			continue
		default:
			out[k] = v
		}
	}
	return out
}

// renderPage renders a page's root component inside its layout.
func (s *Site) renderPage(p *Page) (string, error) {
	root, err := s.renderComponent(p.Root, p.Data)
	if err != nil {
		return "", err
	}
	if s.emitProps && p.Props != nil {
		root, err = injectDataProps(root, p.Props)
		if err != nil {
			return "", err
		}
	}
	_, ok := s.layouts[p.Layout]
	if !ok {
		return "", fmt.Errorf("ssg: undefined layout %q", p.Layout)
	}
	s.markUsed(p.Layout)
	version := ""
	if m, ok := p.Data.(map[string]any); ok {
		if v, ok := m["Version"].(string); ok {
			version = v
		}
	}
	// Derive a clean cache-busting token: strip the leading "v" if present
	// so URLs get ?v=0.1.0 instead of ?v=v0.1.0. VersionToken owns that rule so
	// the template helper and automatic injection cannot spell it differently.
	assetVersion := VersionToken(version)
	var buf bytes.Buffer
	if err := s.set.ExecuteTemplate(&buf, p.Layout, LayoutData{
		Title:        p.Title,
		Content:      root,
		Data:         p.Data,
		Version:      version,
		AssetVersion: assetVersion,
		BasePath:     s.templateBasePath(),
	}); err != nil {
		return "", err
	}
	// Inject the layout scope attribute onto the document root so layout
	// styles can be scoped under [data-kiw-layout="name"].
	scoped, err := scopeDocument(p.Layout, buf.String())
	if err != nil {
		return "", err
	}
	body := string(scoped)
	// Auto-inject CSS/JS after layout scoping and before page scripts are
	// appended, so injected tags sit inside <head>/<body> and page-scoped
	// scripts keep landing last.
	body, err = s.injectAssets(body, assetVersion)
	if err != nil {
		return "", err
	}
	var allScripts []string
	if l, ok := s.layouts[p.Layout]; ok && len(l.Scripts) > 0 {
		for _, script := range l.Scripts {
			allScripts = append(allScripts, s.prefixBaseURL(script))
		}
	}
	if len(p.Scripts) > 0 {
		for _, script := range p.Scripts {
			allScripts = append(allScripts, s.prefixBaseURL(script))
		}
	}
	if len(allScripts) > 0 {
		body = injectScripts(body, allScripts)
	}
	if s.wasm.Enabled || strings.Contains(body, mount.AttrMount) {
		body = s.injectWasmLoader(body)
	}
	return body, nil
}

// injectWasmLoader appends the Go WebAssembly glue and initialization script before </body>.
func (s *Site) injectWasmLoader(doc string) string {
	jsURL := s.wasm.JSURL
	if jsURL == "" {
		jsURL = s.asset("wasm_exec.js")
	}
	wasmURL := s.wasm.WASMURL
	if wasmURL == "" {
		wasmURL = s.asset("runtime.wasm")
	}

	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return doc
	}
	bodyNode := findElement(root, "body")
	if bodyNode == nil {
		return doc
	}
	referenced := referencedAssets(root)

	// Inject wasm_exec.js if not already present
	if !referenced[jsURL] && !referenced[path.Base(jsURL)] {
		bodyNode.AppendChild(&html.Node{
			Type:     html.ElementNode,
			Data:     "script",
			DataAtom: atom.Script,
			Attr:     []html.Attribute{{Key: "src", Val: jsURL}},
		})
	}

	loaderScript := fmt.Sprintf(`if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}
const go = new Go();
WebAssembly.instantiateStreaming(fetch("%s"), go.importObject).then((result) => {
  go.run(result.instance);
});`, wasmURL)

	scriptNode := &html.Node{
		Type:     html.ElementNode,
		Data:     "script",
		DataAtom: atom.Script,
	}
	scriptNode.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: loaderScript,
	})
	bodyNode.AppendChild(scriptNode)

	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return doc
	}
	return buf.String()
}

// markUsed records that a component or layout was referenced.
func (s *Site) markUsed(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.used[name] = true
}

// collectedCSS concatenates the scoped styles of every component and layout
// referenced during the last build, in deterministic order.
func (s *Site) collectedCSS() string {
	var out strings.Builder
	names := make([]string, 0, len(s.used))
	for name := range s.used {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if c, ok := s.comps[name]; ok && strings.TrimSpace(c.Style) != "" {
			out.WriteString("/* " + name + " */\n")
			out.WriteString(scopeCSS(`[data-kiw-component="`+name+`"]`, c.Style, true))
			out.WriteString("\n")
			continue
		}
		if l, ok := s.layouts[name]; ok && strings.TrimSpace(l.Style) != "" {
			out.WriteString("/* " + name + " */\n")
			out.WriteString(scopeCSS(`[data-kiw-layout="`+name+`"]`, l.Style, false))
			out.WriteString("\n")
		}
	}
	return out.String()
}

// asset resolves a logical asset path to its fingerprinted URL from the
// build manifest, falling back to a plain /assets/ path (KWF-DR5YU).
// The resolved URL is prefixed with the site's base path so a site mounted
// under a sub-path (e.g. "/guide/") links its own assets.
func (s *Site) asset(name string) string {
	if u, ok := s.manifest[name]; ok {
		return s.prefixBaseURL(u)
	}
	return s.prefixBaseURL(assetURL(name))
}

// assetLinks generates <link rel="stylesheet"> and <script src> HTML tags for
// all CSS and JS assets in resolved plan order, so the helper and automatic
// injection can never disagree about cascade order.
//
// The version argument is appended as a cache-busting query string (?v=<ver>).
// Usage in a layout: {{assetLinks .AssetVersion}}
func (s *Site) assetLinks(version string) template.HTML {
	versionSuffix := VersionQuery(version)
	cssURLs, jsURLs := s.InjectedAssets()
	var sb strings.Builder
	for _, url := range cssURLs {
		sb.WriteString(`<link rel="stylesheet" href="` + url + versionSuffix + `">` + "\n  ")
	}
	for _, url := range jsURLs {
		sb.WriteString(`<script src="` + url + versionSuffix + `"></script>` + "\n  ")
	}
	return template.HTML(strings.TrimRight(sb.String(), "\n  "))
}

// templateBasePath exposes the site's URL base to templates so hand-written
// links resolve against one source of truth instead of hardcoding a
// sub-path per layout. Root sites get "/", mounted sites "/guide/".
func (s *Site) templateBasePath() string {
	if s == nil || s.BasePath == "" {
		return "/"
	}
	return s.BasePath
}

// templateBaseURL prefixes one root-absolute path with the site's base for
// use inside templates: {{baseURL "/assets/app.js"}},
// {{baseURL "/#section"}}. Anything else passes through untouched.
func (s *Site) templateBaseURL(path string) string {
	return s.prefixBaseURL(path)
}

// dict builds a map from alternating key-value pairs for template calls like
// {{component "Card" (dict "Title" "Go" "Desc" "...")}} — frontmatter-free.
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments")
	}
	m := make(map[string]any, len(values))
	for i := 0; i < len(values); i += 2 {
		k, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key must be string, got %T", values[i])
		}
		var val any = values[i+1]
		if k == "Body" {
			if s, ok := values[i+1].(string); ok {
				val = template.HTML(s)
			}
		}
		m[k] = val
		if len(k) > 0 {
			upper := strings.ToUpper(k[:1]) + k[1:]
			lower := strings.ToLower(k[:1]) + k[1:]
			if _, exists := m[upper]; !exists {
				m[upper] = val
			}
			if _, exists := m[lower]; !exists {
				m[lower] = val
			}
		}
	}
	return m, nil
}

// prepare parses component and layout templates into one template set.
func (s *Site) prepare() error {
	if s.set != nil {
		return nil
	}
	funcs := template.FuncMap{}
	for name, fn := range s.funcs {
		funcs[name] = fn
	}
	funcs["safeHTML"] = func(s any) template.HTML {
		return template.HTML(fmt.Sprint(s))
	}
	funcs["raw"] = func(s any) template.HTML {
		return template.HTML(fmt.Sprint(s))
	}
	funcs["component"] = s.renderComponent
	funcs["mount"] = s.renderMount
	funcs["dict"] = dict
	funcs["asset"] = s.asset
	// assetLinks emits <link> and <script> tags for every asset that matches
	// the well-known CSS/JS patterns (style.css, tailwind.css, ui.css, app.js …).
	// An optional version suffix is appended as a cache-busting query string.
	// Templates use it as: {{assetLinks .Version}} or {{assetLinks ""}}
	funcs["assetLinks"] = s.assetLinks
	// basePath returns the site's URL base ("/" for root sites, "/guide/"
	// for a site mounted under a sub-path), so hand-written template links
	// can resolve against one source of truth: {{basePath}}assets/app.js,
	// {{basePath}}#section, and so on.
	funcs["basePath"] = s.templateBasePath
	// baseURL prefixes a root-absolute path with the site's base, leaving
	// external, anchor, and relative URLs untouched.
	funcs["baseURL"] = s.templateBaseURL
	// not is a convenience boolean negation helper for template conditionals.
	funcs["not"] = func(v any) bool {
		switch x := v.(type) {
		case bool:
			return !x
		case string:
			return x == ""
		default:
			return v == nil
		}
	}
	// A component body may itself contain component calls — the DSL desugars
	// them to {{component "X" ...}}, but the body travels as a string literal
	// inside the parent's (dict ...), so html/template would emit that text
	// verbatim instead of running it. Resolve such bodies here instead.
	funcs["dict"] = s.componentDict
	// class collapses conditional class composition: base + modifiers + trailing tokens (KWL-DSL-CLASS)
	funcs["class"] = ClassHelper
	set := template.New("").Funcs(funcs)
	compNames := make([]string, 0, len(s.comps))
	for name := range s.comps {
		compNames = append(compNames, name)
	}
	sort.Strings(compNames)
	for _, name := range compNames {
		if _, err := set.New(name).Parse(s.comps[name].Body); err != nil {
			return fmt.Errorf("ssg: component %q: %w", name, err)
		}
	}
	layoutNames := make([]string, 0, len(s.layouts))
	for name := range s.layouts {
		layoutNames = append(layoutNames, name)
	}
	sort.Strings(layoutNames)
	for _, name := range layoutNames {
		if _, err := set.New(name).Parse(s.layouts[name].Body); err != nil {
			return fmt.Errorf("ssg: layout %q: %w", name, err)
		}
	}
	s.set = set
	s.templateFns = funcs
	return nil
}

// componentDict is the template `dict` used for component props. It behaves
// exactly like dict except for the Body key, which is resolved as a template
// fragment rather than passed through as text.
//
// The DSL desugars a nested child (<BreadcrumbItem />) inside a parent's body
// to the literal text `{{component "BreadcrumbItem" (dict ...)}}`. Because
// that text arrives as a *string argument* to (dict ...), html/template would
// print it into the page verbatim and the child would silently never render —
// exactly what a real build showed. Parsing the body and executing it here
// makes nesting work, and is a no-op for a body with no actions in it.
func (s *Site) componentDict(values ...any) (map[string]any, error) {
	m := make(map[string]any, len(values))
	for i := 0; i < len(values); i += 2 {
		k, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key must be string, got %T", values[i])
		}
		var val any = values[i+1]
		if k == "Body" {
			if src, ok := values[i+1].(string); ok {
				rendered, err := s.renderFragment(src)
				if err != nil {
					return nil, err
				}
				val = rendered
			}
		}
		m[k] = val
		if len(k) > 0 {
			upper := strings.ToUpper(k[:1]) + k[1:]
			lower := strings.ToLower(k[:1]) + k[1:]
			if _, exists := m[upper]; !exists {
				m[upper] = val
			}
			if _, exists := m[lower]; !exists {
				m[lower] = val
			}
		}
	}
	return m, nil
}

// renderFragment executes a component body as a template fragment, so nested
// component calls inside it are resolved instead of printed.
//
// The fragment is built as its own template rather than added to the site's set:
// adding to the set would redefine the shared root template while it is being
// executed. It reuses the site's func map, so `component` and friends resolve.
func (s *Site) renderFragment(src string) (template.HTML, error) {
	if !strings.Contains(src, "{{") {
		return template.HTML(src), nil
	}
	if err := s.prepare(); err != nil {
		return "", err
	}
	frag, err := template.New("fragment").Funcs(s.templateFns).Parse(src)
	if err != nil {
		return "", fmt.Errorf("ssg: component body: %w", err)
	}
	var buf bytes.Buffer
	if err := frag.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("ssg: component body: %w", err)
	}
	return template.HTML(buf.String()), nil
}

// injectDataProps serializes props to a data-props attribute on the first
// element of a rendered fragment, giving client-side frameworks a stable,
// opt-in hydration payload.
func injectDataProps(fragHTML template.HTML, props any) (template.HTML, error) {
	data, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("ssg: encode page props: %w", err)
	}
	frag, err := html.ParseFragment(strings.NewReader(string(fragHTML)), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", err
	}
	target := firstElement(frag)
	if target == nil {
		return fragHTML, nil
	}
	target.Attr = append(target.Attr, html.Attribute{Key: "data-props", Val: string(data)})
	var buf bytes.Buffer
	for _, n := range frag {
		if err := html.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return template.HTML(buf.String()), nil
}

// injectScripts inserts <script src> tags before the closing </body> tag,
// the documented mount script slot, which is empty by default.
func injectScripts(body string, scripts []string) string {
	var sb strings.Builder
	for _, src := range scripts {
		sb.WriteString(`<script src="`)
		sb.WriteString(src)
		sb.WriteString(`"></script>`)
	}
	sb.WriteString("</body>")
	idx := strings.LastIndex(body, "</body>")
	if idx < 0 {
		return body + sb.String()
	}
	return body[:idx] + sb.String()
}
