# `book` — Markdown Content Compiler

`github.com/krewire/krewire/pkg/book` assembles an ordered Markdown content tree
into a book-shaped static website: chapter pages, sidebar navigation, a table of
contents, previous/next links, a client-side search index, and a versioned
stylesheet.

It is the engine behind the `book` workload kind and depends only on
`pkg/markdown`, `pkg/config`, and the standard library — never on `pkg/web` or
`pkg/web/ssg`.

## Quick start

```go
b, err := book.Load("content", "Krewire Handbook", "Krewire Contributors")
if err != nil {
	return err
}

_, err = book.Build(book.Config{
	Title:  b.Title,
	Author: b.Author,
	Input:  "content",
	Output: ".krewire/build",
})
```

## Layout

- `chapter.go` — chapter model
- `load.go` — content loading and frontmatter
- `naming.go` — ordering, slugs, and titles
- `links.go` — base-path link prefixing
- `pages.go` — page-tree rendering
- `build.go` — export orchestration
- `serve.go` — HTTP serving
- `search.go` — client search index
- `theme.go` — light/dark theming

## Documentation

- [Package reference](../../docs/pkg/mod/book.md)
- [Book workload guide](../../docs/workloads/book.md)
