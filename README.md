# Krewire

Official monorepo for Krewire platform applications and ecosystem services.

## Architecture & Layout

This repository is structured as a Go monorepo coordinating multi-module applications and packages using Go workspaces (`go.work`).

```text
.
├── apps/         # Deployable applications and services
├── packages/     # Shared domain libraries and internal packages
├── tools/        # Developer tooling and automation scripts
├── go.work       # Go workspace definition
└── README.md
```

## Prerequisites

- [Go](https://go.dev/) 1.27.1 or higher
- [Git](https://git-scm.com/)

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/krewire/krewire.git
   cd krewire
   ```

2. Synchronize workspace dependencies:
   ```bash
   go work sync
   ```

## License

[MIT](LICENSE) © 2026 Krewire Contributors
