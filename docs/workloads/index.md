# Workload Spectrum & Dispatch Model

The Krewire framework unifies eight software workload kinds under a single declarative control plane and developer toolchain.

---

## 1. The Eight Canonical Kinds

Every project in the Krewire ecosystem declares a primary `kind` in its [`krewire.yaml`](../../pkg/model/project.go) file or relies on deterministic filesystem markers:

| Kind | Target | Primary Packages | Build / Run Behavior |
|---|---|---|---|
| [`app`](./app.md) | Fullstack web applications | `web`, `app`, `ui`, `runtime` | Server-Side Rendering (SSR) + Client WebAssembly (WASM) islands |
| [`cli`](./cli.md) | Standalone command-line tools | `tui`, `term`, `pkg/errs` | Compiles a single standalone CLI binary |
| [`site`](./site.md) | Static websites & marketing | `web/ssg`, `markdown` | Static Site Generation (SSG) into `.krewire/build`, local preview with HMR |
| [`book`](./book.md) | Technical books & documentation | `book`, `markdown` | Compiles `content/` chapters into a static HTML site with TOC and search |
| [`worker`](./worker.md) | Background workers & queues | `cloud/worker`, `pkg/lifecycle` | Supervised long-running daemon consuming background queues |
| [`service`](./service.md) | Backend microservices & APIs | `cloud/service` | High-throughput headless HTTP/REST/RPC microservice |
| [`gateway`](./gateway.md) | API gateways & reverse proxies | `web/gateway`, `cloud/service/registry` | Proxies requests along a route table, applies middleware, discovers upstreams |
| [`infra`](./infra.md) | Declarative cloud blueprints | `cloud/infra`, `cloud/deploy` | Evaluates declarative blueprints, outputs plan, provisions cloud state |

---

## 2. Kind Detection (`shape`)

When executing commands like `kiw info` or `kiw build`, the CLI uses the `shape` detector ([`shape.Detect`](../../tools/kiw/internal/shape/shape.go)) in this order:
1. **Explicit Config (`krewire.yaml`):** the `project.kind` field holds highest precedence.
2. **Declarative battery keys:** `worker:`, `service:`, `gateway:`, or `infra:` in `krewire.yaml` selects that kind.
3. **Pre-init bootstrap (not a kind):** `go.mod` + `krewire.yaml` + the starter `main.go` (containing `hello, krewire`) and none of `pages/`, `content/`, `cmd/`, `internal/`, `components/`. Detection reports `KindNone` with the marker `pre-init`; workload commands refuse to run and point at `kiw init`.
4. **Application entry point:** `main.go`, or a `cmd/*` package declaring `func main` $\rightarrow$ `app`.
5. **Declarative site:** `pages/` or the `ssg:` key $\rightarrow$ `site`.
6. **Content directory:** `content/` $\rightarrow$ `book`.

`cli` is not inferred from the filesystem; it is declared explicitly through `project.kind` (for example by `kiw init --cli`).

---

## 3. Progressive Kind Evolution (`kiw init`)

A project starts from the minimal pre-init bootstrap layout and gains workload capabilities without rewrites. `kiw new` writes that layout and equips a variant in one step (fullstack `app` by default); `kiw init` equips an existing directory.

```bash
# New project: choose the variant up front (default is app)
kiw new my-site --site
kiw new my-app

# Existing directory: initialize, then equip a kind
cd my-project
kiw init              # git, .gitignore, krewire.yaml only
kiw init --site       # Equips SSG layout & pages
kiw init --book       # Equips content/ structure & book configuration
kiw init --cli        # Equips TUI app runner & flag parsing
kiw init --gateway    # Equips the API gateway route table
kiw init --template https://github.com/example/starter.git  # Installs a custom starter template
```

---

## 4. Workload Dispatching in `kiw`

The unified CLI (`kiw`) routes tasks based on the active kind:

- **`kiw build`**:
  - `site` $\rightarrow$ Executes the SSG build pipeline into `.krewire/build`.
  - `book` $\rightarrow$ Invokes the native book builder to render content chapters.
  - `app` / `cli` / `service` / `worker` / `gateway` $\rightarrow$ Compiles the Go binary into `.krewire/build/bin/<name>`.
  - `infra` $\rightarrow$ Compiles and validates the infrastructure execution plan.
- **`kiw dev`**:
  - `site` / `book` $\rightarrow$ Serves a preview and rebuilds on change.
  - `app` / `cli` / `worker` / `service` / `gateway` $\rightarrow$ Rebuilds and restarts the supervised process on change.
  - `infra` is not a dev target; use `kiw build --plan` or `kiw deploy`.
- **`kiw serve`**:
  - `app` $\rightarrow$ Boots the local web server.
  - `cli` $\rightarrow$ Executes the command tree.
  - `site` / `book` $\rightarrow$ Serves the rendered output over HTTP.
  - `worker` / `service` / `gateway` / `infra` are not servable; use `kiw run` or `kiw dev`.
- **`kiw run`**:
  - `app` / `worker` / `service` / `gateway` $\rightarrow$ Builds to a temp binary and runs it.
  - `cli` $\rightarrow$ Runs the command tree with the remaining arguments.
  - `site` / `book` / `infra` are rejected with a hint to `kiw build` + `kiw serve` or `kiw deploy`.
  - A pre-init bootstrap project is rejected with a hint to `kiw init`.
