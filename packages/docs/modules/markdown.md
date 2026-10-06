# `markdown`

Import: `github.com/krewire/libs/markdown`

## Purpose

Shared Goldmark-based Markdown-to-HTML rendering for Krewire documentation, books, and static sites.

## Main API

- `Render(source)` renders Markdown to HTML.
- `RenderWithBase(source, base)` renders and prefixes eligible links with a base path.
- `PrefixLinks(html, base)` applies link prefixing to rendered HTML.

## Example

```go
html, err := markdown.RenderWithBase(source, "/docs/")
if err != nil { return err }
_ = html
```

## Design boundary

`markdown` is a rendering leaf package. It does not load files, serve HTTP, or decide application navigation. Callers own input loading, output storage, and content policy.
