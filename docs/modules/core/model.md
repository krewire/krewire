# Package: `model` (Declarative Domain Models)

The `model` library declares pure data structures, validation invariants, workload matrices, scope hierarchies, and licensing tiers for the Krewire ecosystem.

---

## 1. Zero-Dependency & Layering Invariant

`model` is strictly standard-library-only plus `errs`. It defines declarative rules without coupling to any execution runtime, network driver, or database.

- **Downwards-only**: Sits at Layer 1 in the ecosystem dependency ladder.
- **Zero Lateral Coupling**: Contains zero string literals or imports referencing higher-level packages.

---

## 2. Key Primitives

### Project Identity (`Project`)
Represents the declarative project descriptor:
```go
type Project struct {
	Name       string
	Module     string
	ConfigPath string
}
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
}

if err := proj.Validate(); err != nil {
    return err
}
```
