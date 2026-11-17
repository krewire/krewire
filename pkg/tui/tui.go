// Package tui provides the terminal UI and command-line application model for the
// Krewire unified framework.
//
// It composes the Go standard library's flag and log/slog packages with the
// ecosystem's term package into a cohesive TUI/CLI application model. The
// package was formerly named cli; it was renamed to tui to avoid confusion
// with CLI tool binaries (e.g., krewire/krewire).
package tui

import (
	"flag"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
)

// Command is a runnable sub-command.
type Command struct {
	Name    string
	About   string
	Group   string
	Example string
	Hidden  bool
	// Core marks an everyday workflow command. When any command is Core,
	// general help lists core commands in full and the rest compactly
	// ("progressive disclosure"); `help --all` lists everything in full.
	Core     bool
	Register func(*flag.FlagSet)
	Run      func(*flag.FlagSet) errs.ExitCode
}

// NewCommand creates a Command with the given name, about, register and run
// functions.
func NewCommand(name, about string, register func(*flag.FlagSet), run func(*flag.FlagSet) errs.ExitCode) *Command {
	return &Command{Name: name, About: about, Register: register, Run: run}
}

// WithGroup sets the help group for the command and returns the command for chaining.
func (c *Command) WithGroup(group string) *Command {
	c.Group = group
	return c
}

// WithExample sets an example invocation and returns the command.
func (c *Command) WithExample(example string) *Command {
	c.Example = example
	return c
}

// WithHidden marks the command as hidden from general help output.
func (c *Command) WithHidden(hidden bool) *Command {
	c.Hidden = hidden
	return c
}

// WithCore marks the command as a core workflow command.
func (c *Command) WithCore() *Command {
	c.Core = true
	return c
}

// App is the framework's CLI application model.
type App struct {
	name        string
	version     string
	description string
	commands    []*Command
	logger      *slog.Logger
	terminal    *term.Terminal
	stderr      io.Writer
	hints       map[string]string
}

// NewApp creates an App with the given name and version.
func NewApp(name, version string) *App {
	return &App{
		name:     name,
		version:  version,
		logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
		terminal: term.NewTerminal(),
		stderr:   os.Stderr,
	}
}

// WithDescription sets a one-line description shown under the banner.
func (a *App) WithDescription(desc string) *App {
	a.description = desc
	return a
}

// WithHint maps a name that is not a command (for example a command from
// another tool, such as "start") to a suggestion shown instead of the
// generic "did you mean" match, e.g. "kiw dev  (live dev server)".
func (a *App) WithHint(name, suggestion string) *App {
	if a.hints == nil {
		a.hints = map[string]string{}
	}
	a.hints[name] = suggestion
	return a
}

// Command registers a Command and returns the App for chaining.
func (a *App) Command(cmd *Command) *App {
	a.commands = append(a.commands, cmd)
	return a
}

// Name returns the application name.
func (a *App) Name() string {
	return a.name
}

// Run parses tokens, dispatches to the matching command, and returns the
// resulting exit code. Help is consistent across all forms (subcommand and flag):
//
//	kiw help                → general help (success)
//	kiw help --all | -a     → complete command reference (success)
//	kiw --help | -h         → general help (success) — flag alias
//	kiw help <command>      → help for <command> (success) — canonical subcommand form
//	kiw <command> help      → help for <command> (success) — alias, kept for ergonomics
//	kiw <command> --help    → help for <command> (success) — flag form
//	kiw <command> -h        → help for <command> (success) — short flag
//
// Both subcommand and flag are supported for every command; they are equivalent
// and always exit 0 when help is shown. The canonical form is `kiw help <command>`.
func (a *App) Run(tokens []string) errs.ExitCode {
	if len(tokens) == 0 {
		a.printHelp()
		return errs.ExitCodeUsage
	}

	// General help via flag: kiw --help, kiw -h, kiw -help (optionally --all)
	if isHelpFlag(tokens[0]) || isAllFlag(tokens[0]) {
		a.printHelpMode(isAllFlag(tokens[0]) || (len(tokens) > 1 && isAllFlag(tokens[1])))
		return errs.ExitCodeSuccess
	}

	// kiw help [command] — canonical subcommand form
	if tokens[0] == "help" {
		if len(tokens) == 1 {
			a.printHelp()
			return errs.ExitCodeSuccess
		}
		// `kiw help --help` → general help; `kiw help --all` → full reference
		if isHelpFlag(tokens[1]) || isAllFlag(tokens[1]) {
			a.printHelpMode(isAllFlag(tokens[1]))
			return errs.ExitCodeSuccess
		}
		if cmd := a.findCommand(tokens[1]); cmd != nil {
			a.printCommandHelp(cmd)
			return errs.ExitCodeSuccess
		}
		a.printUnknownCommand(tokens[1])
		a.printHelp()
		return errs.ExitCodeUsage
	}

	name := tokens[0]
	rest := tokens[1:]

	cmd := a.findCommand(name)
	if cmd == nil {
		a.printUnknownCommand(name)
		a.printHelp()
		return errs.ExitCodeUsage
	}

	// kiw <command> help | --help | -h  → show command help without running
	for _, arg := range rest {
		if arg == "help" || arg == "--help" || arg == "-h" || arg == "-help" {
			a.printCommandHelp(cmd)
			return errs.ExitCodeSuccess
		}
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if cmd.Register != nil {
		cmd.Register(fs)
	}

	a.logger.Debug("dispatching command", "command", cmd.Name)
	parsedArgs := reorderArgs(fs, rest)
	if err := fs.Parse(parsedArgs); err != nil {
		return errs.ExitCodeUsage
	}
	return cmd.Run(fs)
}

func isHelpFlag(s string) bool { return s == "--help" || s == "-h" || s == "-help" }

func isAllFlag(s string) bool { return s == "--all" || s == "-a" || s == "-all" }

func reorderArgs(fs *flag.FlagSet, args []string) []string {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			hasEqual := strings.Contains(name, "=")
			if hasEqual {
				flags = append(flags, arg)
				continue
			}
			f := fs.Lookup(name)
			if f != nil {
				if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
					flags = append(flags, arg)
				} else if i+1 < len(args) {
					flags = append(flags, arg, args[i+1])
					i++
				} else {
					flags = append(flags, arg)
				}
			} else {
				flags = append(flags, arg)
			}
		} else {
			positionals = append(positionals, arg)
		}
	}
	return append(flags, positionals...)
}

// findCommand returns the Command with the given name, or nil.
func (a *App) findCommand(name string) *Command {
	for _, c := range a.commands {
		if c.Name == name {
			return c
		}
	}
	return nil
}
