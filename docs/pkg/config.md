# Package Reference: `config`

Import: `github.com/krewire/krewire/pkg/config`

## 1. Purpose & Architecture

The `pkg/config` package provides declarative YAML configuration loading with environment-variable overlays for all Krewire applications and CLI tools.

It enforces strict precedence:
$$\text{Built-in Struct Defaults} < \text{YAML Configuration File} < \text{Environment Variables}$$

For deep configuration schema design and validation rules, see [Configuration & Validation Guide](./system/config-and-validation.md).

---

## 2. Main API Reference

- **`config.Load(path string, dst any) error`**: Unmarshals YAML into `dst` (must be a non-nil pointer to a struct). Missing files leave `dst` unchanged and return `nil`.
- **`config.LoadOrDefault(path string, dst any) error`**: Loads YAML if present; tolerates absent files without error.
- **`config.Override(dst any, lookup func(string) (string, bool), opts ...Option) error`**: Overlays environment variables onto struct fields based on `env` tags or auto-derived uppercase keys.
- **`config.WithPrefix(prefix string) Option`**: Customizes the environment variable prefix (default: `APP`).

---

## 3. Usage Example

```go
package main

import (
    "fmt"
    "os"
    "github.com/krewire/krewire/pkg/config"
)

type Config struct {
    Port int    `yaml:"port" env:"PORT"`
    Host string `yaml:"host" env:"HOST"`
    DB   string `yaml:"db_url" env:"DATABASE_URL"`
}

func main() {
    cfg := Config{Port: 8080, Host: "127.0.0.1"}

    // 1. Load from file
    _ = config.Load("config.yaml", &cfg)

    // 2. Overlay environment variables
    _ = config.Override(&cfg, os.LookupEnv, config.WithPrefix("APP"))

    fmt.Printf("Configured on %s:%d\n", cfg.Host, cfg.Port)
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Sensitive database URLs and API keys can be passed strictly via environment variables, avoiding committed secrets.
- **Sustainable (S2)**: Zero third-party runtime frameworks; uses standard Go reflection and standard YAML parser.
- **Scalable (S3)**: Common configuration layer across microservices, static sites, background workers, and developer CLI tools.
