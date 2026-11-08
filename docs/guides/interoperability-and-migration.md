# Interoperability & Incremental Adoption Guide

This document defines Krewire's technical boundary, coexistence guarantees, and incremental adoption path alongside external development stacks and mixed repositories.

---

## 1. Architectural Philosophy: Coexistence Without Rewrite

Krewire is designed for incremental adoption. Small teams and independent developers are never forced into an all-or-nothing framework migration or rewrite. Instead, Krewire coexists transparently with existing host stacks, such as Node.js fullstack applications, Laravel PHP backends, or static repositories.

```text
Host Project (Node.js / Laravel / Python)
├── package.json / composer.json (Untouched)
├── src/ or app/ (Untouched)
└── [Co-located Workload]
    ├── api/ (Krewire Service)
    │   ├── go.mod
    │   └── krewire.yaml
    └── worker/ (Krewire Queue Processor)
        ├── go.mod
        └── krewire.yaml
```

---

## 2. Detection Capabilities Matrix

`kiw info` automatically detects external runtime environments and co-located workloads:

| Runtime / Host | Manifest Detected | Package Managers Identified | Frameworks Detected |
| :--- | :--- | :--- | :--- |
| **Node.js** | `package.json` | `npm`, `pnpm`, `yarn`, `bun` | SvelteKit, Nuxt, Next.js, Remix, Astro, Fastify, Express |
| **Laravel** | `composer.json`, `artisan` | `composer` | Laravel |
| **PHP** | `composer.json` | `composer` | Generic Composer PHP |

### Explicit Limitations & Non-Detection
- **No Automatic Code Transpilation**: Krewire does not transpile JavaScript or PHP into Go.
- **No Host File Mutation**: `kiw` commands will never modify, overwrite, or delete `package.json`, `package-lock.json`, `composer.json`, or host dependency directories (`node_modules/`, `vendor/`).
- **Detection Only**: Detection provides configuration awareness, port conflict prevention, and build suggestions (`build.depends_on`). It does not inject unrequested dependencies.

---

## 3. Port Conflict Detection & Prevention

When host dev servers (e.g. Vite on `:5173`, Next.js on `:3000`) and Krewire servers (`:8080`) run in the same project directory:
- `kiw info` and `kiw dev` inspect declared ports in both host manifests and `krewire.yaml`.
- If conflicting ports are found, `kiw` emits warning `FRK-CONF-030` with suggestions to remap port bindings via `server.port` or CLI `--addr`.

---

## 4. Cross-Runtime Inter-Process Communication (IPC)

For zero-network, high-performance inter-process communication:
- **Standard Input/Output JSON Streaming**: Launch a compiled Go worker or service as a sub-process via `kiw run <workload>` or direct binary execution.
- **Distributed Tracing**: Krewire automatically extracts and forwards standard W3C `traceparent` headers (`00-<trace_id>-<span_id>-<flags>`), ensuring end-to-end tracing across language boundaries.
- **Structured Log Tagging**: Emits standardized `source="go"` and `[go]` tags for unified log aggregation alongside host logs.

---

## 5. Verification & Automated Coverage

All coexistence capabilities are backed by automated tests:
- Node.js & SvelteKit detection: `TestDetectHost_NodeJS`, `TestRunInfo_NodeJSEcosystemAndColocated`
- Laravel detection: `TestDetectHost_Laravel`, `TestRunInfo_LaravelEcosystemAndColocated`
- Co-located directory discovery: `TestFindColocated`
- Multi-workload integration: `TestWorkflow_HybridApplication`
