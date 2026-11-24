# Workspace Graph & Affected-Task Engine (`workgraph`)

Import: `github.com/krewire/krewire/workgraph`

## Purpose

`workgraph` is the deterministic dependency graph and affected-task engine
shared by the Krewire developer toolchain and its AI-native workflows (#135).
It answers three questions without an LLM:

1. **What depends on what, and where is that declared?** Every edge carries
   provenance (file, line, detail).
2. **What does this change affect, and why?** Changed files or a `base...head`
   Git range map to affected nodes, each with a cause.
3. **Which targets must run, which may be restored, and which are skipped?**
   Targets run in topological order with bounded parallelism behind a
   content-addressed local cache, and every decision is reported with a reason.

The package is a library. Its CLI surface belongs to `krew` (see
[Krew integration](#krew-integration)); foundational domain modules never import tooling.

---

## Graph model

| Node kind  | ID example                                   | Owned files (hash inputs)                                   |
|------------|----------------------------------------------|-------------------------------------------------------------|
| `module`   | `module:github.com/krewire/krewire`          | `go.mod`, `go.sum` (lock state, Go/toolchain directives)    |
| `package`  | `package:github.com/krewire/krewire/web` | `*.go` in the directory, `testdata/**`, `//go:embed` trees, manifest `inputs` |
| `external` | `external:golang.org/x/net`                  | none; the required version is part of the key              |
| `template` | `template:templates/init/app`                | every file under the directory                              |
| `workload` | `workload:app`                               | manifest `fixtures` directories                             |

| Edge kind     | Meaning (`from` depends on `to`)                          | Cycles allowed |
|---------------|-----------------------------------------------------------|----------------|
| `import`      | non-test Go import                                        | no             |
| `test-import` | import only from `_test.go` files (external test packages) | yes            |
| `member`      | package belongs to its module (inherits lock state)       | no             |
| `require`     | `go.mod` require between modules (version contract)       | no             |
| `uses`        | workload/template relies on a package                     | no             |
| `template`    | workload is instantiated from a template                  | no             |

`Graph.Validate` returns a `*CycleError` for any cycle through a kind that
forbids cycles, including cross-repository `require` cycles. Build constraints
are ignored on purpose: the graph is a superset, so no platform-specific
dependency is ever missed.

### Discovery

`Discover(root, manifest)` reads `go.work` `use` directives, then manifest
`modules` (sibling repositories such as `../krew`), defaulting to `.`. Each
module is walked like the Go tool walks it (`testdata`, `_*`, `.*` are not
packages; nested modules are skipped unless listed; `.git`, `.krewire`,
`vendor`, `node_modules` are never read). Imports resolve to workspace
packages first, then to the longest matching `require` path; standard-library
imports are not nodes.

### Workspace manifest (`go.work`)

Optional. It only declares what imports cannot express; unknown fields are
rejected so a typo cannot silently drop an invalidation rule.

```yaml
version: 1
modules: ["../krew"]               # extra module roots (cross-repo)
exclude: ["examples/**"]           # directory globs that are not packages
global_inputs:                     # any change affects every node
  - .github/workflows/*.yml
  - "**/schemas/*.json"
inputs:                            # extra files owned by a directory's node
  ".": ["*/**/*.go"]               # root guard tests read all module sources
templates: ["templates/init/*", "templates/new/*", "templates/boost"]
workloads:
  - name: app
    template: templates/init/app
    packages: [web, app, runtime/boot]
    fixtures: [examples/app]
tasks: []                          # overrides DefaultTasks when non-empty
```

`go.work`, `go.work.sum`, `.go-version`, `.tool-versions`, and the manifest
itself are always global inputs (`DefaultGlobalInputs`). The core repository
ships its own manifest at the repository root.

---

## Affected calculation

```go
files, _ := workgraph.ChangedFiles(ctx, root, "origin/main", "HEAD") // "" head = working tree + untracked
set := ws.Affected(files, workgraph.AffectedOptions{Base: "origin/main", Head: "HEAD"})
full := ws.All() // explicit full-run mode: bypasses affected calculation
```

* Owners of changed files are affected (`cause.kind = "changed"`). Deleted Go
  files are attributed to their package directory; files under templates and
  fixtures to the enclosing tree. Renames count as delete + add.
* A changed global input affects every node (`"global"`).
* Affectedness then propagates to all transitive dependents over every edge
  kind, including test-only imports (`"dependency"`, with the edge kind and
  its provenance).
* Files nobody owns (for example `docs/**`) are listed in `unowned_files`.
  `UnownedIsGlobal: true` turns them into global inputs for repositories that
  prefer maximal safety.

| Change                                      | Invalidates                                                 |
|---------------------------------------------|-------------------------------------------------------------|
| a package's source, testdata, or embed      | the package and all transitive dependents, templates, workloads, other repositories |
| `go.mod` / `go.sum` (incl. `go`/`toolchain`) | the module, all its packages, every module that requires it |
| `go.work`, `.go-version`, shared schemas     | everything                                                  |
| a template file                             | the template and workloads built from it                    |
| a workload fixture                          | the workload                                                |

---

## Tasks and targets

A `TaskSpec` becomes one target per node of kind `on` (default `package`).
Target IDs are `<task>#<node id>`. Placeholders: `{package}`, `{dir}`
(`./`-relative to the module), `{module}`, `{name}`. Commands run in the
module directory.

`DependsOn` uses Nx semantics: `"build"` is the same node's build target,
`"^build"` the build targets of its direct dependencies (test-only edges do
not order builds). `DefaultTasks()` = `format` (`gofmt -l {dir}`,
fail-on-output), `vet`, `build` (`^build`), and `test` (`build`). Kinds
`generate`, `package`, and `release` are available for custom specs; mark
side-effecting tasks `no_cache: true`.

`BuildTasks` rejects duplicate names, unknown dependencies, and cycles.
`TaskGraph.ValidatePlan` rejects externally proposed steps (for example from
an AI plan) that name unknown targets or claim dependencies the graph does not
contain.

---

## Cache keys and correctness

`Hasher.Inputs(target)` returns the `KeyInputs`; `KeyInputs.Key()` is the
SHA-256 of their canonical JSON. A key covers:

* key schema version (`workgraph-key/v1`), target ID, expanded argv, work dir,
  and fail-on-output flag;
* the content digest of every file owned by the node and by all transitive
  dependencies, global inputs, and task `inputs`;
* dependency lock state: every `go.mod`/`go.sum` in the closure (listed
  separately under `lock`) and the versions of external modules;
* the active Go version (`GoVersion` = `go env GOVERSION`);
* environment values of `DefaultEnv` (`GOOS`, `GOARCH`, `CGO_ENABLED`,
  `GOFLAGS`, `GOEXPERIMENT`, `GOTOOLCHAIN`) plus task-declared `env`.

Guarantees enforced by the runner and `LocalCache`:

* Only passing results are stored; a failed target always re-runs.
* A target whose dependency failed or was blocked is `blocked`, never
  restored, so a cache hit can never hide a failed or missing verification.
* Entries carry a checksum and their key; a malformed, mismatched, tampered,
  or non-passing entry is deleted and the target executes
  (`"cache entry corrupt; fell back to execution"`).
* Writes are atomic (temp file + rename). Keys are recomputed from disk, so
  editing reported inputs cannot force a restore.

The default cache lives in `.krewire/cache/workgraph` (git-ignored). Remote
caching is intentionally out of scope until local invalidation is proven.

---

## Running and reporting

```go
tg, _ := workgraph.BuildTasks(ws.Graph, workgraph.DefaultTasks())
gv, _ := workgraph.GoVersion(ctx, ws.Root)
report, _ := tg.Run(ctx, ws, workgraph.RunOptions{
    Affected:    set,                         // nil = full run
    Tasks:       []string{"test"},            // dependencies still run
    Concurrency: 8,
    Cache:       workgraph.LocalCache{Dir: filepath.Join(ws.Root, workgraph.DefaultCacheDir)},
    Hasher:      workgraph.NewHasher(ws, gv, os.Getenv),
})
report.WriteJSON(os.Stdout) // or WriteText
ok := report.OK()           // false if anything failed or was blocked
```

Every target ends in exactly one status with a reason:

| Status     | Typical reasons                                                    |
|------------|--------------------------------------------------------------------|
| `executed` | `cache miss`, `cache disabled`, `restore disabled`, `cache entry corrupt; fell back to execution` |
| `restored` | `cache hit <key> (passed with identical inputs)`                   |
| `skipped`  | `not affected`, `task not requested`                               |
| `failed`   | `...; exit code N`, `...; produced output on a fail-on-output gate` |
| `blocked`  | `dependency <id> failed`, `cancelled: ...`                         |

All JSON documents carry `api_version: workgraph.krewire.dev/v1` and a `kind`
(`WorkspaceGraph`, `AffectedSet`, `TaskReport`). Output is deterministic:
nodes, edges, and targets are sorted, and absolute paths never appear.

---

## AI-native integration

The graph is the single source of dependency facts for both humans and
agents; agents query it instead of inferring relationships from prose.

| Need (#135)                                   | API                                                        |
|-----------------------------------------------|------------------------------------------------------------|
| direct/transitive dependencies with provenance | `Graph.Dependencies/Dependents(id, transitive)`, `Graph.EdgesBetween`, `Graph.Path` |
| map proposed changes to affected targets      | `Workspace.Affected(files, ...)` and `AffectedSet.Cause`   |
| validate an agent plan against the graph      | `TaskGraph.ValidatePlan([]PlanStep)`                       |
| run required checks deterministically         | `TaskGraph.Run` (agents cannot skip gates or override keys) |
| results as evidence                           | `Report.Evidence(commit) []model.VerificationEvidence`     |
| pin the graph in project context              | `Graph.Ref()` (JSON-identical to the toolchain contract `GraphRef`: `nodes`, `edges`, `digest`, `source`) |

`Report.Evidence` maps task kinds to `model.QualityGateType`
(`format`→`FORMAT`, `lint`→`STATIC_ANALYSIS`, `test`→`TEST`,
`build`→`BUILD`, others→`CUSTOM`) and statuses to `model.VerificationStatus`
(`executed`/`restored`→`PASSED`, `failed`→`FAILED`, `skipped`→`SKIPPED`,
`blocked`→`BLOCKED`). Restored evidence records the cache key in its output so
a reviewer can tell a restore from a re-run.

---

## Krew integration

`krew` owns the commands; each is a thin adapter over this package and
produces the same JSON documents for CI and AI workflows. Recommended wiring
(`tui.NewCommand` in `internal/cli/app.go`, group `inspect`/`develop`):

| Command | Behaviour |
|---------|-----------|
| `krew graph [--json] [--deps ID] [--dependents ID] [--transitive] [--why FROM TO]` | `Discover(root, nil)`, `Validate()`; print `WriteText`/`WriteJSON`, or the filtered query / `Path` with provenance. Exit 1 on `*CycleError`. |
| `krew affected --base REF [--head REF] [--json] [--strict]` | `ChangedFiles` + `Affected` (`--strict` = `UnownedIsGlobal`); print `AffectedSet`. |
| `krew run TASK... [--all \| --affected --base REF [--head REF]] [--parallel N] [--no-cache] [--no-restore] [--json] [--inputs]` | `BuildTasks` (manifest `tasks` or `DefaultTasks`), `Run`; `--all` passes `Affected: nil`; `--no-cache` passes `Cache: nil`; exit 1 unless `report.OK()`. |

CI: `krew run format vet build test --affected --base origin/main --json > workgraph-report.json`
on pull requests, `--all` on `main` and release tags. The report's summary
and per-target reasons are the required "skipped / restored / executed"
evidence.

Toolchain contracts (#131/#140, `toolchain` on the `fino-131-140`
branch): the project-context builder sets `ProjectContext.Graph` from
`Graph.Ref()` (the JSON fields match `contract.GraphRef`; set `Source` to
`workgraph`), the plan stage validates `PlanStep` targets with
`TaskGraph.ValidatePlan`, and the verify stage runs the affected targets
through `TaskGraph.Run` and appends `Report.Evidence(commit)` to
`EvidenceReport.Verification.Gates` (both are `[]model.VerificationEvidence`). This package does not import `toolchain`, so the
two can merge in either order; the adapter lives in krew or in a follow-up
change once both are on `main`.

---

## Limits and follow-ups

* Build constraints are ignored (superset graph); tasks run on the host
  platform.
* Cache entries store outcomes and logs, not build artifacts.
* Cross-repository changes are attributed per repository: run
  `ChangedFiles` in each repository and prefix its paths with the module
  directory used in `go.work` or `modules`.
* Remote cache: deferred by design (#135 non-goal).
