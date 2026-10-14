# Testing & Quality Assurance Guide

Standards for writing spec-driven tests, running quality gates, and verifying code coverage (`KWL-TEST-P8M4L`).

---

## 1. Spec-Driven Testing

Every feature originates from a formal Specification ID:
- Prefixes: `KWF-*` (Framework), `KWN-*` (Devtool CLI), `KWL-*` (Libraries/Packages), `KWG-*` (Boost/Guild).
- In test files, cite the spec at the file header:
  ```go
  // Tests for KWL-KERN-X8P3L
  package kern_test
  ```
- Individual test functions cite their requirement ID:
  ```go
  // Spec: KWL-KERN-X8P3L REQ-001 Scope: Unit
  func TestSupervisor_GracefulShutdown(t *testing.T) {
      // ...
  }
  ```

---

## 2. Test Execution Filters

Filter tests dynamically by spec ID or requirement ID using `kiw test`:
```bash
# Run all tests
kiw test

# Filter by spec
kiw test --spec KWL-KERN-X8P3L

# Run via go test regex
go test -run TestSupervisor ./...
```

---

## 3. Mandatory Quality Gates

Before pushing code or submitting a PR, the following quality gates must pass cleanly:
```bash
gofmt -l .        # Must output nothing
go vet ./...      # Must pass with zero errors
go test ./...     # All tests must pass
go build -o bin/kiw ./tools/kiw/cmd/kiw # CLI must compile cleanly
```
