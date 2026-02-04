# User Interface (TUI)

Aseity features a modern, keyboard-centric Terminal User Interface (TUI) built for efficiency and aesthetics.

## Key Bindings

| Key Combination | Action |
|-----------------|--------|
| **Enter** | Send message to the agent. |
| **Alt+Enter** | Insert a new line in the input box without sending. |
| **Ctrl+C** | Interrupt the current action. If generating, it stops the stream. If thinking, it cancels the thought. Pressing twice exits the app. |
| **Ctrl+T** | Toggle "Thinking" visibility. Useful for reasoning models (like DeepSeek-R1) to hide/show their internal monologue. |
| **Esc** | Exit the application. |

## Interactive Prompts
When a tool requires permission (e.g., executing a bash command), a confirmation dialog appears.
- **y / Enter**: Approve the action.
- **n**: Deny the action.

## Slash Commands
Use these commands in the chat for quick actions:

- `/help`: Show this list of commands.
- `/clear`: Reset the current conversation context (wipes history).
- `/compact`: Summarize previous turns to save context window tokens.
- `/save [filename]`: Export the current chat history to a Markdown file.
- `/tokens`: Show current token usage stats.
- `/quit`: Exit Aseity.

## Visuals
- **Spinner**: Shows when the agent is "thinking" or running a tool.
- **Markdown**: Response text is actively rendered with syntax highlighting for code blocks.
- **Status Bar**: Shows the current model, provider, and tool status.
