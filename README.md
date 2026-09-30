# Krewire — krewire/krewire

Source for the unified Krewire site (landing + docs), **100% built with Krewire** (`site` workload, file-based `.kiw`).

- **Stack:** `krewire.yaml` (kind `site`, `base:"/"`) + `pages/*.kiw` + `components/*.kiw` + `layouts/*.kiw` + `content/docs/*.md` → `krewire build` → `site/` (no `go.mod` needed per `KWF-DF3PL`).
- **Design:** Inspired by `laravel.com` — sparse hero with code snippet, 8 workload cards, ecosystem strip, docs sidebar. Theme toggle (`auto`/`light`/`dark`) via `localStorage`, scoped CSS (`data-kiw-*`), `framework/ui` vars (`--color-primary` `#39D353`).
- **Version:** `v0.1.0` — single source `krewire.yaml` `project.version`; injected as `.Version` into every page (badges/footer), never hardcoded in content.
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
- **Production:** built output will be deployed to the Krewire-hosted machine with Docker; domain configuration is managed separately.

The `main` branch is the source of truth. The production deployment is independent of `krewire/krewire.github.io`.

## Structure (not too scoped)

```
krewire.yaml          # devtool config: kind, build output, dev server, project metadata
pages/                # file-based routes
layouts/              # Base (shell) + Docs (sidebar)
components/           # Hero, FeatureCard, CodeWindow, Ecosystem, Callout, DocNav (frontmatter-free)
content/docs/         # Markdown collections (getting-started, workloads, dsl)
public/               # favicon.svg, copied verbatim
```

## Ecosystem

Source lives across `krewire/*`: `framework`, `libs`, `mdbind`, `boost`, `kiw`. This site dogfoods `framework/web/ssg` + `kiw/dsl` (`<markdown>`, `dict` helper, optional frontmatter) and validates the `site` path end-to-end.
