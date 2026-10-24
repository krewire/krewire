package commands

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/krewire/krew"
	"github.com/krewire/krew/pkg/agent"
	"github.com/krewire/krew/pkg/embed"
	"github.com/krewire/krew/pkg/persona"
	"github.com/krewire/krew/pkg/transport"
	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
)

var (
	krewFlagPersona  string
	krewFlagModel    string
	krewFlagMaxSteps int
	krewFlagPort     int
	krewFlagHost     string
	krewFlagDir      string
)

// RegisterKrew registers flags for the "kiw krew" command.
func RegisterKrew(fs *flag.FlagSet) {
	fs.StringVar(&krewFlagPersona, "persona", "coding", "agent persona (coding, devops, worker, general)")
	fs.StringVar(&krewFlagModel, "model", "", "model name override (e.g. gemini-2.5-flash, gpt-4o)")
	fs.IntVar(&krewFlagMaxSteps, "max-steps", 25, "maximum reasoning steps per turn")
	fs.IntVar(&krewFlagPort, "port", 8080, "port for background serve daemon")
	fs.StringVar(&krewFlagHost, "host", "0.0.0.0", "host address for background serve daemon")
	fs.StringVar(&krewFlagDir, "dir", ".", "workspace directory root")
}

// RunKrew executes the Krew AI coding agent inside the kiw CLI.
func RunKrew(fs *flag.FlagSet) errs.ExitCode {
	args := fs.Args()
	if len(args) == 0 {
		return printKrewHelp()
	}

	sub := args[0]
	switch sub {
	case "help", "-h", "--help":
		return printKrewHelp()
	case "version", "-v", "--version":
		tm := term.NewTerminal()
		bold := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
		green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
		fmt.Printf("%s %s (krew %s)\n", bold("kiw krew"), green("AI Coding Assistant"), krew.Version)
		return errs.ExitCodeSuccess
	case "suggest":
		return runKrewSuggest(args[1:])
	case "explain":
		return runKrewExplain(args[1:])
	case "chat":
		return runKrewChat(args[1:])
	case "run":
		return runKrewRun(args[1:])
	case "serve":
		return runKrewServe(args[1:])
	case "mcp":
		return runKrewMCP(args[1:])
	default:
		// Direct prompt invocation: kiw krew "what is the purpose of this repo?"
		return runKrewDirect(strings.Join(args, " "))
	}
}

func printKrewHelp() errs.ExitCode {
	tm := term.NewTerminal()
	boldCyan := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
	yellow := func(s string) string { return tm.Paint(s, term.ColorYellow, nil) }
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }

	fmt.Printf("\n  %s — %s\n\n", boldCyan("kiw krew"), green("The Krewire In-CLI Autonomous AI Companion"))
	fmt.Printf("  %s\n", dim("Embeds the platform-agnostic Krew agent engine directly into your developer workflow."))
	fmt.Println()
	fmt.Printf("  %s\n", boldCyan("USAGE:"))
	fmt.Printf("    kiw krew <command> [options]\n")
	fmt.Printf("    kiw krew \"<prompt>\" [options]\n\n")

	fmt.Printf("  %s\n", boldCyan("COMMANDS:"))
	fmt.Printf("    %-20s %s\n", yellow("suggest <task>"), "Suggest terminal commands or code for a goal (like 'gh copilot suggest')")
	fmt.Printf("    %-20s %s\n", yellow("explain <cmd|code>"), "Explain shell commands, errors, or code structures (like 'gh copilot explain')")
	fmt.Printf("    %-20s %s\n", yellow("chat"), "Interactive conversational coding assistant session in terminal")
	fmt.Printf("    %-20s %s\n", yellow("run <prompt>"), "Execute an autonomous coding or refactoring task headless")
	fmt.Printf("    %-20s %s\n", yellow("serve"), "Start background HTTP/SSE daemon for remote IDE or webhook calls")
	fmt.Printf("    %-20s %s\n", yellow("mcp"), "Start standard Model Context Protocol (stdio) server")
	fmt.Printf("    %-20s %s\n", yellow("version"), "Print Krew agent engine version")
	fmt.Println()

	fmt.Printf("  %s\n", boldCyan("EXAMPLES:"))
	fmt.Printf("    %s\n", dim("kiw krew suggest \"how to build site with minified assets\""))
	fmt.Printf("    %s\n", dim("kiw krew explain \"kiw bundle --minify\""))
	fmt.Printf("    %s\n", dim("kiw krew chat"))
	fmt.Printf("    %s\n", dim("kiw krew \"periksa dan tambahkan unit test untuk pkg/assets\""))
	fmt.Println()
	return errs.ExitCodeSuccess
}

func resolvePersona(name string) persona.Persona {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "devops", "ops", "sre":
		return persona.DevOpsRunner
	case "worker", "autonomous":
		return persona.AutonomousWorker
	case "general", "assistant":
		return persona.GeneralAssistant
	case "coding", "code", "dev":
		fallthrough
	default:
		return persona.CodingAssistant
	}
}

func initKrewInstance(customSysPrompt string) (*embed.Instance, errs.ExitCode) {
	pers := resolvePersona(krewFlagPersona)
	tm := term.NewTerminal()
	red := func(s string) string { return tm.Paint(s, term.ColorRed, []term.Style{term.StyleBold}) }
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }

	opts := embed.Options{
		Workspace:    krewFlagDir,
		Persona:      &pers,
		SystemPrompt: customSysPrompt,
		Model:        krewFlagModel,
		MaxSteps:     krewFlagMaxSteps,
		OnEvent:      krewEventHandler,
	}

	inst, err := embed.New(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s %v\n\n", red("kiw krew error:"), err)
		fmt.Fprintf(os.Stderr, "  %s\n", dim("To use Krew AI, configure one of the following environment variables:"))
		fmt.Fprintf(os.Stderr, "    export GEMINI_API_KEY=\"...\"    # Recommended (Gemini 2.5 Flash / Pro)\n")
		fmt.Fprintf(os.Stderr, "    export OPENAI_API_KEY=\"...\"    # OpenAI (GPT-4o)\n")
		fmt.Fprintf(os.Stderr, "    export ANTHROPIC_API_KEY=\"...\" # Anthropic (Claude 3.7 Sonnet)\n")
		fmt.Fprintf(os.Stderr, "    export OLLAMA_HOST=\"http://localhost:11434\" # Local offline model\n\n")
		return nil, errs.ExitCodeFailure
	}

	return inst, errs.ExitCodeSuccess
}

func runKrewSuggest(args []string) errs.ExitCode {
	if len(args) == 0 {
		return usageMessage("usage: kiw krew suggest \"<task or question>\"")
	}
	goal := strings.Join(args, " ")

	sysPrompt := `You are Krew CLI Assistant (like gh copilot suggest).
Your goal is to suggest the exact terminal command(s) or concise code snippet to accomplish the user's task.
Prefer Krewire CLI commands ('kiw ...') when appropriate for the ecosystem.
Format your answer with:
1. The exact command or code inside a markdown code block.
2. A brief, 1-2 sentence explanation of what the command does and any key flags.`

	inst, code := initKrewInstance(sysPrompt)
	if code != errs.ExitCodeSuccess {
		return code
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	res, err := inst.Run(ctx, goal)
	if err != nil {
		return fail(err)
	}

	fmt.Printf("\n%s\n\n", res.Content)
	return errs.ExitCodeSuccess
}

func runKrewExplain(args []string) errs.ExitCode {
	if len(args) == 0 {
		return usageMessage("usage: kiw krew explain \"<command or code snippet>\"")
	}
	query := strings.Join(args, " ")

	sysPrompt := `You are Krew CLI Assistant (like gh copilot explain).
Your goal is to explain the provided shell command, error message, or code snippet in simple, concise terms.
Highlight the purpose, key flags/arguments, potential side effects, and relevant Krewire ecosystem context.`

	inst, code := initKrewInstance(sysPrompt)
	if code != errs.ExitCodeSuccess {
		return code
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	res, err := inst.Run(ctx, query)
	if err != nil {
		return fail(err)
	}

	fmt.Printf("\n%s\n\n", res.Content)
	return errs.ExitCodeSuccess
}

func runKrewDirect(prompt string) errs.ExitCode {
	inst, code := initKrewInstance("")
	if code != errs.ExitCodeSuccess {
		return code
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	res, err := inst.Run(ctx, prompt)
	if err != nil {
		return fail(err)
	}

	fmt.Printf("\n%s\n\n", res.Content)
	return errs.ExitCodeSuccess
}

func runKrewRun(args []string) errs.ExitCode {
	if len(args) == 0 {
		return usageMessage("usage: kiw krew run \"<prompt>\"")
	}
	return runKrewDirect(strings.Join(args, " "))
}

func runKrewChat(args []string) errs.ExitCode {
	inst, code := initKrewInstance("")
	if code != errs.ExitCodeSuccess {
		return code
	}

	tm := term.NewTerminal()
	boldCyan := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, []term.Style{term.StyleBold}) }
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }

	fmt.Printf("\n  %s %s\n", boldCyan("kiw krew chat"), dim("— Autonomous AI Coding Session"))
	fmt.Printf("  %s %s\n", dim("Workspace:"), krewFlagDir)
	fmt.Printf("  %s\n\n", dim("Type your prompt. Type 'exit' or press Ctrl+C to quit."))

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Printf("  %s ", green("you>"))
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Printf("  %s\n\n", dim("Goodbye!"))
			break
		}

		fmt.Println()
		res, err := inst.Chat(ctx, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n\n", err)
			continue
		}

		fmt.Printf("\n  %s %s\n\n", boldCyan("krew:"), res.Content)
	}

	return errs.ExitCodeSuccess
}

func runKrewServe(args []string) errs.ExitCode {
	inst, code := initKrewInstance("")
	if code != errs.ExitCodeSuccess {
		return code
	}

	server := transport.NewHTTPServer(inst.Agent())
	addr := fmt.Sprintf("%s:%d", krewFlagHost, krewFlagPort)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.Handler(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	tm := term.NewTerminal()
	boldCyan := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }

	fmt.Printf("\n  %s\n", boldCyan("Krew HTTP/SSE Agent Daemon"))
	fmt.Printf("  %s http://%s\n", dim("Address:  "), green(addr))
	fmt.Printf("  %s POST http://%s/api/agent/run\n", dim("Endpoint: "), addr)
	fmt.Printf("  %s\n\n", dim("Press Ctrl+C to stop."))

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	<-stop
	fmt.Println("\nShutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	return errs.ExitCodeSuccess
}

func runKrewMCP(args []string) errs.ExitCode {
	inst, code := initKrewInstance("")
	if code != errs.ExitCodeSuccess {
		return code
	}

	mcpServer := transport.NewMCPServer(inst.Agent(), os.Stdin, os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := mcpServer.Serve(ctx); err != nil && err != context.Canceled {
		return fail(err)
	}
	return errs.ExitCodeSuccess
}

func krewEventHandler(ev agent.Event) {
	tm := term.NewTerminal()
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }
	yellow := func(s string) string { return tm.Paint(s, term.ColorYellow, nil) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, nil) }
	red := func(s string) string { return tm.Paint(s, term.ColorRed, nil) }

	switch ev.Type {
	case agent.EventThought:
		fmt.Printf("  %s %s\n", dim("• thought:"), dim(ev.Content))
	case agent.EventToolCall:
		fmt.Printf("  %s %s(%s)\n", yellow("⚙ tool call:"), ev.ToolName, dim(ev.ToolArgs))
	case agent.EventToolResult:
		preview := fmt.Sprintf("%v", ev.Result)
		if len(preview) > 100 {
			preview = preview[:100] + "... (truncated)"
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		fmt.Printf("  %s %s\n", green("✔ tool result:"), dim(preview))
	case agent.EventError:
		fmt.Printf("  %s %s\n", red("✖ error:"), ev.Error)
	}
}
