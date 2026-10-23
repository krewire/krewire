package app

import (
	"context"
	"net/http"
	"time"

	"github.com/krewire/krewire/packages/cloud/runner"
	"github.com/krewire/krewire/packages/cloud/service"
)

// HttpRunner adapts any http.Handler into a runner.Runner supervised by Application.
func HttpRunner(handler http.Handler, addr string) runner.Runner {
	return runner.Func(func(ctx context.Context, _ service.Registry) error {
		srv := &http.Server{Addr: addr, Handler: handler}
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
