package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/model"
	"github.com/krewire/krewire/pkg/term"
	"github.com/krewire/krewire/tools/kiw/internal/buildinfo"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	"github.com/krewire/krewire/tools/kiw/internal/mcp"
	"github.com/krewire/krewire/tools/kiw/internal/shape"
)

// RegisterDashboard registers flags for the dashboard command.
func RegisterDashboard(fs *flag.FlagSet) {
	fs.String("port", "4000", "port for the dashboard HTTP server")
	fs.String("host", "127.0.0.1", "host address to bind dashboard HTTP server")
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
		port = "4000"
	}
	host := flagValue(fs, "host")
	if host == "" {
		host = "127.0.0.1"
	}

	srv := newDashboardServer(rt.root, rt.cfg, kind, string(rt.env), host, port)
	if err := srv.Start(); err != nil {
		return fail(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

// ProjectRoute represents an HTTP page, content article, or handler detected in the project.
type ProjectRoute struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "page", "content", "api"
	File string `json:"file"`
}

// ProjectService represents a configured or running microservice in the project.
type ProjectService struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// ProjectInfo provides full metadata about the project, its shape, and environment.
type ProjectInfo struct {
	Name      string           `json:"name"`
	Kind      string           `json:"kind"`
	Env       string           `json:"env"`
	Root      string           `json:"root"`
	Version   string           `json:"version"`
	GoVersion string           `json:"go_version"`
	Platform  string           `json:"platform"`
	Port      string           `json:"port"`
	Services  []ProjectService `json:"services"`
	Workers   []string         `json:"workers"`
	Routes    []ProjectRoute   `json:"routes"`
	Config    map[string]any   `json:"config"`
}

// RuntimeMetrics encapsulates real-time memory and goroutine telemetry.
type RuntimeMetrics struct {
	Goroutines    int     `json:"goroutines"`
	AllocMB       float64 `json:"alloc_mb"`
	TotalAllocMB  float64 `json:"total_alloc_mb"`
	SysMB         float64 `json:"sys_mb"`
	NumGC         uint32  `json:"num_gc"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	Timestamp     string  `json:"timestamp"`
}

// HealthResponse holds the standardized health check contract.
type HealthResponse struct {
	Status        string  `json:"status"`
	Kind          string  `json:"kind"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	Goroutines    int     `json:"goroutines"`
	MemoryMB      float64 `json:"memory_mb"`
	Timestamp     string  `json:"timestamp"`
}

type dashboardServer struct {
	root      string
	cfg       *config.Config
	kind      string
	env       string
	host      string
	port      string
	startTime time.Time
	listener  net.Listener
	server    *http.Server
}

func newDashboardServer(root string, cfg *config.Config, kind, env, host, port string) *dashboardServer {
	return &dashboardServer{
		root:      root,
		cfg:       cfg,
		kind:      kind,
		env:       env,
		host:      host,
		port:      port,
		startTime: time.Now(),
	}
}

func (d *dashboardServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleHome)
	mux.HandleFunc("/api/info", d.handleInfo)
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	mux.HandleFunc("/api/health", d.handleHealth)
	mux.HandleFunc("/api/events", d.handleEvents)

	mcpSrv := mcp.NewServer(d.root, d.cfg)
	mcpTransport := mcp.NewHTTPTransport(mcpSrv)
	mcpTransport.Mount(mux, "/mcp")

	return mux
}

func (d *dashboardServer) Start() error {
	addr := net.JoinHostPort(d.host, d.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("dashboard listen on %s: %w", addr, err)
	}
	d.listener = ln
	d.server = &http.Server{
		Handler: d.Handler(),
	}
	go func() {
		_ = d.server.Serve(ln)
	}()
	return nil
}

func (d *dashboardServer) Addr() string {
	if d.listener != nil {
		return d.listener.Addr().String()
	}
	return net.JoinHostPort(d.host, d.port)
}

func (d *dashboardServer) Stop(ctx context.Context) error {
	if d.server != nil {
		return d.server.Shutdown(ctx)
	}
	return nil
}

func (d *dashboardServer) collectMetrics() RuntimeMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := int64(time.Since(d.startTime).Seconds())
	return RuntimeMetrics{
		Goroutines:    runtime.NumGoroutine(),
		AllocMB:       float64(m.Alloc) / 1024 / 1024,
		TotalAllocMB:  float64(m.TotalAlloc) / 1024 / 1024,
		SysMB:         float64(m.Sys) / 1024 / 1024,
		NumGC:         m.NumGC,
		UptimeSeconds: uptime,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
}

func (d *dashboardServer) collectInfo() ProjectInfo {
	name := ""
	if d.cfg != nil {
		name = d.cfg.Title
	}
	if name == "" {
		name = filepath.Base(d.root)
	}
	routes := discoverRoutes(d.root)
	svcs := discoverServices(d.cfg)
	workers := discoverWorkers(d.cfg)

	cfgMap := map[string]any{
		"kind": d.kind,
		"env":  d.env,
	}
	if d.cfg != nil {
		cfgMap["title"] = d.cfg.Title
		cfgMap["base"] = d.cfg.Base
		cfgMap["output"] = d.cfg.Output
		cfgMap["version"] = d.cfg.Version
	}

	return ProjectInfo{
		Name:      name,
		Kind:      d.kind,
		Env:       d.env,
		Root:      d.root,
		Version:   qualifiedVersion(buildinfo.ModKrewire),
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		Port:      d.port,
		Services:  svcs,
		Workers:   workers,
		Routes:    routes,
		Config:    cfgMap,
	}
}

func (d *dashboardServer) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	info := d.collectInfo()
	initialJSON, err := json.Marshal(info)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		http.Error(w, "Template Render Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, map[string]any{
		"Title":       info.Name,
		"InitialJSON": template.JS(initialJSON),
	})
}

func (d *dashboardServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := d.collectInfo()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(info)
}

func (d *dashboardServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := d.collectMetrics()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(metrics)
}

func (d *dashboardServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	metrics := d.collectMetrics()
	res := HealthResponse{
		Status:        "ok",
		Kind:          d.kind,
		UptimeSeconds: metrics.UptimeSeconds,
		Goroutines:    metrics.Goroutines,
		MemoryMB:      metrics.AllocMB,
		Timestamp:     metrics.Timestamp,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (d *dashboardServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial pulse immediately
	initialMetrics := d.collectMetrics()
	if b, err := json.Marshal(initialMetrics); err == nil {
		fmt.Fprintf(w, "event: pulse\ndata: %s\n\n", b)
		flusher.Flush()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			m := d.collectMetrics()
			b, err := json.Marshal(m)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: pulse\ndata: %s\n\n", b); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func discoverRoutes(root string) []ProjectRoute {
	var routes []ProjectRoute
	seen := make(map[string]bool)

	addRoute := func(p, kind, file string) {
		if !seen[p] {
			seen[p] = true
			routes = append(routes, ProjectRoute{Path: p, Kind: kind, File: file})
		}
	}

	// 1. Pages directory (.kiw, .html, .md)
	pagesDir := filepath.Join(root, "pages")
	if info, err := os.Stat(pagesDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(pagesDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".kiw" && ext != ".html" && ext != ".md" {
				return nil
			}
			rel, err := filepath.Rel(pagesDir, path)
			if err != nil {
				return nil
			}
			routePath := "/" + filepath.ToSlash(rel)
			routePath = strings.TrimSuffix(routePath, ext)
			if strings.HasSuffix(routePath, "/index") {
				routePath = strings.TrimSuffix(routePath, "index")
			}
			if routePath == "" {
				routePath = "/"
			}
			addRoute(routePath, "page", filepath.Join("pages", rel))
			return nil
		})
	}

	// 2. Content directory (.md, .kiw)
	for _, dirName := range []string{"content", "manuscript"} {
		dir := filepath.Join(root, dirName)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				ext := filepath.Ext(path)
				if ext != ".md" && ext != ".kiw" {
					return nil
				}
				rel, err := filepath.Rel(dir, path)
				if err != nil {
					return nil
				}
				routePath := "/" + filepath.ToSlash(rel)
				routePath = strings.TrimSuffix(routePath, ext)
				if strings.HasSuffix(routePath, "/index") {
					routePath = strings.TrimSuffix(routePath, "index")
				}
				if routePath == "" {
					routePath = "/"
				}
				addRoute(routePath, "content", filepath.Join(dirName, rel))
				return nil
			})
		}
	}

	// 3. Handlers directory (.go)
	for _, dirName := range []string{"handlers", filepath.Join("internal", "handlers"), "api"} {
		dir := filepath.Join(root, dirName)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return nil
				}
				rel, err := filepath.Rel(dir, path)
				if err != nil {
					return nil
				}
				base := strings.TrimSuffix(rel, ".go")
				routePath := "/api/" + filepath.ToSlash(base)
				addRoute(routePath, "api", filepath.Join(dirName, rel))
				return nil
			})
		}
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})
	return routes
}

func discoverServices(cfg *config.Config) []ProjectService {
	var svcs []ProjectService
	if cfg == nil {
		return svcs
	}
	for _, s := range cfg.Services {
		svcs = append(svcs, ProjectService{
			Name:     s.Name,
			Port:     s.Port,
			Protocol: "http",
		})
	}
	if cfg.Runtime.Port > 0 {
		svcs = append(svcs, ProjectService{
			Name:     "runtime",
			Port:     cfg.Runtime.Port,
			Protocol: "http",
		})
	}
	if cfg.Dev.Port > 0 {
		svcs = append(svcs, ProjectService{
			Name:     "dev",
			Port:     cfg.Dev.Port,
			Protocol: "http",
		})
	}
	if cfg.Server.Port > 0 {
		svcs = append(svcs, ProjectService{
			Name:     "server",
			Port:     cfg.Server.Port,
			Protocol: "http",
		})
	}
	return svcs
}

func discoverWorkers(cfg *config.Config) []string {
	var workers []string
	if cfg == nil {
		return workers
	}
	if cfg.Kind() == string(model.KindWorker) {
		workers = append(workers, "default (worker kind)")
	}
	return workers
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}} — Krewire Dev Dashboard</title>
  <style>
    :root {
      --bg: #090d16;
      --card-bg: #111827;
      --card-border: #1f2937;
      --card-hover: #1f293d;
      --text-main: #f3f4f6;
      --text-muted: #9ca3af;
      --text-dim: #6b7280;
      --accent-cyan: #06b6d4;
      --accent-emerald: #10b981;
      --accent-indigo: #818cf8;
      --accent-amber: #f59e0b;
      --accent-rose: #f43f5e;
      --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      --font-sans: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text-main);
      font-family: var(--font-sans);
      line-height: 1.5;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }
    header {
      background: rgba(17, 24, 39, 0.85);
      backdrop-filter: blur(8px);
      border-bottom: 1px solid var(--card-border);
      position: sticky;
      top: 0;
      z-index: 50;
      padding: 0.85rem 1.5rem;
    }
    .header-inner {
      max-width: 1280px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      text-decoration: none;
      color: inherit;
    }
    .logo-badge {
      background: linear-gradient(135deg, #06b6d4, #3b82f6);
      color: #fff;
      font-weight: 900;
      font-size: 0.85rem;
      padding: 0.25rem 0.55rem;
      border-radius: 6px;
      letter-spacing: 0.05em;
    }
    .brand-title {
      font-weight: 700;
      font-size: 1.05rem;
      letter-spacing: -0.01em;
    }
    .brand-sub {
      color: var(--accent-cyan);
      font-size: 0.75rem;
      font-family: var(--font-mono);
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      margin-left: 0.25rem;
    }
    .header-pills {
      display: flex;
      align-items: center;
      gap: 0.6rem;
    }
    .pill {
      font-size: 0.75rem;
      font-family: var(--font-mono);
      padding: 0.2rem 0.6rem;
      border-radius: 9999px;
      border: 1px solid var(--card-border);
      background: #172033;
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      font-weight: 500;
    }
    .pulse-dot {
      width: 7px;
      height: 7px;
      border-radius: 50%;
      background: var(--accent-emerald);
      box-shadow: 0 0 8px var(--accent-emerald);
      animation: pulseAnim 2s infinite;
    }
    @keyframes pulseAnim {
      0% { transform: scale(0.95); opacity: 0.8; }
      50% { transform: scale(1.3); opacity: 1; }
      100% { transform: scale(0.95); opacity: 0.8; }
    }
    .container {
      max-width: 1280px;
      width: 100%;
      margin: 0 auto;
      padding: 1.75rem 1.5rem;
      flex: 1;
    }
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 1rem;
      margin-bottom: 2rem;
    }
    .stat-card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 10px;
      padding: 1.25rem;
      transition: border-color 0.15s, transform 0.15s;
    }
    .stat-card:hover {
      border-color: #374151;
      transform: translateY(-2px);
    }
    .stat-label {
      font-size: 0.75rem;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.05em;
      font-weight: 600;
      margin-bottom: 0.35rem;
    }
    .stat-value {
      font-size: 1.75rem;
      font-weight: 800;
      font-family: var(--font-mono);
      color: #fff;
    }
    .stat-desc {
      font-size: 0.72rem;
      color: var(--text-dim);
      margin-top: 0.3rem;
    }
    .tabs {
      display: flex;
      gap: 0.5rem;
      border-bottom: 1px solid var(--card-border);
      margin-bottom: 1.5rem;
    }
    .tab-btn {
      background: none;
      border: none;
      color: var(--text-muted);
      font-size: 0.88rem;
      font-weight: 600;
      padding: 0.65rem 1rem;
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: color 0.15s, border-color 0.15s;
    }
    .tab-btn:hover {
      color: var(--text-main);
    }
    .tab-btn.active {
      color: var(--accent-cyan);
      border-bottom-color: var(--accent-cyan);
    }
    .panel {
      display: none;
    }
    .panel.active {
      display: block;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 10px;
      padding: 1.5rem;
      margin-bottom: 1.5rem;
    }
    .card-title {
      font-size: 1.05rem;
      font-weight: 700;
      margin-bottom: 1rem;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.88rem;
    }
    th {
      text-align: left;
      padding: 0.75rem 1rem;
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-muted);
      border-bottom: 1px solid var(--card-border);
      background: #0f1523;
    }
    td {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #1a2233;
    }
    tr:last-child td {
      border-bottom: none;
    }
    tr:hover td {
      background: #141c2e;
    }
    .badge {
      display: inline-block;
      font-size: 0.7rem;
      font-family: var(--font-mono);
      font-weight: 600;
      padding: 0.15rem 0.5rem;
      border-radius: 4px;
      text-transform: uppercase;
    }
    .badge-page { background: rgba(6, 182, 212, 0.15); color: #38bdf8; border: 1px solid rgba(6, 182, 212, 0.3); }
    .badge-api { background: rgba(129, 140, 248, 0.15); color: #a5b4fc; border: 1px solid rgba(129, 140, 248, 0.3); }
    .badge-content { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
    .search-box {
      width: 100%;
      max-width: 320px;
      background: #0d121f;
      border: 1px solid var(--card-border);
      color: var(--text-main);
      padding: 0.45rem 0.85rem;
      border-radius: 6px;
      font-size: 0.85rem;
      outline: none;
    }
    .search-box:focus {
      border-color: var(--accent-cyan);
    }
    .log-feed {
      font-family: var(--font-mono);
      font-size: 0.8rem;
      background: #0a0e17;
      border: 1px solid var(--card-border);
      border-radius: 6px;
      padding: 1rem;
      max-height: 380px;
      overflow-y: auto;
      display: flex;
      flex-direction: column-reverse;
      gap: 0.4rem;
    }
    .log-entry {
      display: flex;
      gap: 0.85rem;
      padding: 0.25rem 0;
      border-bottom: 1px solid #141b2b;
    }
    .log-time { color: var(--text-dim); }
    .log-msg { color: #e2e8f0; }
    .kv-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 1rem;
    }
    .kv-item {
      background: #0d121f;
      padding: 0.85rem 1rem;
      border-radius: 6px;
      border: 1px solid #1a2233;
    }
    .kv-key {
      font-size: 0.72rem;
      color: var(--text-dim);
      text-transform: uppercase;
      font-weight: 600;
      letter-spacing: 0.05em;
    }
    .kv-val {
      font-family: var(--font-mono);
      font-size: 0.9rem;
      color: #f1f5f9;
      margin-top: 0.2rem;
      word-break: break-all;
    }
    footer {
      border-top: 1px solid var(--card-border);
      padding: 1.25rem;
      text-align: center;
      font-size: 0.75rem;
      color: var(--text-dim);
      font-family: var(--font-mono);
    }
    footer a { color: var(--accent-cyan); text-decoration: none; }
  </style>
</head>
<body>
  <header>
    <div class="header-inner">
      <div class="brand">
        <span class="logo-badge">KIW</span>
        <span class="brand-title" id="brand-project">{{.Title}}</span>
        <span class="brand-sub">Dashboard</span>
      </div>
      <div class="header-pills">
        <span class="pill" id="pill-kind">kind: <strong id="val-kind">-</strong></span>
        <span class="pill" id="pill-env">env: <strong id="val-env">-</strong></span>
        <span class="pill">
          <span class="pulse-dot" id="pulse-dot"></span>
          <span id="stream-status">LIVE STREAM</span>
        </span>
      </div>
    </div>
  </header>

  <main class="container">
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">Active Goroutines</div>
        <div class="stat-value" id="stat-goroutines">-</div>
        <div class="stat-desc">Concurrent workers & handlers</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Memory Alloc</div>
        <div class="stat-value" id="stat-alloc">-</div>
        <div class="stat-desc" id="stat-sys">Heap in active use</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">GC Cycles</div>
        <div class="stat-value" id="stat-gc">-</div>
        <div class="stat-desc">Completed garbage collections</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Dashboard Uptime</div>
        <div class="stat-value" id="stat-uptime">0s</div>
        <div class="stat-desc" id="stat-platform">-</div>
      </div>
    </div>

    <div class="tabs">
      <button class="tab-btn active" onclick="switchTab('routes')">Routes & Endpoints</button>
      <button class="tab-btn" onclick="switchTab('services')">Services & Workers</button>
      <button class="tab-btn" onclick="switchTab('overview')">Project & Environment</button>
      <button class="tab-btn" onclick="switchTab('logs')">Live Telemetry Feed</button>
    </div>

    <!-- TAB 1: Routes -->
    <div class="panel active" id="panel-routes">
      <div class="card">
        <div class="card-title">
          <span>Discovered Routes (<span id="route-count">0</span>)</span>
          <input type="text" class="search-box" id="route-filter" placeholder="Filter routes..." oninput="filterRoutes(this.value)">
        </div>
        <div style="overflow-x: auto;">
          <table id="table-routes">
            <thead>
              <tr>
                <th>Path</th>
                <th>Kind</th>
                <th>Source File</th>
              </tr>
            </thead>
            <tbody id="routes-body">
              <tr><td colspan="3" style="text-align:center; color: var(--text-dim);">No routes detected.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 2: Services & Workers -->
    <div class="panel" id="panel-services">
      <div class="card">
        <div class="card-title">Configured Services</div>
        <div style="overflow-x: auto;">
          <table>
            <thead>
              <tr>
                <th>Service Name</th>
                <th>Port</th>
                <th>Protocol</th>
              </tr>
            </thead>
            <tbody id="services-body">
              <tr><td colspan="3" style="text-align:center; color: var(--text-dim);">No extra services declared.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="card-title">Background Workers</div>
        <div id="workers-body" style="font-family: var(--font-mono); font-size: 0.88rem; color: var(--text-muted);">
          No active background worker queues detected.
        </div>
      </div>
    </div>

    <!-- TAB 3: Project Overview -->
    <div class="panel" id="panel-overview">
      <div class="card">
        <div class="card-title">Environment & Runtime</div>
        <div class="kv-grid" id="env-grid"></div>
      </div>
      <div class="card">
        <div class="card-title">Configuration Metadata</div>
        <div class="kv-grid" id="cfg-grid"></div>
      </div>
    </div>

    <!-- TAB 4: Live Telemetry Feed -->
    <div class="panel" id="panel-logs">
      <div class="card">
        <div class="card-title">
          <span>Real-time SSE Pulse Stream</span>
          <span style="font-size: 0.75rem; color: var(--text-dim); font-family: var(--font-mono);">Endpoint: /api/events</span>
        </div>
        <div class="log-feed" id="log-feed"></div>
      </div>
    </div>
  </main>

  <footer>
    Krewire Dev Dashboard &bull; <a href="/api/health" target="_blank">/api/health</a> &bull; <a href="/api/metrics" target="_blank">/api/metrics</a> &bull; <a href="/api/info" target="_blank">/api/info</a>
  </footer>

  <script>
    const state = {
      info: {{.InitialJSON}},
      routes: []
    };

    function init() {
      if (!state.info) return;
      document.getElementById('brand-project').textContent = state.info.name || 'Krewire App';
      document.getElementById('val-kind').textContent = state.info.kind || 'app';
      document.getElementById('val-env').textContent = state.info.env || 'local';
      document.getElementById('stat-platform').textContent = state.info.platform + ' (' + state.info.go_version + ')';

      // Render routes
      state.routes = state.info.routes || [];
      renderRoutes(state.routes);

      // Render services
      const svcsBody = document.getElementById('services-body');
      if (state.info.services && state.info.services.length > 0) {
        svcsBody.innerHTML = state.info.services.map(s => ` + "`" + `
          <tr>
            <td><strong>${escapeHTML(s.name)}</strong></td>
            <td><code>:${s.port}</code></td>
            <td><span class="badge badge-api">${escapeHTML(s.protocol)}</span></td>
          </tr>
        ` + "`" + `).join('');
      }

      // Render workers
      const workersBody = document.getElementById('workers-body');
      if (state.info.workers && state.info.workers.length > 0) {
        workersBody.innerHTML = state.info.workers.map(w => ` + "`" + `<div style="padding: 0.35rem 0;">&bull; ${escapeHTML(w)}</div>` + "`" + `).join('');
      }

      // Render env grid
      const envGrid = document.getElementById('env-grid');
      envGrid.innerHTML = ` + "`" + `
        <div class="kv-item"><div class="kv-key">Project Root</div><div class="kv-val">${escapeHTML(state.info.root)}</div></div>
        <div class="kv-item"><div class="kv-key">Go Toolchain</div><div class="kv-val">${escapeHTML(state.info.go_version)}</div></div>
        <div class="kv-item"><div class="kv-key">Architecture</div><div class="kv-val">${escapeHTML(state.info.platform)}</div></div>
        <div class="kv-item"><div class="kv-key">Krewire Version</div><div class="kv-val">${escapeHTML(state.info.version)}</div></div>
      ` + "`" + `;

      // Render config grid
      const cfgGrid = document.getElementById('cfg-grid');
      const cfg = state.info.config || {};
      cfgGrid.innerHTML = Object.entries(cfg).map(([k, v]) => ` + "`" + `
        <div class="kv-item"><div class="kv-key">${escapeHTML(k)}</div><div class="kv-val">${escapeHTML(String(v))}</div></div>
      ` + "`" + `).join('');

      connectSSE();
    }

    function renderRoutes(routes) {
      document.getElementById('route-count').textContent = routes.length;
      const tbody = document.getElementById('routes-body');
      if (routes.length === 0) {
        tbody.innerHTML = '<tr><td colspan="3" style="text-align:center; color: var(--text-dim);">No routes matched.</td></tr>';
        return;
      }
      tbody.innerHTML = routes.map(r => {
        let badgeClass = 'badge-page';
        if (r.kind === 'api') badgeClass = 'badge-api';
        if (r.kind === 'content') badgeClass = 'badge-content';
        return ` + "`" + `
          <tr>
            <td><code style="color: var(--accent-cyan); font-weight: 600;">${escapeHTML(r.path)}</code></td>
            <td><span class="badge ${badgeClass}">${escapeHTML(r.kind)}</span></td>
            <td><code style="color: var(--text-muted);">${escapeHTML(r.file)}</code></td>
          </tr>
        ` + "`" + `;
      }).join('');
    }

    function filterRoutes(q) {
      q = (q || '').toLowerCase();
      const filtered = state.routes.filter(r => r.path.toLowerCase().includes(q) || r.file.toLowerCase().includes(q));
      renderRoutes(filtered);
    }

    function switchTab(name) {
      document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
      document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
      event.target.classList.add('active');
      document.getElementById('panel-' + name).classList.add('active');
    }

    function connectSSE() {
      const feed = document.getElementById('log-feed');
      const dot = document.getElementById('pulse-dot');
      const statusText = document.getElementById('stream-status');

      const es = new EventSource('/api/events');
      es.addEventListener('pulse', e => {
        try {
          const data = JSON.parse(e.data);
          document.getElementById('stat-goroutines').textContent = data.goroutines;
          document.getElementById('stat-alloc').textContent = data.alloc_mb.toFixed(2) + ' MB';
          document.getElementById('stat-sys').textContent = 'Sys: ' + data.sys_mb.toFixed(2) + ' MB (Total: ' + data.total_alloc_mb.toFixed(2) + ' MB)';
          document.getElementById('stat-gc').textContent = data.num_gc;
          document.getElementById('stat-uptime').textContent = formatUptime(data.uptime_seconds);

          dot.style.background = 'var(--accent-emerald)';
          dot.style.boxShadow = '0 0 8px var(--accent-emerald)';
          statusText.textContent = 'LIVE STREAM';

          const entry = document.createElement('div');
          entry.className = 'log-entry';
          entry.innerHTML = ` + "`" + `
            <span class="log-time">[${data.timestamp.split('T')[1].replace('Z','')}]</span>
            <span class="log-msg">pulse telemetry: goroutines=${data.goroutines}, alloc=${data.alloc_mb.toFixed(2)}MB, num_gc=${data.num_gc}</span>
          ` + "`" + `;
          feed.prepend(entry);
          if (feed.children.length > 50) {
            feed.removeChild(feed.lastChild);
          }
        } catch (err) {}
      });

      es.onerror = () => {
        dot.style.background = 'var(--accent-amber)';
        dot.style.boxShadow = '0 0 8px var(--accent-amber)';
        statusText.textContent = 'RECONNECTING';
      };
    }

    function formatUptime(s) {
      if (s < 60) return s + 's';
      const m = Math.floor(s / 60);
      const rem = s % 60;
      if (m < 60) return m + 'm ' + rem + 's';
      const h = Math.floor(m / 60);
      return h + 'h ' + (m % 60) + 'm';
    }

    function escapeHTML(str) {
      if (!str) return '';
      return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    }

    window.addEventListener('DOMContentLoaded', init);
  </script>
</body>
</html>
`
