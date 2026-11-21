---
title: ".kiw Rendering & Hydration Conformance Matrix"
description: "Canonical conformance matrix, automated test fixtures, and browser baseline for the .kiw language compiler and client island runtime."
date: "2026-11-21"
---

# `.kiw` Rendering & Hydration Conformance Matrix

This document defines the automated conformance baseline for the `.kiw` Single File Component (SFC) specification, the Go-native template desugaring engine, server-side HTML rendering (SSR/SSG), and client-side island progressive hydration.

The automated conformance suite lives in [`pkg/runtime/conformance`](file:///home/krewire/workspace/krewire/krewire/pkg/runtime/conformance) and runs continuously in CI quality gates.

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
| `KIW-PRG-001` | Progressive | Zero-JS Semantic Fallback | `TestConformance_Progressive_ZeroJSFallback` | **Supported** |
| `KIW-PRG-002` | Progressive | Fault-Tolerant Hydration | `TestConformance_Progressive_FaultTolerance` | **Supported** |
| `KIW-BRW-001` | Browser | Headless DOM & Marker Simulation | `TestConformance_Browser_HeadlessSimulation` | **Supported** |

---

## 2. Browser & Runtime Compatibility Baseline

All client-side hydration directives (`load`, `idle`, `visible`) target a modern browser baseline while ensuring non-JavaScript and legacy browsers receive 100% accessible server-rendered HTML.

| Platform | Minimum Version | Support Tier | Required Capabilities |
| :--- | :--- | :--- | :--- |
| **Chromium / Chrome** | 100+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, `requestIdleCallback`, CustomElements |
| **Mozilla Firefox** | 100+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, `requestIdleCallback`, CustomElements |
| **Apple Safari / WebKit**| 15.4+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, CustomElements (polyfilled `requestIdleCallback`) |
| **Microsoft Edge** | 100+ | **Tier 1 (Supported)** | ES6 Modules, `IntersectionObserver`, `requestIdleCallback`, CustomElements |
| **Mobile WebKit (iOS)** | 15.4+ | **Tier 2 (Progressive)**| ES6 Modules, `IntersectionObserver` (adapts idle schedules to power constraints) |
| **Legacy Engines (IE11, Opera Mini)** | < All | **Tier 3 (Unsupported)**| *Explicitly unsupported for client hydration.* Server-rendered HTML remains readable. |

---

## 3. Automated Test Execution

### Running Locally
To run all conformance tests:
```bash
go test -v ./pkg/runtime/conformance/...
```

To run with coverage profiling:
```bash
go test -coverprofile=coverage.out ./pkg/runtime/conformance/...
go tool cover -func=coverage.out
```

### Continuous Integration (CI)
The conformance suite runs automatically on every pull request, nightly build (`.github/workflows/nightly.yaml`), and release qualification pipeline (`krew qualify`).
