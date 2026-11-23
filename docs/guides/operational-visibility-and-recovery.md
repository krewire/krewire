# Production Operational Visibility & Recovery Baseline

This guide establishes the canonical operational baseline, telemetry standards, deployment topologies, and disaster recovery procedures across the Krewire digital SDLC ecosystem per the **Three Pillars (3S)**:
- **Secure (S1)**: Zero exposed administrative ports, least-privilege containers, and unprivileged loopback bindings.
- **Sustain (S2)**: 10–30 MB lean RAM footprint, sub-millisecond cold starts, and predictable operations on a $5/month VPS.
- **Scalable (S3)**: Progressive operational evolution from single-binary local services to distributed container meshes without rewrite.

---

## 1. Workload Operational Capability Matrix

Every canonical workload in Krewire (`pkg/model.AllKinds`) adheres to an explicit, evidence-backed operational contract:

| Workload Kind | Liveness Probe | Readiness Probe | Structured Logs | Telemetry Correlation | Metrics Export | Graceful Drain | Default Deployment | Backup & Recovery |
|---|---|---|---|---|---|---|---|---|
| **App** (`app`) | `GET /healthz` (200 OK) | `GET /readyz` (200/503 downstream checks) | JSON (`slog`) | `request_id`, `trace_id` | Prometheus `/metrics` | Yes (10s LIFO) | Docker Compose / Linux VPS | Stateful snapshot + WAL |
| **Service** (`service`) | `GET /healthz` (200 OK) | `GET /readyz` (200/503 downstream checks) | JSON (`slog`) | `request_id`, `trace_id` | Prometheus `/metrics` | Yes (10s LIFO) | Docker Compose / Nomad / K8s | Database snapshot or stateless |
| **Gateway** (`gateway`) | `GET /healthz` (200 OK) | `GET /readyz` (Upstreams reachable) | JSON access logs | `X-Request-ID`, `traceparent` | Request count, duration, 5xx rate | Yes (15s LIFO) | Host Nginx / Edge Proxy | Stateless route configuration |
| **Worker** (`worker`) | Internal heartbeat watchdog | Broker connectivity (PING) | JSON (`slog`) | `job_id`, `queue_name` | Processed jobs, latency summary | Yes (30s LIFO) | Systemd / Container (no port) | Broker persistence (AOF/RDB) |
| **Site** (`site`) | `GET /` (200 OK) | `GET /` (200 OK) | Edge access logs | N/A | Edge CDN request analytics | Instant | Cloudflare Pages / Alpine Nginx | Stateless (Git SSOT) |
| **Book** (`book`) | `GET /index.html` (200 OK) | `GET /index.html` (200 OK) | Edge access logs | N/A | Edge CDN request analytics | Instant | Git-backed documentation host | Stateless (Git SSOT) |
| **Infra** (`infra`) | Container health check exit 0 | Dependent health conditions met | JSON (`journald`) | N/A | cAdvisor / Node Exporter | Yes (30s LIFO) | Docker Engine / Compose | Volume snapshots + env vault |
| **CLI** (`cli`) | Exit code 0 on completion | Toolchain check (`krew doctor`) | Stdout / Stderr | N/A | Execution duration & status | Yes (2s drain) | Pure-Go static binary in PATH | Stateless |

---

## 2. Distinct Health vs. Readiness Semantics

Production systems must never conflate process survival with service availability:

### Liveness Probe (`/healthz`)
- **Semantic Purpose**: "Is the process execution loop intact, or is it deadlocked / crashed?"
- **Behavior**: Evaluates process memory and internal scheduler responsiveness. Returns HTTP `200 OK` (`{"status":"ok"}`) immediately.
- **Orchestrator Action**: If `/healthz` fails (e.g., 3 consecutive timeouts), the orchestrator (Docker, Systemd, or Kubernetes) forcefully terminates and restarts the container or process.

### Readiness Probe (`/readyz`)
- **Semantic Purpose**: "Is the application currently capable of fulfilling user requests, or are downstream dependencies degraded?"
- **Behavior**: Executes registered dependency checks (`web.WithCheck("db", checkDB)`) concurrently with a bounded 5-second timeout:
  - When all checks pass: returns HTTP `200 OK` (`{"status":"ok","checks":{"db":"ok"}}`).
  - When any check fails: returns HTTP `503 Service Unavailable` (`{"status":"degraded","checks":{"db":"connection refused"}}`).
- **Orchestrator Action**: If `/readyz` fails, the reverse proxy or load balancer immediately removes the instance from the routing pool. The process is **never killed**, allowing it to reconnect to the database or recover gracefully without cascading restart loops.

### Startup Probe (`/startupz`) and the shared contract
- **Semantic purpose**: "Has initialization finished?" Until it passes, orchestrators ignore liveness and readiness verdicts, so slow migrations or cache warm-up never trigger restart loops.
- The cross-workload contract for all three probes, including the shared `Report` JSON, optional (`warn`) checks, the `starting` and `draining` states, and worker heartbeats, is [`pkg/observe`](../pkg/observe.md). Its `/healthz` and `/readyz` paths match `web.Health`.

---

## 3. Structured Logging & Telemetry Correlation

### Telemetry Architecture & Boundaries
Krewire draws a strict architectural boundary between built-in runtime diagnostics and external telemetry platforms:
- **Built-in Telemetry**: Standard library Go `slog` with JSON output in production, `/healthz` and `/readyz` probe endpoints, and constant-time memory footprints.
- **External Telemetry Integrations**: Standard Prometheus scraping (`pkg/web/metrics`), OpenTelemetry (OTel) context propagation, and centralized ingestion (Grafana, Loki, Datadog) consume standard HTTP/JSON contracts without proprietary agent runtimes.

### Correlation ID Contract
All state mutations, incoming HTTP requests, and background jobs are tagged with correlation identifiers:
- `request_id`: Extracted from incoming `X-Request-ID` header, or generated as a cryptographically random UUIDv4.
- `trace_id`: Extracted from standard W3C `traceparent` headers (`00-<trace_id>-<span_id>-<flags>`).

```go
// Inject correlation into context
ctx := log.WithCorrelation(r.Context(), requestID, traceID)

// Decorate logger with structured correlation fields
logger := log.WithContext(slog.Default(), ctx)
logger.InfoContext(ctx, "payment processed", "amount_cents", 4500)
```

Example JSON log record emitted to `stderr`:
```json
{
  "time": "2026-11-21T07:05:00.123Z",
  "level": "INFO",
  "msg": "payment processed",
  "request_id": "req-4fc69929-04a7",
  "trace_id": "trc-a77b6ed4e6a4",
  "amount_cents": 4500,
  "source": "go"
}
```

---

## 4. Graceful Shutdown & Interruption Lifecycle

Long-running workloads (`app`, `service`, `gateway`, `worker`) manage process termination via `pkg/lifecycle.Supervisor`:

1. **Signal Interception**: Traps `SIGINT` (`Ctrl+C`) and `SIGTERM` (Docker / systemd stop).
2. **Reverse Dependency Drain (LIFO)**:
   - Ingress listeners stop accepting new TCP connections.
   - Active in-flight HTTP requests or queue messages are allowed to complete.
   - Background asynchronous workers finish current jobs.
   - Database connection pools, cache clients, and file handles close last.
3. **Bounded Timeout**:
   - The shutdown timeout (`Supervisor.WithShutdownTimeout(10*time.Second)`) prevents hanging processes. If in-flight work exceeds the deadline, context cancellation forces immediate termination.

---

## 5. Production Deployment Hardening

Production workloads deploy using standalone static binaries or containerized Docker Compose stacks:

### Docker Compose Reference Deployment (`infra`)
```yaml
version: "3.8"

services:
  app:
    image: myorg/myapp:v1.2.0
    restart: unless-stopped
    user: "1000:1000"
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=64m
    ports:
      - "127.0.0.1:8090:8080" # Bound strictly to loopback
    environment:
      - ENV=production
      - DB_DSN=${env:DB_DSN}
      - SECRET_KEY=${env:SECRET_KEY}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://127.0.0.1:8080/healthz"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s
    volumes:
      - ./data:/app/data:rw
```

### Invariant Network Topologies
- **Loopback Binding**: Application containers bind only to `127.0.0.1` on the host, never `0.0.0.0`.
- **Edge SSL Termination**: Public HTTPS (port 443) terminates at the Cloudflare Edge or Host Nginx reverse proxy. Origin servers do not manage ACME/SSL certificates locally.

---

## 6. Stateful Workloads: Backup & Restore

Workloads utilizing embedded or managed persistence require tested disaster recovery procedures:

### SQLite Snapshotting & WAL Checkpointing
For embedded SQLite databases:
```bash
# 1. Perform online zero-locking backup using SQLite online backup API
sqlite3 /app/data/app.db ".backup '/app/data/backups/backup-$(date +%Y%m%d%H%M%S).db'"

# 2. Checkpoint and truncate the Write-Ahead Log (WAL)
sqlite3 /app/data/app.db "PRAGMA wal_checkpoint(TRUNCATE);"
```

### PostgreSQL Dumps & Point-in-Time Recovery
For external PostgreSQL databases:
```bash
# Take compressed custom-format logical dump
pg_dump -Fc -h "$DB_HOST" -U "$DB_USER" "$DB_NAME" > "/backups/pg-$(date +%Y%m%d%H%M%S).dump"

# Restore database into fresh instance
pg_restore -C -d postgres -h "$NEW_DB_HOST" -U "$DB_USER" "/backups/pg-snapshot.dump"
```

---

## 7. Rollback & Incident Recovery Runbooks

When an incident or regression occurs, operational recovery must be deterministic:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   Incident Response Decision Matrix                    │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
               ┌────────────────────┴────────────────────┐
               ▼                                         ▼
      [ Application Bug ]                      [ Corrupted Schema ]
• Stateless container regression           • Database migration failure
• Action: Instant Image Rollback           • Action: Down-Migration Rollback
  1. Update image tag in compose             1. Run 'kiw migrate rollback'
  2. docker compose up -d                    2. Or restore database snapshot
  3. Verify /healthz and /readyz             3. Redeploy matching app version
```

### Procedure 1: Instant Image Rollback (< 2 seconds)
```bash
cd /opt/myapp
# Revert to known stable release tag
sed -i 's/:v1.2.1/:v1.2.0/g' docker-compose.yaml
docker compose up -d
curl -s http://127.0.0.1:8090/healthz
```

### Procedure 2: Database Schema Rollback
```bash
# Roll back the most recent batch of database migrations
kiw migrate rollback

# Verify migration integrity
kiw migrate status
```

---

## 8. Self-Hosting Prerequisites & Boundaries

Krewire is engineered for sustainable self-hosting on minimal commodity infrastructure:

### Hardware & OS Prerequisites
- **Compute**: 1 vCPU, 512 MB – 1 GB RAM (Krewire idle consumption is 10–30 MB).
- **Disk**: 10 GB SSD/NVMe storage.
- **Operating System**: Linux (Ubuntu 22.04+, Debian 12+, Alpine Linux 3.19+), pure-Go with `CGO_ENABLED=0`.
- **Target Cost**: Predictably deployable on a $5/month VPS (Hetzner, DigitalOcean, Linode).

### Unsupported Architectures & Provider Combinations
To maintain simplicity and zero lock-in, the following architectures are explicitly outside Krewire's core design scope:
- **Distributed ACID Transactions across Multi-Cloud**: Avoid distributed two-phase commits across heterogeneous providers; prefer decoupled event queues.
- **Ephemeral Serverless Without Local Disk**: Serverless runtimes that forbid disk caching or background goroutine drains are incompatible with stateful workers.
- **Proprietary Kernel Extensions**: Custom kernel modules or CGO-dependent drivers violating pure-Go portability.

---

## 9. Diagnosis Runbooks (`pkg/observe`)

Each runbook starts from the same evidence: the diagnostics bundle (`krew doctor --json`, or `GET /debug/diagnostics` on the admin listener), the probe `Report`s, and logs filtered by `trace_id`. Bundle findings carry a stable `code` and a `trace_id`, and their hints link to the sections below. See [`pkg/observe`](../pkg/observe.md) for the full contract.

### Diagnosing startup failure
1. **Symptom:** `/startupz` returns `503` with `"reason":"starting"`, or the bundle reports `health.startup.incomplete` or `health.startup.<check>`.
2. Read the failing startup checks in the report. Their errors are redacted but keep the host, port, and cause.
3. Filter logs by the bundle's `trace_id` and by `workload`/`resource`. The last record before the stall names the step that failed (config load, migration, dependency dial).
4. Check the `config` section of the bundle for a missing or wrong (redacted) setting.
5. **Fix and verify:** after a successful init the process calls `Health.MarkStarted()`, and `/startupz` returns `200`.

### Diagnosing readiness failure
1. **Symptom:** `/readyz` returns `503` while `/healthz` returns `200`. The load balancer has removed the instance, but nothing restarts it.
2. Each failing required check in the report has a `health.readiness.<check>` finding. Optional checks show `warn` and leave the instance serving.
3. Correlate in metrics. Look at `krewire_health_check_status{probe="readiness"}` over time, `krewire_resilience_breaker_state` (a value of 2 means the breaker is open), and the 5xx rate in `krewire_http_server_requests_total` by `status_class`.
4. Readiness returns on its own when the dependency recovers. If `/healthz` also fails (for example a stale worker `Heartbeat`), the orchestrator restarts the process. Check `krewire_go_goroutines` and `krewire_go_heap_alloc_bytes` for exhaustion.

### Diagnosing retries
1. **Signals:** `krewire_resilience_retries_total{operation}`, `krewire_worker_jobs_total{outcome="retry"|"dead_letter"}`, and the bundle findings `resilience.retries`, `worker.dead_letters`, and `resilience.breaker_open`.
2. Every retry attempt runs inside the job's trace. Use `observe.WrapJob` at enqueue and filter logs by the `trace_id` to see each attempt's error.
3. Classify by `failure_class`. Provider failures (`provider`) call for backoff or a breaker. Workload failures (`workload`) usually call for a code or data fix. Inspect the DLQ for poison messages before requeueing.

### Diagnosing shutdown
1. **Expected sequence:** `Health.MarkDraining()`, readiness fails with `"reason":"draining"`, the load balancer drains, the `lifecycle.Supervisor` LIFO drain runs within the profile's `DrainTimeout`, and the process exits.
2. A `health.readiness.draining` finding that persists, or a process killed at the timeout, means in-flight work outlived the drain. Check `krewire_http_server_in_flight_requests` and `krewire_worker_job_duration_seconds` against the timeout.
3. CLI and infra workloads report through their exit code. A non-zero exit together with the bundle's `findings` is the evidence that AI workflow reports cite, not a narrative summary.
