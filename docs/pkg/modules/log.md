# Structured Logging (`packages/log`)

The `packages/log` package configures structured `slog` logging tailored to target environments (`env.Env`) with source tagging and diagnostic error integration.

## Usage

```go
import (
    "github.com/krewire/krewire/pkg/env"
    "github.com/krewire/krewire/pkg/log"
)

logger := log.Install(env.EnvProduction, false)
logger.Info("application started", "port", 8080)
```
