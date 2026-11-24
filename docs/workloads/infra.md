# Workload: Declarative Infrastructure (`infra`)

The `infra` workload manages infrastructure resources declaratively, in Go and
YAML, without an external DSL. Its lifecycle — `plan`, `apply`, `inspect`,
`reconcile`, and `destroy` — is executed by one canonical, deterministic engine
so that humans, CI, and AI agents all drive the same contract.

> **Maturity: Experimental.** The lifecycle contract below is implemented and
> covered by conformance, golden-plan, and failure-injection tests with the
> `localfile` reference provider. The workload stays Experimental until the
> promotion gates in §9 are met (see the [maturity matrix](./maturity-matrix.md)).

---

## 1. Packages

| Package | Role |
|---|---|
| `cloud/infra/contract` | Versioned contract (`infra.krewire.dev/v1`): desired `Resource`, recorded `Snapshot`, provider `Observed` state, `SecretRef`, `KindSchema` capability declarations, and the `Provider` interface. |
| `cloud/infra/store` | Versioned state store: exclusive locks, compare-and-swap serials, lineage, sealed checksums. `File` (local directory) and `Memory` backends. |
| `cloud/infra/engine` | Canonical engine: validation, deterministic plans, approval policy, journaled apply, drift inspection, reconciliation, destroy, evidence. |
| `cloud/infra/conformance` | Reusable provider conformance suite. |
| `cloud/infra/providers/localfile` | Reference provider: directories and files under a root directory. |
| `cloud/infra/providers/memory` | In-memory fake with failure injection and out-of-band tampering, for tests. |

The older `cloud/infra/provider`, `schema`, and `state` packages remain for
compatibility. They are not the lifecycle contract: their file state backend
does not lock and their provider interface plans inside the provider. New code
uses the packages above.

---

## 2. Model: desired, recorded, actual

- **Desired state** — `[]contract.Resource`. Each resource has a `kind`, a
  `name` (address `kind/name`), a `schemaVersion`, plain `properties`, secret
  references in `secrets`, and `dependsOn` addresses.
- **Recorded state** — `contract.Snapshot`, persisted by a store: format
  version, lineage, serial, environment, provider, resources (`ready` or
  `pending`), and a checksum.
- **Actual state** — `contract.Observed`, returned by `Provider.Read`.

`plan` compares desired with recorded state. `inspect` compares recorded with
actual state. Configuration errors (`*engine.ValidationError`) and drift
(`*engine.DriftReport`) are separate results and never mixed.

---

## 3. Provider contract (v1)

```go
type Provider interface {
    Capabilities() contract.Capabilities // contract version, provider version, kind schemas
    Read(ctx, rec ResourceState) (Observed, error)                  // no side effects; ErrNotFound if absent
    Create(ctx, r Resource, s SecretResolver) (Observed, error)       // converges an existing object
    Update(ctx, r Resource, prior ResourceState, s SecretResolver) (Observed, error)
    Delete(ctx, rec ResourceState) error                             // idempotent
}
```

Each `KindSchema` declares its schema version, required/optional properties,
which properties may come from secrets, `ForceNew` properties (replacement),
whether in-place update is supported, sensitive outputs, and risk traits
(`ExternallyVisible`, `CostIncurring`). The engine rejects providers whose
`Capabilities().Contract` is not `contract.Version`.

A provider is accepted when it passes `conformance.Run`:

```go
func TestConformance(t *testing.T) {
    conformance.Run(t, func(t *testing.T) conformance.Harness {
        return conformance.Harness{Provider: myprovider.New(...), Resources: ..., Mutate: ..., Tamper: ..., Remove: ...}
    })
}
```

---

## 4. Plan

```go
e := engine.New(localfile.New("./.krewire/infra/root"), store.NewFile(".krewire/infra/dev"), "dev")
p, err := e.Plan(ctx, desired) // or e.PlanDestroy(ctx), e.PlanReconcile(ctx, desired, drift)
data, _ := p.JSON()
```

- Planning reads state but never locks, writes state, or calls provider
  mutations.
- Actions are `create`, `update`, `replace`, `delete`, or `noop`. Deletes run
  first in reverse dependency order, then the remaining actions in dependency
  order; ties break lexically.
- Every action carries a `risk` (`none`, `low`, `high`, `destructive`), a
  reason, and field-level `changes`.
- Identical inputs produce byte-identical JSON and the same `digest`
  (`sha256:` over the canonical plan). Declaration order does not matter.
- Golden plans live in `cloud/infra/engine/testdata/`; refresh them with
  `go test ./modules/cloud/infra/engine -update`.

---

## 5. Apply, approval, and recovery

```go
res, err := e.Apply(ctx, p, engine.ApplyOptions{Approval: &engine.Approval{
    PlanDigest: p.Digest, Environment: p.Environment, Approver: "alice",
}})
```

Apply, in order:

1. Verifies the plan digest (`ErrPlanTampered`) and environment (`ErrWrongEnv`).
2. Enforces the approval policy. Plans at or above `Policy.ApprovalThreshold`
   (default `high`: destructive, externally visible, or cost-incurring
   changes) need an `Approval` bound to the exact digest and environment.
   Agent principals need approval for every change and cannot approve their
   own plans (`ErrSelfApproval`). `engine.ApprovalScope(p)` yields the scope
   string for binding an external signal such as `sec.ApprovalSignal`.
3. Takes the state lock (`*store.LockedError` names the holder).
4. Re-plans from current state under the lock; any difference returns
   `ErrStalePlan`, so a changed plan always needs renewed approval.
5. Executes actions. Each mutation is journaled: the resource is recorded as
   `pending` before the provider call and `ready` (or removed) after it.
6. Stops at the first failure, marks remaining actions `skipped`, and returns
   a `Result` with per-action status and operator `Guidance`.

Unchanged desired state yields an all-`noop` plan; applying it performs no
provider calls and no state writes.

**Recovery after partial failure.** Completed actions stay recorded and are not
repeated. The failed resource stays `pending`; the next plan emits a `pending`
action and apply re-reads the object first, then creates, updates, or deletes
as needed. If a process crashed while holding the lock, verify nothing is
running and release it with `Store.ForceUnlock(lockID)` (`krew infra unlock
<id>`). A corrupted or hand-edited state file fails its checksum
(`store.ErrIntegrity`); restore the last good copy from `history/`.

---

## 6. Drift and explicit reconciliation

```go
rep, _ := e.Inspect(ctx) // read-only: in-sync | modified | missing | pending | unreadable
p, _ := e.PlanReconcile(ctx, desired, rep) // restore desired state, using observed values as the baseline
// or
serial, _ := e.AcceptDrift(ctx, rep, engine.ApplyOptions{}) // record actual state as the new recorded state
```

`Plan` never silently absorbs drift: it reflects recorded state only.
Reconciliation is always an explicit, reviewable choice. Applying a reconcile
plan re-inspects and fails with `ErrStalePlan` if actual state moved again;
`AcceptDrift` fails with `ErrStaleDrift` on an outdated report and needs
approval when an agent runs it.

---

## 7. Secrets and redaction

- Sensitive values enter only through `secrets` references
  (`{source, key}`), resolved by a `contract.SecretResolver` at apply time.
- Validation rejects non-empty string properties whose names look sensitive
  (`password`, `token`, `secret`, `apiKey`, ...).
- Plans, snapshots, drift reports, results, and evidence store references,
  never values. Outputs named in `SensitiveOutputs` or with sensitive names are
  recorded as `(sensitive)`.
- Resolved values are scrubbed from provider error messages before they reach
  results, hints, or evidence.

---

## 8. AI-native integration and evidence

The engine, not an agent, computes the authoritative plan and executes
mutations. An agent may inspect state, explain drift, propose desired-state
changes, and request a plan; the proposal is re-planned by the engine and the
resulting digest is what a reviewer approves.

`e.NewEvidence(taskID, plan, approval, result, drift)` returns one structured
record (`infra.krewire.dev/evidence/v1`) linking the plan, approval, apply
result, and drift report to a task execution. `Evidence.Verification()` maps it
onto `model.VerificationEvidence` so infra runs appear beside build and test
gates.

**Integration notes.** The execution-context contract (#140) should carry the
`Evidence` document as the task's infra record and supply `Principal`. The
application resource graph (#136) keeps its own model; it can reference infra
resources by address (`kind/name`) and consume recorded `outputs`.

### `krew` CLI wiring (to be implemented in the krew repository)

| Command | Engine call |
|---|---|
| `krew infra plan [--env E] [--out plan.json]` | `Plan` → `Plan.JSON` |
| `krew infra apply [--env E] --plan plan.json [--approval approval.json]` | `Apply` |
| `krew infra inspect [--env E] [--json]` | `Inspect` |
| `krew infra reconcile [--env E] [--accept]` | `PlanReconcile` / `AcceptDrift` |
| `krew infra destroy [--env E]` | `PlanDestroy` → `Apply` |
| `krew infra unlock <lock-id>` | `Store.ForceUnlock` |

The CLI loads desired state from `krewire.yaml` (below), uses
`store.NewFile(".krewire/infra/<env>")`, and exits non-zero on validation
errors, missing approval, stale plans, and partial failures, printing
`errs.HintOf(err)`. `krew build --plan` and `krew deploy --target infra` should
delegate to `infra plan` and `infra apply` instead of the placeholder
`deploy.InfraDeployer`.

```yaml
infra:
  provider: localfile
  root: ".krewire/infra/root"
  resources:
    - kind: directory
      name: conf
      schemaVersion: v1
      properties: { path: conf }
    - kind: file
      name: app-env
      schemaVersion: v1
      properties: { path: conf/app.env, content: "PORT=8080\n", mode: "0640" }
      dependsOn: [directory/conf]
    - kind: file
      name: api-token
      schemaVersion: v1
      properties: { path: conf/token }
      secrets: { content: { source: env, key: API_TOKEN } }
      dependsOn: [directory/conf]
```

---

## 9. Promotion gates (Experimental → Usable)

| Gate | Status |
|---|---|
| Versioned provider contract with reusable conformance suite | Implemented (`contract`, `conformance`) |
| Deterministic, non-mutating plan with golden tests | Implemented |
| Idempotent apply for unchanged desired state | Implemented, conformance-tested |
| Locking against concurrent mutation | Implemented (`store.File`, `store.Memory`) |
| Drift reported separately from validation errors | Implemented |
| No sensitive values in plans, state, results, or evidence | Implemented, tested with a leaking fake provider |
| Recoverable partial failure with operator guidance | Implemented, failure-injection tested |
| One provider passing end-to-end lifecycle tests | `localfile` passes; a deployment-target provider (Docker or systemd) is still required |
| `krew infra` commands wired to the engine | Pending (krew repository) |
| Distributed lock backend for shared environments | Pending |
