package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/krewire/krewire/packages/kern"
	"github.com/krewire/krewire/tools/kiw/internal/config"
)

// debugEnabled records whether the current command resolved debug mode, so
// low-level printers can include stack traces (KWL-P8W2N KWL-DIAGV-007).
var debugEnabled bool

// runtimeEnv is the resolved local-run context shared by serve, run, and dev
// (KWN-6K41E RND-SRV-002): the module root, its configuration, and the
// effective environment/debug switches (KWL-K4T7W).
// Vein is applied: env and diagnostics via kern.
type runtimeEnv struct {
	root  string
	cfg   *config.Config
	env   kern.Env
	debug bool
}

// bootRuntime resolves the module root, loads krewire.yaml and .env, and
// resolves env/debug with strict precedence: flag > KIW_ENV/KIW_DEBUG >
// krewire.yaml > default. The returned code is kern.ExitCodeSuccess or a
// terminal failure/usage code.
func bootRuntime(fs *flag.FlagSet) (*runtimeEnv, kern.ExitCode) {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return nil, kern.ExitCodeUsage
	}
	c, err := config.Load(root)
	if err != nil {
		return nil, fail(kern.WithStack(err))
	}
	if err := c.LoadDotEnv(root); err != nil {
		return nil, fail(kern.WithStack(err))
	}
	env, err := c.ResolveEnv(flagValue(fs, "env"))
	if err != nil {
		return nil, usageOrFail(err)
	}
	debug := c.ResolveDebug(flagValue(fs, "debug"), flagProvided(fs, "debug"))
	debugEnabled = debug
	kern.InstallLogger(kern.Env(env), debug)
	return &runtimeEnv{root: root, cfg: c, env: kern.Env(env), debug: debug}, kern.ExitCodeSuccess
}

// registerRuntimeFlags registers the flags shared by every command that
// boots a project locally.
func registerRuntimeFlags(fs *flag.FlagSet) {
	fs.String("kind", "", "force project kind: app, cli, site, or book (default auto)")
	fs.String("env", "", "target environment: local, production, or testing (default: krewire.yaml, then KIW_ENV)")
	fs.Bool("debug", false, "enable debug mode (default: krewire.yaml, then KIW_DEBUG)")
}
