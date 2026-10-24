package commands

import (
	"flag"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
)

func TestRunKrew_Help(t *testing.T) {
	fs := flag.NewFlagSet("krew", flag.ContinueOnError)
	RegisterKrew(fs)
	_ = fs.Parse([]string{"help"})

	code := RunKrew(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("expected ExitCodeSuccess for help, got %v", code)
	}
}

func TestRunKrew_Version(t *testing.T) {
	fs := flag.NewFlagSet("krew", flag.ContinueOnError)
	RegisterKrew(fs)
	_ = fs.Parse([]string{"version"})

	code := RunKrew(fs)
	if code != errs.ExitCodeSuccess {
		t.Fatalf("expected ExitCodeSuccess for version, got %v", code)
	}
}

func TestRunKrew_SuggestNoArgs(t *testing.T) {
	fs := flag.NewFlagSet("krew", flag.ContinueOnError)
	RegisterKrew(fs)
	_ = fs.Parse([]string{"suggest"})

	code := RunKrew(fs)
	if code != errs.ExitCodeUsage {
		t.Fatalf("expected ExitCodeUsage when suggest has no args, got %v", code)
	}
}

func TestRunKrew_ExplainNoArgs(t *testing.T) {
	fs := flag.NewFlagSet("krew", flag.ContinueOnError)
	RegisterKrew(fs)
	_ = fs.Parse([]string{"explain"})

	code := RunKrew(fs)
	if code != errs.ExitCodeUsage {
		t.Fatalf("expected ExitCodeUsage when explain has no args, got %v", code)
	}
}

func TestRunKrew_NoProviderFailsGracefully(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OLLAMA_HOST", "")

	fs := flag.NewFlagSet("krew", flag.ContinueOnError)
	RegisterKrew(fs)
	_ = fs.Parse([]string{"suggest", "how to build"})

	code := RunKrew(fs)
	if code != errs.ExitCodeFailure {
		t.Fatalf("expected ExitCodeFailure when no provider env configured, got %v", code)
	}
}
