// Command fixture is the service workload conformance fixture: a headless
// REST service whose dependencies are framework providers with a managed
// start/stop lifecycle.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/krewire/krewire/pkg/app"
	"github.com/krewire/krewire/pkg/cloud/service"
	"github.com/krewire/krewire/pkg/config"
	"github.com/krewire/krewire/pkg/web"
)

// store is a provider whose backing store is a required dependency.
type store struct{ items []string }

func (*store) Name() string { return "store" }

func (s *store) Register(r service.Registry) error {
	kind := os.Getenv("SERVICE_STORE")
	if kind == "" {
		return errors.New("store provider: missing dependency: SERVICE_STORE is not set")
	}
	s.items = []string{"alpha", "beta"}
	return r.Set("store", s)
}

func (*store) Start(context.Context, service.Registry) error {
	fmt.Println("PROVIDER started")
	return nil
}

func (*store) Stop(context.Context, service.Registry) error {
	fmt.Println("PROVIDER stopped")
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var cfg struct {
		Project struct {
			Name string `yaml:"name"`
		} `yaml:"project"`
	}
	if err := config.Load("krewire.yaml", &cfg); err != nil {
		return err
	}
	if cfg.Project.Name == "" {
		return errors.New("krewire.yaml: project.name is required")
	}
	if len(args) == 0 || args[0] == "check" {
		fmt.Println("ok", cfg.Project.Name)
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s := &store{}
	application := app.NewApplication()
	if err := application.Use(s); err != nil {
		return err
	}
	api := web.NewApp()
	api.Use(web.Health())
	api.Router().Get("/v1/items", func(w http.ResponseWriter, _ *http.Request, _ web.Params) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": s.items})
	})
	addr := os.Getenv("KREWIRE_ADDR")
	fmt.Println("READY", addr)
	err := application.Run(ctx, app.HttpRunner(api.Handler(), addr))
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	fmt.Println("STOPPED")
	return nil
}
