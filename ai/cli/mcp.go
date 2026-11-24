package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/krewire/krewire/ai/embed"
	"github.com/krewire/krewire/ai/transport"
)

// RunMCP starts the MCP server over stdin/stdout.
func RunMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	personaFlag := fs.String("persona", DefaultPersonaName, "Agent persona")
	workspaceFlag := fs.String("workspace", DefaultWorkspaceDir, "Workspace directory")
	_ = fs.Parse(args)

	pers := ParsePersona(*personaFlag)

	inst, err := embed.New(embed.Options{
		Workspace: *workspaceFlag,
		Persona:   &pers,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing agent for MCP: %v\n", err)
		os.Exit(1)
	}

	mcpServer := transport.NewMCPServer(inst.Agent(), os.Stdin, os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := mcpServer.Serve(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server exited: %v\n", err)
		os.Exit(1)
	}
}
