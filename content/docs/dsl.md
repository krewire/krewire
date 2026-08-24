---
title: ".kiw DSL"
description: "One file per component"
date: "2026-08-24"
---

# .kiw DSL

A `.kiw` file is frontmatter (optional) + HTML/Markdown + scoped CSS + tiered scripts.

```kiw
<h1>{{.Title}}</h1>

<style>h1{color:var(--primary)}</style>

<script lang="go" server>
package hello
import "context"
func Load(ctx context.Context, p Props) (Props, error) { return p, nil }
</script>

<markdown>
# Prose

Write docs **inside** `.kiw` without leaving the file.
- HTML for structure
- Markdown for prose
- CSS scoped by default
</markdown>
```

Components can start directly with HTML — frontmatter is only for pages that need metadata or props.

Use `<markdown>...</markdown>` when you want prose inside `.kiw` without leaving the file; it renders to HTML at parse time and preserves directives like `{{.Title}}`. For long-form collections, keep Markdown in `content/docs/*.md` and render via `pages/docs/[slug].kiw` with `{{.Content}}`.

## Props

```yaml
props:
  initial: { type: int, default: 0 }
  label: { type: string, default: "Count" }
```

Generates `Props` for Go/TS/Rust from one schema.
