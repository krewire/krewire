package book

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// Page is a renderable page for export or serve. Path is the extensionless
// URL (e.g. "/getting-started"); Render writes the HTML to w.
type Page struct {
	Path   string
	Render func(w io.Writer) error
}

// Pages builds the site's Page tree: the root landing page and one page
// per chapter. Exported page paths are always root-relative; only the links
// emitted into the rendered HTML carry the book's base path.
func (b *Book) Pages() ([]Page, error) {
	tpls, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	toc := b.chapterList(nil)
	pages := make([]Page, 0, len(b.Chapters)+1)
	if !b.noRootTOC {
		root := Page{
			Path: "/",
			Render: func(w io.Writer) error {
				if b.indexChapter != nil {
					data := b.pageData(b.indexChapter, toc)
					firstCh := (*Chapter)(nil)
					if flat := b.flattened(); len(flat) > 0 {
						firstCh = flat[0]
					}
					data.Prev = b.resolveLinkRef(b.indexChapter.CustomPrev, nil)
					data.Next = b.resolveLinkRef(b.indexChapter.CustomNext, firstCh)
					return tpls.ExecuteTemplate(w, "chapter.tmpl", data)
				}
				data := b.pageData(nil, toc)
				if flat := b.flattened(); len(flat) > 0 {
					data.First = &linkRef{Title: flat[0].Title, Link: b.link(flat[0].Path())}
					data.Pages = len(flat)
				} else {
					data.Pages = 0
				}
				return tpls.ExecuteTemplate(w, "index.tmpl", data)
			},
		}
		pages = append(pages, root)
	}
	for i, ch := range b.flattened() {
		data := b.pageData(ch, b.chapterList(ch))
		data.Crumbs = b.crumbs(ch)
		fallbackPrev := ch.Prev
		if fallbackPrev == nil && i == 0 && b.indexChapter != nil {
			fallbackPrev = b.indexChapter
		}
		data.Prev = b.resolveLinkRef(ch.CustomPrev, fallbackPrev)
		data.Next = b.resolveLinkRef(ch.CustomNext, ch.Next)
		pages = append(pages, Page{
			Path: ch.Path(),
			Render: func(w io.Writer) error {
				return tpls.ExecuteTemplate(w, "chapter.tmpl", data)
			},
		})
		if len(ch.Subs) > 0 {
			pages = append(pages, Page{
				Path: ch.Path() + "/index",
				Render: func(w io.Writer) error {
					return tpls.ExecuteTemplate(w, "chapter.tmpl", data)
				},
			})
		}
	}
	return pages, nil
}

// pageData assembles the shared template payload for a page.
func (b *Book) pageData(ch *Chapter, toc []tocEntry) pageData {
	version := b.version
	if version != "" {
		version = "v" + strings.TrimPrefix(version, "v")
	}
	homeURL := b.homeURL
	loc := b.locale()
	if homeURL == "" || homeURL == "/" {
		if loc != "" && loc != "en" {
			// A non-default locale lives under /<locale>/, so its home must
			// include the mount too — /<locale> alone is not an exported page.
			homeURL = "/" + loc + "/"
			if m := strings.Trim(b.mount, "/"); m != "" {
				homeURL += m + "/"
			}
		} else if b.mount != "" {
			homeURL = "/"
		} else {
			homeURL = b.base
		}
	}
	return pageData{
		BasePath:   b.base,
		HomeURL:    homeURL,
		Locale:     loc,
		Book:       b,
		Chapter:    ch,
		Toc:        toc,
		NavLinks:   b.localizedNavLinks(),
		FooterText: b.footerText,
		Version:    version,
		Theme:      b.theme,
		Stylesheet: b.stylesheetURL(version),
	}
}

// stylesheetURL is the URL of the book's own stylesheet, carrying the same
// cache-busting query the host site uses. Without it this one asset would be
// the only unversioned stylesheet on the page, so a reader could keep a stale
// book stylesheet after a release while every other asset refreshes.
func (b *Book) stylesheetURL(version string) string {
	u := b.base + StylesheetName
	if version == "" {
		return u
	}
	return u + "?v=" + strings.TrimPrefix(version, "v")
}

// autoSubList renders the auto-generated chapter page for a directory without
// an index file: a list of its subchapters with numeric labels.
func (b *Book) autoSubList(ch *Chapter) string {
	var sb strings.Builder
	sb.WriteString("<p class=\"lede\">This chapter has the following sections.</p>\n<ul class=\"sub-toc\">\n")
	for _, sub := range ch.Subs {
		link := b.link(sub.Path())
		fmt.Fprintf(&sb, "<li><a href=\"%s\"><strong>%s</strong> %s</a></li>\n", link, sub.Label(), template.HTMLEscapeString(sub.Title))
	}
	sb.WriteString("</ul>\n")
	return sb.String()
}

// pageData is the payload for the index and chapter page templates.
type pageData struct {
	BasePath   string
	HomeURL    string
	Locale     string
	Book       *Book
	Chapter    *Chapter
	Toc        []tocEntry
	Crumbs     []crumb
	Pages      int
	First      *linkRef
	Prev       *linkRef
	Next       *linkRef
	NavLinks   []Link
	FooterText string
	Version    string
	Theme      *Theme
	// Stylesheet is the full URL of the book's own stylesheet, including the
	// cache-busting query. Templates link this instead of assembling the path
	// from BasePath and the file name, so the name has a single owner.
	Stylesheet string
}
