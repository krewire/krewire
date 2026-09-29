---
title: "Getting Started"
description: "Comprehensive onboarding guide to Krewire: prerequisites, system verification, 5-minute hands-on quickstart, development lifecycle, workload archetype guide, and chapter roadmap."
date: "2026-09-29"
---

# Getting Started

Welcome to the **Krewire Getting Started Guide**.

This chapter is your complete onboarding manual for the Krewire ecosystem. It walks you through setting up your environment, understanding the developer toolchain, scaffolding and running your first project, and organizing your codebase according to battle-tested Go architectural standards.

Whether you are authoring a lightweight static documentation portal, developing an interactive terminal utility, building a fullstack web monolith, or operating high-throughput microservices, this guide equips you with the foundational workflows to build and ship software with zero toolchain fatigue.

---

## 1. Prerequisites & System Verification

Before building with Krewire, ensure your local development machine meets the baseline requirements. Krewire requires **no Node.js, no npm, no bundlers, and no external runtimes**—only standard Go and a Unix-compatible shell or Windows terminal.

### System Requirements Matrix

| Requirement | Supported Specifications | Verification Command | Notes |
| :--- | :--- | :--- | :--- |
| **Go Toolchain** | **Go 1.22 or higher** | `go version` | Required to compile Go packages and workloads. |
| **Operating System** | Linux (Ubuntu, Debian, Fedora, Arch, Alpine)<br>macOS (12+ Monterey, Ventura, Sonoma)<br>Windows (10/11 via PowerShell / CMD / WSL2) | `uname -srm` (Unix)<br>`[System.Environment]::OSVersion` (Win) | Native multi-arch binaries provided for all platforms. |
| **CPU Architecture** | `amd64` (x86_64), `arm64` (Apple Silicon, AWS Graviton) | `uname -m` | Zero emulation overhead. |
| **Version Control** | `git` (2.30+) | `git --version` | Required for scaffolding and module resolution. |
| **Terminal Client** | Bash, Zsh, Fish, PowerShell | `$SHELL --version` | Shell tab completion scripts available for all shells. |

### Run Environment Check

Run the following one-liner in your terminal to verify your development environment readiness:

```bash
printf "Go:      %s\nGit:     %s\nOS/Arch: %s/%s\n" \
  "$(go version 2>/dev/null || echo 'NOT INSTALLED')" \
  "$(git --version 2>/dev/null || echo 'NOT INSTALLED')" \
  "$(uname -s 2>/dev/null || echo 'Unknown')" \
  "$(uname -m 2>/dev/null || echo 'Unknown')"
```

> [!NOTE]
> If Go is not installed or your version is older than 1.22, download the official installer from [golang.org/dl](https://golang.org/dl) before continuing.

---

## 2. The Krewire Development Lifecycle

Traditional fullstack development often requires juggling multiple disjointed build tools: package managers (`npm`, `pnpm`), bundlers (`vite`, `webpack`), process supervisors, CSS processors, and deployment CLIs.

Krewire replaces this cognitive overhead with a unified, circular development lifecycle orchestrated entirely through a single binary (**`kiw`**) and a single configuration (**`krewire.yaml`**):

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  1. Scaffold │ ──> │ 2. Configure │ ──> │  3. Develop  │
│   kiw new    │     │ krewire.yaml │     │   kiw dev    │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                 │ (Hot Reload)
┌──────────────┐     ┌──────────────┐     ┌──────▼───────┐
│   6. Ship    │ <── │   5. Build   │ <── │   4. Verify  │
│  kiw deploy  │     │  kiw build   │     │   kiw test   │
└──────────────┘     └──────────────┘     └──────────────┘
```

### Lifecycle Stages:

1. **Scaffold (`kiw new`):** Generate a clean, idiomatic project structure tailored to your chosen workload archetype (`--site`, `--app`, `--cli`, `--book`).
2. **Configure (`krewire.yaml`):** Declare build outputs, dev server ports, asset directories, and task runner scripts in a single declarative file.
3. **Develop (`kiw dev`):** Launch a local development server with integrated hot reloading, instant template compilation, and live browser refresh.
4. **Verify (`kiw test` & `kiw vet`):** Validate code correctness with native Go tests, static analysis, and workload contract audits.
5. **Build (`kiw build`):** Compile optimized, zero-dependency release artifacts—either static HTML/CSS/WASM assets or a self-contained Go binary with embedded static files.
6. **Ship (`kiw deploy`):** Package deployment containers or stage distribution bundles into `dist/`.

---

## 3. Hands-On 5-Minute Quickstart

Follow this end-to-end tutorial to install `kiw`, scaffold a project, experience hot reloading, and generate a production build.

### Step 1: Install the `kiw` CLI

On Linux or macOS, run the automated installation script:

```bash
curl -fsSL https://krewire.com/scripts/install.sh | sh
```

For Windows or Go-centric workflows, install directly via `go install`:

```bash
go install github.com/krewire/kiw/cmd/kiw@latest
```

Verify that the CLI is accessible in your system `PATH`:

```bash
kiw version
```

*Expected output:*
```text
kiw v0.1.0 (linux/amd64) built with go1.22
```

---

### Step 2: Scaffold a New Project

Create a new static site project named `my-krewire-app`:

```bash
kiw new my-krewire-app --site --title "My Krewire App"
cd my-krewire-app
```

The `kiw new` command creates a clean, minimal project structure:

```text
my-krewire-app/
├── krewire.yaml          # Project devtool configuration
├── go.mod                # Go module definition
├── layouts/
│   └── Base.kiw          # Root HTML document wrapper
├── pages/
│   ├── index.kiw         # Home page component
│   └── 404.html          # Custom 404 page
├── components/           # Reusable UI components
├── public/               # Static assets (images, icons)
│   ├── favicon.svg
│   └── robots.txt
└── assets/
    └── style.css         # Global stylesheet
```

---

### Step 3: Start the Development Server

Launch the local development server with file watching and live reloading:

```bash
kiw dev
```

*Terminal output:*
```text
2026/09/29 20:00:00 INFO krewire dev server listening at http://localhost:8080
2026/09/29 20:00:00 INFO watching: pages/, layouts/, components/, public/, assets/
```

Open `http://localhost:8080` in your web browser. You will see the default Krewire welcome page rendered instantly.

---

### Step 4: Make a Live Edit

Open `pages/index.kiw` in your favorite code editor and modify the content:

```html
---
title: "Welcome to Krewire"
layout: Base
---

<section class="hero">
  <h1>Building with Krewire is Fast.</h1>
  <p>Zero JavaScript overhead, compiled Go speed, and instant hot reload.</p>
  <button class="btn" onclick="alert('Hello from Go!')">Click Me</button>
</section>

<style>
.hero {
  padding: 4rem 1rem;
  text-align: center;
}
h1 {
  color: var(--primary, #39D353);
  font-size: 2.5rem;
}
.btn {
  padding: 0.75rem 1.5rem;
  font-weight: bold;
  border-radius: 8px;
  cursor: pointer;
}
</style>
```

Save the file. Observe your terminal and browser: the development server detects the file change, recompiles the template in `< 10ms`, and hot-reloads your browser without resetting client state.

---

### Step 5: Build for Production

When you are ready to prepare your application for deployment, run:

```bash
kiw build
```

*Terminal output:*
```text
2026/09/29 20:01:00 INFO building SSG site from file layout root=. output=.krewire/build
created index.html
created 404.html
created assets/style.css
created favicon.svg
created robots.txt
2026/09/29 20:01:00 INFO build complete: 5 files generated in 12ms
```

To preview your production build locally exactly as it will run on a web server:

```bash
kiw serve --port 3000
```

Visit `http://localhost:3000` to verify your optimized production artifact.

---

## 4. Workload Archetype Selection Guide

Krewire supports eight first-class workload archetypes. When bootstrapping a new project, select the archetype that matches your architectural objective:

| Workload Kind | CLI Scaffold Flag | Primary Purpose | Key Packages | Primary Output |
| :--- | :--- | :--- | :--- | :--- |
| **Static Site (`site`)** | `kiw new <name> --site` | Marketing websites, landing pages, blogs | `libs/kiw`, `libs/web` | Static HTML / CSS / Assets |
| **Documentation (`book`)** | `kiw new <name> --book` | Technical books, engineering docs, manuals | `mdbind`, `libs/web` | Multi-chapter static portal |
| **Fullstack App (`app`)** | `kiw new <name> --app` | SaaS platforms, dashboards, dynamic monoliths | `framework/web`, `libs/orm` | Single Go binary + embedded assets |
| **Terminal Tool (`cli`)** | `kiw new <name> --cli` | Developer tools, system utilities, TUIs | `framework/tui`, `libs/cli` | Single standalone CLI binary |
| **Background Worker (`worker`)** | `kiw init --worker` | Async job processors, queue consumers | `framework/queue`, `libs/cron`| Long-running worker daemon |
| **Microservice (`service`)** | `kiw init --service` | High-throughput HTTP/gRPC API microservices | `framework/rpc`, `libs/telemetry` | Cloud-native service binary |
| **Cloud Infra (`infra`)** | `kiw init --infra` | Declarative cloud topologies & provisioning | `framework/infra` | Infrastructure definition & plans |
| **WASM Client (`runtime`)** | `kiw init --runtime` | Client-side reactive UI runtimes in Go | `framework/vdom`, Go WASM | `.wasm` binary + JS bridge |

> [!TIP]
> **Progressive Growth:** You can start with `--site` and later evolve your codebase into an `--app` or extract parts into a `--service` without rewriting your component files or folder layouts.

---

## 5. What is in this Chapter?

This **Getting Started** chapter is divided into three focused, in-depth subchapters designed to give you complete mastery over the Krewire toolchain:

<div class="overview-grid" style="display:grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1.25rem; margin: 1.5rem 0;">

  <div style="border: var(--pop-border); border-radius: 12px; padding: 1.5rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <div style="display:flex; align-items:center; gap:0.5rem; margin-bottom:0.75rem;">
      <span style="display:inline-flex; align-items:center; justify-content:center; width:28px; height:28px; background:var(--primary); color:var(--primary-content); font-weight:900; font-size:0.85rem; border-radius:6px;">2.1</span>
      <h3 style="margin:0; font-size:1.15rem; font-weight:800;">
        <a href="/getting-started/installation" style="color:var(--primary); text-decoration:none;">Installation →</a>
      </h3>
    </div>
    <p style="margin-bottom:1rem; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Complete installation instructions across Linux, macOS, and Windows. Learn automated shell scripting, Go source installation, cryptographic checksum validation, custom install paths, and shell tab completions.
    </p>
    <ul style="margin:0; padding-left:1.2rem; font-size:0.85rem; color:var(--muted);">
      <li>Automated shell installer & checksums</li>
      <li><code>go install</code> & binary releases</li>
      <li>Bash, Zsh, and Fish completions</li>
    </ul>
  </div>

  <div style="border: var(--pop-border); border-radius: 12px; padding: 1.5rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <div style="display:flex; align-items:center; gap:0.5rem; margin-bottom:0.75rem;">
      <span style="display:inline-flex; align-items:center; justify-content:center; width:28px; height:28px; background:var(--primary); color:var(--primary-content); font-weight:900; font-size:0.85rem; border-radius:6px;">2.2</span>
      <h3 style="margin:0; font-size:1.15rem; font-weight:800;">
        <a href="/getting-started/configuration" style="color:var(--primary); text-decoration:none;">Configuration →</a>
      </h3>
    </div>
    <p style="margin-bottom:1rem; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Deep dive into <code>krewire.yaml</code>: project metadata, build targets, server port bindings, asset directory mappings, documentation book integration, and task automation scripts.
    </p>
    <ul style="margin:0; padding-left:1.2rem; font-size:0.85rem; color:var(--muted);">
      <li>Canonical YAML schema specification</li>
      <li>Environment variable overrides (<code>KIW_PORT</code>, <code>KIW_ENV</code>)</li>
      <li>Task automation & custom scripts</li>
    </ul>
  </div>

  <div style="border: var(--pop-border); border-radius: 12px; padding: 1.5rem; background: var(--base-2); box-shadow: var(--pop-shadow-sm);">
    <div style="display:flex; align-items:center; gap:0.5rem; margin-bottom:0.75rem;">
      <span style="display:inline-flex; align-items:center; justify-content:center; width:28px; height:28px; background:var(--primary); color:var(--primary-content); font-weight:900; font-size:0.85rem; border-radius:6px;">2.3</span>
      <h3 style="margin:0; font-size:1.15rem; font-weight:800;">
        <a href="/getting-started/directory-structure" style="color:var(--primary); text-decoration:none;">Directory Structure →</a>
      </h3>
    </div>
    <p style="margin-bottom:1rem; font-size:0.9rem; color:var(--base-2-content); line-height:1.5;">
      Explore the canonical directory layouts for every workload. Master the separation between presentation (<code>pages/</code>, <code>layouts/</code>, <code>components/</code>), static files (<code>public/</code>), manuscripts (<code>content/</code>), and private Go logic (<code>internal/</code>).
    </p>
    <ul style="margin:0; padding-left:1.2rem; font-size:0.85rem; color:var(--muted);">
      <li>Standard project anatomy & file roles</li>
      <li>Workload-specific layout variants</li>
      <li>Clean Architecture & SRP boundaries</li>
    </ul>
  </div>

</div>

---

## 6. Primary CLI Commands Cheat Sheet

Keep these everyday `kiw` commands at your fingertips during development:

| Command | Syntax | Description |
| :--- | :--- | :--- |
| **`new`** | `kiw new <name> [--site\|--app\|--cli\|--book]` | Scaffold a new project kernel with the selected workload template. |
| **`dev`** | `kiw dev` | Start the local development server with file watching and hot reload. |
| **`build`** | `kiw build` | Compile the project into production-ready static assets or binaries. |
| **`serve`** | `kiw serve [--port 3000]` | Preview the static build output locally over HTTP. |
| **`run`** | `kiw run [task\|file.go] [-- args]` | Execute a custom task from `krewire.yaml` or run a Go entry point. |
| **`test`** | `kiw test` | Execute unit and integration tests across all packages. |
| **`vet`** | `kiw vet` | Run standard Go static analysis (`go vet`) across the project. |
| **`fmt`** | `kiw fmt --write` | Format all Go and `.kiw` component files according to conventions. |
| **`version`** | `kiw version` | Display current CLI and framework release versions. |
| **`info`** | `kiw info` | Print detailed project environment and diagnostic information. |
| **`help`** | `kiw help [command]` | Display comprehensive documentation for any CLI command. |

---

## 7. Common Beginner Pitfalls & Solutions

### 1. `kiw: command not found` after Installation

**Cause:** The directory where `kiw` was installed is not included in your shell's `PATH` environment variable.

**Resolution:**
- If installed via the shell installer as non-root, add `~/.local/bin` to your shell profile (`~/.bashrc`, `~/.zshrc`):
  ```bash
  export PATH="$HOME/.local/bin:$PATH"
  ```
- If installed via `go install`, ensure Go's binary path is in your `PATH`:
  ```bash
  export PATH="$(go env GOPATH)/bin:$PATH"
  ```
- Reload your profile: `source ~/.bashrc` or `source ~/.zshrc`.

---

### 2. Port Collision (`address already in use: 8080`)

**Cause:** Another service (e.g. Tomcat, Docker, or another dev server) is occupying port `8080`.

**Resolution:**
- Specify a custom port in `krewire.yaml`:
  ```yaml
  dev:
    port: "3000"
  ```
- Or override on the command line using environment variables:
  ```bash
  KIW_PORT=3000 kiw dev
  ```

---

### 3. Outdated Go Version (`requires go >= 1.22`)

**Cause:** Your system has an older Go distribution (e.g. Go 1.18 or 1.20) installed via standard OS package managers (such as older Ubuntu `apt`).

**Resolution:**
- Check your current Go binary: `go version`.
- Remove the stale system package and install the official upstream distribution from [golang.org/dl](https://golang.org/dl).

---

## Next Steps

Now that you understand the big picture and development workflow, dive deep into the implementation details:

- Proceed to [**2.1 Installation →**](/getting-started/installation) to install and configure `kiw` for your specific operating system.
- Or explore the architectural foundation in [**1. Overview →**](/docs).
