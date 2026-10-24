# Multi-Runtime Coexistence — Node.js & Laravel Integration Guide

## Description
This guide explains how Krewire applications, services, and background workers seamlessly coexist, coordinate, and integrate with foreign host ecosystems (such as Node.js fullstack frameworks and Laravel PHP applications) within a single project directory.

---

## 1. Architectural Philosophy

Krewire operates on the principle that developers should not be forced into an all-or-nothing framework migration. Modern architectures frequently pair:
- A **Node.js frontend / meta-framework** (e.g., SvelteKit, Nuxt, Next.js, Remix, Astro) with high-performance Go backend microservices.
- A **Laravel web monolith** (Blade, Eloquent ORM, auth) with compiled Go background workers, stream processors, or gRPC endpoints.

Rather than conflicting, Krewire provides first-class primitives for:
1. **Zero-interference directory layout** (side-by-side scaffolding).
2. **Deterministic port detection and conflict prevention**.
3. **Dual-direction environment variable coordination and namespacing**.
4. **Standard IPC process bridges via stdin/stdout JSON streaming**.
5. **Cross-runtime distributed tracing (W3C `traceparent`)**.
6. **Multi-runtime structured logging with distinguishable source markers (`[go]`)**.
7. **Unified build orchestration (`build.depends_on`)**.

---

## 2. Directory Layout & Side-by-Side Scaffolding

When equipping a Go service inside an existing Node.js or Laravel codebase, use `kiw new <name> --dir <host-dir>`:

```bash
# In a Node.js project:
kiw new api --dir .

# In a Laravel project:
kiw new worker --dir .
```

### Resulting Coexistent Structure

```text
my-project/
├── package.json              # Node.js manifest (untouched)
├── composer.json             # Laravel manifest (untouched)
├── artisan                   # Laravel CLI (untouched)
├── .gitignore                # Host .gitignore (untouched)
├── src/                      # Host frontend or app source (untouched)
│
└── api/                      # Krewire sub-project
    ├── go.mod                # Isolated Go module
    ├── krewire.yaml          # Krewire workload configuration
    ├── main.go               # Go entrypoint
    └── .gitignore            # Subdirectory .gitignore (.krewire)
```

Host files (`package.json`, `node_modules/`, `composer.json`, `app/`, `config/`) are completely isolated. `kiw` never overwrites or mutates foreign manifests.

---

## 3. Host Ecosystem & Co-located Project Discovery

Running `kiw info` from anywhere in the project tree inspects both the local environment and foreign host manifests:

```text
$ kiw info
─ Environment ─────────────────────────────────
  CLI              kiw v0.1.0
  Ecosystem        Krewire v0.1.0 (dev)
  Signature        build with krewire
  Go               go1.27.1 (linux/amd64)

─ Project ────────────────────────────────────
  Directory        /workspace/my-project
  Project kind     none

─ Host Ecosystem ─────────────────────────────
  Platform         Node.js (package.json)
  Framework        SvelteKit
  Package manager  pnpm

─ Co-located Krewire ─────────────────────────
  api/             app (my-project/api)
  worker/          worker (my-project/worker)

─ Ports & Coexistence ────────────────────────
  Node.js port     5173 (package.json: scripts.dev)
  Krewire port     8080 (krewire.yaml)
```

`kiw info` automatically detects:
- Package managers: `npm`, `pnpm`, `yarn`, `bun`, `composer`.
- Frontend & backend frameworks: `SvelteKit`, `Nuxt`, `Next.js`, `Remix`, `Astro`, `Fastify`, `Express`, `Laravel`.
- Co-located Krewire workloads in immediate subdirectories.

---

## 4. Port Conflict Prevention

When Node.js dev servers (`:3000`, `:5173`) and Krewire servers (`:8080`) run simultaneously:
1. `kiw` scans `package.json`, `composer.json`, and `.env` for declared ports.
2. If both runtimes attempt to bind to the same port, `kiw` logs an explicit conflict warning (`FRK-CONF-030`).
3. Port resolution precedence:
   - CLI flag: `--addr :<port>` (highest priority)
   - Configuration: `server.port` in `krewire.yaml`
   - Dotenv file: `PORT=` or `KREWIRE_<NAMESPACE>_PORT=`
   - Default fallback: `:8080`

---

## 5. Process Bridge Pattern (JSON IPC)

For lightweight, zero-network inter-process communication between Node.js or Laravel and Go, `kiw run` provides a standard process bridge:
- Standard input (`os.Stdin`) is connected directly to the spawned binary.
- Standard output (`os.Stdout`) and standard error (`os.Stderr`) stream back to the caller.

### Node.js Example (`child_process`)

```javascript
import { spawn } from "node:child_process";

const child = spawn("kiw", ["run", "api"]);

child.stdin.write(JSON.stringify({ action: "compute", data: [1, 2, 3] }) + "\n");
child.stdin.end();

child.stdout.on("data", (data) => {
  const response = JSON.parse(data.toString());
  console.log("Response from Go service:", response);
});
```

### Laravel Example (Symfony `Process`)

```php
use Symfony\Component\Process\Process;

$process = new Process(['kiw', 'run', 'worker']);
$process->setInput(json_encode(['task' => 'generate_pdf', 'id' => 42]));
$process->run();

$output = json_decode($process->getOutput(), true);
```

---

## 6. Cross-Runtime Distributed Tracing

Requests traveling across the runtime boundary (Node.js $\rightarrow$ Go or Laravel $\rightarrow$ Go) maintain end-to-end distributed trace context:
- Standard W3C header **`traceparent`** (format: `00-<trace_id>-<span_id>-<flags>`) is extracted automatically by `web.RequestID()` and `tracing.Extract()`.
- The 16-byte trace ID is extracted and propagated into request contexts, access logs, and response headers (`X-Request-ID` and `traceparent`).
- Works transparently with OpenTelemetry instrumentation in Node.js (`@opentelemetry/api`) and Laravel/PHP (OpenTelemetry PHP / Guzzle).

---

## 7. Multi-Runtime Structured Logging

When Go microservice logs and host logs are aggregated into shared log collectors (Datadog, Grafana Loki, AWS CloudWatch):
- Use `log.WithSource(logger, "go")` to tag structured records with `source="go"`.
- Use `log.SourceTag("go")` to emit standardized human-readable prefixes like `[go]` alongside `[js]` or `[laravel]`.

```go
import "github.com/krewire/krewire/pkg/log"

logger := log.WithSource(slog.Default(), "go")
logger.Info("processing order", "order_id", 1234)
// Output JSON: {"time":"...","level":"INFO","msg":"processing order","source":"go","order_id":1234}
```

---

## 8. AI Boost Non-Interference

Running `kiw boost install <dir>` installs the AI Agent Constitution (`AGENTS.md`), OpenCode configuration (`opencode.json`), and triad skills (`.agents/`) into any Node.js or Laravel project without modifying host code:
- The top-level `AGENTS.md` contains an explicit project-agnostic disclaimer:
  > *If the current project is not a Krewire project, follow the generic principles and skip Krewire-specific sections marked Krewire.*
- Agents will safely operate on JavaScript, TypeScript, and PHP codebases using universal SDLC principles.

---

## Quick References

- [`krewire.yaml` Configuration Reference](../tools/kiw/configuration.md) — Multi-runtime and port options.
- [`kiw` Commands Reference](../tools/kiw/commands.md) — CLI commands and flags.
- [Architecture Blueprint](../foundation/architecture.md) — Core layering and dependency rules.
