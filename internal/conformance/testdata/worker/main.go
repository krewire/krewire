// Command fixture is the worker workload conformance fixture: a supervised
// daemon consuming a queue with retries, driven by krewire.yaml.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/krewire/krewire/pkg/cloud/worker"
	"github.com/krewire/krewire/pkg/config"
)

type settings struct {
	Worker struct {
		Schedule    string `yaml:"schedule"`
		Concurrency int    `yaml:"concurrency"`
	} `yaml:"worker"`
}

var processed atomic.Int32

const total = 3

type job struct {
	id       int
	flaky    bool
	attempts atomic.Int32
}

func (j *job) Run(context.Context) error {
	n := j.attempts.Add(1)
	if j.flaky && n == 1 {
		return errors.New("transient failure")
	}
	if n > 1 {
		fmt.Println("RETRIED job", j.id, "attempt", n)
	}
	if processed.Add(1) == total {
		fmt.Println("PROCESSED", total)
	}
	return nil
}

type tick struct{}

func (tick) Run(context.Context) error { return nil }

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var cfg settings
	if err := config.Load("krewire.yaml", &cfg); err != nil {
		return err
	}
	if _, err := worker.ParseCron(cfg.Worker.Schedule); err != nil {
		return fmt.Errorf("krewire.yaml worker.schedule: %w", err)
	}
	if len(args) == 0 || args[0] == "check" {
		fmt.Println("ok")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	q := worker.NewInMemoryQueue()
	retry := &worker.RetryPolicy{MaxAttempts: 3, InitialBackoff: 50 * time.Millisecond, MaxBackoff: 200 * time.Millisecond}
	for i := 1; i <= total; i++ {
		if _, err := q.Enqueue(ctx, &job{id: i, flaky: i == 2}, worker.Options{Retry: retry}); err != nil {
			return err
		}
	}
	if _, err := q.Enqueue(ctx, tick{}, worker.Options{Cron: cfg.Worker.Schedule}); err != nil {
		return err
	}
	fmt.Println("READY queue")
	err := worker.NewRunner(q, worker.WithConcurrency(cfg.Worker.Concurrency)).Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	fmt.Println("STOPPED")
	return nil
}
