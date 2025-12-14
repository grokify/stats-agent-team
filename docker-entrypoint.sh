#!/bin/sh
set -e

echo "Starting Statistics Agent Team with Eino Orchestration"
echo "========================================================"
echo ""
echo "Research Agent:        http://localhost:8001"
echo "Verification Agent:    http://localhost:8002"
echo "Orchestration (Eino):  http://localhost:8003"
echo ""

# Start research agent in background
echo "Starting Research Agent on :8001..."
/app/research &
RESEARCH_PID=$!

# Start verification agent in background
echo "Starting Verification Agent on :8002..."
/app/verification &
VERIFICATION_PID=$!

# Wait a bit for agents to initialize
sleep 2

# Start orchestration agent in foreground
echo "Starting Eino Orchestration Agent on :8003..."
/app/orchestration-eino &
ORCHESTRATION_PID=$!

echo ""
echo "All agents started successfully!"
echo "========================================================"
echo ""

# Function to handle shutdown
shutdown() {
    echo ""
    echo "Shutting down agents..."
    kill -TERM $RESEARCH_PID $VERIFICATION_PID $ORCHESTRATION_PID 2>/dev/null || true
    wait $RESEARCH_PID $VERIFICATION_PID $ORCHESTRATION_PID 2>/dev/null || true
    echo "All agents stopped."
    exit 0
}

# Trap signals
trap shutdown SIGTERM SIGINT

# Wait for all background processes
wait $RESEARCH_PID $VERIFICATION_PID $ORCHESTRATION_PID
