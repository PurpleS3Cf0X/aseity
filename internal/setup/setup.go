package setup

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Styles - we use simple ANSI codes to avoid importing tui (circular dep risk)
const (
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	bold   = "\033[1m"
	reset  = "\033[0m"
	dim    = "\033[2m"
)

func info(msg string)    { fmt.Printf("  %s●%s %s\n", green, reset, msg) }
func warn(msg string)    { fmt.Printf("  %s●%s %s\n", yellow, reset, msg) }
func fail(msg string)    { fmt.Printf("  %s●%s %s\n", red, reset, msg) }
func success(msg string) { fmt.Printf("  %s%s✓ %s%s\n", green, bold, msg, reset) }
func step(msg string)    { fmt.Printf("\n  %s%s%s\n", bold, msg, reset) }

// askYN prompts the user for y/n input. Default is the value returned on empty input.
func askYN(prompt string, defaultYes bool) bool {
	hint := "[Y/n]"
	if !defaultYes {
		hint = "[y/N]"
	}
	fmt.Printf("  %s%s %s%s %s ", yellow, bold, prompt, reset, hint)

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))

	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

// --- Detection ---

func IsOllamaInstalled() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}

func IsOllamaRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func IsDockerInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func IsDockerRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func IsDockerComposeAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Try "docker compose" (v2) first
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if cmd.Run() == nil {
		return true
	}
	// Try docker-compose (v1)
	_, err := exec.LookPath("docker-compose")
	return err == nil
}

func IsModelAvailable(model string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) != nil {
		return false
	}
	for _, m := range result.Models {
		// Match "deepseek-r1" against "deepseek-r1:latest" etc
		if m.Name == model || strings.HasPrefix(m.Name, model+":") {
			return true
		}
	}
	return false
}

// --- Actions ---

func InstallOllama() error {
	step("Installing Ollama...")

	if runtime.GOOS == "darwin" {
		// macOS: use the official install script
		info("Downloading Ollama for macOS...")
		cmd := exec.Command("bash", "-c", "curl -fsSL https://ollama.ai/install.sh | sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Fallback: try brew
			info("Trying Homebrew as fallback...")
			brewCmd := exec.Command("brew", "install", "ollama")
			brewCmd.Stdout = os.Stdout
			brewCmd.Stderr = os.Stderr
			if brewErr := brewCmd.Run(); brewErr != nil {
				return fmt.Errorf("failed to install Ollama: %v (brew: %v)", err, brewErr)
			}
		}
	} else if runtime.GOOS == "linux" {
		info("Downloading Ollama for Linux...")
		cmd := exec.Command("bash", "-c", "curl -fsSL https://ollama.ai/install.sh | sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Ollama: %v", err)
		}
	} else {
		return fmt.Errorf("automatic Ollama installation not supported on %s — visit https://ollama.ai to install manually", runtime.GOOS)
	}

	// Verify
	if !IsOllamaInstalled() {
		return fmt.Errorf("Ollama installation completed but binary not found in PATH")
	}
	success("Ollama installed")
	return nil
}

func StartOllama() error {
	step("Starting Ollama...")

	if IsOllamaRunning() {
		success("Ollama is already running")
		return nil
	}

	// Start ollama serve in background
	cmd := exec.Command("ollama", "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Detach from parent process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Ollama: %v", err)
	}

	// Release the process so it continues after we exit
	go func() { cmd.Wait() }()

	// Wait for it to be ready (up to 30 seconds)
	info("Waiting for Ollama to be ready...")
	for i := 0; i < 60; i++ {
		time.Sleep(500 * time.Millisecond)
		if IsOllamaRunning() {
			success("Ollama is running")
			return nil
		}
	}

	return fmt.Errorf("Ollama started but not responding after 30 seconds")
}

func PullModel(model string) error {
	step(fmt.Sprintf("Pulling model %s...", model))
	info("This may take a while depending on model size and network speed")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	payload := fmt.Sprintf(`{"name":"%s"}`, model)
	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/pull", strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed (HTTP %d): %s", resp.StatusCode, string(b))
	}

	// Stream progress
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lastPercent := -1

	for scanner.Scan() {
		var progress struct {
			Status    string `json:"status"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
		}
		if json.Unmarshal(scanner.Bytes(), &progress) != nil {
			continue
		}

		if progress.Total > 0 {
			pct := int(float64(progress.Completed) / float64(progress.Total) * 100)
			if pct != lastPercent {
				lastPercent = pct
				bar := pct / 2
				fmt.Printf("\r  %s [%s%s%s%s] %d%%  %s",
					progress.Status,
					green, strings.Repeat("█", bar), reset,
					strings.Repeat("░", 50-bar),
					pct,
					dim+humanBytes(progress.Completed)+"/"+humanBytes(progress.Total)+reset,
				)
			}
		} else if progress.Status != "" {
			fmt.Printf("\r  %-60s", progress.Status)
		}
	}
	fmt.Println()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream error: %v", err)
	}

	// Verify model is actually available after pull
	info("Verifying model is ready...")
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		if IsModelAvailable(model) {
			success(fmt.Sprintf("Model %s ready", model))
			return nil
		}
	}

	return fmt.Errorf("model pull completed but model not available after 15 seconds")
}

// OllamaBaseURL is exposed for testing
var OllamaBaseURL = "http://localhost:11434"

func LoadModel(ggufPath string) error {
	absPath, err := filepath.Abs(ggufPath)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("file not found: %s", absPath)
	}

	modelName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	step(fmt.Sprintf("Loading custom model '%s' from %s...", modelName, absPath))

	// Creating a temporary Modelfile
	modelfileContent := fmt.Sprintf("FROM %s\n", absPath)
	payload := fmt.Sprintf(`{"name":"%s", "modelfile":%q}`, modelName, modelfileContent)

	info("Sending create request to Ollama...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	url := fmt.Sprintf("%s/api/create", OllamaBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create failed (HTTP %d): %s", resp.StatusCode, string(b))
	}

	// Stream progress
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var progress struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(scanner.Bytes(), &progress) == nil {
			if progress.Status != "" {
				fmt.Printf("\r  %s%-60s%s", dim, progress.Status, reset)
			}
		}
	}
	fmt.Println()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream error: %v", err)
	}

	success(fmt.Sprintf("Model '%s' created successfully!", modelName))
	info(fmt.Sprintf("You can now use it with: aseity --model %s", modelName))
	return nil
}

func StartDockerOllama() error {
	step("Starting Ollama via Docker Compose...")

	// Find docker-compose.yml
	composePath := findComposeFile()
	if composePath == "" {
		return fmt.Errorf("docker-compose.yml not found — run from the aseity project directory or install Ollama directly")
	}

	dir := filepath.Dir(composePath)
	cmd := exec.Command("docker", "compose", "up", "-d", "ollama")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up failed: %v", err)
	}

	// Wait for health check
	info("Waiting for Ollama container to be healthy...")
	for i := 0; i < 60; i++ {
		time.Sleep(2 * time.Second)
		if IsOllamaRunning() {
			success("Ollama container is running")
			return nil
		}
	}

	return fmt.Errorf("Ollama container started but not responding after 2 minutes")
}

func StartDockerVLLM() error {
	step("Starting vLLM via Docker Compose...")

	composePath := findComposeFile()
	if composePath == "" {
		return fmt.Errorf("docker-compose.yml not found")
	}

	dir := filepath.Dir(composePath)
	cmd := exec.Command("docker", "compose", "--profile", "vllm", "up", "-d")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up failed: %v", err)
	}

	info("Waiting for vLLM to load the model (this can take a few minutes)...")
	for i := 0; i < 120; i++ {
		time.Sleep(3 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8000/v1/models", nil)
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			success("vLLM is running with deepseek-r1")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return fmt.Errorf("vLLM started but not responding after 6 minutes")
}

// --- Main Wizard ---

// RunSetup runs the interactive setup wizard. Returns true if setup succeeded.
func RunSetup(providerName, modelName string) bool {
	fmt.Println()
	step("Aseity Setup Wizard")
	fmt.Printf("  %sNo running backend detected. Let's get you set up.%s\n", dim, reset)
	fmt.Println()

	// Strategy 1: Ollama (preferred — simplest)
	if tryOllamaSetup(modelName) {
		return true
	}

	// Strategy 2: Docker
	if tryDockerSetup(providerName, modelName) {
		return true
	}

	// Nothing worked
	fmt.Println()
	fail("Could not set up a backend automatically.")
	fmt.Println()
	fmt.Printf("  %sManual setup options:%s\n", bold, reset)
	fmt.Printf("  1. Install Ollama:  %scurl -fsSL https://ollama.ai/install.sh | sh%s\n", green, reset)
	fmt.Printf("  2. Start Ollama:    %sollama serve%s\n", green, reset)
	fmt.Printf("  3. Pull a model:    %sollama pull %s%s\n", green, modelName, reset)
	fmt.Printf("  4. Or use Docker:   %sdocker compose up -d ollama%s\n", green, reset)
	fmt.Println()
	return false
}

// RunSetupDocker forces Docker-based setup.
func RunSetupDocker(modelName string) bool {
	fmt.Println()
	step("Aseity Docker Setup")
	return tryDockerSetup("ollama", modelName)
}

func tryOllamaSetup(modelName string) bool {
	// Check if Ollama is installed
	if !IsOllamaInstalled() {
		info("Ollama is not installed")
		if !askYN("Install Ollama now?", true) {
			return false
		}
		if err := InstallOllama(); err != nil {
			fail(err.Error())
			return false
		}
	} else {
		success("Ollama is installed")
	}

	// Check if running
	if !IsOllamaRunning() {
		info("Ollama is not running")
		if !askYN("Start Ollama now?", true) {
			return false
		}
		if err := StartOllama(); err != nil {
			fail(err.Error())
			return false
		}
	} else {
		success("Ollama is running")
	}

	// Check if model is available
	if !IsModelAvailable(modelName) {
		info(fmt.Sprintf("Model %s is not downloaded yet", modelName))
		if !askYN(fmt.Sprintf("Download %s now?", modelName), true) {
			return false
		}
		if err := PullModel(modelName); err != nil {
			fail(err.Error())
			return false
		}
	} else {
		success(fmt.Sprintf("Model %s is available", modelName))
	}

	fmt.Println()
	success("Setup complete! Starting aseity...")
	return true
}

func tryDockerSetup(providerName, modelName string) bool {
	if !IsDockerInstalled() {
		info("Docker is not installed")
		fmt.Printf("  %sInstall Docker from: https://docs.docker.com/get-docker/%s\n", dim, reset)
		return false
	}
	success("Docker is installed")

	if !IsDockerRunning() {
		warn("Docker daemon is not running")
		fmt.Printf("  %sPlease start Docker Desktop or the Docker daemon, then try again.%s\n", dim, reset)
		return false
	}
	success("Docker is running")

	if !IsDockerComposeAvailable() {
		fail("Docker Compose not available")
		return false
	}

	composePath := findComposeFile()
	if composePath == "" {
		fail("docker-compose.yml not found in current or parent directories")
		fmt.Printf("  %sClone the aseity repo first: git clone https://github.com/PurpleS3Cf0X/aseity%s\n", dim, reset)
		return false
	}

	if !askYN("Start Ollama via Docker Compose?", true) {
		return false
	}

	if err := StartDockerOllama(); err != nil {
		fail(err.Error())
		return false
	}

	// Pull model inside Docker Ollama
	if !IsModelAvailable(modelName) {
		if !askYN(fmt.Sprintf("Download %s now?", modelName), true) {
			return false
		}
		if err := PullModel(modelName); err != nil {
			fail(err.Error())
			return false
		}
	}

	fmt.Println()
	success("Docker setup complete! Starting aseity...")
	return true
}

// --- Helpers ---

func findComposeFile() string {
	// Check current directory and up to 3 parent directories
	dir, _ := os.Getwd()
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(dir, "docker-compose.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		candidate = filepath.Join(dir, "docker-compose.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Check if the binary was installed from a known location
	exe, _ := os.Executable()
	if exe != "" {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "docker-compose.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
