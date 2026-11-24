// Package app is the developer code the app starter template's main.go
// (templates/init/app/main.go.tmpl) calls into. It composes an HTTP server
// and a background worker under one framework-managed lifecycle.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kwapp "github.com/krewire/krewire/app"
	"github.com/krewire/krewire/cloud/worker"
	"github.com/krewire/krewire/config"
	"github.com/krewire/krewire/web"
)

type project struct {
	Project struct {
		Name string `yaml:"name"`
	} `yaml:"project"`
}

type job struct{}

func (job) Run(context.Context) error {
	fmt.Println("JOB done")
	return nil
}

// Run serves the app until interrupted; "check" validates configuration.
func Run() error {
	var cfg project
	if err := config.Load("krewire.yaml", &cfg); err != nil {
		return err
	}
	if cfg.Project.Name == "" {
		return errors.New("krewire.yaml: project.name is required")
	}
	if len(os.Args) > 1 && os.Args[1] == "check" {
		fmt.Println("ok", cfg.Project.Name)
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	w := web.NewApp()
	w.Use(web.Health())
	w.Router().Get("/", func(rw http.ResponseWriter, _ *http.Request, _ web.Params) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(rw, "<!doctype html><title>app</title><h1>Conformance App</h1>")
	})
	w.Router().Get("/api/greet", func(rw http.ResponseWriter, r *http.Request, _ web.Params) {
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{"message": "hello " + html.EscapeString(r.URL.Query().Get("name"))})
	})

	q := worker.NewInMemoryQueue()
	if _, err := q.Enqueue(ctx, job{}, worker.Options{}); err != nil {
		return err
	}
	addr := os.Getenv("KREWIRE_ADDR")
	fmt.Println("READY", addr)
	err := kwapp.NewApplication().Run(ctx, kwapp.HttpRunner(w.Handler(), addr), worker.NewRunner(q).AsRunner())
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	fmt.Println("STOPPED")
	return nil
}
