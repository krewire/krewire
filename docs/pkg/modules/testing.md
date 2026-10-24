# `testing`

Import: `github.com/krewire/krewire/pkg/testing`

## Purpose

Unified test harness, assertions, temporary sandboxes, HTTP test servers, and headless browser automation.

## Subpackages

- **`testing`**: Assertion helpers, temp file sandbox management, and test lifecycle hooks.
- **`testing/browser`**: Headless browser automation harness for end-to-end SSR and WASM verification.

## Example

```go
package mytest_test

import (
	"testing"
	"github.com/krewire/krewire/pkg/testing/browser"
)

func TestPage_Render(t *testing.T) {
	b := browser.New(t)
	b.Navigate("http://localhost:8080")
	b.AssertTextVisible("Krewire")
}
```

## Design boundary

Testing utility suite. See the full [Testing Reference](../platform/testing.md).
