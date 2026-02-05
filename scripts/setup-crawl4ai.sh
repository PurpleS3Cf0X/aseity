#!/bin/bash
# Setup script for Crawl4AI integration
# This script checks dependencies and starts the Crawl4AI service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Crawl4AI Setup for Aseity"
echo "=============================="
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC}  $1"
}

print_info() {
    echo -e "ℹ  $1"
}

# Check if Docker is installed
echo "Checking dependencies..."
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    echo ""
    echo "Please install Docker Desktop:"
    echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
    echo "  Linux: https://docs.docker.com/engine/install/"
    echo ""
    print_warning "Aseity will continue to work with basic web_fetch"
    exit 1
fi
print_success "Docker installed"

# Check if Docker is running
if ! docker info &> /dev/null; then
    print_error "Docker is not running"
    echo ""
    echo "Please start Docker Desktop and try again"
    exit 1
fi
print_success "Docker running"

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    print_error "docker-compose is not available"
    echo ""
    echo "Please install docker-compose or update Docker Desktop"
    exit 1
fi
print_success "docker-compose available"

echo ""
echo "Starting Crawl4AI service..."
echo ""

# Pull the latest image
print_info "Pulling Crawl4AI image (this may take a few minutes)..."
if docker pull unclecode/crawl4ai:latest; then
    print_success "Image pulled successfully"
else
    print_error "Failed to pull image"
    exit 1
fi

# Start the service
print_info "Starting Crawl4AI container..."
if docker compose --profile crawl4ai up -d crawl4ai; then
    print_success "Container started"
else
    print_error "Failed to start container"
    exit 1
fi

# Wait for health check
print_info "Waiting for Crawl4AI to become healthy..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if docker inspect crawl4ai --format='{{.State.Health.Status}}' 2>/dev/null | grep -q "healthy"; then
        print_success "Crawl4AI is healthy!"
        break
    fi
    
    attempt=$((attempt + 1))
    if [ $attempt -eq $max_attempts ]; then
        print_error "Crawl4AI failed to become healthy"
        echo ""
        echo "Check logs with: docker logs crawl4ai"
        exit 1
    fi
    
    echo -n "."
    sleep 2
done

echo ""
echo ""

# Test the service
print_info "Testing Crawl4AI..."
if curl -f http://localhost:11235/health &> /dev/null; then
    print_success "Health check passed"
else
    print_warning "Health check failed, but container is running"
fi

# Run a test crawl
print_info "Running test crawl..."
test_response=$(curl -s -X POST http://localhost:11235/crawl \
    -H "Content-Type: application/json" \
    -d '{"urls": ["https://example.com"], "priority": 10}' 2>/dev/null || echo "failed")

if echo "$test_response" | grep -q "markdown\|results"; then
    print_success "Test crawl successful"
else
    print_warning "Test crawl failed, but service is running"
fi

echo ""
echo "=============================="
print_success "Crawl4AI setup complete!"
echo ""
echo "Service Information:"
echo "  Endpoint: http://localhost:11235"
echo "  Dashboard: http://localhost:11235/dashboard"
echo "  Playground: http://localhost:11235/playground"
echo ""
echo "Useful Commands:"
echo "  Check status:  ./scripts/check-services.sh"
echo "  View logs:     docker logs crawl4ai"
echo "  Restart:       docker restart crawl4ai"
echo "  Stop:          docker stop crawl4ai"
echo ""
echo "In Aseity, use the web_crawl tool for advanced scraping!"
echo ""
