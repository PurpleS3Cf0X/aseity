# Providers & Configuration

Aseity supports multiple LLM providers, allowing you to mix and match local privacy with cloud power.

## Configuration File
All settings are stored in `~/.config/aseity/config.yaml`.

### Basic Structure
```yaml
default_provider: ollama
default_model: qwen2.5:3b

providers:
  ollama:
    type: openai
    base_url: http://localhost:11434/v1
    
  openai:
    type: openai
    api_key: $OPENAI_API_KEY
    model: gpt-4o

tools:
  auto_approve:
    - file_read
    - web_search
```

## Supported Providers

### 1. Ollama (Local)
Best for running models on your own machine (Llama 3, Qwen, Mistral).
- **Setup**: Install Ollama (`curl -fsSL https://ollama.ai/install.sh | sh`).
- **Config**: 
  ```yaml
  ollama:
    type: openai
    base_url: http://localhost:11434/v1
  ```
- **Usage**: `aseity --provider ollama --model qwen2.5:3b`

### 2. OpenAI
Best for state-of-the-art reasoning (GPT-4o).
- **Setup**: Get an API key from OpenAI.
- **Config**:
  ```yaml
  openai:
    type: openai
    api_key: sk-proj-... # Can use $ENV_VARS here
    model: gpt-4o
  ```

### 3. Anthropic
Best for large Context Windows and coding (Claude 3.5 Sonnet).
- **Setup**: Get an API Key.
- **Config**:
  ```yaml
  anthropic:
    type: anthropic
    api_key: $ANTHROPIC_API_KEY
    model: claude-3-5-sonnet-20240620
  ```

### 4. Google Gemini
Best for speed and huge context (Gemini 2.0 Flash).
- **Setup**: Get a Google AI Studio key.
- **Config**:
  ```yaml
  google:
    type: google
    api_key: $GEMINI_API_KEY
    model: gemini-2.0-flash
  ```

### 5. vLLM (Self-Hosted)
For high-performance inference on your own GPU server.
- **Config**: treat it like an OpenAI endpoint.
  ```yaml
  vllm:
    type: openai
    base_url: http://my-gpu-server:8000/v1
    api_key: ignored
  ```
