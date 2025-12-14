# Multi-stage Dockerfile for Stats Agent Team
# Builds all agents and runs them with Eino orchestration

# Stage 1: Build all agents
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all agents
RUN go build -o /build/bin/research agents/research/main.go && \
    go build -o /build/bin/synthesis agents/synthesis/main.go && \
    go build -o /build/bin/verification agents/verification/main.go && \
    go build -o /build/bin/orchestration-eino agents/orchestration-eino/main.go

# Stage 2: Create runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create app directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/research /app/research
COPY --from=builder /build/bin/synthesis /app/synthesis
COPY --from=builder /build/bin/verification /app/verification
COPY --from=builder /build/bin/orchestration-eino /app/orchestration-eino

# Copy startup script
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Expose ports
# Research agent: 8001
# Verification agent: 8002
# Eino orchestration agent: 8003
# Synthesis agent: 8004
EXPOSE 8001 8002 8003 8004

# Health check for orchestration agent
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8003/health || exit 1

# Run all agents
ENTRYPOINT ["/app/docker-entrypoint.sh"]
