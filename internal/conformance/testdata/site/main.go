// Command fixture is the site workload conformance fixture. It drives the
// framework the way a developer project does: load the declarative site,
// compile it, and serve the preview. The harness owns all assertions.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/krewire/krewire/pkg/web/ssg"
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
	site, err := ssg.LoadFromDir(".")
	if err != nil {
		return err
	}
	switch args[0] {
	case "check":
		out, err := os.MkdirTemp("", "site-check-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(out)
		if _, err := site.Build(out); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
			return fmt.Errorf("site has no index page: %w", err)
		}
		fmt.Println("ok")
		return nil
	case "build":
		if len(args) < 2 {
			return errors.New("build needs an output directory")
		}
		created, err := site.Build(args[1])
		fmt.Println("built", len(created), "files")
		return err
	case "serve":
		return serve(site.Handler())
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
