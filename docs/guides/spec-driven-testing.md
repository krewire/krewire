# Spec-Driven Testing Cheat Sheet (`KWL-TEST-P8M4L`)

A practical guide for developers and AI agents on writing, tagging, and running spec-driven tests across the Krewire ecosystem.

---

## 1. Traceability Architecture

Every requirement from a Specification Issue (`[Spec: <SpecID>]`) is deterministically linked to code and tests.

```
GitHub Spec Issue (SSOT)
       │
       ▼
Test File Header (`// Tests for <SpecID>`)
       │
       ▼
Test Function (`// Spec: <SpecID> <ReqID> Scope: <Scope>`)
       │
       ▼
Quality Gate / CI (`kiw test --spec <SpecID>` or `go test -run Test<ReqID>`)
```

---

## 2. Test File Tagging Standards

### A. File Header
The first line of any test file must declare the SpecID(s) it covers:
```go
// Tests for KWL-TEST-P8M4L, KWL-KERN-X8P3L
package mypkg_test
```

### B. Individual Test Functions
Each test function cites its requirement ID and scope level:
```go
// Spec: KWL-TEST-P8M4L KWL-TST-002 Scope: Unit
func TestSpecRequirement_Traceability(t *testing.T) {
    // ...
}
```

Or using the naming convention:
```go
func TestKWL_TST_002_SpecRequirement_Traceability(t *testing.T) {
    // ...
}
```

---

## 3. Scope-Driven Placement Rules (`KWL-TST-011`)

| Scope Level | Implementation Location | Test Location | Test Nature |
|---|---|---|---|
| **Workspace** | `tools/kiw/internal/commands` | `tools/kiw/internal/commands/*_test.go` | End-to-end / CLI integration |
| **Module** | Repository Root / Monorepo Root | Root integration tests | Multi-package boundary |
| **Domain** | `packages/<domain>/` | `packages/<domain>/*_test.go` | Domain business logic |
| **Service** | `apps/<service>/` | `apps/<service>/*_test.go` | Service boundary |
| **Unit** | Any Package | Same package `*_test.go` | Scalar & pure function unit tests |

---

## 4. Execution & Verification Commands

```bash
# Run all tests
kiw test
go test ./...

# Filter tests by Spec ID
kiw test --spec KWL-TEST-P8M4L

# Filter by requirement regex
go test -run TestKWL_TST_002 ./...

# Audit spec coverage across codebase
rg -n "Spec: KWL-TEST-P8M4L"
```

---

## 5. Mandatory Quality Gates

Before any commit or PR:
```bash
gofmt -l .        # Must print nothing
go vet ./...      # Must pass
go test ./...     # Must pass
go build -o bin/kiw ./tools/kiw/cmd/kiw
```
