package book

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// exportPages writes each page under outDir as the sibling .html file its
// extensionless URL maps to: "/" becomes index.html and "/a/b" becomes
// outDir/a/b.html. A non-empty mount sub-path relocates the files so a book
// can share one output with an ssg site.
func exportPages(outDir string, pages []Page, mount string) error {
	root := outDir
	if mount != "" {
		root = filepath.Join(outDir, filepath.FromSlash(mount))
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	for _, p := range pages {
		name := "index.html"
		if rel := strings.Trim(p.Path, "/"); rel != "" {
			name = rel + ".html"
		}
		out := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		if err := p.Render(f); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

// createdPaths computes the files the build will produce for the book,
// mirroring the export shape for deterministic reporting.
func (b *Book) createdPaths(outDir string) []string {
	baseOut := outDir
	if b.mount != "" {
		baseOut = filepath.Join(outDir, filepath.FromSlash(b.mount))
	}
	var paths []string
	if !b.noRootTOC {
		paths = append(paths, filepath.Join(baseOut, "index.html"))
	}
	if len(b.extraCSS) == 0 {
		paths = append(paths, filepath.Join(baseOut, StylesheetName))
	}
	paths = append(paths, filepath.Join(baseOut, "search-index.json"))
	paths = append(paths, filepath.Join(baseOut, "nav.json"))
	for _, ch := range b.flattened() {
		rel := strings.TrimPrefix(ch.Path(), "/")
		paths = append(paths, filepath.Join(baseOut, filepath.FromSlash(rel)+".html"))
		if len(ch.Subs) > 0 {
			paths = append(paths, filepath.Join(baseOut, filepath.FromSlash(rel), "index.html"))
		}
	}
	sort.Strings(paths)
	return paths
}
