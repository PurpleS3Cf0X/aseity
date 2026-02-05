# Aseity

A powerful AI coding assistant that runs in your terminal. Connect to local models via Ollama, or use cloud providers like OpenAI, Anthropic, and Google.

![License](https://img.shields.io/badge/license-MIT-green)
![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)

<img width="1055" height="808" alt="Screenshot 2026-02-05 at 6 39 48 PM" src="https://github.com/user-attachments/assets/d69e5a94-9a0b-4b3d-acc0-056dcc760c34" />


## Features

- **Local-first** — Run models on your own machine with Ollama or vLLM
- **Multi-provider** — Switch between Ollama, OpenAI, Anthropic, Google, or any OpenAI-compatible API
- **Parallel Web Crawling** — **New in v2.8.1!** Scrape multiple sites concurrently using Crawl4AI or local browsers
- **Interactive TUI** — Beautiful terminal interface with "Premium Matrix" aesthetics, syntax highlighting, and animated boundaries
- **Interactive Tools** — **New in v1.0.5!** Supports interactive commands like `sudo`, `ssh`, and scripts that ask for passwords or input
- **Tool use** — Execute shell commands, read/write files, search the web
- **Sub-agents** — Spawn autonomous agents recursively for complex task decomposition
- **Streaming Output** — Real-time output feedback for long-running commands
- **Mac Compatible** — Docker setup is now optimized for Apple Silicon
- **Smart Diff View** — See exactly what changes when files are edited
- **Thinking visibility** — See the model's reasoning process (for supported models)
- **Auto-setup** — First-run wizard installs and configures everything
- **Auto-approve** — Optional `-y` flag to skip permission prompts for trusted sessions
- **Headless Mode** — Scriptable CLI mode for automation (`aseity --headless "scan" > report.txt`)
- **Smart Output** — Automatically truncates and saves large tool outputs (e.g., nmap scans) to files
- **Hybrid Evaluation** — **New in v1.1.2!** Built-in "Judge" tool allows agents to request semantic review of their work
- **Auto-Verification** — **New in v1.1.2!** Agents can self-correct by looping through a draft-review-fix cycle automatically

## Quick Start

### Install

```bash
curl -fsSL https://raw.githubusercontent.com/PurpleS3Cf0X/aseity/master/install.sh | sh
```

Or build from source:

```bash
git clone https://github.com/PurpleS3Cf0X/aseity.git
cd aseity
make install
```

### Run

```bash
aseity
```

On first run, Aseity will:
1. Check if Ollama is installed (offers to install if not)
2. Start Ollama if it's not running
3. Download the default model (qwen2.5:14b)
4. Launch the interactive chat

## Usage Examples

### Basic Chat

```bash
# Start with default provider (Ollama) and model (qwen2.5:3b)
aseity

# Use a specific model
aseity --model llama3.2

# Use a different provider
aseity --provider anthropic --model claude-sonnet-4-20250514
# Non-interactive mode (auto-approve all tools)
aseity -y "create a bash file called test.sh"

### Interactive Commands (New in v1.0.5)

Aseity can now handle commands that require user input, such as passwords or confirmation prompts.

```bash
# This will pause and ask you for the sudo password in the UI
aseity -y "install nmap using sudo"
```

### Headless & Scripting (New in v1.0.4)

Aseity can run without the UI, perfect for piping and automation:

```bash
# Explicit headless mode
aseity --headless "Summarize README.md"

# Implicit headless (just by providing a command)
aseity "Check disk space on /" > output.txt

# Pipe output to other tools
aseity -y "Scan localhost ports" | grep "80/tcp"
```

### Parallel Crawling (New in v2.8.1)

Research multiple topics effectively with concurrent browsing:

```bash
aseity "Research Go concurrency patterns on go.dev AND concurrency in Rust on rust-lang.org simultaneously"
```
Aseity will spawn parallel browser instances to fetch both pages at once.

### In the Chat

```
You: What files are in this directory?

Aseity: I'll check the directory contents for you.
  ● bash
  $ ls -la

  [Aseity shows the command and asks for approval]
  Allow bash? [y/n]
```

After you approve, Aseity executes the command and shows the results.

### Slash Commands

Type these in the chat for quick actions:

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Clear conversation history |
| `/compact` | Compress conversation to save context |
| `/save` | Export conversation to markdown |
| `/tokens` | Show estimated token usage |
| `/quit` | Exit aseity |

### Model Management

```bash
# List local models
aseity models

# Pull a new model
aseity pull llama3.2
aseity pull codellama:34b
aseity pull qwen2.5:14b  # Default model
aseity pull deepseek-r1

# Search HuggingFace for models
aseity search "code assistant"

# Remove a model
aseity remove llama2
```

### Diagnostics

```bash
# Check service health
aseity doctor

# Re-run setup wizard
aseity setup

# Setup with Docker instead of local Ollama
aseity setup --docker
```

## Recommended Models

Aseity works best with models that support **function calling** or can follow structured output formats. Here's a guide:

### ✅ Best for Tool Use (Agentic Workflows)
These models reliably execute commands instead of explaining them:

| Model | Size | RAM Required | Best For |
|-------|------|--------------|----------|
| `qwen2.5:14b` | ~8.5GB | 10GB+ | **Recommended** - Best balance of speed & capability |
| `qwen2.5:32b` | ~19GB | 24GB+ | Production workflows, complex tasks |
| `deepseek-r1:7b` | ~4.5GB | 6GB+ | Reasoning-focused, good tool use |
| `llama3.2:8b` | ~5GB | 8GB+ | Fast, decent tool support |

### ⚠️ Limited Tool Support
These models work but may explain instead of executing:

| Model | Size | RAM Required | Notes |
|-------|------|--------------|-------|
| `qwen2.5:3b` | ~2GB | 4GB | **Default** - Fast but inconsistent tool use |
| `llama3.2:3b` | ~2GB | 4GB | Similar limitations to qwen2.5:3b |

### 🚀 Cloud Models (Always Reliable)
- `gpt-4o` (OpenAI) - Excellent function calling
- `claude-sonnet-4` (Anthropic) - Best reasoning + tool use
- `gemini-2.0-flash` (Google) - Fast, reliable

### Upgrading Your Model

```bash
# Pull the recommended model
ollama pull qwen2.5:14b

# Use it
aseity --model qwen2.5:14b
```

**Your System**: M4 Mac with 8GB RAM can comfortably run `qwen2.5:14b` ✅



### Advanced Capabilities

#### Recursive Agents
Aseity can spawn sub-agents to handle complex tasks:

```bash
aseity "Spawn an agent to research the Log4j vulnerability and save a report"
```
The main agent will delegate the task, wait for the sub-agent to finish, and present the final result.

#### Safety & Large Outputs
- **Safety**: `aseity -y` bypasses confirmation, but dangerous commands (like `rm -rf /`) remain blocked by default.
- **Smart Truncation**: If a tool (like `nmap` or `grep`) produces thousands of lines, Aseity automatically saves the **full output** to a temp file and only shows a preview to the agent to prevent context overflow.

### Hybrid Evaluation & Auto-Verification (New in v1.1.2)
Aseity now includes a "Trust but Verify" system.
- **Judge Tool**: An agent can ask for a critique: `judge_output(original_goal="...", content="...")`.
- **Auto-Loop**: When spawning a sub-agent, you can set `require_review: true`. The worker agent will be automatically peer-reviewed by a "Critic" agent. If the critic rejects the work, the worker is re-spawned with feedback to fix it (up to 3 times).

### Consultant Persona (Default)
Aseity comes with a built-in "Consultant" persona that is proactive and inquisitive.
```bash
aseity "Ask the Consultant to review my architecture plans"
```


### Custom Agents (New in v1.1.0+)
You can create persistent agent personas that know exactly how you like to work.

**Creating an Agent:**
Just tell Aseity what you want.
```bash
> Create a 'LinuxExpert' agent that specializes in bash scripting and system administration.
```
Aseity will ask you for details if needed and save the agent to `~/.config/aseity/agents/LinuxExpert.yaml`.

**Knowledge Repositories (New in v1.1.1):**
You can attach specific documentation folders to an agent. It will prioritize searching these paths for answers.
```bash
> Create a 'SecBot' agent with knowledge path '/Users/me/docs/security-protocols'.
```

**Using an Agent:**
Spawn it by name for specific tasks:
```bash
> Ask LinuxExpert to audit my ssh configuration.
```

**Deleting an Agent:**
You can ask Aseity to delete it:
```bash
> Delete the LinuxExpert agent.
```

## Configuration

Create `~/.config/aseity/config.yaml`:

```yaml
# Default provider and model
default_provider: ollama
default_model: qwen2.5:14b

# Provider configurations
providers:
  ollama:
    type: openai
    base_url: http://localhost:11434/v1

  openai:
    type: openai
    base_url: https://api.openai.com/v1
    api_key: $OPENAI_API_KEY
    model: gpt-4o

  anthropic:
    type: anthropic
    api_key: $ANTHROPIC_API_KEY
    model: claude-sonnet-4-20250514

  google:
    type: google
    api_key: $GEMINI_API_KEY
    model: gemini-2.0-flash

# Tool settings
tools:
  auto_approve:
    - file_read
    - file_search
    - web_search
  disallowed_commands:
    - rm -rf /
    - mkfs
```

Environment variables (like `$OPENAI_API_KEY`) are automatically expanded.

## Docker

Run Aseity with Ollama in Docker:

```bash
# Start Ollama container
docker compose up -d ollama

# Run Aseity
docker compose run --rm aseity
```

Or use the Makefile shortcuts:

```bash
make docker-up      # Start Ollama + run Aseity
make docker-up-vllm # Start with vLLM backend
make docker-down    # Stop all services
```

## Available Tools

Aseity can use these tools to help you:

| Tool | Description | Requires Approval |
|------|-------------|-------------------|
| `bash` | Execute shell commands (starts PTY, supports interactive prompts) | Yes |
| `file_read` | Read file contents | No |
| `file_write` | Create or edit files | Yes |
| `file_search` | Find files by pattern or content | No |
| `web_search` | Search the web via DuckDuckGo | No |
| `web_fetch` | Fetch and read web pages | No |
| `web_crawl` | Crawl dynamic websites using a real browser (or HTTP fallback) | No |
| `spawn_agent` | Create a sub-agent (supports recursive delegation) | Yes |
| `list_agents` | List running sub-agents | No |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Alt+Enter` | New line in message |
| `Ctrl+T` | Toggle thinking visibility |
| `Ctrl+C` | Cancel current operation |
| `Esc` | Quit |

## Providers

### Ollama (Local)

Best for: Privacy, offline use, no API costs

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
ollama pull qwen2.5:14b

# Run Aseity
aseity
```

### OpenAI

Best for: GPT-4o, o1, latest OpenAI models

```bash
export OPENAI_API_KEY="sk-..."
aseity --provider openai --model gpt-4o
```

### Anthropic

Best for: Claude models, long context

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
aseity --provider anthropic --model claude-sonnet-4-20250514
```

### Google

Best for: Gemini models

```bash
export GEMINI_API_KEY="..."
aseity --provider google --model gemini-2.0-flash
```

### vLLM (Self-hosted)

Best for: High-performance inference on your own GPU

```bash
# Start vLLM with a model
docker compose --profile vllm up -d

# Use it
aseity --provider vllm
```

## Troubleshooting

### "Connection refused" error

Ollama isn't running. Start it:

```bash
ollama serve
```

Or let Aseity start it for you:

```bash
aseity setup
```

### "Model not found" error

Pull the model first:

```bash
aseity pull qwen2.5:14b
```

### Check service status

```bash
aseity doctor
```

This shows the status of all configured providers.

## Building from Source

Requirements:
- Go 1.23+
- Git

```bash
git clone https://github.com/PurpleS3Cf0X/aseity.git
cd aseity
make build      # Build binary to ./bin/aseity
make install    # Install to $GOPATH/bin
make release    # Cross-compile for all platforms
```

## Project Structure

```
aseity/
├── cmd/aseity/         # Main entry point
├── internal/
│   ├── agent/          # Agent loop, conversation, sub-agents
│   ├── config/         # Configuration loading
│   ├── health/         # Provider health checks
│   ├── model/          # Model management (pull, list, remove)
│   ├── provider/       # LLM providers (OpenAI, Anthropic, Google)
│   ├── setup/          # First-run setup wizard
│   ├── tools/          # Tool implementations
│   └── tui/            # Terminal UI
├── pkg/version/        # Version info
├── docker-compose.yml  # Docker setup
├── Dockerfile
├── Makefile
└── install.sh          # One-liner installer
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT License. See [LICENSE](LICENSE) for details.

---

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).
