# 4-Agent Architecture Implementation

## Overview

The Statistics Agent Team now uses a **4-agent architecture** with clear separation of concerns:

```
┌─────────────────────────────────────────────────────┐
│         Orchestration Agent (Eino/ADK)              │
│              Port 8003 / 8000                       │
└────────┬──────────────┬──────────────┬──────────────┘
         │              │              │
         ▼              ▼              ▼
┌────────────────┐ ┌──────────────┐ ┌────────────────┐
│   Research     │ │  Synthesis   │ │ Verification   │
│    Agent       │ │    Agent     │ │    Agent       │
│   Port 8001    │ │  Port 8004   │ │   Port 8002    │
│                │ │              │ │                │
│ - Web Search   │ │ - Fetch URLs │ │ - Verify URLs  │
│ - Find Sources │ │ - LLM Extract│ │ - Check Facts  │
│ - Filter       │ │ - Parse Stats│ │ - Validate     │
│   Reputable    │ │              │ │   Excerpts     │
└────────────────┘ └──────────────┘ └────────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
   Serper/SerpAPI    Webpage Content    Source Validation
```

## Agent Responsibilities

### 1. Research Agent (Port 8001)
**Role**: Source Discovery
**Technology**: Web Search (Serper/SerpAPI)
**No LLM Required**

**Tasks**:
- Perform web searches via `pkg/search` service
- Return URLs with metadata (title, snippet, domain)
- Filter for reputable sources (.gov, .edu, research orgs)
- **Output**: List of SearchResult objects

**Files**:
- `agents/research/main.go` - Simplified to focus on search only
- `pkg/search/service.go` - Metasearch integration

**API**:
```json
POST /research
{
  "topic": "climate change",
  "max_statistics": 20,
  "reputable_only": true
}

Response:
{
  "topic": "climate change",
  "candidates": [/* URLs as placeholder candidates */],
  "timestamp": "2025-12-13T10:30:00Z"
}
```

### 2. Synthesis Agent (Port 8004) ⭐ NEW
**Role**: Statistics Extraction
**Technology**: ADK + LLM (Gemini/Claude/OpenAI/Ollama)
**LLM-Heavy**

**Tasks**:
- Fetch webpage content from URLs
- Use LLM to analyze text
- Extract numerical values, units, and context
- Find verbatim excerpts containing statistics
- Create candidate statistics with proper metadata
- **Output**: List of CandidateStatistic objects

**Files**:
- `agents/synthesis/main.go` - NEW agent with LLM analysis
- Uses ADK for intelligent content analysis
- Current implementation: Regex-based (TODO: Full LLM integration)

**API**:
```json
POST /synthesize
{
  "topic": "climate change",
  "search_results": [
    {
      "url": "https://www.iea.org/...",
      "title": "Renewable Energy Report",
      "snippet": "...",
      "domain": "iea.org"
    }
  ],
  "min_statistics": 5,
  "max_statistics": 20
}

Response:
{
  "topic": "climate change",
  "candidates": [
    {
      "name": "Renewable energy growth",
      "value": 83,
      "unit": "%",
      "source": "iea.org",
      "source_url": "https://www.iea.org/...",
      "excerpt": "Renewable capacity grew by 83% in 2023..."
    }
  ],
  "sources_analyzed": 5,
  "timestamp": "2025-12-13T10:30:15Z"
}
```

### 3. Verification Agent (Port 8002)
**Role**: Fact Checking
**Technology**: ADK + LLM (light usage)
**LLM-Light**

**Tasks**:
- Re-fetch source URLs
- Verify excerpts exist verbatim in source
- Check numerical values match
- Flag hallucinations or mismatches
- **Output**: VerificationResult objects with pass/fail

**Files**:
- `agents/verification/main.go` - Unchanged

**API**: (Unchanged)

### 4. Orchestration Agent (Ports 8003/8000)
**Role**: Workflow Coordination
**Technology**: Eino (recommended) or ADK
**No LLM** (Eino) or **LLM-driven** (ADK)

**Workflow**:
1. Call Research Agent → get URLs
2. Call Synthesis Agent → extract statistics from URLs
3. Call Verification Agent → validate statistics
4. Retry logic if needed
5. Return verified statistics

**Files**:
- `agents/orchestration-eino/main.go` - Eino version (deterministic)
- `agents/orchestration/main.go` - ADK version (LLM-driven)
- `pkg/orchestration/eino.go` - Shared Eino logic

## Data Flow

```
User Request
     │
     ▼
┌─────────────────┐
│ Orchestration   │
└────────┬────────┘
         │
         │ 1. Search for sources
         ▼
┌─────────────────┐
│ Research Agent  │ ──► Returns: [{url, title, snippet, domain}, ...]
└────────┬────────┘
         │
         │ 2. Extract statistics from URLs
         ▼
┌─────────────────┐
│ Synthesis Agent │ ──► Returns: [{name, value, unit, source_url, excerpt}, ...]
└────────┬────────┘
         │
         │ 3. Verify statistics
         ▼
┌─────────────────┐
│ Verification    │ ──► Returns: [{statistic, verified: true/false, reason}, ...]
└────────┬────────┘
         │
         ▼
    Verified Statistics
```

## Models (pkg/models/statistic.go)

### New Models Added:

```go
// SearchResult - Output from Research Agent
type SearchResult struct {
    URL      string
    Title    string
    Snippet  string
    Domain   string
    Position int
}

// SynthesisRequest - Input to Synthesis Agent
type SynthesisRequest struct {
    Topic         string
    SearchResults []SearchResult
    MinStatistics int
    MaxStatistics int
}

// SynthesisResponse - Output from Synthesis Agent
type SynthesisResponse struct {
    Topic           string
    Candidates      []CandidateStatistic
    SourcesAnalyzed int
    Timestamp       time.Time
}
```

## Implementation Status

### ✅ Completed

1. **Synthesis Agent Created** (`agents/synthesis/main.go`)
   - ADK integration
   - Webpage fetching
   - Basic regex extraction (placeholder)
   - HTTP API on port 8004

2. **Research Agent Refactored** (`agents/research/main.go`)
   - Removed LLM/ADK dependencies
   - Focus on search only
   - Returns SearchResult objects
   - Reputable source filtering

3. **Models Updated** (`pkg/models/statistic.go`)
   - SearchResult model
   - SynthesisRequest/Response models

4. **Configuration Updated** (`pkg/config/config.go`)
   - Added SynthesisAgentURL (port 8004)

### 🚧 In Progress

1. **Orchestration Update** - Need to update to call all 4 agents

### ⏳ TODO

1. **Eino Orchestration** (`pkg/orchestration/eino.go`)
   - Add synthesis agent call between research and verification
   - Update workflow graph

2. **ADK Orchestration** (`agents/orchestration/main.go`)
   - Add HTTP client for synthesis agent
   - Update orchestrate() method

3. **Docker Configuration**
   - Add synthesis agent to Dockerfile
   - Update docker-compose.yml with port 8004
   - Add to docker-entrypoint.sh

4. **Makefile Updates**
   - Add `run-synthesis` target
   - Update `run-all-eino` to include synthesis
   - Update `build` target

5. **Documentation Updates**
   - README.md - Update architecture diagram
   - DOCKER.md - Add synthesis agent info
   - Architecture documentation

6. **LLM Integration in Synthesis** (Future Enhancement)
   - Replace regex with full LLM analysis
   - Use ADK to intelligently parse content
   - Extract context and validate statistics

## Benefits of 4-Agent Architecture

✅ **Separation of Concerns** - Each agent has one job
✅ **Better Caching** - Research results can be reused
✅ **Parallel Processing** - Synthesize multiple URLs concurrently
✅ **Cost Optimization** - Only use LLM where needed (synthesis)
✅ **Easier Testing** - Mock each agent independently
✅ **Scalability** - Scale synthesis agents based on load
✅ **Flexibility** - Can run all in-process or as microservices

## Next Steps

To complete the 4-agent architecture:

1. Update orchestration agents to call synthesis
2. Update Docker/Makefile for deployment
3. Enhance synthesis agent with full LLM integration
4. Update all documentation
5. Test end-to-end workflow

## Port Mapping

| Agent | Port | Role |
|-------|------|------|
| Orchestration (ADK) | 8000 | Workflow (LLM-driven) |
| Research | 8001 | Search (no LLM) |
| Verification | 8002 | Validation (LLM-light) |
| Orchestration (Eino) | 8003 | Workflow (deterministic) |
| **Synthesis** | **8004** | **Extraction (LLM-heavy)** ⭐ |
