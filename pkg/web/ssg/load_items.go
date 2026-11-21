package ssg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/pkg/i18n"
	"github.com/krewire/ui"
	kiwdsl "github.com/krewire/ui/kiw"
)

func setupI18n(site *Site, root string, meta *meta) {
	langPath := "lang"
	if meta.I18n != nil && meta.I18n.BasePath != "" {
		langPath = meta.I18n.BasePath
	}
	if st, err := os.Stat(filepath.Join(root, langPath)); err == nil && st.IsDir() {
		var opts []i18n.Option
		opts = append(opts, i18n.WithBasePath(langPath))
		if meta.I18n != nil {
			if meta.I18n.DefaultLocale != "" {
				opts = append(opts, i18n.WithDefaultLocale(meta.I18n.DefaultLocale))
			}
			if meta.I18n.FallbackLocale != "" {
				opts = append(opts, i18n.WithFallbackLocale(meta.I18n.FallbackLocale))
			}
		}
		if bundle, err := i18n.New(os.DirFS(root), opts...); err == nil {
			defaultLoc := bundle.DefaultLocale()
			site.Func("t", func(key string, args ...any) string {
				return bundle.Translate(defaultLoc, key, args...)
			})
			site.Func("translate", func(key string, args ...any) string {
				return bundle.Translate(defaultLoc, key, args...)
			})
			site.Func("tLocale", func(locale, key string, args ...any) string {
				return bundle.Translate(locale, key, args...)
			})
			site.Func("locales", func() []string {
				return bundle.Locales()
			})
			if meta.Data == nil {
				meta.Data = map[string]any{}
			}
			meta.Data["Locales"] = bundle.Locales()
			meta.Data["DefaultLocale"] = defaultLoc
		}
	}
}

func loadBuiltinUIComponents(site *Site) {
	entries, err := ui.ComponentsFS.ReadDir(".")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kiw") {
			continue
		}
		content, err := ui.ComponentsFS.ReadFile(e.Name())
		if err != nil {
			continue
		}
		mod, err := kiwdsl.ParseKiw(string(content))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".kiw")
		style := strings.Join(mod.Styles, "\n")
		site.Component(Component{
			Name:     name,
			Body:     mod.Body,
			Style:    style,
			Defaults: mod.Defaults,
			Source:   "kui/" + e.Name(),
			Raw:      mod.Raw,
		})
	}
}

func loadInstalledPackages(site *Site, root string) {
	for _, pkgDir := range []string{filepath.Join(root, ".krewire", "pkg"), filepath.Join(root, "kiw_modules")} {
		if st, err := os.Stat(pkgDir); err == nil && st.IsDir() {
			loadKiwPackages(site, root, pkgDir)
		}
	}
}

func loadProjectComponents(site *Site, root string) {
	compsDir := filepath.Join(root, "components")
	for _, p := range findKiwFiles(compsDir) {
		mod, err := kiwdsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(compsDir, p)
		rel = filepath.ToSlash(rel)
		relNoExt := strings.TrimSuffix(rel, ".kiw")
		baseName := filepath.Base(relNoExt)
		if strings.HasPrefix(baseName, "_") {
			continue
		}
		style := strings.Join(mod.Styles, "\n")

		site.Component(Component{
			Name:     baseName,
			Body:     mod.Body,
			Style:    style,
			Defaults: mod.Defaults,
			Source:   relSource(root, p),
			Raw:      mod.Raw,
		})

		if strings.Contains(relNoExt, "/") {
			colonName := strings.ReplaceAll(relNoExt, "/", ":")
			dotName := strings.ReplaceAll(relNoExt, "/", ".")
			underscoreName := strings.ReplaceAll(relNoExt, "/", "_")
			site.Component(Component{Name: relNoExt, Body: mod.Body, Style: style, Defaults: mod.Defaults, Source: relSource(root, p), Raw: mod.Raw})
			site.Component(Component{Name: colonName, Body: mod.Body, Style: style, Defaults: mod.Defaults, Source: relSource(root, p), Raw: mod.Raw})
			site.Component(Component{Name: dotName, Body: mod.Body, Style: style, Defaults: mod.Defaults, Source: relSource(root, p), Raw: mod.Raw})
			site.Component(Component{Name: underscoreName, Body: mod.Body, Style: style, Defaults: mod.Defaults, Source: relSource(root, p), Raw: mod.Raw})
		}

		if len(mod.Imports) > 0 {
			resolveImports(site, root, p, mod.Imports)
		}
	}
}

func loadLayouts(site *Site, root string) {
	for _, p := range findKiwFiles(filepath.Join(root, "layouts")) {
		mod, err := kiwdsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(p), ".kiw")
		if strings.HasPrefix(name, "_") {
			continue
		}
		if len(mod.Imports) > 0 {
			resolveImports(site, root, p, mod.Imports)
		}
		style := strings.Join(mod.Styles, "\n")
		var scripts []string
		for i, sc := range mod.Scripts {
			if strings.TrimSpace(sc) == "" {
				continue
			}
			spath := "/" + filepath.Join("assets", fmt.Sprintf("layout-%s-%d.js", name, i))
			site.ScriptAsset(strings.TrimPrefix(spath, "/"), sc)
			scripts = append(scripts, spath)
		}
		site.Layout(Layout{Name: name, Body: mod.Body, Style: style, Scripts: scripts, Source: relSource(root, p), Raw: mod.Raw})
	}
}

func loadCollections(root string) (map[string]*Collection, map[string]any) {
	collections := map[string]*Collection{}
	collectionData := map[string]any{}
	contentRoot := filepath.Join(root, "content")
	st, err := os.Stat(contentRoot)
	if err != nil || !st.IsDir() {
		return collections, collectionData
	}

	entries, _ := os.ReadDir(contentRoot)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "_") {
			continue
		}
		cc := CollectionConfig{Name: name, Dir: filepath.Join("content", name)}
		col, err := BuildCollection(cc, root)
		if err != nil {
			continue
		}
		col.FilterDrafts(false)
		col.FilterFuture(false)
		if len(col.Pages) == 0 {
			continue
		}
		collections[name] = col
		pagesData := make([]map[string]any, len(col.Pages))
		for i, p := range col.Pages {
			pagesData[i] = map[string]any{
				"Title":       p.Config.Title,
				"Date":        p.RawDate,
				"Permalink":   p.URL,
				"Data":        p.Config.Extra,
				"Content":     p.Content,
				"Draft":       p.Config.Draft,
				"Slug":        p.Config.Slug,
				"Description": p.Config.Description,
			}
		}
		collectionData[name] = pagesData
	}
	return collections, collectionData
}

func loadPages(site *Site, root string, meta *meta, collections map[string]*Collection) {
	for _, p := range findKiwFiles(filepath.Join(root, "pages")) {
		rel, _ := filepath.Rel(filepath.Join(root, "pages"), p)
		if strings.HasPrefix(filepath.Base(p), "_") || strings.Contains(rel, string(filepath.Separator)+"_") {
			continue
		}
		mod, err := kiwdsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		// Dynamic [slug] route
		if strings.Contains(filepath.Base(p), "[slug]") {
			dir := filepath.Dir(rel)
			collectionName := filepath.Base(dir)
			if dir == "." {
				collectionName = ""
			}
			col, ok := collections[collectionName]
			if !ok {
				continue
			}
			for _, item := range col.Pages {
				slug := item.Config.Slug
				if slug == "" {
					slug = strings.TrimSuffix(filepath.Base(item.Path), ".md")
				}
				route := "/" + filepath.ToSlash(filepath.Join(dir, slug))
				title := strFromFM(mod.Frontmatter, "title", item.Config.Title)
				if title == "" {
					title = meta.Title
				}
				layout := strFromFM(mod.Frontmatter, "layout", "")
				if layout == "" {
					if _, err := os.Stat(filepath.Join(root, "layouts", collectionName+".kiw")); err == nil {
						layout = collectionName
					} else if _, err := os.Stat(filepath.Join(root, "layouts", "Post.kiw")); err == nil {
						layout = "Post"
					} else if _, err := os.Stat(filepath.Join(root, "layouts", "Base.kiw")); err == nil {
						layout = "Base"
					}
				}
				data := mergeMeta(meta.Data, mod.Frontmatter)
				data["Title"] = title
				data["Content"] = item.Content
				data["Page"] = item
				for k, v := range item.Config.Extra {
					if _, ok := data[k]; !ok {
						data[k] = v
					}
				}
				body := mod.Body
				style := strings.Join(mod.Styles, "\n")
				compName := "page:" + route
				if style != "" {
					site.Component(Component{Name: compName, Body: body, Style: style, Source: relSource(root, p), Raw: mod.Raw})
					body = ""
				} else {
					compName = ""
					if body == "" {
						body = "<div></div>"
					}
				}
				rootName := compName
				if rootName == "" {
					rootName = "page:" + route + ":body"
					site.Component(Component{Name: rootName, Body: body, Source: relSource(root, p), Raw: mod.Raw})
				}
				site.Page(Page{
					Path:   route,
					Title:  title,
					Layout: layout,
					Root:   rootName,
					Data:   data,
				})
			}
			continue
		}

		route := routeFromRel(rel)
		if len(mod.Imports) > 0 {
			resolveImports(site, root, p, mod.Imports)
		}
		title := strFromFM(mod.Frontmatter, "title", meta.Title)
		layout := strFromFM(mod.Frontmatter, "layout", "")
		if layout == "" {
			if _, err := os.Stat(filepath.Join(root, "layouts", "Base.kiw")); err == nil {
				layout = "Base"
			}
		}
		data := mergeMeta(meta.Data, mod.Frontmatter)
		data["Title"] = title
		if d := strFromFM(mod.Frontmatter, "description", ""); d != "" {
			data["Description"] = d
		}
		body := mod.Body
		style := strings.Join(mod.Styles, "\n")
		pageCompName := "page:" + route
		if style != "" {
			site.Component(Component{Name: pageCompName, Body: body, Style: style, Source: relSource(root, p), Raw: mod.Raw})
			body = ""
		} else {
			pageCompName = ""
			if body == "" {
				body = "<div></div>"
			}
		}
		rootName := pageCompName
		if rootName == "" {
			rootName = "page:" + route + ":body"
			site.Component(Component{Name: rootName, Body: body, Source: relSource(root, p), Raw: mod.Raw})
		}
		var scriptPaths []string
		for i, sc := range mod.Scripts {
			if strings.TrimSpace(sc) == "" {
				continue
			}
			spath := "/" + filepath.Join("assets", "page"+strings.ReplaceAll(route, "/", "-")+string(rune('0'+i))+".js")
			site.ScriptAsset(strings.TrimPrefix(spath, "/"), sc)
			scriptPaths = append(scriptPaths, spath)
		}
		site.Page(Page{
			Path:    route,
			Title:   title,
			Layout:  layout,
			Root:    rootName,
			Data:    data,
			Scripts: scriptPaths,
		})
	}
}

func loadStaticAssets(site *Site, root string, meta *meta) {
	publicDir := filepath.Join(root, "public")
	if st, err := os.Stat(publicDir); err == nil && st.IsDir() {
		_ = filepath.WalkDir(publicDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(publicDir, path)
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			site.Asset(rel, string(b))
			return nil
		})
	}

	assetsDir := filepath.Join(root, "assets")
	if st, err := os.Stat(assetsDir); err == nil && st.IsDir() {
		_ = filepath.WalkDir(assetsDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if _, exists := site.AssetBody(rel); !exists {
				b, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				site.Asset(rel, string(b))
			}
			return nil
		})
	}

	if meta.ThemeCSS != "" {
		if _, exists := site.AssetBody("assets/theme.css"); !exists {
			site.Asset("assets/theme.css", meta.ThemeCSS)
		}
	}
}
