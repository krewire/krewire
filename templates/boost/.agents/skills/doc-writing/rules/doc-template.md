# Documentation Template — Global Standard (Krewire Adaptation)

> Choose the section for the doc you write. Keep docs in English per `AGENTS.md`. Human may write Indonesian but agent replies English.

---

## 1. Common Frontmatter (all docs)

```markdown
# Title — One-line, outcome-focused
> One-paragraph summary: what this doc is, who it is for, and where it sits in the hierarchy (vision → specs → architecture → README → manuscript).
```

---

## 2. README Template

```markdown
# <Repo> — One-line purpose

One-paragraph overview: what this repo is in the unified 9-workload matrix.

## Workload Matrix (if repo is workload-facing)

| Workload | Kind | Package | Status |
|----------|------|---------|--------|
| ... | `cli` | `framework/tui` | Shipped |
...

## Ecosystem Layout (if workspace root)

| Repo | Module | Role |
|------|--------|------|
...

## Quick Start

```bash
go build ./...
go test ./...
```

## Architecture

Link to `docs/architecture.md` for module tree.

## Philosophy

Link to `docs/philosophy.md` for principles.

## Specifications

Link to `docs/specs/index.md` (central) — note `docs/specs/` is `MOVED.md` redirect.

## Related Repositories

- ...
```

**Checklist:** Workload matrix matches `project-vision.md` + `core.Workloads` (9 rows, `framework/tui` not `cli`).

---

## 3. Architecture Template (`docs/architecture.md`)

```markdown
# Architecture — <Repo>

## Module Structure (Optimal & Efficient)

Aligned to unified vision `KWF-M8K2Q`:

```
repo/
├── pkg/                  # One package per concern, flat, opt-in
├── docs/
│   ├── architecture.md   # This file
│   ├── philosophy.md
│   └── index.md
└── go.mod
```

## Design Decisions

- **Flat, opt-in packages.** Import `framework/tui` only when CLI needed; `app` alone imports `web`/`ui`/`app`.
- **Control plane.** `libs/core` (declarative) + `libs/kern` (imperative); `framework`/`krewire` compose via `kern`.
- **Stdlib-first.** `net/http`, `flag`, `log/slog` before third-party.

## Dependency Graph

```
framework → libs (core, kern, term, config, validate)
docs (book) → mdbind → framework → libs
```

## Conventions

- English, Markdown, spec-driven (`docs/specs/` in `krewire/internal`).
- Quality gates: `gofmt -l .`, `go vet ./...`, `go test ./...` + per-kind `krewire build` spot-checks.
```

---

## 4. Philosophy Template (`docs/philosophy.md`)

```markdown
# Philosophy — <Repo>

## Philosophy

**One-line principle.** E.g., "One framework, every workload — opt-in batteries, zero-cost monolith."

**Principles:**

- **Opt-in batteries.** Monolith imports `app`; `service`/`infra` only for `service`/`infra` kinds.
- **Stdlib-first.** Prefer stdlib over wrappers.
- **Single config, single CLI.** `krewire.yaml` only; `krewire` drives all 8 kinds.
- **Spec-driven.** Every feature has a `KWF-*` spec with `FRK-*` rows before code.

## Contribution

- Read `docs/project-vision.md` and `docs/specs/index.md` before changing behavior.
- Keep suite green; update `README.md`/`docs/` when public behavior changes.
```

---

## 5. Index Template (`docs/index.md`)

```markdown
# <Repo> — Documentation

`<Repo>` (`module`) — `Role`.

## Contents

- [Architecture](./architecture.md) — module structure, design decisions, dependency graph
- [Philosophy](./philosophy.md) — principles and contribution guidance
- [Specifications](./specs/index.md) — formal specs ( in `krewire/internal` at `docs/specs/<repo>/`; local `specs/` are redirects — see `specs/MOVED.md`)

## Getting Started

- Read `README.md` for build/test instructions.
- For unified matrix and roadmap, see `docs/project-vision.md` (source spec `KWF-ARCH-M8K2Q`).

## Conventions

- English, Markdown, spec-driven; file name `{ProjectId}-{Scope}-{SpecID}-{slug}.md`, random 5-char SpecID.
```

---

## 6. Manuscript Template (`docs/manuscript/*.md`)

```markdown
# Chapter Title

One-paragraph hook: what the reader will build.

## What You Will Build

- ...

## Prerequisites

- Go 1.22+

## Steps

1. **Step — imperative title**
   ```bash
   command
   ```
   Explanation. Cite file:line if referencing code.

## Verification

```bash
go test ./...
```

## Next Steps

- Link to next manuscript chapter or `docs/project-vision.md` workload.
```

**Diátaxis:** Tutorials (learning-oriented), How-to guides (task-oriented), Reference (information-oriented), Explanation (understanding-oriented) — choose one per manuscript file.

---

### Global Standard Notes

- **IEEE 830 / 1016:** Structure, traceability, and verification.
- **Diátaxis:** Documentation system (tutorials, how-to, reference, explanation).
- **Krewire specifics:** Workload matrix 9 rows, `framework/tui` not `framework/cli`, `tui.NewApp`,  specs.

### Checklist Before Submitting Docs

- [ ] Architecture tree matches `docs/architecture.md` module structure and `docs/specs/index.md` Impl Status
- [ ] Philosophy mentions `core` declarative + `kern` imperative control plane
- [ ] `sync-docs` reports `In-sync`
```

