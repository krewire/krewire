package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
)

func disclosureApp(buf *bytes.Buffer) *App {
	app := NewApp("demo", "0.1.0").
		Command(NewCommand("new", "create a project", nil, successHandler).WithGroup("project").WithCore().WithExample("demo new app")).
		Command(NewCommand("dev", "live server", nil, successHandler).WithGroup("develop").WithCore()).
		Command(NewCommand("init", "set up in place", nil, successHandler).WithGroup("project").WithExample("demo init --site")).
		Command(NewCommand("tier", "inspect tiers", nil, successHandler).WithGroup("inspect")).
		WithHint("start", "demo dev")
	app.stderr = buf
	app.terminal = &term.Terminal{}
	return app
}

func TestHelpProgressiveDisclosure(t *testing.T) {
	var buf bytes.Buffer
	app := disclosureApp(&buf)
	if code := app.Run([]string{"help"}); code != errs.ExitCodeSuccess {
		t.Fatalf("code %v", code)
	}
	out := buf.String()
	for _, want := range []string{"CORE WORKFLOW", "new   create a project", "→ demo new app", "ADVANCED CAPABILITIES", "PROJECT  init", "INSPECT  tier", "demo help --all"} {
		if !strings.Contains(out, want) {
			t.Errorf("general help missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "set up in place") || strings.Contains(out, "demo init --site") {
		t.Errorf("advanced commands should be listed by name only:\n%s", out)
	}
	if strings.Index(out, "CORE WORKFLOW") > strings.Index(out, "ADVANCED CAPABILITIES") {
		t.Error("core section must come first")
	}
}

func TestHelpAllListsEverything(t *testing.T) {
	for _, args := range [][]string{{"help", "--all"}, {"help", "-a"}, {"--help", "--all"}, {"--all"}} {
		var buf bytes.Buffer
		app := disclosureApp(&buf)
		if code := app.Run(args); code != errs.ExitCodeSuccess {
			t.Fatalf("%v: code %v", args, code)
		}
		out := buf.String()
		for _, want := range []string{"CORE WORKFLOW", "set up in place", "→ demo init --site", "inspect tiers"} {
			if !strings.Contains(out, want) {
				t.Errorf("%v missing %q:\n%s", args, want, out)
			}
		}
		if strings.Contains(out, "help --all to describe") {
			t.Errorf("%v: full reference should not point to --all", args)
		}
	}
}

func TestHelpWithoutCoreKeepsFlatGroups(t *testing.T) {
	var buf bytes.Buffer
	app := NewApp("demo", "0.1.0").Command(NewCommand("ping", "ping the app", nil, successHandler))
	app.stderr, app.terminal = &buf, &term.Terminal{}
	app.Run([]string{"help"})
	if out := buf.String(); !strings.Contains(out, "COMMANDS") || !strings.Contains(out, "ping  ping the app") || strings.Contains(out, "CORE WORKFLOW") {
		t.Errorf("help:\n%s", out)
	}
}

func TestUnknownCommandHint(t *testing.T) {
	var buf bytes.Buffer
	app := disclosureApp(&buf)
	if code := app.Run([]string{"start"}); code != errs.ExitCodeUsage {
		t.Fatalf("code %v", code)
	}
	if out := buf.String(); !strings.Contains(out, `unknown command "start"`) || !strings.Contains(out, "did you mean demo dev") {
		t.Errorf("output:\n%s", out)
	}
}

func TestSuggest(t *testing.T) {
	cands := []string{"build", "bundle", "serve"}
	for in, want := range map[string]string{"buidl": "build", "bun": "bundle", "SERVE": "serve", "zzzzzz": ""} {
		if got := Suggest(in, cands); got != want {
			t.Errorf("Suggest(%q) = %q, want %q", in, got, want)
		}
	}
}
