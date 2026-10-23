# Workload Spectrum & Dispatch Model

The Krewire framework unifies eight software workload kinds under a single declarative control plane and developer toolchain.

---

## 1. The Eight Canonical Kinds

Every project in the Krewire ecosystem declares a primary `kind` in its [`krewire.yaml`](../../packages/model/project.go) file or relies on deterministic filesystem markers:

| Kind | Target | Primary Packages | Build / Run Behavior |
|---|---|---|---|
| [`kernel`](./kernel.md) | Minimal initial seed | `kern` | Standard Go build; foundation for progressive upgrading |
| [`site`](./site.md) | Static websites & marketing | `web/ssg`, `markdown` | Static Site Generation (SSG) into `.krewire/build`, local preview with HMR |
| [`book`](./book.md) | Technical books & documentation | `mdbind`, `markdown` | Compiles `manuscript/` chapters into HTML/EPUB/PDF outputs |
| [`app`](./app.md) | Fullstack web applications | `web`, `app`, `ui`, `runtime` | Server-Side Rendering (SSR) + Client WebAssembly (WASM) islands |
| [`cli`](./cli.md) | Standalone command-line tools | `tui`, `term`, `kern/errs` | Compiles single standalone CLI binary |
| [`worker`](./worker.md) | Background workers & queues | `cloud/worker`, `kern/lifecycle` | Supervised long-running daemon consuming background queues |
| [`service`](./service.md) | Backend microservices & APIs | `web/gateway`, `cloud/service` | High-throughput headless HTTP/REST/RPC microservice |
| [`infra`](./infra.md) | Declarative cloud blueprints | `cloud/infra`, `cloud/deploy` | Evaluates declarative blueprints, outputs plan, provisions cloud state |

---

## 2. Kind Detection (`shape`)

When executing commands like `kiw info` or `kiw build`, the CLI uses the `shape` detector to infer the kind:
1. **Explicit Config (`krewire.yaml`):** The `project.kind` field holds highest precedence.
2. **Filesystem Markers (Fallback):**
   - If `manuscript/` exists $\rightarrow$ `book`.
   - If `infra/` exists $\rightarrow$ `infra`.
   - If `pages/` or `static/` exists $\rightarrow$ `site`.
   - If `main.go` exists $\rightarrow$ `app` or `cli`.

---

## 3. Progressive Kind Evolution (`kiw init`)

A project begins at the minimal [`kernel`](./kernel.md) kind. As requirements expand, developers equip the project with workload capabilities without rewrites:

```bash
# Start with minimal kernel
kiw new my-project

# Progressively equip with a workload kind
cd my-project
kiw init --site      # Equips SSG layout & pages
kiw init --book      # Equips manuscript/ structure & mdbind configuration
kiw init --cli       # Equips TUI app runner & flag parsing
kiw init --template  # Installs custom starter template
```

---

## 4. Workload Dispatching in `kiw`

The unified CLI (`kiw`) routes tasks based on the active kind:

- **`kiw build`**:
  - `site` $\rightarrow$ Executes SSG build pipeline to output static assets.
  - `book` $\rightarrow$ Invokes `mdbind` to render chapter manuscripts.
  - `app` / `cli` / `service` / `worker` $\rightarrow$ Executes `go build` with embedded assets.
  - `infra` $\rightarrow$ Compiles and validates infrastructure execution plan.
- **`kiw dev` / `kiw serve`**:
  - `site` $\rightarrow$ Serves HTTP preview with WebSocket-driven live-reload (HMR).
  - `app` $\rightarrow$ Boots local web server with automatic reload on code change.
  - `cli` / `worker` / `service` $\rightarrow$ Runs in watch mode with supervision.
