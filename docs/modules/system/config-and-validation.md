# Package: `config` & `validation` (Configuration & Struct Validation)

The `config` and `validation` libraries handle type-safe YAML configuration loading, environment overrides, variable expansion, and struct-level validation rules across the Krewire ecosystem.

Both libraries operate with standard library primitives, zero external framework locks, and pure Go (`CGO_ENABLED=0`).

---

## 1. Architectural Overview

```text
krewire/
├── config/             # YAML configuration loader & environment variable overlay
│   ├── config.go       # Load, LoadOrDefault, Override, WithPrefix
│   └── env.go          # Custom environment lookup & type coercion
└── validation/         # Reflection-based struct validation engine
    ├── validation.go   # Struct(v), Field(val, tag), ValidationError, FieldError
    └── rules/          # Built-in validation rule validators
        ├── required.go # required rule (non-zero value check)
        ├── min.go      # min=N rule (numeric minimum, string/slice min length)
        ├── max.go      # max=N rule (numeric maximum, string/slice max length)
        ├── len.go      # len=N rule (exact length matching)
        ├── email.go    # email rule (RFC 5322 format)
        ├── oneof.go    # oneof=a b c rule (whitelist enum check)
        └── pattern.go  # pattern=regex rule (regular expression match)
```

---

## 2. Declarative Config Loader (`config`)

`config` enforces strict precedence:
$$\text{Built-in Defaults} < \text{YAML File} < \text{Environment Variables}$$

### API Reference

- **`config.Load(path string, dst any) error`**: Unmarshals YAML from `path` into `dst` (must be a non-nil pointer to a struct). If the file does not exist, leaves `dst` untouched and returns `nil`.
- **`config.LoadOrDefault(path string, dst any) error`**: Tolerates absent configuration files, populating defaults.
- **`config.Override(dst any, lookup func(string) (string, bool), opts ...Option) error`**: Applies environment variables on top of `dst`.
- **`config.WithPrefix(prefix string) Option`**: Customizes the environment prefix (default: `APP_`).

### Struct Tag Mapping & Environment Inheritance

Fields are mapped using standard Go struct tags:
- `yaml:"field_name"`: Maps YAML keys.
- `env:"EXACT_ENV_KEY"`: Directly binds an explicit environment variable name.
- Untagged fields derive an implicit environment key: `<PREFIX>_<NESTED_STRUCT>_<FIELD_NAME>` (e.g. `Server.Port` becomes `APP_SERVER_PORT`).

### Complete Config Example

```go
package main

import (
    "fmt"
    "os"
    "github.com/krewire/krewire/config"
    "github.com/krewire/krewire/validation"
)

type ServerConfig struct {
    Host string `yaml:"host" env:"HOST" validate:"required"`
    Port int    `yaml:"port" env:"PORT" validate:"required,min=1,max=65535"`
}

type AppConfig struct {
    AppName string       `yaml:"app_name" validate:"required"`
    Env     string       `yaml:"env" env:"APP_ENV" validate:"oneof=local staging production"`
    Server  ServerConfig `yaml:"server"`
}

func main() {
    // 1. Initialize struct with sensible defaults
    cfg := AppConfig{
        AppName: "Krewire Service",
        Env:     "local",
        Server: ServerConfig{
            Host: "127.0.0.1",
            Port: 8080,
        },
    }

    // 2. Load from krewire.yaml (or custom config file)
    if err := config.Load("config.yaml", &cfg); err != nil {
        panic(err)
    }

    // 3. Overlay environment variables using APP prefix
    if err := config.Override(&cfg, os.LookupEnv, config.WithPrefix("APP")); err != nil {
        panic(err)
    }

    // 4. Validate all invariants
    if err := validation.Struct(&cfg); err != nil {
        panic(fmt.Sprintf("Invalid configuration: %v", err))
    }

    fmt.Printf("Configuration loaded successfully: %s running on %s:%d [%s]\n",
        cfg.AppName, cfg.Server.Host, cfg.Server.Port, cfg.Env)
}
```

---

## 3. Struct Validation Engine (`validation`)

`validation` validates Go structs and scalar fields using `validate` tags without external heavy validator dependencies.

### Supported Rules

| Rule | Syntax | Description | Example |
| :--- | :--- | :--- | :--- |
| `required` | `validate:"required"` | Value must not be zero/empty/nil | `Name string validate:"required"` |
| `min` | `validate:"min=5"` | Numeric $\ge N$, string/slice length $\ge N$ | `Age int validate:"min=18"` |
| `max` | `validate:"max=100"` | Numeric $\le N$, string/slice length $\le N$ | `Title string validate:"max=100"` |
| `len` | `validate:"len=8"` | String or slice must have exact length $N$ | `Code string validate:"len=6"` |
| `email` | `validate:"email"` | Value must be a valid RFC 5322 email | `Email string validate:"required,email"` |
| `oneof` | `validate:"oneof=a b c"` | Value must equal one of whitespace-separated choices | `Role string validate:"oneof=admin dev user"` |
| `pattern`| `validate:"pattern=^[a-z]+$"` | Value must match compiled regular expression | `Slug string validate:"pattern=^[a-z0-9-]+$"` |

### Rule Combination & Multiple Constraints

Combine multiple rules using comma delimiters:

```go
type UserRegistration struct {
    Username string `json:"username" validate:"required,min=3,max=32,pattern=^[a-zA-Z0-9_-]+$"`
    Email    string `json:"email"    validate:"required,email"`
    Role     string `json:"role"     validate:"required,oneof=developer admin observer"`
    Age      int    `json:"age"      validate:"min=13,max=120"`
}
```

### Inspecting Detailed Validation Errors

When validation fails, `validation.Struct` returns a `*validation.ValidationError` detailing all failing fields:

```go
package main

import (
    "errors"
    "fmt"
    "github.com/krewire/krewire/validation"
)

func ValidateInput(u UserRegistration) {
    err := validation.Struct(u)
    if err == nil {
        return
    }

    var valErr *validation.ValidationError
    if errors.As(err, &valErr) {
        for _, f := range valErr.Fields {
            fmt.Printf("Field %q failed rule %q (got: %v, param: %q)\n",
                f.Field, f.Rule, f.Value, f.Param)
        }
    }
}
```

---

## 4. Scalar Field Validation (`validation.Field`)

For runtime query parameters, form inputs, or individual variables without a containing struct:

```go
err := validation.Field("invalid-email-address", "required,email")
if err != nil {
    fmt.Println("Invalid email provided!")
}
```

---

## 5. Architectural Alignment with the Three Pillars (3S)

- **Secure (S1)**: Invariants are validated at the edge before untrusted data touches databases, command executors, or file systems.
- **Sustainable (S2)**: Pure standard library reflection; no heavy third-party dependencies, zero CGO.
- **Scalable (S3)**: Single unified validation and configuration model shared across HTTP handlers, CLI flags, workers, and background jobs.
