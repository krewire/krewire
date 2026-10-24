# Workload: Technical Book & Docs (`book`)

The `book` workload powers multi-chapter technical documentation, books, and reference manuals using Krewire's native book compiler (`pkg/book`).

---

## 1. Key Capabilities

- **Content Hierarchy**: Reads ordered Markdown chapters from `content/` (default), including nested subchapters.
- **Book Navigation & Search**: Generates a table of contents, sidebar, breadcrumbs, previous/next links, and an offline client search index.
- **Single-Output HTML**: Produces a static, book-shaped HTML site; EPUB and PDF export are not implemented.
- **Hybrid Placement**: `book.mount` and `book.toc` let a book share `.krewire/build` with an `ssg` site.

---

## 2. Directory Layout

The `kiw init --book` scaffold writes:

```text
my-book/
├── krewire.yaml
└── content/
    └── docs/
        ├── 01-introduction.md
        └── 02-getting-started.md
```

Kind detection recognizes a `content/` directory. A different source directory can be selected with the top-level `input:` key.

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-book
  kind: book

title: "Krewire Handbook"
author: "Krewire Contributors"
input: "content"
output: ".krewire/build"
theme:
  default: "auto"      # auto | light | dark

book:
  mount: "/docs"       # optional sub-path in the output
  toc: true            # generated root table of contents
```

---

## 4. Development & Build

```bash
# Preview book locally with live reload
kiw serve

# Build book into the output directory
kiw build

# Stage deployable artifacts
kiw deploy
```
