# Project Blueprints & Scaffolding Engine

Detailed breakdown of starter blueprints, file layout templates, and customization options.

---

## 1. Fresh Project Blueprints (`templates/new/`)

### Kernel Blueprint (`templates/new/kernel/`)
The default seed generated when running `kiw new <name>`:
- `go.mod`: Initialized with Go 1.27.1+ and project module path.
- `krewire.yaml`: Set to `kind: kernel`.
- `main.go`: Minimal runnable Go program.
- `.gitignore`: Standard Go and Krewire ignores (`dist/`, `.krewire/`, `bin/`).

---

## 2. In-Place Workload Initializers (`templates/init/`)

Equips an existing directory without clobbering existing git history:

- **`templates/init/app/`**: Adds `handlers/`, `services/`, and `views/` for fullstack web applications.
- **`templates/init/book/`**: Adds `manuscript/` structure and chapter markdown boilerplate.
- **`templates/init/cli/`**: Adds command subtrees using `packages/tui`.
- **`templates/init/static/`**: Adds `content/`, `templates/`, and asset directories for SSG sites.

---

## 3. Creating Custom Templates

Custom templates follow standard Krewire directory structures and can be loaded via:
```bash
kiw init --template <path-or-git-url>
```
All template files support variable substitution (`{{.ProjectName}}`, `{{.ModulePath}}`).
