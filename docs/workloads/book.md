# Workload: Technical Book & Docs (`book`)

The `book` workload powers multi-chapter technical documentation, handbooks, API guides, and reference manuals using Krewire's native manuscript compiler (`pkg/book`) and markdown renderer (`pkg/markdown`).

It produces fast, accessible, offline-capable static documentation sites with automated table of contents, sidebar hierarchies, and client-side modal search — with zero Node.js/npm dependencies.

---

## 1. Architectural Overview & Capabilities

- **Ordered Chapter Tree**: Discovers and sorts Markdown files by numerical prefix (e.g. `01-introduction.md`, `02-architecture/01-overview.md`).
- **Automated Navigation**: Emits hierarchical sidebar navigation, previous/next chapter pagination, and breadcrumb trails.
- **Client-Side Full-Text Search**: Generates an inverted index (`search-index.json`) during build for instant client-side modal search (`Cmd+K` / `Ctrl+K`).
- **Bilingual & Multi-Locale Support**: Supports localized sub-trees (`content/en/docs/`, `content/id/docs/`) with automated fallback routing.
- **Fluid Co-Location**: Can be mounted alongside landing pages (`site`) or administrative panels (`app`) inside a shared `.krewire/build` directory.

---

## 2. Standard Directory Layout

Scaffolded via `kiw new <book-name> --book` or `kiw init --book`:

```text
my-handbook/
├── krewire.yaml          # Project configuration manifest
├── layouts/              # (Optional) Custom .kiw layout overrides
│   └── Book.kiw
└── content/              # Documentation source tree
    └── docs/
        ├── 01-overview/
        │   ├── 01-mission.md
        │   └── 02-architecture.md
        ├── 02-guides/
        │   ├── 01-installation.md
        │   └── 02-quickstart.md
        └── 03-api-reference.md
```

---

## 3. Configuration Contract (`krewire.yaml`)

```yaml
project:
  name: "krewire-handbook"
  kind: "book"
  version: "0.2.0"

title: "Krewire Handbook"
author: "Krewire Community"
input: "content"
output: ".krewire/build"

theme:
  default: "auto"        # auto | light | dark
  toggle: true           # Enable theme switcher

book:
  mount: "/docs"         # Output URL sub-path prefix
  toc: true              # Generate root Table of Contents index
  search: true           # Generate client search index
  pagination: true       # Render prev/next chapter footer links
```

---

## 4. Development & Build Pipelines

```bash
# 1. Preview book locally with live reload and WebSocket HMR
kiw serve

# 2. Build production static HTML documentation
kiw build --minify

# 3. Deploy to production static hosting / GitHub Pages
kiw deploy --target gh-pages
```

### Build Artifacts Produced

The compiler generates clean, standalone HTML pages:
- `.krewire/build/docs/01-overview/01-mission/index.html`
- `.krewire/build/docs/search-index.json`
- `.krewire/build/assets/krewire.css`
- `.krewire/build/assets/krewire.js`

---

## 5. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Contextual HTML auto-escaping prevents script injection; script-bearing links (`javascript:`) are automatically stripped by `pkg/markdown`.
- **Sustainable (S2)**: Requires zero Node.js runtime or complex documentation generators (Docusaurus, VitePress, GitBook); compiles in pure Go under 500ms.
- **Scalable (S3)**: Transitions smoothly from a standalone handbook into a multi-language developer portal (`site` + `book` hybrid) without restructuring chapter content.
