package app

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewHttpServer_Timeouts(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	addr := "127.0.0.1:0"

	srv := NewHttpServer(handler, addr)

	if srv.Addr != addr {
		t.Errorf("expected Addr %q, got %q", addr, srv.Addr)
	}
	if srv.ReadHeaderTimeout != DefaultReadHeaderTimeout {
		t.Errorf("expected ReadHeaderTimeout %v, got %v", DefaultReadHeaderTimeout, srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != DefaultReadTimeout {
		t.Errorf("expected ReadTimeout %v, got %v", DefaultReadTimeout, srv.ReadTimeout)
	}
	if srv.WriteTimeout != DefaultWriteTimeout {
		t.Errorf("expected WriteTimeout %v, got %v", DefaultWriteTimeout, srv.WriteTimeout)
	}
	if srv.IdleTimeout != DefaultIdleTimeout {
		t.Errorf("expected IdleTimeout %v, got %v", DefaultIdleTimeout, srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != DefaultMaxHeaderBytes {
		t.Errorf("expected MaxHeaderBytes %d, got %d", DefaultMaxHeaderBytes, srv.MaxHeaderBytes)
	}
}

func TestHttpRunner_GracefulShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	runner := HttpRunner(handler, "127.0.0.1:0")

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- runner.Run(ctx, nil)
	}()

	// Allow server to start listening
	time.Sleep(50 * time.Millisecond)

	// Cancel context to initiate graceful shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected error on shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("HttpRunner failed to shut down within timeout")
	}
}
