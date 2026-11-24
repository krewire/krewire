# Testing & Quality Assurance Guide

Standards, patterns, test harnesses, and quality gate commands for verifying the Krewire codebase and generated workloads.

---

## 1. Testing Philosophy & Standards

Krewire mandates rigorous, deterministic test coverage across all layers:
- **Colocated Tests**: Unit tests live directly alongside source files in `*_test.go`.
- **Zero Third-Party Test Frameworks**: Tests utilize Go's standard `testing` package with optional helpers from `testing`. No complex assertion DSLs or external runner dependencies.
- **Deterministic & Parallel-Safe**: Tests must not rely on wall-clock timing or external internet connectivity; use `t.Parallel()` wherever state is isolated.
- **Pure Go Execution**: All tests run under `CGO_ENABLED=0`.

---

## 2. Test Suites in the Monorepo

### A. Layering & Architectural Invariant Tests
Verified by [`layering_test.go`](../../conformance/layering_test.go):
- `TestFoundationLayering`: Asserts that foundation packages (`errs`, `model`, `version`, `env`, `lifecycle`, `log`) import only the standard library and lower-ranked foundation packages.
- `TestFoundationNamesNoHigherPackage`: Verifies that foundation packages name zero higher-layer packages in string literals.
- `TestPackagesNeverImportKiw`: Ensures no foundational domain module ever imports the developer CLI (`krew`).
- `TestUIDoesNotImportWeb`: Validates that `ui` does not import `web`.
- `TestSQLiteDriverIsOptIn`: Verifies that third-party database drivers (sqlite, postgres, mysql, redis) reside exclusively in `hub/*`.
- `TestWebDoesNotImportORMOrCloud`: Enforces that `web` never imports `orm` or `cloud`.

### B. Standard Library Boundary Verification
Ensures leaf packages maintain a hermetic boundary without external bloat:
- `errs/dependency_test.go` (`TestImportBoundary`): Verifies standard library only.
- `model/dependency_test.go` (`TestModel_StdlibBoundary`): Verifies standard library + `errs` only.

---

## 3. Testing Web Handlers & Routers

Test HTTP handlers using standard `net/http/httptest`:

```go
package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/krewire/web"
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

## 4. End-to-End Browser Automation (`testing/browser`)

For testing server-side rendered pages and interactive WebAssembly widgets:

```go
package e2e_test

import (
	"testing"
	"github.com/krewire/krewire/testing/browser"
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

## 5. In-Memory Filesystem Mocking (`fs`)

Isolate disk operations without leaving temporary files on host runners:

```go
package file_test

import (
	"testing"
	"github.com/krewire/krewire/fs"
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

The quality-gate contract below is executable: `.github/workflows/ci.yaml`
runs exactly these commands on every pull request, every push to `main`, the
merge queue, and (via `workflow_call`) before every stable and nightly
release. Each gate is a separate job, so a red check names the gate that
failed. No secrets or model-provider credentials are needed.

| CI check (job name) | Command | Notes |
|---|---|---|
| `gofmt` | `gofmt -l .` | Must print nothing. |
| `mod-tidy` | `go mod tidy -diff` and `go mod verify` | Module files are tidy; downloaded dependencies match `go.sum`. |
| `vet` | `go vet ./...` | |
| `architecture` | `go test -count=1 -v . ./modules/errs ./modules/model` | Layering, peer-decoupling, pure-Go and wasm platform guards (`layering_test.go`, `platform_test.go`) plus the stdlib-boundary tests. |
| `ai-native` | `go test -count=1 -v -run 'TestReferenceEngineConforms\|TestSpawnedEngineProcess\|TestEndToEndFlow' ./modules/toolchain/{fakeengine,adapter,workflow}` | AI-native criterion: the toolchain contract conformance suite against the deterministic reference engine in-process, over stdio and as a spawned process, plus the end-to-end workflow, with `KREW_ENGINE` and model API keys empty. |
| `test (min)`, `test (stable)` | `CGO_ENABLED=0 go test -count=1 ./...` | `min` = the `go` directive in `go.mod` (canonical minimum, currently 1.27.1); `stable` = latest Go release. |
| `race` | `CGO_ENABLED=1 go test -race -count=1 ./...` | The race detector needs cgo; this is the only gate that enables it. No package is excluded. |
| `build-purego` | `CGO_ENABLED=0 go build ./...` | Clean-checkout build: `GOFLAGS=-mod=readonly`, `GOWORK=off`, `GOTOOLCHAIN=local`. |
| `build-wasm` | `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath ./modules/runtime/...` | Dedicated WebAssembly target check. |
| `release-snapshot` | `.github/scripts/build-release.sh v0.0.0-ci dev <dir>` twice | Every release target builds, two builds are bit-for-bit identical, and `ai --version` / `--help` work. |
| `ci-ok` | — | Aggregate: succeeds only if every gate above succeeded. |

**Branch protection.** Require the `ci-ok` check on `main` (it stays stable
when individual gates are added or renamed). The per-gate names above are also
stable and may be required individually.

Run the same gates locally before pushing:

```bash
gofmt -l .                                   # must print nothing
go mod tidy -diff && go mod verify
go vet ./...
go test -count=1 -v . ./modules/errs ./modules/model # architecture guards
CGO_ENABLED=0 go test ./...
go test -race ./...                          # needs cgo
CGO_ENABLED=0 go build ./...
GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./modules/runtime/...
```

Only deterministic command results satisfy these checks; an agent reporting
success is never a substitute for a green gate. Cross-module compatibility
(`krew compat`) runs from the `krew` toolchain repository and is not part of
this repository's CI.

Run filtered tests via `krew`:
```bash
krew test --filter TestSupervisor
```
