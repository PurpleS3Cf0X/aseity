# Crawl4AI Integration Guide

## 🚀 Overview

Aseity now supports advanced web scraping via Crawl4AI, an optional Docker microservice that provides:
- JavaScript rendering with headless browser
- Clean Markdown output for LLMs
- Dynamic content extraction
- Screenshot capabilities
- Bot detection bypass

## 📦 Installation

### Quick Start (Recommended)

```bash
# Run the setup script
./scripts/setup-crawl4ai.sh
```

This will:
1. Check Docker installation
2. Pull the Crawl4AI image
3. Start the service
4. Run health checks
5. Test the integration

### Manual Setup

```bash
# Start Crawl4AI service
docker compose --profile crawl4ai up -d crawl4ai

# Check status
./scripts/check-services.sh
```

## 🎮 Usage

### In Aseity

The `web_crawl` tool is automatically available:

```
User: Crawl https://example.com and extract the main content
Agent: [TOOL:web_crawl|{"url": "https://example.com"}]
```

### Parallel Crawling (New!) 🚀

You can also request multiple URLs at once:

```
User: Research concurrent programming on https://go.dev and https://rust-lang.org
Agent: [TOOL:web_crawl|{"urls": ["https://go.dev", "https://rust-lang.org"]}]
```

The tool will process these in parallel:
- **With Crawl4AI**: Sends a batch request for efficient processing.
- **Without Crawl4AI**: Spawns concurrent headless browsers (up to 3) for faster execution.

### Tool Arguments

```json
{
  "url": "https://example.com",       // Single URL (Legacy)
  "urls": ["https://a.com", "..."],   // List of URLs (Recommended for batch)
  "wait_for": "#content",             // Optional: CSS selector
  "screenshot": true                  // Optional: Capture screenshot
}
```

## 🔧 Configuration

### Memory Optimization

The docker-compose.yml includes memory optimization:

```yaml
# Memory limits
deploy:
  resources:
    limits:
      memory: 2g
    reservations:
      memory: 1g

# Shared memory for browser
shm_size: 512m

# Automatic cleanup
environment:
  - ENABLE_AUTO_CLEANUP=true
  - CLEANUP_INTERVAL=300  # 5 minutes
  - MAX_CACHE_SIZE=500m
```

### Environment Variables

```bash
# In docker-compose.yml or .env
BROWSER_POOL_SIZE=2              # Number of browser instances
MAX_CONCURRENT_REQUESTS=5        # Request queue size
BROWSER_TIMEOUT=30               # Timeout per request
ENABLE_MONITORING=true           # Dashboard
```

## 📊 Monitoring

### Dashboard

Access the Crawl4AI dashboard at:
```
http://localhost:11235/dashboard
```

Shows:
- Browser pool status
- Request queue
- Memory usage
- Performance metrics

### Health Check

```bash
curl http://localhost:11235/health
```

### Logs

```bash
docker logs crawl4ai
```

## 🛠️ Troubleshooting

### Crawl4AI Not Starting

**Problem**: Container fails to start

**Solutions**:
```bash
# Check if port is in use
lsof -i :11235

# Check Docker resources
docker stats crawl4ai

# Restart service
docker restart crawl4ai
```

### Out of Memory

**Problem**: Container killed due to OOM

**Solutions**:
```bash
# Increase Docker memory limit
# Docker Desktop → Settings → Resources → Memory: 4GB

# Or reduce browser pool size
# Edit docker-compose.yml:
BROWSER_POOL_SIZE=1
```

### Slow Performance

**Problem**: Crawls taking too long

**Solutions**:
```yaml
# Disable images for faster loading
DISABLE_IMAGES_ON_LOW_MEM=true

# Reduce timeout
BROWSER_TIMEOUT=20

# Increase browser pool
BROWSER_POOL_SIZE=3
```

### Fallback to Basic Fetch

**Problem**: Tool uses basic HTTP instead of Crawl4AI

**Cause**: Crawl4AI service unavailable

**Check**:
```bash
./scripts/check-services.sh
```

**Fix**:
```bash
docker start crawl4ai
```

## 🔄 Graceful Degradation

If Crawl4AI is unavailable, Aseity automatically falls back to:
1. **Chromedp** (if Chrome installed)
2. **Basic HTTP fetch** (always available)

This ensures Aseity continues working even without Crawl4AI.

## 📈 Performance

### With Crawl4AI

- ✅ JavaScript rendering
- ✅ Clean Markdown
- ✅ Dynamic content
- ⏱️ ~2-5 seconds per page

### Without Crawl4AI (Fallback)

- ⚠️ No JavaScript
- ⚠️ Raw HTML
- ⚠️ Static content only
- ⏱️ ~1-2 seconds per page

## 🎯 Best Practices

### 1. Use for Dynamic Sites

```
✅ Good: Single-page apps (React, Vue)
✅ Good: JavaScript-heavy sites
❌ Overkill: Static HTML pages
```

### 2. Specify wait_for

```json
{
  "url": "https://example.com",
  "wait_for": "#main-content"  // Wait for specific element
}
```

### 3. Monitor Memory

```bash
# Check memory usage
docker stats crawl4ai

# Clean up cache
docker exec crawl4ai rm -rf /app/cache/*
```

## 🔐 Security

### Network Isolation

Crawl4AI runs in isolated Docker network:
```yaml
networks:
  - aseity-net
```

### Resource Limits

Prevents resource exhaustion:
```yaml
mem_limit: 2g
cpus: '2.0'
```

### Log Rotation

Prevents disk bloat:
```yaml
logging:
  options:
    max-size: "10m"
    max-file: "3"
```

## 🚦 Service Management

### Start

```bash
docker compose --profile crawl4ai up -d crawl4ai
```

### Stop

```bash
docker stop crawl4ai
```

### Restart

```bash
docker restart crawl4ai
```

### Remove

```bash
docker compose down crawl4ai
docker volume rm aseity12_crawl4ai_cache
```

## 📚 API Reference

### Endpoints

```
POST /crawl
GET /health
GET /dashboard
GET /playground
```

### Example Request

```bash
curl -X POST http://localhost:11235/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://example.com"],
    "priority": 10
  }'
```

## ✅ Verification

Run the test suite:

```bash
# Test Crawl4AI service
curl http://localhost:11235/health

# Test from Aseity
# In Aseity TUI:
> Use web_crawl to fetch https://example.com
```

## 🆘 Support

### Check Status

```bash
./scripts/check-services.sh
```

### View Logs

```bash
docker logs crawl4ai --tail 100 -f
```

### Reset Service

```bash
docker compose down crawl4ai
docker volume rm aseity12_crawl4ai_cache
./scripts/setup-crawl4ai.sh
```

---

## 🎉 Summary

Crawl4AI adds powerful web scraping to Aseity with:
- ✅ One-command setup
- ✅ Automatic fallback
- ✅ Memory optimization
- ✅ Health monitoring
- ✅ Zero configuration

**Ready to use!** 🚀
