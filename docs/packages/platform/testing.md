# Package: `testing` (Test Harness & Browser Automation)

The `packages/testing` library delivers testing utilities, assertions, HTTP test servers, and headless browser test runners (`KWL-TEST-P8M4L`).

---

## 1. Capabilities

- **`testing`**: Assertion helpers, temporary test directories, and mock filesystem generators.
- **`testing/browser`**: Headless browser automation harness for end-to-end testing of SSR and client WASM widgets.

---

## 2. Example Test

```go
package testing_test

import (
	"testing"
	"github.com/krewire/krewire/packages/testing/browser"
)

func TestApp_PageRender(t *testing.T) {
	b := browser.New(t)
	b.Navigate("http://localhost:8080")
	b.AssertTextVisible("Hello from Krewire Web!")
}
```
