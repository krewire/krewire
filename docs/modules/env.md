# Target Environment Resolution (`env`)

Import: `github.com/krewire/krewire/env`

## Purpose

The `env` package defines canonical target execution environments across the Krewire ecosystem, standardizing environment names, resolution order, and validation.

---

## Canonical Environments

| Environment Constant | String Value | Purpose |
| :--- | :--- | :--- |
| **`env.EnvLocal`** | `"local"` | Local development machines (`krew dev`, local debug) |
| **`env.EnvProduction`** | `"production"` | Production hosting, containers, and live traffic |
| **`env.EnvTesting`** | `"testing"` | Automated testing (`go test`, CI test pipelines) |

The default fallback when no environment is declared is `EnvLocal`.

---

## Main API

- **`Envs() []Env`**: Returns all canonical environments in standard order (`[EnvLocal, EnvProduction, EnvTesting]`).
- **`ParseEnv(s string) (Env, error)`**: Resolves a string into a valid `Env`. Trims whitespace and evaluates case-insensitively. Returns `errs.UsageError` if unrecognized.
- **`(*Env).String() string`**: Returns the canonical lowercase string identifier.

---

## Example Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/krewire/krewire/env"
)

func main() {
	// Parse environment from environment variable or configuration
	target, err := env.ParseEnv(os.Getenv("APP_ENV"))
	if err != nil {
		fmt.Printf("Invalid environment: %v\n", err)
		return
	}

	switch target {
	case env.EnvProduction:
		fmt.Println("Running in production mode: strict security & JSON logging enabled")
	case env.EnvLocal:
		fmt.Println("Running in local mode: pretty logging & live reload active")
	case env.EnvTesting:
		fmt.Println("Running in test mode: mock services attached")
	}
}
```

---

## Design Boundaries

- **Pure Enum & Parser**: Contains zero network or disk side effects.
- **Strict Validation**: Fails fast with actionable `errs.UsageError` when an invalid environment is supplied.
