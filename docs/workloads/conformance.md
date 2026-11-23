# Workload Conformance & Qualification Evidence

Workload maturity labels in the [maturity matrix](./maturity-matrix.md) are backed by an executable, end-to-end conformance harness, not by declarations. The harness lives in [`internal/conformance`](../../internal/conformance) (tracked by krewire/krewire#141).

---

## 1. What a fixture proves

Every workload has one maintained fixture under [`internal/conformance/testdata/<workload>`](../../internal/conformance/testdata). For each one, the harness:

| Step | Gate ID suffix | What is checked |
|---|---|---|
| Scaffold | `scaffold` | Renders the real starter template (`templates/init/{static,book,app,cli}` or the pre-init `templates/new/bootstrap`) into a **clean temporary directory** and adds the fixture's developer code and `krewire.yaml` battery keys. |
| Resolve | `resolve` | `go mod tidy` against this checkout (`replace` to the repo root), so the fixture builds the code under test, not a published release. |
| Static analysis | `vet` | `go vet ./...` on the scaffolded project. |
| Build | `build` | `CGO_ENABLED=0 go build -trimpath`; the binary's SHA-256 is recorded. |
| Validate config | `validate` | `fixture check` loads `krewire.yaml` through the framework. |
| Compile / run | `compile`, `state`, `run`, `reproducible` | Workload-specific: SSG output, book output, infra state round-trip, CLI command and bit-for-bit reproducible rebuild. |
| Infra lifecycle | `plan`, `apply`, `drift` | `pkg/cloud/infra` engine with the `localfile` provider: deterministic plan, approval-gated apply (unapproved apply must be refused), no-op re-plan, drift classification without mutation, reconcile to in-sync. |
| Start | `start` | Process starts and prints `READY`. |
| Representative behaviour | `probe.*` | HTTP probes (pages, APIs, proxying, health), worker job processing and retry, provider lifecycle. |
| Graceful shutdown | `shutdown` | `SIGINT` → exit 0 and `STOPPED` within 15 s. |
| Diagnostics | `diagnose.*` | A common invalid configuration or missing dependency must fail with a specific diagnostic. |

Fixtures only drive the framework and print `READY`/`STOPPED` markers; every assertion lives in the harness. A failed required gate blocks every later gate of that workload (recorded as `BLOCKED`), except behavioural probes, which never block shutdown and diagnostics.

**No fake passes.** A lifecycle step that core cannot exercise is recorded as an *exclusion* with a reason and the issue that owns it, and the workload is reported `partial` (never `passed`). A workload with no gates is `failed`.

---

## 2. Running it

```bash
# Part of the normal suite (skipped with -short); fails on regressions only.
go test ./internal/conformance/

# Restrict and keep the evidence.
KREWIRE_CONFORMANCE_WORKLOADS=site,book KREWIRE_CONFORMANCE_OUT=out go test ./internal/conformance/

# Release qualification runner: writes out/report.json and out/summary.md.
go run ./internal/conformance/cmd/qualify -out out           # regression gate
go run ./internal/conformance/cmd/qualify -out out -strict   # also requires every gate to pass
go run ./internal/conformance/cmd/qualify -update-baseline   # deliberately raise the floor
```

Fixtures run in parallel (`min(4, NumCPU)`) with a per-workload timeout (default 5 minutes). A warm run of all eight takes roughly 10–20 seconds. CI runs it in [`.github/workflows/conformance.yml`](../../.github/workflows/conformance.yml), which uploads the evidence as a build artifact and can be called by the release pipeline with `strict: true`.

---

## 3. Evidence format

`report.json` (`api_version: conformance.krewire.dev/v1`, `kind: QualificationReport`) records:

- `commit`, `dirty`, `module`, `version`, `toolchain` (Go version, GOOS/GOARCH, `CGO_ENABLED`), `started_at`, `duration_ms`;
- per workload: `declared_maturity` (parsed from the matrix), `evidenced_maturity`, `status`, `end_to_end`, `gates`, `artifacts` (path, SHA-256, size), `exclusions`, `findings`;
- `ai_workflow`: the integrated AI-native workflow qualification, reported **separately**;
- `workload_qualification` and the combined `release_readiness` verdict.

Each gate is a [`model.VerificationEvidence`](../../pkg/model/evidence.go) (ID, type, command, status, exit code, bounded output, duration, and an evidence hash bound to the commit), evaluated with `model.EvaluateTaskVerification`. The gates therefore drop directly into the AI-native toolchain `EvidenceReport.verification.gates` defined by #131/#140 without conversion.

### Integration points

- **AI-native workflow (#131/#140).** The `pkg/toolchain` contracts and the deterministic fake engine are not on `main`, so `ai_workflow` is `excluded` and `release_readiness` stays false. Once they land, the harness runs the mock-backed workflow through the unified entrypoint and attaches its `EvidenceReport`.
- **Workload list and probes (#139).** The harness keeps no workload list of its own: `Workloads` is derived from `pkg/observe.Workloads` (reports follow that order) and the HTTP probe paths come from `observe.ProfileFor`.

---

## 4. Regression gate

[`internal/conformance/baseline.json`](../../internal/conformance/baseline.json) is the committed floor: for each workload, its status and the gate IDs that passed. The gate fails when a workload's status drops or a baseline gate stops passing. Newly passing gates are reported as improvements; raising the baseline is a deliberate, reviewed change (`-update-baseline`).

Known failing gates stay visible in the report. They are not hidden and do not count as passes.

---

## 5. Objective promotion criteria

| Level | Evidence required |
|---|---|
| `Unsupported` | Explicitly out of scope; no fixture is maintained. |
| `Planned` | Specified in an ADR or roadmap; no fixture, or every lifecycle step is excluded. |
| `Experimental` | Fixture exists and scaffold, resolve, vet, build, and validate pass; lifecycle exclusions or failing probes are allowed but listed. |
| `Usable` | `status: passed` and `end_to_end: true` on `main`: every gate, including startup, probes, graceful shutdown, and diagnostics, passes with no exclusions, and the gates are in the baseline. |
| `Stable` | `Usable`, plus gates this harness does not yet produce: cross-module compatibility (#133), a reproducible benchmark baseline, verified production use, and `strict` qualification on two consecutive releases. |

The harness reports `evidenced_maturity` as the highest level its evidence supports, capped at `Usable`. Any declared level above that is listed as a finding.

### Coverage

No coverage percentage is claimed. A coverage threshold becomes a promotion criterion only after CI enforces it.
