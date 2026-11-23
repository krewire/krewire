// Command fixture is the gateway workload conformance fixture: the route table
// comes from krewire.yaml and upstreams are discovered through the service
// registry.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krewire/krewire/pkg/cloud/service/registry"
	"github.com/krewire/krewire/pkg/config"
	"github.com/krewire/krewire/pkg/web"
	"github.com/krewire/krewire/pkg/web/gateway"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func routes() ([]gateway.Route, error) {
	var raw struct {
		Gateway struct {
			Routes []map[string]any `yaml:"routes"`
		} `yaml:"gateway"`
	}
	if err := config.Load("krewire.yaml", &raw); err != nil {
		return nil, err
	}
	// gateway.Route carries json tags; route through JSON to honour them.
	b, err := json.Marshal(raw.Gateway.Routes)
	if err != nil {
		return nil, err
	}
	var rs []gateway.Route
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, fmt.Errorf("krewire.yaml gateway.routes: %w", err)
	}
	if len(rs) == 0 {
		return nil, errors.New("krewire.yaml: gateway.routes is empty")
	}
	return rs, nil
}

func run(args []string) error {
	rs, err := routes()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "check" {
		fmt.Println("ok", len(rs), "routes")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	reg := registry.NewMemoryRegistry()
	if err := reg.Register(ctx, registry.Service{ID: "backend-1", Name: "backend", Address: os.Getenv("UPSTREAM_ADDR")}); err != nil {
		return err
	}
	gw := gateway.New(gateway.WithRegistry(reg))
	gw.Reload(rs)

	l, err := net.Listen("tcp", os.Getenv("KREWIRE_ADDR"))
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: web.Health()(gw), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(l) }()
	fmt.Println("READY", l.Addr())
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return err
	}
	fmt.Println("STOPPED")
	return nil
}
