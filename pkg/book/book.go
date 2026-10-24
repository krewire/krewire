// Package book assembles a Markdown content tree into book-shaped websites.
// The content directory is an ordered tree of Markdown files and
// subdirectories; the builder renders it into pages and assets that can be
// exported to disk or served over HTTP. It is intentionally lightweight and
// depends only on pkg/markdown and stdlib — not on pkg/web or pkg/web/ssg — so
// it can be depended on alongside those packages progressively.
//
// File responsibilities: chapter.go (chapter model), load.go (content
// loading), naming.go (ordering/slugs/titles), links.go (base-path links),
// pages.go (page-tree rendering), build.go (export orchestration),
// serve.go (HTTP serving), theme.go (local light/dark theming).
package book

// Book is loaded content, organized into an ordered set of chapters.
type Book struct {
	// Title is the book title.
	Title string
	// Author is the book author.
	Author string
	// Chapters holds the top-level chapters in reading order.
	Chapters []Chapter
	// base is the URL base prefix applied to generated links, e.g. "/guide/".
	base string
	// mount is the export sub-path ("", "docs", …) prefixed to every exported
	// page path; it mirrors base without the trailing slash.
	mount string
	// homeURL is the URL the brand links to; defaults to "/" when mounted or base.
	homeURL string
	// navLinks holds optional navbar links.
	navLinks []Link
	// footerText holds optional footer text.
	footerText string
	// version holds the product version shown in the credit line.
	version string
	// theme, when set, enables the light/dark theme switcher.
	theme *Theme
	// noRootTOC suppresses the generated root TOC page so a book can share
	// one output with an ssg site whose landing page owns "/". Set via
	// Config.NoRootTOC.
	noRootTOC bool
	// indexChapter holds the root index page content when index.md is present
	// at the root of the content directory.
	indexChapter *Chapter
	// extraCSS holds additional stylesheet URLs linked on
	// every page. Set from Config.ExtraCSS.
	extraCSS []string
	// extraJS holds additional script URLs loaded at the end of body on every
	// page. Set from Config.ExtraJS.
	extraJS []string
}

// ExtraCSS returns the additional stylesheet URLs linked on
// every page. Templates read it through pageData.
func (b *Book) ExtraCSS() []string { return b.extraCSS }

// ExtraJS returns the additional script URLs loaded at the end of body on every
// page. Templates read it through pageData.
func (b *Book) ExtraJS() []string { return b.extraJS }

// Link is a named navigation target shown in the navbar.
type Link struct {
	Text string
	URL  string
}

// flattened returns all pages in reading order: each chapter page followed by
// its subchapters.
func (b *Book) flattened() []*Chapter {
	var flat []*Chapter
	for i := range b.Chapters {
		flat = append(flat, &b.Chapters[i])
		flat = append(flat, b.Chapters[i].Subs...)
	}
	return flat
}
