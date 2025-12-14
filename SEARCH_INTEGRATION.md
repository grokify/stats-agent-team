# Search Integration Guide

This document explains how the Statistics Agent Team integrates with web search APIs to find real statistics.

## Overview

The research agent uses the [grokify/metasearch](https://github.com/grokify/metasearch) library to perform web searches across multiple search engine providers. This enables the system to find real, verifiable statistics from reputable sources on the web.

## Supported Search Providers

| Provider | Website | Features | Cost |
|----------|---------|----------|------|
| **Serper** | [serper.dev](https://serper.dev) | Fast, affordable, all search types | $50/month for 5,000 searches |
| **SerpAPI** | [serpapi.com](https://serpapi.com) | Comprehensive, reliable | $50/month for 5,000 searches |

Both providers offer:
- Real-time Google search results
- Structured JSON responses
- High reliability and speed
- Free trial credits for testing

## Configuration

### 1. Get an API Key

**For Serper (Recommended):**
1. Visit https://serper.dev
2. Sign up for an account
3. Get your API key from https://serper.dev/api-key
4. Free trial: 2,500 searches

**For SerpAPI:**
1. Visit https://serpapi.com
2. Sign up for an account
3. Get your API key from dashboard
4. Free trial: 100 searches/month

### 2. Set Environment Variables

```bash
# Option 1: Serper (recommended)
export SEARCH_PROVIDER=serper
export SERPER_API_KEY=your-serper-api-key-here

# Option 2: SerpAPI
export SEARCH_PROVIDER=serpapi
export SERPAPI_API_KEY=your-serpapi-key-here
```

Or add to your `.env` file:

```bash
# Copy from example
cp .env.example .env

# Edit .env with your API key
SEARCH_PROVIDER=serper
SERPER_API_KEY=your-serper-api-key-here
```

### 3. Verify Configuration

The research agent will log which search provider it's using:

```
Research Agent: Using serper search provider
```

If no search API is configured, you'll see:

```
Warning: Search service not available: SERPER_API_KEY is required when using serper provider
Research agent will use mock data. Set SERPER_API_KEY or SERPAPI_API_KEY to enable real search.
```

## How It Works

### Architecture

```
┌─────────────────────────────────────────────┐
│          Research Agent (ADK)               │
│                                             │
│  ┌────────────────────────────────────┐    │
│  │   Search Service                   │    │
│  │   (pkg/search/service.go)          │    │
│  │                                    │    │
│  │   ┌──────────────────────────┐    │    │
│  │   │  Metasearch Library      │    │    │
│  │   │  - Serper Client         │    │    │
│  │   │  - SerpAPI Client        │    │    │
│  │   └──────────────────────────┘    │    │
│  └────────────────────────────────────┘    │
│                                             │
│  LLM analyzes search results to extract    │
│  statistics, sources, and excerpts         │
└─────────────────────────────────────────────┘
                    │
                    ▼
        ┌──────────────────────┐
        │  Search API          │
        │  (Serper/SerpAPI)    │
        └──────────────────────┘
                    │
                    ▼
        ┌──────────────────────┐
        │  Google Search       │
        │  Results             │
        └──────────────────────┘
```

### Search Flow

1. **User requests statistics** on a topic (e.g., "climate change")
2. **Research agent** calls search service
3. **Search service** uses metasearch to query Serper/SerpAPI
4. **Search API** returns Google search results
5. **Research agent** analyzes results (currently extracts metadata, will use LLM in future)
6. **Candidate statistics** are created with:
   - Source URLs for verification
   - Snippets containing potential statistics
   - Source attribution
7. **Verification agent** fetches actual URLs to verify excerpts

### Current Implementation

The current implementation:
- ✅ Performs real web searches via Serper/SerpAPI
- ✅ Returns actual URLs and snippets from search results
- ✅ Gracefully falls back to mock data if search unavailable
- ⚠️ **TODO**: Use LLM to analyze search result content and extract actual statistics

### Future Enhancement

The next step is to use the LLM to:
1. Fetch full page content from search result URLs
2. Analyze content to find numerical statistics
3. Extract exact values, units, and context
4. Identify reputable sources (academic, government, research)
5. Create high-quality candidate statistics

## Usage Examples

### Basic Search

```bash
# Start research agent with search configured
export SERPER_API_KEY=your-key-here
make run-research
```

### Via HTTP API

```bash
curl -X POST http://localhost:8001/research \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "renewable energy adoption",
    "min_statistics": 5,
    "max_statistics": 10,
    "reputable_only": true
  }'
```

Response will include real search results:

```json
{
  "topic": "renewable energy adoption",
  "candidates": [
    {
      "name": "Statistic about renewable energy adoption from iea.org",
      "value": 10,
      "unit": "%",
      "source": "Iea.org",
      "source_url": "https://www.iea.org/reports/renewables-2023",
      "excerpt": "Renewable energy capacity is set to expand by 50% between 2023 and 2028..."
    }
  ],
  "timestamp": "2025-12-13T10:30:00Z"
}
```

### Via Docker

```bash
# Add to .env file
SEARCH_PROVIDER=serper
SERPER_API_KEY=your-key-here

# Start with Docker
docker-compose up -d

# Check logs to verify search is working
docker-compose logs -f | grep "search provider"
```

## Monitoring and Debugging

### Check Search Service Status

Look for these log messages:

**Success:**
```
Research Agent: Using serper search provider
Research Agent: Found 10 search results
```

**Fallback to Mock Data:**
```
Warning: Search service not available: SERPER_API_KEY is required
Research agent will use mock data
Using mock data (search service not configured)
```

**Search Error:**
```
Search failed, falling back to mock data: search failed: API error
```

### Common Issues

**Issue: "SERPER_API_KEY is required"**
- Solution: Set the appropriate API key environment variable
- Verify: `echo $SERPER_API_KEY`

**Issue: Search returns no results**
- Check API key is valid
- Verify API quota hasn't been exceeded
- Check serper.dev or serpapi.com dashboard for usage

**Issue: "unsupported search provider"**
- Solution: Use `SEARCH_PROVIDER=serper` or `SEARCH_PROVIDER=serpapi`
- Default is `serper` if not specified

## API Costs

### Serper Pricing

- **Free Plan**: 2,500 searches (trial)
- **Hobby**: $50/month for 5,000 searches ($0.01 per search)
- **Startup**: $200/month for 25,000 searches ($0.008 per search)

### SerpAPI Pricing

- **Free**: 100 searches/month
- **Developer**: $50/month for 5,000 searches
- **Production**: $250/month for 30,000 searches

### Cost Optimization

The research agent is designed to be cost-effective:
- Configurable number of search results per query
- Graceful fallback to mock data on errors
- Single search per research request
- Results cached within the session

**Estimated costs** for typical usage:
- 100 statistics queries/day = 100 searches = ~$1/month (Serper)
- 1,000 statistics queries/day = 1,000 searches = ~$10/month (Serper)

## Implementation Details

### Code Structure

```
pkg/search/
  └── service.go          # Search service wrapper

agents/research/
  └── main.go            # Research agent with search integration
      ├── searchForStatistics()    # Uses search service
      ├── extractSource()          # Helper to parse URLs
      └── generateMockCandidates() # Fallback data
```

### Key Functions

**pkg/search/service.go:**
- `NewService(cfg)` - Creates search service with provider
- `Search(ctx, query, num)` - Basic web search
- `SearchForStatistics(ctx, topic, num)` - Optimized for statistics

**agents/research/main.go:**
- `searchForStatistics()` - Performs search and creates candidates
- Graceful fallback on errors
- LLM will be integrated for content analysis

## Switching Providers

To switch from Serper to SerpAPI (or vice versa):

```bash
# Stop current service
make stop # or docker-compose down

# Update environment
export SEARCH_PROVIDER=serpapi
export SERPAPI_API_KEY=your-serpapi-key

# Restart
make run-research # or docker-compose up -d
```

The change takes effect immediately on restart.

## Next Steps

Future enhancements planned:

1. **LLM Content Analysis** - Use ADK agent to analyze search result pages
2. **Source Ranking** - Prioritize reputable sources (academic, government)
3. **Statistics Extraction** - Parse numerical values and context from text
4. **Caching** - Cache search results to reduce API calls
5. **Rate Limiting** - Intelligent request throttling
6. **Multiple Search Types** - News, scholar, images for different use cases

## References

- [Metasearch Library](https://github.com/grokify/metasearch)
- [Serper API Docs](https://serper.dev/documentation)
- [SerpAPI Docs](https://serpapi.com/search-api)
- [Google ADK Docs](https://google.github.io/adk-docs/)

## Support

If you encounter issues with search integration:

1. Check the [Troubleshooting](#monitoring-and-debugging) section
2. Review search provider dashboard for errors
3. Enable debug logging: `export LOG_LEVEL=debug`
4. Open an issue with logs and error messages
