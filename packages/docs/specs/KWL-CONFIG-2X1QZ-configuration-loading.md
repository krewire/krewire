# Specification — Configuration Loading

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWL-2X1QZ                                   |
| Title       | Configuration Loading                       |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Libraries — Configuration                   |

## 1. Context

Krewire applications (CLI tools, servers, static builders) all need typed
configuration resolved from files and the environment. Today each consumer
hand-rolls loading: `krewire` parses `krewire.yaml` in `internal/siteconfig`,
scaffolded projects receive env handling from nothing, and upcoming fullstack
apps would each re-implement precedence and struct binding.

`config` is a Krewire Libraries package providing one shared, framework-agnostic
loading model: a YAML file bound to a typed struct via `yaml` tags, overridden
by environment variables via `env` tags, with a fixed precedence. It never
imports `framework` or `net/http`, so any Go program in the ecosystem can use
it.

The package also hosts the ecosystem's flat key-value store (`Vars`) and
`.env` loading primitives, absorbing the former `libs/cfg` package so that
configuration resolution has exactly one home.

## 2. Problem Statement

Configuration in the Krewire ecosystem is duplicated and inconsistent:

- Every project reimplements "read file, overlay env, bind to struct".
- No common precedence rule means LDAP-style surprises: env flags, file
  values, and defaults combine differently per project.
- Struct-focused configuration (not `map[string]any`) keeps settings typed,
  validated, and documented, but requires a shared loader to be practical.

The result: config code is copy-pasted, precedence is ad hoc, and settings
drift out of sync with documentation. `config` removes that duplication.

## 3. Goals

- G1 — Bind configuration to typed structs using `yaml` and `env` tags.
- G2 — Overlay sources in a fixed precedence order.
- G3 — Treat a missing file as an empty config, never an error.
- G4 — Remain framework-agnostic: no `framework` imports, no HTTP concerns.
- G5 — Stay stdlib-first; add only `gopkg.in/yaml.v3` for YAML decoding.
- G6 — Produce deterministic, wrapped, debuggable errors.

## 4. Non-Goals

- NG1 — Config file watching or hot reload.
- NG2 — More serialization formats beyond YAML in this phase.
- NG3 — Remote secrets managers or encryption.
- NG4 — Config validation rules (see KWL-LHANF); precedence is separate.
- NG5 — Defaults expressed *outside* the target struct (zero values are the base).

## 5. Requirements

### 5.1 Package & Loading

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| CFG-LD-001  | Provide package `config` in `github.com/krewire/libs/config`.      | Must     |
| CFG-LD-002  | Provide `Load(path string, dst any) error` reading a YAML file and unmarshalling into `dst` (a pointer to a struct). | Must   |
| CFG-LD-003  | A missing file yields a zero `dst` and no error; a malformed file yields a wrapped error naming the path. | Must |
| CFG-LD-004  | File values bind by `yaml` tag name; a field without a tag binds by its lowercased field name (mirroring `yaml.v3`). | Must |
| CFG-LD-005  | Provide `Override(dst any, lookup func(string) (string, bool)) error` applying environment values on top of an already-loaded struct. | Must |

### 5.2 Environment Overlay

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| CFG-EN-001  | A field configured with `env:"SERVER_ADDR"` is overridden from the environment using that exact key. | Must |
| CFG-EN-002  | A field without an `env` tag derives its key from `yaml` tag/name, uppercased with `-`/`.` replaced by `_` and the package prefix applied (e.g. `APP_TITLE`). | Should |
| CFG-EN-003  | The environment prefix (`config.WithPrefix`) overrides the default package prefix for implicit keys. | Should |
| CFG-EN-004  | Empty environment values are treated as unset (no override).      | Must |
| CFG-EN-005  | Nested struct fields recurse; path segments join with `_`.        | Must |

### 5.3 Precedence & Semantics

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| CFG-PR-001  | Precedence is strictly: built-in zero values < file < environment. Later sources win. | Must |
| CFG-PR-002  | Slices and maps decode from YAML; slice elements are not environment-overlayed in this phase. | Must |
| CFG-PR-003  | An env value failing to decode into its field's type produces a wrapped error naming field and key. | Must |
| CFG-PR-004  | `Load` never panics on nil `dst`; it returns a usage-style wrapped error. | Must |

### 5.4 Org & Testing

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| CFG-OR-001  | Provide `config.LoadOrDefault(path, dst) error` used by tooling that tolerates absent files. | Must |
| CFG-OR-002  | Every loading rule has an `*_test.go` verifying binding, precedence, and error paths. | Must |

### 5.5 Flat Variables & Dotenv

Absorbed from the former `libs/cfg` package; requirement IDs continue the
scheme introduced by KWN-Q7X4M (`KWL-DOTV-*` → `CFG-DOTV-*`,
`KWL-CFGV-004` → `CFG-KV-005`).

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| CFG-KV-001  | Provide `Vars`, a flat `map[string]string` store with dot-notation keys, and `Get`, `Set`, `Delete`, `Keys` accessors. | Must |
| CFG-KV-002  | Provide `LoadVars(path string) (Vars, error)` flattening a YAML document into dot-notation keys; a missing file yields an empty `Vars` and nil error; a malformed file yields a wrapped error naming the path. | Must |
| CFG-KV-003  | Provide `(Vars).Save(path)` writing the map back as nested YAML, creating parent directories. | Must |
| CFG-KV-004  | Provide `(Vars).Merge`, `(Vars).WithDefaults`, and `(Vars).EnvOverride(prefix)` combining sources with later sources winning. | Must |
| CFG-KV-005  | Provide `(Vars).GetOr(key, fallback)` returning the stored value when present and non-empty, otherwise the fallback. | Must |
| CFG-DOTV-001 | Provide `LoadDotEnv(path)` exporting `KEY=VALUE` pairs into the process environment without overwriting already-set variables; a missing file is not an error. | Must |
| CFG-DOTV-002 | Provide `ParseDotEnv(data []byte) ([]DotEnvPair, error)` tolerating comments (`#`), blanks, optional `export ` prefixes, and single/double-quoted values. | Must |
| CFG-DOTV-003 | A malformed non-comment `.env` line returns an error naming the line number. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** Only Go stdlib + `gopkg.in/yaml.v3`.
- NFR3 — **Portability.** Linux, macOS, Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.
- NFR5 — **Determinism.** Identical file + env yield identical structs.

## 7. Success Criteria

- S1 — A struct with `yaml` + `env` tags loads identically in a CLI tool, a
      server, and a test.
- S2 — Precedence test proves env wins over file, file wins over zero value.
- S3 — Removing the config file produces a zero config without error.
- S4 — `config` imports no `framework` package and no HTTP code.

## 8. Related Specifications

| SpecID    | Title                                           |
| --------- | ----------------------------------------------- |
| [KWL-M1ZKS](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-M1ZKS-krewire-libraries.md) | Krewire Libraries — Initial Specification |
| [KWL-LHANF](./KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation |
| [KWN-Q7X4M](https://github.com/krewire/kiw/blob/main/docs/specs/KWN-CONF-Q7X4M-config-directory-and-dotenv.md) | Config Directory & Dotenv (consumer of `LoadDotEnv`/`Vars`) |
| [KWN-6K41E](https://github.com/krewire/kiw/blob/main/docs/specs/KWN-RUN-6K41E-krewire-run-dev-deploy.md) | krewire run/dev/deploy (consumer) |

## 9. References

- [KWF-M07QS](https://github.com/krewire/framework/blob/main/docs/specs/KWF-WEB-M07QS-krewire-web-framework.md) — Krewire Web Framework (consumer).
- `gopkg.in/yaml.v3` — YAML library used for file decoding.