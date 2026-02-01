package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/config"
	"github.com/jeanpaul/aseity/internal/health"
	"github.com/jeanpaul/aseity/internal/model"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/tui"
	"github.com/jeanpaul/aseity/pkg/version"
)

func main() {
	providerFlag := flag.String("provider", "", "Provider name (ollama, vllm, openai, anthropic, google, huggingface)")
	modelFlag := flag.String("model", "", "Model name")
	versionFlag := flag.Bool("version", false, "Print version")
	flag.Parse()

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

	prov, err := makeProvider(cfg, provName, modelName)
	if err != nil {
		fatal("%s", err)
	}

	// Startup health check: verify provider is reachable
	fmt.Print(tui.BannerStyle.Render(tui.Banner))
	fmt.Printf("\n  %s  %s\n",
		tui.StatusProviderStyle.Render(" "+provName+" "),
		tui.StatusBarStyle.Render(" "+modelName+" "),
	)

	pcfg, _ := cfg.ProviderFor(provName)
	fmt.Printf("  %s", tui.SpinnerStyle.Render("● Checking provider connectivity..."))
	status := health.Check(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey)
	if !status.Reachable {
		fmt.Printf("\r  %s\n", tui.ErrorStyle.Render("✗ "+status.Error))
		fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Run 'aseity doctor' for diagnostics"))
		os.Exit(1)
	}
	fmt.Printf("\r  %s (%s)\n", tui.BannerStyle.Render("✓ Connected"), status.Latency.Round(time.Millisecond))

	// Check if model is available
	if pcfg.Type == "openai" {
		if err := health.CheckModel(context.Background(), pcfg.Type, pcfg.BaseURL, pcfg.APIKey, modelName); err != nil {
			fmt.Printf("  %s\n", tui.ErrorStyle.Render("✗ "+err.Error()))
			fmt.Printf("  %s\n\n", tui.HelpStyle.Render("Pull it with: aseity pull "+modelName))
			os.Exit(1)
		}
		fmt.Printf("  %s\n", tui.BannerStyle.Render("✓ Model "+modelName+" available"))
	}
	fmt.Println()

	toolReg := tools.NewRegistry(cfg.Tools.AutoApprove)
	tools.RegisterDefaults(toolReg)

	// Set up agent manager and register agent tools
	agentMgr := agent.NewAgentManager(prov, toolReg, 3)
	toolReg.Register(tools.NewSpawnAgentTool(agentMgr))
	toolReg.Register(tools.NewListAgentsTool(agentMgr))

	m := tui.NewModel(prov, toolReg, provName, modelName)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fatal("TUI error: %s", err)
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
	fmt.Printf("%s Pulling %s...\n", tui.SpinnerStyle.Render("●"), ref)
	err := mgr.Pull(context.Background(), ref, func(p model.PullProgress) {
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

	fmt.Print(tui.BannerStyle.Render(tui.Banner))
	fmt.Println(tui.BannerStyle.Render("  Service Health Check"))
	fmt.Println()

	allOk := true
	for name, pcfg := range cfg.Providers {
		fmt.Printf("  %s %s ... ",
			tui.ToolCallStyle.Render("●"),
			tui.UserLabelStyle.Render(name),
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
			allOk = false
			fmt.Printf("%s\n", tui.ErrorStyle.Render("✗ "+status.Error))
		}
	}

	// Check Docker
	fmt.Printf("\n  %s %s ... ", tui.ToolCallStyle.Render("●"), tui.UserLabelStyle.Render("docker"))
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		fmt.Println(tui.BannerStyle.Render("✓ Available"))
	} else {
		fmt.Println(tui.HelpStyle.Render("- Not detected"))
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
	if allOk {
		fmt.Println(tui.BannerStyle.Render("  All services healthy!"))
	} else {
		fmt.Println(tui.ErrorStyle.Render("  Some services are unreachable."))
		fmt.Println(tui.HelpStyle.Render("  For local models, start Ollama: ollama serve"))
		fmt.Println(tui.HelpStyle.Render("  For Docker: docker compose up -d ollama"))
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, tui.ErrorStyle.Render("error: "+fmt.Sprintf(format, args...))+"\n")
	os.Exit(1)
}
