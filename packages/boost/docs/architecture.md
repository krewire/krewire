# Architecture — Krewire Guild

## Module Structure

```
guild/
├── guild.go              # `//go:embed all:template` + Install library (force/dry-run, conflict detection)
├── template/
│   ├── AGENTS.md         # Unified constitution (8 kinds, full command matrix, spec conventions)
│   ├── opencode.json     # Base opencode config
│   └── .agents/
│       ├── agents/       # 20 agents: build, plan, orchestrator, scout, tester, runtime, infra, service, worker, ...
│       ├── commands/     # 16 commands: kickoff, vision, runtime, infra, service, worker, new-project, ...
│       ├── skills/       # 26 skills: vision, wasm-runtime, infra-provision, service-mesh, worker-queue, ...
└── docs/
```

**Design decisions:**

- **Embed.FS + Install.** Template is shipped as `guild.Template` embed.FS; `guild.Install(target)` copies managed paths, refusing to overwrite unless `WithForce`.
- **Stdlib-only module.** No third-party imports; preserves `KWG-*` spec requirements.
- **Managed paths.** `AGENTS.md`, `opencode.json`, `.agents/agents|commands|skills|context`, `.agents/README.md` — installed via `kiw guild install`.


## Conventions

- Documentation in English, Markdown, spec-driven (`docs/specs/`).
- Quality gates: `gofmt -l .`, `go vet ./...`, `go test ./...` in each Go repo; per-kind `kiw build` / `kiw build --plan` spot-checks.
- Cross-repo testing via the hub `go.work` workspace; temporary `replace` directives only for single-repo clones outside the workspace.
