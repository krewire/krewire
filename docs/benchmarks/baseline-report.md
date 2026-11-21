---
title: "Performance Benchmark Baseline Report"
description: "Canonical performance benchmarks, measured baselines, and regression policies for Krewire Core, CLI, and .kiw runtime."
date: "2026-11-21"
---

# Performance Benchmark Baseline Report

This document records the official, reproducible performance baseline for Krewire Core primitives, developer CLI commands (`krew`), and the `.kiw` component runtime.

---

## 1. Benchmark Inventory & Rationale

| Operation ID | Category | Operation Name | Fixture Description | Target Budget | Rationale |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `CORE-CFG-001` | Core | Project Config Loading | `krewire.yaml` (standard site) | < 500 µs, < 50 KB | Dispatched on every CLI command to determine root config & active plugins. |
| `CORE-MOD-001` | Core | Model & Kind Resolution | Multi-workload dependency graph | < 1 ms, < 100 KB | Validates project graph topology, downward rules, and battery opt-ins. |
| `CORE-KIW-001` | Core | `.kiw` Component Parsing | Landing SFC (frontmatter, styles, script) | < 350 µs, < 40 KB | High-frequency leaf concern executed on every build, dev edit, and test. |
| `CORE-DSG-001` | Core | Template Desugaring | Nested loops, conditionals, slots & props | < 200 µs, < 30 KB | Converts authoring DSL into pure Go `html/template` trees with 0 npm. |
| `CORE-MDN-001` | Core | Markdown Rendering | 10 KB technical document with code blocks | < 800 µs, < 150 KB | Powers documentation portals and books across thousands of markdown pages. |
| `CLI-LAT-001` | CLI | `krew version` Cold-Start | Bare process spawn from disk | < 30 ms | Measures bare process spawn and Go runtime startup overhead. |
| `CLI-LAT-002` | CLI | `krew help` Discovery | Top-level command list overview | < 35 ms | Measures command registration, flag parsing, and help formatting latency. |
| `CLI-LAT-003` | CLI | `krew tier` Pre-Commit Check | `krew tier --check free` | < 60 ms | Ensures pre-commit checks and quality gates remain instantaneous. |
| `WFK-SSG-001` | Workflow | SSG Multi-Page Site Build | 5-page site with layouts & components | < 150 ms, < 10 MB | Production build throughput for continuous deployment and static sites. |

---

## 2. Canonical Measured Baseline (v0.2.0)

**Environment:** Linux amd64 (AMD EPYC 9754, 2 vCPUs) · Go 1.27.3  
**JSON Baseline Artifact:** [`docs/benchmarks/baseline-v0.2.0.json`](file:///home/krewire/workspace/krewire/krewire/docs/benchmarks/baseline-v0.2.0.json)

| ID | Operation | Category | Measured Latency | Memory / Op | Allocs / Op | Target Budget | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `CORE-CFG-001` | Project Configuration Loading | Core | **31.04 µs** | 10.2 KB | 82 allocs | < 500 µs | ✅ PASS |
| `CORE-MOD-001` | Model & Kind Dependency Resolution | Core | **1.47 µs** | 48 B | 1 alloc | < 1.00 ms | ✅ PASS |
| `CORE-KIW-001` | Single-File Component Parsing (.kiw) | Core | **185.49 µs** | 23.4 KB | 290 allocs | < 350 µs | ✅ PASS |
| `CORE-DSG-001` | Template Desugaring & Clean Interpolation | Core | **24.84 µs** | 11.8 KB | 120 allocs | < 200 µs | ✅ PASS |
| `CORE-MDN-001` | Documentation Markdown Rendering | Core | **33.83 µs** | 4.3 KB | 40 allocs | < 800 µs | ✅ PASS |
| `CLI-LAT-001` | `krew version` Cold-Start Latency | CLI | **5.64 ms** | N/A | N/A | < 30 ms | ✅ PASS |
| `CLI-LAT-002` | `krew help` Command Discovery Latency | CLI | **5.70 ms** | N/A | N/A | < 35 ms | ✅ PASS |
| `CLI-LAT-003` | `krew tier` Pre-Commit Check Latency | CLI | **5.30 ms** | N/A | N/A | < 60 ms | ✅ PASS |
| `WFK-SSG-001` | SSG Multi-Page Site Compilation | Workflow | **926.87 µs** | 322 KB | 4,761 allocs | < 150 ms | ✅ PASS |

---

## 3. Performance Regression Policy

To prevent developer friction from accumulating unnoticed, all future releases and commits are audited against this baseline:

1. **Noise Floor**: Microsecond measurements below 50 µs are subject to a noise floor to avoid false alerts caused by OS context switches.
2. **Warning Threshold**:
   - Latency increase $\ge +15.0\%$
   - Memory allocation increase $\ge +20.0\%$
   - *Action:* Logged as warning in benchmark reports and PR qualification.
3. **Critical Regression Threshold**:
   - Latency increase $\ge +30.0\%$
   - Memory allocation increase $\ge +40.0\%$
   - *Action:* Blocks release qualification (`krew release --qualify`) and requires profiling analysis (`pprof`) before merging.

---

## 4. Execution & Verification Commands

### Running via `krew` CLI
```bash
# Run all benchmarks
krew bench

# Run specific categories
krew bench --core
krew bench --cli
krew bench --workflow

# Output machine-readable JSON
krew bench --json

# Compare against canonical baseline to detect regressions
krew bench --compare docs/benchmarks/baseline-v0.2.0.json
```

### Running via Go Toolchain
```bash
go test -v -bench=. -benchmem ./krew/internal/bench/...
```
