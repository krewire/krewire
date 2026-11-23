// Package cmd is the command tree the cli starter template's main.go
// (templates/init/cli/main.go.tmpl) executes. In a real project it is
// equipped by `krew init --cli`; the fixture keeps it stdlib-only.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/krewire/krewire"
	"github.com/krewire/krewire/pkg/config"
)

// Execute runs the command line; a returned error makes main exit 1.
func Execute() error {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return err
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: fixture check|greet -name <n>|version")
	}
	switch args[0] {
	case "check":
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
		fmt.Println("ok", cfg.Project.Name)
		return nil
	case "greet":
		fs := flag.NewFlagSet("greet", flag.ContinueOnError)
		name := fs.String("name", "world", "who to greet")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		fmt.Printf("Hello, %s\n", *name)
		return nil
	case "version":
		fmt.Println("krewire", krewire.VersionString())
		return nil
	}
	return fmt.Errorf("unknown command %q", args[0])
}
