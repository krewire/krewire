package app

import (
	"context"
	"net/http"
	"time"

	"github.com/krewire/krewire/pkg/cloud/runner"
	"github.com/krewire/krewire/pkg/cloud/service"
)

// Default HTTP server timeout and header limits (ADR-D1F3N).
const (
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 60 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
	DefaultMaxHeaderBytes    = 1 << 20 // 1 MB
)

// NewHttpServer constructs an http.Server configured with hardened timeouts
// and header limits against Slowloris and connection starvation.
func NewHttpServer(handler http.Handler, addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
	}
}

// HttpRunner adapts any http.Handler into a runner.Runner supervised by Application.
func HttpRunner(handler http.Handler, addr string) runner.Runner {
	return runner.Func(func(ctx context.Context, _ service.Registry) error {
		srv := NewHttpServer(handler, addr)
		errCh := make(chan error, 1)
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
			return nil
		case err := <-errCh:
			return err
		}
	})
}
