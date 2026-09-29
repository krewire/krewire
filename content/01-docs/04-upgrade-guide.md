---
title: "Upgrade Guide"
description: "Versioning policies, CLI upgrade instructions, and configuration architecture for Krewire."
date: "2026-09-29"
---

# Upgrade Guide

This guide details the versioning policies, upgrade procedures, and configuration standards across the Krewire ecosystem.

---

## 1. Versioning Strategy & Compatibility Promise

Krewire strictly adheres to **Semantic Versioning (SemVer 2.0.0)** across all ecosystem repositories (`framework`, `libs`, `kiw`, `mdbind`):

$$\text{vMAJOR}.\text{MINOR}.\text{PATCH}$$

- **`PATCH` releases (`v0.1.0` → `v0.1.1`):** Bug fixes, performance optimizations, and documentation updates. Guaranteed 100% backward compatible with zero breaking changes.
- **`MINOR` releases (`v0.1.0` → `v0.2.0`):** New workload capabilities, additive packages, and new CLI flags. Per specification `KWF-META-CMBZJ` and `KWN-DEVTOOL-Z0VFC`:
  > *"New capabilities are introduced as additive packages; breaking changes are not allowed in minor/patch releases."*
- **`MAJOR` releases (`v1.0.0`):** Significant architectural evolutions. Any backward-incompatible changes will be preceded by deprecation warnings and automated migration helpers.

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

## 4. Configuration Architecture: `krewire.yaml`

Krewire enforces strict **Separation of Concerns** in project configuration:

`krewire.yaml` is dedicated solely to **`kiw` devtool and build pipeline configuration**:

```yaml
# Krewire Devtool Configuration (kiw)
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

### Separation of Concerns Matrix

| Setting / Data | Target Location | Rationale |
| :--- | :--- | :--- |
| **Project Identity & Kind** | `project.name`, `project.kind` in `krewire.yaml` | Declares workload and metadata for devtools |
| **Build Output Directory** | `build.output:` in `krewire.yaml` | Configures target build destination |
| **Dev Server Port** | `dev.port:` in `krewire.yaml` | Local development server port |
| **Page Title & SEO** | YAML frontmatter in `.kiw` (`pages/*.kiw`) or Markdown | Content-specific metadata belongs with content |
| **Navigation & Links** | Reusable layout shell (`layouts/Base.kiw` or `components/Nav.kiw`) | Presentation structure belongs in UI components |
| **Footer & Copyright** | Layout footer section (`layouts/Base.kiw`) | UI components manage responsive layout display |
| **Theme & Color Palettes** | CSS design tokens (`theme.css` / `mdbind.css`) | Styling variables belong in stylesheets |
| **Database & API Keys** | Standard environment variables (`.env`) | Sensitive credentials must never be committed |

---

## 5. Clean Section Index Routing

Technical manuscript builds export both `<section>.html` and `<section>/index.html` (for example, `.krewire/build/docs/index.html` and `.krewire/build/docs.html`).

If you host your static site behind Nginx or a Docker container, configure your `try_files` directive to enable clean URLs without trailing-slash redirects:

```nginx
location / {
    try_files $uri $uri.html $uri/index.html $uri/ /index.html =404;
}
```

---

## Need Help Upgrading?

If you encounter unexpected build behaviors or test failures:

- Open a discussion on the [Krewire GitHub Forum](https://github.com/orgs/krewire/discussions).
- Check the [issue tracker](https://github.com/krewire/krewire/issues) for known issues.
- Return to [**1.1 Krewire Framework →**](/docs/krewire-framework) or proceed to [**2. Getting Started →**](/getting-started).
