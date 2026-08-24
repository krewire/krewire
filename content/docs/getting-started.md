---
title: "Getting Started"
description: "From zero to site in 30 seconds"
date: "2026-08-24"
---

# Getting Started

Krewire is Go-first. One CLI `kiw`, one config `krewire.yaml`.

```bash
kiw new my-site --site
cd my-site
kiw build
kiw serve
```

## File-based routing

```
pages/index.kiw          → /
pages/docs/index.kiw     → /docs
pages/docs/[slug].kiw    → /docs/:slug  (from content/docs/*.md)
components/Hero.kiw      → {{component "Hero" .}}
layouts/Base.kiw         → wraps {{.Content}}
public/logo.svg          → copied verbatim
```

No `go.mod` needed for `site` — `krewire.yaml` with `project.kind: site` is enough.

## Frontmatter is optional

Components like `Hero.kiw` need no `---` boilerplate:

```kiw
<section class="hero">
  <h1>{{.Title}}</h1>
</section>
<style>.hero{padding:40px}</style>
```

Pages with metadata keep it:

```kiw
---
title: Hello
layout: Base
---
<h1>{{.Title}}</h1>
```

## Next

- [Workloads](/docs/workloads) — the 8 kinds
- [.kiw DSL](/docs/dsl) — HTML · CSS · JS/TS · Go · Rust in one file
