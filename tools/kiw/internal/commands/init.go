package commands

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/gomod"
	"github.com/krewire/krewire/tools/kiw/internal/scaffold"
)

// RegisterInit registers flags for the init command.
func RegisterInit(fs *flag.FlagSet) {
	fs.Bool("force", false, "force reset krewire.yaml without interactive confirmation")
	fs.Bool("f", false, "short alias for --force")
	fs.Bool("site", false, "equip a declarative static site (ssg: key in krewire.yaml)")
	fs.Bool("book", false, "equip a manuscript book (mdbind)")
	fs.Bool("cli", false, "equip a command-line application (framework/tui)")
	fs.Bool("app", false, "equip a fullstack monolith application")
	fs.String("template", "", "bootstrap from a remote git template (git URL)")
	fs.String("title", "", "site title for the site and book variants")
}

// RunInit initializes a minimal Krewire Ecosystem project in the target directory
// (default: current directory), creating krewire.yaml, .gitignore, and running git init.
// If krewire.yaml already exists, it prompts the user to reset it to default (or resets with --force).
// If a variant flag is given, it equips that variant into the project.
func RunInit(fs *flag.FlagSet) kern.ExitCode {
	dir := fs.Arg(0)
	if dir == "" {
		dir = "."
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fail(err)
	}

	name := filepath.Base(dir)
	if name == "/" || name == "." || name == "" {
		name = "krewire-project"
	}

	force := flagBool(fs, "force") || flagBool(fs, "f")

	// 1. Run minimal Krewire Ecosystem initialization (git init, .gitignore, krewire.yaml)
	initCreated, err := scaffold.Init(scaffold.InitOptions{
		Dir:   dir,
		Name:  name,
		Force: force,
		ConfirmReset: func() bool {
			fmt.Printf("krewire.yaml already exists in %s.\nReset to default configuration? [y/N]: ", dir)
			var resp string
			_, _ = fmt.Scanln(&resp)
			resp = strings.ToLower(strings.TrimSpace(resp))
			return resp == "y" || resp == "yes"
		},
	})
	if err != nil {
		return fail(err)
	}

	templateURL := flagValue(fs, "template")
	site := flagBool(fs, "site")
	book := flagBool(fs, "book")
	cli := flagBool(fs, "cli")
	app := flagBool(fs, "app")

	hasVariant := site || book || cli || app || templateURL != ""
	if !hasVariant {
		slog.Info("initialized minimal Krewire project", "dir", dir, "files", len(initCreated))
		for _, path := range initCreated {
			fmt.Println("created " + path)
		}
		fmt.Printf("Initialized empty Krewire project in %s\n", dir)
		return kern.ExitCodeSuccess
	}

	if count := boolCount(site, book, cli, app, templateURL != ""); count > 1 {
		fmt.Fprintln(os.Stderr, "kiw init: choose at most one variant: --site, --book, --cli, --app, or --template")
		return kern.ExitCodeUsage
	}

	opts := scaffold.EquipOptions{
		Dir:         dir,
		Title:       flagValue(fs, "title"),
		TemplateURL: templateURL,
	}
	switch {
	case site:
		opts.Variant = scaffold.VariantStatic
		opts.Name = name
		opts.Title = firstNonEmpty(opts.Title, name)
	case book:
		opts.Variant = scaffold.VariantBook
		opts.Name = name
		opts.Title = firstNonEmpty(opts.Title, name)
	case cli:
		opts.Variant = scaffold.VariantCLI
		mod, _ := gomod.Find(dir)
		if mod != nil {
			opts.Module = mod.Path
		} else {
			opts.Module = name
		}
		opts.Name = name
		fw, libs := resolveVersions()
		opts.FrameworkVersion = fw
		opts.LibsVersion = libs
	case templateURL != "":
		opts.Variant = scaffold.VariantTemplate
	default: // app
		opts.Variant = scaffold.VariantApp
		mod, _ := gomod.Find(dir)
		if mod != nil {
			opts.Module = mod.Path
		} else {
			opts.Module = name
		}
		opts.Name = name
		fw, libs := resolveVersions()
		opts.FrameworkVersion = fw
		opts.LibsVersion = libs
	}

	created, err := scaffold.Equip(opts)
	if err != nil {
		if isScaffoldUsage(err) {
			return commandError(err)
		}
		return fail(err)
	}
	slog.Info("equipped Krewire project", "dir", dir, "variant", opts.Variant, "files", len(created))
	for _, path := range created {
		fmt.Println("created " + path)
	}
	return kern.ExitCodeSuccess
}

// moduleBase returns the last path segment of a module path, used as the
// project name for the app variant.
func moduleBase(module string) string {
	parts := strings.Split(strings.TrimSuffix(module, "/"), "/")
	return parts[len(parts)-1]
}

// flagBool returns the boolean value of a registered flag.
func flagBool(fs *flag.FlagSet, name string) bool {
	if f := fs.Lookup(name); f != nil {
		return f.Value.String() == "true"
	}
	return false
}

// boolCount returns how many of the given conditions are true.
func boolCount(vals ...bool) int {
	n := 0
	for _, v := range vals {
		if v {
			n++
		}
	}
	return n
}

// isScaffoldUsage reports whether err is a scaffold usage-class error that
// should exit with code 2.
func isScaffoldUsage(err error) bool {
	return errors.Is(err, scaffold.ErrProjectExists) ||
		errors.Is(err, scaffold.ErrNotEmpty) ||
		errors.Is(err, scaffold.ErrConflict)
}
