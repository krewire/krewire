# Getting Started with Krewire Core Monorepo

This guide helps developers get up and running with the **Krewire core monorepo** (`github.com/krewire/krewire`), containing all domain libraries (`pkg/`), the developer CLI (`krew`), and templates (`templates/`).

## Prerequisites

- **Go**: Version 1.27.1 or newer.
- **Git**: Configured for your GitHub account.
- **Make / Bash**: For executing development targets.

## Local Setup

1. **Clone the repository** (or navigate to `krewire/` within the workspace):
   ```bash
   git clone https://github.com/krewire/krewire.git
   cd krewire
   ```

2. **Build the `krew` CLI**:
   The `krew` CLI is the unified entrypoint for the ecosystem:
   ```bash
   cd ../krew && go build -o ../bin/krew ./cmd/krew
   ```
   Verify that `krew` runs:
   ```bash
   ../bin/krew version
   ```

3. **Verify Go Modules**:
   ```bash
   go mod tidy
   ```

## Running the Automated Test Suite

Run tests across all modular packages:

```bash
go test ./...
```

For package-specific testing:

```bash
go test -v ./pkg/web/...
go test -v ./pkg/model/...
```

## First-Project Quickstart (Zero to Shipped in 60 Seconds)

Krewire enables an immediate, reproducible onboarding experience with zero npm/node dependencies:

### 1. Scaffold a New Project
```bash
krew new hello-world
cd hello-world
```
*Expected Output*:
```text
Created project hello-world (kind: app, module: example.com/hello-world)
Equipped starter template with main.go and krewire.yaml.
```

### 2. Inspect Project Shape
```bash
krew info
```
*Expected Output*:
```text
Project: hello-world
Shape:   app (main.go)
Go:      1.24+ (pure Go, CGO_ENABLED=0)
Output:  .krewire/build
```

### 3. Build Production Artifacts
```bash
krew build
```
*Expected Output*:
```text
INFO building binary kind=app entry=. output=.krewire/build/bin/hello-world
created .krewire/build/bin/hello-world
```

### 4. Run the Application
```bash
krew run
```
*Expected Output*:
```text
Starting hello-world...
HTTP server listening on http://127.0.0.1:8080
```

### 5. Everyday Commands
`krew help` lists the core workflow first (`new`, `dev`, `build`, `test`, `add`,
`make`); `krew help --all` describes every command. While editing, use
`krew dev` (watches and reloads); `krew run` runs once without watching, and
`krew serve` previews a finished `krew build`. Run `krew new` without arguments to
be asked for the name and kind, or `krew init` to set up an existing directory.
Add ready-made UI components with `krew add ui/<name>` (browse with `krew ui list`).
See the [command reference](krew/commands.md).

### 6. Troubleshooting & FAQ
- **Error: `kiw: command not found`**: Ensure `$(go env GOPATH)/bin` or `./bin` is added to your `$PATH`.
- **Port Conflict (`bind: address already in use`)**: Specify a custom port via `PORT=8088 krew run` or configure `dev.port: 8088` in `krewire.yaml`.
- **Pure-Go Invariant**: All builds enforce `CGO_ENABLED=0` by default; no host C compilers or native shared libraries are required.

---

## Developing with `krew`

Use the local `krew` binary to test project lifecycle operations:

- `bin/krew new my-app --app`: Scaffolds a new fullstack application.
- `bin/krew info`: Inspects the detected project shape and workload kind.
- `bin/krew dev`: Starts development server with live reload.
- `bin/krew build`: Builds production release artifacts.

## Next Steps

- Explore the [Architecture Specification](foundation/architecture.md) for layering rules.
- Review [Contributing Guidelines](contributing.md) before submitting code changes.
