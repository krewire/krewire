# Package: `pkg/model` (Declarative Domain Models)

The `pkg/model` library declares pure data structures, validation invariants, workload matrices, scope hierarchies, and licensing tiers for the Krewire ecosystem.

---

## 1. Zero-Dependency & Layering Invariant

`pkg/model` is strictly standard-library-only plus `pkg/errs`. It defines declarative rules without coupling to any execution runtime, network driver, or database.

- **Downwards-only**: Sits at Layer 1 in the ecosystem dependency ladder.
- **Zero Lateral Coupling**: Contains zero string literals or imports referencing higher-level packages.

---

## 2. Key Primitives

### Workload Kind (`Kind`)
Enumerates the eight canonical workload kinds:
```go
const (
	KindApp     Kind = "app"
	KindCLI     Kind = "cli"
	KindSite    Kind = "site"
	KindBook    Kind = "book"
	KindWorker  Kind = "worker"
	KindService Kind = "service"
	KindGateway Kind = "gateway"
	KindInfra   Kind = "infra"
)

var AllKinds = []Kind{KindApp, KindCLI, KindSite, KindBook, KindWorker, KindService, KindGateway, KindInfra}
```

### Scope Hierarchy (`Scope`)
Codifies architectural boundaries from coarse to fine:
```text
Workspace  ───►  Module  ───►  Domain  ───►  Service  ───►  Unit
```

- **`ScopeWorkspace`**: The top-level workspace root (`go.work`).
- **`ScopeModule`**: A distributable Go module (e.g. `github.com/krewire/krewire`).
- **`ScopeDomain`**: A DDD bounded context grouping services within a module.
- **`ScopeService`**: A deployable runtime process (`main` package).
- **`ScopeUnit`**: The smallest code unit: a Go package, file, type, or function.

### Ecosystem Tiers (`Tier`)
Represents the open-core tier ladder:
```go
const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierTeam       Tier = "team"
	TierEnterprise Tier = "enterprise"
)

var AllTiers = []Tier{TierFree, TierPro, TierTeam, TierEnterprise}
```
All code in the core repository is free-tier open-source (`TierFree`).

### Domain Events (`DomainEvent`)
Represents cross-module decoupled domain events:
```go
type DomainEvent struct {
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
	At      time.Time `json:"at"`
}

event := model.NewDomainEvent("user.registered", map[string]string{"id": "usr_123"})
```

### Project Validation (`Project`)
Ensures project metadata satisfies naming and security invariants:
- **Project Name**: Enforces kebab-case (`^[a-z][a-z0-9-]*$`).
- **Module Path**: Validates safe module paths against directory traversal (`CWE-22`).
- **Configuration SSOT**: `ConfigFileName = "krewire.yaml"`.

```go
proj := model.Project{
    Name:       "my-service",
    ModulePath: "example.com/my-service",
    Kind:       model.KindService,
}

if err := proj.Validate(); err != nil {
    return err
}
```

### Workload Matrix & Opt-In Batteries (`OptInRule`)
Prevents unintended package bloat across kinds:
```go
type OptInRule[P ~string] struct {
	For     Kind
	Owner   Kind
	Blocked []P
}

func ViolatesOptIn[P ~string](kind Kind, imported []P, rules []OptInRule[P]) bool
```
Lightweight kinds (like `cli` or `site`) are guarded against accidentally importing heavy batteries owned by `app` or `worker` unless explicitly opted in.
