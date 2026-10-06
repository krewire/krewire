# Architecture — Krewire Devtool

## Module Structure

```
kiw/
├── cmd/kiw/               # Entry — built via `go build -o kiw ./cmd/kiw`, dogfoods framework/tui
├── internal/
│   ├── commands/         # new/init, build/serve (site+book), run/dev (app+cli), deploy (stages dist/), test/vet/fmt, info/version, guild, ws (planned: worker, dashboard, generate)
│   ├── scaffold/         # `kiw new` / `kiw init` templates for 8 kinds
│   ├── shape/            # Kind detection (krewire.yaml + manuscript/ + infra/ + main.go)
│   ├── config/           # ssg: / book: config handling
│   ├── buildinfo/        # version embedding
│   ├── gomod/            # go.mod helpers
│   └── version/          # version reporting
└── docs/
```

**Design decisions:**

- **Kind dispatch.** `kiw info` prints detected kind; `kiw build` picks pipeline (binary / `.krewire/build` / `manuscript/`→`.krewire/build` / infra plan) based on `project.kind` or markers.
- **Dogfooding.** `cmd/kiw` is built on `packages/tui` with `packages/kern` exit codes and terminal output.
- **No per-project `cmd/`.** All CLI behavior lives in the `kiw` CLI; projects have no custom `cmd/` binaries for build/serve/run.
- **Binary name.** The CLI binary is `kiw`.

## Conventions

- Documentation in English, Markdown, spec-driven (`docs/specs/`).
- Quality gates: `gofmt -l .`, `go vet ./...`, `go test ./...` in each Go repo; per-kind `kiw build` / `kiw build --plan` spot-checks.
- Cross-repo testing via `go.work` workspace (`./krewire`, `./mdbind`) at workspace root; `go work sync` updates `go.work.sum`.
