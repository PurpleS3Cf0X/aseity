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
	flag.Usage = showHelp
	flag.Parse()

	if *helpFlag {
		showHelp()
		os.Exit(0)
	}
	// ... (content skipped, targeting lines around 412 for NewRegistry)

	if *versionFlag {
		fmt.Printf("aseity %s (%s)\n", version.Version, version.Commit)
		os.Exit(0)
	}

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
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
			showHelp()
			os.Exit(1)
		}
	}

	// Interactive mode
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

	// Startup health check: verify provider is reachable
	fmt.Print(tui.GradientBanner())
	fmt.Printf("\n  %s  %s\n",
		tui.StatusProviderStyle.Render(" "+provName+" "),
		tui.StatusBarStyle.Render(" "+modelName+" "),
	)

	pcfg, _ := cfg.ProviderFor(provName)
	fmt.Printf("  %s", tui.SpinnerStyle.Render("● Checking provider connectivity..."))
	status := health.Check(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey)
	if !status.Reachable {
		fmt.Printf("\r  %s\n", tui.ErrorStyle.Render("✗ "+status.Error))

		// Launch setup wizard instead of just exiting
		if setup.RunSetup(provName, modelName) {
			// Retry health check after setup
			status = health.Check(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey)
		}
		if !status.Reachable {
			fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Run 'aseity doctor' for diagnostics or 'aseity setup' to retry"))
			os.Exit(1)
		}
	}
	if status.Reachable {
		fmt.Printf("\r  %s (%s)\n", tui.BannerStyle.Render("✓ Connected"), status.Latency.Round(time.Millisecond))
	}

	// Check if model is available
	if pcfg.Type == "openai" {
		if err := health.CheckModel(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey, modelName); err != nil {
			fmt.Printf("  %s\n", tui.ErrorStyle.Render("✗ "+err.Error()))
			// Offer to pull the model
			if setup.IsOllamaRunning() {
				fmt.Printf("\n  %s", tui.ConfirmStyle.Render("Download "+modelName+" now? [Y/n] "))
				var response string
				fmt.Scanln(&response)
				response = strings.ToLower(strings.TrimSpace(response))
				if response == "" || response == "y" || response == "yes" {
					fmt.Println()
					if err := setup.PullModel(modelName); err != nil {
						fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Pull it manually: aseity pull "+modelName))
						os.Exit(1)
					}
				} else {
					fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Pull it later with: aseity pull "+modelName))
					os.Exit(0)
				}
			} else {
				fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Pull it with: aseity pull "+modelName))
				os.Exit(1)
			}
		} else {
			fmt.Printf("  %s\n", tui.BannerStyle.Render("✓ Model "+modelName+" available"))
		}
	}
	fmt.Println()

	launchTUI(cfg, provName, modelName, *yesFlag)
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
	launchTUI(cfg, cfg.DefaultProvider, modelName, false)
}

// launchTUI starts the interactive chat interface
func launchTUI(cfg *config.Config, provName, modelName string, allowAll bool) {
	prov, err := makeProvider(cfg, provName, modelName)
	if err != nil {
		fatal("%s", err)
	}
	prov = provider.WithRetry(prov, 3)

	toolReg := tools.NewRegistry(cfg.Tools.AutoApprove, allowAll)
	tools.RegisterDefaults(toolReg, cfg.Tools.AllowedCommands, cfg.Tools.DisallowedCommands)

	agentMgr := agent.NewAgentManager(prov, toolReg, 3)
	toolReg.Register(tools.NewSpawnAgentTool(agentMgr))
	toolReg.Register(tools.NewListAgentsTool(agentMgr))

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			agentMgr.Cleanup(30 * time.Minute)
		}
	}()

	m := tui.NewModel(prov, toolReg, provName, modelName)

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
  providers                   List configured providers
  doctor                      Check health of all services
  setup [--docker]            Run first-time setup wizard
  help                        Show this help

` + tui.UserLabelStyle.Render("FLAGS:") + `
  --provider <name>           Use specific provider (ollama, openai, anthropic, google)
  --model <name>              Use specific model
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
