---
title: "DSL (.kiw)"
description: "In-depth guide to the .kiw component templating language: YAML frontmatter, Go html/template expressions, scoped CSS, client scripts, and embedded Markdown."
date: "2026-09-29"
---

# DSL (.kiw)

The **`.kiw` Component DSL** (`github.com/krewire/kiw/dsl`) is Krewire's unified, file-based component templating language. It brings modern Single-File Component (SFC) ergonomics—reminiscent of Astro, Svelte, or Vue—directly to the Go ecosystem without requiring Node.js, `npm`, webpack, or JavaScript build fatigue.

Every `.kiw` file encapsulates its metadata, markup, scoped styles, and client interactions in a single, readable file that compiles down to high-performance Go template trees and static assets in milliseconds.

---

## 1. Anatomy of a `.kiw` File

A canonical `.kiw` component consists of four distinct, optional sections:

```html
---
# 1. YAML Frontmatter
title: "User Profile"
layout: Base
roles: ["admin", "editor"]
---

<!-- 2. Template Body (Go html/template) -->
<article class="profile-card">
  <h2>{{.Title}}</h2>
  <p>Status: <span class="badge">Active</span></p>

  <!-- 3. Embedded Markdown (Optional) -->
  <markdown>
  ### Bio
  Fullstack developer passionate about **Go** and systems engineering.
  </markdown>
</article>

<!-- 4. Scoped CSS (Automatically Scoped) -->
<style>
.profile-card {
  border: var(--pop-border, 2px solid #0B1F3B);
  border-radius: 12px;
  padding: 1.5rem;
  background: var(--bg-2, #f0ede5);
}
h2 {
  color: var(--primary, #39D353);
  margin-top: 0;
}
.badge {
  font-weight: bold;
  text-transform: uppercase;
}
</style>

<!-- 5. Client Script (Automatically Extracted) -->
<script hydrate="idle">
console.log("Profile card mounted and active.");
</script>
```

---

## 2. YAML Frontmatter

The top of any page or layout file can declare frontmatter delimited by triple dashes (`---`). Frontmatter is parsed into typed Go data structures via `gopkg.in/yaml.v3`:

```yaml
---
title: "Documentation Guide"
description: "Learn how to build documentation portals."
layout: Base
date: "2026-09-29"
draft: false
tags:
  - golang
  - architecture
author:
  name: "Krewire Team"
  email: "team@krewire.com"
---
```

### Special Frontmatter Keys:

| Key | Purpose | Used In |
| :--- | :--- | :--- |
| `title` | Page title injected into the layout's `<title>` tag and OpenGraph headers. | `pages/` |
| `layout` | Name of the layout wrapper inside `layouts/` (e.g. `layout: Base` wraps with `layouts/Base.kiw`). | `pages/` |
| `description` | Meta description for search engines and social cards. | `pages/` |
| `draft` | When set to `true`, the page is excluded from production builds. | `pages/` |
| `date` | ISO 8601 publication or modification timestamp. | `pages/`, `content/` |

All custom keys declared in frontmatter are exposed directly inside template expressions via the root context: `{{.Title}}`, `{{.Tags}}`, `{{.Author.Name}}`.

---

## 3. Template Body & Go Expressions

The body of a `.kiw` file uses Go's standard `html/template` syntax. It provides context-aware HTML escaping to prevent Cross-Site Scripting (XSS) by default.

### Common Expressions:

#### Data Interpolation
```html
<h1>{{.Title}}</h1>
<p>Author: {{.Author.Name}}</p>
```

#### Conditional Rendering
```html
{{if .IsAdmin}}
  <a href="/admin/dashboard" class="btn btn-danger">Admin Panel</a>
{{else if .IsSubscriber}}
  <span class="badge">Subscriber</span>
{{else}}
  <a href="/signup" class="btn">Join Now</a>
{{end}}
```

#### Looping over Slices and Maps
```html
<ul class="tag-list">
  {{range .Tags}}
    <li class="tag-item">{{.}}</li>
  {{else}}
    <li class="empty">No tags available</li>
  {{end}}
</ul>
```

#### Layout Content Injection
In layout components (`layouts/*.kiw`), the page's rendered body is injected using the `{{.Content}}` directive:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
  <header><Header /></header>
  <main id="main">
    {{.Content}}
  </main>
  <footer><Footer /></footer>
</body>
</html>
```

---

## 4. Reusable Component Composition

Components located in the `components/` directory can be instantiated dynamically inside any page, layout, or other component using the `component` helper:



```

The equivalent template-helper form is `{{component "ComponentName" .}}`; both forms produce the same component composition.




### Example: Component with Scoped Props

Define `components/Alert.kiw`:

```html
<div class="alert alert-{{.Type}}">
  <span class="icon">ℹ</span>
  <div class="message">{{.Message}}</div>
</div>

<style>
.alert {
  display: flex;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 8px;
}
.alert-info { background: #e0f2fe; color: #0369a1; }
.alert-warning { background: #fef3c7; color: #92400e; }
</style>
```

Instantiate `Alert` inside `pages/index.kiw`:

```html
<Alert Type="warning" Message="Maintenance scheduled tonight at 23:00 UTC" />
```

---

## 5. Automatic Scoped CSS

Styles defined inside `<style>` blocks are **scoped to the component automatically** at build time. You never have to worry about CSS class name collisions.

### How Scoping Works:

1. When parsing a component (e.g. `components/Card.kiw`), Krewire injects a unique attribute into the component's root elements:
   ```html
   <div class="card" data-kiw-component="Card">...</div>
   ```
2. The CSS compiler (`framework/web/ssg`) rewrites your selectors to bind strictly to that component's scope:
   ```css
   /* Your authored CSS */
   .card { padding: 20px; }
   h3 { font-size: 1.25rem; }

   /* Compiled CSS output in assets/style.css */
   [data-kiw-component="Card"] .card, [data-kiw-component="Card"].card { padding: 20px; }
   [data-kiw-component="Card"] h3, [data-kiw-component="Card"]h3 { font-size: 1.25rem; }
   ```
3. All component styles are bundled and minified into a single `assets/style.css` file, eliminating render-blocking CSS requests.

---

## 6. Client Scripts & Hydration Directives

Scripts inside `<script>` tags are extracted from the HTML and compiled into static JavaScript asset files. This guarantees clean Separation of Concerns and strict Content Security Policy (CSP) compliance without inline script execution risks.

```html
<script hydrate="visible">
  const button = document.querySelector(".btn-interactive");
  button.addEventListener("click", () => {
    console.log("Button clicked!");
  });
</script>
```

### Hydration Tiers (`hydrate` attribute):

| Hydration Directive | Execution Timing | Ideal Use Case |
| :--- | :--- | :--- |
| `hydrate="load"` *(default)* | Executes immediately when the DOM loads (`DOMContentLoaded`). | Navigation menus, critical UI toggles, theme switchers. |
| `hydrate="idle"` | Executes when the main browser thread is idle (`requestIdleCallback`). | Analytics, non-critical telemetry, background widgets. |
| `hydrate="visible"` | Executes only when the element enters the viewport (`IntersectionObserver`). | Image carousels, lazy-loaded interactive charts, comments. |

---

## 7. Embedded Markdown

For content-heavy sections, you can write native Markdown directly within your `.kiw` component using the `<markdown>` tag:

```html
<section class="changelog">
  <h2>Release Notes</h2>
  <markdown>
  ### Version 0.1.0 (Initial Release)
  - **Framework:** Core modular monolith architecture.
  - **Kiw CLI:** Scaffolding, dev server, and build pipeline.
  - **Zero Bloat:** 100% Go standard library, zero npm dependencies.
  </markdown>
</section>
```

The Krewire compiler automatically converts the Markdown block to clean, sanitized HTML at build time using `github.com/krewire/libs/markdown`.

---

## 8. Built-in Helpers & Internationalization

Krewire provides built-in template helper functions:

| Helper | Syntax | Description |
| :--- | :--- | :--- |
| Component tags | `<Name />` | Renders a child component using JSX-like syntax. |
| `component` | `{{component "Name" .}}` | Compatible template-helper form for rendering a child component with context or dict props. |
| `t` / `translate` | `{{t "home.welcome"}}` | Translates a key using the active language bundle (`lang/*.json`). |
| `tLocale` | `{{tLocale "fr" "home.welcome"}}` | Translates a key for a specific locale. |
| `locales` | `{{locales}}` | Returns a list of all configured language codes. |

---

## Next Steps

Now that you have mastered the `.kiw` DSL and Krewire's component system:

- Explore the [**1.3 Krewire Workloads →**](/docs/krewire-workloads) matrix to see how `.kiw` powers static sites, books, and web applications.
- Return to the [**2. Getting Started Index →**](/getting-started).
