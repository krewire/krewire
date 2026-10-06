package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/scaffold"
)

func RegisterNew(fs *flag.FlagSet) {
	fs.String("module", "", "module path for the new project (defaults to the project name)")
	fs.String("dir", "", "directory to create the project in (defaults to the current directory)")
	fs.Bool("site", false, "equip a static site (pages/, layouts/, components/)")
	fs.Bool("book", false, "equip a manuscript book (mdbind)")
	fs.Bool("cli", false, "equip a command-line application (framework/tui)")
	fs.Bool("app", false, "equip a fullstack monolith application")
	fs.String("template", "", "bootstrap from a remote git template (git URL)")
	fs.String("title", "", "site title for the site and book variants")
}

// RunNew creates a new Krewire project: initializing minimal ecosystem files and scaffolding.
func RunNew(fs *flag.FlagSet) kern.ExitCode {
	name := fs.Arg(0)
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: kiw new <project> [--site|--book|--cli|--app|--template <git-url>]")
		return kern.ExitCodeUsage
	}

	site := flagBool(fs, "site")
	book := flagBool(fs, "book")
	cli := flagBool(fs, "cli")
	app := flagBool(fs, "app")
	templateURL := flagValue(fs, "template")

	if count := boolCount(site, book, cli, app, templateURL != ""); count > 1 {
		fmt.Fprintln(os.Stderr, "kiw new: choose at most one variant: --site, --book, --cli, --app, or --template")
		return kern.ExitCodeUsage
	}

	parentDir := flagValue(fs, "dir")
	cleanedPath := filepath.Clean(name)
	projectName := filepath.Base(cleanedPath)
	if projectName == "." || projectName == "/" || projectName == "" {
		projectName = "krewire-project"
	}

	modulePath := flagValue(fs, "module")
	if modulePath == "" {
		modulePath = projectName
	}

	var targetDir string
	if filepath.IsAbs(cleanedPath) {
		targetDir = cleanedPath
	} else if parentDir != "" {
		targetDir = filepath.Join(parentDir, cleanedPath)
	} else {
		targetDir = cleanedPath
	}

	// 1. Run minimal Init (git init, .gitignore, krewire.yaml) in the target directory
	if _, err := scaffold.New(scaffold.Options{
		Name:   name,
		Dir:    parentDir,
		Module: modulePath,
	}); err != nil {
		return commandError(err)
	}

	// 2. Run scaffolding (default is fullstack monolith VariantApp)
	opts := scaffold.EquipOptions{
		Dir:         targetDir,
		Name:        projectName,
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
		opts.Module = modulePath
		opts.Name = name
		fw, libs := resolveVersions()
		opts.FrameworkVersion = fw
		opts.LibsVersion = libs
	case templateURL != "":
		opts.Variant = scaffold.VariantTemplate
	default: // app fullstack monolith scaffolding
		opts.Variant = scaffold.VariantApp
		opts.Module = modulePath
		opts.Name = name
		fw, libs := resolveVersions()
		opts.FrameworkVersion = fw
		opts.LibsVersion = libs
	}

	eqCreated, err := scaffold.Equip(opts)
	if err != nil {
		if isScaffoldUsage(err) {
			return commandError(err)
		}
		return fail(err)
	}

	slog.Info("scaffolded Krewire project", "name", name, "variant", opts.Variant, "files", len(eqCreated))
	for _, path := range eqCreated {
		fmt.Println("created " + filepath.Join(name, path))
	}
	fmt.Printf("\nNext steps:\n  cd %s\n  kiw dev\n", name)
	return kern.ExitCodeSuccess
}
