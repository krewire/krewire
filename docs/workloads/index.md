# Workload Spectrum & Dispatch Model

The Krewire framework unifies eight software workload kinds under a single declarative control plane and developer toolchain.

---

## 1. The Eight Canonical Kinds

Every project in the Krewire ecosystem declares a primary `kind` in its [`krewire.yaml`](../../model/project.go) file or relies on deterministic filesystem markers. See the comprehensive [Workload Maturity & Readiness Matrix](./maturity-matrix.md) for evidence-based production statuses.

| Kind | Target | Primary Packages | Build / Run Behavior |
|---|---|---|---|
| [`app`](./app.md) | Fullstack web applications | `web`, `app`, `ui`, `runtime` | Server-Side Rendering (SSR) + Client WebAssembly (WASM) islands |
| [`cli`](./cli.md) | Standalone command-line tools | `tui`, `term`, `errs` | Compiles a single standalone CLI binary |
| [`site`](./site.md) | Static websites & marketing | `web/ssg`, `markdown` | Static Site Generation (SSG) into `.krewire/build`, local preview with HMR |
| [`book`](./book.md) | Technical books & documentation | `book`, `markdown` | Compiles `content/` chapters into a static HTML site with TOC and search |
| [`worker`](./worker.md) | Background workers & queues | `cloud/worker`, `lifecycle` | Supervised long-running daemon consuming background queues |
| [`service`](./service.md) | Backend microservices & APIs | `cloud/service` | High-throughput headless HTTP/REST/RPC microservice |
| [`gateway`](./gateway.md) | API gateways & reverse proxies | `web/gateway`, `cloud/service/registry` | Proxies requests along a route table, applies middleware, discovers upstreams |
| [`infra`](./infra.md) | Declarative cloud blueprints | `cloud/infra`, `cloud/deploy` | Evaluates declarative blueprints, outputs plan, provisions cloud state |

---

## 2. Capability Detection (`shape`)

When executing commands like `krew info` or `krew build`, the CLI uses the `shape` detector ([`shape.Detect`](../../../krew/core/shape/shape.go)):
1. **Declarative battery keys:** `worker:`, `service:`, `gateway:`, or `infra:` in `krewire.yaml` selects that capability.
2. **Pre-init bootstrap:** `go.mod` + `krewire.yaml` + the starter `main.go` (containing `hello, krewire`) and none of `pages/`, `content/`, `cmd/`, `internal/`, `components/`. Detection reports marker `pre-init`; workload commands point at `krew init`.
3. **Application entry point:** `main.go`, or a `cmd/*` package declaring `func main`.
4. **Declarative site:** `pages/` or the `ssg:` key.
5. **Content directory:** `content/` or the `book:` key.

---

## 3. Progressive Kind Evolution (`krew init`)

A project starts from the minimal pre-init bootstrap layout and gains workload capabilities without rewrites. `krew new` writes that layout and equips a variant in one step (fullstack `app` by default); `krew init` equips an existing directory.

```bash
# New project: choose the variant up front (default is app)
krew new my-site --site
krew new my-app

# Existing directory: initialize, then equip a kind
cd my-project
krew init              # git, .gitignore, krewire.yaml only
krew init --site       # Equips SSG layout & pages
krew init --book       # Equips content/ structure & book configuration
krew init --cli        # Equips TUI app runner & flag parsing
krew init --gateway    # Equips the API gateway route table
krew init --template https://github.com/example/starter.git  # Installs a custom starter template
```

---

## 4. Workload Dispatching in `krew`

The unified CLI (`krew`) routes tasks based on active capabilities:

- **`krew build`**:
  - `site` $\rightarrow$ Executes the SSG build pipeline into `.krewire/build`.
  - `book` $\rightarrow$ Invokes the native book builder to render content chapters.
  - `app` / `cli` / `service` / `worker` / `gateway` $\rightarrow$ Compiles the Go binary into `.krewire/build/bin/<name>`.
  - `infra` $\rightarrow$ Compiles and validates the infrastructure execution plan.
- **`krew dev`**:
  - `site` / `book` $\rightarrow$ Serves a preview and rebuilds on change.
  - `app` / `cli` / `worker` / `service` / `gateway` $\rightarrow$ Rebuilds and restarts the supervised process on change.
  - `infra` is not a dev target; use `krew build --plan` or `krew deploy`.
- **`krew serve`**:
  - `app` $\rightarrow$ Boots the local web server.
  - `cli` $\rightarrow$ Executes the command tree.
  - `site` / `book` $\rightarrow$ Serves the rendered output over HTTP.
  - `worker` / `service` / `gateway` / `infra` are not servable; use `krew run` or `krew dev`.
- **`krew run`**:
  - `app` / `worker` / `service` / `gateway` $\rightarrow$ Builds to a temp binary and runs it.
  - `cli` $\rightarrow$ Runs the command tree with the remaining arguments.
  - `site` / `book` / `infra` are rejected with a hint to `krew build` + `krew serve` or `krew deploy`.
  - A pre-init bootstrap project is rejected with a hint to `krew init`.
