# AI-Native Toolchain Contract (v1)

Normative specification of the contract between the unified `krew` toolchain,
its deterministic commands, and AI engines. Decision record:
[ADR-T1C4A](../../adr/ADR-T1C4A-unified-ai-native-toolchain-contract.md).
Issues: #131, #140.

| Item | Value |
|---|---|
| API version | `toolchain.krewire.dev/v1` |
| Contract version | `1.0.0` (SemVer; engines negotiate on the major) |
| Go packages | `toolchain/contract`, `projctx`, `policy`, `adapter`, `workflow`, `fakeengine`, `conformance` |
| JSON Schemas | `toolchain/contract/schemas/<Kind>.json` (draft 2020-12, generated from the Go types) |
| Examples | `toolchain/contract/testdata/golden/*.json` (validated by tests) |

## 1. Command model

One executable, `krew`. Every command is classified; the classification is
published as the `CommandCatalog` document (`contract.Catalog()`).

| Class | Meaning |
|---|---|
| `deterministic` | Never needs an engine, model credentials, or network access to an AI provider. |
| `agentic` | Delegates reasoning to an engine through the adapter; without one it fails with `engine_unavailable`. |
| `orchestrated` | Runs the whole lifecycle; deterministic stages run without an engine. |

Deterministic commands:

| Command | Stage | Contract I/O |
|---|---|---|
| `krew version` | — | prints toolchain, framework, and contract versions |
| `krew doctor` | — | — |
| `krew context` | discover | produces `ProjectContext` (`--json`), lints it (`--lint`, exits non-zero on error diagnostics) |
| `krew config` | — | — |
| `krew compat` | — | consumes `ProjectContext` |
| `krew contract schema` | — | prints `schemas/<Kind>.json` |
| `krew contract validate` | — | validates any contract document (`contract.Decode`) |
| `krew task validate` | — | validates a `TaskBrief` against the context |
| `krew fmt`, `krew vet`, `krew test`, `krew build` | verify | quality-gate commands |
| `krew verify` | verify | runs the brief's gates, emits `EvidenceReport` verification |
| `krew report` | report | renders or checks an `EvidenceReport` |
| `krew engine status` | — | shows the configured engine and runs `initialize` |
| `krew release` | — | consumes `ProjectContext` |
| `krew completion`, `krew self-update` | — | — |

Agentic and orchestrated commands:

| Command | Stages | Contract I/O |
|---|---|---|
| `krew plan` | plan, decompose | `ProjectContext` + `TaskBrief` → `Plan` |
| `krew agent` | implement | + `Plan` → `ExecutionRecord`s |
| `krew review` | review | findings and claims |
| `krew chat` | — | interactive, bound to `ProjectContext` |
| `krew flow` | all | → `Plan`, `ExecutionRecord`s, `EvidenceReport` |

`krew --version` and `krew version` must print the contract version next to
the toolchain version. Existing `krew` commands not listed keep their meaning;
the catalog lists the commands the contract depends on.

## 2. Lifecycle

`discover → plan → decompose → implement → verify → review → report`
(`contract.Stages`). Each stage is recorded in the report with its actor
(`toolchain` or `engine:<name>`), status (`ok`, `failed`, `skipped`), duration,
and error. Discover, decompose, verify, and report are always run by the
toolchain. Verification is never delegated to an engine.

## 3. Documents

All documents carry `api_version` and `kind`. Unknown fields are rejected.

- **ProjectContext** — module identity, toolchain identity, workload
  kind/maturity, package inventory with internal imports, dependencies, graph
  reference (`nodes`, `edges`, `digest`), Three Pillars, rules (architecture,
  pillar, convention, decision), redacted configuration, quality gates (argv,
  env, `fail_on_output`), compatibility (Go version, toolchain, targets),
  diagnostics. Every fact carries `provenance` (`path`, `key`, `line`,
  `schema`). `fingerprint` hashes all inputs; unchanged inputs give
  byte-identical output.
- **TaskBrief** — objective, expected outcome, scope (`allowed`, `prohibited`,
  `commands`), `depends_on`, constraints, risk, `max_autonomy`, approval policy
  (`plan`, `actions`, `approvers`), verification (`gates`, `condition:
  all_required_passed`), deliverables, limits (timeout, tool-call budget,
  retries, network).
- **Plan** — ordered steps (`depends_on` may reference only earlier steps;
  targets must be in scope). The host computes `digest`; plan approvals bind to it.
- **Policy** — host-side: file writes, network, destructive-command guard,
  categories that always need approval, tool-call budget, tool timeout,
  output cap.
- **ApprovalRecord** — HMAC-authenticated `sec.ApprovalSignal` bound to a
  scope: `task:<id>:plan:<digest>` or `task:<id>:call:<call digest>`. Reports
  keep `verified` and drop `token`.
- **ExecutionRecord** — one tool call (content replaced by `content_sha256`),
  the decision, status (`applied`, `denied`, `failed`), exit code, redacted
  output, file mutation.
- **EvidenceReport** — outcome (`verified`, `verification_failed`, `denied`,
  `cancelled`, `engine_unavailable`, `failed`), stages, approvals,
  executions, mutations, verification (raw `model.VerificationEvidence`,
  `verified`, ledger block hash), claims, findings.
- **CommandCatalog** — §1.

### Claims versus evidence

`verification.verified` must equal `model.EvaluateTaskVerification` over the
raw gates. Engine statements are `claims` with a `source`; a claim may set
`verified_by` only to a gate that passed. The host clears any other citation
before writing the report.

## 4. Adapter protocol

Transport: JSON-RPC 2.0, one UTF-8 JSON message per line, engine stdin/stdout.
Engines log to stderr only. The toolchain launches the command line in
`KREW_ENGINE` (or links an `adapter.Engine` in-process).

| Method | Params | Result |
|---|---|---|
| `initialize` | `contract_version`, `host` | `engine`, `methods`, `ready`, `reason` |
| `plan` | `meta`, `context`, `task` | `plan`, `claims` |
| `implement` | `meta`, `context`, `task`, `plan`, `step_id`, `previous` | `calls`, `claims`, `more` (call again for this step after the host executes `calls`) |
| `review` | `meta`, `context`, `task`, `plan`, `executions`, `verification` | `findings`, `claims` |
| `shutdown` | — | `null`; the engine exits after in-flight requests end |
| `$/cancelRequest` | notification `{"id": n}` | the request fails with `cancelled` |

- `initialize` must come first; a major-version mismatch fails with
  `version_unsupported`. `ready: false` (no credentials/provider) makes the
  host report `engine_unavailable` for agentic stages only.
- `meta.timeout_ms` is the remaining deadline; the engine abandons the request
  with `timeout`. The host also stops waiting at its own deadline.
- Engines propose `calls` (`write_file`, `delete_file`, `run_command`,
  `network_fetch`); they never mutate the workspace themselves.

### Error model

`contract.Error{code, message, data}` is the JSON-RPC error object. Codes are
append-only:

| Code | Name | Code | Name |
|---|---|---|---|
| -32700 | `parse_error` | -32011 | `engine_unavailable` |
| -32600 | `invalid_request` | -32012 | `cancelled` |
| -32601 | `method_not_found` | -32013 | `timeout` |
| -32602 | `invalid_params` | -32014 | `scope_violation` |
| -32603 | `internal_error` | -32015 | `approval_required` |
| -32010 | `version_unsupported` | -32016 | `policy_denied` |
| -32017 | `invalid_document` | -32018 | `verification_failed` |

## 5. Policy

For each proposed call the host checks, in order: tool known; writes enabled;
path inside `scope.allowed`, outside `scope.prohibited`, and inside the
workspace (`sec.SafePath`); commands are a declared gate or match a
`scope.commands` prefix and pass the destructive-command guard; network
allowed by both policy and task; approval present when the category is in
`policy.approval_required` or `task.approval.actions`; the tool-call budget
is not exhausted. Output is redacted (`sec.RedactSensitiveOutput` plus the
host's secrets) and truncated.

## 6. Context lint codes

Only `error` blocks CI (`TestProjectContextLint`, `krew context --lint`).

| Code | Class | Severity | Meaning |
|---|---|---|---|
| `KWC1001` | MISSING | error | go.mod is missing |
| `KWC1002` | MISSING | error | go.mod lacks module path or go directive |
| `KWC1003` | MISSING | warning | AGENTS.md missing; default gates apply |
| `KWC1004` | MISSING | warning | no Quality Gates block; default gates apply |
| `KWC1005` | MISSING | info | no root `Version` declaration |
| `KWC2001` | CONFLICT | error | krewire.yaml `project.version` ≠ version.go |
| `KWC2002` | CONFLICT | error | quality gate declared twice |
| `KWC2003` | CONFLICT | error | rule ID declared twice |
| `KWC3001` | STALE | warning | maintained doc invokes the retired `kiw` |
| `KWC4001` | MALFORMED | error | go.mod line cannot be parsed |
| `KWC4002` | MALFORMED | error | krewire.yaml is not valid YAML |
| `KWC4003` | MALFORMED | error | version is not SemVer |
| `KWC4004` | MALFORMED | warning | Go source cannot be parsed |

## 7. Conformance

An engine conforms when `conformance.Run` and `conformance.RunStdio` pass
against it: compatible `initialize`; a valid, in-scope plan for the fixture
task; known, in-scope tool calls; attributed claims that never cite a failed
gate; prompt cancellation. `fakeengine` is the reference implementation;
`toolchain/adapter` also runs the suite against a spawned engine process.

## 8. Documented flow

```text
krew context --lint                       # discover (deterministic)
krew task validate task.json              # deterministic
krew plan task.json > plan.json           # engine
krew agent task.json --plan plan.json     # engine proposes, krew authorizes and executes
krew verify task.json                     # deterministic gates, no LLM
krew review && krew report                # engine claims kept apart from evidence
krew flow task.json                       # all of the above in one command
```

The same flow is executed end to end, without an LLM, by
`toolchain/workflow` tests (`TestEndToEndFlow`).
