# `book`

Import: `github.com/krewire/krewire/pkg/book`

## Purpose

Assembles an ordered Markdown content tree into a book-shaped website: chapter
pages, sidebar navigation, a table of contents, prev/next links, a client-side
search index, and a versioned stylesheet. It is the engine behind the `book`
workload kind and the `docs/` output of `krewire.com`.

The package depends only on `pkg/markdown` and the standard library — never on
`pkg/web` or `pkg/web/ssg` — so a book can be adopted progressively alongside a
site without pulling in the HTTP framework.

## Main API

- `book.Load(input, title, author string) (*Book, error)`: Loads a content
  directory into ordered chapters.
- `book.LoadWithRules(input, title, author, base string, include, exclude []string) (*Book, error)`: Loads with glob-based content selection.
- `book.Build(cfg Config) ([]string, error)`: Renders the content to a static
  site and returns the created paths, sorted for determinism.
- `book.Serve(b *Book, addr string) error` / `book.ServeConfig(cfg Config, addr string) error`:
  Serve a loaded book over HTTP.
- `book.LiveHandler(cfg Config) http.Handler`: Serves the book and rebuilds on
  request for local development.
- `(*Book).BuildSearchIndex() SearchIndex` / `(*Book).BuildNavManifest() NavManifest`:
  Generate the search and navigation payloads embedded in the output.
- `book.LoadConfigFile(path string) (*Config, error)` / `book.FindConfigFile(dir string) string`:
  Read `book.yaml`, `book.yml`, or the compatible subset of `krewire.yaml`.

`Config` controls `Title`, `Author`, `Input`, `Output`, `BasePath`, `MountPath`
(place the book under `/docs/` in a shared output), `NoRootTOC`, `Include` /
`Exclude` globs, `NavLinks`, `FooterText`, `Version`, `Theme`, `ExtraCSS`, and
`ExtraJS`.

## Example

```go
package main

import (
	"log"

	"github.com/krewire/krewire/pkg/book"
)

func main() {
	b, err := book.Load("content", "Krewire Guide", "Krewire Contributors")
	if err != nil {
		log.Fatal(err)
	}

	if _, err := book.Build(book.Config{
		Title:  b.Title,
		Author: b.Author,
		Input:  "content",
		Output: ".krewire/build",
	}); err != nil {
		log.Fatal(err)
	}
}
```

## Design boundary

- **Progressive companion:** `MountPath` and `NoRootTOC` let a book share one
  output directory with an `ssg` site, so a landing page can own `/` while the
  content lives under `/docs/`.
- **Deterministic output:** chapters are ordered by frontmatter `order`/`weight`
  then slug; `Build` returns sorted paths so repeated builds are reproducible.
- **Downward only:** imports `pkg/markdown` and `pkg/config`; never imports
  `tools/kiw` or `pkg/web`.
