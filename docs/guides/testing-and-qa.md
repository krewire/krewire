# Testing & Quality Assurance Guide

Standards, patterns, test harnesses, and quality gate commands for verifying the Krewire codebase and generated workloads.

---

## 1. Testing Philosophy & Standards

Krewire mandates rigorous, deterministic test coverage across all layers:
- **Colocated Tests**: Unit tests live directly alongside source files in `*_test.go`.
- **Zero Third-Party Test Frameworks**: Tests utilize Go's standard `testing` package with optional helpers from `pkg/testing`. No complex assertion DSLs or external runner dependencies.
- **Deterministic & Parallel-Safe**: Tests must not rely on wall-clock timing or external internet connectivity; use `t.Parallel()` wherever state is isolated.
- **Pure Go Execution**: All tests run under `CGO_ENABLED=0`.

---

## 2. Test Suites in the Monorepo

### A. Layering & Architectural Invariant Tests
Verified by [`layering_test.go`](../../layering_test.go):
- `TestFoundationLayering`: Asserts that foundation packages (`errs`, `model`, `version`, `env`, `lifecycle`, `log`) import only the standard library and lower-ranked foundation packages.
- `TestFoundationNamesNoHigherPackage`: Verifies that foundation packages name zero higher-layer packages in string literals.
- `TestPackagesNeverImportKiw`: Ensures no library in `pkg/` ever imports the developer CLI (`krew`).
- `TestUIDoesNotImportWeb`: Validates that `pkg/ui` does not import `pkg/web`.
- `TestSQLiteDriverIsOptIn`: Verifies that only `pkg/orm/sqlite` imports the modernc SQLite driver.
- `TestWebDoesNotImportORMOrCloud`: Enforces that `pkg/web` never imports `pkg/orm` or `pkg/cloud`.

### B. Standard Library Boundary Verification
Ensures leaf packages maintain a hermetic boundary without external bloat:
- `pkg/errs/dependency_test.go` (`TestImportBoundary`): Verifies standard library only.
- `pkg/model/dependency_test.go` (`TestModel_StdlibBoundary`): Verifies standard library + `pkg/errs` only.

---

## 3. Testing Web Handlers & Routers

Test HTTP handlers using standard `net/http/httptest`:

```go
package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/pkg/web"
)

func TestRouter_GetRoute(t *testing.T) {
	r := web.NewRouter()
	r.Get("/health", func(c *web.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
```

---

## 4. End-to-End Browser Automation (`pkg/testing/browser`)

For testing server-side rendered pages and interactive WebAssembly widgets:

```go
package e2e_test

import (
	"testing"
	"github.com/krewire/krewire/pkg/testing/browser"
)

func TestApp_PageInteraction(t *testing.T) {
	b := browser.New(t)
	b.Navigate("http://localhost:8080")
	b.AssertTextVisible("Welcome to Krewire")
	b.Click("#cta-button")
	b.AssertTextVisible("Success")
}
```

---

## 5. In-Memory Filesystem Mocking (`pkg/fs`)

Isolate disk operations without leaving temporary files on host runners:

```go
package file_test

import (
	"testing"
	"github.com/krewire/krewire/pkg/fs"
)

func TestInMemoryFileSystem(t *testing.T) {
	memFS := fs.NewMemory()

	err := memFS.WriteFile("config.json", []byte(`{"key":"value"}`), 0o644)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := memFS.ReadFile("config.json")
	if err != nil || string(data) != `{"key":"value"}` {
		t.Fatalf("unexpected content: %s", string(data))
	}
}
```

---

## 6. Mandatory Quality Gates

Before committing changes or creating a pull request, run all four quality gates:

```bash
# 1. Format verification (must produce no output)
gofmt -l .

# 2. Static analysis & vet
go vet ./...

# 3. Unit, integration, and layering test suite
go test ./...

# 4. Binary build verification
go build -o bin/krew ./github.com/krewire/krew/cmd/krew
```

Run filtered tests via `krew`:
```bash
krew test --filter TestSupervisor
```
