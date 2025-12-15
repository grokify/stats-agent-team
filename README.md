# Statistics Agent Team

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

A multi-agent system for finding and verifying statistics from reputable web sources using Go, built with [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) and [Eino](https://github.com/cloudwego/eino).

## Overview

This project implements a sophisticated multi-agent architecture that leverages LLMs and web search to find verifiable statistics, prioritizing well-known and respected publishers. The system ensures accuracy through automated verification of sources.

## Architecture

The system implements a **4-agent architecture** with clear separation of concerns:

> **Architecture**: Built with **Google ADK** for LLM-based operations. **Two orchestration options** available: ADK-based (LLM-driven decisions) and Eino-based (deterministic graph workflow). See [4_AGENT_ARCHITECTURE.md](4_AGENT_ARCHITECTURE.md) for complete details.

```
┌─────────────────────────────────────────────────────────┐
│                   User Request                          │
│              "Find climate change statistics"           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            ORCHESTRATION AGENT                          │
│         (Port 8000 ADK / 8003 Eino)                     │
│  • Coordinates 4-agent workflow                         │
│  • Manages retry logic                                  │
│  • Ensures quality standards                            │
└───┬──────────────┬──────────────┬───────────────────────┘
    │              │              │
    ▼              ▼              ▼
┌────────────┐ ┌──────────┐ ┌─────────────────┐
│  RESEARCH  │ │SYNTHESIS │ │  VERIFICATION   │
│   AGENT    │ │  AGENT   │ │     AGENT       │
│ Port 8001  │ │Port 8004 │ │   Port 8002     │
│            │ │          │ │                 │
│ • Search   │─│• Fetch   │─│• Re-fetch URLs  │
│   Serper   │ │  URLs    │ │• Validate text  │
│ • Filter   │ │• LLM     │ │• Check numbers  │
│   Sources  │ │  Extract │ │• Flag errors    │
└────────────┘ └──────────┘ └─────────────────┘
     │              │              │
     ▼              ▼              ▼
  URLs only    Statistics     Verified Stats
```

### Agent Responsibilities

#### 1. Research Agent (`agents/research/`) - Web Search Only
- **No LLM required** - Pure search functionality
- Web search via Serper/SerpAPI integration
- Returns URLs with metadata (title, snippet, domain)
- Prioritizes reputable sources (`.gov`, `.edu`, research orgs)
- Output: List of `SearchResult` objects
- Port: **8001**

#### 2. Synthesis Agent (`agents/synthesis/`) - Google ADK ⭐ NEW
- **LLM-heavy** extraction agent
- Built with Google ADK and LLM (Gemini/Claude/OpenAI/Ollama)
- Fetches webpage content from URLs
- Extracts numerical statistics using LLM analysis
- Finds verbatim excerpts containing statistics
- Creates `CandidateStatistic` objects with proper metadata
- Port: **8004**

#### 3. Verification Agent (`agents/verification/`) - Google ADK
- **LLM-light** validation agent
- Re-fetches source URLs to verify content
- Checks excerpts exist verbatim in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with pass/fail reasons
- Port: **8002**

#### 4a. Orchestration Agent - Google ADK (`agents/orchestration/`)
- Built with Google ADK for LLM-driven workflow decisions
- Coordinates: Research → Synthesis → Verification
- Implements adaptive retry logic
- Dynamic quality control
- Port: **8000**

#### 4b. Orchestration Agent - Eino (`agents/orchestration-eino/`) ⭐ RECOMMENDED
- **Deterministic graph-based workflow** (no LLM for orchestration)
- Type-safe orchestration with Eino framework
- Predictable, reproducible behavior
- Faster and lower cost
- Workflow: ValidateInput → Research → Synthesis → Verification → QualityCheck → Format
- Port: **8003**
- **Recommended for production use**

## Features

- ✅ **Two search modes**: Direct LLM (fast, like ChatGPT) and Multi-agent verification pipeline ⭐ NEW
- ✅ **Direct LLM search** - Single LLM call with URLs, no agents needed 🚀 NEW
- ✅ **Multi-agent orchestration** with chains and workflow coordination
- ✅ **Google ADK integration** for LLM-based agents
- ✅ **Eino framework** for deterministic graph orchestration
- ✅ **Human-in-the-loop retry** - Prompts user when partial results found ⭐ NEW
- ✅ **Multi-LLM providers** (Gemini, Claude, OpenAI, Ollama, xAI Grok) via unified interface ⭐
- ✅ **MCP Server** for integration with Claude Code and other MCP clients
- ✅ **Docker deployment** for easy containerized setup 🐳
- ✅ **Real web search** via Serper/SerpAPI for finding actual statistics 🔍
- ✅ **Source verification** to prevent hallucinations
- ✅ **Reputable source prioritization** (government, academic, research orgs)
- ✅ **Structured JSON output** with complete metadata
- ✅ **HTTP APIs** for all agents
- ✅ **Function tools** for structured agent capabilities

## Output Format

The system returns verified statistics in JSON format:

```json
[
  {
    "name": "Global temperature increase since pre-industrial times",
    "value": 1.1,
    "unit": "°C",
    "source": "IPCC Sixth Assessment Report",
    "source_url": "https://www.ipcc.ch/...",
    "excerpt": "Global surface temperature has increased by approximately 1.1°C since pre-industrial times...",
    "verified": true,
    "date_found": "2025-12-13T10:30:00Z"
  }
]
```

### Field Descriptions

- **name**: Description of the statistic
- **value**: Numerical value (float32)
- **unit**: Unit of measurement (e.g., "°C", "%", "million", "billion")
- **source**: Name of source organization/publication
- **source_url**: URL to the original source
- **excerpt**: Verbatim quote containing the statistic
- **verified**: Whether the verification agent confirmed it
- **date_found**: Timestamp when statistic was found

## Installation

### Prerequisites

- Go 1.21 or higher
- LLM API key for your chosen provider:
  - **Gemini** (default): Google API key (set as `GOOGLE_API_KEY` or `GEMINI_API_KEY`)
  - **Claude**: Anthropic API key (set as `ANTHROPIC_API_KEY` or `CLAUDE_API_KEY`)
  - **OpenAI**: OpenAI API key (set as `OPENAI_API_KEY`)
  - **Ollama**: Local Ollama installation (default: `http://localhost:11434`)
- Optional: API keys for search provider (Google Search, etc.)

### Setup

1. Clone the repository:
```bash
git clone https://github.com/grokify/stats-agent.git
cd stats-agent
```

2. Install dependencies:
```bash
make install
# or
go mod download
```

3. Configure environment variables:
```bash
# For Gemini (default)
export GOOGLE_API_KEY="your-google-api-key"

# For Claude
export LLM_PROVIDER="claude"
export ANTHROPIC_API_KEY="your-anthropic-api-key"

# For OpenAI
export LLM_PROVIDER="openai"
export OPENAI_API_KEY="your-openai-api-key"

# For Ollama (local)
export LLM_PROVIDER="ollama"
export OLLAMA_URL="http://localhost:11434"
export LLM_MODEL="llama3.2"

# Optional: Create .env file
cp .env.example .env
# Edit .env with your API keys
```

4. Build the agents:
```bash
make build
```

## Usage

You can run the system either with Docker (containerized) or locally. Choose the method that best fits your needs.

| Method | Best For | Command |
|--------|----------|---------|
| **Docker** 🐳 | Production, quick start, isolated environment | `docker-compose up -d` |
| **Local** 💻 | Development, debugging, customization | `make run-all-eino` |

### Quick Start with Docker 🐳

The fastest way to get started:

```bash
# Start all agents with Docker Compose
docker-compose up -d

# Test the orchestration endpoint
curl -X POST http://localhost:8003/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"topic": "climate change", "min_verified_stats": 5}'

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

See [DOCKER.md](DOCKER.md) for complete Docker deployment guide.

---

### Local Development Setup

#### Running the Agents Locally

##### Option 1: Run all agents with Eino orchestrator (Recommended)
```bash
make run-all-eino
```

##### Option 2: Run all agents with ADK orchestrator
```bash
make run-all
```

##### Option 3: Run each agent separately (in different terminals)
```bash
# Terminal 1: Research Agent (ADK)
make run-research

# Terminal 2: Verification Agent (ADK)
make run-verification

# Terminal 3: Orchestration Agent (choose one)
make run-orchestration       # ADK version (LLM-based)
make run-orchestration-eino  # Eino version (deterministic, recommended)
```

#### Using the CLI

The CLI supports two modes: **Direct LLM search** (fast, like ChatGPT) and **Multi-agent verification pipeline** (thorough, verified).

##### Direct Mode (Fast, Recommended for Quick Results)

Direct mode uses a single LLM call to find statistics with sources - similar to ChatGPT:

```bash
# Fast search using direct LLM (no agents needed)
./bin/stats-agent search "climate change" --direct

# Request specific number of statistics
./bin/stats-agent search "AI adoption" --direct --min-stats 15

# JSON output only
./bin/stats-agent search "renewable energy" --direct --output json

# With specific LLM provider
LLM_PROVIDER=openai OPENAI_API_KEY=your_key \
  ./bin/stats-agent search "cybersecurity 2024" --direct --min-stats 20
```

**Advantages of Direct Mode:**
- ⚡ **Fast** - Single LLM call, no multi-agent pipeline
- 🔗 **URLs included** - Returns source URLs like ChatGPT
- 🚀 **No agent servers needed** - Works standalone
- 💰 **Lower cost** - Single LLM call instead of multiple

##### Multi-Agent Pipeline Mode (Thorough Verification)

For verified, web-scraped statistics (requires agents running):

```bash
# Start agents first
make run-all-eino

# Then in another terminal:
# Basic search with verification pipeline
./bin/stats-agent search "climate change"

# Request specific number of verified statistics
./bin/stats-agent search "global warming" --min-stats 15

# Increase candidate search space
./bin/stats-agent search "AI trends" --min-stats 10 --max-candidates 100

# Only reputable sources
./bin/stats-agent search "COVID-19 statistics" --reputable-only

# JSON output only
./bin/stats-agent search "renewable energy" --output json

# Text output only
./bin/stats-agent search "climate data" --output text
```

**Advantages of Multi-Agent Mode:**
- ✅ **Verified sources** - Actually fetches and checks web pages
- 🔍 **Web search** - Finds current statistics from the web
- 🎯 **Accuracy** - Validates excerpts and values match
- 🔄 **Human-in-the-loop** - Prompts to continue if target not met

##### CLI Options

```bash
stats-agent search <topic> [options]

Options:
  -d, --direct              Use direct LLM search (fast, like ChatGPT)
  -m, --min-stats <n>       Minimum statistics to find (default: 10)
  -c, --max-candidates <n>  Max candidates for pipeline mode (default: 50)
  -r, --reputable-only      Only use reputable sources
  -o, --output <format>     Output format: json, text, both (default: both)
      --orchestrator-url    Override orchestrator URL
  -v, --verbose             Show verbose debug information
      --version             Show version information
```

---

### Using with Claude Code (MCP Server)

The system can be used as an MCP server with Claude Code and other MCP clients:

```bash
# Build the MCP server
make build-mcp

# Configure in Claude Code's MCP settings (see MCP_SERVER.md)
```

See [MCP_SERVER.md](MCP_SERVER.md) for detailed setup instructions.

### API Usage

You can also call the agents directly via HTTP (works with both Docker and local deployment):

```bash
# Call Eino orchestration agent (recommended - deterministic)
curl -X POST http://localhost:8003/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change",
    "min_verified_stats": 10,
    "max_candidates": 30,
    "reputable_only": true
  }'

# Or call ADK orchestration agent (LLM-based)
curl -X POST http://localhost:8000/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change",
    "min_verified_stats": 10,
    "max_candidates": 30,
    "reputable_only": true
  }'
```

## Configuration

### Environment Variables

#### LLM Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROVIDER` | LLM provider: `gemini`, `claude`, `openai`, `ollama` | `gemini` |
| `LLM_MODEL` | Model name (provider-specific) | See defaults below |
| `LLM_API_KEY` | Generic API key (overrides provider-specific) | - |
| `LLM_BASE_URL` | Base URL for custom endpoints (Ollama, etc.) | - |

**Provider-Specific API Keys:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GOOGLE_API_KEY` / `GEMINI_API_KEY` | Google API key for Gemini | **Required for Gemini** |
| `ANTHROPIC_API_KEY` / `CLAUDE_API_KEY` | Anthropic API key for Claude | **Required for Claude** |
| `OPENAI_API_KEY` | OpenAI API key | **Required for OpenAI** |
| `OLLAMA_URL` | Ollama server URL | `http://localhost:11434` |

**Default Models by Provider:**
- Gemini: `gemini-2.0-flash-exp`
- Claude: `claude-3-5-sonnet-20241022`
- OpenAI: `gpt-4`
- Ollama: `llama3.2`

See [LLM_CONFIGURATION.md](LLM_CONFIGURATION.md) for detailed LLM setup.

#### Search Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARCH_PROVIDER` | Search provider: `serper`, `serpapi` | `serper` |
| `SERPER_API_KEY` | Serper API key (get from serper.dev) | Required for real search |
| `SERPAPI_API_KEY` | SerpAPI key (alternative provider) | Required for SerpAPI |

**Note:** Without a search API key, the research agent will use mock data. See [SEARCH_INTEGRATION.md](SEARCH_INTEGRATION.md) for setup details.

#### Other Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `RESEARCH_AGENT_URL` | Research agent URL | `http://localhost:8001` |
| `VERIFICATION_AGENT_URL` | Verification agent URL | `http://localhost:8002` |
| `ORCHESTRATOR_URL` | ADK orchestrator URL | `http://localhost:8000` |
| `ORCHESTRATOR_EINO_URL` | Eino orchestrator URL | `http://localhost:8003` |

### Port Configuration

| Agent | HTTP Port | Description |
|-------|-----------|-------------|
| Research (ADK) | 8001 | Gemini 2.0 Flash-based research |
| Verification (ADK) | 8002 | Gemini 2.0 Flash-based verification |
| Orchestration (ADK) | 8000 | LLM-based orchestration |
| **Orchestration (Eino)** ⭐ | **8003** | **Deterministic graph orchestration (recommended)** |

## Project Structure

```
stats-agent/
├── agents/
│   ├── orchestration/      # Orchestration agent (Google ADK)
│   │   └── main.go
│   ├── orchestration-eino/ # Orchestration agent (Eino) ⭐
│   │   └── main.go
│   ├── research/           # Research agent (Google ADK)
│   │   └── main.go
│   └── verification/       # Verification agent (Google ADK)
│       └── main.go
├── pkg/
│   ├── config/            # Configuration management
│   │   └── config.go
│   └── models/            # Shared data models
│       └── statistic.go
├── main.go                # CLI entry point
├── Makefile              # Build and run commands
├── go.mod                # Go dependencies
├── .env.example          # Environment template
└── README.md             # This file
```

## Development

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Cleaning Build Artifacts

```bash
make clean
```

## Technology Stack

- **Language**: Go 1.21+
- **Agent Frameworks**:
  - [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) - LLM-based agents ⭐
  - [Eino](https://github.com/cloudwego/eino) - Deterministic graph orchestration ⭐
- **LLM Providers** (configurable):
  - Google Gemini (default) - Gemini 2.0 Flash ⭐
  - Anthropic Claude - Claude 3.5 Sonnet
  - OpenAI - GPT-4
  - Ollama - Local models (llama3.2, etc.)
- **Search**: Configurable (Google Search, etc.)

## How It Works

1. **User Request**: User provides a topic via CLI or API
2. **Orchestration**: Orchestrator receives request and initiates workflow
3. **Research Phase**: Research agent searches web for candidate statistics
4. **Verification Phase**: Verification agent validates each candidate
5. **Quality Control**: Orchestrator checks if minimum verified stats met
6. **Retry Logic**: If needed, request more candidates and verify
7. **Response**: Return verified statistics in structured JSON format

## Reputable Sources

The research agent prioritizes these source types:

- **Government Agencies**: CDC, NIH, Census Bureau, EPA, etc.
- **Academic Institutions**: Universities, research journals
- **Research Organizations**: Pew Research, Gallup, McKinsey, etc.
- **International Organizations**: WHO, UN, World Bank, IMF, etc.
- **Respected Media**: With proper citations (NYT, WSJ, Economist, etc.)

## Error Handling

- **Source Unreachable**: Marked as failed with reason
- **Excerpt Not Found**: Verification fails with explanation
- **Value Mismatch**: Flagged as discrepancy
- **Insufficient Results**: Automatic retry with more candidates
- **Max Retries Exceeded**: Returns partial results with warning

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Acknowledgments

- Built with [Google ADK (Agent Development Kit)](https://github.com/google/adk-go)
- Uses [Eino](https://github.com/cloudwego/eino) for deterministic orchestration
- Powered by Google Gemini 2.0 Flash model
- Inspired by multi-agent collaboration frameworks

 [used-by-svg]: https://sourcegraph.com/github.com/grokify/stats-agent-team/-/badge.svg
 [used-by-url]: https://sourcegraph.com/github.com/grokify/stats-agent-team?badge
 [build-status-svg]: https://github.com/grokify/stats-agent-team/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/grokify/stats-agent-team/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/grokify/stats-agent-team/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/grokify/stats-agent-team/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/stats-agent-team
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/stats-agent-team
 [codeclimate-status-svg]: https://codeclimate.com/github/grokify/stats-agent-team/badges/gpa.svg
 [codeclimate-status-url]: https://codeclimate.com/github/grokify/stats-agent-team
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/stats-agent-team
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/stats-agent-team
 [loc-svg]: https://tokei.rs/b1/github/grokify/stats-agent-team
 [repo-url]: https://github.com/grokify/stats-agent-team
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/stats-agent-team/blob/master/LICENSE
