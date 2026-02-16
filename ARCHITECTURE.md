# Aseity Architecture (v3.0.0)

Aseity is designed as a modular, event-driven terminal AI assistant. It operates in two distinct modes: **Standard Agent** (single-loop) and **Orchestrator** (multi-agent pipeline).

## High-Level Overview

```mermaid
graph TD
    User[User / Terminal] --> |Input| App[Cmd / TUI]
    App --> |Config| Config[Viper Configuration]
    App --> |Events| Agent[Active Agent]
    
    subgraph Core "Core Logic"
        Agent --> |Uses| Registry[Tool Registry]
        Agent --> |Uses| Memory[Memory Store]
        Agent --> |Calls| Provider[Provider Interface]
    end

    subgraph MemorySystem "Memory & Context"
        Memory --> |Persists| JSON[JSON Store]
        Memory -.-> |Future| Vector[Vector DB (RAG)]
    end

    subgraph Providers "LLM Providers"
        Provider --> OpenAI
        Provider --> Anthropic
        Provider --> Google
        Provider --> Ollama[(Ollama / Local)]
    end

    Registry --> Tools[Tools (Bash, Files, Web)]
```

## Core Components

### 1. The Agent Loop (`internal/agent`)
The core of Aseity is an event-driven loop that follows the ReAct (Reason + Act) pattern.
- **Input**: User messages or tool outputs.
- **Process**: 
  1. Determine intent (Chat vs Tool Call).
  2. Execute tools via `Registry`.
  3. Update conversation history.
  4. Stream feedback to TUI via `events` channel.
- **Output**: Final response or follow-up question.

### 2. Orchestrator Mode (`internal/orchestrator`)
For complex queries, Aseity switches to a multi-stage pipeline:
1.  **Intent Parser**: Analyzes requirements.
2.  **Planner**: Generates a dependency graph of steps.
3.  **Executor**: Runs steps (supports parallelism).
4.  **Validator**: Critiques results against the goal.
5.  **Synthesizer**: Formats the final answer.

### 3. Provider Abstraction (`internal/provider`)
A unified interface decouples the application from specific LLM APIs.
- **Standardization**: All providers (OpenAI, Anthropic, Google, Ollama) conform to a single `Chat()` and `Stream()` interface.
- **Normalization**: differences in tool call formats and error codes are handled internally, identifying "Chat-Only" models automatically.

### 4. Memory System (`internal/memory`)
Refactored in v3.0.0 to use a `Store` interface.
- **AutoMemory**: Persists user preferences and facts to `~/.config/aseity/memory/auto_memory.json`.
- **ProjectContext**: Loads project-specific rules from `ASEITY.md` or `CLAUDE.md`.
- **Interface**: Designed to plug in a Vector DB implementation in the future without changing agent logic.

### 5. Configuration (`internal/config`)
Managed by `spf13/viper`.
- **Precedence**: Flags > Env Vars (`ASEITY_MODEL`) > Config File > Defaults.
- **Formats**: Supports `config.yaml`, `.env`, and automatic environment expansion.

## Directory Structure

- `cmd/aseity`: Entry point and CLI flags.
- `internal/agent`: Core logic and state management.
- `internal/tui`: Bubble Tea interface (View layer).
- `internal/tools`: Tool implementations (Controller layer).
- `internal/model`: Ollama management commands.
- `internal/health`: Diagnostic checks (`doctor`).
