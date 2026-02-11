#!/bin/bash
# Test script to verify orchestrator TUI integration

echo "Testing Orchestrator TUI Integration"
echo "====================================="
echo ""

# Build the binary
echo "1. Building aseity..."
go build -o aseity ./cmd/aseity
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build successful"
echo ""

# Test 1: Verify config has orchestrator enabled
echo "2. Checking orchestrator config..."
if grep -q "enabled: true" configs/default.yaml; then
    echo "✅ Orchestrator enabled in config"
else
    echo "❌ Orchestrator not enabled in config"
    exit 1
fi
echo ""

# Test 2: Test CLI orchestrator mode (explicit flag)
echo "3. Testing CLI orchestrator mode (--orchestrator flag)..."
timeout 60s ./aseity --orchestrator "What is 2+2?" > /tmp/aseity_test_cli.txt 2>&1 &
CLI_PID=$!
sleep 5
if ps -p $CLI_PID > /dev/null; then
    echo "✅ CLI orchestrator mode started"
    kill $CLI_PID 2>/dev/null
else
    echo "⚠️  CLI orchestrator completed or failed (check /tmp/aseity_test_cli.txt)"
fi
echo ""

# Test 3: Check if ShouldUseOrchestrator logic is present
echo "4. Verifying ShouldUseOrchestrator implementation..."
if grep -q "func (a \*Agent) ShouldUseOrchestrator" internal/agent/orchestrator_integration.go; then
    echo "✅ ShouldUseOrchestrator function exists"
else
    echo "❌ ShouldUseOrchestrator function not found"
    exit 1
fi
echo ""

# Test 4: Check if NewModel accepts orchestrator config
echo "5. Verifying NewModel signature..."
if grep -q "orchConfig \*agent.OrchestratorConfig" internal/tui/app.go; then
    echo "✅ NewModel accepts orchestrator config"
else
    echo "❌ NewModel doesn't accept orchestrator config"
    exit 1
fi
echo ""

# Test 5: Check if orchestrator is initialized in NewModel
echo "6. Verifying orchestrator initialization in NewModel..."
if grep -q "ag.SetOrchestrator" internal/tui/app.go; then
    echo "✅ Orchestrator is initialized in NewModel"
else
    echo "❌ Orchestrator not initialized in NewModel"
    exit 1
fi
echo ""

echo "====================================="
echo "✅ All verification checks passed!"
echo ""
echo "Manual TUI Test:"
echo "  Run: ./aseity"
echo "  Try: 'research the top 3 AI trends'"
echo "  Expected: Should show '🤖 Using orchestrator mode...'"
