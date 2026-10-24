# Workload: Fullstack App (`app`)

The `app` workload targets fullstack, dynamic web applications that combine server-side rendering (SSR), dependency injection, and client-side interactive WebAssembly (WASM) islands.

---

## 1. Architecture

An `app` project leverages:
- **`pkg/web`**: Fast HTTP router with middleware pipeline (CORS, CSRF, security headers, compression).
- **`pkg/app`**: Dependency injection container for assembling database connections, services, and handlers.
- **`pkg/ui`**: Shared UI component library written in Go.
- **`pkg/runtime`**: Client-side WebAssembly runtime (`GOOS=js GOARCH=wasm`) providing Virtual DOM (VDOM) and interactive widget islands.

```text
Browser Client
  ├── HTML / SSR from server
  └── WebAssembly Island (pkg/runtime: VDOM + reactive widgets)
       ▲
       │ HTTP / JSON API / WebSocket
       ▼
Server (Go Binary)
  ├── pkg/web (Router, Middleware, Gateway)
  ├── pkg/app (DI Container, Service Assembly)
  ├── pkg/sec (Security headers, CSRF, PII redaction)
  └── pkg/storage (Database, Object Storage)
```

---

## 2. Project Layout

The `kiw init --app` scaffold writes:

```text
my-app/
├── krewire.yaml
├── main.go
├── internal/
│   ├── app/              # Application assembly & lifecycle
│   ├── config/           # Typed configuration
│   └── http/             # HTTP server & routing
├── web/
│   ├── layouts/          # Page shells
│   ├── pages/            # Page components
│   ├── components/       # Reusable components
│   └── theme/            # Theme and palette
├── assets/
│   ├── embed.go          # Embedded asset bundle
│   └── public/           # CSS and JS assets
├── .env
├── .env.example
└── README.md
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

# Optional client WASM build (KWF-T4X9P)
wasm:
  entry: "./wasm"     # main package path
  name: "runtime"     # output module name
```

---

## 4. Development & Build

```bash
# Run local development with live-reload
kiw dev

# Run test suite
kiw test

# Compile the binary into .krewire/build/bin/<name>
kiw build

# Run the built binary
./.krewire/build/bin/my-app
```
