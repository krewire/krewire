# `cloud/apphost`

Import: `github.com/krewire/krewire/cloud/apphost`

Status: Experimental (introduced for #136, .NET Aspire parity).

## Purpose

One declarative **application resource graph** and a **local orchestrator**
for it. A project declares every resource it needs locally — services,
workers, gateways, databases, caches, queues, object stores and external
dependencies — in `krewire.yaml`. The orchestrator validates the graph,
starts resources in dependency order, waits for readiness, injects service
discovery variables, supervises restarts, aggregates logs, reports
machine-readable state and stops everything in reverse order without
orphaning child processes.

The same `Topology` value is the contract for deterministic tooling (`krew`)
and AI-assisted workflows: both read, validate, plan and execute it through
this package.

## Schema (`topology:` in krewire.yaml, version 1)

```yaml
topology:
  version: 1                 # required; SchemaVersion
  name: shop
  resources:
    db:
      kind: database          # service|worker|gateway|database|cache|queue|objectstore|external
      run: {command: postgres, args: [-D, .krewire/pg]}   # or container: / external:
      endpoints: [{name: tcp, scheme: postgres, port: 5432}]
      readiness: {exec: {command: pg_isready}, interval: 200ms, timeout: 30s}
    cache:
      kind: cache
      container: {image: redis:7}            # needs docker/podman or a custom Driver
      endpoints: [{name: tcp, scheme: redis, targetPort: 6379}]   # port 0 = allocate
    billing:
      kind: external
      external: {url: https://billing.example.com}
    api:
      kind: service
      run: {command: go, args: [run, ./cmd/api]}
      endpoints: [{name: http, scheme: http}]
      references: [db, cache]                # inject discovery env + depend
      dependsOn: [billing]                   # order only
      env:
        PORT: ${self.port}
        DATABASE_URL: ${db.url}
        REDIS_ADDR: ${cache.tcp.host}:${cache.tcp.port}
        LITERAL: $$not-an-expression
      readiness: {http: {endpoint: http, path: /healthz}}
      restart: {policy: on-failure, maxRestarts: 3, backoff: 500ms}
      shutdownTimeout: 10s
    worker:
      kind: worker
      run: {command: go, args: [run, ./cmd/worker]}
      references: [cache]
```

| Field | Meaning |
| --- | --- |
| `run` / `container` / `external` | Exactly one; selects the `Driver` (`process`, `container`) or marks a resource that is probed but never started. |
| `endpoints` | Named ports. `port: 0` is allocated at start; fixed ports are checked for collisions (validation) and for being free (preflight). Containers need `targetPort`. |
| `env` | Values may use `${resource.url\|host\|port}`, `${resource.endpoint.url\|host\|port}` and `${self...}`; `$$` escapes `$`. Referenced resources become dependencies. |
| `references` | Dependency plus discovery variables `KREWIRE_<REF>_URL/_HOST/_PORT` and `KREWIRE_<REF>_<EP>_URL/_HOST/_PORT`. Every resource also gets `KREWIRE_RESOURCE`, `KREWIRE_KIND`, `KREWIRE_SELF_<EP>_PORT`. |
| `dependsOn` | Startup ordering without discovery variables. |
| `readiness` | One of `tcp: <endpoint>`, `http: {endpoint, path}`, `exec: {command, args}`; `interval` (200ms) and `timeout` (30s). Managed resources with endpoints default to TCP on the first endpoint; resources without endpoints are ready once launched. |
| `restart` | `never` (default), `on-failure`, `always`; `maxRestarts` (3), exponential `backoff` (500ms, capped at 30s). |
| `shutdownTimeout` | Grace period after SIGTERM before the process group is killed (10s). |

## Validation

`Parse`/`LoadFile`/`Topology.Validate` return a `*ValidationError` listing
every `Issue` with a stable `Code` before anything starts:
`unsupported-version`, `invalid-name`, `invalid-kind`, `invalid-runner`,
`invalid-endpoint`, `missing-dependency`, `dependency-cycle`,
`port-collision`, `invalid-env-reference`, `invalid-probe`,
`invalid-restart`. `Start` adds `port-in-use` during preflight.

## API

```go
top, err := apphost.LoadFile("krewire.yaml")      // parse + normalize + validate
plan, _ := apphost.NewPlan(top)                    // waves, stop order (JSON)
fmt.Print(plan.Explain())                          // human text
prop, err := apphost.ValidateProposal(top, edited) // diff + plan, or issues

o, _ := apphost.New(top, apphost.Options{
    LogSink:  os.Stdout,           // "resource | line"
    Observer: obs,                 // Event{Type, Resource, Phase, Message}
    Approver: approver,            // gates Start (never Stop/rollback)
    Recorder: rec, TaskID: "t-42", // journal entries for the evidence record
})
err = o.Start(ctx)       // *StartError{Resource, Stage, Message, Hint, LogTail, RolledBack}
st := o.Status()         // State{Running, Resources: []ResourceStatus} (JSON)
lines, _ := o.Logs("api", 100)
err = o.Stop(ctx)        // reverse order; errors.Is(err, apphost.ErrShutdownTimeout)
// or: o.Run(ctx) — start, block until ctx is done, stop.
```

Guarantees:

- Start order is `Topology.Levels()` (waves, names sorted); each resource is
  launched only after all its dependencies passed readiness. Stop is the
  exact reverse.
- A failure at `approval`, `preflight`, `launch` or `readiness` stops the
  failing resource and then everything already started, newest first, and
  returns a `*StartError` naming what was rolled back plus the last 20 log
  lines and a hint.
- Process resources run in their own process group (unix). Stop sends
  SIGTERM to the group, waits `shutdownTimeout` (capped by the `ctx`
  deadline), then SIGKILLs the group; when a process exits on its own the
  rest of its group is reaped too, so no grandchildren are orphaned.
- `Stop` interrupts an in-flight `Start`, is idempotent, and is never gated
  by the Approver.

## Extension points

- `Driver` (`Start(ctx, LaunchSpec) (Process, error)`) adds runners for system
  services or other adapters through `Options.Drivers`; no resource kind is
  hard-coded in the orchestrator. `ContainerDriver` (docker/podman CLI) is
  installed automatically only when a container resource exists and a CLI is
  on `PATH`. Discovery values use host addresses; containers reaching host
  services need a host-gateway address.
- `Observer` and `Recorder` must not block; they are called synchronously.

## Exposing it from `krew` (not implemented here)

The CLI lives in the `krew` repository and should stay a thin shell over this
package — no orchestration logic in the command layer:

| Command | Library call |
| --- | --- |
| `krew app validate [--json]` | `LoadFile` → print `ValidationError.Issues` |
| `krew app plan [--json]` / `krew app explain` | `NewPlan` → JSON or `Plan.Explain()` |
| `krew app up` | `New` + `Run(signalCtx)` with `LogSink: os.Stdout`; write `Status()` JSON to `.krewire/apphost/state.json` on every `Observer` event |
| `krew app status [--json]` / `krew app logs <resource> [-n]` | read the state file / a control socket served by `up` that wraps `Status()` and `Logs()` |
| `krew app down` | signal the `up` process (SIGTERM) so its `Stop` runs the ordered shutdown |

AI-native workflows use the same calls: `ValidateProposal` before applying an
agent-proposed topology, an `Approver` backed by the shared policy engine
(`Operation.Intent()` yields a `model.ActionIntent`), and a `Recorder` that
attaches `OperationRecord`s to the task's evidence record. Agents get
`Plan`, `State` and `StartError` as structured context and must not launch
processes outside the orchestrator, so readiness, shutdown and cleanup
guarantees always apply.

## Integration notes

- **Observability (#139):** `Event` (phase changes, failed probes,
  operations) and `ResourceStatus` (phase, ready, restarts, last probe and
  error) are the health/status surface; an adapter implementing `Observer`
  can map them onto the standard health model without new hooks here.
- **Dependency graph (#135) / typed contracts (#137):** `Topology.Levels`,
  `Dependents` and `Dependencies` expose the graph; typed service contracts
  can be attached later as endpoint metadata without changing the schema
  version for existing fields.
- **Infrastructure lifecycle (#138):** cloud provisioning of the same
  resources can implement `Driver`, or feed `external:` resources with the
  provisioned URLs.
