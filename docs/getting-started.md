# Getting Started with Krewire Core Monorepo

This guide helps developers get up and running with the **Krewire core monorepo** (`github.com/krewire/krewire`), containing all domain libraries (`pkg/`), the developer CLI (`tools/kiw`), and templates (`templates/`).

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

2. **Build the `kiw` CLI**:
   The `kiw` CLI is the unified entrypoint for the ecosystem:
   ```bash
   go build -o bin/kiw ./tools/kiw/cmd/kiw
   ```
   Verify that `kiw` runs:
   ```bash
   ./bin/kiw version
   ```

3. **Verify Go Modules**:
   ```bash
   go mod tidy
   ```

## Running the Automated Test Suite

Run tests across all 37 modular packages:

```bash
go test ./...
```

For package-specific testing:

```bash
go test -v ./pkg/web/...
go test -v ./tools/kiw/...
```

## Developing with `kiw`

Use the local `kiw` binary to test project lifecycle operations:

- `bin/kiw new my-app --app`: Scaffolds a new fullstack application.
- `bin/kiw info`: Inspects the detected project shape and workload kind.
- `bin/kiw dev`: Starts development server with live reload.
- `bin/kiw build`: Builds production release artifacts.

## Next Steps

- Explore the [Architecture Specification](foundation/architecture.md) for layering rules.
- Review [Contributing Guidelines](contributing.md) before submitting code changes.
