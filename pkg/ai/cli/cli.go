package cli

import (
	"fmt"
	"os"

	"github.com/krewire/krewire/pkg/version"
)

// Run executes the AI CLI command with the given arguments.
func Run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 0
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "chat":
		RunChat(cmdArgs)
	case "agent", "run":
		RunSingleShot(cmdArgs)
	case "daemon", "serve":
		RunServe(cmdArgs)
	case "sessions":
		RunSessions(cmdArgs)
	case "export":
		RunExport(cmdArgs)
	case "mcp":
		RunMCP(cmdArgs)
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Println("ai " + version.Build().String())
	default:
		fmt.Fprintf(os.Stderr, "unknown ai command %q. Run 'ai help' for usage.\n", cmd)
		return 1
	}
	return 0
}

func printUsage() {
	fmt.Println(`Krewire AI - Autonomous AI Agent Engine

Usage:
  ai <command> [arguments]

Available Commands:
  chat       Start an interactive AI coding session in the terminal
  agent      Execute an autonomous task or prompt headless to completion
  daemon     Launch HTTP/REST and Server-Sent Events (SSE) server for remote hosting
  sessions   List past conversation sessions with message and token counts
  export     Export conversation session transcript to markdown
  mcp        Run native Model Context Protocol (MCP) server for agents
  version    Print the release version, source commit, and build metadata

Flags:
  -h, --help     Show help for command
  -v, --version  Print version and build metadata`)
}
