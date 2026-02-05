# Changelog

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
