# Setup & Installation

Welcome to Aseity! This guide will get you up and running quickly.

## Requirements
- **Operating System**: macOS (Apple Silicon/Intel) or Linux.
- **Dependencies**: 
  - `curl` (for quick install)
  - `git` & `go` 1.23+ (only if building from source)
  - `docker` (optional, for containerized usage)

## Quick Install (Recommended)
The fastest way to install Aseity is via our install script:

```bash
curl -fsSL https://raw.githubusercontent.com/PurpleS3Cf0X/aseity/master/install.sh | sh
```

This will:
1. Detect your OS and architecture.
2. Download the latest binary.
3. Install it to your user path (typically `~/bin` or `/usr/local/bin`).

## Building from Source
If you prefer to compile it yourself:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/PurpleS3Cf0X/aseity.git
   cd aseity
   ```

2. **Build and Install**:
   ```bash
   make install
   ```
   This compiles the project and moves the binary to `$GOPATH/bin`. Ensure this directory is in your `$PATH`.

## First Run
Once installed, simply run:

```bash
aseity
```

### The Setup Wizard
On your first launch, Aseity runs a helpful setup wizard:
1. **Ollama Check**: It looks for a local installation of [Ollama](https://ollama.ai).
   - If not found, it offers to install it for you.
   - If found but stopped, it attempts to start the service.
2. **Model Download**: It checks for the default model (`qwen2.5:14b`).
   - If missing, it will pull it automatically (approx 2GB download).
3. **Launch**: Once ready, it drops you directly into the chat interface.

## Configuration
Aseity uses a configuration file located at `~/.config/aseity/config.yaml`.
While the wizard sets up defaults, you can edit this file to:
- Change default providers (e.g., switch to OpenAI).
- Set API keys.
- Configure allowed/disallowed tools.

See [Provider Configuration](providers.md) for details.
