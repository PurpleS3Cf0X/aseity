# Aseity: Issues and Improvement Opportunities

## Critical UX Issues

### 1. Unknown Commands Silently Ignored
**[RESOLVED]**
The `main.go` file now handles unknown subcommands in the `default` case of the switch statement (lines 82-85), printing an error and showing help.

### 2. Poor Help Output
**[RESOLVED]**
A comprehensive `showHelp` function has been implemented (lines 455-502 of `main.go`) which lists all subcommands, flags, and examples.

### 3. Version Shows "dev (unknown)"
**Problem:** When built with `go build ./...`, version info is not embedded.
**Fix:** Build with `make build` or document the required ldflags

### 4. Model Auto-Download Without Consent
**[RESOLVED]**
The `main.go` file now prompts the user before pulling a model: `Download +modelName+ now? [Y/n]` (lines 137-147).

---

## UX Improvements Needed

### 5. No Welcome Message in TUI
**[RESOLVED]**
The `NewModel` function in `internal/tui/app.go` now initializes `m.messages` with a welcome message (lines 137-141).

### 6. No Typing/Waiting Indicator Before First Token
**Problem:** After sending a message, user waits with no feedback until first token arrives.
**Fix:** Show "Connecting to model..." or similar immediately

### 7. Doctor Shows Failures for Optional Services
**Problem:** vLLM is optional, but doctor shows "✗" making it look like a failure.
```
  ● vllm ... ✗ cannot reach http://localhost:8000/v1
  Some services are unreachable.    <- Scary message for optional service
```
**Fix:** Distinguish required vs optional services, or only warn about configured provider

### 8. No Way to Gracefully Cancel Model Download
**Problem:** During large model downloads, Ctrl+C behavior is undefined.
**Fix:** Handle SIGINT properly in PullModel function

### 9. Tool Result Truncation Not Obvious
**Problem:** Tool results are truncated at 500 chars in TUI, but full result is in conversation context.
**Fix:** Show "[truncated - full output sent to model]" clearly

### 10. Subcommands Not Listed in Help
**[RESOLVED]**
See item #2.

---

## Missing Features (vs Claude Code)

### High Priority

| Feature | Description | Effort |
|---------|-------------|--------|
| **Welcome screen** | Show tips, example prompts when TUI starts | Low |
| **Better help** | Full help with subcommands and examples | Low |
| **Command validation** | Error on unknown subcommands | Low |
| **Diff view for edits** | Show what changed in file edits | Medium |
| **Streaming bash output** | Show output as command runs, not after | Medium |
| **Conversation restore** | Resume previous sessions | Medium |
| **Syntax highlighting** | Highlight code in responses | Medium |

### Medium Priority

| Feature | Description | Effort |
|---------|-------------|--------|
| **Multi-line input** | Better UX for pasting code blocks | Medium |
| **Tab completion** | Complete file paths and commands | Medium |
| **Git shortcuts** | /commit, /diff, /status commands | Medium |
| **Cost estimation** | Show $ cost for cloud providers | Medium |
| **Clipboard integration** | Copy code blocks easily | Medium |
| **Project awareness** | Auto-detect repo structure, languages | Medium |

### Lower Priority

| Feature | Description | Effort |
|---------|-------------|--------|
| **Image support** | Read/process image files | High |
| **MCP support** | Model Context Protocol integration | High |
| **Memory/preferences** | Remember user preferences between sessions | Medium |
| **Export to JSON** | In addition to markdown export | Low |
| **URL preview** | Preview URLs mentioned in chat | Medium |
| **Search engine options** | Support Google, Bing, not just DDG | Low |

---

## Code Quality Issues

### 1. No Input Validation for Subcommands
```go
// Current code - unknown commands fall through to TUI
if len(args) > 0 {
    switch args[0] {
    case "models": ...
    case "pull": ...
    // No default case!
    }
}
```

### 2. Hardcoded Search Engine
Only DuckDuckGo is supported. Should be configurable.

### 3. No Rate Limiting
Could hit API rate limits on rapid tool usage.

### 4. Web Fetch Redirect Handling
Some URLs with redirects may fail.

### 5. No Caching
Repeated file reads within same session aren't cached.

---

## Recommended Priority Order

1. **Fix unknown command handling** (5 min)
2. **Add comprehensive help output** (30 min)
3. **Add welcome screen to TUI** (30 min)
4. **Add confirmation before auto-pull** (10 min)
5. **Fix doctor to distinguish optional services** (15 min)
6. **Add diff view for file edits** (2 hrs)
7. **Add syntax highlighting for code** (3 hrs)
8. **Add streaming bash output** (2 hrs)
9. **Add conversation restore** (3 hrs)
10. **Add git shortcuts** (2 hrs)

---

## Quick Wins (< 1 hour each)

1. Unknown command error message
2. Better `--help` output
3. Welcome message in TUI
4. Confirmation before model download
5. Fix doctor optional service display
6. Add `/model` command to switch models
7. Add `/provider` command to switch providers
8. Show keyboard shortcuts overlay on first launch
