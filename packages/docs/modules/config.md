# `config`

Import: `github.com/krewire/libs/config`

## Purpose

Typed YAML configuration loading and environment overlays for Krewire projects.

## Main API

- `Load(path, dst)`
- `LoadOrDefault(path, dst)`
- `Override(dst, lookup, options...)`
- `WithPrefix(prefix)`
- `ParseDotEnv(data)` and `LoadDotEnv(path)`
- `LoadVars(path)` and `Vars`

Configuration precedence is zero-value defaults, then file values, then environment values. Later sources win.

## Example

```go
var cfg ServerConfig
if err := config.Load("krewire.yaml", &cfg); err != nil { return err }
if err := config.Override(&cfg, os.LookupEnv, config.WithPrefix("APP_")); err != nil { return err }
```

## Design boundary

`config` handles decoding and overlays. Domain validation belongs to [`core`](./core.md) and struct-tag validation belongs to [`validation`](./validation.md).
