---
title: "Upgrade Guide"
description: "Versioning policies, CLI upgrade instructions, and configuration migration guide for Krewire."
date: "2026-09-29"
---

# Upgrade Guide

This guide details the versioning policies, upgrade procedures, and configuration migrations across the Krewire ecosystem.

---

## 1. Versioning Strategy & Compatibility Promise

Krewire strictly adheres to **Semantic Versioning (SemVer 2.0.0)** across all ecosystem repositories (`framework`, `libs`, `kiw`, `mdbind`):

$$\text{vMAJOR}.\text{MINOR}.\text{PATCH}$$

- **`PATCH` releases (`v0.3.1` → `v0.3.2`):** Bug fixes, performance optimizations, and documentation updates. Guaranteed 100% backward compatible with zero breaking changes.
- **`MINOR` releases (`v0.3.0` → `v0.4.0`):** New workload capabilities, additive packages, and new CLI flags. Per specification `KWF-META-CMBZJ` and `KWN-DEVTOOL-Z0VFC`:
  > *"New capabilities are introduced as additive packages; breaking changes are not allowed in minor/patch releases."*
- **`MAJOR` releases (`v1.0.0`):** Significant architectural evolutions. Any backward-incompatible changes will be preceded by at least one minor release containing deprecation warnings and automated migration helpers.

---

## 2. Upgrading the `kiw` CLI

To upgrade your local installation of `kiw` to the latest release:

### Option A: Shell Script (Linux & macOS)

Re-run the automated installer:

```bash
curl -fsSL https://krewire.com/scripts/install.sh | sh
```

The script automatically downloads the latest release matching your CPU architecture and replaces the binary in `/usr/local/bin/kiw` (or `~/.local/bin/kiw`).

### Option B: Via `go install`

```bash
go install github.com/krewire/kiw/cmd/kiw@latest
```

### Verify the Updated Version

```bash
kiw version
```

Confirm that the CLI version matches the current release (`v0.1.0`+).

---

## 3. Upgrading Go Dependencies

In your application repository, update the Krewire framework and library modules:

```bash
# Update framework and runtime packages
go get -u github.com/krewire/framework@latest

# Update core standard-library utilities
go get -u github.com/krewire/libs@latest

# Prune unused dependencies and sync go.sum
go mod tidy
```

Run your automated test suite to verify compatibility:

```bash
kiw test
# Or standard go test
go test ./...
```

---

## 4. Configuration Migration Guide: `krewire.yaml`

In release `v0.3.0`, Krewire redesigned `krewire.yaml` to enforce strict **Separation of Concerns**.

### The Problem in Earlier Versions (v0.1 – v0.2)

In early versions, `krewire.yaml` conflated devtool settings with application runtime state: page titles, SEO descriptions, navbar URLs, footer copyright text, and color palettes were all placed in `krewire.yaml`. This violated the Single Responsibility Principle and made multi-page metadata cumbersome.

### The Modern Standard (v0.3+)

`krewire.yaml` is now dedicated solely to **`kiw` devtool and build pipeline configuration**:

#### Before (v0.1 / v0.2 Legacy):

```yaml
# ❌ LEGACY: Mixed devtool and application concerns
project:
  name: "my-site"
  kind: "site"
version: "v0.1.0"
title: "My Awesome Site"
description: "A fast website"
base: "/"
output: ".krewire/build"
nav:
  - text: "Docs"
    url: "/docs"
  - text: "GitHub"
    url: "https://github.com/..."
footer: "© 2026 My Company"
theme:
  light:
    primary: "#39D353"
```

#### After (v0.3+ Modern):

```yaml
# ✅ MODERN: Strict devtool-only configuration
project:
  name: "my-site"
  kind: "site"
  version: "v0.1.0"
  author: "Krewire Contributors"

build:
  output: ".krewire/build"
  base: "/"

dev:
  port: "8080"
```

### Where Settings Belong Now

| Setting / Data | Previous Location | Modern Location |
| :--- | :--- | :--- |
| **Project Identity & Kind** | `krewire.yaml` (`name`, `kind`) | `project.name`, `project.kind` in `krewire.yaml` |
| **Build Output Directory** | `output:` in `krewire.yaml` | `build.output:` in `krewire.yaml` |
| **Dev Server Port** | Hardcoded or flags | `dev.port:` in `krewire.yaml` |
| **Page Title & SEO** | `title:`, `description:` | YAML frontmatter in `.kiw` (`pages/*.kiw`) or Markdown |
| **Navigation & Links** | `nav:` array in `krewire.yaml` | Reusable layout shell (`layouts/Base.kiw` or `components/Nav.kiw`) |
| **Footer & Copyright** | `footer:` string | Layout footer section (`layouts/Base.kiw`) |
| **Theme & Color Palettes** | `theme.light`, `theme.dark` | CSS design tokens (`theme.css` / `mdbind.css`) |
| **Database & API Keys** | Injected via custom YAML | Standard environment variables (`.env`) |

---

## 5. Backward Compatibility Bindings

To ensure that projects configured on older CLI versions do not break, `kiw` v0.3+ maintains backward-compatible fallback bindings for top-level keys (`version`, `author`, `output`, `base`). If present, `kiw` loads them transparently while issuing a gentle upgrade recommendation.

---

## 6. Clean Section Index Routing

Starting with `mdbind v0.3.2`, technical manuscript builds export both `<section>.html` and `<section>/index.html` (for example, `.krewire/build/docs/index.html` and `.krewire/build/docs.html`).

If you host your static site behind Nginx or a Docker container, update your `try_files` directive to enable clean URLs without trailing-slash redirects:

```nginx
location / {
    try_files $uri $uri.html $uri/index.html $uri/ /index.html =404;
}
```

---

## Need Help Upgrading?

If you encounter unexpected build behaviors or test failures during an upgrade:

- Open a discussion on the [Krewire GitHub Forum](https://github.com/orgs/krewire/discussions).
- Check the [issue tracker](https://github.com/krewire/krewire/issues) for known issues.
- Return to [**1.1 Krewire Framework →**](/docs/krewire-framework) or [**1.3 Getting Started →**](/docs/getting-started).
