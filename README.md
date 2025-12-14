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

The system consists of **3 specialized agents** with **2 orchestration options**:

> **Architecture**: Research and Verification agents are built with **Google ADK** for LLM-based operations. **Two orchestration agents** available: ADK-based (LLM-driven decisions) and Eino-based (deterministic graph workflow). See [README_EINO.md](README_EINO.md) for Eino orchestrator details.

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
│  (Port 8001 / 9001)  │    │  (Port 8002 / 9002)         │
│                      │    │                             │
│  • Web searches      │───▶│  • Fetches source URLs      │
│  • Finds candidates  │    │  • Validates excerpts       │
│  • Extracts stats    │    │  • Checks numerical values  │
└──────────────────────┘    └─────────────────────────────┘
```

### Agent Responsibilities

#### 1. Research Agent (`agents/research/`) - Google ADK
- Built with Google ADK and Gemini 2.0 Flash model
- Executes web searches for statistics on given topics
- Prioritizes reputable sources (academic, government, research organizations)
- Extracts candidate statistics with context
- Returns structured data for verification
- Port 8001 (HTTP)

#### 2. Verification Agent (`agents/verification/`) - Google ADK
- Built with Google ADK and Gemini 2.0 Flash model
- Fetches actual source content from URLs
- Searches for verbatim excerpts in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with reasons
- Port 8002 (HTTP)

#### 3a. Orchestration Agent - Google ADK (`agents/orchestration/`)
- Built with Google ADK and Gemini 2.0 Flash model
- LLM-based decision making for workflow
- Coordinates Research → Verification workflow
- Implements adaptive retry logic
- Dynamic quality control
- Port 8000 (HTTP)

#### 3b. Orchestration Agent - Eino (`agents/orchestration-eino/`) ⭐ RECOMMENDED
- Deterministic graph-based workflow
- Type-safe orchestration with compile-time checks
- Predictable, reproducible behavior
- Faster and lower cost (no LLM for orchestration)
- Calls ADK-based research and verification agents
- Port 8003 (HTTP)
- **Recommended for production use**

## Features

- ✅ **Multi-agent orchestration** with chains and workflow coordination
- ✅ **Google ADK integration** for LLM-based agents
- ✅ **Eino framework** for deterministic graph orchestration
- ✅ **Gemini 2.0 Flash model** for fast, accurate LLM operations
- ✅ **Source verification** to prevent hallucinations
- ✅ **Reputable source prioritization** (government, academic, research orgs)
- ✅ **Structured JSON output** with complete metadata
- ✅ **HTTP APIs** for all agents
- ✅ **Retry logic** for ensuring quality results
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
- Google API key for Gemini (set as `GOOGLE_API_KEY` environment variable)
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
export GOOGLE_API_KEY="your-google-api-key"

# Optional: Create .env file
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

#### Option 2: Run all agents with ADK orchestrator
```bash
make run-all
```

#### Option 3: Run each agent separately (in different terminals)
```bash
# Terminal 1: Research Agent (ADK)
make run-research

# Terminal 2: Verification Agent (ADK)
make run-verification

# Terminal 3: Orchestration Agent (choose one)
make run-orchestration       # ADK version (LLM-based)
make run-orchestration-eino  # Eino version (deterministic, recommended)
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

| Variable | Description | Default |
|----------|-------------|---------|
| `GOOGLE_API_KEY` | Google API key for Gemini | **Required** |
| `SEARCH_PROVIDER` | Search provider | `google` |
| `SEARCH_API_KEY` | Search API key | Optional |
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
- **LLM Model**: Google Gemini 2.0 Flash (via `google.golang.org/genai`)
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
 [build-status-svg]: https://github.com/grokify/stats-agent-team/actions/workflows/ci.yaml/badge.svg?branch=master
 [build-status-url]: https://github.com/grokify/stats-agent-team/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/grokify/stats-agent-team/actions/workflows/lint.yaml/badge.svg?branch=master
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
 