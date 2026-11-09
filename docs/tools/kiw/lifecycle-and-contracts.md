# Unified Project Lifecycle & Toolchain Contracts

This specification defines the canonical developer project lifecycle, CLI commands, and configuration contracts managed by the `kiw` toolchain.

---

## 1. Canonical Project Lifecycle

The `kiw` toolchain orchestrates the 5 stages of a project's lifecycle:

```text
1. Initialize (kiw new / kiw init)
       ↓
2. Develop & Iterate (kiw dev / kiw run)
       ↓
3. Validate & Test (kiw vet / kiw fmt / kiw test / kiw compat)
       ↓
4. Build & Package (kiw build / kiw bundle)
       ↓
5. Ship & Deploy (kiw deploy / kiw release)
```

### Command Responsibilities
- `kiw new <name> [--variant]`: Scaffolds a new project directory with standard structure and starter files.
- `kiw init`: Initializes a Krewire workspace in an existing directory without overwriting foreign host manifests.
- `kiw dev`: Starts development watcher, local web server, and hot module reload (HMR).
- `kiw test`: Executes automated unit and integration tests.
- `kiw build`: Produces compiled static binaries or static site generator (SSG) assets.
- `kiw info`: Reports environment information, project shape, and co-located external stacks.
- `kiw compat`: Verifies semantic version compatibility across modules.

---

## 2. Configuration Contract (`krewire.yaml`)

A project declares its metadata and build configuration in `krewire.yaml`:

```yaml
# Canonical Project Manifest
project:
  name: "my-service"
  kind: "service"           # Optional workload shape (app, site, docs, worker, service, infra, cli)
  version: "0.1.0"          # Semantic version

server:
  port: 8080                # HTTP listening port
  host: "127.0.0.1"

output: ".krewire/build"    # Output artifact directory

build:
  depends_on: []            # Foreign build hooks or script tasks
```

### Fluid Hybrid Support
Projects are not restricted to a single rigid kind. If a project contains both Go entrypoints (`main.go`) and SSG pages (`pages/`), `kiw build` fluidly compiles both binary and HTML artifacts.

---

## 3. UI System Boundary (`.kiw` and `kui`)

- `kiw` provides the compiler and template engine for `.kiw` files (`pkg/kiw`).
- UI component definitions (buttons, dialogs, drawers) are managed by the Pop-Brutalist `kui` component library (`ui/components`).
- `kiw` remains decoupled from specific UI component business logic.
