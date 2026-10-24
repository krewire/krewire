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

// Source is a loaded manuscript file.
type source struct {
	file     string
	title    string
	body     []byte
	order    int
	hasOrder bool
	prev     string
	next     string
}

// item is one top-level manuscript unit: a chapter file or a directory that
// becomes a chapter with subchapters.
type item struct {
	// name is the entry name used for ordering and slugging.
	name string
	// chapter holds the chapter body; for directories this is the optional
	// index.md/_index.md, empty when the chapter page is auto-generated.
	chapter source
	// subs holds the subchapter sources for a directory unit.
	subs []source
	// order and hasOrder are derived from frontmatter
	order    int
	hasOrder bool
}

// Load reads a manuscript directory into a Book in reading order, served from
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

// LoadWithBase reads a manuscript directory into a Book in reading order. Top
// level Markdown files become chapters; a directory becomes a chapter whose
// subchapters are the Markdown files inside it, ordered by numeric prefix. An
// index.md or _index.md inside a directory becomes the chapter page body;
// without one, the chapter page lists its subchapters automatically. The base
// is the URL prefix the site will be served under, e.g. "/guide/".
// YAML frontmatter (leading --- block) is tolerated and stripped so content
// files stay compatible with framework/web/ssg collections.
func LoadWithBase(input, title, author, base string) (*Book, error) {
	return loadWithRules(input, title, author, base, nil, nil)
}

func loadWithRules(input, title, author, base string, include, exclude []string) (*Book, error) {
	if input == "" {
		input = defaultInput
	}
	entries, err := os.ReadDir(input)
	if err != nil {
		return nil, fmt.Errorf("book: read manuscript %s: %w", input, err)
	}

	var items []item
	var rootIndex *source
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			it := item{name: name}
			subEntries, err := os.ReadDir(filepath.Join(input, name))
			if err != nil {
				return nil, fmt.Errorf("book: read %s: %w", name, err)
			}
			for _, se := range subEntries {
				sn := se.Name()
				if se.IsDir() || !strings.HasSuffix(sn, ".md") {
					continue
				}
				if !accepts(include, exclude, name+"/"+sn) {
					continue
				}
				data, err := os.ReadFile(filepath.Join(input, name, sn))
				if err != nil {
					return nil, fmt.Errorf("book: read %s: %w", filepath.Join(name, sn), err)
				}
				meta, cleanBody := parseFrontmatterAndBody(data)
				order, hasOrder := meta.resolveOrder()
				if isIndex(sn) {
					it.chapter = source{
						file:     sn,
						body:     cleanBody,
						order:    order,
						hasOrder: hasOrder,
						prev:     meta.resolvePrev(),
						next:     meta.Next,
					}
					if hasOrder {
						it.order = order
						it.hasOrder = true
					}
					continue
				}
				it.subs = append(it.subs, source{
					file:     sn,
					body:     cleanBody,
					order:    order,
					hasOrder: hasOrder,
					prev:     meta.resolvePrev(),
					next:     meta.Next,
				})
			}
			sort.Slice(it.subs, func(i, j int) bool {
				if it.subs[i].hasOrder && it.subs[j].hasOrder {
					if it.subs[i].order != it.subs[j].order {
						return it.subs[i].order < it.subs[j].order
					}
				} else if it.subs[i].hasOrder {
					return true
				} else if it.subs[j].hasOrder {
					return false
				}
				ni, si := splitOrder(it.subs[i].file)
				nj, sj := splitOrder(it.subs[j].file)
				if ni != nj {
					return ni < nj
				}
				return si < sj
			})
			if it.chapter.file != "" || len(it.subs) > 0 {
				items = append(items, it)
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if !accepts(include, exclude, name) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(input, name))
		if err != nil {
			return nil, fmt.Errorf("book: read %s: %w", name, err)
		}
		meta, cleanBody := parseFrontmatterAndBody(data)
		order, hasOrder := meta.resolveOrder()
		if isIndex(name) {
			rootIndex = &source{
				file:     name,
				body:     cleanBody,
				order:    order,
				hasOrder: hasOrder,
				prev:     meta.resolvePrev(),
				next:     meta.Next,
			}
			continue
		}
		items = append(items, item{
			name:     name,
			order:    order,
			hasOrder: hasOrder,
			chapter: source{
				file:     name,
				body:     cleanBody,
				order:    order,
				hasOrder: hasOrder,
				prev:     meta.resolvePrev(),
				next:     meta.Next,
			},
		})
	}

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
		if it.chapter.file != "" {
			html, err := renderMarkdown(it.chapter.body, base)
			if err != nil {
				return nil, fmt.Errorf("book: render %s: %w", it.chapter.file, err)
			}
			ch.Body = template.HTML(html)
		} else {
			ch.Body = template.HTML(book.autoSubList(ch, it.subs))
		}
		for j, s := range it.subs {
			slug := slugFor(s.file)
			sub := Chapter{
				Number:     i + 1,
				Sub:        j + 1,
				Slug:       slug,
				Title:      titleFor(s.body, slug),
				Parent:     ch,
				CustomPrev: s.prev,
				CustomNext: s.next,
			}
			html, err := renderMarkdown(s.body, base)
			if err != nil {
				return nil, fmt.Errorf("book: render %s: %w", s.file, err)
			}
			sub.Body = template.HTML(html)
			ch.Subs = append(ch.Subs, &sub)
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
// (--- ... ---) so content files shared with framework/web/ssg collections
// render clean.
func stripFrontmatter(body []byte) []byte {
	_, clean := parseFrontmatterAndBody(body)
	return clean
}
