# Package: `packages/model` (Declarative Domain Models)

The `packages/model` library declares pure data structures, validation invariants, workload matrices, and scope hierarchies for the Krewire ecosystem (`KWF-M8K2Q`, `KWL-K1N2Q`).

---

## 1. Zero-Dependency Invariant

`packages/model` is strictly standard-library-only. It defines business rules without coupling to any execution runtime, network driver, or database.

---

## 2. Key Primitives

### Workload Kind (`Kind`)
Enumerates the 8 canonical workload kinds:
```go
const (
	KindApp     Kind = "app"
	KindCLI     Kind = "cli"
	KindSite    Kind = "site"
	KindBook    Kind = "book"
	KindWorker  Kind = "worker"
	KindService Kind = "service"
	KindInfra   Kind = "infra"
	KindKernel  Kind = "kernel"
)
```

### Scope Hierarchy (`Scope`)
Codifies architectural boundaries (`KWL-ARCH-J2K9Q`):
```text
Workspace  ───►  Module  ───►  Domain  ───►  Service  ───►  Unit
```

### Opt-In Batteries (`OptInRule`)
Prevents unintended package bloat across kinds:
```go
type OptInRule[P ~string] struct {
	For     Kind
	Owner   Kind
	Blocked []P
}

func ViolatesOptIn[P ~string](kind Kind, imported []P, rules []OptInRule[P]) bool
```

### Project Validation (`Project`)
Ensures project names follow kebab-case (`^[a-z][a-z0-9-]*$`), paths are safe against traversal (CWE-22), and configuration targets `krewire.yaml`.
