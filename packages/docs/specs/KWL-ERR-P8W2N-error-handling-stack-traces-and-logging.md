# Specification — Error Handling, Stack Traces & Logging

| Field       | Value                                            |
| ----------- | ------------------------------------------------ |
| SpecID      | KWL-P8W2N                                        |
| Title       | Error Handling, Stack Traces & Structured Logging |
| Status      | Draft                                            |
| Date        | 2026-08-23                                       |
| Author      | Krewire Contributors                              |
| Domain      | Core — Diagnostics                               |

## 1. Context

Failures today surface as single-line messages: `fail()` prints `kiw: <err>`,
panics are recovered without caller context, and every binary configures
`slog.Default()` ad-hoc regardless of environment. Debugging production
incidents requires knowing *where* an error was created and *which* code path
logged it — information the current pipeline discards.

## 2. Problem Statement

Errors carry no origin metadata, so support sessions start from guesswork;
panics are logged as bare values without stacks; log format and verbosity do
not follow the environment (JSON vs text, Info vs Debug), and the web layer
ships without recovery or request logging enabled by default.

## 3. Goals

- G1 — Errors can carry an immutable, captured stack at creation/wrap point
  without breaking `errors.Is/As`.
- G2 — One canonical logger factory driven by `core.Env` + debug
  (KWL-K4T7W): text locally, JSON in production, Debug level + source when
  debugging. Lives at `github.com/krewire/libs/log` (`package log`).
- G3 — The web layer recovers panics with full stacks server-side and never
  leaks internals to clients; request logging ships enabled.
- G4 — The devtool adopts the same logger and surfaces attached stacks on
  failure when debugging.
- G5 — User-facing failures can carry one actionable hint ("run `kiw init`",
  "check `project.kind`") rendered by every Krewire entry point, closing part
  of Go's terse-error weakness at the ecosystem layer.
- G6 — A client-visible correlation id links an error response to its server
  log line, so support sessions start from evidence instead of guesswork.

## 4. Non-Goals

- NG1 — Distributed tracing export (OTLP spans); hooks exist via
  `framework/app.Tracer`, transport comes later.
- NG2 — Log shipping/aggregation integrations.
- NG3 — Retrofitting stacks onto every existing error construction site;
  adoption is incremental at failure boundaries.
- NG4 — Custom log destinations/files; stderr only (12-factor).

## 5. Requirements

| ID           | Requirement                                                                                     | Priority | Scope    |
| ------------ | ----------------------------------------------------------------------------------------------- | -------- | -------- |
| KWL-ERRV-001 | `core.WithStack(err)` captures the calling goroutine's PCs at wrap time; `errors.Is/As` and `Unwrap` pass through unchanged. | Must | Unit |
| KWL-ERRV-002 | `core.StackOf(err)` extracts frames via `errors.As`; nil when absent. `core.FormatStack` renders `func file:line` lines. | Must | Unit |
| KWL-ERRV-003 | Stacking the same error twice keeps both traces extractable.                                    | Should   | Package  |
| KWL-LOGV-004 | `logging.Setup(env, debug)` returns a stderr logger: Debug level with source when debug; JSON records in production; text otherwise. | Must | Unit |
| KWF-HTTPV-005| `RecoverMiddleware` logs the panic value together with a captured goroutine stack; clients receive a 500 without internals. | Must | Unit |
| KWF-HTTPV-006| `App` attaches recovery and access-log middleware by default; explicit `Use` composes outside them. | Must     | Package  |
| KWL-DIAGV-007| `bootRuntime` installs the environment-appropriate default logger; the devtool prints extracted stacks on failure when debug is on. | Must | Domain   |
| KWL-ERRV-008 | `core.WithAttrs(err, attrs...)` attaches structured key/value pairs to an error without altering its message or `errors.Is/As` behavior; `core.AttrsOf(err)` extracts them in order. Attrs flow into `slog` records when logged through `libs/log`. | Must | Unit |
| KWL-ERRV-009 | `core.Error` gains an optional hint; `core.WithHint(err, text)` attaches one to any error and `core.HintOf(err)` returns the nearest hint through the wrap chain. Hints are actionable next steps, never internal detail. | Must | Unit |
| KWL-ERRV-010 | `core.FormatTree(err)` renders a human error tree: message chain top-down, each link annotated with its creation file:line when available, attrs inline, nearest hint as footer. The devtool's failure path prints this tree by default; `--debug` appends full `FormatStack` output. | Must | Unit |
| KWF-HTTPV-011| Error responses (including recovered panics) carry a short correlation id in the body; recovery and error logs log the same id beside the stack, so one id links client report to server trace. | Must | Unit |

## 6. Non-Functional Requirements

- NFR1 — `libs/core` stays stdlib-only (`runtime` permitted).
- NFR2 — Zero allocation on the happy path of middleware beyond one wrapper
  writer per request.
- NFR3 — Quality gates pass in every touched repo (`gofmt`, `vet`, `test`).

## 7. Success Criteria

- S1 — A panicking route returns HTTP 500 with body `internal server error`
  while the server log shows the panic plus a stack naming the handler frame.
- S2 — `kiw run --debug` on a failing project logs at Debug with source
  references; without `--debug` nothing below Info appears.
- S3 — `errors.Is(wrapped, target)` remains true through `WithStack`.
- S4 — A scaffolded project missing `krewire.yaml` fails with a tree whose
  footer reads a hint like "run `kiw init` to create one" — without debug
  flags and without reading source.
- S5 — A panicking route returns HTTP 500 whose body contains the same
  correlation id that appears in the server log line holding the panic stack.

## 8. Related Specifications

| SpecID    | Title                                    |
| --------- | ---------------------------------------- |
| KWL-M1ZKS | Krewire Libraries — Initial Specification |
| KWL-K4T7W | Environments & Debug Mode                |
| KWL-W0J2X | Core Errors & Exit Codes                 |
| KWN-6K41E | krewire run/dev/deploy                    |
