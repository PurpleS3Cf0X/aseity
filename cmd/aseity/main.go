package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/config"
	"github.com/jeanpaul/aseity/internal/headless"
	"github.com/jeanpaul/aseity/internal/health"
	"github.com/jeanpaul/aseity/internal/model"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/setup"
	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/tui"
	"github.com/jeanpaul/aseity/pkg/version"
)

func main() {
	providerFlag := flag.String("provider", "", "Provider name (ollama, vllm, openai, anthropic, google)")
	modelFlag := flag.String("model", "", "Model name")
	versionFlag := flag.Bool("version", false, "Print version")
	helpFlag := flag.Bool("help", false, "Show help")
	flag.BoolVar(helpFlag, "h", false, "Show help")
	yesFlag := flag.Bool("yes", false, "Auto-approve all tool execution")
	flag.BoolVar(yesFlag, "y", false, "Auto-approve all tool execution")
	headlessFlag := flag.Bool("headless", false, "Run in headless mode (no TUI)")
	sessionFlag := flag.String("session", "", "Load a previous session (by ID or file path)")
	flag.Usage = showHelp
	flag.Parse()

	if *helpFlag {
		showHelp()
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Printf("aseity %s (%s)\n", version.Version, version.Commit)
		os.Exit(0)
	}

	// Helper to determine mode
	// If args exist (except known subcommands) OR headless flag is set -> Headless
	isHeadless := *headlessFlag
	initialPrompt := ""

	// Handle subcommands
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "models":
			cmdModels()
			return
		case "pull":
			if len(args) < 2 {
				fatal("usage: aseity pull <model-ref>")
			}
			cmdPull(args[1])
			return
		case "remove":
			if len(args) < 2 {
				fatal("usage: aseity remove <model-name>")
			}
			cmdRemove(args[1])
			return
		case "search":
			query := ""
			if len(args) > 1 {
				query = strings.Join(args[1:], " ")
			}
			cmdSearch(query)
			return
		case "providers":
			cmdProviders()
			return
		case "tools":
			cmdTools()
			return
		case "doctor":
			cmdDoctor()
			return
		case "setup":
			docker := len(args) > 1 && args[1] == "--docker"
			cmdSetup(docker)
			return
		case "help":
			showHelp()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s. Assuming it is a prompt.\n", args[0])
			initialPrompt = strings.Join(args, " ")
			// If arguments are provided but not a subcommand, we default to headless
			// unless explicitly wanted TUI?
			// Actually, `aseity "hello"` traditionally launched TUI in other tools.
			// But user asked for headless mode.
			// Let's check: if --headless is set implicitly or explicitly.
			// Let's decide: if args present, default to headless?
			// User said "Implement a Headless Mode (--headless)".
			// So let's require the flag OR be smart.
			// Ideally: `aseity -y "scan"` -> Headless.
			isHeadless = true
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("config error: %s", err)
	}

	provName := *providerFlag
	if provName == "" {
		provName = cfg.DefaultProvider
	}
	modelName := *modelFlag
	if modelName == "" {
		modelName = cfg.DefaultModel
	}

	// Startup health check (skip in headless for speed? No, keep it for safety unless ignored)
	// Actually for "scriptable" tools, we might want to be quiet.
	// But let's keep it for now, TUI banners need to be suppressed in headless.

	if !isHeadless {
		// ... TUI Health Checks (only showing banner if not headless)
		fmt.Print(tui.GradientBanner())
		fmt.Printf("\n  %s  %s\n", tui.StatusProviderStyle.Render(" "+provName+" "), tui.StatusBarStyle.Render(" "+modelName+" "))

		// ... (Health check logic, same as before) ...
		pcfg, _ := cfg.ProviderFor(provName)
		fmt.Printf("  %s", tui.SpinnerStyle.Render("● Checking provider connectivity..."))
		status := health.Check(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey)
		if !status.Reachable {
			fmt.Printf("\r  %s\n", tui.ErrorStyle.Render("✗ "+status.Error))
			if setup.RunSetup(provName, modelName) {
				status = health.Check(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey)
			}
			if !status.Reachable {
				fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Run 'aseity doctor' for diagnostics"))
				os.Exit(1)
			}
		}
		if status.Reachable {
			fmt.Printf("\r  %s (%s)\n", tui.BannerStyle.Render("✓ Connected"), status.Latency.Round(time.Millisecond))
		}
		// ... End TUI Health Checks
		fmt.Println()

		launchTUI(cfg, provName, modelName, *yesFlag, initialPrompt, *sessionFlag)
	} else {
		// Headless Mode
		launchHeadless(cfg, provName, modelName, *yesFlag, initialPrompt)
	}
}

func makeProvider(cfg *config.Config, name, modelName string) (provider.Provider, error) {
	// Check env overrides
	if baseURL := os.Getenv("ASEITY_BASE_URL"); baseURL != "" {
		return provider.NewOpenAI(name, baseURL, os.Getenv("ASEITY_API_KEY"), modelName), nil
	}

	pcfg, ok := cfg.ProviderFor(name)
	if !ok {
		return nil, fmt.Errorf("unknown provider %q — configure it in ~/.config/aseity/config.yaml", name)
	}

	model := modelName
	if model == "" {
		model = pcfg.Model
	}

	switch pcfg.Type {
	case "openai":
		return provider.NewOpenAI(name, pcfg.BaseURL, pcfg.APIKey, model), nil
	case "anthropic":
		if pcfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic requires api_key (set ANTHROPIC_API_KEY)")
		}
		return provider.NewAnthropic(pcfg.APIKey, model), nil
	case "google":
		if pcfg.APIKey == "" {
			return nil, fmt.Errorf("google requires api_key (set GEMINI_API_KEY)")
		}
		return provider.NewGoogle(pcfg.APIKey, model), nil
	default:
		return nil, fmt.Errorf("unknown provider type %q", pcfg.Type)
	}
}

func cmdModels() {
	cfg, _ := config.Load()
	ollamaURL := "http://localhost:11434"
	if p, ok := cfg.Providers["ollama"]; ok {
		ollamaURL = strings.TrimSuffix(p.BaseURL, "/v1")
	}
	mgr := model.NewManager(ollamaURL, "")
	models, err := mgr.List(context.Background())
	if err != nil {
		fatal("failed to list models: %s", err)
	}
	fmt.Println(tui.BannerStyle.Render("  Local Models"))
	fmt.Println()
	for _, m := range models {
		size := float64(m.Size) / (1024 * 1024 * 1024)
		fmt.Printf("  %s  %s  %.1fGB\n",
			tui.UserLabelStyle.Render(m.Name),
			tui.ToolCallStyle.Render(m.Parameters),
			size,
		)
	}
}

func cmdPull(ref string) {
	cfg, _ := config.Load()
	ollamaURL := "http://localhost:11434"
	if p, ok := cfg.Providers["ollama"]; ok {
		ollamaURL = strings.TrimSuffix(p.BaseURL, "/v1")
	}
	mgr := model.NewManager(ollamaURL, os.Getenv("HF_TOKEN"))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("%s Pulling %s...\n", tui.SpinnerStyle.Render("●"), ref)
	err := mgr.Pull(ctx, ref, func(p model.PullProgress) {
		if p.Percent > 0 {
			bar := int(p.Percent / 2)
			fmt.Printf("\r  %s [%s%s] %.0f%%",
				p.Status,
				tui.BannerStyle.Render(strings.Repeat("█", bar)),
				strings.Repeat("░", 50-bar),
				p.Percent,
			)
		} else {
			fmt.Printf("\r  %s", p.Status)
		}
	})
	fmt.Println()
	if err != nil {
		fatal("pull failed: %s", err)
	}
	fmt.Println(tui.BannerStyle.Render("  ✓ Done"))
}

func cmdRemove(name string) {
	cfg, _ := config.Load()
	ollamaURL := "http://localhost:11434"
	if p, ok := cfg.Providers["ollama"]; ok {
		ollamaURL = strings.TrimSuffix(p.BaseURL, "/v1")
	}
	mgr := model.NewManager(ollamaURL, "")
	if err := mgr.Remove(context.Background(), name); err != nil {
		fatal("remove failed: %s", err)
	}
	fmt.Println(tui.BannerStyle.Render("  ✓ Removed " + name))
}

func cmdSearch(query string) {
	mgr := model.NewManager("", os.Getenv("HF_TOKEN"))
	models, err := mgr.SearchHuggingFace(context.Background(), query, 20)
	if err != nil {
		fatal("search failed: %s", err)
	}
	if len(models) == 0 {
		fmt.Println(tui.HelpStyle.Render("  No models found"))
		return
	}
	fmt.Println(tui.BannerStyle.Render("  HuggingFace Models (GGUF)"))
	fmt.Println()
	for _, m := range models {
		fmt.Printf("  %s\n", tui.UserLabelStyle.Render(m.Name))
	}
	fmt.Println()
	fmt.Println(tui.HelpStyle.Render("  Pull with: aseity pull " + models[0].Name))
}

func cmdProviders() {
	cfg, _ := config.Load()
	fmt.Println(tui.BannerStyle.Render("  Configured Providers"))
	fmt.Println()
	for name, p := range cfg.Providers {
		status := tui.DimGreen
		label := "configured"
		if p.BaseURL != "" {
			label = p.BaseURL
		}
		fmt.Printf("  %s  %s  %s\n",
			tui.UserLabelStyle.Render(name),
			tui.ToolCallStyle.Render(p.Type),
			tui.HelpStyle.Foreground(status).Render(label),
		)
	}
}

func cmdDoctor() {
	cfg, err := config.Load()
	if err != nil {
		fatal("config error: %s", err)
	}

	fmt.Print(tui.GradientBanner())
	fmt.Println(tui.BannerStyle.Render("  Service Health Check"))
	fmt.Println()

	defaultOk := true
	otherIssues := 0

	for name, pcfg := range cfg.Providers {
		isDefault := name == cfg.DefaultProvider
		label := name
		if isDefault {
			label = name + " (default)"
		}

		fmt.Printf("  %s %s ... ",
			tui.ToolCallStyle.Render("●"),
			tui.UserLabelStyle.Render(label),
		)
		status := health.Check(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey)
		if status.Reachable {
			modelCount := ""
			if len(status.Models) > 0 {
				modelCount = fmt.Sprintf(" (%d models)", len(status.Models))
			}
			fmt.Printf("%s%s %s\n",
				tui.BannerStyle.Render("✓ OK"),
				tui.HelpStyle.Render(modelCount),
				tui.HelpStyle.Render(status.Latency.Round(time.Millisecond).String()),
			)
		} else {
			if isDefault {
				defaultOk = false
				fmt.Printf("%s\n", tui.ErrorStyle.Render("✗ "+status.Error))
			} else {
				otherIssues++
				fmt.Printf("%s\n", tui.HelpStyle.Render("- "+status.Error+" (optional)"))
			}
		}
	}

	// Check Docker
	fmt.Printf("\n  %s %s ... ", tui.ToolCallStyle.Render("●"), tui.UserLabelStyle.Render("docker"))
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		fmt.Println(tui.BannerStyle.Render("✓ Available"))
	} else {
		fmt.Println(tui.HelpStyle.Render("- Not detected (optional)"))
	}

	// Check config file
	fmt.Printf("  %s %s ... ", tui.ToolCallStyle.Render("●"), tui.UserLabelStyle.Render("config"))
	home, _ := os.UserHomeDir()
	configPath := home + "/.config/aseity/config.yaml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println(tui.BannerStyle.Render("✓ " + configPath))
	} else {
		fmt.Printf("%s\n", tui.HelpStyle.Render("- Using defaults (create "+configPath+" to customize)"))
	}

	fmt.Println()
	if defaultOk {
		if otherIssues > 0 {
			fmt.Println(tui.BannerStyle.Render("  Default provider healthy!"))
			fmt.Println(tui.HelpStyle.Render(fmt.Sprintf("  (%d optional provider(s) not configured)", otherIssues)))
		} else {
			fmt.Println(tui.BannerStyle.Render("  All services healthy!"))
		}
	} else {
		fmt.Println(tui.ErrorStyle.Render("  Default provider is unreachable."))
		fmt.Println(tui.HelpStyle.Render("  For local models, start Ollama: ollama serve"))
		fmt.Println(tui.HelpStyle.Render("  Or run: aseity setup"))
	}
}

func cmdTools() {
	cfg, err := config.Load()
	if err != nil {
		fatal("config error: %s", err)
	}

	// Create a temporary registry to see what tools are available
	// We don't need a real provider or agent manager here, just the definitions.
	reg := tools.NewRegistry(cfg.Tools.AutoApprove, false)

	// We pass nil for AgentFactory because we just want to list the tool, not execute it.
	// But Wait, tools.RegisterDefaults needs an interface.
	// We can pass a dummy implementation or just nil if SpawnAgentTool handles nil gracefully during registration (it doesn't typically).
	// Actually, SpawnAgentTool struct just holds it. Implementation matters only during Execution.
	// Let's check NewSpawnAgentTool.
	// tools.NewSpawnAgentTool(nil) should be fine structurally.
	tools.RegisterDefaults(reg, cfg.Tools.AllowedCommands, cfg.Tools.DisallowedCommands)

	fmt.Println(tui.BannerStyle.Render("  Available Tools"))
	fmt.Println()

	// Convert map to slice for sorting? Registry.ToolDefs() returns slice.
	defs := reg.ToolDefs()

	for _, t := range defs {
		confirm := " "
		if reg.NeedsConfirmation(t.Name) {
			confirm = tui.WarningStyle.Render("(requires approval)")
		} else {
			confirm = tui.HelpStyle.Render("(auto)")
		}

		fmt.Printf("  %s %s\n", tui.UserLabelStyle.Render(t.Name), confirm)
		fmt.Printf("    %s\n\n", tui.HelpStyle.Render(t.Description))
	}
}

func cmdSetup(docker bool) {
	cfg, _ := config.Load()
	modelName := cfg.DefaultModel
	fmt.Print(tui.GradientBanner())

	var ok bool
	if docker {
		ok = setup.RunSetupDocker(modelName)
	} else {
		ok = setup.RunSetup(cfg.DefaultProvider, modelName)
	}
	if !ok {
		os.Exit(1)
	}

	// Setup succeeded — launch the TUI directly instead of asking user to run again
	fmt.Println()
	fmt.Println()
	launchTUI(cfg, cfg.DefaultProvider, modelName, false, "", "")
}

func launchHeadless(cfg *config.Config, provName, modelName string, allowAll bool, initialPrompt string) {
	if initialPrompt == "" {
		fatal("Headless mode requires an initial prompt (e.g., aseity \"do this\")")
	}

	prov, toolReg, _, err := setupAgentEnv(cfg, provName, modelName, allowAll)
	if err != nil {
		fatal("%s", err)
	}

	// We don't need the agent manager here unless we want cleanup for subagents?
	// setupAgentEnv registers generic tools.
	// runner.Run creates its own agent.

	err = headless.Run(context.Background(), prov, toolReg, initialPrompt)
	if err != nil {
		fatal("Execution error: %s", err)
	}
}

func setupAgentEnv(cfg *config.Config, provName, modelName string, allowAll bool) (provider.Provider, *tools.Registry, *agent.AgentManager, error) {
	prov, err := makeProvider(cfg, provName, modelName)
	if err != nil {
		return nil, nil, nil, err
	}
	prov = provider.WithRetry(prov, 3)

	toolReg := tools.NewRegistry(cfg.Tools.AutoApprove, allowAll)
	tools.RegisterDefaults(toolReg, cfg.Tools.AllowedCommands, cfg.Tools.DisallowedCommands)

	agentMgr := agent.NewAgentManager(prov, toolReg, 3)
	toolReg.Register(tools.NewSpawnAgentTool(agentMgr))
	toolReg.Register(tools.NewWaitForAgentTool(agentMgr))
	toolReg.Register(tools.NewListAgentsTool(agentMgr))
	toolReg.Register(tools.NewJudgeTool(agentMgr))

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			agentMgr.Cleanup(30 * time.Minute)
		}
	}()

	return prov, toolReg, agentMgr, nil
}

// launchTUI starts the interactive chat interface
func launchTUI(cfg *config.Config, provName, modelName string, allowAll bool, initialPrompt string, sessionID string) {
	prov, toolReg, _, err := setupAgentEnv(cfg, provName, modelName, allowAll)
	if err != nil {
		fatal("%s", err)
	}

	var conv *agent.Conversation
	if sessionID != "" {
		// heuristic: if it contains just alphanumeric, treat as ID in ~/.config/aseity/sessions/ID.json
		// if contains / or .json, treat as path
		path := sessionID
		if !strings.Contains(sessionID, "/") && !strings.Contains(sessionID, ".") {
			home, _ := os.UserHomeDir()
			path = fmt.Sprintf("%s/.config/aseity/sessions/%s.json", home, sessionID)
		}

		fmt.Printf("  Loading session from %s...\n", path)
		c, err := agent.LoadConversation(path)
		if err != nil {
			fmt.Printf("  %s\n\n", tui.ErrorStyle.Render("✗ Failed to load session: "+err.Error()))
			// Fallback to new session? Or exit?
			// Let's fallback but warn
			time.Sleep(2 * time.Second)
		} else {
			conv = c
		}
	}

	m := tui.NewModel(prov, toolReg, provName, modelName, conv)

	// Create program with appropriate options based on terminal capabilities
	var opts []tea.ProgramOption

	// Check if we have a proper terminal
	if isTerminal() {
		opts = append(opts, tea.WithAltScreen())
	}

	// Always try to use mouse support if available
	opts = append(opts, tea.WithMouseCellMotion())

	p := tea.NewProgram(m, opts...)

	if _, err := p.Run(); err != nil {
		fatal("TUI error: %s", err)
	}
}

// isTerminal checks if stdin is a terminal
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, tui.ErrorStyle.Render("error: "+fmt.Sprintf(format, args...))+"\n")
	os.Exit(1)
}

func showHelp() {
	help := `
` + tui.BannerStyle.Render("Aseity") + ` - AI coding assistant for your terminal

` + tui.UserLabelStyle.Render("USAGE:") + `
  aseity [flags]              Start interactive chat
  aseity <command> [args]     Run a command

` + tui.UserLabelStyle.Render("COMMANDS:") + `
  models                      List downloaded models
  pull <model>                Download a model (e.g., deepseek-r1, llama3.2)
  remove <model>              Remove a downloaded model
  search <query>              Search HuggingFace for GGUF models
  search <query>              Search HuggingFace for GGUF models
  providers                   List configured providers
  tools                       List available tools
  doctor                      Check health of all services
  setup [--docker]            Run first-time setup wizard
  help                        Show this help

` + tui.UserLabelStyle.Render("FLAGS:") + `
  --provider <name>           Use specific provider (ollama, openai, anthropic, google)
  --model <name>              Use specific model
  --session <id|path>         Resume a previous session
  --version                   Show version
  --yes, -y                   Auto-approve all tool execution (dangerous)
  --help, -h                  Show this help

` + tui.UserLabelStyle.Render("EXAMPLES:") + `
  aseity                      Start chat with default provider (Ollama)
  aseity --model llama3.2     Start chat with specific model
  aseity --provider openai    Use OpenAI (requires OPENAI_API_KEY)
  aseity pull deepseek-r1     Download the deepseek-r1 model
  aseity doctor               Check if services are running

` + tui.UserLabelStyle.Render("CHAT COMMANDS:") + `
  /help                       Show available chat commands
  /clear                      Clear conversation history
  /compact                    Compress conversation to save tokens
  /save [path]                Export conversation to markdown
  /tokens                     Show estimated token usage
  /quit                       Exit aseity

` + tui.UserLabelStyle.Render("KEYBOARD SHORTCUTS:") + `
  Enter                       Send message
  Alt+Enter                   New line in message
  Ctrl+T                      Toggle thinking visibility
  Ctrl+C                      Cancel current operation
  Esc                         Quit

` + tui.HelpStyle.Render("Documentation: https://github.com/PurpleS3Cf0X/aseity") + `
`
	fmt.Println(help)
}
