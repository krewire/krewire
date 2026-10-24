# `boost`

Import: `github.com/krewire/krewire/pkg/boost`

## Purpose

Embedded AI-agent constitution, conventions loader, and scaffolding coordinator for the Boost multi-agent workflows.

## Main API

- `boost.Install(targetDir, opts)`: Installs the embedded Boost agent framework into `.agents/`.
- `boost.Verify(targetDir)`: Verifies agent integrity, rules, and skills configurations.
- `boost.Templates()`: Accesses embedded agent workflows, skills, and personas.

## Example

```go
package main

import (
	"github.com/krewire/krewire/pkg/boost"
)

func main() {
	_ = boost.Install(".", boost.InstallOptions{
		Overwrite: false,
	})
}
```

## Design boundary

Driven by `kiw boost install` during project initialization.
