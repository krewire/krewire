package book

import (
	"fmt"
	"html/template"
	"strconv"
)

// Chapter is one manuscript unit, named by its slug and rendered from Markdown.
// A chapter with subchapters mirrors a part of a physical book: the chapter
// page leads, and its Subs follow as "N.M" subchapters.
type Chapter struct {
	// Number is the 1-based reading order index of the top-level chapter.
	Number int
	// Sub is the 1-based subchapter index within the parent chapter, or 0 for
	// a top-level chapter.
	Sub int
	// Slug is the URL-safe identifier derived from the filename.
	Slug string
	// Title is the chapter title derived from the first H1 heading or slug.
	Title string
	// Body is the rendered HTML body.
	Body template.HTML
	// Prev and Next point to adjacent pages in reading order.
	Prev *Chapter
	Next *Chapter
	// CustomPrev and CustomNext hold explicit frontmatter next/back overrides.
	CustomPrev string
	CustomNext string
	// Subs holds the chapter's subchapters in reading order; empty for a
	// plain chapter.
	Subs []*Chapter
	// Parent points to the containing chapter, or nil for top-level chapters.
	Parent *Chapter
}

// Path returns the chapter page URL: /{slug} for top-level chapters and
// /{parent}/{slug} for subchapters, mirroring the manuscript tree. URLs stay
// extensionless — each page is emitted as a sibling .html file served at that
// address — and are prefixed with the book's base path by Book.link.
func (c Chapter) Path() string {
	if c.Parent != nil {
		return "/" + c.Parent.Slug + "/" + c.Slug
	}
	if c.Slug == "" || c.Slug == "index" {
		return "/"
	}
	return "/" + c.Slug
}

// Label returns the numeric label shown in the table of contents: "7" for a
// top-level chapter and "7.1" for a subchapter.
func (c Chapter) Label() string {
	if c.Sub > 0 {
		return fmt.Sprintf("%d.%d", c.Number, c.Sub)
	}
	return strconv.Itoa(c.Number)
}
