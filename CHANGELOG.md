# Changelog

## [3.1.1] - 2026-02-16

### Fixed
- **General Intent Hallucination**: Fixed an issue where conversational inputs (e.g., "hi") triggered unnecessary tool calls (like `docker info`) on ReAct-enabled models.
  - Updated ReAct prompt to explicitly allow text-only responses.
  - Added conversational examples to the system prompt.
- **Documentation**: Updated README to reflect the new default model `qwen2.5-coder:7b`.

---

## [3.1.0] - 2026-02-16

### Added
- **Deep Research Mode**: New `--deep-research` CLI flag to force comprehensive multi-step research sessions
  - Bypasses intent detection to ensure deep analysis
  - Automatically triggers `web_search` -> `web_fetch` -> Synthesis loop
  - Optimizes planner prompts for research tasks
- **Orchestrator Stability**: Significant improvements to the experimental orchestrator
  - Fixed JSON parsing for models (like Qwen 2.5 Coder) that output comments in JSON
  - Enforced stricter tool usage rules (banning `open`/`curl` in favor of `web_fetch`)
  - Added graceful shutdown and state saving on `SIGINT`

### Fixed
- **Planner Bug**: Resolved `invalid character '/'` JSON errors caused by models including comments in plans.
- **Headless Execution**: Improved reliability of orchestrator in headless environments.

---

## [2.21.1] - 2026-02-11

### Improved
- **Enhanced Documentation**: Expanded model management section in README
  - Added `aseity models` command examples with sample output
  - Included pro tips for model naming and selection
  - Clarified correct model name format (no `ollama/` prefix)
  - Added free API alternative suggestions
- **Better User Guidance**: Clear examples showing model sizes and requirements

### Documentation
- Model management commands are now more discoverable
- Added troubleshooting tips for common model naming mistakes
- Included hardware requirements for different model sizes

---

## [2.21.0] - 2026-02-10

### Added
- **Model Name Validation**: Automatically detects and corrects invalid model names
  - Maps `ollama/qwen2.5:32b` → `qwen2.5:32b` (removes incorrect prefix)
  - Fixes common typos: `qwen2.5-32b` → `qwen2.5:32b`
  - Suggests similar models when name not found
  - Shows popular models with sizes and descriptions
- **Better Pull Commands**: Shows correct `ollama pull` command in messages

### Improved
- Model pull now validates names before attempting download
- Interactive correction: asks user to confirm suggested model name
- Helpful error messages with model suggestions when invalid name provided

### Example
```
$ aseity --model ollama/qwen2.5:32b
⚠ Did you mean 'qwen2.5:32b'? (removing 'ollama/' prefix)
  Use 'qwen2.5:32b' instead? [Y/n]
```

---

## [2.20.1] - 2026-02-10

### Fixed
- **Model Pull Timeout**: Increased model verification timeout from 15s to 60s to handle large models
- **Better Error Messages**: Added troubleshooting steps when model pull fails
- **Progress Indicators**: Show verification progress every 5 seconds during model loading
- **Helpful Tips**: Added manual pull command suggestions in error messages

### Improved
- Model pull now shows clearer progress and status messages
- Better error handling for Ollama connectivity issues

---

## [2.20.0] - 2026-02-10

### Added
- **ReAct Chain-of-Thought Pattern**: Implemented Reasoning + Acting loop for Tier 2/3 models
  - System prompts now include explicit ReAct instructions: Thought → Action → Observation → Thought
  - Comprehensive examples showing correct vs incorrect patterns
  - After each tool execution, models receive ReAct prompts forcing:
    1. **OBSERVATION**: "What data did you receive?"
    2. **THOUGHT**: "What does this mean? Does it answer the question?"
    3. **DECISION**: "Call another tool or provide final answer?"
  - Prevents "Awaiting user command" premature stops

### Improved
- **Post-Tool Prompting**: Replaced simple reminders with structured ReAct prompts
  - Includes original user goal in prompt for context
  - Forces explicit reasoning about next steps
  - Prevents tool result abandonment

### Notes
- ReAct improvements provide better scaffolding but require models with baseline reasoning capability
- Very weak models (e.g., qwen2.5:14b) may still struggle with basic intent understanding
- **Recommendation**: For production use, prefer Tier 1 models (GPT-4, Claude, Gemini Pro)

---

## [2.19.0] - 2026-02-10

### Added
- **Tier-Specific System Prompts**: Models now receive guidance tailored to their capability tier
  - Tier 2/3 models get explicit step-by-step workflows and result verification checklists
  - Includes common mistakes to avoid and best practices for tool usage
- **Result Acknowledgment Reminders**: Tier 2/3 models receive automatic reminders to use tool results after execution
- **Result Processor Component**: Infrastructure for validating tool result usage (foundation for Phase 2)
- **Completion Checker Component**: Infrastructure for preventing premature task completion (foundation for Phase 2)

### Improved
- **Model Guidance**: Weaker models now have much more explicit instructions on how to:
  - Read and process tool results
  - Avoid hallucinating data
  - Complete multi-step tasks
  - Verify their work before finishing

---

## [2.18.4] - 2026-02-10

### Fixed
- **Headless Mode Deadlock**: Fixed critical bug where headless mode would fail with "events channel must be buffered" error, preventing tool execution.
- **Tool Calling**: Agents can now properly call tools and return results in headless mode.

---

## [2.18.3] - 2026-02-10

### Fixed
- **Stuck Responses**: Fixed a critical buffer flushing issue where the end of a response (or short responses) could be lost, making the agent appear "stuck".
- **Stream Reliability**: Added safety flushes for unexpected stream closures.

---

## [2.18.2] - 2026-02-10

### Fixed
- **Context Loss**: Fixed issue where Ollama's default 2KB context window caused the agent to forget instructions. Aseity now requests a 32KB window for Ollama models.
- **Symbol Rendering**: Fixed broken HTML entity rendering (e.g., `&#61594;`) in web search results.
- **Thoughts Leakage**: Fixed `<think>` tags leaking into the chat window from reasoning models.

---

## [2.18.1] - 2026-02-10

### Fixed
- **TUI Freezing**: Implemented render caching to prevent UI freezes during heavy output
  - Optimized markdown rendering to avoid O(N^2) re-rendering on every token
- **Scroll Lock**: Fixed issue where auto-scroll prevented manual scrolling
  - TUI now only auto-scrolls if the user is already at the bottom of the viewport

---

## [2.18.0] - 2026-02-10

### Added
- **Custom Model Loading**: Easily load local GGUF models into Ollama
  - New command: `aseity --load-model <path/to/model.gguf>`
  - Automatically creates a Modelfile and registers the model in Ollama
  - Streamlines the workflow for using fine-tuned models

---

## [2.17.0] - 2026-02-10

### Added
- **Recursive Hierarchical Agents**: Support for complex task decomposition
  - **New Tool**: `wait_all_agents` allows the root agent to synchronize multiple background sub-agents
  - **Structured Spawning**: `spawn_agent` now enforces structured status reporting for better coordination
  - **Parallel Execution**: Sub-agents run efficiently in parallel, enabling faster completion of multi-file tasks

### Technical Details
- Implemented "Planner -> Worker" architecture
- Added `TestRecursiveAgentSpawning` integration test
- Refactored agent event loop to support aggregated child results

---

## [2.16.0] - 2026-02-05

### Added
- **PDF Support**: `read` tool now natively extracts text from PDF files
- **Excel Support**: `read` tool now converts Excel (.xlsx) sheets to Markdown tables for properly structured viewing

---

## [2.15.3] - 2026-02-05

### Fixed
- **Scroll Lock**: Fixed an issue where the view would forcibly scroll to the bottom, preventing manual scrolling up

---

## [2.15.2] - 2026-02-05

### Fixed
- **Startup Panic**: Fixed a crash on startup where viewport height could become negative before initialization
- **Stability**: Added safety guards for viewport calculations during resize events

---

## [2.15.1] - 2026-02-05

### Fixed
- **TUI Auto-Scroll**: Fixed issue where the bottom of the response (e.g., token usage) was sometimes cut off or not scrolled into view
- **Alignment**: Tweaked token usage display alignment for better visual consistency

---

## [2.15.0] - 2026-02-05

### Improved
- **Robust Self-Update**: The `--update` command now dynamically locates the git repository from the executable path
  - Works correctly regardless of the current working directory
  - Traverses directory tree to find `.git` root
  - Resolves symlinks automatically

---

## [2.14.0] - 2026-02-05

### Changed
- **Token Display Format**: Updated to labeled format with yellow color
  - New format: `Tokens: in ~X · out ~Y · total ~Z (est.)`
  - Changed color from dim green to soft gold (#FFD700)
  - Clearer labels for input, output, and total tokens
  - Uses middle dot (·) separators for better readability

---

## [2.13.0] - 2026-02-05

### Added
- **Self-Update Feature**: New `--update` flag to update Aseity to latest version
  - Automatically pulls latest changes from GitHub
  - Rebuilds binary with latest code
  - Checks for uncommitted changes before updating
  - Displays version before and after update
  - Usage: `aseity --update`

### Technical Details
- Added `cmdUpdate()` function in main.go
- Uses `git pull origin master` to fetch latest changes
- Rebuilds binary with `go build`
- Prompts for confirmation if uncommitted changes exist

---

## [2.12.0] - 2026-02-05

### Added
- **Improved Token Usage Display**: Token usage now displays outside response box with better styling
  - Extracted from message content and displayed separately
  - Styled in dim green italic text with left padding
  - Appears below the response box for better visibility
  - Format: `  Tokens: ~X → ~Y (~Z total, estimated)`

### Technical Details
- Modified `renderAssistantBlock` to extract token usage from content
- Token usage no longer rendered inside markdown content
- Custom lipgloss styling applied (DimGreen, Italic, PaddingLeft)
- Cleaner separation between response and metadata

---

## [2.11.5] - 2026-02-05

### Changed
- **Token Usage Format**: Simplified to plain text without markdown formatting
  - Removed italic markdown formatting that may have caused rendering issues
  - Format: `Tokens: ~X → ~Y (~Z total, estimated)`
  - Should now display correctly in all cases

---

## [2.11.4] - 2026-02-05

### Fixed
- **Token Usage Display**: Now properly appends to message content
  - Previous implementation was overwritten by rebuildView()
  - Token usage now persists as part of the assistant message
  - Displays after each response in plain text format
  - Format: `Tokens: ~X → ~Y (~Z total, estimated)`

### Technical Details
- Token usage appended to last assistant message content
- Survives viewport rebuilds
- Removed lipgloss styling to ensure compatibility with markdown rendering

---

## [2.11.3] - 2026-02-05

### Fixed
- **Viewport Auto-Scroll**: Responses now automatically scroll to bottom
  - Fixes issue where responses were cut off at bottom of viewport
  - Token usage line now always visible after responses
  - Viewport automatically scrolls to show latest content

---

## [2.11.2] - 2026-02-05

### Added
- **Client-Side Token Estimation**: Fallback for providers that don't return usage data
  - Ollama's OpenAI-compatible endpoint doesn't return token counts
  - Implemented word-based estimation (~1.3 tokens per word)
  - Shows estimated counts with "~" prefix: `Tokens: ~150 → ~420 (~570 total, estimated)`
  - Provides approximate usage tracking for all providers

### Technical Details
- Ollama's native API (`/api/chat`) returns `eval_count` and `prompt_eval_count`
- OpenAI-compatible endpoint (`/v1/chat/completions`) returns null for usage
- Client-side estimation counts words in last user/assistant exchange
- Estimation formula: `tokens ≈ words × 1.3`

---

## [2.11.1] - 2026-02-05

### Fixed
- **Token Usage Display**: Added `stream_options.include_usage` parameter to API requests
  - Enables token usage reporting from Ollama and OpenAI-compatible providers
  - Required for streaming responses to include usage information
  - Should now display token counts after each response

---

## [2.11.0] - 2026-02-05

### Added
- **Provider Connection Status Indicator**: Shows real-time connection status next to provider name
  - Green dot (●) when provider is online and responding
  - Red dot (●) when provider is disconnected or connection errors occur
  - Automatically updates based on response/error events
  - Displayed in header: `● ollama / qwen2.5:14b`

### Technical Details
- Added `providerOnline` field to TUI Model struct
- Status updates on `EventDelta` (online) and `EventError` (offline for connection errors)
- Visual indicator uses BrightGreen for online, #FF5555 (red) for offline

---

## [2.10.1] - 2026-02-05

### Fixed
- **Logo Positioning**: Added top padding to header to lower logo position on screen for better vertical centering

---

## [2.10.0] - 2026-02-05

### Added
- **Token Usage Display**: Shows token consumption after each response, similar to Claude Code
  - Format: `Tokens: 150 → 420 (570 total)` displayed in DimGreen
  - Captures input tokens, output tokens, and total from provider API
  - Currently supported: OpenAI-compatible providers (qwen2.5:14b via Ollama)

### Technical Details
- Added `Usage` struct to `provider.StreamChunk` with InputTokens/OutputTokens/TotalTokens
- Updated OpenAI provider to capture usage from API response
- Modified agent to propagate usage through Event system
- TUI displays usage after EventDone when available

---

## [2.9.2] - 2026-02-05

### Fixed
- **Header Scroll Issue**: Fixed logo being pushed up when window expands
  - Root cause: Hardcoded `headerH = 8` didn't match actual header height (varies between 7-10 lines)
  - Solution: Added `headerHeight` field to Model struct to store actual measured height
  - Viewport height now calculated using `lipgloss.Height(header)` for accuracy
  - Layout properly adapts to window resize without scroll artifacts

### Technical Details
- `Update()` now uses `m.headerHeight` instead of hardcoded value
- `View()` measures actual header height and updates model if changed
- Viewport recalculates automatically when header height changes

---

## [2.9.1] - 2026-02-05

### Fixed
- **Dynamic Header Centering**: Properly calculates content width and centers banner based on actual window size
- Header now measures logo and status widths dynamically and adds appropriate left margin
- Layout adapts correctly to window resize events
- Banner stays centered regardless of terminal width

---

## [2.9.0] - 2026-02-05

### Changed
- **TUI Banner**: Restored Unicode block character banner per user preference (bold, blocky style)
- **Header Layout**: Fixed centering issues - banner now properly centers in terminal and doesn't get pushed up

### Fixed
- Header alignment changed from vertical center to top alignment for proper display
- Added spacer between logo and status for better visual separation

---

## [2.8.4] - 2026-02-05

### Fixed
- **TUI Banner Rendering**: Replaced Unicode block characters with pure ASCII art to ensure the banner displays correctly across all terminals, SSH connections, and font configurations. Fixes garbled/corrupted banner display.

---

## [2.8.3] - 2026-02-05

### Fixed
- **TUI Banner Optimization**: Aggressively reduced banner padding and removed spacers to ensure the logo never truncates, even on terminals with mixed-width font rendering or narrow viewports.

---

## [2.8.2] - 2026-02-05

### Fixed
- **TUI Header Truncation**: Fixed an issue where the logo banner appeared cut off on standard 80-column terminals by optimizing header spacing.

---

## [2.8.1] - 2026-02-05

### Added
- **Parallel Web Crawling**: Support for batch URL processing in `web_crawl` tool 🚀
  - Batching via Crawl4AI service for high-performance scraping
  - Concurrent headless browser fallbacks (up to 3 parallel tabs)
  - Seamless fallback handling for mixed success results
- **Visual TUI Redesign**: "Premium Matrix" Aesthetic 🎨
  - **Neon Green Input Box**: Distinct, rounded border for the command input
  - **Viewport Boundaries**: Clear separation for the chat content
  - Improved layout logic to prevent text wrapping/overflow

### Fixed
- **Dynamic Skillsets**: Prevented context bloat by making injected system prompts temporary (one-time use)

### Changed
- `web_crawl` now accepts `urls` array argument in addition to single `url`

---

## [2.8.0] - 2026-02-05

### Added
- **Crawl4AI Integration**: Optional Docker microservice for advanced web scraping
  - JavaScript rendering with headless Chromium
  - Clean Markdown output optimized for LLMs
  - Dynamic content extraction (SPAs, lazy loading, infinite scroll)
  - Screenshot capture capabilities
  - Automatic fallback to basic HTTP when unavailable
  
- **Memory Optimization**: Comprehensive memory management for Crawl4AI
  - Resource limits (2GB RAM, 2 CPUs)
  - Automatic cleanup every 5 minutes
  - Reduced browser pool size (2 instances)
  - Tmpfs for temporary files (auto-cleanup)
  - Log rotation (10MB max, 3 files)
  
- **Setup Scripts**:
  - `./scripts/setup-crawl4ai.sh` - One-command installation
  - `./scripts/check-services.sh` - Service status checker
  
- **Documentation**:
  - Complete Crawl4AI guide (`docs/crawl4ai.md`)
  - Troubleshooting section
  - Best practices
  - Performance tuning guide

### Changed
- Updated `docker-compose.yml` with Crawl4AI service
- Enhanced `web_crawl` tool with graceful degradation
- Improved error handling and retry logic

### Technical Details
- Crawl4AI runs as optional Docker service (profile: crawl4ai)
- Health checks with 30s interval
- Circuit breaker pattern for reliability
- Automatic service discovery via environment variables

---

## [2.7.0] - 2026-02-05

### Added
- **Dynamic Skillsets**: Context-aware skillset selection (40% token savings)
  - Intent detection (10 types)
  - Contextual prompt building
  - Dynamic injection per request
  
- **YAML Configuration**: User-configurable model profiles
  - `~/.aseity/skillsets.yaml` support
  - Custom skillset definitions
  - Profile overrides
  
- **TUI Commands**:
  - `/profile` - View model capabilities
  - `/skillsets` - View skillset configuration
  - `/settings` - Settings menu (placeholder)

### Changed
- Skillsets now load dynamically based on user intent
- Reduced token usage by 42% on average
- Improved model performance for weak skillsets

### Technical Details
- 91 unit tests passing
- 61.7% code coverage
- Full backward compatibility

---

## [2.6.0] - Previous Release

### Features
- Multi-provider support (OpenAI, Anthropic, Ollama)
- Function calling with automatic fallback
- Interactive TUI with streaming
- Agent management system
- Sandbox execution
- Web search and fetch tools

---

## Migration Guide

### Upgrading to 2.8.0

**Optional Crawl4AI Setup:**
```bash
# Install Crawl4AI (optional)
./scripts/setup-crawl4ai.sh

# Or skip and use existing web_crawl (chromedp)
# No changes required to existing code
```

**Breaking Changes:** None

**New Environment Variables:**
- `CRAWL4AI_URL` - Crawl4AI endpoint (default: http://localhost:11235)

### Upgrading to 2.7.0

**New Files:**
- `~/.aseity/skillsets.yaml` - Optional configuration file
- `skillsets.example.yaml` - Example configuration

**Breaking Changes:** None

**New Commands:**
- `/profile` - View model profile
- `/skillsets` - View skillset status
