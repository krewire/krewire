# Workload: Static Website (`site`)

The `site` workload powers static websites, documentation hubs, blogs, and landing pages using Static Site Generation (SSG) with Markdown support and Hot Module Replacement (HMR).

---

## 1. Key Capabilities

- **Asset Pipeline (`packages/web/ssg`)**: Generates optimized static HTML, CSS, JavaScript, and WebP images.
- **Live Preview (`packages/web/ssg/hmr`)**: WebSocket-driven instant browser refresh on file changes during `kiw serve`.
- **Markdown & Frontmatter (`packages/markdown`)**: Parses markdown files with YAML frontmatter into structured content collections.

---

## 2. Directory Layout

```text
my-site/
├── krewire.yaml          # Project configuration (kind: site)
├── content/              # Markdown articles and pages
│   ├── index.md
│   └── about.md
├── templates/            # HTML layout templates (Go html/template)
│   ├── base.html
│   └── post.html
└── static/               # Static assets (images, css, js)
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-site
  kind: site

ssg:
  content_dir: "content"
  templates_dir: "templates"
  output_dir: ".krewire/build"
  clean_urls: true
  minify: true
```

---

## 4. Development Workflow

```bash
# Preview website with live reload (default port 3000)
kiw serve --port 3000

# Build static output into .krewire/build
kiw build

# Stage for deployment
kiw deploy
```
