# Package Reference: `markdown`

Import: `github.com/krewire/krewire/markdown`

## 1. Purpose & Architecture

The `markdown` package provides a shared, native zero-dependency Markdown-to-HTML rendering engine for Krewire documentation, books, and static sites. Both `book` and `web/ssg` rely on it so that documentation sites can start as lightweight Markdown files and progressively enhance into full SSG applications without re-parsing or duplicating dependencies.

```text
markdown/
├── markdown.go   # Render, RenderSafe, RenderWithBase, StripScriptURLs, BufferPool
└── toc.go        # Heading extraction, anchor generation & Table of Contents parser
```

---

## 2. Main API Reference

- **`markdown.Render(src []byte) (string, error)`**: Converts GitHub Flavored Markdown (GFM) with AutoHeadingID into HTML. Strips script-bearing URLs (`javascript:`) while preserving trusted HTML tags for `.kiw` component embedding.
- **`markdown.RenderSafe(src []byte) (string, error)`**: Converts Markdown and strictly sanitizes the output against a safe allowlist. Strips `<script>`, `<iframe>`, and inline event handlers (`onerror`, `onload`). Use this for untrusted user-generated content (comments, issues, CMS inputs).
- **`markdown.RenderWithBase(src []byte, base string) (string, error)`**: Renders Markdown and automatically prefixes relative links and images with the specified base URL path (e.g. `/docs/` or `/id/docs/`).
- **`markdown.PrefixLinks(html string, base string) (string, error)`**: Rewrites relative HTML href/src attributes to align with sub-path base URLs.
- **`markdown.ExtractTOC(src []byte) ([]Heading, error)`**: Parses Markdown headings (`#`, `##`, `###`), generates slugified anchor IDs, and returns a structured Table of Contents hierarchy.
- **`markdown.GetBuffer() *bytes.Buffer` / `PutBuffer(buf *bytes.Buffer)`**: High-performance buffer pool reducing garbage collector pressure during bulk site generation.

---

## 3. Usage Examples

### Standard Site Rendering with Base Path

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/markdown"
)

func main() {
    doc := []byte(`# Getting Started

Welcome to [Installation](./02-install.md). Check out our [architecture](architecture.md).
`)

    html, err := markdown.RenderWithBase(doc, "/docs/")
    if err != nil {
        panic(err)
    }

    // Relative links are automatically prefixed to "/docs/02-install.md"
    fmt.Println(html)
}
```

### Safe Rendering for Untrusted Input

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/markdown"
)

func RenderUserComment(commentMarkdown string) string {
    // Strips malicious scripts and dangerous attributes
    safeHTML, err := markdown.RenderSafe([]byte(commentMarkdown))
    if err != nil {
        return "<p>Error rendering content</p>"
    }
    return safeHTML
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: `RenderSafe` eliminates stored Cross-Site Scripting (XSS) vectors by stripping script injection attempts. `Render` blocks `javascript:` pseudo-protocols across all link definitions.
- **Sustainable (S2)**: Pure Go implementation; sync.Pool memory recycling avoids memory spikes when compiling books with hundreds of pages on a minimal VPS.
- **Scalable (S3)**: Leaf package with zero inward dependencies; shared identically across CLI, SSG, book compiler, and web workloads.
