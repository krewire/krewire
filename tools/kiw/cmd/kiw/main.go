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
		WithDescription("One CLI for every workload — site, book, app, worker, service, infra").
		Command(tui.NewCommand("version", "print CLI and framework versions", nil, commands.RunVersion).WithGroup("inspect").WithExample("kiw version")).
		Command(tui.NewCommand("info", "print environment and project information", nil, commands.RunInfo).WithGroup("inspect").WithExample("kiw info")).
		Command(tui.NewCommand("compat", "check version compatibility across all modules", nil, commands.RunCompat).WithGroup("inspect").WithExample("kiw compat")).
		Command(tui.NewCommand("tier", "inspect open-core tier posture and validate boundary rules", commands.RegisterTier, commands.RunTier).WithGroup("inspect").WithExample("kiw tier --check free")).
		Command(tui.NewCommand("new", "scaffold a minimal Krewire project kernel", commands.RegisterNew, commands.RunNew).WithGroup("project").WithExample("kiw new my-site")).
		Command(tui.NewCommand("init", "equip the project with a variant: monolith, --site, --book, --cli, --worker, --service, --infra, or --template", commands.RegisterInit, commands.RunInit).WithGroup("project").WithExample("kiw init --site")).
		Command(tui.NewCommand("build", "build the project for production (site, book, app, cli, worker, service, infra)", commands.RegisterBuild, commands.RunBuild).WithGroup("build").WithExample("kiw build")).
		Command(tui.NewCommand("bundle", "bundle static CSS and JS into assets/krewire.css and assets/krewire.js", commands.RegisterBundle, commands.RunBundle).WithGroup("build").WithExample("kiw bundle --minify")).
		Command(tui.NewCommand("serve", "preview the current project's website over HTTP", commands.RegisterServe, commands.RunServe).WithGroup("build").WithExample("kiw serve --port 3000")).
		Command(tui.NewCommand("run", "build and run the current app, a Go file, or a task from krewire.yaml\n\nModes:\n  kiw run [args...]         build and run project app/cli binary\n  kiw run <file.go> [args]  execute standalone Go file with project environment\n  kiw run <task>            run a named task defined under scripts in krewire.yaml", commands.RegisterRun, commands.RunRun).WithGroup("develop").WithExample("kiw run [task|path/to/file.go] [-- args]")).
		Command(tui.NewCommand("dev", "run the current app in dev mode with auto-restart", commands.RegisterDev, commands.RunDev).WithGroup("develop").WithExample("kiw dev")).
		Command(tui.NewCommand("test", "run the tests of the current project", commands.RegisterTest, commands.RunTest).WithGroup("develop").WithExample("kiw test")).
		Command(tui.NewCommand("vet", "run go vet, security posture audit, and vulnerability checks", commands.RegisterVet, commands.RunVet).WithGroup("develop").WithExample("kiw vet --sec --vuln")).
		Command(tui.NewCommand("fmt", "check formatting with gofmt/go fmt", commands.RegisterFmt, commands.RunFmt).WithGroup("develop").WithExample("kiw fmt --write")).
		Command(tui.NewCommand("add", "add a package or plugin (package@version, @latest for latest)", commands.RegisterAdd, commands.RunAdd).WithGroup("project").WithExample("kiw add twcss@latest")).
		Command(tui.NewCommand("remove", "remove a package or plugin", commands.RegisterRemove, commands.RunRemove).WithGroup("project").WithExample("kiw remove twcss")).
		Command(tui.NewCommand("deploy", "stage deployable artifacts in dist/", commands.RegisterDeploy, commands.RunDeploy).WithGroup("ship").WithExample("kiw deploy --preview")).
		Command(tui.NewCommand("worker", "run background workers / job queues", commands.RegisterWorker, commands.RunWorker).WithGroup("develop").WithExample("kiw worker")).
		Command(tui.NewCommand("krew", "autonomous AI coding agent (suggest, explain, chat, run, serve, mcp)", commands.RegisterKrew, commands.RunKrew).WithGroup("develop").WithExample("kiw krew \"periksa unit test\"")).
		Command(tui.NewCommand("dashboard", "local dev dashboard (services, logs, traces)", commands.RegisterDashboard, commands.RunDashboard).WithGroup("develop").WithExample("kiw dashboard")).
		Command(tui.NewCommand("gateway", "run API gateway reverse proxy for microservices", commands.RegisterGateway, commands.RunGateway).WithGroup("develop").WithExample("kiw gateway --port 8080")).
		Command(tui.NewCommand("mcp", "native Model Context Protocol server for AI coding agents", commands.RegisterMCP, commands.RunMCP).WithGroup("develop").WithExample("kiw mcp --stdio")).
		Command(tui.NewCommand("ws", "workspace for monorepo, multi-repo, multi-project, microservices", commands.RegisterWs, commands.RunWs).WithGroup("ship").WithExample("kiw ws help")).
		Command(tui.NewCommand("boost", "install the Boost AI agent template into a project", commands.RegisterBoost, commands.RunBoost).WithGroup("ship").WithExample("kiw boost install")).
		Command(tui.NewCommand("release", "bump versions and stage a release across modules", commands.RegisterRelease, commands.RunRelease).WithGroup("ship").WithExample("kiw release framework --bump minor --apply")).
		Command(tui.NewCommand("make", "scaffold components, pages, handlers, migrations", commands.RegisterMake, commands.RunMake).WithGroup("project").WithExample("kiw make component Card")).
		Command(tui.NewCommand("kiw", "easter egg: playful client catcalls to developers", commands.RegisterKiw, commands.RunKiw).WithHidden(true).WithExample("kiw kiw"))

	os.Exit(int(app.Run(os.Args[1:])))
}
