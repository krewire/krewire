---
title: "Getting Started"
description: "Get up and running with Krewire: installation, configuration, and directory structure."
date: "2026-09-29"
---

# Getting Started

Welcome to the **Krewire Getting Started Guide**.

This chapter covers everything you need to go from a clean machine to developing, configuring, and organizing a production-ready Krewire project across any of the eight supported workloads.

---

## What is in this Chapter?

This chapter is organized into three comprehensive sections:

<div class="overview-grid" style="display:grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1rem; margin: 1.5rem 0;">

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/getting-started/installation" style="color:var(--primary); text-decoration:none;">2.1 Installation →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Install the <code>kiw</code> CLI binary on Linux, macOS, or Windows via the automated shell installer or <code>go install</code>.
    </p>
  </div>

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/getting-started/configuration" style="color:var(--primary); text-decoration:none;">2.2 Configuration →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Master <code>krewire.yaml</code>: project metadata, workload kinds, dev server settings, build targets, and task runner scripts.
    </p>
  </div>

  <div style="border: var(--pop-border); border-radius: 10px; padding: 1.25rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <h3 style="margin-top:0; font-size:1.1rem; font-weight:800;">
      <a href="/getting-started/directory-structure" style="color:var(--primary); text-decoration:none;">2.3 Directory Structure →</a>
    </h3>
    <p style="margin-bottom:0; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Understand the canonical Krewire directory conventions across static sites, documentation books, fullstack monoliths, and microservices.
    </p>
  </div>

</div>

---

## 30-Second Fast Track

If you are already familiar with Go development, you can bootstrap a new project immediately:

```bash
# 1. Install kiw CLI
curl -fsSL https://krewire.com/scripts/install.sh | sh

# 2. Scaffold a static site project
kiw new my-site --site

# 3. Enter project and launch hot-reloading dev server
cd my-site
kiw dev
```

Your application is now live at `http://localhost:8080`.

---

## Next Steps

To begin the in-depth walkthrough, proceed to [**2.1 Installation →**](/getting-started/installation).
