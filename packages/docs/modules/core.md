# `core`

Import: `github.com/krewire/libs/core`

## Purpose

Shared declarative domain primitives for the Krewire ecosystem: project kinds, workloads, scopes, versions, identifiers, project invariants, domain events, and compatibility checks.

## Main API

- `Kind`, `Workload`, `Project`
- `Scope`, `SpecID`, `RequirementID`
- `Version`, `ParseVersion`, `CheckCompatibility`
- `ParseKind`, `ParseScope`, `ParseSpecID`
- `DomainEvent`, `NewDomainEvent`
- `ValidateKrewireYamlPath`, `HasOptInViolation`

`core` recognizes `app`, `cli`, `site`, `book`, `worker`, `service`, `infra`, and `kernel`. Scope hierarchy is `Workspace → Module → Domain → Service → Unit`.

## Example

```go
kind, err := core.ParseKind("service")
if err != nil { return err }
workload, ok := core.WorkloadFor(kind)
if !ok { return fmt.Errorf("unsupported kind: %s", kind) }
_ = workload
```

## Design boundary

`core` is the declarative center and should not depend on application frameworks or perform I/O. Observability aliases remain for compatibility; new logging, diagnostics, stack, and error code integrations should use [`vein`](./vein.md).
