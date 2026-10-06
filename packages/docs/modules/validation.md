# `validation`

Import: `github.com/krewire/libs/validation`

## Purpose

Reflection-based, stdlib-only struct validation using `validate` tags. The package is framework-agnostic and shared by web, CLI, and configuration consumers.

## Main API

- `Struct(value)` validates exported fields recursively.
- `Field(value, tag)` validates a scalar or collection value.
- `ValidationError` aggregates field failures.
- `FieldError` describes field, value, and failed rule.
- `validation/rules` provides built-in rules and `rules.Register` for extensions.

Supported rules include `required`, `omitempty`, `min`, `max`, `len`, `email`, `pattern`, and `oneof`.

## Example

```go
type User struct {
    Email string `validate:"required,email"`
    Role  string `validate:"oneof=admin viewer"`
}
if err := validation.Struct(User{Email: "user@example.com", Role: "viewer"}); err != nil {
    return err
}
```

## Design boundary

The package validates local struct constraints. Cross-field business invariants and workload rules belong to [`core`](./core.md). Malformed or unknown rules return errors; they do not panic.
