# Docker Deployment Guide

This guide explains how to run the Statistics Agent Team using Docker.

> **Note:** For local development without Docker, see the [main README](README.md).

## Overview

The Docker setup runs all **four agents** in a single container:
- **Research Agent** (port 8001): Web search for source URLs
- **Synthesis Agent** (port 8004): Extract statistics from URLs using LLM ⭐ NEW
- **Verification Agent** (port 8002): Verifies statistics from sources
- **Eino Orchestration Agent** (port 8003): Coordinates the 4-agent workflow

## Quick Start

### Using Docker Compose (Recommended)

1. **Set up environment variables**

Create a `.env` file in the project root:

```bash
# LLM Provider (gemini, claude, openai, ollama)
LLM_PROVIDER=gemini

# LLM API Keys (provide at least one)
GEMINI_API_KEY=your_gemini_api_key_here
CLAUDE_API_KEY=your_claude_api_key_here
OPENAI_API_KEY=your_openai_api_key_here

# Search API Keys (required for real web search)
SEARCH_PROVIDER=serper
SERPER_API_KEY=your_serper_api_key_here
# OR use SerpAPI
# SEARCH_PROVIDER=serpapi
# SERPAPI_API_KEY=your_serpapi_key_here

# For Ollama (local models)
OLLAMA_URL=http://host.docker.internal:11434
OLLAMA_MODEL=llama2

# Optional: Override default models
# GEMINI_MODEL=gemini-2.0-flash-exp
# CLAUDE_MODEL=claude-3-5-sonnet-20241022
# OPENAI_MODEL=gpt-4
```

2. **Start the agents**

```bash
docker-compose up -d
```

3. **Check status**

```bash
docker-compose ps
docker-compose logs -f
```

4. **Test the orchestration endpoint**

```bash
curl -X POST http://localhost:8003/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change",
    "min_verified_stats": 5,
    "max_candidates": 20,
    "reputable_only": true
  }'
```

5. **Stop the agents**

```bash
docker-compose down
```

### Using Docker Directly

1. **Build the image**

```bash
docker build -t stats-agent-team .
```

2. **Run the container**

```bash
docker run -d \
  --name stats-agent-eino \
  -p 8001:8001 \
  -p 8002:8002 \
  -p 8003:8003 \
  -p 8004:8004 \
  -e LLM_PROVIDER=gemini \
  -e GEMINI_API_KEY=your_api_key_here \
  -e SEARCH_PROVIDER=serper \
  -e SERPER_API_KEY=your_serper_key_here \
  stats-agent-team
```

3. **View logs**

```bash
docker logs -f stats-agent-eino
```

4. **Stop the container**

```bash
docker stop stats-agent-eino
docker rm stats-agent-eino
```

## Service Endpoints

Once running, the following endpoints are available:

### Research Agent (Port 8001)
- `POST http://localhost:8001/research` - Web search for source URLs
- `GET http://localhost:8001/health` - Health check

### Synthesis Agent (Port 8004) ⭐ NEW
- `POST http://localhost:8004/synthesize` - Extract statistics from URLs
- `GET http://localhost:8004/health` - Health check

### Verification Agent (Port 8002)
- `POST http://localhost:8002/verify` - Verify statistics
- `GET http://localhost:8002/health` - Health check

### Orchestration Agent (Port 8003)
- `POST http://localhost:8003/orchestrate` - Full 4-agent workflow
- `GET http://localhost:8003/health` - Health check

**Note:** While all agents are exposed for testing and troubleshooting, typical usage only requires calling the orchestration endpoint.

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LLM_PROVIDER` | LLM provider to use | `gemini` | Yes |
| `GEMINI_API_KEY` | Google Gemini API key | - | If using Gemini |
| `CLAUDE_API_KEY` | Anthropic Claude API key | - | If using Claude |
| `OPENAI_API_KEY` | OpenAI API key | - | If using OpenAI |
| `OLLAMA_URL` | Ollama server URL | `http://host.docker.internal:11434` | If using Ollama |
| `GEMINI_MODEL` | Gemini model name | `gemini-2.0-flash-exp` | No |
| `CLAUDE_MODEL` | Claude model name | `claude-3-5-sonnet-20241022` | No |
| `OPENAI_MODEL` | OpenAI model name | `gpt-4` | No |
| `OLLAMA_MODEL` | Ollama model name | `llama2` | No |
| `SEARCH_PROVIDER` | Search provider | `serper` | No |
| `SERPER_API_KEY` | Serper API key | - | For real search |
| `SERPAPI_API_KEY` | SerpAPI key | - | Alternative search |

### LLM Provider Options

**Gemini (Default)**
```bash
LLM_PROVIDER=gemini
GEMINI_API_KEY=your_key_here
```

**Claude**
```bash
LLM_PROVIDER=claude
CLAUDE_API_KEY=your_key_here
```

**OpenAI**
```bash
LLM_PROVIDER=openai
OPENAI_API_KEY=your_key_here
```

**Ollama (Local)**
```bash
LLM_PROVIDER=ollama
OLLAMA_URL=http://host.docker.internal:11434
OLLAMA_MODEL=llama2
```

## API Examples

### Orchestrate Full Workflow

```bash
curl -X POST http://localhost:8003/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "renewable energy adoption",
    "min_verified_stats": 10,
    "max_candidates": 30,
    "reputable_only": true
  }'
```

Response:
```json
{
  "topic": "renewable energy adoption",
  "statistics": [
    {
      "name": "Global renewable capacity growth",
      "value": 83,
      "unit": "%",
      "source": "International Energy Agency",
      "source_url": "https://www.iea.org/...",
      "excerpt": "Renewable capacity grew by 83% in 2023...",
      "verified": true,
      "date_found": "2025-12-13T10:30:00Z"
    }
  ],
  "total_candidates": 25,
  "verified_count": 12,
  "failed_count": 13,
  "timestamp": "2025-12-13T10:30:15Z"
}
```

### Research Only

```bash
curl -X POST http://localhost:8001/research \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "electric vehicles",
    "min_statistics": 5,
    "max_statistics": 10,
    "reputable_only": true
  }'
```

### Verification Only

```bash
curl -X POST http://localhost:8002/verify \
  -H "Content-Type: application/json" \
  -d '{
    "candidates": [
      {
        "name": "EV market share",
        "value": 18,
        "unit": "%",
        "source": "Bloomberg NEF",
        "source_url": "https://about.bnef.com/...",
        "excerpt": "Electric vehicles reached 18% market share..."
      }
    ]
  }'
```

## Troubleshooting

### Container won't start

Check logs:
```bash
docker-compose logs -f stats-agent-eino
```

Common issues:
- Missing API keys: Ensure you've set the appropriate `*_API_KEY` variable
- Port conflicts: Another service using ports 8001-8003
- Invalid LLM provider: Use `gemini`, `claude`, `openai`, or `ollama`

### Health check failing

Test individual agents:
```bash
curl http://localhost:8001/health  # Research
curl http://localhost:8002/health  # Verification
curl http://localhost:8003/health  # Orchestration
```

### Ollama connection issues

If using Ollama on the host:
- Use `http://host.docker.internal:11434` (Mac/Windows)
- Use `http://172.17.0.1:11434` (Linux)
- Ensure Ollama is running: `ollama serve`

### View agent communication

Enable debug logging:
```bash
docker-compose logs -f stats-agent-eino | grep "Orchestration\|Research\|Verification"
```

## Building for Production

### Multi-platform builds

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t stats-agent-team:latest \
  .
```

### Optimize image size

The Dockerfile uses multi-stage builds and Alpine Linux for minimal size:
- Builder stage: ~1.2GB (includes Go toolchain)
- Runtime stage: ~50MB (only binaries + Alpine)

### Security scanning

```bash
docker scan stats-agent-team:latest
```

## Performance Tuning

### Resource limits (docker-compose.yml)

```yaml
services:
  stats-agent-eino:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '1.0'
          memory: 1G
```

### Timeout configuration

Agents have built-in HTTP timeouts:
- Research: 30s read/write, 60s idle
- Verification: 45s read/write, 90s idle
- Orchestration: 60s read/write, 120s idle

## Integration with Other Services

### Behind a reverse proxy (nginx)

```nginx
upstream stats_orchestration {
    server localhost:8003;
}

server {
    listen 80;
    server_name stats.example.com;

    location / {
        proxy_pass http://stats_orchestration;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Kubernetes deployment

See [KUBERNETES.md](KUBERNETES.md) for Kubernetes manifests (if available).

## Monitoring

### Health checks

Docker Compose includes automatic health checks every 30 seconds.

### Metrics

To add Prometheus metrics, mount a volume for logs:
```yaml
volumes:
  - ./logs:/var/log/stats-agent
```

## References

- [Search Integration Guide](SEARCH_INTEGRATION.md) - Setup web search for real statistics
- [LLM Configuration Guide](LLM_CONFIGURATION.md) - Configure LLM providers
- [MCP Server Guide](MCP_SERVER.md) - Claude Code integration
- [Main README](README.md) - Project overview and local setup
