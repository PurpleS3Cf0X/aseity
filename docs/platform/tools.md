# Tools

Tools are the hands of the AI. Aseity is equipped with a powerful set of tools to interact with your system and the web.

## System Tools

### `bash`
The primary tool for system interaction. Aseity runs commands in a pseudo-terminal (PTY), meaning it can handle interactive commands.

- **Capabilities**: Run scripts, install packages (`brew`, `apt`), check system stats, run git commands.
- **Interactivity**: If a command asks for a password (like `sudo`) or confirmation (`[y/n]`), Aseity's UI allows you to input the response directly.
- **Approval**: Dangerous commands (like `rm`, `dd`) require explicit user approval unless running in `-y` mode.

### `spawn_agent`
Allows the main agent to delegate work. See [Custom Agents](agents.md).

## File Tools

### `file_read`
Reads the content of a file.
- **Limits**: Reads up to 2000 lines by default.
- **Line Numbers**: Adds line numbers to help with editing.

### `file_write`
Creates or edits files.
- **Modes**:
  - **Create**: Write entirely new content.
  - **Edit**: Replace a specific string block (`old_string`) with new content (`new_string`).

### `file_search`
Finds files in your project.
- **Capabilities**: Fuzzy search filenames or grep content within files.

## Web Tools

### `web_search`
Performs a search using DuckDuckGo.
- **Use Case**: Finding documentation, error solutions, or recent news.

### `web_fetch`
Downloads the raw HTML of a page and converts it to readable Markdown.
- **Use Case**: Reading a specific documentation page found via search.

### `web_crawl` (New in v1.0.8)
A powerful crawler using a headless Chrome browser.
- **Use Case**: Reading dynamic, JavaScript-heavy websites (SPAs) that `web_fetch` cannot handle.
- **Features**: Can wait for specific elements to load before capturing text.

## Reliability Features

### Text-Based Fallback (New in v1.1.0)
Aseity includes a robust fallback mechanism. If a model tries to call a tool but fails to use the correct API format (common with smaller local models), Aseity scans its text response for patterns like:
`[TOOL:bash|{"command": "ls"}]`
And executes it automatically. This ensures high reliability across different LLM providers.

## Review Tools (v1.1.2)

### `judge_output`
Spawns a specialized "Critic" agent to semantic review your work.
- **Parameters**: 
  - `original_goal`: The user's original request.
  - `content`: The output to review (code, plan, text).
- **Output**: JSON verdict (`pass` or `fail`) with feedback.
- **Use Case**:
  > "Review this Python script to ensure it handles edge cases."
  ```bash
  judge_output(original_goal="Write robust python script", content="...")
  ```
