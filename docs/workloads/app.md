# Workload: Fullstack App (`app`)

The `app` workload targets fullstack, dynamic web applications that combine server-side rendering (SSR), dependency injection, and client-side interactive WebAssembly (WASM) islands.

---

## 1. Architecture

An `app` project leverages:
- **`web`**: Fast HTTP router with middleware pipeline (CORS, CSRF, security headers, compression).
- **`app`**: Dependency injection container for assembling database connections, services, and handlers.
- **`ui`**: Shared UI component library written in Go.
- **`runtime`**: Client-side WebAssembly runtime (`GOOS=js GOARCH=wasm`) providing Virtual DOM (VDOM) and interactive widget islands.

```text
Browser Client
  ├── HTML / SSR from server
  └── WebAssembly Island (runtime: VDOM + reactive widgets)
       ▲
       │ HTTP / JSON API / WebSocket
       ▼
Server (Go Binary)
  ├── web (Router, Middleware, Gateway)
  ├── app (DI Container, Service Assembly)
  ├── sec (Security headers, CSRF, PII redaction)
  └── storage (Database, Object Storage)
```

---

## 2. Project Layout

The `krew init --app` scaffold writes:

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
project:
  name: my-app
  version: "0.1.0"

dev:
  port: 8080

server:
  host: "0.0.0.0"
  port: 8080

# Optional client WASM build
wasm:
  entry: "./wasm"     # main package path
  name: "runtime"     # output module name
```

---

## 4. Development & Build

```bash
# Run local development with live-reload
krew dev

# Run test suite
krew test

# Compile the binary into .krewire/build/bin/<name>
krew build

# Run the built binary
./.krewire/build/bin/my-app
```
