# Package: `testing` (Test Harness & Browser Automation)

The `testing` package provides an idiomatic, pure-Go testing toolkit for the Krewire ecosystem. It includes fluent HTTP request builders, chainable response assertion matchers, temporary fixture management, golden file diffing, UI component inspection, and headless browser automation.

---

## 1. Architectural Overview & Capabilities

```text
testing/
├── test.go         # Core assertion primitives (Equal, NoError, HasError, Contains, True)
├── http.go         # Low-level request and response recorder helpers (NewRequest, Record, EqualStatus)
├── http_chain.go   # Fluent chained HTTP request builder & response assertion expecter
├── ui.go           # Component render harness and HTML DOM assertion helpers
├── golden.go       # Golden file snapshot comparison (AssertGolden)
├── file.go         # Temp directory management and filesystem fixtures
├── coverage.go     # Code coverage verification and boundary analysis
└── browser/        # Headless browser automation harness (Browser, DOM selectors, text assertions)
```

---

## 2. Fluent HTTP Testing Chain (`testing`)

The fluent HTTP client simplifies testing web handlers, middleware pipelines, and REST APIs without boilerplate error checks:

### Request Builder API

- **`test.Request(t)`**: Creates a configurable HTTP request builder.
- **`test.GET(t, path)`**: Prepares a `GET` request.
- **`test.POST(t, path, body)`**: Prepares a `POST` request with an optional `io.Reader`.
- **`test.JSONRequest(t, method, path, payload)`**: Encodes payload to JSON and sets `Content-Type: application/json`.
- **Chainable Methods**: `.Header(k, v)`, `.Query(k, v)`, `.Form(k, v)`, `.Cookie(c)`, `.WithContext(ctx)`.

### Response Expecter API

- **`test.Do(t, handler, req)`**: Dispatches the request against any `http.Handler` and returns a chainable `*Response`.
- **`.Status(code)`**: Asserts the HTTP status code.
- **`.Header(key, want)`**: Asserts the response header value.
- **`.Contains(text)` / `.NotContains(text)`**: Asserts text presence in the response body.
- **`.JSON(&target)`**: Strictly decodes response JSON into a target struct.
- **`.Cookie(name)`**: Retrieves and verifies set cookies.

### Example: Testing a Protected API Endpoint

```go
package web_test

import (
    "net/http"
    "testing"
    ftest "github.com/krewire/krewire/testing"
)

func TestUserEndpoint(t *testing.T) {
    handler := SetupServer() // Returns http.Handler

    // 1. Unauthenticated request should be rejected
    req := ftest.GET(t, "/api/profile").Request()
    ftest.Do(t, handler, req).
        Status(http.StatusUnauthorized).
        Header("Content-Type", "application/json").
        Contains("missing authentication token")

    // 2. Authenticated JSON request should succeed
    payload := map[string]string{"name": "Alice"}
    req = ftest.JSONRequest(t, "POST", "/api/profile", payload).
        Header("Authorization", "Bearer valid-token").
        Request()

    var res struct {
        Status string `json:"status"`
        ID     string `json:"id"`
    }

    ftest.Do(t, handler, req).
        Status(http.StatusCreated).
        Header("Content-Type", "application/json").
        JSON(&res)

    ftest.Equal(t, res.Status, "ok")
}
```

---

## 3. UI Component Testing Harness (`testing`)

`testing/ui.go` enables unit testing of `.kiw` and Go UI components without running a full browser daemon:

```go
package ui_test

import (
    "testing"
    ftest "github.com/krewire/krewire/testing"
    "github.com/krewire/ui/widget"
)

func TestButtonComponent(t *testing.T) {
    btn := widget.NewButton("Submit").
        Variant(widget.VariantPrimary).
        Size(widget.SizeLarge)

    html := btn.Render()

    ftest.Contains(t, html, "data-kiw-component=\"button\"")
    ftest.Contains(t, html, "class=\"btn btn-primary btn-lg\"")
    ftest.Contains(t, html, "Submit")
}
```

---

## 4. Golden File Snapshot Testing (`testing`)

Golden files ensure rendered HTML, generated CSS, and JSON outputs do not suffer unintended regressions:

```go
package compiler_test

import (
    "testing"
    ftest "github.com/krewire/krewire/testing"
)

func TestMarkdownRendering(t *testing.T) {
    output := RenderMarkdown("# Hello World\n\nWelcome to Krewire.")
    
    // Compares output with testdata/markdown_hello.golden
    // Update snapshots by running: go test -update-golden ./...
    ftest.AssertGolden(t, "testdata/markdown_hello.golden", []byte(output))
}
```

---

## 5. Headless Browser Automation (`testing/browser`)

The `browser` package provides an automated end-to-end testing harness for checking server-side rendered pages and client-side WebAssembly / JS reactive islands.

### Features

- Zero external drivers required for standard SSR tests (hermetic HTML DOM parser).
- Seamless opt-in to full headless Chrome testing via `TEST_BROWSER=1`.
- Fast assertions for element visibility, text content, attributes, and form submissions.

### Complete Browser Test Example

```go
package e2e_test

import (
    "net/http/httptest"
    "testing"
    "github.com/krewire/krewire/testing/browser"
)

func TestFullstackWorkflow(t *testing.T) {
    // 1. Spin up an in-process test server
    srv := httptest.NewServer(SetupAppRouter())
    defer srv.Close()

    // 2. Initialize headless browser
    b := browser.New(t, srv.URL)

    // 3. Navigate and perform assertions
    b.Navigate("/")
    b.AssertStatus(200)
    b.AssertTitle("Krewire — Home")
    b.AssertTextVisible("Welcome to Krewire")

    // 4. Interact with forms
    b.FillInput("#search-box", "docs")
    b.Click("#search-submit")
    b.AssertContainsURL("/search?q=docs")
    b.AssertElementExists(".search-results")
}
```

---

## 6. Testing Philosophy & Quality Gates

Per **Pillar 1 (Secure)** and **Pillar 2 (Sustainable)**:
- **Hermetic & Fast**: All tests execute in-process using standard library constructs (`net/http/httptest`, `io/fs`). Tests run within milliseconds and require zero third-party testing daemons.
- **Pure Go**: No CGO requirements (`CGO_ENABLED=0`).
- **Deterministic**: Tests must not rely on live internet connections or unpinned remote resources.
