# Workload Observability & Diagnostics (`observe`)

Import: `github.com/krewire/krewire/observe`

## Purpose

`observe` is the one observability contract shared by all eight workload kinds and by the unified toolchain that drives them. It covers logs, traces, metrics, health, and diagnostics without a hosted service or a heavyweight runtime dependency:

- **Dependencies:** the standard library and `log` only. `observe/dependency_test.go` (`TestImportBoundary`) enforces this. There is no OpenTelemetry SDK or Prometheus client dependency: the Prometheus text format is a stable, tiny wire format, and `Family` snapshots map one-to-one onto OpenTelemetry data points, so an OTel bridge can live in an application or adapter module that already accepts that dependency.
- **Opt-in:** a nil `*Registry`, `*Counter`, `*Gauge`, `*Histogram`, or `*Health` is a valid no-op. Disabling an exporter or the whole registry never changes business logic.

---

## 1. Health model

| Probe | Question | Failure means | Default path | Runs |
|---|---|---|---|---|
| `ProbeLiveness` | Can the process make progress? | Orchestrator restarts it | `/healthz` | Liveness checks only (e.g. `Heartbeat`); never external dependencies |
| `ProbeReadiness` | Should traffic or work be routed here? | Removed from routing, **not** restarted | `/readyz` | Checks with no `Probes` set, or listing readiness |
| `ProbeStartup` | Has initialization finished? | Liveness/readiness are not acted on yet | `/startupz` | Checks listing startup |

- Verdicts are `pass`, `warn`, `fail`. An `Optional` check that fails yields `warn` and the probe still returns `200`; any required failure returns `503`.
- Before `Health.MarkStarted()` the startup and readiness probes fail with reason `starting`. `Health.MarkDraining()` fails readiness with reason `draining` while liveness keeps passing: call it first on shutdown, before the `lifecycle.Supervisor` drain.
- Checks run concurrently, each under `Check.Timeout` (default 2s), with panics isolated. Error strings are redacted before they are served.
- `Health.Heartbeat(name, maxAge)` returns a `beat()` function; liveness fails if it is not called within `maxAge`. It is the liveness source for workers.
- Every probe responds with the same `Report` JSON:

```json
{"probe":"readiness","status":"fail","checks":[{"name":"db","status":"fail","error":"dial postgres://[REDACTED]@db/app: connection refused","duration_ms":3.1}],"time":"2026-11-23T10:00:00Z"}
```

`/healthz` and `/readyz` match the paths of the existing `sec.Health` (and `web.Health`) middleware, so orchestrator configuration does not change when a workload adopts `observe.Health`.

## 2. Per-workload defaults

`ProfileFor(workload)` returns the defaults; `Mount(mux, profile, health, registry, diagnostics)` wires them.

| Workload | Mode | Probes | Metrics path | Drain | Extra correlation fields |
|---|---|---|---|---|---|
| `app` | `http` | `/healthz` `/readyz` `/startupz` | `/metrics` | 10s | `trace_id` `span_id` `request_id` |
| `service` | `http` | same | `/metrics` | 10s | same |
| `gateway` | `http` | same | `/metrics` | 15s | same |
| `worker` | `heartbeat` | same, on an optional admin listener | `/metrics` | 30s | + `job_id` `queue` |
| `site` | `static` | `GET /` | — | 0 | — |
| `book` | `static` | `GET /index.html` | — | 0 | — |
| `infra` | `exit_code` | exit status + diagnostics bundle | — | 30s | `trace_id` `span_id` `session_id` `task_id` `agent_run_id` `tool_call_id` `process_id` |
| `cli` | `exit_code` | exit status + diagnostics bundle | — | 2s | same as `infra` |

## 3. Logs

`observe.NewLogger(base, identity)` (or `observe.NewHandler`) wraps any `slog.Handler`, so formatting stays with `log.Setup`. Each record gets:

- **Identity** (stable, low-cardinality): `service`, `workload`, `resource`, `instance`, `version`, `environment`.
- **Trace:** `trace_id`, `span_id`, `parent_span_id` from the active span.
- **Correlation:** `request_id`, `session_id`, `task_id`, `agent_run_id`, `tool_call_id`, `process_id` from `observe.WithIDs`.
- **Classification (when relevant):** `origin` (`agent`, `command`, `provider`, `runtime`) and `failure_class` (`model`, `toolchain`, `workload`, `provider`).

The `Field*` constants are the only spellings. `trace_id` and `request_id` are shared with `log`, and `observe` keeps `log.WithCorrelation` in step, so existing `log.WithContext` callers see the same IDs.

### Redaction rules

Redaction is applied to messages, attributes, nested groups, `error` values, probe errors, diagnostic findings, and configuration:

- **Sensitive keys** (`IsSensitiveKey`): `password`, `secret`, `token`, `authorization`, `cookie`, `api_key`, `private_key`, `access_key`, `credential(s)`, `signature`, `dsn`, `connection_string`, `email`, `phone`, `ssn`, `card_number`, `cvv`, `iban`, `ip_address`, `client_ip`. Single words must match a whole `_` segment, so `access_token` is redacted and `input_tokens` is not.
- **Values** (`RedactString`): URL userinfo, `Bearer`/`Basic` credentials, `password=`/`token=`/`api_key=` pairs, and email addresses.
- **URLs** (`RedactURL`): userinfo removed and every query value replaced.
- Use opaque IDs for people (`session_id`), never names or email addresses.

## 4. Trace context

W3C Trace Context (`traceparent`, `tracestate`) plus W3C Baggage for correlation IDs (`krewire.task_id`, `krewire.agent_run_id`, and so on). `X-Request-ID` is carried as well, for continuity with `web.RequestID`.

| Boundary | Sender | Receiver |
|---|---|---|
| HTTP server | — | `observe.TraceHTTP(handler)` |
| HTTP client / gateway upstream | `&http.Client{Transport: observe.Transport{}}`; `proxy.Transport = observe.Transport{}` | `TraceHTTP` |
| Worker enqueue → dequeue | `queue.Enqueue(ctx, observe.WrapJob(ctx, job), opts)` | `task.Job.Run(ctx)` continues the trace |
| Serialized job payloads | Persist `TracedJob.Carrier` (a `MapCarrier`) beside the payload | `observe.Extract(ctx, carrier)` then `observe.StartSpan` |
| Child process | `cmd.Env = append(os.Environ(), observe.ChildEnv(ctx)...)` | `ctx := observe.FromProcessEnv(ctx)` |
| In-process step (tool call, job attempt) | `ctx, span := observe.StartSpan(ctx)` | — |

`TestMultiWorkloadTraceContinuity` in `observe/fixture_test.go` is the integration fixture. A toolchain task calls a gateway, which proxies to a service; the service enqueues on the real `cloud/worker` queue, and the job spawns a child process. All five log records share one `trace_id`, `task_id`, `agent_run_id`, and `request_id`, and each hop is a child span.

## 5. Metrics

`observe.NewRegistry()` provides counters, gauges, and histograms:

- Names must match `^[a-z_][a-z0-9_]*$`, counters must end in `_total`, and the unit is a name suffix (`_seconds`, `_bytes`) and is also recorded in `Desc.Unit` (`s`, `By`, `1`).
- **Cardinality bound:** each family keeps at most `DefaultMaxSeries` (200) label combinations. Extra combinations fold into one series whose labels are all `_overflow`, and the diagnostics bundle reports `metrics.cardinality_overflow`. Labels must have bounded value sets: route patterns, outcomes, queue names. Never use IDs, raw paths, or user input.
- `InstrumentHTTP` labels by the `ServeMux` pattern (`GET /orders/{id}`), never the raw path.
- **Exposition:** `MetricsHandler(r)` serves Prometheus text 0.0.4. `r.Snapshot()` returns `[]Family`, and `r.ExportTo(ctx, exporters...)` pushes snapshots to optional `Exporter`s, such as an OTLP bridge.

### Standard metrics (`StandardMetrics`)

| Name | Kind | Unit | Labels |
|---|---|---|---|
| `krewire_http_server_requests_total` | counter | 1 | `method`, `route`, `status_class` |
| `krewire_http_server_request_duration_seconds` | histogram | s | `method`, `route` |
| `krewire_http_server_in_flight_requests` | gauge | 1 | — |
| `krewire_worker_jobs_total` | counter | 1 | `queue`, `outcome` (`success` `failure` `retry` `dead_letter`) |
| `krewire_worker_job_duration_seconds` | histogram | s | `queue` |
| `krewire_worker_queue_depth` | gauge | 1 | `queue` |
| `krewire_resilience_retries_total` | counter | 1 | `operation` |
| `krewire_resilience_breaker_state` | gauge | 1 | `breaker` (0 closed, 1 half-open, 2 open) |
| `krewire_health_check_status` | gauge | 1 | `probe`, `check` (1 pass, 0.5 warn, 0 fail) |
| `krewire_toolchain_commands_total` | counter | 1 | `command`, `outcome`, `origin` |
| `krewire_toolchain_command_duration_seconds` | histogram | s | `command` |
| `krewire_process_start_time_seconds` | gauge | s | — |
| `krewire_go_goroutines` | gauge | 1 | — |
| `krewire_go_heap_alloc_bytes` | gauge | By | — |

These names are a stable contract. Renaming one is a breaking change.

## 6. Diagnostics bundle (`doctor`)

`(&observe.Diagnostics{...}).Collect(ctx)` returns a `Bundle` (schema `krewire.diagnostics/v1`) containing:

- identity, profile, correlation IDs, and `trace_id`;
- runtime info (Go version, OS/arch, CPUs, goroutines, heap, PID, uptime) and build info (module, version, VCS revision);
- all three probe reports and a metrics snapshot;
- redacted configuration (`Config`);
- `Findings`.

Each finding has a stable `code`, a `severity` (`info`, `warn`, `error`), an `origin`, a failure `class`, a redacted `message`, a runbook `hint`, and a `trace_id`. Findings are derived automatically from failing probes, draining, dead letters, retries, open breakers, and cardinality overflow. `DoctorCheck`s add custom findings. A finding that a check returns without an origin defaults to `runtime`, and findings an agent reports must use `OriginAgent` so they are never confused with measured evidence.

`Bundle.ExitCode(strict)` returns 1 on any `error` finding (or, with `strict`, on any `warn`), which is what CI gates use. `Diagnostics.Handler()` serves the bundle as JSON at `PathDiagnostics` (`/debug/diagnostics`). Mount it on an admin or loopback listener only.

## 7. Toolchain (`krew`) wiring

The CLI lives outside this repository. Wire it through this package rather than re-implementing the contract:

1. **Correlation at entry:** `ctx := observe.FromProcessEnv(ctx)`. A parent toolchain or agent then continues the trace, and a fresh root starts otherwise. Next, call `ctx = observe.WithIDs(ctx, observe.IDs{SessionID: ..., TaskID: ..., AgentRunID: ...})` with values from the shared context contract (#140).
2. **Logs:** `observe.NewLogger(log.Setup(env, debug), observe.Identity{Service: "krew", Workload: "cli", Resource: command})`.
3. **Each tool call or deterministic command:** `ctx, _ = observe.StartSpan(observe.WithIDs(ctx, observe.IDs{ToolCallID: id}))`, then run the command with `cmd.Env = append(os.Environ(), observe.ChildEnv(ctx)...)`, then call `observe.ObserveCommand(reg, name, outcome, observe.OriginCommand, elapsed)`. Report model and agent failures as `FailureModel`, build/test failures as `FailureToolchain`, and provider errors as `FailureProvider`.
4. **`krew doctor --json`:** build a `Diagnostics` for the project's workload, add `DoctorCheck`s (toolchain version, config present, compatibility), print `bundle.WriteJSON(os.Stdout)`, and exit with `bundle.ExitCode(strict)`. AI workflow reports cite `trace_id` and finding `code`s instead of restating results.
5. **Local orchestrator (#136):** poll each resource's `/startupz` and `/readyz`, decode `observe.Report`, and aggregate resource status from `Report.Status`. Non-HTTP resources use the exit code and the bundle.

See also the [Operational Visibility & Recovery guide](../guides/operational-visibility-and-recovery.md) for diagnosis runbooks.
