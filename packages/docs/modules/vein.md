# `vein`

Import: `github.com/krewire/libs/vein`

## Purpose

Preferred observability foundation: logging, diagnostics, structured attributes, hints, errors, exit codes, and stack traces.

## Main API

- `Setup(env, debug)` and `Install(env, debug)` for configured `slog.Logger`
- `WithAttrs`, `AttrsOf`, `WithHint`, `HintOf`
- `WithStack`, `StackOf`, `FormatStack`
- `FormatTree`, `ErrAttrs`, `LogError`
- `Env`, `ParseEnv`, `Envs`
- `Error`, `NewError`, `UsageError`, `FailureError`
- `ExitCode` and `ExitCodeFromInt`

## Example

```go
err := vein.WithHint(vein.NewError("invalid input", vein.ExitCodeUsage), "check --config")
vein.LogError(vein.Setup(vein.EnvLocal, true), "command failed", err)
```

## Design boundary

Use `vein` directly for new observability code. `core` keeps compatibility aliases for existing consumers. For automatic PII masking in `slog`, wrap the handler with `sec.NewPIIHandler`.
