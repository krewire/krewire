# Krewire — krewire/krewire

Source for **https://krewire.github.io/** — the unified Krewire site (landing + docs), **100% built with Krewire** (`site` workload, file-based `.kiw`).

- **Stack:** `krewire.yaml` (kind `site`, `base:"/"`) + `pages/*.kiw` + `components/*.kiw` + `layouts/*.kiw` + `content/docs/*.md` → `krewire build` → `site/` (no `go.mod` needed per `KWF-DF3PL`).
- **Design:** Inspired by `laravel.com` — sparse hero with code snippet, 8 workload cards, ecosystem strip, docs sidebar. Theme toggle (`auto`/`light`/`dark`) via `localStorage`, scoped CSS (`data-kiw-*`), `framework/ui` vars (`--color-primary` `#00c853`).
- **Version:** `v0.1.0` (all repos start at `v0.1.0`).
- **Spec:** `docs/specs/KRW-SITE-X7K9Q-landing-site.md` (broad scope: landing + docs, not narrow).

## Quick start

```bash
# from this repo (no go.mod needed)
kiw build        # → site/
kiw serve        # preview at http://localhost:8080

# file-based routing
pages/index.kiw          → /
pages/docs/index.kiw     → /docs
pages/docs/[slug].kiw    → /docs/:slug  (from content/docs/*.md)
```

## Deploy

- **Source:** `krewire/krewire` `main` (this repo)
- **Built artifact:** pushed to `krewire/krewire` `gh-deploy` (site at root, `.nojekyll`)
- **Live:** `krewire/krewire.github.io` `gh-pages` → `https://krewire.github.io/` (org pages, `base:"/"`)

Pushing built site to `gh-deploy` triggers `.github/workflows/deploy.yml` (on `gh-deploy`) which deploys to `krewire.github.io`.

## Structure (not too scoped)

```
krewire.yaml          # kind: site, title, nav, footer, theme
pages/                # file-based routes
layouts/              # Base (shell) + Docs (sidebar)
components/           # Hero, FeatureCard, CodeWindow, Ecosystem, Callout, DocNav (frontmatter-free)
content/docs/         # Markdown collections (getting-started, workloads, dsl)
public/               # favicon.svg, copied verbatim
```

## Ecosystem

Source lives across `krewire/*`: `framework`, `libs`, `mdbind`, `guild`, `kiw`. This site dogfoods `framework/web/ssg` + `framework/dsl` (`<markdown>`, `dict` helper, optional frontmatter) and validates the `site` path end-to-end.
