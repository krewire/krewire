package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/krewire/krewire/packages/errs"
	"github.com/krewire/krewire/tools/kiw/internal/scaffold"
)

// RegisterMake registers flags for the make command.
func RegisterMake(fs *flag.FlagSet) {
	fs.String("kind", "", "generator kind: component, page, handler, migration, model, middleware, service, job")
	fs.String("name", "", "name for the generated artifact")
}

// RunMake scaffolds new components, pages, handlers, migrations, models, middleware, services, or jobs.
func RunMake(fs *flag.FlagSet) errs.ExitCode {
	kind := flagValue(fs, "kind")
	name := flagValue(fs, "name")

	// Support positional arguments: kiw make <kind> <name>
	args := fs.Args()
	if kind == "" && len(args) > 0 {
		kind = args[0]
		args = args[1:]
	}
	// Support colon syntax e.g. "make:component" or "component:Card"
	if strings.Contains(kind, ":") {
		parts := strings.SplitN(kind, ":", 2)
		if strings.ToLower(parts[0]) == "make" {
			kind = parts[1]
		} else if name == "" && parts[1] != "" {
			kind = parts[0]
			name = parts[1]
		}
	}
	if name == "" && len(args) > 0 {
		name = args[0]
	}

	if kind == "" || name == "" {
		fmt.Fprintln(os.Stderr, "kiw make: usage: kiw make <kind> <name>")
		fmt.Fprintln(os.Stderr, "supported kinds: component, page, handler, migration, model, middleware, service, job")
		return errs.ExitCodeUsage
	}

	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return errs.ExitCodeUsage
	}

	if err := Make(root, kind, name); err != nil {
		fmt.Fprintln(os.Stderr, "kiw make: "+err.Error())
		if errors.Is(err, scaffold.ErrUnknownKind) {
			return errs.ExitCodeUsage
		}
		return errs.ExitCodeFailure
	}
	return errs.ExitCodeSuccess
}

// Make executes the scaffolding generator for a given kind and artifact name.
func Make(root, kind, name string) error {
	return scaffold.MakeArtifact(root, kind, name)
}
