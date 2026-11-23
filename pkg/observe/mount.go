package observe

import "net/http"

// Mount wires the observability endpoints a profile calls for onto mux:
// probes for ModeHTTP and ModeHeartbeat, the metrics endpoint when r is
// non-nil, and the diagnostics bundle when d is non-nil. Mount diagnostics
// only on an admin or loopback mux.
func Mount(mux *http.ServeMux, p Profile, h *Health, r *Registry, d *Diagnostics) {
	if p.Mode == ModeHTTP || p.Mode == ModeHeartbeat {
		h.Mount(mux, p.Probes)
	}
	if r != nil && p.MetricsPath != "" {
		mux.Handle("GET "+p.MetricsPath, MetricsHandler(r))
	}
	if d != nil {
		mux.Handle("GET "+PathDiagnostics, d.Handler())
	}
}
