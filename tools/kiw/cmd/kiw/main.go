// The kiw devtool: composes the Krewire ecosystem from the command line.
// Binary is named "kiw" for fast typing; module remains github.com/krewire/kiw.
package main

import (
	"os"

	"github.com/krewire/krewire/pkg/tui"
	"github.com/krewire/krewire/pkg/version"
	"github.com/krewire/krewire/tools/kiw/internal/commands"
	kiwver "github.com/krewire/krewire/tools/kiw/internal/version"
)

// Signature is the ecosystem watermark embedded into the binary at compile time via -ldflags.
var Signature = version.DefaultSignature

func main() {
	app := tui.NewApp("kiw", kiwver.VersionString()).
		WithDescription("One CLI for every workload — site, book, app, worker, service, gateway, infra").
		Command(tui.NewCommand("version", "print CLI and framework versions", nil, commands.RunVersion).WithGroup("inspect").WithExample("kiw version")).
		Command(tui.NewCommand("info", "print environment and project information", nil, commands.RunInfo).WithGroup("inspect").WithExample("kiw info")).
		Command(tui.NewCommand("compat", "check version compatibility across all modules", nil, commands.RunCompat).WithGroup("inspect").WithExample("kiw compat")).
		Command(tui.NewCommand("tier", "inspect open-core tier posture and validate boundary rules", commands.RegisterTier, commands.RunTier).WithGroup("inspect").WithExample("kiw tier --check free")).
		Command(tui.NewCommand("config", "validate krewire.yaml against its JSON Schema (line:col, did-you-mean), or print the schema", commands.RegisterConfig, commands.RunConfig).WithGroup("inspect").WithExample("kiw config validate --strict")).
		Command(tui.NewCommand("doctor", "diagnose developer environment, toolchain prerequisites, and workspace", commands.RegisterDoctor, commands.RunDoctor).WithGroup("inspect").WithExample("kiw doctor")).
		Command(tui.NewCommand("self-update", "update the kiw devtool to the latest stable release", commands.RegisterSelfUpdate, commands.RunSelfUpdate).WithGroup("inspect").WithExample("kiw self-update --check")).
		Command(tui.NewCommand("new", "create a project in a new directory (app by default; --site, --book, --cli, …); asks for name and kind when run bare", commands.RegisterNew, commands.RunNew).WithGroup("project").WithCore().WithExample("kiw new my-site --site")).
		Command(tui.NewCommand("init", "set up Krewire in the current directory, optionally equipping a kind (--site, --book, …); asks for the kind when run bare", commands.RegisterInit, commands.RunInit).WithGroup("project").WithExample("kiw init --site")).
		Command(tui.NewCommand("dev", "live development server: watches files, rebuilds, and hot-reloads sites/books or restarts apps and services", commands.RegisterDev, commands.RunDev).WithGroup("develop").WithCore().WithExample("kiw dev")).
		Command(tui.NewCommand("build", "build the project for production (site, book, app, cli, worker, service, gateway, infra)", commands.RegisterBuild, commands.RunBuild).WithGroup("build").WithCore().WithExample("kiw build")).
		Command(tui.NewCommand("bundle", "bundle static CSS and JS into assets/krewire.css and assets/krewire.js", commands.RegisterBundle, commands.RunBundle).WithGroup("build").WithExample("kiw bundle --minify")).
		Command(tui.NewCommand("serve", "preview production output without watching: serves the built site/book from .krewire/build (run 'kiw build' first); apps and CLIs run as with 'kiw run'", commands.RegisterServe, commands.RunServe).WithGroup("build").WithExample("kiw build && kiw serve --addr :3000")).
		Command(tui.NewCommand("run", "run once without watching: build and execute the app or CLI binary, a Go file, or a krewire.yaml script\n\nModes:\n  kiw run [args...]         build and run project app/cli binary\n  kiw run <file.go> [args]  execute standalone Go file with project environment\n  kiw run <task>            run a named task defined under scripts in krewire.yaml", commands.RegisterRun, commands.RunRun).WithGroup("develop").WithExample("kiw run [task|path/to/file.go] [-- args]")).
		Command(tui.NewCommand("test", "run the tests of the current project", commands.RegisterTest, commands.RunTest).WithGroup("develop").WithCore().WithExample("kiw test")).
		Command(tui.NewCommand("vet", "run go vet, security posture audit, and vulnerability checks", commands.RegisterVet, commands.RunVet).WithGroup("develop").WithExample("kiw vet --sec --vuln")).
		Command(tui.NewCommand("fmt", "check formatting with gofmt/go fmt", commands.RegisterFmt, commands.RunFmt).WithGroup("develop").WithExample("kiw fmt --write")).
		Command(tui.NewCommand("add", "add a package or plugin (package@version), or copy kui components into ./components (ui/<name>, ui:<pack>)", commands.RegisterAdd, commands.RunAdd).WithGroup("project").WithCore().WithExample("kiw add ui/button ui:minimal")).
		Command(tui.NewCommand("remove", "remove a package or plugin", commands.RegisterRemove, commands.RunRemove).WithGroup("project").WithExample("kiw remove twcss")).
		Command(tui.NewCommand("update", "update project dependencies to their latest versions", commands.RegisterUpdate, commands.RunUpdate).WithGroup("project").WithExample("kiw update")).
		Command(tui.NewCommand("deploy", "stage deployable artifacts in .krewire/dist/", commands.RegisterDeploy, commands.RunDeploy).WithGroup("ship").WithExample("kiw deploy --dry-run")).
		Command(tui.NewCommand("worker", "run background workers / job queues", commands.RegisterWorker, commands.RunWorker).WithGroup("develop").WithExample("kiw worker")).
		Command(tui.NewCommand("dashboard", "local dev dashboard (services, logs, traces)", commands.RegisterDashboard, commands.RunDashboard).WithGroup("develop").WithExample("kiw dashboard")).
		Command(tui.NewCommand("gateway", "run API gateway reverse proxy for microservices", commands.RegisterGateway, commands.RunGateway).WithGroup("develop").WithExample("kiw gateway --port 8080")).
		Command(tui.NewCommand("mcp", "native Model Context Protocol server for AI coding agents", commands.RegisterMCP, commands.RunMCP).WithGroup("develop").WithExample("kiw mcp --stdio")).
		Command(tui.NewCommand("ws", "workspace for monorepo, multi-repo, multi-project, microservices", commands.RegisterWs, commands.RunWs).WithGroup("ship").WithExample("kiw ws help")).
		Command(tui.NewCommand("boost", "manage Boost AI agent template (install, status)", commands.RegisterBoost, commands.RunBoost).WithGroup("ship").WithExample("kiw boost status")).
		Command(tui.NewCommand("release", "bump versions and stage a release across modules", commands.RegisterRelease, commands.RunRelease).WithGroup("ship").WithExample("kiw release framework --bump minor --apply")).
		Command(tui.NewCommand("make", "generate a new component, page, handler, migration, model, service, job, or chapter", commands.RegisterMake, commands.RunMake).WithGroup("project").WithCore().WithExample("kiw make component Card")).
		Command(tui.NewCommand("migrate", "apply, roll back, or list SQL migrations: up [n], down [n], redo, status, create <name>", commands.RegisterMigrate, commands.RunMigrate).WithGroup("project").WithExample("kiw migrate up")).
		Command(tui.NewCommand("ui", "browse, install, and diff kui components: list, info <name>, add <name|pack:x>, diff <name>", commands.RegisterUI, commands.RunUI).WithGroup("project").WithExample("kiw ui list")).
		Command(tui.NewCommand("kiw", "easter egg: playful client catcalls to developers", commands.RegisterKiw, commands.RunKiw).WithHidden(true).WithExample("kiw kiw"))
	for name, hint := range commands.Hints {
		app.WithHint(name, hint)
	}

	os.Exit(int(app.Run(os.Args[1:])))
}
