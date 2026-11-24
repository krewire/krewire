# `.kiw` Rendering and Hydration Conformance Suite

Package `runtime/conformance` automates conformance testing for the supported `.kiw` language specification and runtime behaviors across the parser, compiler, server-side renderer (SSR/SSG), and browser client-island hydration engine.

---

## 1. Conformance Matrix

| Feature ID | Category | Feature Name | Automated Test Target | Support Tier |
| :--- | :--- | :--- | :--- | :--- |
| `KIW-GRM-001` | Grammar | YAML Frontmatter Extraction | `TestConformance_Grammar_Frontmatter` | **Supported** |
| `KIW-GRM-002` | Grammar | Type-Safe Props & Defaults | `TestConformance_Grammar_PropsAndDefaults` | **Supported** |
| `KIW-GRM-003` | Grammar | Slot Outlets & Named Slots | `TestConformance_Grammar_Slots` | **Supported** |
| `KIW-GRM-004` | Grammar | Conditionals (`{#if ...}`) | `TestConformance_Grammar_Conditionals` | **Supported** |
| `KIW-GRM-005` | Grammar | Loops (`{#each ...}`) | `TestConformance_Grammar_Loops` | **Supported** |
| `KIW-GRM-006` | Grammar | Scoped Styles & Class Directives | `TestConformance_Grammar_StylesAndClasses` | **Supported** |
| `KIW-GRM-007` | Grammar | Compound & Namespaced Components | `TestConformance_Grammar_CompoundAndNamespaced` | **Supported** |
| `KIW-DIA-001` | Diagnostics | 1-Based Line & Column Accuracy | `TestConformance_Diagnostics_LineColAccurate` | **Supported** |
| `KIW-DIA-002` | Diagnostics | Malformed Block Recovery | `TestConformance_Diagnostics_MalformedBlocks` | **Supported** |
| `KIW-SSR-001` | SSR/SSG | Contextual Auto-Escaping (XSS Defense) | `TestConformance_SSR_ContextualEscaping` | **Supported** |
| `KIW-SSR-002` | SSR/SSG | Deterministic Idempotent Rendering | `TestConformance_SSR_DeterministicRendering` | **Supported** |
| `KIW-HYD-001` | Hydration | Directives (`load`, `idle`, `visible`) | `TestConformance_Hydration_Directives` | **Supported** |
| `KIW-HYD-002` | Hydration | Multi-Framework Markers (`svelte`, `vue`, `wasm`) | `TestConformance_Hydration_FrameworkMarkers` | **Supported** |
| `KIW-HYD-003` | Hydration | Loader Injection (`kiw-islands.js`) | `TestConformance_Hydration_LoaderInjection` | **Supported** |
| `KIW-PRG-001` | Progressive | Semantic HTML Fallback | `TestConformance_Progressive_SemanticFallback` | **Supported** |
| `KIW-PRG-002` | Progressive | Fault-Tolerant Hydration | `TestConformance_Progressive_FaultTolerance` | **Supported** |
| `KIW-BRW-001` | Browser | Headless DOM & Marker Simulation | `TestConformance_Browser_HeadlessSimulation` | **Supported** |

---

## 2. Browser & Runtime Compatibility Baseline

| Platform | Minimum Version | Support Tier | Required Capabilities |
| :--- | :--- | :--- | :--- |
| **Chromium / Chrome** | 100+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, `requestIdleCallback`, CustomElements |
| **Mozilla Firefox** | 100+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, `requestIdleCallback`, CustomElements |
| **Apple Safari / WebKit**| 15.4+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, CustomElements (polyfilled `requestIdleCallback`) |
| **Microsoft Edge** | 100+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, `requestIdleCallback`, CustomElements |
| **Mobile WebKit (iOS)** | 15.4+ | **Tier 2 (Progressive)**| ES6 Modules, `IntersectionObserver` (adapts idle schedules to power constraints) |
| **Legacy Engines (IE11, Opera Mini)** | < All | **Tier 3 (Unsupported)**| *Explicitly unsupported for client hydration.* Server-rendered HTML remains readable. |

---

## 3. Local Execution Commands

Run the full conformance suite locally:
```bash
go test -v ./modules/runtime/conformance/...
```

Run with benchmark profiling:
```bash
go test -v -bench=. -benchmem ./modules/runtime/conformance/...
```

Run specific category tests:
```bash
go test -v -run TestConformance_SSR ./modules/runtime/conformance/...
go test -v -run TestConformance_Hydration ./modules/runtime/conformance/...
```
