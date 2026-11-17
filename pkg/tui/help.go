package tui

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/krewire/krewire/pkg/term"
)

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
