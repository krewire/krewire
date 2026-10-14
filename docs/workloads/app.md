# Workload: Fullstack App (`app`)

The `app` workload targets fullstack, dynamic web applications that combine server-side rendering (SSR), dependency injection, and client-side interactive WebAssembly (WASM) islands.

---

## 1. Architecture

An `app` project leverages:
- **`packages/web`**: Fast HTTP router with middleware pipeline (CORS, CSRF, security headers, compression).
- **`packages/app`**: Dependency injection container for assembling database connections, services, and handlers.
- **`packages/ui`**: Shared UI component library written in Go.
- **`packages/runtime`**: Client-side WebAssembly runtime (`GOOS=js GOARCH=wasm`) providing Virtual DOM (VDOM) and interactive widget islands.

```text
Browser Client
  ├── HTML / SSR from server
  └── WebAssembly Island (packages/runtime: VDOM + reactive widgets)
       ▲
       │ HTTP / JSON API / WebSocket
       ▼
Server (Go Binary)
  ├── packages/web (Router, Middleware, Gateway)
  ├── packages/app (DI Container, Service Assembly)
  ├── packages/sec (Security headers, CSRF, PII redaction)
  └── packages/storage (Database, Object Storage)
```

---

## 2. Project Layout

```text
my-app/
├── krewire.yaml          # Declarative configuration
├── main.go               # Application entrypoint & DI assembly
├── handlers/             # HTTP route handlers
├── services/             # Business logic layer
├── views/                # HTML templates & server components
├── client/               # WebAssembly interactive widgets (compiled to wasm)
└── static/               # CSS, images, and public assets
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-app
  kind: app

server:
  host: "0.0.0.0"
  port: 8080

web:
  ssr: true
  wasm:
    enabled: true
    source: "./client"
    output: "./static/app.wasm"
```

---

## 4. Development & Build

```bash
# Run local development with live-reload
kiw dev

# Run test suite
kiw test

# Compile deployable binary with embedded assets
kiw build

# Run production binary
./bin/my-app
```
