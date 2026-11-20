package model

import "time"

// OperationalCapability describes the telemetry and runtime operational features
// supported by a workload target.
type OperationalCapability struct {
	Kind                 Kind          `json:"kind"`
	LivenessProbe        string        `json:"liveness_probe"`        // e.g. "/healthz", "process-exit", "none"
	ReadinessProbe       string        `json:"readiness_probe"`       // e.g. "/readyz", "broker-connected", "none"
	LoggingFormat        string        `json:"logging_format"`        // "Structured JSON (slog) / Text"
	CorrelationSupported bool          `json:"correlation_supported"` // supports X-Request-ID / trace_id context
	MetricsExport        string        `json:"metrics_export"`        // "Prometheus /metrics", "N/A"
	TracingExport        string        `json:"tracing_export"`        // "OpenTelemetry (OTel) / W3C TraceContext", "N/A"
	GracefulShutdown     bool          `json:"graceful_shutdown"`     // handles SIGINT/SIGTERM with drain
	ShutdownTimeout      time.Duration `json:"shutdown_timeout"`      // default drain timeout
	DeploymentTarget     string        `json:"deployment_target"`     // "Docker Compose / Systemd / Single Binary"
	RollbackSupport      string        `json:"rollback_support"`      // "Instant binary / image swap, schema rollback"
	BackupRequirement    string        `json:"backup_requirement"`    // "SQLite / Postgres snapshot", "Stateless"
}

// CanonicalWorkloadOperationalBaselines returns the objective operational capability matrix
// for all eight canonical workload kinds in the Krewire ecosystem.
func CanonicalWorkloadOperationalBaselines() map[Kind]OperationalCapability {
	return map[Kind]OperationalCapability{
		KindApp: {
			Kind:                 KindApp,
			LivenessProbe:        "/healthz (HTTP 200 process alive)",
			ReadinessProbe:       "/readyz (HTTP 200/503 downstream DB/cache verified)",
			LoggingFormat:        "Structured JSON (slog) in prod, colored text in dev",
			CorrelationSupported: true,
			MetricsExport:        "Prometheus /metrics endpoint via pkg/web/metrics",
			TracingExport:        "W3C TraceContext / OTel HTTP client propagation",
			GracefulShutdown:     true,
			ShutdownTimeout:      10 * time.Second,
			DeploymentTarget:     "Docker Compose / Systemd / Bare-Metal Linux VPS",
			RollbackSupport:      "Instant image rollback (docker compose up -d) + kiw migrate rollback",
			BackupRequirement:    "Stateful: automated daily SQL snapshots + WAL archiving",
		},
		KindService: {
			Kind:                 KindService,
			LivenessProbe:        "/healthz (HTTP 200 process alive)",
			ReadinessProbe:       "/readyz (HTTP 200/503 downstream dependencies verified)",
			LoggingFormat:        "Structured JSON (slog) with trace_id attribute",
			CorrelationSupported: true,
			MetricsExport:        "Prometheus /metrics endpoint via pkg/web/metrics",
			TracingExport:        "W3C TraceContext / OTel spans across service-mesh calls",
			GracefulShutdown:     true,
			ShutdownTimeout:      10 * time.Second,
			DeploymentTarget:     "Docker Compose / K8s / Nomad / Systemd",
			RollbackSupport:      "Instant container rollback + traffic routing swap",
			BackupRequirement:    "Depends on backing store: snapshot database or stateless",
		},
		KindGateway: {
			Kind:                 KindGateway,
			LivenessProbe:        "/healthz (HTTP 200 proxy alive)",
			ReadinessProbe:       "/readyz (HTTP 200/503 upstream targets reachable)",
			LoggingFormat:        "Structured access log JSON (client IP, status, latency)",
			CorrelationSupported: true,
			MetricsExport:        "Prometheus request count, duration histogram, and 5xx rate",
			TracingExport:        "Injects & propagates X-Request-ID and traceparent headers",
			GracefulShutdown:     true,
			ShutdownTimeout:      15 * time.Second,
			DeploymentTarget:     "Edge Nginx / Host Port 80/443 Reverse Proxy",
			RollbackSupport:      "Immediate Nginx reload (nginx -s reload) to prior config",
			BackupRequirement:    "Stateless: versioned gateway route configuration",
		},
		KindWorker: {
			Kind:                 KindWorker,
			LivenessProbe:        "Process heartbeat file or internal watchdog",
			ReadinessProbe:       "Broker connectivity check (Redis PING / AMQP status)",
			LoggingFormat:        "Structured JSON with job_id and queue_name attributes",
			CorrelationSupported: true,
			MetricsExport:        "Prometheus processed job counter & latency summary",
			TracingExport:        "Job payload metadata correlation ID",
			GracefulShutdown:     true,
			ShutdownTimeout:      30 * time.Second,
			DeploymentTarget:     "Systemd service / Docker container without ingress port",
			RollbackSupport:      "Stop worker, drain inflight message queue, deploy previous binary",
			BackupRequirement:    "Message broker persistence (AOF/RDB) & dead-letter queue",
		},
		KindSite: {
			Kind:                 KindSite,
			LivenessProbe:        "HTTP 200 on / or index.html",
			ReadinessProbe:       "HTTP 200 on / or index.html",
			LoggingFormat:        "Web server edge access logs (Cloudflare / Nginx)",
			CorrelationSupported: false,
			MetricsExport:        "Edge CDN analytics / Nginx stub_status",
			TracingExport:        "N/A — static pre-rendered content",
			GracefulShutdown:     false,
			ShutdownTimeout:      0,
			DeploymentTarget:     "Cloudflare Pages / GitHub Pages / Alpine Nginx container",
			RollbackSupport:      "Instant git commit revert / edge CDN deployment rollback",
			BackupRequirement:    "Stateless: full static assets versioned in Git repo",
		},
		KindBook: {
			Kind:                 KindBook,
			LivenessProbe:        "HTTP 200 on /index.html",
			ReadinessProbe:       "HTTP 200 on /index.html",
			LoggingFormat:        "Web server edge access logs",
			CorrelationSupported: false,
			MetricsExport:        "Edge CDN analytics",
			TracingExport:        "N/A — static documentation content",
			GracefulShutdown:     false,
			ShutdownTimeout:      0,
			DeploymentTarget:     "Static web hosting / Git-backed documentation portal",
			RollbackSupport:      "Instant static asset switch / CDN cache purge",
			BackupRequirement:    "Stateless: markdown source committed in Git",
		},
		KindInfra: {
			Kind:                 KindInfra,
			LivenessProbe:        "docker compose ps / container health check exit code",
			ReadinessProbe:       "All dependent service health conditions met",
			LoggingFormat:        "Docker daemon json-file / journald",
			CorrelationSupported: false,
			MetricsExport:        "cAdvisor / Node Exporter container resource metrics",
			TracingExport:        "N/A — infrastructure orchestration definitions",
			GracefulShutdown:     true,
			ShutdownTimeout:      30 * time.Second,
			DeploymentTarget:     "Docker Engine / Podman / Systemd",
			RollbackSupport:      "docker compose down && docker compose -f previous.yaml up -d",
			BackupRequirement:    "Docker volume snapshots & environment file encryption",
		},
		KindCLI: {
			Kind:                 KindCLI,
			LivenessProbe:        "Exit code 0 upon execution completion",
			ReadinessProbe:       "Prerequisite toolchain detection (kiw doctor)",
			LoggingFormat:        "Standard streams (stdout for results, stderr for diagnostics)",
			CorrelationSupported: false,
			MetricsExport:        "Local execution elapsed time and exit code",
			TracingExport:        "N/A — ephemeral CLI execution",
			GracefulShutdown:     true,
			ShutdownTimeout:      2 * time.Second,
			DeploymentTarget:     "Standalone CGO-free static binary in PATH (/usr/local/bin)",
			RollbackSupport:      "Re-download or symlink prior binary version",
			BackupRequirement:    "Stateless local utility",
		},
	}
}

// GetWorkloadOperationalBaseline returns the operational baseline capability for kind.
func GetWorkloadOperationalBaseline(kind Kind) (OperationalCapability, bool) {
	baselines := CanonicalWorkloadOperationalBaselines()
	b, ok := baselines[kind]
	return b, ok
}
