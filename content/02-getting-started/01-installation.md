---
title: "Installation"
description: "Comprehensive installation guide for the kiw CLI developer tool on Linux, macOS, and Windows."
date: "2026-09-29"
---

# Installation

The **`kiw`** CLI is the unified developer multi-tool for the Krewire ecosystem. A single binary powers project scaffolding, code generation, local development servers, testing, asset compilation, and production deployment across all eight workloads.

---

## 1. System Requirements

Before installing `kiw`, verify that your development environment satisfies the baseline requirements:

| Requirement | Supported Specifications | Notes |
| :--- | :--- | :--- |
| **Operating System** | Linux (Ubuntu, Debian, Fedora, Arch, Alpine), macOS (12+), Windows (10/11) | Native binaries available for all platforms. |
| **Architecture** | `amd64` (x86_64), `arm64` (Apple Silicon, ARM64 servers) | Verified multi-arch static builds. |
| **Go Runtime** | **Go 1.22 or higher** | Required when building from source or running Go workloads. |
| **Version Control** | `git` (2.30+) | Required for template bootstrapping and module fetching. |

Check your current Go version:

```bash
go version
```

> [!NOTE]
> If you do not have Go installed, download and install the official package from [golang.org/dl](https://golang.org/dl).

---

## 2. Installation Methods

Choose the installation method best suited to your platform and workflow:

### Method A: Automated Shell Installer (Linux & macOS)

The recommended installation method for Linux and macOS systems uses the automated installation script:

```bash
curl -fsSL https://krewire.com/scripts/install.sh | sh
```

#### How the Installer Works:
1. **Platform Detection:** Inspects your kernel (`Linux`/`Darwin`) and CPU architecture (`x86_64`/`arm64`).
2. **Tag Resolution:** Resolves the latest official stable release (e.g. `v0.1.0`) from the GitHub release registry.
3. **Checksum Verification:** Downloads the official `kiw_checksums.txt` manifest and cryptographically validates the SHA256 hash of the downloaded archive before extraction.
4. **Atomic Placement:** Places the binary into `/usr/local/bin` using `install -m 755`. If run without root permissions, it automatically falls back to `~/.local/bin`.
5. **Secure Cleanup:** Traps exit signals (`INT`, `TERM`, `EXIT`) to guarantee temporary extraction directories are safely erased.

To customize the installation directory, pass the `INSTALL_DIR` environment variable:

```bash
curl -fsSL https://krewire.com/scripts/install.sh | INSTALL_DIR="$HOME/bin" sh
```

---

### Method B: Install via Go Toolchain (Cross-Platform / Windows)

If you already have Go installed and `$GOPATH/bin` configured in your system `PATH`, install `kiw` directly from source:

```bash
go install github.com/krewire/kiw/cmd/kiw@latest
```

To pin a specific version (e.g. `v0.1.0`):

```bash
go install github.com/krewire/kiw/cmd/kiw@v0.1.0
```

#### Configuring Your PATH:
Ensure your Go binary directory is included in your shell profile (`~/.bashrc`, `~/.zshrc`, or Windows Environment Variables):

```bash
# Add to ~/.bashrc or ~/.zshrc if not already present:
export PATH="$PATH:$(go env GOPATH)/bin"
```

On Windows PowerShell:

```powershell
$env:Path += ";$((go env GOPATH))\bin"
```

---

## 3. Verifying the Installation

After installation completes, verify that the binary is accessible and operational:

### 1. Version Inspection

```bash
kiw version
```

Expected output:

```text
kiw Krewire Devtool
  version:   v0.1.0
  built:     2026-09-29
  target:    site, book, app, cli, worker, service, infra, runtime
```

### 2. Environment Diagnostics

Run `kiw info` to inspect your active toolchain, operating system, and Go environment:

```bash
kiw info
```

### 3. Compatibility Check

Run `kiw compat` inside any Krewire project directory to verify that your local CLI matches the framework dependencies in `go.mod`:

```bash
kiw compat
```

---

## 4. Upgrading `kiw`

To upgrade to the latest stable release:

- **Via Shell Installer:**
  ```bash
  curl -fsSL https://krewire.com/scripts/install.sh | sh
  ```
- **Via Go Toolchain:**
  ```bash
  go install github.com/krewire/kiw/cmd/kiw@latest
  ```

---

## 5. Uninstalling `kiw`

Because `kiw` is distributed as a single static binary with no background daemons or registry entries, uninstallation is straightforward:

```bash
# For standard system-wide install:
sudo rm -f /usr/local/bin/kiw

# For user-local install:
rm -f ~/.local/bin/kiw

# For Go bin install:
rm -f "$(go env GOPATH)/bin/kiw"
```

---

## Next Steps

Now that your toolchain is installed, proceed to [**2.2 Configuration →**](/getting-started/configuration) to learn how to configure projects using `krewire.yaml`.
