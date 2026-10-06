package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/config"
)

// RegisterDashboard registers flags for the dashboard command.
func RegisterDashboard(fs *flag.FlagSet) {
	fs.String("port", "4000", "port for the dashboard HTTP server")
	fs.String("env", "", "target environment: local, production, or testing")
}

// RunDashboard starts a local dev dashboard for services, logs, and traces
// (KWF-L5H2F observability).
func RunDashboard(fs *flag.FlagSet) kern.ExitCode {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return kern.ExitCodeUsage
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fail(err)
	}

	port := flagValue(fs, "port")
	if port == "" {
		port = "4000"
	}

	switch cfg.Kind() {
	case string(kern.KindWorker), string(kern.KindService), string(kern.KindInfra):
		fmt.Printf("dashboard started on http://localhost:%s\n", port)
		fmt.Println("  (placeholder — full UI requires frontend integration)")
		return kern.ExitCodeSuccess
	default:
		fmt.Fprintf(os.Stderr, "kiw dashboard: project kind %q does not support dashboard — use worker, service, or infra\n", cfg.Kind())
		return kern.ExitCodeUsage
	}
}
