// Package krewire is the foundational Go monorepo providing domain libraries,
// kernel lifecycle models, error contracts, and the autonomous AI engine for
// the Krewire digital SDLC ecosystem.
//
// # Architectural Layers
//
// Following the 6-Layer Pyramid Architecture (ADR-E8P9D) and Three Pillars (ADR-3S001),
// packages are layered with strictly downward-only dependencies:
//
//   - Layer 1: Core Foundation & Kernel
//
//   - pkg/model: Declarative domain models, architectural scopes, licensing tiers, project invariants.
//
//   - pkg/errs: POSIX-compatible exit codes, structured error wrapping, diagnostics.
//
//   - pkg/version: Semantic versioning, build watermarking, compatibility checks.
//
//   - pkg/env: Target environment resolution (local, production, testing).
//
//   - pkg/lifecycle: Process supervision, lifecycle hooks, and graceful drain.
//
//   - Layer 2: Modular Domain Libraries
//
//   - pkg/web: HTTP routing, middleware, SSR, SSG, edge gateway, HMR.
//
//   - pkg/orm: Lightweight SQL query builder, migrations, and database drivers.
//
//   - pkg/cache: High-performance caching abstractions (in-memory, redis).
//
//   - pkg/sec: Secure defaults, SafePath traversal protection, CSP/CSRF guards.
//
//   - pkg/storage: Unified storage provider contracts (local disk, cloud bucket).
//
//   - pkg/ai: Autonomous ReAct agent loop, persona definitions, embed API.
//
//   - pkg/validation: Declarative schema and struct validation rules.
//
//   - pkg/resilience: Circuit breakers, rate limiters, retry policies.
//
//   - pkg/i18n: Localization dictionaries and translation helpers.
//
// All packages enforce a pure-Go invariant (CGO_ENABLED=0) and stdlib-first design.
package krewire
