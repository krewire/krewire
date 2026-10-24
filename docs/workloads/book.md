# Workload: Technical Book & Docs (`book`)

The `book` workload powers multi-chapter technical documentation, books, and reference manuals using Krewire's native book compiler (`pkg/book`).

---

## 1. Key Capabilities

- **Manuscript Hierarchy**: Automatically reads multi-chapter markdown books organized under `manuscript/`.
- **Book Navigation & Search**: Automatically generates table of contents, breadcrumbs, previous/next chapter navigation, and offline client search indexes.
- **Multi-Format Compilation**: Supports building single-page reading views, multi-page HTML documentation hubs, EPUB, and PDF distributions.

---

## 2. Directory Layout

```text
my-book/
├── krewire.yaml          # Project configuration (kind: book)
└── manuscript/           # Chapter markdown files
    ├── 00-intro.md
    ├── 01-getting-started.md
    ├── 02-architecture.md
    └── images/
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-book
  kind: book

book:
  source_dir: "manuscript"
  output_dir: ".krewire/build"
  title: "Krewire Handbook"
  author: "Krewire Contributors"
  theme: "default"
```

---

## 4. Development & Build

```bash
# Preview book locally with live reload
kiw serve

# Build book into output directory
kiw build

# Deploy staged book
kiw deploy
```
