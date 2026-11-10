package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/tools/kiw/internal/scaffold"
)

// RegisterForge registers flags for the forge command.
func RegisterForge(fs *flag.FlagSet) {
	fs.String("model", "", "resource: model name to bind to (e.g. User, Product)")
	fs.String("fields", "", "form: comma-separated field definitions (e.g. title:string,price:float,status:select)")
	fs.Int("port", DefaultForgeStudioPort, fmt.Sprintf("studio: HTTP bind port (default: %d)", DefaultForgeStudioPort))
	fs.Bool("dry-run", false, "preview generated files without writing to disk")
}

const forgeUsage = `Krewire Forge — Declarative Web, Form, and Admin Panel Builder for Go & .kiw

Usage:
  kiw forge panel <name>             Scaffold a new admin panel dashboard
  kiw forge resource <name>          Scaffold a CRUD resource (model, table view, and forms)
  kiw forge form <name>              Generate a declarative .kiw form with validation struct
  kiw forge studio [--port=8089]     Launch the local interactive visual builder studio

Examples:
  kiw forge panel admin
  kiw forge resource product --model=Product
  kiw forge form contact --fields="name:string,email:string,message:textarea"
  kiw forge studio --port=8089
`

// RunForge dispatches Krewire Forge scaffolding and studio subcommands.
func RunForge(fs *flag.FlagSet) errs.ExitCode {
	args := fs.Args()
	if len(args) == 0 {
		fmt.Print(forgeUsage)
		return errs.ExitCodeUsage
	}

	sub := strings.ToLower(args[0])
	dryRun := flagBool(fs, "dry-run")
	model := flagValue(fs, "model")
	fields := flagValue(fs, "fields")
	port := flagInt(fs, "port")
	if port <= 0 {
		port = DefaultForgeStudioPort
	}

	root, err := findRoot()
	if err != nil {
		root, _ = os.Getwd()
	}

	switch sub {
	case "panel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: panel name required (e.g. kiw forge panel admin)")
			return errs.ExitCodeUsage
		}
		if err := scaffold.ForgePanel(root, args[1], dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge panel: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "resource":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: resource name required (e.g. kiw forge resource product)")
			return errs.ExitCodeUsage
		}
		if err := scaffold.ForgeResource(root, args[1], model, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge resource: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "form":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: form name required (e.g. kiw forge form contact)")
			return errs.ExitCodeUsage
		}
		if err := scaffold.ForgeForm(root, args[1], fields, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge form: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "studio":
		if err := RunForgeStudio(port); err != nil {
			fmt.Fprintf(os.Stderr, "kiw forge studio: %v\n", err)
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess

	case "help", "-h", "--help":
		fmt.Print(forgeUsage)
		return errs.ExitCodeSuccess

	default:
		fmt.Fprintf(os.Stderr, "kiw forge: unknown subcommand %q\n\n%s", args[0], forgeUsage)
		return errs.ExitCodeUsage
	}
}

// Backward-compatible delegates for commands internal tests.
func forgePanel(root, name string, dryRun bool) error {
	return scaffold.ForgePanel(root, name, dryRun)
}

func forgeResource(root, name, model string, dryRun bool) error {
	return scaffold.ForgeResource(root, name, model, dryRun)
}

func forgeForm(root, name, fieldsSpec string, dryRun bool) error {
	return scaffold.ForgeForm(root, name, fieldsSpec, dryRun)
}

func forgeStudio(port int) error {
	return RunForgeStudio(port)
}

func forgeToSlug(s string) string {
	return scaffold.ToSlug(s)
}

func forgeToPascal(s string) string {
	return scaffold.ToPascal(s)
}

func forgeToTitle(s string) string {
	return scaffold.ToTitle(s)
}
