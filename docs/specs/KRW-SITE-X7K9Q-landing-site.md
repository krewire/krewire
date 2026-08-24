# Specification — Krewire Site (Landing + Documentation)

| Field  | Value |
| ------ | ----- |
| SpecID | KRW-X7K9Q |
| Title  | Krewire Site — Landing + Documentation, 100% Ecosystem Dogfood |
| Status | Draft |
| Date   | 2026-08-24 |
| Author | Krewire Contributors |
| Domain | Site — Landing + Docs |

## 1. Summary

Build `krewire/krewire` (public repo) as the **unified site** for the Krewire ecosystem — **landing + documentation** — **100% built with the ecosystem itself** (file-based `.kiw` DSL, `krewire.yaml`, `pages/`/`components/`/`layouts/`/`content/`/`public/` → `krewire build` → `site/`), inspired by `laravel.com`'s clarity but Go-native and docs-ready. The site proves the `site` workload is production-ready for both marketing and long-form docs, validates the static-site pipeline (file-based routing, scoped CSS, theme toggle, asset hashing, content collections), and is published to `https://krewire.github.io/` (via `krewire/krewire.github.io`) — not too narrowly scoped to a one-page landing.

## 2. Background & Context

`framework/web/ssg` already builds file-based sites from `pages/*.kiw` with scoped CSS and content collections (`KWF-DF3PL`, `KWF-PT8OD`, `KWF-D57UK`). Smoke-tests (`web/ssg/testdata/landing`, `web/ssg/testdata/mount`) pass, and `kiw build` dispatch via `pages/` works after the `findRoot` fix for site-kind without `go.mod`. The ecosystem lacks a dogfooded public presence that serves both **first impression** and **continued learning**; `laravel.com` shows the bar: sparse hero with code snippet, feature grid, product pillars, frontend support, package grid, and deep docs navigation. Krewire mirrors this with `kiw new`, `kiw build`, 8 workload kinds (app/cli/site/book/worker/service/infra/runtime), and Go-first ergonomics — but generalizes beyond a marketing page: the same engine will host guides, specs, and API docs.

Inspiration (not plagiarism): `laravel.com` distributes **laravel new**, code snippet, product pillars (Cloud/Forge/Vapor…), Frontend (React/Vue via Inertia), Packages grid, Community, and enterprise logos. Krewire mirrors this with `kiw new`, `kiw build`, 8 workload kinds, and Go-first ergonomics, plus a docs system inspired by Laravel's `/docs` section (sidebar nav, prose, code windows, callouts) — all from the same file-based pipeline.

## 3. Problem Statement

- No public Krewire presence; visitors cannot differentiate the 8 kinds or see the "one CLI" promise.
- The `site` path has not been exercised end-to-end with a real design system, theme toggle, and gh-pages deploy, so regressions stay hidden.
- Without dogfooding, spec gaps (e.g., `site` requiring `go.mod`) persist.

## 4. Goals & Non-Goals

### Goals
- G1 — Ship `krewire/krewire` public repo as the **source** for the unified site; publish the built `site/` to `krewire/krewire.github.io` so `https://krewire.github.io/` (root, no `/krewire/` prefix) serves the site, built solely via `kiw build`.
- G2 — Landing surface: elegant display typography, code snippet (`kiw new my-app && kiw build`), CTAs (Get Started → docs, View on GitHub), feature cards, ecosystem strip — not over-scoped to a single hero.
- G3 — Documentation surface: file-based docs under `/docs` (getting-started, workload matrix, `.kiw` DSL) with sidebar nav, prose, code windows, callouts — same engine as landing, proving the site is **general-purpose**.
- G4 — Reusable system: `layouts/Base.kiw` + docs-aware layout (`Docs.kiw`) and generic components (`Hero`, `FeatureCard`, `Ecosystem`, `CodeWindow`, `Section`, `Prose`, `Callout`, `DocNav`) — landing and docs share the same design tokens.
- G5 — Footer, nav, theme switcher (auto/light/dark, `localStorage`, no JS required for first paint), responsive, accessible, ≤ 50KB CSS+JS gzipped.

### Non-Goals
- NG1 — No client interactivity beyond theme toggle in v0.1.0 (static `site`, no WASM mounts yet — hydrations are next phase).
- NG2 — No CMS, blog, or analytics in v0.1.0 (but content collections `content/` are validated for future posts; blog reuses the same pipeline).
- NG3 — No backend; all content is Markdown/`.kiw` files — the site must stay hostable on static GitHub Pages.

### 4.5 Assumptions & Constraints

| ID | Assumption / Constraint | Type |
|----|-------------------------|------|
| A1 | Host is GitHub Pages (`gh-pages` branch, root) | Assumption |
| A2 | `krewire build` on site-kind requires only `krewire.yaml` + `pages/` (`go.mod` not required) | Constraint |
| C1 | Config in `krewire.yaml` only; output to `site/` | Constraint |
| C2 | Styles scoped by default (`data-kiw-*`), `:root` stays global | Constraint |

## 5. Requirements

### Functional Requirements

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| KRW-LAND-001 | Repo `krewire/krewire` is public; `main` holds sources (`krewire.yaml`, `pages/`, `components/`, `layouts/`, `public/`, `docs/specs/`); `gh-pages` holds built `site/` (index.html, assets) | Must | MUST |
| KRW-LAND-010 | `krewire.yaml` declares `project.kind: site`, `title`, `description`, `theme` (light #00c853 / dark #5cff8e per `framework/ui`) | Must | MUST |
| KRW-LAND-011 | `pages/index.kiw` is the landing; routes extensionless (`/` → `index.html`), no trailing slashes | Must | MUST |
| KRW-LAND-012 | `layouts/Base.kiw` provides HTML shell, `<head>` theme script (`localStorage krewire-theme`), nav, footer, content slot `{{.Content}}` | Must | MUST |
| KRW-LAND-020 | Components: `Hero.kiw` (title, subtitle, code window, CTAs), `FeatureCard.kiw`, `Ecosystem.kiw`, `CodeWindow.kiw`, `Section.kiw` — invocable via `{{component "Hero" .}}` | Must | MUST |
| KRW-LAND-021 | Styles: scoped by default, `:root` global for theme vars `--color-primary`, `--show-sun/moon` from `framework/ui`; collected to `assets/style.css` | Must | MUST |
| KRW-LAND-030 | Content: hero (tagline "One Go Framework. Every Workload." — Krewire's own positioning, not borrowed taglines, `kiw new my-app`, CTA), 8 workload cards, code snippet (file-based routing + .kiw DSL), ecosystem links, community placeholder | Must | MUST |
| KRW-LAND-040 | `public/` assets (favicon, logo) copied verbatim; no extra toolchain | Must | MUST |
| KRW-LAND-050 | `krewire build` in repo root builds deterministic `site/`; `krewire serve` previews locally | Must | MUST |
| KRW-LAND-060 | Deploy: `gh-pages` branch contains built site at root; `krewire/.nojekyll` present; GitHub Pages serves from `gh-pages` | Must | MUST |

### Non-Functional Requirements

| ID | Category | Requirement |
|----|----------|-------------|
| NFR1 | Perf | First contentful paint without JS; total CSS+JS ≤ 50KB gzipped |
| NFR2 | A11y | Semantic HTML, landmark nav/main/footer, color contrast AA |
| NFR3 | Gates | `kiw build` on fixture passes; `site/index.html` contains hero strings + `data-kiw-component` markers |

## 6. Detailed Design

### 6.1 Architecture

- Kind `site` file-based: `pages/index.kiw` + `components/*.kiw` + `layouts/Base.kiw` + `public/*` → `ssg.LoadFromDir` → `site/`
- `krewire.yaml` metadata-only (no `ssg:`) — write-once
- `layouts/Base.kiw` wraps `{{.Content}}`; `{{component "Hero" .}}` injects scoped fragments
- `framework/web/ssg` CSS scoping (`[data-kiw-component]`, `[data-kiw-layout]`), `framework/ui` theme vars + toggle CSS

### 6.2 Routes & Data

- `/` from `pages/index.kiw`; future `/docs/*` maps to `pages/docs/*.kiw` (file-based)
- Page data merges site metadata + frontmatter (`title`, `description`)

### 6.3 System Style (laravel.com-inspired, Go-native)

- Typography: display serif for hero title (70px, -0.04em tracking), sans for body (Inter/system)
- Grid: 12-col, max-w 1280, gapped cards (rounded 16px, subtle border/shadow)
- Hero: left copy (title, subtitle, CTAs, terminal code), right decorative code window with `.kiw` snippet
- Code window: dark bg #0a0a0a, green accent #00c853, mono 13px, rounded, shadow
- Feature cards: icon emoji/box, title, one-line desc, hover lift
- Ecosystem: mono list with repo links

## 7. Testing & Verification Plan

- Unit: `krewire build` in repo root → `site/index.html` contains "One Go Framework", workload names, `data-kiw-component="Hero"`, `data-kiw-layout="Base"`; `site/assets/style.css` contains scoped selectors; `curl` readable without JS
- Gates: `gofmt -l .` (no Go in repo, N/A), `kiw build` deterministic, `site/` size check

## 8. Rollout

- Phase: Spec draft → scaffold `krewire/krewire` (krewire.yaml + pages/layouts/components/public) → `kiw build` → verify `site/` → push `main` → copy `site/` to `gh-pages` branch → enable Pages (gh-pages, root) → `curl https://krewire.github.io/krewire/` verification

## 9. Open Questions

- Future `/docs` section maps to `content/` collections or `manuscript/`? Deferred.

## 10. Success Criteria

- S1 — `https://krewire.github.io/krewire/` renders hero, 8 cards, ecosystem, footer with theme toggle (no JS required for first paint)
- S2 — `site/index.html` byte-identical on rebuild (deterministic)
- S3 — View-source shows `data-kiw-component`/`layout` markers, no inline YAML config

## 11. Related Specifications

| SpecID | Title | Relationship |
|--------|-------|--------------|
| KWF-DF3PL | File-Based Site Pipeline | Host |
| KWF-PT8OD | Static Site Generator | Host |
| KWF-0Z671 | UI Framework | Theme vars |

## 12. References

- `laravel.com` (inspiration, not clone): https://laravel.com
- `framework/web/ssg/testdata/landing` fixture
- `internal/docs/project-vision.md`
