# Statistics Agent

A multi-agent system for finding and verifying statistics from reputable web sources using Go, built with [trpc-agent-go](https://github.com/trpc-group/trpc-agent-go) and [trpc-a2a-go](https://github.com/trpc-group/trpc-a2a-go).

## Overview

This project implements a sophisticated multi-agent architecture that leverages LLMs and web search to find verifiable statistics, prioritizing well-known and respected publishers. The system ensures accuracy through automated verification of sources.

## Architecture

The system consists of **3 specialized agents** with **2 orchestration options**:

> **NEW**: This project now includes **two orchestration agents** - one using trpc-agent (LLM-based) and one using Eino (deterministic graph-based). See [README_EINO.md](README_EINO.md) for details on the Eino orchestrator.

```
┌─────────────────────────────────────────────────────────┐
│                   User Request                          │
│              "Find climate change statistics"           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            ORCHESTRATION AGENT                          │
│            (Port 8000 / 9000)                           │
│  • Coordinates workflow                                 │
│  • Manages retry logic                                  │
│  • Ensures quality standards                            │
└──────────┬──────────────────────────┬───────────────────┘
           │                          │
           ▼                          ▼
┌──────────────────────┐    ┌─────────────────────────────┐
│  RESEARCH AGENT      │    │  VERIFICATION AGENT         │
│  (Port 8001 / 9001)  │    │  (Port 8002 / 9002)        │
│                      │    │                             │
│  • Web searches      │───▶│  • Fetches source URLs      │
│  • Finds candidates  │    │  • Validates excerpts       │
│  • Extracts stats    │    │  • Checks numerical values  │
└──────────────────────┘    └─────────────────────────────┘
```

### Agent Responsibilities

#### 1. Research Agent (`agents/research/`)
- Executes web searches for statistics on given topics
- Prioritizes reputable sources (academic, government, research organizations)
- Extracts candidate statistics with context
- Returns structured data for verification

#### 2. Verification Agent (`agents/verification/`)
- Fetches actual source content from URLs
- Searches for verbatim excerpts in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with reasons

#### 3a. Orchestration Agent - trpc-agent (`agents/orchestration/`)
- LLM-based decision making for workflow
- Coordinates Research → Verification workflow
- Implements adaptive retry logic
- Dynamic quality control
- Port 8000 (HTTP), 9000 (A2A)

#### 3b. Orchestration Agent - Eino (`agents/orchestration-eino/`) ⭐ NEW
- Deterministic graph-based workflow
- Type-safe orchestration with compile-time checks
- Predictable, reproducible behavior
- Faster and lower cost (no LLM for orchestration)
- Port 8003 (HTTP), 9003 (A2A)
- **Recommended for production use**

## Features

- ✅ **Multi-agent orchestration** with chains and workflow coordination
- ✅ **A2A protocol support** for agent-to-agent communication
- ✅ **LLM integration** via trpc-agent-go
- ✅ **Source verification** to prevent hallucinations
- ✅ **Reputable source prioritization** (government, academic, research orgs)
- ✅ **Structured JSON output** with complete metadata
- ✅ **HTTP and A2A APIs** for both protocols
- ✅ **Retry logic** for ensuring quality results

## Output Format

The system returns verified statistics in JSON format:

```json
[
  {
    "name": "Global temperature increase since pre-industrial times",
    "value": "1.1°C",
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
- **value**: Numerical value (can be number or percentage)
- **source**: Name of source organization/publication
- **source_url**: URL to the original source
- **excerpt**: Verbatim quote containing the statistic
- **verified**: Whether the verification agent confirmed it
- **date_found**: Timestamp when statistic was found

## Installation

### Prerequisites

- Go 1.21 or higher
- API keys for LLM provider (OpenAI, etc.)
- API keys for search provider (Google, etc.)

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
go get github.com/trpc-group/trpc-agent-go
go get github.com/trpc-group/trpc-a2a-go
```

3. Configure environment variables:
```bash
cp .env.example .env
# Edit .env with your API keys
```

4. Build the agents:
```bash
make build
```

## Usage

### Running the Agents

#### Option 1: Run all agents with Eino orchestrator (Recommended)
```bash
make run-all-eino
```

#### Option 2: Run all agents with trpc-agent orchestrator
```bash
make run-all
```

#### Option 3: Run each agent separately (in different terminals)
```bash
# Terminal 1: Research Agent
make run-research

# Terminal 2: Verification Agent
make run-verification

# Terminal 3: Orchestration Agent (choose one)
make run-orchestration       # trpc-agent version
make run-orchestration-eino  # Eino version (recommended)
```

### Using the CLI

Once the agents are running, use the CLI to search for statistics:

```bash
# Basic search
./bin/stats-agent search "climate change"

# More examples
./bin/stats-agent search "artificial intelligence adoption rates"
./bin/stats-agent search "cybersecurity threats 2024"
./bin/stats-agent search "renewable energy statistics"
```

### API Usage

You can also call the agents directly via HTTP:

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

# Or call trpc-agent orchestration agent (LLM-based)
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

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROVIDER` | LLM provider name | `openai` |
| `LLM_API_KEY` | API key for LLM | Required |
| `LLM_MODEL` | Model to use | `gpt-4` |
| `SEARCH_PROVIDER` | Search provider | `google` |
| `SEARCH_API_KEY` | Search API key | Required |
| `RESEARCH_AGENT_URL` | Research agent URL | `http://localhost:8001` |
| `VERIFICATION_AGENT_URL` | Verification agent URL | `http://localhost:8002` |
| `ORCHESTRATOR_URL` | trpc-agent orchestrator URL | `http://localhost:8000` |
| `ORCHESTRATOR_EINO_URL` | Eino orchestrator URL | `http://localhost:8003` |
| `A2A_ENABLED` | Enable A2A protocol | `true` |
| `A2A_AUTH_TYPE` | A2A auth type | `apikey` |
| `A2A_AUTH_TOKEN` | A2A auth token | Optional |

### Port Configuration

| Agent | HTTP Port | A2A Port |
|-------|-----------|----------|
| Orchestration (trpc-agent) | 8000 | 9000 |
| Research | 8001 | 9001 |
| Verification | 8002 | 9002 |
| **Orchestration (Eino)** ⭐ | **8003** | **9003** |

## Project Structure

```
stats-agent/
├── agents/
│   ├── orchestration/      # Orchestration agent (trpc-agent)
│   │   └── main.go
│   ├── orchestration-eino/ # Orchestration agent (Eino) ⭐
│   │   └── main.go
│   ├── research/           # Research agent
│   │   └── main.go
│   └── verification/       # Verification agent
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
  - [trpc-agent-go](https://github.com/trpc-group/trpc-agent-go) - LLM-based orchestration
  - [Eino](https://github.com/cloudwego/eino) - Deterministic graph orchestration ⭐
- **A2A Protocol**: [trpc-a2a-go](https://github.com/trpc-group/trpc-a2a-go)
- **LLM**: Configurable (OpenAI, etc.)
- **Search**: Configurable (Google, etc.)

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

## License

[Your chosen license]

## Contact

[Your contact information]

## Acknowledgments

- Built with [trpc-agent-go](https://github.com/trpc-group/trpc-agent-go)
- Uses [A2A protocol](https://github.com/trpc-group/trpc-a2a-go)
- Inspired by multi-agent collaboration frameworks
