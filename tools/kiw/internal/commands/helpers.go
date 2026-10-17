package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	"github.com/krewire/krewire/tools/kiw/internal/gomod"
	"github.com/krewire/krewire/tools/kiw/internal/scaffold"
)

func flagValue(fs *flag.FlagSet, name string) string {
	if fs == nil {
		return ""
	}
	if f := fs.Lookup(name); f != nil {
		return f.Value.String()
	}
	return ""
}

func commandError(err error) kern.ExitCode {
	fmt.Fprint(os.Stderr, kern.FormatTree(err))
	switch {
	case errors.Is(err, scaffold.ErrInvalidName),
		errors.Is(err, scaffold.ErrProjectExists),
		errors.Is(err, scaffold.ErrNotEmpty),
		errors.Is(err, scaffold.ErrConflict):
		return kern.ExitCodeUsage
	default:
		return kern.ExitCodeFailure
	}
}

func fail(err error) kern.ExitCode {
	fmt.Fprint(os.Stderr, kern.FormatTree(err))
	printStackIfDebug(err)
	return kern.ExitCodeFailure
}

// printStackIfDebug renders an attached stack after the error line when the
// command resolved debug mode (KWL-P8W2N KWL-DIAGV-007).
func printStackIfDebug(err error) {
	if !debugEnabled {
		return
	}
	frames := kern.StackOf(err)
	if len(frames) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "stack trace (newest first):")
	fmt.Fprint(os.Stderr, kern.FormatStack(frames))
}

// usageOrFail prints err and maps core usage errors to exit code 2
// (KWL-K4T7W KWL-ENVV-007).
func usageOrFail(err error) kern.ExitCode {
	var ce interface{ ExitCode() kern.ExitCode }
	if errors.As(err, &ce) && ce.ExitCode() == kern.ExitCodeUsage {
		fmt.Fprint(os.Stderr, kern.FormatTree(err))
		printStackIfDebug(err)
		return kern.ExitCodeUsage
	}
	return fail(err)
}

// flagProvided reports whether the named flag was explicitly set on the
// command line, as opposed to holding its zero value.
func flagProvided(fs *flag.FlagSet, name string) bool {
	if fs == nil {
		return false
	}
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// truthStr renders a boolean as the KIW_DEBUG wire value.
func truthStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// defaultAddr is the fallback listen address when no address is specified.
const defaultAddr = ":8080"

// resolvedAddr resolves the listening address, preferring an explicit command-line
// flag over a declared port in krewire.yaml, falling back to defaultAddr (:8080)
// (FRK-CONF-031).
func resolvedAddr(flagAddr string, cfg *config.Config) string {
	if flagAddr != "" {
		return flagAddr
	}
	if cfg != nil && cfg.DeclaredPort() > 0 {
		return fmt.Sprintf(":%d", cfg.DeclaredPort())
	}
	return defaultAddr
}

// childEnviron composes the child-process environment for run and dev,
// exporting APP_ADDR plus KIW_ENV/KIW_DEBUG (KWL-K4T7W KWL-ENVV-006).
// Vein is applied: env via kern.Env.
func childEnviron(base []string, addr string, env kern.Env, debug bool) []string {
	e := append([]string(nil), base...)
	if addr != "" {
		e = append(e, "APP_ADDR="+addr)
	}
	return append(e, "KIW_ENV="+env.String(), "KIW_DEBUG="+truthStr(debug))
}

// firstNonEmpty returns the first non-empty string among the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// findRoot locates the project root walking up from the working directory.
// For app/cli/worker/service it is the Go module root (go.mod). For site/book
// which have no entry point, krewire.yaml or a declarative layout (pages/,
// content/, manuscript/) is sufficient (KWF-DF3PL FRK-FLS-001/002, KWL-K1N2Q).
func findRoot() (string, error) {
	cur, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur, err = filepath.Abs(cur)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil && !info.IsDir() {
			return cur, nil
		}
		if info, err := os.Stat(filepath.Join(cur, "krewire.yaml")); err == nil && !info.IsDir() {
			return cur, nil
		}
		if info, err := os.Stat(filepath.Join(cur, "pages")); err == nil && info.IsDir() {
			return cur, nil
		}
		for _, marker := range []string{"content", "manuscript"} {
			if info, err := os.Stat(filepath.Join(cur, marker)); err == nil && info.IsDir() {
				return cur, nil
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("not inside a Go module — run 'kiw new <project>' first")
		}
		cur = parent
	}
}

// joinRoot resolves a flag value relative to the project root, falling back to
// def when the flag is unset.
func joinRoot(root, value, def string) string {
	if value == "" {
		value = def
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

// hasDir reports whether path exists inside root as a directory.
func hasDir(root, path string) bool {
	info, err := os.Stat(filepath.Join(root, path))
	return err == nil && info.IsDir()
}

// hasFile reports whether path exists inside root as a regular file.
func hasFile(root, path string) bool {
	info, err := os.Stat(filepath.Join(root, path))
	return err == nil && !info.IsDir()
}

// moduleName returns the last path element of the module declared in root's
// go.mod.
func moduleName(root string) string {
	mod, err := gomod.Read(filepath.Join(root, "go.mod"))
	if err != nil || mod.Path == "" {
		return filepath.Base(root)
	}
	parts := strings.Split(strings.TrimSuffix(mod.Path, "/"), "/")
	return parts[len(parts)-1]
}

// moduleDir walks upward from dir to find the directory containing go.mod.
func moduleDir(dir string) string {
	cur := dir
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

// containsGoPackageArg reports whether args contains a package-like path argument.
func containsGoPackageArg(args []string) bool {
	for _, a := range args {
		if strings.Contains(a, "/") || strings.HasPrefix(a, "./") || a == "." || a == "..." {
			return true
		}
	}
	return false
}
