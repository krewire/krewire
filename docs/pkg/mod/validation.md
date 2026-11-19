# Package Reference: `validation`

Import: `github.com/krewire/krewire/pkg/validation`

## 1. Purpose & Architecture

The `pkg/validation` package provides a framework-agnostic, reflection-based struct and scalar validation engine in 100% standard-library Go. It allows HTTP handlers, CLI commands, and domain models to share one unified validation rule syntax without external dependencies.

For deep validation rules and custom rule registration, see [Configuration & Validation Guide](../system/config-and-validation.md).

---

## 2. Main API Reference

- **`validation.Struct(v any) error`**: Validates a struct or struct pointer against its `validate` tags. Returns `nil` on success or `*validation.ValidationError` detailing failing fields.
- **`validation.Field(value any, tag string) error`**: Validates an individual scalar value against a rule string.
- **Built-in Rules**:
  - `required`: Non-zero value.
  - `min=N` / `max=N`: Minimum/maximum numeric value or string/slice length.
  - `len=N`: Exact length.
  - `email`: Valid RFC 5322 email address format.
  - `oneof=a b c`: Value must be one of the listed choices.
  - `pattern=regex`: Value must match regular expression.

---

## 3. Usage Example

```go
package main

import (
    "fmt"
    "github.com/krewire/krewire/pkg/validation"
)

type SignupPayload struct {
    Username string `validate:"required,min=3,max=30"`
    Email    string `validate:"required,email"`
    Role     string `validate:"required,oneof=user admin"`
}

func main() {
    payload := SignupPayload{
        Username: "al",
        Email:    "invalid-email",
        Role:     "superuser",
    }

    err := validation.Struct(payload)
    if err != nil {
        fmt.Println("Validation failed:", err)
    }
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Validates invariants before untrusted user input reaches SQL builders, shell executors, or filesystem storage.
- **Sustainable (S2)**: Pure Go standard library; no bloated external dependencies.
- **Scalable (S3)**: Reusable across web APIs, TUI command arguments, and background job payloads.
