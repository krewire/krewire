# ADR T1C4A — Unified AI-Native Developer Toolchain Contract

| Field | Value |
|-------|-------|
| ID | ADR-T1C4A |
| Title | AI-Native Engineering is a first-class capability of the unified `krew` toolchain, integrated through a versioned public contract |
| Status | Proposed (awaiting maintainer approval) |
| Date | 2026-11-23 |
| Author | Krewire Contributors |
| Deciders | Krewire core maintainers |
| Context | Issues [#131](https://github.com/krewire/krewire/issues/131), [#140](https://github.com/krewire/krewire/issues/140), EPIC [#142](https://github.com/krewire/krewire/issues/142); ADR-A8I4N (AI engine), ADR-E8P9D (layering), ADR-R8C2P (workload kinds as copywriting), ADR-V8M2C (single-module core) |

## Context

Krewire deliberately combines deterministic developer tooling (discover, configure,
graph, build, test, compatibility, release) with AI-native engineering
(contextualize, plan, decompose, implement, verify, review, report) and with
governance (scope, permissions, approvals, limits, verifiable outcomes).

Before this decision the repository could be read as describing two products:
documentation alternated between `krew` and the retired `kiw`; the agent
engine had its own command surface; `krew`'s project-context loader lived in
private code; and nothing in the open core defined how an engine receives
context, is bounded by policy, or reports results. Agents therefore inferred
project facts independently and their summaries could be mistaken for
verification.

The proprietary Krew agent engine lives in its own repository under its own
license (`krewire/krew`). That boundary is intentional and must survive the
integration.

## Decision

1. **One executable.** `krew` is the only developer-facing executable. `kiw`
   is retired; no second top-level CLI is introduced. AI workflows are `krew`
   commands (`krew plan`, `krew agent`, `krew review`, `krew chat`,
   `krew flow`) beside deterministic ones (`krew context`, `krew verify`,
   `krew test`, …). The canonical hierarchy is published as data
   (`contract.Catalog()`, `CommandCatalog` schema) and classifies every command
   as `deterministic`, `agentic`, or `orchestrated`.
2. **AI-native engineering is first class.** Agentic commands consume the same
   versioned `ProjectContext` and `TaskBrief` as deterministic commands and call
   deterministic build/test/quality operations only through the toolchain
   (gates declared in the context, executed by the host).
3. **Deterministic execution stays foundational.** Deterministic commands
   never require an engine, model credentials, or network access to an AI
   provider. This is enforced by the catalog invariant
   (`requires_engine` ⇔ `agentic`) and by tests that run discovery,
   verification, and reporting with no engine at all.
4. **One public contract in the open core.** `pkg/toolchain/contract` defines
   versioned Go types and generated JSON Schemas (`toolchain.krewire.dev/v1`,
   contract SemVer `1.0.0`) for ProjectContext, TaskBrief, Plan, Policy,
   ApprovalRecord, ExecutionRecord, EvidenceReport, and CommandCatalog, plus
   a stable numeric error model.
5. **One context implementation.** `pkg/toolchain/projctx` derives the context
   from source-of-truth files (go.mod, version.go, krewire.yaml, AGENTS.md,
   Go sources, executable guard tests, ADRs) with per-fact provenance,
   deterministic output, secret redaction, and stable `KWC` lint codes. Krew
   consumes it instead of maintaining its own loader.
6. **Engines reason; the toolchain acts.** The engine boundary is the adapter
   protocol (`pkg/toolchain/adapter`): JSON-RPC 2.0 over stdio with methods
   `initialize`, `plan`, `implement`, `review`, `shutdown`, and
   `$/cancelRequest`, per-request `meta.timeout_ms`, and contract error codes.
   Engines only *propose* tool calls. The toolchain authorizes each call
   (`pkg/toolchain/policy`: scope, prohibited paths, destructive-command
   guard, network policy, HMAC approvals from `pkg/sec.ApprovalSignal`),
   executes it, runs verification gates, and writes the evidence report.
7. **One evidence model.** `EvidenceReport` keeps raw deterministic gate
   outcomes (`model.VerificationEvidence`, ledger-chained) apart from engine
   claims. A claim may cite a gate only if that gate passed; the host strips
   any other citation, and `verified` must equal
   `model.EvaluateTaskVerification` over the raw gates.
8. **Licensing boundary unchanged.** The Krew engine stays proprietary in
   `krewire/krew`. Integration requires no repository merge and no license
   change; third-party engines integrate the same way and prove it with the
   public conformance suite (`pkg/toolchain/conformance`) and the reference
   engine (`pkg/toolchain/fakeengine`).

## Consequences

- Positive: one coherent developer experience; agents and commands share
  verified project facts; governance and evidence live in one place;
  deterministic gates stay runnable offline; third parties can validate the
  contract without private code.
- Negative: Krew must replace its private context loader and route its tool
  execution through the host instead of executing mutations itself
  ([integration note](../tools/krew/krew-integration.md)).
- Risks: contract churn. Mitigated by SemVer negotiation on the major version,
  generated schemas pinned by drift tests, and append-only error/lint codes.

## Alternatives Considered

1. Keep `krew` (agent) and `kiw` (toolchain) as separate CLIs — rejected: two
   competing developer experiences (#131 §1).
2. Move the Krew engine into the open core — rejected: breaks the licensing
   boundary (#131 §5, #142 guardrails).
3. Let engines execute tools directly under their own policy — rejected:
   duplicates policy and evidence and lets agent summaries stand in for
   verification (#140 non-goals).
4. Library-only integration (engine linked into `krew`) — rejected as the
   *only* path: it couples release cadences and excludes non-Go engines; the
   in-process `adapter.Engine` interface remains available alongside stdio.

## References

- Spec: [AI-native toolchain contract](../tools/krew/ai-native-contract.md)
- Integration: [Krew engine integration note](../tools/krew/krew-integration.md)
- Code: `pkg/toolchain/{contract,projctx,policy,adapter,workflow,fakeengine,conformance}`
