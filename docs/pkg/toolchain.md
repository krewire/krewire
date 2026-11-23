# Toolchain Contract (`pkg/toolchain`)

Public, versioned contracts that make AI-native engineering a first-class part
of the unified `krew` toolchain ([ADR-T1C4A](../adr/ADR-T1C4A-unified-ai-native-toolchain-contract.md),
[spec](../tools/krew/ai-native-contract.md)).

| Package | Responsibility |
|---|---|
| `contract` | Go types, generated JSON Schemas, validation, error codes, and the command catalog. Depends only on stdlib and `pkg/model`. |
| `projctx` | Deterministic project-context derivation and `KWC` lint from source-of-truth files |
| `policy` | Tool-call authorization: scope, destructive guard, network, and HMAC approvals (`pkg/sec`) |
| `adapter` | Engine interface and JSON-RPC 2.0 stdio transport (`Serve`, `Client`, `Spawn`) |
| `workflow` | Lifecycle orchestrator and deterministic gate runner (`Run`, `Verify`, `ExecRunner`) |
| `fakeengine` | Deterministic reference engine without an LLM |
| `conformance` | Engine compatibility suite (`Run`, `RunStdio`, `Pipe`, `Fixture`) |

```go
pctx, _ := projctx.Generate(projctx.Options{Root: "."})
report, _ := workflow.Run(ctx, workflow.Config{
	Root: ".", Context: pctx, Task: brief, Policy: contract.DefaultPolicy(),
	Engine: engine, // nil: agentic stages report engine_unavailable; verification still runs
})
```

Regenerate the schemas after you change a type: `go test ./pkg/toolchain/contract -update`.
