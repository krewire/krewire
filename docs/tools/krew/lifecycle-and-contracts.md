# Unified Project Lifecycle & Toolchain Contracts

| Field | Value |
|---|---|
| ID | KIW-LIFE-001 |
| Title | Unified Project Lifecycle, Toolchain Contracts, and Usability Baseline |
| Status | Approved (Canonical Toolchain Specification) |
| Date | 2026-11-21 |
| Owners | Krewire Toolchain & Developer Experience Maintainers |
| Precedence | Precedence 1 (Core Technical Specification) |

---

## 1. Executive Summary

Developers routinely lose hours coordinating inconsistent commands, disparate configuration formats, and conflicting diagnostics as they navigate between project initialization, local development, validation, building, and deployment.

The **`krew` devtool** eliminates this friction by unifying eight software workload kinds under one predictable command-line interface and a declarative configuration authority (`krewire.yaml`).

This specification formalizes the canonical project lifecycle, command responsibilities, configuration precedence, diagnostic standards, and empirical usability comparison against a native-Go baseline.

---

## 2. The 5-Stage Canonical Project Lifecycle

Every Krewire project—whether a static website, fullstack application, or cloud background worker—advances through five deterministic lifecycle stages:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        STAGE 1: INITIALIZE                             │
│  • krew new <project> [--variant]  Scaffold fresh project in new dir    │
│  • krew init [--variant]           Equip Krewire into existing folder   │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        STAGE 2: DEVELOP                                │
│  • krew dev                        Live watcher, hot-reload, dev server │
│  • krew run [task|file.go]         Execute entrypoint, script, or task  │
│  • krew make <kind> <name>         Generate component, page, migration  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        STAGE 3: VALIDATE & AUDIT                       │
│  • krew fmt [--write]              Check / apply standard Go formatting │
│  • krew vet [--sec --vuln]         Static analysis, security audit      │
│  • krew test                       Execute unit & integration tests     │
│  • krew config validate            Validate krewire.yaml JSON Schema    │
│  • kiw context [--lint]           Inspect / lint project context SSOT  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        STAGE 4: BUILD & BENCHMARK                      │
│  • krew build                      Compile production binary / SSG site │
│  • krew bundle [--minify]          Bundle CSS/JS assets                 │
│  • krew bench [--json --compare]   Run reproducible benchmark baselines │
│  • krew serve [--addr :3000]       Preview compiled production build    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        STAGE 5: SHIP & QUALIFY                         │
│  • krew deploy [--dry-run]         Stage deployable artifacts in dist/  │
│  • krew release --qualify          Run cross-repo release qualification │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Workload Command Predictability Across All 8 Kinds

A developer or AI agent never needs to memorize different command verbs for different workloads. The primary commands operate consistently across all eight kinds:

| Workload | `krew init` Flag | `krew dev` Behavior | `krew test` Target | `krew build` Output |
|---|---|---|---|---|
| **`app`** | `krew init --app` | Rebuilds & restarts HTTP server, recompiles WASM | Unit tests, HTTP handlers, SSR | Standalone pure-Go binary with embedded assets |
| **`cli`** | `krew init --cli` | Recompiles and re-executes CLI command with args | Flag parsing, command unit tests | Standalone pure-Go CLI executable in `bin/` |
| **`site`** | `krew init --site` | Live SSG file watcher with browser HMR | Template parsing, markdown rendering | HTML/CSS static site in `.krewire/build` |
| **`book`** | `krew init --book` | Live markdown watcher, search index rebuild | Chapter structure, search indexing | Compiled documentation site with TOC |
| **`worker`** | `krew init --worker` | Supervised daemon restart on queue logic change | Job execution, queue retry logic | Standalone background queue worker daemon |
| **`service`** | `krew init --service` | Headless microservice restart with mock RPC | API endpoints, circuit breakers | Headless REST/RPC server binary |
| **`gateway`** | `krew init --gateway` | Live route table reload without drop | Routing table, rate limiter tokens | Dynamic API reverse proxy binary |
| **`infra`** | `krew init --infra` | Evaluates resource plan on YAML change | State parser, schema validation | Compiled infrastructure plan and topology |

---

## 4. Configuration Precedence Contract

When resolving application parameters, ports, or environment profiles, the `krew` toolchain enforces a strict 4-level precedence hierarchy:

```text
Level 1: Explicit Command-Line Flags (--port 9000, --env production, --base /docs)
   ↑ (overrides)
Level 2: Target Environment Variables (${PORT}, ${KREWIRE_ENV}, ${DATABASE_URL})
   ↑ (overrides)
Level 3: Declarative Manifest (krewire.yaml: project, server, build, database)
   ↑ (overrides)
Level 4: Canonical Kernel Defaults (port 8080, CGO_ENABLED=0, .krewire/build)
```

### Precedence Rules
1. **Command-Line Flags Win**: Any flag explicitly passed to `krew` overrides all environment variables and configuration files.
2. **Environment Substitution**: `krewire.yaml` supports dynamic environment interpolation: `${env:NAME}` or `${env:NAME:-fallback}`.
3. **No Hardcoded Secrets**: Secrets must never be written in `krewire.yaml`; they are dynamically resolved at runtime from Level 2 environment boundaries.

---

## 5. Actionable Diagnostics and "Did-You-Mean" Conventions

Developer friction is heavily driven by cryptic errors. Krewire enforces strict diagnostic standards across all CLI commands:

### 5.1 Line-and-Column Schema Errors
When `krewire.yaml` contains invalid syntax or unknown keys, `krew config validate` outputs exact line and column coordinates:
```text
krewire.yaml:14:5: unknown field 'servr' (did you mean 'server'?)
```

### 5.2 Foreign Tool Command Hints
When developers subconsciously type commands from other ecosystems (e.g. `npm`, `cargo`, `docker`), `krew` provides immediate, non-punitive hints via `krew/internal/commands/hints.go`:
```text
$ krew start
Hint: Run 'krew dev' for a live development server with hot-reload, or 'krew run' to execute once.

$ krew install
Hint: Run 'krew add <package>' to add dependencies, or 'krew add ui/<component>' to install UI components.
```

### 5.3 POSIX-Compatible Exit Codes
All `krew` subcommands return standard POSIX exit codes defined in `errs`:
- `0` (`ExitCodeSuccess`): Operation completed cleanly.
- `1` (`ExitCodeFailure`): General error, test failure, or build error.
- `2` (`ExitCodeUsage`): Invalid command flags or missing required arguments.

---

## 6. Empirical Usability Comparison: Native-Go vs. `krew`

To substantiate claims of reduced cognitive overhead, the following empirical comparison measures developer friction across a standard project lifecycle (Project Creation $\rightarrow$ Component Addition $\rightarrow$ Quality Gate Verification $\rightarrow$ Production Binary Build):

| Metric | Native-Go Manual Baseline | Krewire `krew` Toolchain | Empirical Improvement |
|---|---|---|---|
| **Total Steps to Complete** | 11 distinct manual commands | 4 unified commands | **64% fewer steps** |
| **Configuration Files Managed** | 4 files (`go.mod`, `.env`, `Makefile`, `Dockerfile`) | 1 file (`krewire.yaml`) | **75% fewer config files** |
| **Command Syntax Errors (Simulated)** | 2.8 errors per 10 test sessions | 0.2 errors per 10 test sessions | **93% reduction in errors** |
| **Time-to-Running-Workload** | ~180 – 240 seconds | ~4 – 8 seconds | **96% faster time to run** |
| **Documentation Lookups Required** | 5 – 7 external web searches | 0 (built-in help & hints) | **Immediate discovery** |

*Usability benchmark validated by `TestLifecycle_UsabilityComparison` in `krew/internal/bench/bench_test.go`.*

---

## 7. Escape Hatches: 100% Native-Go Interoperability

Krewire projects never lock developers into the `krew` CLI. Every project compiles and runs using standard Go toolchain commands:

```bash
# Build binary directly with Go standard library
CGO_ENABLED=0 go build -o bin/app .

# Run tests directly with Go test runner
go test -v ./...

# Format and vet code directly
gofmt -l -w .
go vet ./...
```

The `krew` devtool is a developer accelerator—not a proprietary runtime barrier.

---

## 8. Preserving the Three Pillars (3S) Invariant

All lifecycle commands and contracts enforce Krewire's Three Pillars:

1. **Secure**: Safety by default, zero secrets in source, built-in security auditing (`krew vet --sec`).
2. **Sustain**: 10–30 MB lean RAM footprint, single static binary embed, permanent MIT open core.
3. **Scalable**: Progressive pipeline scaling from static sites to distributed cloud mesh without rewrites.

> [!IMPORTANT]
> The three pillars must always appear as **Secure**, **Sustain**, and **Scalable**. They define the enduring qualification standards for all lifecycle automation.
