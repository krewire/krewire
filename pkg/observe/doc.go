// Package observe is the transport-neutral observability contract shared by
// every Krewire workload (app, service, gateway, worker, site, book, infra,
// cli) and by the unified toolchain that drives them.
//
// It provides, using only the standard library and pkg/log:
//
//   - W3C trace-context propagation (traceparent, tracestate, baggage) across
//     HTTP, gateway, worker enqueue/dequeue, and child-process boundaries.
//   - Standard log fields, correlation IDs (request, session, task, agent run,
//     tool call, process), a redacting slog.Handler, and redaction helpers.
//   - Counters, gauges, and histograms with stable names, explicit units, and
//     bounded label cardinality, exposed in Prometheus text format and as a
//     JSON snapshot that OpenTelemetry adapters can translate.
//   - Liveness, readiness, and startup probes with one JSON report model.
//   - A diagnostics bundle (a "doctor" report) for humans, CI, and AI
//     workflows, redacted before it leaves the process.
//   - Per-workload defaults (probe paths, drain timeouts, standard metrics).
//
// Everything is opt-in: a nil *Registry, *Counter, *Gauge, *Histogram, or
// *Health is a valid no-op, so exporters can be disabled without touching
// business logic. See docs/pkg/observe.md for the full contract.
package observe
