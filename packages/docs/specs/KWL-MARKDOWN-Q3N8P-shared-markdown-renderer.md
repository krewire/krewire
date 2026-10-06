# Specification — Shared Markdown Renderer

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWL-MARKDOWN-Q3N8P                           |
| Title       | Shared Markdown Renderer (Goldmark)         |
| Status      | Draft                                       |
| Date        | 2026-08-25                                  |
| Author      | Krewire Contributors                         |
| Domain      | Markdown                                    |

## 1. Context

The ecosystem has two Markdown consumers: `mdbind` (book: `manuscript/*.md → .krewire/build`) and `framework/web/ssg + dsl` (site: collections + `<markdown>` in `.kiw`). Both previously re-implemented GFM rendering — `mdbind` with `yuin/goldmark` + `extension.GFM`, `framework` with `gomarkdown/markdown` + `CommonExtensions|AutoHeadingIDs`. Duplicate parsers cause drift (heading IDs, GFM tables/strike) and prevent a docs site from progressively enhancing from `book` to `site` without re-parsing.

`libs` is the workspace bottom (`go.work` `use ./libs`). A shared Package under `libs/markdown` can be depended on by both `mdbind` and `framework` simultaneously, keeping `mdbind` framework-free while allowing co-existence in a user project (e.g. `go get github.com/krewire/framework` + `go get github.com/krewire/mdbind` in the same `go.mod`).

## 2. Problem Statement

* No single source of truth for Markdown → HTML: two parsers, two link-rewriting logics (`base` prefix for book, none for ssg).
* `mdbind` should stay lightweight (no `framework/web` or `framework/ui`) but currently leaks `ui.Theme`/`web.Router` — progressive enhancement requires both modules be importable together without a cycle.
* Changing a Markdown rendering rule (e.g. adding `AutoHeadingID`) must be changed in two places.

## 3. Goals

* G1 — One Goldmark-based renderer (`extension.GFM` + `parser.WithAutoHeadingID`) shared by all Krewire code.
* G2 — Expose `Render(src []byte) (string, error)` and `RenderWithBase(src []byte, base string) (string, error)` that rewrites absolute `href="/x"`/`src="/x"` under `base` (e.g. `/guide/`), preserving extensionless form and idempotence for already-prefixed links.
* G3 — Keep `libs/markdown` stdlib-only except `goldmark`; no `framework` or `mdbind` imports.
* G4 — Allow `mdbind` and `framework` to be depended on together; a `book` project can later add `pages/*.kiw` + `ssg:` without rewrite (progressive: `manuscript/` and `ssg` build to same `.krewire/build`).

## 4. Non-Goals

* NG1 — Frontmatter parsing (belongs to `ssg/content.go` + `dsl/kiw.go`).
* NG2 — Theming, routing, or page model — those stay in `mdbind/book` and `framework/web`.
* NG3 — Supporting `gomarkdown` — Goldmark is the canonical renderer; `HrefTargetBlank` is not added by default (callers may post-process if needed).

## 5. Requirements

| ID          | Requirement | Priority |
| ----------- | ----------- | -------- |
| KWL-MD-001 | Provide `Package: libs/markdown` with `Render` using Goldmark `WithExtensions(GFM)` + `WithParserOptions(WithAutoHeadingID())`, deterministic. | Must |
| KWL-MD-002 | Provide `RenderWithBase` that calls `Render` then `PrefixLinks(html, base)` where `base` is normalized (`""`/`"/"` → no rewrite, `"/guide/"` → prefix). | Must |
| KWL-MD-003 | Provide `PrefixLinks(html, base string) string` (exported) that rewrites `href="/rest"`/`src="/rest"` to `href="/<prefix>/rest"` but leaves `href="/<prefix>/..."`, `href="/"` root, and protocol-relative `//` unchanged. | Must |
| KWL-MD-004 | `mdbind/book` must use `libs/markdown` for `renderMarkdown` and remove direct `goldmark` import; `framework/web/ssg/content.go` and `kiw/dsl/kiw.go` must use `libs/markdown` instead of `gomarkdown`. | Must |
| KWL-MD-005 | Remove `gomarkdown/markdown` direct requirement from `framework/go.mod` (kept transitively only if needed via other deps). | Must |
| KWL-MD-006 | `mdbind` must not import `framework/*`; public `book.Config.Theme` is local `book.Theme`/`Palette` (mirroring `ui.Theme` shape) and `Book.Handler() http.Handler` (stdlib `ServeMux`), not `*ui.Theme`/`*web.Router`. | Must |
| KWL-MD-007 | Default output for `mdbind/book.Build` is `.krewire/build` (aligned with `kiw` `config.DefaultOutput` and `framework/web/ssg`). | Must |
| KWL-MD-008 | A user `go.mod` can `require github.com/krewire/framework` + `require github.com/krewire/mdbind` together without import cycle or type conflict; `kiw build` builds both `manuscript/` and `ssg` (pages/`ssg:`) into the same output when both are present (progressive). | Must |

## 6. Non-Functional Requirements

* NFR1 — `gofmt`, `go vet ./...`, `go test ./...` must pass in `libs`, `framework`, `mdbind`, `kiw`.
* NFR2 — `go.work` with `use ./libs` must resolve `libs/markdown` for single-repo clones (add `replace github.com/krewire/libs => ../libs` in `mdbind/go.mod` for fallback).
* NFR3 — Deterministic: identical `src` → identical HTML.

## 7. Success Criteria

* S1 — `go test ./libs/markdown` (or `book` using it) passes `TestPrefixLinks` vectors.
* S2 — `go vet ./...` passes in all four repos with `libs/markdown` imported by both consumers.
* S3 — A temp project with `manuscript/01-a.md` + `pages/index.kiw` + `go.mod` requiring both `framework` and `mdbind` builds to `.krewire/build` via `kiw build` producing both `index.html` (ssg) and `a.html` (book) without conflict.
* S4 — `kiw init --site` + `kiw init --book` in same project (sequential) leaves both `ssg:` and `manuscript/` and `kiw build` emits merged output.

## 8. Related Specifications

| SpecID    | Title |
| --------- | ----- |
| [KWM-FX9H2](https://github.com/krewire/mdbind/blob/main/docs/specs/KWM-BUILDER-FX9H2-mdbind-site-builder.md) | mdbind Site Builder |
| [KWF-PT8OD](https://github.com/krewire/framework/blob/main/docs/specs/KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator |
| [KWN-1QGI2](https://github.com/krewire/kiw/blob/main/docs/specs/KWN-BUILD-1QGI2-project-building.md) | Project Building (progressive: book+site) |

## 9. References

* `libs/markdown/markdown.go` — implementation.
* `KWL-ARCH-J2K9Q` — Scope levels (Package `libs/markdown`).
