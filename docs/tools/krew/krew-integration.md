# Krew Engine Integration Note (contract v1)

How the `krewire/krew` repository adopts the
[AI-native toolchain contract](./ai-native-contract.md)
([ADR-T1C4A](../../adr/ADR-T1C4A-unified-ai-native-toolchain-contract.md)).
The open core ships the contract, the context generator, the policy, the
adapter transport, the orchestrator, the reference engine, and the conformance
suite. Krew changes are listed here because the core does not modify the
proprietary repository. Nothing below moves Krew source into the core or
changes its license.

## 1. Dependency

- Require the `github.com/krewire/krewire` version that contains
  `toolchain/...` and drop the relative `replace … => ../krewire` before a
  release (#129, ADR-V8M2C).
- Import only `toolchain/{contract,projctx,policy,adapter,workflow}` and,
  in tests, `toolchain/conformance`. Foundational modules never import `krew`
  (`TestPackagesNeverImportTooling`), so the dependency stays one-way.

## 2. One context implementation

Current state: `internal/context/loader.go` and `linter.go` build a
`model.ProjectContext` from hard-coded rules and emit diagnostics without
codes.

Required:

1. `krew context` calls
   `projctx.Generate(projctx.Options{Root: root, ToolchainVersion: buildinfo.Version})`.
2. `--json` encodes the returned `contract.ProjectContext` unchanged
   (two-space indent, trailing newline).
3. `--lint` prints each diagnostic as `CODE severity path:line: message (hint)`
   and exits `1` when `ctx.Blocking()` is true, `0` otherwise.
4. Delete `internal/context` once no caller remains. `krew chat`, `krew agent`,
   `krew verify`, `krew compat`, and `krew release` obtain context from
   `projctx.Generate` only.

## 3. Command surface

Implement the catalog in `contract.Catalog()` in `internal/cli/app.go`:

| Command | Implementation |
|---|---|
| `krew version` | also print `contract.ContractVersion` |
| `krew contract schema <Kind>` | write `contract.Schema(kind)` |
| `krew contract validate <file>` | `contract.Decode(data)`; exit `1` on error and print the error code name |
| `krew task validate <file>` | `Decode` and then `brief.Validate(ctx)` |
| `krew engine status` | resolve the engine (§4), call `initialize`, print identity, `ready`, and `reason` |
| `krew verify [task.json]` | `workflow.Verify(ctx, root, pctx, gates, workflow.ExecRunner{Commit: head}, ledger, head)`; with no brief, use every required gate in the context |
| `krew plan <task.json>` | `engine.Plan`, then `plan.Validate(task)` and `plan.ComputeDigest()`; write the `Plan` |
| `krew agent <task.json> --plan <plan.json>` | `workflow.Run` with the stored plan approval; keep the free-form prompt mode for `krew agent "<prompt>"` |
| `krew review`, `krew report` | engine `Review`; render or validate an `EvidenceReport` |
| `krew flow <task.json>` | `workflow.Run(ctx, workflow.Config{…})` and write the `EvidenceReport` |

Approvals: interactive sessions implement `workflow.Approver` by prompting
on the TTY. Dash or a CI approver signs with `policy.Sign` using the shared
key from `KREW_APPROVAL_KEY`. The key is never written to any document.

## 4. Engine boundary

1. In `internal/agent`, wrap the existing ReAct engine (built on `ai`) in
   a type that implements `adapter.Engine`:
   - `Initialize`: report `ready: false` and a `reason` when no provider
     credentials are configured. Never fail deterministic commands.
   - `Plan`: prompt with the JSON `ProjectContext` and `TaskBrief`, then
     return a `contract.Plan`.
   - `Implement`: the engine may read the workspace directly. Mutating tools
     (`write_file`, `edit_file`, `apply_patch`, `exec`) must not run in
     the engine. Translate each one into a `contract.ToolCall` (`edit_file` and
     `apply_patch` become `write_file` with the full new content). Return the
     calls with `more: true` to continue the ReAct loop after the host executes
     them, and read the results from `previous`.
   - `Review`: return findings and claims. Set `verified_by` only from passed
     gates in `verification`.
2. Add a hidden `krew engine serve --stdio` command:
   `adapter.Serve(ctx, eng, os.Stdin, os.Stdout)`, logging to stderr.
3. Resolve the engine in this order: `KREW_ENGINE` (spawned with
   `adapter.Spawn`, for third-party engines) → the built-in Krew engine
   in-process → `adapter.Unavailable`.
4. `krew mcp` and `krew daemon` keep their protocols. Route their mutating
   tools through `policy.Authorizer` (follow-up).

## 5. Tests in `krewire/krew`

```go
func TestKrewEngineConforms(t *testing.T) {
	conformance.Run(t, func(t *testing.T) adapter.Engine { return newEngineWithMockProvider(t) })
	conformance.RunStdio(t, func(t *testing.T) adapter.Engine { return newEngineWithMockProvider(t) })
}
```

These use the offline mock provider (ADR-A8I4N §4), so CI needs no
credentials. Also add a CLI test showing that `krew context --lint`,
`krew verify`, and `krew test` pass with every provider variable unset.

## 6. Core follow-ups

- `ai/sdlc` keeps its own `TaskBrief` and workflow. During migration it
  consumes `contract.TaskBrief.ToModel()`; retire it once Krew runs on
  `toolchain/workflow`.
- `cmd/ai` is a development harness for `ai`, not a user-facing
  executable. Users run `krew chat` and `krew agent`. Remove it or mark it
  internal under #132.
- `KWC3001` lists the docs that still invoke `kiw`. Fix them under #132.
- Shell completion, `self-update`, and release artifacts identify `krew` only
  (#134).
