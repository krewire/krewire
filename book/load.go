package book

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type frontmatterMeta struct {
	Title  string `yaml:"title"`
	Order  *int   `yaml:"order"`
	Weight *int   `yaml:"weight"`
	Prev   string `yaml:"prev"`
	Back   string `yaml:"back"`
	Next   string `yaml:"next"`
}

func (fm frontmatterMeta) resolveOrder() (int, bool) {
	if fm.Order != nil {
		return *fm.Order, true
	}
	if fm.Weight != nil {
		return *fm.Weight, true
	}
	return 0, false
}

func (fm frontmatterMeta) resolvePrev() string {
	if fm.Prev != "" {
		return fm.Prev
	}
	return fm.Back
}

// Source is a loaded content file.
type source struct {
	file     string
	title    string
	body     []byte
	order    int
	hasOrder bool
	prev     string
	next     string
}

// item is one content unit: a chapter file or a directory that
// becomes a chapter with subchapters.
type item struct {
	// name is the entry name used for ordering and slugging.
	name string
	// isDir indicates whether this item is a directory.
	isDir bool
	// chapter holds the chapter body; for directories this is the optional
	// index.md/_index.md, empty when the chapter page is auto-generated.
	chapter source
	// subs holds child items (files or subdirectories).
	subs []item
	// order and hasOrder are derived from frontmatter
	order    int
	hasOrder bool
}

// Load reads a content directory into a Book in reading order, served from
// the site root. Default include/exclude rules apply (README notes skipped).
func Load(input, title, author string) (*Book, error) {
	return LoadWithBase(input, title, author, "/")
}

// LoadWithRules reads a content directory into a Book applying explicit
// include/exclude glob rules (slash-separated patterns relative to input;
// ** crosses segments). nil selects the defaults — include "**/*.md",
// exclude README/readme developer notes.
func LoadWithRules(input, title, author, base string, include, exclude []string) (*Book, error) {
	return loadWithRules(input, title, author, base, include, exclude)
}

// LoadWithBase reads a content directory into a Book in reading order. Top
// level Markdown files become chapters; a directory becomes a chapter whose
// subchapters are the Markdown files inside it, ordered by numeric prefix. An
// index.md or _index.md inside a directory becomes the chapter page body;
// without one, the chapter page lists its subchapters automatically. The base
// is the URL prefix the site will be served under, e.g. "/guide/".
// YAML frontmatter (leading --- block) is tolerated and stripped so content
// files stay compatible with pkg/web/ssg collections.
func LoadWithBase(input, title, author, base string) (*Book, error) {
	return loadWithRules(input, title, author, base, nil, nil)
}

func sortItems(items []item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].hasOrder && items[j].hasOrder {
			if items[i].order != items[j].order {
				return items[i].order < items[j].order
			}
		} else if items[i].hasOrder {
			return true
		} else if items[j].hasOrder {
			return false
		}
		ni, si := splitOrder(items[i].name)
		nj, sj := splitOrder(items[j].name)
		if ni != nj {
			return ni < nj
		}
		return si < sj
	})
}

func loadDir(dirPath, relPrefix string, include, exclude []string) ([]item, *source, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, nil, fmt.Errorf("book: read %s: %w", dirPath, err)
	}

	var items []item
	var dirIndex *source

	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(dirPath, name)
		relPath := name
		if relPrefix != "" {
			relPath = relPrefix + "/" + name
		}

		if e.IsDir() {
			subItems, subIndex, err := loadDir(fullPath, relPath, include, exclude)
			if err != nil {
				return nil, nil, err
			}
			it := item{
				name:  name,
				isDir: true,
				subs:  subItems,
			}
			if subIndex != nil {
				it.chapter = *subIndex
				if subIndex.hasOrder {
					it.order = subIndex.order
					it.hasOrder = true
				}
			}
			if it.chapter.file != "" || len(it.subs) > 0 {
				items = append(items, it)
			}
			continue
		}

		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if !accepts(include, exclude, relPath) {
			continue
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, nil, fmt.Errorf("book: read %s: %w", fullPath, err)
		}
		meta, cleanBody := parseFrontmatterAndBody(data)
		order, hasOrder := meta.resolveOrder()
		src := source{
			file:     name,
			body:     cleanBody,
			order:    order,
			hasOrder: hasOrder,
			prev:     meta.resolvePrev(),
			next:     meta.Next,
		}
		if isIndex(name) {
			dirIndex = &src
			continue
		}
		items = append(items, item{
			name:     name,
			isDir:    false,
			chapter:  src,
			order:    order,
			hasOrder: hasOrder,
		})
	}

	sortItems(items)
	return items, dirIndex, nil
}

func buildChapterTree(items []item, parent *Chapter, parentNumber int, base string) []*Chapter {
	var chapters []*Chapter
	for j, it := range items {
		slug := slugFor(it.name)
		ch := &Chapter{
			Number:     parentNumber,
			Sub:        j + 1,
			Slug:       slug,
			Title:      titleFor(it.chapter.body, slug),
			Parent:     parent,
			CustomPrev: it.chapter.prev,
			CustomNext: it.chapter.next,
		}
		if it.isDir {
			if it.chapter.file != "" {
				html, err := renderMarkdown(it.chapter.body, base)
				if err == nil {
					ch.Body = template.HTML(html)
				}
			}
			ch.Subs = buildChapterTree(it.subs, ch, parentNumber, base)
		} else {
			html, err := renderMarkdown(it.chapter.body, base)
			if err == nil {
				ch.Body = template.HTML(html)
			}
		}
		chapters = append(chapters, ch)
	}
	return chapters
}

func loadWithRules(input, title, author, base string, include, exclude []string) (*Book, error) {
	if input == "" {
		input = defaultInput
	}
	items, rootIndex, err := loadDir(input, "", include, exclude)
	if err != nil {
		return nil, fmt.Errorf("book: read content %s: %w", input, err)
	}

	base = normalizeBase(base)
	book := &Book{Title: title, Author: author, base: base, Chapters: make([]Chapter, 0, len(items))}
	if rootIndex != nil {
		html, err := renderMarkdown(rootIndex.body, base)
		if err != nil {
			return nil, fmt.Errorf("book: render %s: %w", rootIndex.file, err)
		}
		book.indexChapter = &Chapter{
			Title:      titleFor(rootIndex.body, "index"),
			Body:       template.HTML(html),
			CustomPrev: rootIndex.prev,
			CustomNext: rootIndex.next,
		}
	}
	for i, it := range items {
		slug := slugFor(it.name)
		book.Chapters = append(book.Chapters, Chapter{
			Number:     i + 1,
			Slug:       slug,
			Title:      titleFor(it.chapter.body, slug),
			CustomPrev: it.chapter.prev,
			CustomNext: it.chapter.next,
		})
		ch := &book.Chapters[len(book.Chapters)-1]
		if it.isDir {
			if it.chapter.file != "" {
				html, err := renderMarkdown(it.chapter.body, base)
				if err != nil {
					return nil, fmt.Errorf("book: render %s: %w", it.chapter.file, err)
				}
				ch.Body = template.HTML(html)
			}
			ch.Subs = buildChapterTree(it.subs, ch, i+1, base)
			if it.chapter.file == "" {
				ch.Body = template.HTML(book.autoSubList(ch))
			}
			var populateAutoList func(c *Chapter)
			populateAutoList = func(c *Chapter) {
				if len(c.Subs) > 0 && c.Body == "" {
					c.Body = template.HTML(book.autoSubList(c))
				}
				for _, sub := range c.Subs {
					populateAutoList(sub)
				}
			}
			populateAutoList(ch)
		} else {
			html, err := renderMarkdown(it.chapter.body, base)
			if err != nil {
				return nil, fmt.Errorf("book: render %s: %w", it.chapter.file, err)
			}
			ch.Body = template.HTML(html)
		}
	}
	flat := book.flattened()
	for i, c := range flat {
		if i > 0 {
			c.Prev = flat[i-1]
		}
		if i < len(flat)-1 {
			c.Next = flat[i+1]
		}
	}
	return book, nil
}

// isIndex reports whether name is a directory chapter index file or root index file.
// Matches index.md, _index.md, 00-index.md, 0-index.md, etc.
func isIndex(name string) bool {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	_, rest := splitOrder(stem)
	clean := strings.Trim(rest, "-_")
	return stem == "index" || stem == "_index" || clean == "index"
}

// parseFrontmatterAndBody strips the leading BOM and YAML frontmatter block
// and unmarshals metadata fields (order, weight, prev, next).
func parseFrontmatterAndBody(raw []byte) (meta frontmatterMeta, body []byte) {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	text := strings.TrimLeft(string(raw), " \t\r\n")
	if !strings.HasPrefix(text, "---") {
		return meta, raw
	}
	rest := text[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return meta, raw
	}
	fmText := rest[:idx]
	_ = yaml.Unmarshal([]byte(fmText), &meta)

	after := strings.TrimLeft(rest[idx+4:], "\r\n")
	if after == "" {
		return meta, []byte("\n")
	}
	return meta, []byte(after)
}

// stripFrontmatter removes a leading UTF-8 BOM and YAML frontmatter block
// (--- ... ---) so content files shared with pkg/web/ssg collections
// render clean.
func stripFrontmatter(body []byte) []byte {
	_, clean := parseFrontmatterAndBody(body)
	return clean
}
