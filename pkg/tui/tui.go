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
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
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

func (a *App) printUnknownCommand(name string) {
	msg := a.terminal.Paint("✗", term.ColorRed, []term.Style{term.StyleBold}) + " unknown command " + a.terminal.Paint(fmt.Sprintf("%q", name), term.ColorYellow, nil)
	a.stderr.Write([]byte(msg + "\n"))
	if hint, ok := a.hints[name]; ok {
		a.stderr.Write([]byte(a.terminal.Paint("  did you mean ", term.ColorDefault, []term.Style{term.StyleDim}) + a.terminal.Paint(hint, term.ColorCyan, nil) + "\n\n"))
		return
	}
	if sug := a.suggestCommand(name); sug != "" {
		hint := a.terminal.Paint("  did you mean ", term.ColorDefault, []term.Style{term.StyleDim}) + a.terminal.Paint(sug, term.ColorCyan, nil) + a.terminal.Paint(" ?", term.ColorDefault, []term.Style{term.StyleDim})
		a.stderr.Write([]byte(hint + "\n"))
	}
	a.stderr.Write([]byte("\n"))
}

func (a *App) suggestCommand(name string) string {
	names := make([]string, 0, len(a.commands))
	for _, c := range a.commands {
		names = append(names, c.Name)
	}
	return Suggest(name, names)
}

// Suggest returns the candidate a mistyped name most likely meant: the first
// candidate that name is a prefix of, otherwise the closest candidate within
// an edit distance of 2 (case-insensitive). It returns "" when none is close.
func Suggest(name string, candidates []string) string {
	best := ""
	bestDist := 3
	lname := strings.ToLower(name)
	for _, c := range candidates {
		lc := strings.ToLower(c)
		if lname != "" && strings.HasPrefix(lc, lname) {
			return c
		}
		if d := levenshtein(lname, lc); d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func (a *App) printHelp() { a.printHelpMode(false) }

// printHelpMode prints general help. With core commands registered it uses
// progressive disclosure: core commands in full, every other command by name
// under its group. With all set, every command is listed in full.
func (a *App) printHelpMode(all bool) {
	t := a.terminal
	dim := func(s string) string { return t.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }
	title := func(s string) string {
		return t.Paint(s, term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
	}
	// Banner
	name := t.Paint(a.name, term.ColorCyan, []term.Style{term.StyleBold})
	a.stderr.Write([]byte(name + " " + dim("v"+a.version) + "\n"))
	if a.description != "" {
		a.stderr.Write([]byte(dim(a.description) + "\n"))
	}
	a.stderr.Write([]byte("\n"))

	// Usage
	a.stderr.Write([]byte(title("USAGE") + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name, term.ColorCyan, nil) + " " + t.Paint("<command>", term.ColorYellow, nil) + " " + dim("[options]") + "\n"))
	a.stderr.Write([]byte("  " + dim(a.name+" help [command]") + dim("          canonical") + "\n"))
	a.stderr.Write([]byte("  " + dim(a.name+" help --all") + dim("              every command") + "\n"))
	a.stderr.Write([]byte("  " + dim(a.name+" --help | -h") + dim("              flag alias") + "\n"))
	a.stderr.Write([]byte("\n"))

	var core []*Command
	maxLen := len("help")
	for _, c := range a.commands {
		if c.Hidden {
			continue
		}
		if c.Core {
			core = append(core, c)
		}
		maxLen = max(maxLen, len(c.Name))
	}
	line := func(cmd *Command) {
		about := cmd.About
		if about == "" {
			about = dim("—")
		}
		about, _, _ = strings.Cut(about, "\n")
		l := "    " + t.Paint(cmd.Name, term.ColorCyan, nil) + strings.Repeat(" ", maxLen-len(cmd.Name)) + "  " + about
		if cmd.Example != "" {
			l += "\n      " + dim("→ "+cmd.Example)
		}
		a.stderr.Write([]byte(l + "\n"))
	}

	if len(core) > 0 {
		a.stderr.Write([]byte(title("CORE WORKFLOW") + "\n"))
		for _, c := range core {
			line(c)
		}
		a.stderr.Write([]byte("\n"))
	}

	groups := a.groupedCommands(len(core) > 0)
	if len(groups) > 0 {
		heading := "COMMANDS"
		if len(core) > 0 {
			heading = "ADVANCED CAPABILITIES"
		}
		a.stderr.Write([]byte(title(heading) + "\n"))
		groupWidth := 0
		for _, g := range groups {
			groupWidth = max(groupWidth, len(g.name))
		}
		for _, g := range groups {
			if !all && len(core) > 0 {
				names := make([]string, len(g.cmds))
				for i, c := range g.cmds {
					names[i] = t.Paint(c.Name, term.ColorCyan, nil)
				}
				label := firstNonEmptyStr(g.name, "other")
				a.stderr.Write([]byte("  " + dim(fmt.Sprintf("%-*s", groupWidth, strings.ToUpper(label))) + "  " + strings.Join(names, dim(" · ")) + "\n"))
				continue
			}
			if g.name != "" {
				a.stderr.Write([]byte("  " + dim(strings.ToUpper(g.name)) + "\n"))
			}
			for _, cmd := range g.cmds {
				line(cmd)
			}
			a.stderr.Write([]byte("\n"))
		}
		if !all && len(core) > 0 {
			a.stderr.Write([]byte("\n"))
		}
	}
	a.stderr.Write([]byte("  " + t.Paint("help", term.ColorCyan, nil) + strings.Repeat(" ", maxLen-4) + "  " + dim("Show help") + "\n\n"))

	// Footer
	a.stderr.Write([]byte(dim("Run ") + t.Paint(a.name+" help <command>", term.ColorCyan, nil) + dim(" or ") + t.Paint(a.name+" <command> --help", term.ColorCyan, nil) + dim(" for details.") + "\n"))
	if !all && len(core) > 0 {
		a.stderr.Write([]byte(dim("Run ") + t.Paint(a.name+" help --all", term.ColorCyan, nil) + dim(" to describe every command.") + "\n"))
	}
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

type cmdGroup struct {
	name string
	cmds []*Command
}

func (a *App) groupedCommands(skipCore bool) []cmdGroup {
	// Preserve insertion order of groups as first seen, but sort ungrouped last
	seen := map[string]int{}
	var groups []cmdGroup
	for _, c := range a.commands {
		if c.Hidden || (skipCore && c.Core) {
			continue
		}
		g := c.Group
		if g == "" {
			g = "other"
		}
		if idx, ok := seen[g]; ok {
			groups[idx].cmds = append(groups[idx].cmds, c)
		} else {
			seen[g] = len(groups)
			groups = append(groups, cmdGroup{name: g, cmds: []*Command{c}})
		}
	}
	// Ensure deterministic order for known groups
	order := map[string]int{
		"project": 0,
		"inspect": 1,
		"build":   2,
		"develop": 3,
		"ship":    4,
		"other":   5,
	}
	sort.Slice(groups, func(i, j int) bool {
		oi, okI := order[groups[i].name]
		oj, okJ := order[groups[j].name]
		if okI && okJ {
			return oi < oj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return groups[i].name < groups[j].name
	})
	// hide "other" label
	for i := range groups {
		if groups[i].name == "other" {
			groups[i].name = ""
		}
	}
	return groups
}

func (a *App) printCommandHelp(cmd *Command) {
	t := a.terminal
	title := t.Paint(a.name+" "+cmd.Name, term.ColorCyan, []term.Style{term.StyleBold})
	a.stderr.Write([]byte(title + "\n"))
	if cmd.About != "" {
		a.stderr.Write([]byte(cmd.About + "\n"))
	}
	if cmd.Example != "" {
		a.stderr.Write([]byte("\n" + t.Paint("Example:", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim}) + "\n"))
		a.stderr.Write([]byte("  " + t.Paint(cmd.Example, term.ColorGreen, nil) + "\n"))
	}
	a.stderr.Write([]byte("\n"))
	usageTitle := t.Paint("USAGE", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
	a.stderr.Write([]byte(usageTitle + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" "+cmd.Name, term.ColorCyan, nil) + " " + t.Paint("[options]", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" help "+cmd.Name, term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint("          canonical", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" "+cmd.Name+" --help | -h", term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint("    flag alias", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("\n"))

	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if cmd.Register != nil {
		cmd.Register(fs)
	}
	hasFlags := false
	fs.VisitAll(func(f *flag.Flag) { hasFlags = true })
	if hasFlags {
		optsTitle := t.Paint("OPTIONS", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
		a.stderr.Write([]byte(optsTitle + "\n"))
		// Custom flag printing for aligned, colored output
		fs.VisitAll(func(f *flag.Flag) {
			defVal := ""
			if f.DefValue != "" && f.DefValue != "false" {
				defVal = t.Paint(" (default "+f.DefValue+")", term.ColorDefault, []term.Style{term.StyleDim})
			}
			name := t.Paint("--"+f.Name, term.ColorYellow, nil)
			if f.DefValue == "false" {
				// boolean flag
				name = t.Paint("--"+f.Name, term.ColorYellow, nil)
			}
			a.stderr.Write([]byte(fmt.Sprintf("  %-22s %s%s\n", name, f.Usage, defVal)))
		})
	} else {
		a.stderr.Write([]byte(t.Paint("No options.", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	}
	if cmd.Example != "" {
		a.stderr.Write([]byte("\n" + t.Paint("Try:", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim}) + " " + t.Paint(cmd.Example, term.ColorGreen, nil) + "\n"))
	}
}
