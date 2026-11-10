package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// DefaultForgeStudioPort is the default HTTP listen port for the Forge visual studio.
	DefaultForgeStudioPort = 8089

	// DefaultForgeStudioHost is the default host address for the Forge visual studio.
	DefaultForgeStudioHost = "127.0.0.1"

	forgeStudioShutdownTimeout = 2 * time.Second
)

const forgeStudioHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Krewire Forge Studio</title>
  <style>
    :root { --bg: #071426; --card: #0c203b; --fg: #e2e8f0; --accent: #ff3366; --border: #1e3a5f; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: var(--bg); color: var(--fg); margin: 0; padding: 2rem; }
    .container { max-width: 900px; margin: 0 auto; }
    .card { background: var(--card); border: 2px solid var(--border); border-radius: 12px; padding: 2rem; margin-top: 1.5rem; box-shadow: 4px 4px 0 #000; }
    h1 { font-size: 2rem; font-weight: 900; margin: 0; }
    .badge { background: var(--accent); color: white; padding: 0.2rem 0.6rem; border-radius: 4px; font-size: 0.75rem; font-weight: 800; }
    pre { background: #040d1a; padding: 1rem; border-radius: 8px; border: 1px solid var(--border); font-family: monospace; overflow-x: auto; color: #38bdf8; }
  </style>
</head>
<body>
  <div class="container">
    <div style="display: flex; align-items: center; justify-content: space-between;">
      <h1>Krewire Forge Studio</h1>
      <span class="badge">LOCAL ENGINE</span>
    </div>
    <p style="color: #94a3b8;">Declarative Web, Form, and Admin Panel Builder for Go & .kiw</p>

    <div class="card">
      <h2 style="margin-top: 0;">Quick Scaffolding Commands</h2>
      <p style="color: #94a3b8; font-size: 0.9rem;">Run these commands directly in your project terminal:</p>
      <pre>$ kiw forge panel admin
$ kiw forge resource product --model=Product
$ kiw forge form contact --fields="name:string,email:string,message:textarea"</pre>
    </div>
  </div>
</body>
</html>`

// RunForgeStudio launches the local interactive visual builder studio HTTP server.
func RunForgeStudio(port int) error {
	if port <= 0 {
		port = DefaultForgeStudioPort
	}
	addr := fmt.Sprintf("%s:%d", DefaultForgeStudioHost, port)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(forgeStudioHTML))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("⚡ Krewire Forge Studio running on http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), forgeStudioShutdownTimeout)
	defer cancel()
	return server.Shutdown(ctx)
}
