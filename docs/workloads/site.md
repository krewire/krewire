# Workload: Static Website (`site`)

The `site` workload powers static websites, documentation hubs, blogs, and landing pages using Static Site Generation (SSG) with Markdown support and Hot Module Replacement (HMR).

---

## 1. Key Capabilities

- **Asset Pipeline (`pkg/web/ssg`)**: Generates optimized static HTML, CSS, and JavaScript with optional minification and fingerprinting.
- **Live Preview (`pkg/web/ssg/hmr`)**: WebSocket-driven browser refresh on file changes during `kiw serve`.
- **Markdown & Frontmatter (`pkg/markdown`)**: Parses Markdown files with YAML frontmatter into structured content collections.
- **`.kiw` DSL (`pkg/kiw`)**: Pages, layouts, and components authored as `.kiw` files with scoped styles and component tags.

---

## 2. Directory Layout

The `kiw init --site` scaffold writes:

```text
my-site/
├── krewire.yaml
├── pages/
│   └── index.kiw
├── layouts/
│   └── Base.kiw
├── components/
│   └── Hero.kiw
├── public/
│   └── favicon.svg
└── README.md
```

Markdown collections can be added under a content directory and declared with `ssg.collections`.

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-site
  kind: site

output: ".krewire/build"
base: "/"

ssg:
  description: "My static site"
  theme:
    default: "auto"        # auto | light | dark
  components: []           # {name, body, style}
  layouts: []              # {name, body, style} or {name, ui}
  pages: []                # declarative output pages
  collections: []          # Markdown + frontmatter collections
  auto_assets:
    enabled: true
    bundle: true
```

---

## 4. Development Workflow

```bash
# Preview website with live reload (default :8080)
kiw serve --addr :3000

# Build static output into .krewire/build
kiw build

# Stage for deployment
kiw deploy
```
