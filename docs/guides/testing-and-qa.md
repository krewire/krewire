# Testing & Quality Assurance Guide

Standards for writing unit and integration tests, running quality gates, and verifying code coverage.

---

## 1. Writing Tests

Krewire tests follow standard Go testing conventions using the standard `testing` package and `pkg/testing` test helpers:
- Place unit tests in `*_test.go` files adjacent to the code they verify.
- Test suites must be deterministic, isolated, and parallel-safe where appropriate.

---

## 2. Test Execution

Run tests dynamically using `kiw test` or `go test`:
```bash
# Run all tests
kiw test

# Filter tests by pattern
kiw test --filter TestSupervisor

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
