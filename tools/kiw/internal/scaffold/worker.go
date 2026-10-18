// Package-level file for the background queue worker application (VariantWorker).
package scaffold

import (
	"fmt"
	"os"
)

// equipWorker shapes the kernel into a background job queue worker application.
func equipWorker(opts EquipOptions) ([]string, error) {
	if opts.Module == "" {
		return nil, fmt.Errorf("equip worker: module path is required (read from go.mod)")
	}
	if opts.Name == "" {
		return nil, fmt.Errorf("equip worker: project name is required")
	}
	frameworkVersion := opts.FrameworkVersion
	if frameworkVersion == "" {
		frameworkVersion = "latest"
	}
	libsVersion := opts.LibsVersion
	if libsVersion == "" {
		libsVersion = "v0.1.0"
	}

	cwd, _ := os.Getwd()
	frameworkPath, libsPath := detectLocalPaths(cwd)

	files := []file{
		{goModFile, goModTemplate(opts.Module, frameworkVersion, libsVersion, frameworkPath, libsPath)},
		{krewireYaml, workerKrewireYamlTemplate(opts.Name)},
		{mainGo, workerMainTemplate(opts.Module, opts.Name)},
		{"internal/jobs/jobs.go", workerJobsTemplate()},
		{"README.md", workerReadmeTemplate(opts.Name)},
		{gitignoreFile, gitignoreTemplate(opts.Name)},
	}
	return writeVariant(opts.Dir, files)
}

func workerKrewireYamlTemplate(name string) string {
	return fmt.Sprintf(`project:
  name: %s
  kind: worker
  version: 0.1.0

worker:
  concurrency: 4
  queues:
    - default
    - critical
`, name)
}

func workerMainTemplate(module, name string) string {
	return fmt.Sprintf(`// Entry point: run the background queue worker.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/krewire/krewire/packages/cloud/worker"
	"%s/internal/jobs"
)

func main() {
	slog.Info("starting %s worker daemon")

	q := worker.NewMemoryQueue()

	// Enqueue initial sample task
	_ = q.Enqueue(context.Background(), &jobs.ExampleJob{Payload: "hello background task"}, worker.Options{
		Queue: "default",
	})

	runner := worker.NewRunner(q, worker.WithConcurrency(4))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down worker...")
		runner.Stop()
		cancel()
	}()

	if err := runner.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "worker error: %%v\n", err)
		os.Exit(1)
	}
	slog.Info("worker stopped cleanly")
}
`, module, name)
}

func workerJobsTemplate() string {
	return `// Package jobs defines runnable task handlers for the background queue.
package jobs

import (
	"context"
	"log/slog"
)

// ExampleJob demonstrates a background queue job.
type ExampleJob struct {
	Payload string
}

func (j *ExampleJob) Name() string {
	return "example.job"
}

func (j *ExampleJob) Run(ctx context.Context) error {
	slog.Info("processing job", "payload", j.Payload)
	return nil
}
`
}

func workerReadmeTemplate(name string) string {
	return fmt.Sprintf(`# %s (Worker)

Background queue processing worker powered by Krewire.

## Development

Run the worker daemon locally:

`+"```bash"+`
kiw run
`+"```"+`

Run in development mode with auto-rebuild:

`+"```bash"+`
kiw dev
`+"```"+`

## Building & Deploying

Compile into a production binary:

`+"```bash"+`
kiw build
`+"```"+`

Stage deployable artifacts:

`+"```bash"+`
kiw deploy
`+"```"+`
`, name)
}
