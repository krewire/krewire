package book

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// defaultServerTimeouts prevents connection exhaustion and Slowloris attacks.
const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

// newServer constructs an http.Server with hardened timeouts.
func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
}

// Handler returns an http.Handler that serves the book's pages and assets
// through a stdlib ServeMux. It is intentionally not dependent on
// framework/web so the book can be served without pulling the framework.
func (b *Book) Handler() (http.Handler, error) {
	pages, err := b.Pages()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	for _, p := range pages {
		pp := p
		route := pp.Path
		if b.mount != "" {
			route = "/" + b.mount + pp.Path
		}
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != route {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = pp.Render(w)
		})
	}
	css := Assets()["assets/krewire.css"]
	mux.HandleFunc(b.mount+"/assets/krewire.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte(css))
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern == "" {
			http.Error(w, "404 — page not found", http.StatusNotFound)
			return
		}
		h.ServeHTTP(w, r)
	})
	return handler, nil
}

// Serve serves the book over HTTP on addr, shutting down gracefully on
// SIGINT/SIGTERM or when ctx is cancelled.
func Serve(b *Book, addr string) error {
	h, err := b.Handler()
	if err != nil {
		return err
	}
	slog.Info("serving book", "addr", addr)
	return serveWithGracefulShutdown(newServer(addr, h))
}

// serveWithGracefulShutdown runs srv until SIGINT/SIGTERM, then drains
// in-flight requests within defaultShutdownTimeout.
func serveWithGracefulShutdown(srv *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		_ = srv.Close()
		return err
	}
	return nil
}

// LiveHandler returns an http.Handler that re-reads Markdown files on every
// request during local development, providing instant updates without server restarts.
func LiveHandler(cfg Config) http.Handler {
	css := Assets()["assets/krewire.css"]
	mount := normalizeBase(cfg.MountPath)
	mountPrefix := strings.TrimSuffix(mount, "/")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cssPath := "/assets/krewire.css"
		if mountPrefix != "" {
			cssPath = "/" + mountPrefix + "/assets/krewire.css"
		}
		if r.URL.Path == cssPath {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			_, _ = w.Write([]byte(css))
			return
		}

		// Rebuild in-memory book representation
		base := cfg.BasePath
		if mount != "/" && (base == "" || base == "/") {
			base = mount
		}
		input := cfg.Input
		if input == "" {
			input = "."
		}
		b, err := loadWithRules(input, cfg.Title, cfg.Author, base, cfg.Include, cfg.Exclude)
		if err != nil {
			http.Error(w, "500 — error loading manuscript: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b.mount = mountPrefix
		b.noRootTOC = cfg.NoRootTOC
		b.version = cfg.Version
		b.WithChrome(cfg.NavLinks, cfg.FooterText)
		if cfg.Theme != nil {
			b.WithTheme(cfg.Theme)
		}

		pages, err := b.Pages()
		if err != nil {
			http.Error(w, "500 — error rendering pages: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for _, p := range pages {
			route := p.Path
			if b.mount != "" {
				route = "/" + b.mount + p.Path
			}
			if r.URL.Path == route {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = p.Render(w)
				return
			}
		}

		http.Error(w, "404 — page not found", http.StatusNotFound)
	})
}

// ServeConfig serves the book using live loading on addr, shutting down
// gracefully on SIGINT/SIGTERM.
func ServeConfig(cfg Config, addr string) error {
	h := LiveHandler(cfg)
	slog.Info("serving live book", "addr", addr)
	return serveWithGracefulShutdown(newServer(addr, h))
}
