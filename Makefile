.PHONY: help build build-mcp docker-build docker-up docker-down docker-logs run-research run-synthesis run-verification run-orchestration run-orchestration-eino run-all run-all-eino run-mcp clean install test

help:
	@echo "Statistics Agent - Make targets"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-build            Build Docker image"
	@echo "  make docker-up               Start all agents with Docker Compose"
	@echo "  make docker-down             Stop all agents"
	@echo "  make docker-logs             View Docker logs"
	@echo ""
	@echo "Build Commands:"
	@echo "  make install                 Install dependencies"
	@echo "  make build                   Build all agents"
	@echo "  make build-mcp               Build MCP server"
	@echo ""
	@echo "Run Commands (Local):"
	@echo "  make run-research            Run research agent"
	@echo "  make run-synthesis           Run synthesis agent"
	@echo "  make run-verification        Run verification agent"
	@echo "  make run-orchestration       Run trpc-agent orchestration"
	@echo "  make run-orchestration-eino  Run Eino orchestration"
	@echo "  make run-all                 Run all agents with trpc-agent orchestrator"
	@echo "  make run-all-eino            Run all agents with Eino orchestrator"
	@echo "  make run-mcp                 Run MCP server (requires agents running)"
	@echo ""
	@echo "Other Commands:"
	@echo "  make test                    Run tests"
	@echo "  make clean                   Clean build artifacts"

install:
	go mod download
	go get github.com/trpc-group/trpc-agent-go
	go get github.com/trpc-group/trpc-a2a-go
	go get github.com/cloudwego/eino

build:
	@echo "Building agents..."
	go build -o bin/research agents/research/main.go
	go build -o bin/synthesis agents/synthesis/main.go
	go build -o bin/verification agents/verification/main.go
	go build -o bin/orchestration agents/orchestration/main.go
	go build -o bin/orchestration-eino agents/orchestration-eino/main.go
	go build -o bin/stats-agent main.go
	@echo "Build complete!"

build-mcp:
	@echo "Building MCP server..."
	go build -o bin/mcp-server mcp/server/main.go
	@echo "MCP server build complete!"

docker-build:
	@echo "Building Docker image..."
	docker build -t stats-agent-team:latest .
	@echo "Docker build complete!"

docker-up:
	@echo "Starting all agents with Docker Compose..."
	docker-compose up -d
	@echo "All agents started! Use 'make docker-logs' to view logs"

docker-down:
	@echo "Stopping all agents..."
	docker-compose down
	@echo "All agents stopped"

docker-logs:
	docker-compose logs -f

run-research:
	@echo "Starting Research Agent on :8001 (HTTP) and :9001 (A2A)..."
	go run agents/research/main.go

run-synthesis:
	@echo "Starting Synthesis Agent on :8004..."
	go run agents/synthesis/main.go

run-verification:
	@echo "Starting Verification Agent on :8002 (HTTP) and :9002 (A2A)..."
	go run agents/verification/main.go

run-orchestration:
	@echo "Starting Orchestration Agent (trpc-agent) on :8000 (HTTP) and :9000 (A2A)..."
	go run agents/orchestration/main.go

run-orchestration-eino:
	@echo "Starting Orchestration Agent (Eino) on :8003 (HTTP) and :9003 (A2A)..."
	go run agents/orchestration-eino/main.go

run-all:
	@echo "Starting all agents with trpc-agent orchestrator..."
	@echo "Research Agent: http://localhost:8001 (A2A: 9001)"
	@echo "Synthesis Agent: http://localhost:8004"
	@echo "Verification Agent: http://localhost:8002 (A2A: 9002)"
	@echo "Orchestration Agent (trpc-agent): http://localhost:8000 (A2A: 9000)"
	@go run agents/research/main.go & \
	go run agents/synthesis/main.go & \
	go run agents/verification/main.go & \
	go run agents/orchestration/main.go & \
	wait

run-all-eino:
	@echo "Starting all agents with Eino orchestrator..."
	@echo "Research Agent: http://localhost:8001 (A2A: 9001)"
	@echo "Synthesis Agent: http://localhost:8004"
	@echo "Verification Agent: http://localhost:8002 (A2A: 9002)"
	@echo "Orchestration Agent (Eino): http://localhost:8003 (A2A: 9003)"
	@go run agents/research/main.go & \
	go run agents/synthesis/main.go & \
	go run agents/verification/main.go & \
	go run agents/orchestration-eino/main.go & \
	wait

run-mcp:
	@echo "Starting MCP server (stdio)..."
	@echo "Note: Ensure research and verification agents are running first!"
	@echo "  Terminal 1: make run-research"
	@echo "  Terminal 2: make run-verification"
	@go run mcp/server/main.go

clean:
	rm -rf bin/
	go clean

test:
	go test ./...
