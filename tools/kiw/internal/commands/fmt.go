package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/gomod"
)

// RegisterFmt configures flags for the `kiw fmt` command.
func RegisterFmt(fs *flag.FlagSet) {
	fs.Bool("write", false, "write formatted files back to disk (default: check only with gofmt -l)")
	fs.Bool("w", false, "alias for --write")
}

// RunFmt checks or fixes formatting with gofmt.
func RunFmt(fs *flag.FlagSet) kern.ExitCode {
	dir, err := os.Getwd()
	if err != nil {
		return fail(err)
	}
	root := dir
	if mod, err := gomod.Find(dir); err == nil {
		if mdir := moduleDir(dir); mdir != "" {
			root = mdir
		} else {
			_ = mod
		}
	}
	write := flagValue(fs, "write") == "true" || flagValue(fs, "w") == "true"
	if fs != nil {
		for _, a := range fs.Args() {
			if a == "--write" || a == "-w" || a == "write" {
				write = true
				break
			}
		}
	}
	if write {
		slog.Info("formatting code", "root", root, "mode", "write")
		cmd := exec.Command("gofmt", "-w", ".") // #nosec G204 — user-invoked devtool
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fallback := exec.Command("go", "fmt", "./...") // #nosec G204
			fallback.Dir = root
			fallback.Stdout = os.Stdout
			fallback.Stderr = os.Stderr
			if ferr := fallback.Run(); ferr != nil {
				return fail(err)
			}
			return kern.ExitCodeSuccess
		}
		return kern.ExitCodeSuccess
	}
	slog.Info("checking format", "root", root)
	cmd := exec.Command("gofmt", "-l", ".") // #nosec G204 — user-invoked devtool
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			fmt.Fprint(os.Stdout, string(out))
		}
		return fail(err)
	}
	if len(out) > 0 {
		fmt.Fprint(os.Stdout, string(out))
		fmt.Fprintln(os.Stderr, "kiw fmt: some files need formatting — run 'kiw fmt --write' or 'gofmt -w .'")
		return kern.ExitCodeFailure
	}
	return kern.ExitCodeSuccess
}
