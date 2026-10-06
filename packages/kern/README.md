# kern

**kern** is the Krewire Kernel — the bottom layer of the Krewire ecosystem.

```go
import "github.com/krewire/kern"
```

## Packages

The root package is a facade over seven focused subpackages, so a consumer
needs a single import to reach the whole kernel.

| Package | Role |
|---|---|
| [`errs`](./errs) | Errors, exit codes, diagnostics, stack traces |
| [`model`](./model) | `Kind`, `Project`, `Scope`, `DomainEvent`, and the generic `Matrix`/`OptInRule` shapes |
| [`spec`](./spec) | `SpecID`, `RequirementID` |
| [`version`](./version) | The `Version` value type and its semver rules |
| [`lifecycle`](./lifecycle) | `Kernel`, `Registry`, `Executor`, `Supervisor` |
| [`env`](./env) | Target environments |
| [`log`](./log) | Structured logging |

Each exported symbol in the root package is an alias of the subpackage that
owns it. Import the subpackage directly when the boundary matters.

## Design

kern depends on **nothing outside the Go standard library** — and nothing
outside itself, ecosystem-wise. That is the whole point: it is the layer every
other module may import without risking a dependency cycle. `libs` and above
build on it; it never reaches back up.

### The kernel names nothing

The kernel is the lowest layer, so it must not know which modules exist above
it. It ships:

- no module roster
- no ecosystem version matrix
- no licensing or tier policy
- no workload table

Those are ecosystem facts. They live in the layer above, written in the shapes
the kernel provides:

```go
// The ecosystem declares its own workload matrix …
var Matrix = model.NewMatrix(
    model.Workload[string]{Kind: model.KindSite, Package: "web/ssg", Title: "Static sites (SSG)"},
    model.Workload[string]{Kind: model.KindWorker, Package: "cloud/worker"},
)

// … and its own opt-in policy.
var Batteries = []model.OptInRule[string]{
    {For: model.KindApp, Owner: model.KindService, Blocked: []string{"github.com/krewire/cloud/service"}},
}
```

The generic parameters are the point: `Matrix` is generic over the owning package
type, and `OptInRule` over the import-path type, so the kernel never hard-codes a
module path.

**Adding a module to the ecosystem never edits this module.**

### The split

- **Rules in `model`** — `Kind`, `Project`, `Scope`, `DomainEvent`, and the
  generic shapes above.
- **Mechanics in `lifecycle`** — boot, dispatch, and shutdown.
- **Primitives in `errs`** — how both report failure.
- **Values in `version`** — the `Version` type and the semver rules that judge
  whatever a module declares.

## Rules the kernel enforces for the layers above

| Rule | Enforced by |
|---|---|
| Imports may only point downward | `TestKWL_LAYER_001_DownwardImportsOnly` |
| The kernel is the floor | `TestKWL_LAYER_002_KernIsTheFloor` |
| The kernel names no ecosystem module | `TestKERN_LAYER_001_NoEcosystemNames` |
| `model` imports only stdlib and this module | `TestKWL_CORE_050_ImportBoundary` |
| Compatibility is caret-semver | `version.Version.IsCompatible` |
| Config is `krewire.yaml` only | `model.ValidateKrewireYamlPath` |
| Project names are kebab-case | `model.Project.Validate` |
| Env is `local`/`production`/`testing` | `env.ParseEnv` |

The layering guards derive everything from `go.work` and each `go.mod` at run
time, so a new module is placed in the stack automatically and no guard needs an
edit.

## Usage

```go
import "github.com/krewire/kern"

p := kern.Project{Name: "my-app", Kind: kern.KindApp, ConfigPath: "krewire.yaml"}
if err := p.Validate(); err != nil {
    return err
}
```

## Tests

```bash
gofmt -l .       # must print nothing
go vet ./...
go test ./...
go build ./...
```

The root package carries structural guards that keep the layer honest, and both
fail CI if the invariant breaks:

- `TestKWL_CORE_050_ImportBoundary` — `model` may import only the standard
  library and this module's own packages.
- `TestKWL_LAYER_001_DownwardImportsOnly` — no module in the workspace may
  import a module above it in the layering, with documented exceptions.

## License

MIT — see [LICENSE](./LICENSE).