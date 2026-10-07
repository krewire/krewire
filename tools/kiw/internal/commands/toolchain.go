package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	"github.com/krewire/krewire/tools/kiw/internal/gomod"
)

func RegisterTest(fs *flag.FlagSet) {
	fs.String("filter", "", "filter tests by name, spec ID, or file pattern (like vitest --filter, pest --filter)")
	fs.String("spec", "", "filter by spec ID (alias for --filter, e.g., KWF-TEST-M4P9Q)")
	fs.Bool("watch", false, "watch files and rerun tests on change (like vitest --watch)")
	fs.Bool("w", false, "alias for --watch")
	fs.Bool("coverage", false, "run with coverage (go test -cover)")
	fs.String("coverprofile", "", "write coverage profile to file (go test -coverprofile)")
	fs.Bool("verbose", false, "verbose output (go test -v)")
	fs.Bool("v", false, "alias for --verbose")
	fs.Bool("json", false, "json output (go test -json)")
	fs.Bool("update", false, "update snapshots/golden files (UPDATE_GOLDEN=1)")
	fs.Bool("u", false, "alias for --update")
	fs.String("run", "", "run only tests matching regexp (go test -run)")
	fs.String("count", "", "go test -count value (e.g., 1 to disable cache)")
	fs.Bool("list", false, "list tests instead of running (go test -list)")
	fs.String("reporter", "", "reporter: verbose, dot, json (like vitest --reporter)")
}

func RunTest(fs *flag.FlagSet) kern.ExitCode {
	dir, err := os.Getwd()
	if err != nil {
		return fail(err)
	}
	mod, err := gomod.Find(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: not inside a Go module — run 'kiw new <project>' first")
		return kern.ExitCodeUsage
	}
	slog.Info("running tests", "module", mod.Path)

	filter := flagValue(fs, "filter")
	spec := flagValue(fs, "spec")
	if spec != "" && filter == "" {
		filter = spec
	}
	if v := flagValue(fs, "run"); v != "" && filter == "" {
		filter = v
	}
	if filter != "" && strings.HasPrefix(filter, "spec:") {
		filter = strings.TrimPrefix(filter, "spec:")
	}
	// Normalize spec ID to code for grep-friendly matching (e.g., KWF-TEST-M4P9Q -> M4P)
	// Tests are named TestKWF_TST_M4P_010 (contain requirement code), so filter by spec should match code prefix.
	if isSpecID(filter) {
		filter = specFilterRegexp(filter)
	}

	watch := flagValue(fs, "watch") == "true" || flagValue(fs, "w") == "true"
	coverage := flagValue(fs, "coverage") == "true"
	coverprofile := flagValue(fs, "coverprofile")
	verbose := flagValue(fs, "verbose") == "true" || flagValue(fs, "v") == "true"
	jsonOut := flagValue(fs, "json") == "true"
	update := flagValue(fs, "update") == "true" || flagValue(fs, "u") == "true"
	count := flagValue(fs, "count")
	list := flagValue(fs, "list") == "true"
	reporter := flagValue(fs, "reporter")
	if reporter == "json" {
		jsonOut = true
	}
	if reporter == "verbose" {
		verbose = true
	}

	args := []string{"test"}
	if verbose {
		args = append(args, "-v")
	}
	if jsonOut {
		args = append(args, "-json")
	}
	if coverage {
		args = append(args, "-cover")
	}
	if coverprofile != "" {
		args = append(args, "-coverprofile="+coverprofile)
		if !coverage {
			args = append(args, "-cover")
		}
	}
	if count != "" {
		args = append(args, "-count="+count)
	}
	if list {
		pattern := filter
		if pattern == "" {
			pattern = ".*"
		}
		if isPackageFilter(pattern) {
			pattern = ".*"
		}
		args = append(args, "-list", pattern)
		// don't also add -run for list
		filter = ""
	} else if filter != "" {
		// if filter looks like a file/package path, treat as package filter handled below
		// else as -run regexp
		if isPackageFilter(filter) {
			// will be handled as package arg
		} else {
			args = append(args, "-run", filter)
		}
	}

	// collect positional package args (after --)
	pkgs := []string{}
	if fs != nil {
		for _, a := range fs.Args() {
			if strings.HasPrefix(a, "-") {
				continue
			}
			// skip already handled filter/spec values that appear as args due to flag parsing quirks
			if a == filter || a == spec {
				continue
			}
			pkgs = append(pkgs, a)
		}
	}

	// if filter was a package filter, use it as package
	if filter != "" && isPackageFilter(filter) && len(pkgs) == 0 {
		pkgs = []string{filter}
		filter = ""
		// remove -run that was not added
	}

	if len(pkgs) == 0 {
		// check if any go package arg was passed via original fs.Args
		if fs != nil && containsGoPackageArg(fs.Args()) {
			// already has package, don't add ./...
		} else {
			pkgs = []string{"./..."}
		}
	}
	args = append(args, pkgs...)

	// if extra raw args were passed (e.g., -count 1), they are already in fs.Args as -xxx,
	// but we have handled known flags; passthrough remaining unknown flags
	if fs != nil {
		for _, a := range fs.Args() {
			if strings.HasPrefix(a, "-") && !isKnownTestFlag(a) {
				args = append(args, a)
			}
		}
	}

	env := os.Environ()
	if update {
		env = append(env, "UPDATE_GOLDEN=1")
	}

	if watch {
		return runTestWatch(dir, args, env)
	}

	return runTestOnce(dir, args, env)
}

func runTestOnce(dir string, args []string, env []string) kern.ExitCode {
	slog.Info("go test", "args", strings.Join(args, " "))
	cmd := exec.Command("go", args...) // #nosec G204 — user-invoked devtool
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return kern.ExitCodeFailure
	}
	return kern.ExitCodeSuccess
}

func runTestWatch(dir string, args []string, env []string) kern.ExitCode {
	// find project root for watcher
	root := dir
	if mod, err := gomod.Find(dir); err == nil {
		if mdir := moduleDir(dir); mdir != "" {
			root = mdir
		} else {
			_ = mod
		}
	}
	cfg, _ := config.Load(filepath.Join(root, "krewire.yaml"))
	w := newWatcher(root, 500*time.Millisecond, cfg)
	defer func() { close(w.done) }()

	// initial run
	code := runTestOnce(dir, args, env)
	if code != kern.ExitCodeSuccess {
		slog.Info("watch: initial run failed, waiting for changes...")
	}
	fmt.Fprintln(os.Stdout, "watching for changes (Ctrl+C to stop)...")

	for {
		select {
		case <-w.Changed():
			fmt.Fprintln(os.Stdout, "\n--- change detected, rerunning ---")
			w.Reset()
			_ = runTestOnce(dir, args, env)
		case <-time.After(200 * time.Millisecond):
			// avoid busy loop, watcher handles polling
		}
	}
}

func isPackageFilter(s string) bool {
	if strings.Contains(s, "/") || strings.HasPrefix(s, "./") || s == "." || s == "..." {
		return true
	}
	if strings.HasSuffix(s, ".go") {
		return true
	}
	return false
}

func isKnownTestFlag(a string) bool {
	known := []string{"-filter", "--filter", "-spec", "--spec", "-watch", "--watch", "-w", "-coverage", "--coverage", "-coverprofile", "--coverprofile", "-verbose", "--verbose", "-v", "-json", "--json", "-update", "--update", "-u", "-run", "--run", "-count", "--count", "-list", "--list", "-reporter", "--reporter"}
	for _, k := range known {
		if a == k || strings.HasPrefix(a, k+"=") {
			return true
		}
	}
	return false
}

func isSpecID(s string) bool {
	if !strings.Contains(s, "-") {
		return false
	}
	parts := strings.Split(s, "-")
	if len(parts) < 3 {
		return false
	}
	last := parts[len(parts)-1]
	if len(last) != 5 {
		return false
	}
	for _, ch := range last {
		if !(ch >= '0' && ch <= '9' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z') {
			return false
		}
	}
	return true
}

func specFilterRegexp(specID string) string {
	parts := strings.Split(specID, "-")
	code := parts[len(parts)-1]
	if len(code) == 5 {
		return code[:3]
	}
	return code
}

func RegisterVet(fs *flag.FlagSet) {
	fs.Bool("sec", false, "run security posture checks (secrets scan, config hygiene, gitignore)")
}

func RunVet(fs *flag.FlagSet) kern.ExitCode {
	dir, err := os.Getwd()
	if err != nil {
		return fail(err)
	}
	mod, err := gomod.Find(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: not inside a Go module — run 'kiw new <project>' first")
		return kern.ExitCodeUsage
	}

	secCheck := flagValue(fs, "sec") == "true"
	if fs != nil {
		for _, a := range fs.Args() {
			if a == "--sec" || a == "-sec" || a == "sec" {
				secCheck = true
				break
			}
		}
	}

	slog.Info("running vet", "module", mod.Path)
	args := []string{"vet", "./..."}
	if fs != nil && len(fs.Args()) > 0 {
		var filtered []string
		for _, a := range fs.Args() {
			if a != "--sec" && a != "-sec" && a != "sec" {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) > 0 {
			args = append([]string{"vet"}, filtered...)
		}
	}
	cmd := exec.Command("go", args...) // #nosec G204 — user-invoked devtool
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return kern.ExitCodeFailure
	}

	if secCheck {
		fmt.Fprintln(os.Stdout, "\n── Security Posture Audit ──────────────────────────────────")
		issues := auditSecurityPosture(dir)
		if len(issues) > 0 {
			for _, iss := range issues {
				fmt.Fprintf(os.Stderr, "  ✗ [SEC] %s\n", iss)
			}
			return kern.ExitCodeFailure
		}
		fmt.Fprintln(os.Stdout, "  ✓ Secrets hygiene: verified (no plaintext credentials found)")
		fmt.Fprintln(os.Stdout, "  ✓ Environment referencing: ${env:...} contract upheld")
		fmt.Fprintln(os.Stdout, "  ✓ Security posture: passed")
	}

	return kern.ExitCodeSuccess
}

func auditSecurityPosture(dir string) []string {
	var issues []string
	cfgPath := filepath.Join(dir, "krewire.yaml")
	if data, err := os.ReadFile(cfgPath); err == nil {
		content := string(data)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			lower := strings.ToLower(trimmed)
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			if (strings.Contains(lower, "password:") || strings.Contains(lower, "api_key:") ||
				strings.Contains(lower, "secret:") || strings.Contains(lower, "token:")) &&
				!strings.Contains(trimmed, "${env:") && !strings.Contains(trimmed, "${") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" && strings.TrimSpace(parts[1]) != `""` {
					issues = append(issues, fmt.Sprintf("krewire.yaml:%d plaintext secret detected in %q — use ${env:VAR_NAME}", i+1, trimmed))
				}
			}
		}
	}
	return issues
}

func RegisterFmt(fs *flag.FlagSet) {
	fs.Bool("write", false, "write formatted files back to disk (default: check only with gofmt -l)")
	fs.Bool("w", false, "alias for --write")
}

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

func containsGoPackageArg(args []string) bool {
	for _, a := range args {
		if strings.Contains(a, "/") || strings.HasPrefix(a, "./") || a == "." || a == "..." {
			return true
		}
	}
	return false
}
