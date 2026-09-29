---
title: "Overview"
description: "Welcome to Krewire — One Go Framework for Every Workload. Learn the core architecture, philosophy, workloads, and developer workflows."
date: "2026-09-29"
---

# Overview

Welcome to the **Krewire Documentation**.

**Krewire** is an open-source, unified Go framework designed to eliminate toolchain fatigue across modern software engineering. It provides a single language (**Go**), a single command-line interface (**`kiw`**), and a unified devtool configuration (**`krewire.yaml`**) to build, test, and deploy **eight distinct workloads**:

1. **Fullstack Web Monoliths (`app`)**
2. **Terminal Interfaces & TUIs (`cli`)**
3. **High-Speed Static Sites (`site`)**
4. **Structured Documentation Books (`book`)**
5. **Background Workers & Job Queues (`worker`)**
6. **Distributed Microservices (`service`)**
7. **Cloud Infrastructure as Code (`infra`)**
8. **Client-Side WebAssembly Runtimes (`runtime`)**

---

## What is in this Chapter?

This **Overview** chapter introduces you to the core philosophy, technical architecture, and developer workflow of Krewire:

<div class="overview-grid" style="display:grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1rem; margin: 1.5rem 0;">

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/docs/krewire-framework" style="color:var(--primary); text-decoration:none;">1.1 Krewire Framework →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Explore the modular monolith architecture, the package ecosystem (<code>framework</code>, <code>libs</code>, <code>kiw</code>, <code>mdbind</code>), and the core web engine.
    </p>
  </div>

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/docs/why-krewire" style="color:var(--primary); text-decoration:none;">1.2 Why Krewire? →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Understand how Krewire solves modern toolchain fragmentation, replaces multi-language stacks, and enables progressive system growth without rewrites.
    </p>
  </div>

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/docs/krewire-workloads" style="color:var(--primary); text-decoration:none;">1.3 Krewire Workloads →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Deep dive into the 8 official workload kinds (<code>app</code>, <code>cli</code>, <code>site</code>, <code>book</code>, <code>worker</code>, <code>service</code>, <code>infra</code>, <code>runtime</code>).
    </p>
  </div>

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/docs/upgrade-guide" style="color:var(--primary); text-decoration:none;">1.4 Upgrade Guide →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Learn the SemVer zero-breakage promise, migration steps for <code>krewire.yaml</code> decoupling, and how to update CLI & framework dependencies.
    </p>
  </div>

</div>

---

## Core Tenets

Krewire is engineered around four guiding pillars:

1. **Go as the Architectural Foundation**  
   Standard Go (`net/http`, `html/template`, `embed`, `context`, `slog`, `flag`) powers every layer. Single-binary deployments with embedded static assets ensure near-instant startups, negligible RAM usage, and zero runtime dependencies.
2. **Modular at Every Scope (SRP & SoC)**  
   Engineered using strict Single Responsibility and High Cohesion principles (`KWF-5ZHQV`). Start with a simple modular monolith and cleanly extract services or background workers as your traffic scales — without restructuring your business logic.
3. **Zero JavaScript Fatigue**  
   Generate lightning-fast static sites, documentation portals, and fullstack applications with `.kiw` component files without requiring Node.js, `npm`, webpack, or fragile `node_modules` dependency trees.
4. **Unified Developer Experience**  
   Manage your entire software lifecycle — creation, local development, asset compilation, testing, and deployment — through **one single CLI binary (`kiw`)** and a single devtool configuration file (**`krewire.yaml`**).

---

## Next Steps

To begin building with Krewire, explore the subchapters in order:

- Proceed to [**1.1 Krewire Framework →**](/docs/krewire-framework)
- Or jump directly to [**2. Getting Started →**](/getting-started)
