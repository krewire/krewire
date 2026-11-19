# Declarative Core Model (`pkg/model`)

Import: `github.com/krewire/krewire/pkg/model`

## Purpose

The `pkg/model` package defines the pure data models, canonical workload matrix, scope hierarchies, licensing tiers, and invariant validation rules for the Krewire ecosystem.

---

## Invariants & Dependencies

- **Layer 1 Placement**: Sits directly on top of `pkg/errs` and the Go standard library.
- **Zero Third-Party Dependencies**: Standard-library-only.
- **Hermetic Invariant**: Contains zero string literals or imports referencing higher-layer packages.

---

## Key Types & Definitions

### 1. Workload Kinds (`model.Kind`)
The eight canonical software workloads supported across the ecosystem:
```go
const (
	KindApp     Kind = "app"     // Fullstack dynamic web applications
	KindCLI     Kind = "cli"     // Standalone command-line tools
	KindSite    Kind = "site"    // Static websites (SSG)
	KindBook    Kind = "book"    // Multi-chapter documentation books
	KindWorker  Kind = "worker"  // Supervised background queue workers
	KindService Kind = "service" // Headless backend microservices
	KindGateway Kind = "gateway" // API reverse proxies
	KindInfra   Kind = "infra"   // Declarative cloud infrastructure blueprints
)

var AllKinds = []Kind{KindApp, KindCLI, KindSite, KindBook, KindWorker, KindService, KindGateway, KindInfra}
```

### 2. Scope Hierarchy (`model.Scope`)
Ordered coarse to fine: `Workspace` < `Module` < `Domain` < `Service` < `Unit`.
```go
const (
	ScopeWorkspace Scope = "Workspace"
	ScopeModule    Scope = "Module"
	ScopeDomain    Scope = "Domain"
	ScopeService   Scope = "Service"
	ScopeUnit      Scope = "Unit"
)
```

### 3. Open-Core Tiers (`model.Tier`)
Defines operational capability tiers:
```go
const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierTeam       Tier = "team"
	TierEnterprise Tier = "enterprise"
)

var AllTiers = []Tier{TierFree, TierPro, TierTeam, TierEnterprise}
```

### 4. Project Validation (`model.Project`)
Validates project names, safe paths, and configuration anchors:
```go
type Project struct {
	Name       string `json:"name"`
	ModulePath string `json:"modulePath"`
	Kind       Kind   `json:"kind"`
	ConfigPath string `json:"configPath"`
}

// Invariants verified by (*Project).Validate():
// - Name is non-empty and follows kebab-case (^[a-z][a-z0-9-]*$)
// - ModulePath is non-empty and safe against directory traversal (CWE-22)
// - Kind is canonical when specified
```

### 5. Workload Matrix & Opt-In Constraints
```go
type OptInRule[P ~string] struct {
	For     Kind
	Owner   Kind
	Blocked []P
}

func ViolatesOptIn[P ~string](kind Kind, imported []P, rules []OptInRule[P]) bool
```

---

## Example Usage

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/pkg/model"
)

func main() {
	proj := model.Project{
		Name:       "payment-gateway",
		ModulePath: "github.com/example/gateway",
		Kind:       model.KindGateway,
	}

	if err := proj.Validate(); err != nil {
		fmt.Printf("Invalid project: %v\n", err)
		return
	}

	fmt.Printf("Project %s (%s) is valid.\n", proj.Name, proj.Kind)
}
```
