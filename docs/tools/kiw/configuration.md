# `kiw` Configuration Specification (`krewire.yaml`)

The `krewire.yaml` file serves as the Single Source of Truth (SSOT) for configuring project metadata, workload behavior, server settings, custom scripts, and dependencies.

---

## 1. Minimal Specification

```yaml
version: "0.1.0"
project:
  name: my-app
  kind: app
```

---

## 2. Full Schema Reference

```yaml
version: "0.1.0"

# Project metadata & invariants
project:
  name: my-service          # Required: kebab-case ^[a-z][a-z0-9-]*$
  kind: service             # Required: app | cli | site | book | worker | service | infra | kernel
  description: "Core API"
  author: "Krewire Team"

# HTTP server settings (for app, service, site)
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "15s"
  write_timeout: "15s"

# SSG settings (kind: site)
ssg:
  content_dir: "content"
  templates_dir: "templates"
  output_dir: ".krewire/build"
  clean_urls: true
  minify: true

# Book settings (kind: book)
book:
  source_dir: "manuscript"
  output_dir: ".krewire/build"
  title: "User Manual"

# Worker settings (kind: worker)
worker:
  concurrency: 4
  queues:
    - name: "default"
      rate_limit: 100

# Custom project scripts executable via `kiw run <script>`
scripts:
  migrate: "go run ./cmd/migrate"
  seed: "go run ./cmd/seed"
  lint: "golangci-lint run"
```

---

## 3. Environment Variables & Overlays

Configuration values can be overridden via `.env` files or environment variables:
- `KREWIRE_SERVER_PORT=9000` overrides `server.port`.
- `KREWIRE_SERVER_HOST="127.0.0.1"` overrides `server.host`.
- `DEBUG=1` enables verbose logging and diagnostic stack traces.
