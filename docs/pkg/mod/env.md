# Environment Resolution (`pkg/env`)

The `pkg/env` package defines canonical target execution environments across the ecosystem.

## Environments

- `env.EnvLocal` (`"local"`): Local development environment.
- `env.EnvProduction` (`"production"`): Production deployment environment.
- `env.EnvTesting` (`"testing"`): Automated testing environment.

## Functions

```go
import "github.com/krewire/krewire/pkg/env"

e, err := env.ParseEnv("production")
if err != nil {
    // handle error
}
```
