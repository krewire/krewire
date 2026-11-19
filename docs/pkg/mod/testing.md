# Package Reference: `testing`

Import: `github.com/krewire/krewire/pkg/testing`

## 1. Purpose & Architecture

The `pkg/testing` package delivers testing assertions, fluent HTTP request builders, chainable response expecters, golden snapshot assertions, UI component render harnesses, and headless browser test runners.

For complete browser testing and end-to-end integration examples, see [Testing & Browser Automation Guide](../platform/testing.md).

---

## 2. Main API Reference

- **Assertions**: `Equal(t, got, want)`, `NoError(t, err)`, `HasError(t, err)`, `Contains(t, s, substr)`, `NotContains(t, s, substr)`, `True(t, cond, msg)`.
- **Fluent HTTP**:
  - `test.Request(t)` / `test.GET(t, path)` / `test.POST(t, path, body)` / `test.JSONRequest(t, method, path, payload)`.
  - `test.Do(t, handler, req)` executing against any `http.Handler`.
  - `.Status(code)`, `.Header(key, want)`, `.Contains(text)`, `.JSON(&dst)`.
- **Snapshots**: `AssertGolden(t, goldenPath, actualBytes)`.
- **Headless Browser**: `browser.New(t, serverURL)` for verifying rendered HTML and client WASM widgets.

---

## 3. Usage Example

```go
package main_test

import (
    "net/http"
    "testing"
    ftest "github.com/krewire/krewire/pkg/testing"
)

func TestHealthCheck(t *testing.T) {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"healthy"}`))
    })

    req := ftest.GET(t, "/health").Request()
    ftest.Do(t, handler, req).
        Status(http.StatusOK).
        Header("Content-Type", "application/json").
        Contains("healthy")
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: In-process test execution; prevents accidental external socket exposure during automated test runs.
- **Sustainable (S2)**: Pure Go without heavyweight third-party test runners or node-based testing tools.
- **Scalable (S3)**: Unified test contracts across HTTP APIs, WebAssembly islands, and CLI tools.
