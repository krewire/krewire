# Project Context Model & Source Precedence

## 1. Overview

In multi-repo or complex Go ecosystems, project knowledge is frequently fragmented across `krewire.yaml`, repository READMEs, architecture decisions, task trackers (GitHub Issues), and package docstrings. When sources conflict or become stale, developers waste time discovering past decisions, and AI agents risk inferring incorrect requirements.

The **Krewire Project Context Model** establishes an authoritative, versioned, and machine-discoverable Single Source of Truth (SSOT). Both human developers and autonomous AI agents (`krew`, `boost`, IDE agents) consume the exact same context via `kiw context`.

---

## 2. Source Precedence Hierarchy

When conflicting instructions or constraints exist across files, resolution follows a strict, deterministic precedence hierarchy (Precedence 0 through 5):

```text
Precedence 0: Toolchain & Standard Library (Go compiler, OS primitives, CGO_ENABLED=0)
   ↑
Precedence 1: Core Foundation & Kernel (model, lifecycle, version, LICENSE)
   ↑
Precedence 2: Workspace Root (Root AGENTS.md, go.work, root README.md)
   ↑
Precedence 3: Project Configuration (krewire.yaml, go.mod)
   ↑
Precedence 4: Architecture Decision Records (docs/adr/ADR-*.md)
   ↑
Precedence 5: Active Task Specification (Current GitHub Issue / task prompt)
```

### Precedence Rules
1. **Higher Precedence Overrides Lower**: A rule defined at Precedence 5 (Active Task) can refine an implementation detail, but cannot violate foundational invariants from Precedence 0–2 (e.g. Pure-Go invariant or Three Pillars).
2. **Invariant Immutability**: Rules tagged with `Category: "pillar"` are non-negotiable invariants and cannot be overridden by task-level instructions.
3. **Traceable Provenance**: Every resolved rule records its origin file, line number, source key, and precedence tier.

---

## 3. Context Schema (`model/context.go`)

The project context is represented canonically by `model.ProjectContext`:

- `Version`: Context schema version (e.g. `1.0.0`).
- `Name`: Project name resolved from `krewire.yaml` or directory name.
- `Workload`: Workload kind (`app`, `site`, `book`, `cli`, `worker`, `service`, `gateway`, `infra`).
- `Tier`: Licensing and commercial posture (`free`, `pro`, `team`, `enterprise`). Core layers are permanently 100% MIT Free Tier.
- `GoVersion`: Go compiler target from `go.mod`.
- `ModulePath`: Canonical Go module path.
- `Workspace`: Absolute path to workspace root.
- `Pillars`: Authoritative rules for Krewire's Three Pillars (3S):
  - `Secure (S1)`: Safety by default, stdlib-first, SafePath path traversal guards, zero committed secrets.
  - `Sustainable (S2)`: Near-zero cost engineering, 10–30 MB lean RAM footprint, single static binary embed, lean pure Go runtime.
  - `Scalable (S3)`: Additive progression, strict downward-only dependencies, zero peer coupling.
- `Rules`: Flattened list of all active operational and architectural rules with provenance.
- `QualityGates`: Deterministic command validations required before committing.
- `Diagnostics`: Detected context integrity issues.

---

## 4. Context Diagnostics & Integrity Checks

The `kiw context --lint` command evaluates context integrity using four severity categories:

- **`MISSING`**: Critical context files (e.g. `go.mod`, `krewire.yaml`, `AGENTS.md`) are absent.
- **`CONFLICT`**: Two or more rules share conflicting IDs, or module metadata disagrees between `go.mod` and `krewire.yaml`.
- **`STALE`**: Deprecated or retired platform references (such as decommissioned tools or old URLs) are detected in context files.
- **`MALFORMED`**: Configuration syntax errors preventing deterministic parsing.

---

## 5. CLI Usage

### Inspect Project Context
```bash
kiw context
```
Displays human-readable project metadata, pillar invariants, quality gates, and diagnostics.

### Structured JSON for AI Agents
```bash
kiw context --json
```
Outputs the complete `ProjectContext` JSON document on `stdout` for ingestion by AI agents or tool chains.

### Lint Context Integrity
```bash
kiw context --lint
```
Runs static diagnostics. Exits with code 0 if context is consistent, or exit code 1 if blocking issues (`MISSING` or `CONFLICT`) are found.

### Filter by Scope
```bash
kiw context --scope workspace
kiw context --scope module
```

---

## 6. Context Lifecycle: Create, Update, Review, Retire

1. **Create**:
   - Running `krew init` or `krew new` establishes the baseline `krewire.yaml` and `go.mod`.
   - Workspace rules are recorded in `AGENTS.md`.
2. **Update**:
   - Architecture decisions are recorded via new ADR documents (`docs/adr/`).
   - Project configuration is adjusted in `krewire.yaml`.
3. **Review**:
   - Continuous integration runs `kiw context --lint` as part of the release qualification pipeline.
   - Code reviews ensure PRs do not introduce conflicting rules or bypass quality gates.
4. **Retire**:
   - Deprecated rules or components are removed directly from configuration and documentation without retaining legacy dead code or obsolete keyword lists.
