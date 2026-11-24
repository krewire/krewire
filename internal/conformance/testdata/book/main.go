// Command fixture is the book workload conformance fixture. It drives the
// framework the way a developer project does: load krewire.yaml, compile
// content/ into a static book, and serve it. The harness owns all assertions.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krewire/krewire/book"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: fixture check|build <out>|serve")
	}
	cfg, err := book.LoadConfigFile("krewire.yaml")
	if err != nil {
		return err
	}
	switch args[0] {
	case "check":
		b, err := book.Load(cfg.Input, cfg.Title, cfg.Author)
		if err != nil {
			return err
		}
		if len(b.Chapters) == 0 {
			return fmt.Errorf("book has no chapters in %s", cfg.Input)
		}
		fmt.Println("ok", len(b.Chapters), "chapters")
		return nil
	case "build":
		if len(args) < 2 {
			return errors.New("build needs an output directory")
		}
		cfg.Output = args[1]
		created, err := book.Build(*cfg)
		fmt.Println("built", len(created), "files")
		return err
	case "serve":
		return serve(book.LiveHandler(*cfg))
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func serve(h http.Handler) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	l, err := net.Listen("tcp", os.Getenv("KREWIRE_ADDR"))
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
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
