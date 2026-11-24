package ssg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/model"
	"github.com/krewire/ui/core"
	kiwdsl "github.com/krewire/ui/kiw"
	"gopkg.in/yaml.v3"
)

func LoadFromDir(root string) (*Site, error) {
	site := New()
	meta := loadMeta(root)

	if meta.Base != "" {
		site.WithBasePath(meta.Base)
	}

	setupI18n(site, root, meta)
	loadBuiltinUIComponents(site)
	loadInstalledPackages(site, root)
	loadProjectComponents(site, root)
	loadLayouts(site, root)

	collections, collectionData := loadCollections(root)
	if len(collectionData) > 0 {
		if meta.Data == nil {
			meta.Data = map[string]any{}
		}
		meta.Data["Collections"] = collectionData
	}

	loadPages(site, root, meta, collections)
	loadStaticAssets(site, root, meta)

	if meta.AutoAssets != nil {
		site.AutoAssets(meta.AutoAssets)
	}

	return site, nil
}

func loadMeta(root string) *meta {
	m := &meta{Data: map[string]any{}}
	for _, name := range []string{model.ConfigFileName, "krewire.yml"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		var raw struct {
			Title       string `yaml:"title"`
			Description string `yaml:"description"`
			Version     string `yaml:"version"`
			Project     *struct {
				Name    string `yaml:"name"`
				Kind    string `yaml:"kind"`
				Version string `yaml:"version"`
			} `yaml:"project"`
			Theme *struct {
				Default string            `yaml:"default"`
				Light   map[string]string `yaml:"light"`
				Dark    map[string]string `yaml:"dark"`
			} `yaml:"theme"`
			Build *struct {
				Base string `yaml:"base"`
			} `yaml:"build"`
			I18n       *I18nConfig      `yaml:"i18n"`
			AutoAssets *AutoAssetConfig `yaml:"auto_assets"`
		}
		if err := yaml.Unmarshal(b, &raw); err == nil {
			if raw.Version == "" && raw.Project != nil && raw.Project.Version != "" {
				raw.Version = raw.Project.Version
			}
			if raw.Title != "" {
				m.Title = raw.Title
			}
			if raw.Description != "" {
				m.Description = raw.Description
			}
			if raw.Version != "" {
				if m.Data == nil {
					m.Data = map[string]any{}
				}
				m.Data["Version"] = raw.Version
			}
			if raw.Theme != nil {
				t := &ui.Theme{Default: raw.Theme.Default}
				applyPalette(&t.Light, raw.Theme.Light)
				applyPalette(&t.Dark, raw.Theme.Dark)
				m.ThemeCSS = string(t.Style())
				if m.Data == nil {
					m.Data = map[string]any{}
				}
				m.Data["Theme"] = t
			}
			if raw.I18n != nil {
				m.I18n = raw.I18n
				if m.Data == nil {
					m.Data = map[string]any{}
				}
				m.Data["I18n"] = raw.I18n
			}
			if raw.AutoAssets != nil {
				m.AutoAssets = raw.AutoAssets
			}
			if raw.Build != nil {
				m.Base = raw.Build.Base
			}
		}
		break
	}
	return m
}

type meta struct {
	Title       string
	Description string
	Base        string
	Data        map[string]any
	ThemeCSS    string
	I18n        *I18nConfig
	AutoAssets  *AutoAssetConfig
}

func mergeMeta(base map[string]any, fm map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range fm {
		out[k] = v
	}
	return out
}

func strFromFM(fm map[string]any, key, fallback string) string {
	if v, ok := fm[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return strings.Trim(s, `"'`)
		}
	}
	return fallback
}

// relSource returns path relative to the site root for diagnostics.
func relSource(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}

func findKiwFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".kiw") {
			out = append(out, path)
		}
		return nil
	})
	return out
}

func routeFromRel(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, ".kiw")
	if rel == "index" {
		return "/"
	}
	if strings.HasSuffix(rel, "/index") {
		rel = strings.TrimSuffix(rel, "/index")
	}
	return "/" + rel
}

func resolveImports(site *Site, root, filePath string, imports []kiwdsl.ImportDecl) {
	for _, imp := range imports {
		if strings.HasPrefix(imp.Source, "./") || strings.HasPrefix(imp.Source, "../") {
			target := filepath.Join(filepath.Dir(filePath), filepath.FromSlash(imp.Source))
			if !strings.HasSuffix(target, ".kiw") {
				target += ".kiw"
			}
			if st, err := os.Stat(target); err == nil && !st.IsDir() {
				mod, err := kiwdsl.ParseKiwFile(target)
				if err == nil {
					style := strings.Join(mod.Styles, "\n")
					site.Component(Component{
						Name:     imp.LocalName,
						Body:     mod.Body,
						Style:    style,
						Defaults: mod.Defaults,
						Source:   relSource(root, target),
						Raw:      mod.Raw,
					})
				}
			}
		} else {
			// Package reference: e.g. @ui/Card or @acme/ui
			targetName := imp.ImportedName
			if targetName == "" {
				targetName = imp.LocalName
			}
			site.Alias(imp.LocalName, targetName)
		}
	}
}

func loadKiwPackages(site *Site, root, modulesDir string) {
	_ = filepath.WalkDir(modulesDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		manifestPath := filepath.Join(p, kiwdsl.PackageManifestFile)
		if st, err := os.Stat(manifestPath); err == nil && !st.IsDir() {
			manifest, err := kiwdsl.LoadPackageManifest(manifestPath)
			if err != nil {
				return nil
			}
			// Derive clean namespace (e.g. @acme/ui -> acme, my-cards -> my_cards)
			ns := strings.TrimPrefix(manifest.Name, "@")
			if idx := strings.Index(ns, "/"); idx >= 0 {
				ns = ns[:idx]
			}
			ns = strings.ReplaceAll(ns, "-", "_")

			for compName, compRel := range manifest.Components {
				compPath := filepath.Join(p, filepath.FromSlash(compRel))
				mod, err := kiwdsl.ParseKiwFile(compPath)
				if err != nil {
					continue
				}
				style := strings.Join(mod.Styles, "\n")
				namespaced := ns + ":" + compName
				pkgNamespaced := manifest.Name + ":" + compName
				compObj := Component{
					Name:     namespaced,
					Body:     mod.Body,
					Style:    style,
					Defaults: mod.Defaults,
					Source:   relSource(root, compPath),
					Raw:      mod.Raw,
				}
				site.Component(compObj)
				site.Component(Component{Name: pkgNamespaced, Body: mod.Body, Style: style, Defaults: mod.Defaults, Source: relSource(root, compPath), Raw: mod.Raw})
				// Also register under compName if not already present
				if _, exists := site.comps[compName]; !exists {
					site.Component(Component{Name: compName, Body: mod.Body, Style: style, Defaults: mod.Defaults, Source: relSource(root, compPath), Raw: mod.Raw})
				}
			}

			if manifest.Styles != "" {
				stylePath := filepath.Join(p, filepath.FromSlash(manifest.Styles))
				if b, err := os.ReadFile(stylePath); err == nil {
					assetName := fmt.Sprintf("pkg-%s.css", strings.ReplaceAll(ns, "/", "-"))
					site.DeclareAsset("/assets/" + assetName)
					site.Asset(assetName, string(b))
				}
			}
			return filepath.SkipDir
		}
		return nil
	})
}
