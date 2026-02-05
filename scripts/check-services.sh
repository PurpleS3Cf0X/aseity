#!/bin/bash
# Check status of all Aseity services

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    local service=$1
    local status=$2
    local details=$3
    
    if [ "$status" = "running" ]; then
        echo -e "${GREEN}✓${NC} $service: Running $details"
    elif [ "$status" = "stopped" ]; then
        echo -e "${YELLOW}○${NC} $service: Stopped $details"
    else
        echo -e "${RED}✗${NC} $service: $details"
    fi
}

echo "Aseity Services Status"
echo "======================"
echo ""

# Check Docker
if ! command -v docker &> /dev/null; then
    print_status "Docker" "error" "Not installed"
else
    if docker info &> /dev/null; then
        print_status "Docker" "running" ""
    else
        print_status "Docker" "stopped" "Not running"
    fi
fi

echo ""

# Check Crawl4AI
if docker ps --format '{{.Names}}' | grep -q "^crawl4ai$"; then
    health=$(docker inspect crawl4ai --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
    uptime=$(docker inspect crawl4ai --format='{{.State.StartedAt}}' 2>/dev/null | xargs -I {} date -j -f "%Y-%m-%dT%H:%M:%S" {} "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "unknown")
    
    if [ "$health" = "healthy" ]; then
        # Get additional stats
        if curl -sf http://localhost:11235/health &> /dev/null; then
            print_status "Crawl4AI" "running" "(healthy, started: $uptime)"
            
            # Try to get browser pool info
            echo "  Endpoint: http://localhost:11235"
            echo "  Dashboard: http://localhost:11235/dashboard"
        else
            print_status "Crawl4AI" "running" "(unhealthy)"
        fi
    else
        print_status "Crawl4AI" "running" "(health: $health)"
    fi
elif docker ps -a --format '{{.Names}}' | grep -q "^crawl4ai$"; then
    print_status "Crawl4AI" "stopped" "(container exists but not running)"
    echo "  Start with: docker start crawl4ai"
else
    print_status "Crawl4AI" "stopped" "(not installed)"
    echo "  Install with: ./scripts/setup-crawl4ai.sh"
fi

echo ""

# Check Ollama
if docker ps --format '{{.Names}}' | grep -q "^ollama$"; then
    print_status "Ollama" "running" ""
elif docker ps -a --format '{{.Names}}' | grep -q "^ollama$"; then
    print_status "Ollama" "stopped" ""
else
    print_status "Ollama" "stopped" "(not running)"
fi

echo ""

# Check Aseity
if docker ps --format '{{.Names}}' | grep -q "^aseity$"; then
    print_status "Aseity" "running" ""
elif docker ps -a --format '{{.Names}}' | grep -q "^aseity$"; then
    print_status "Aseity" "stopped" ""
else
    print_status "Aseity" "stopped" "(not running)"
fi

echo ""
echo "Quick Commands:"
echo "  Start all:     docker compose up -d"
echo "  Start Crawl4AI: docker compose --profile crawl4ai up -d crawl4ai"
echo "  Stop all:      docker compose down"
echo "  View logs:     docker logs <service-name>"
echo ""
