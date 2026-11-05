package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
	"github.com/krewire/krewire/tools/kiw/internal/shape"
)

const (
	// DefaultDashboardPort is the default HTTP listen port for the developer dashboard.
	DefaultDashboardPort = "4000"

	// DefaultDashboardHost is the default host address binding for the developer dashboard.
	DefaultDashboardHost = "127.0.0.1"

	// DefaultDashboardShutdownTimeout is the maximum duration allocated for server drainage.
	DefaultDashboardShutdownTimeout = 3 * time.Second

	// DefaultDashboardPulseInterval is the SSE telemetry push cadence.
	DefaultDashboardPulseInterval = 2 * time.Second
)

// RegisterDashboard registers flags for the dashboard command.
func RegisterDashboard(fs *flag.FlagSet) {
	fs.String("port", DefaultDashboardPort, "port for the dashboard HTTP server")
	fs.String("host", DefaultDashboardHost, "host address to bind dashboard HTTP server")
	registerRuntimeFlags(fs)
}

// RunDashboard starts a local developer dashboard for services, routes, logs,
// and runtime observability (KWN-DX-003, KWF-L5H2F).
func RunDashboard(fs *flag.FlagSet) errs.ExitCode {
	rt, code := bootRuntime(fs)
	if code != errs.ExitCodeSuccess {
		return code
	}

	explicit := firstNonEmpty(flagValue(fs, "kind"), rt.cfg.Kind())
	res, err := shape.Detect(rt.root, explicit)
	var kind string
	if err == nil && res.Kind != shape.KindNone {
		kind = string(res.Kind)
	} else if rt.cfg.Kind() != "" {
		kind = rt.cfg.Kind()
	} else {
		kind = "app"
	}

	port := flagValue(fs, "port")
	if port == "" {
		port = DefaultDashboardPort
	}
	host := flagValue(fs, "host")
	if host == "" {
		host = DefaultDashboardHost
	}

	srv := newDashboardServer(rt.root, rt.cfg, kind, string(rt.env), host, port)
	if err := srv.Start(); err != nil {
		return fail(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultDashboardShutdownTimeout)
		defer cancel()
		_ = srv.Stop(ctx)
	}()

	tm := term.NewTerminal()
	boldCyan := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }

	addr := srv.Addr()
	info := srv.collectInfo()

	fmt.Printf("\n  %s\n", boldCyan("Krewire Dev Dashboard"))
	fmt.Printf("  %s %s (kind: %s, env: %s)\n", dim("Project:    "), info.Name, info.Kind, info.Env)
	fmt.Printf("  %s http://%s\n", dim("Dashboard:  "), green(addr))
	fmt.Printf("  %s http://%s/api/health\n", dim("Health API: "), addr)
	fmt.Printf("  %s http://%s/api/metrics\n", dim("Metrics API:"), addr)
	fmt.Printf("  %s http://%s/mcp/sse\n\n", dim("MCP Server: "), addr)
	fmt.Printf("  %s\n\n", dim("Press Ctrl+C to stop."))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down dev dashboard...")
	return errs.ExitCodeSuccess
}
