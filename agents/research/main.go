package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/llm"
	"github.com/grokify/stats-agent-team/pkg/models"
	"github.com/grokify/stats-agent-team/pkg/search"
)

// ResearchAgent wraps an ADK LLM agent for finding statistics
type ResearchAgent struct {
	cfg       *config.Config
	client    *http.Client
	adkAgent  agent.Agent
	searchSvc *search.Service
}

// ResearchInput defines the input for the research tool
type ResearchInput struct {
	Topic         string `json:"topic" jsonschema:"description=The topic to research statistics for"`
	MinStatistics int    `json:"min_statistics" jsonschema:"description=Minimum number of statistics to find"`
	MaxStatistics int    `json:"max_statistics" jsonschema:"description=Maximum number of statistics to find"`
}

// ResearchOutput defines the output from the research tool
type ResearchOutput struct {
	Candidates []models.CandidateStatistic `json:"candidates"`
}

// NewResearchAgent creates a new ADK-based research agent
func NewResearchAgent(cfg *config.Config) (*ResearchAgent, error) {
	ctx := context.Background()

	// Create model using factory
	modelFactory := llm.NewModelFactory(cfg)
	model, err := modelFactory.CreateModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	log.Printf("Research Agent: Using %s", modelFactory.GetProviderInfo())

	// Create search service
	searchSvc, err := search.NewService(cfg)
	if err != nil {
		log.Printf("Warning: Search service not available: %v", err)
		log.Printf("Research agent will use mock data. Set SERPER_API_KEY or SERPAPI_API_KEY to enable real search.")
	} else {
		log.Printf("Research Agent: Using %s search provider", cfg.SearchProvider)
	}

	ra := &ResearchAgent{
		cfg:       cfg,
		client:    &http.Client{Timeout: 30 * time.Second},
		searchSvc: searchSvc,
	}

	// Create the research tool function
	researchTool, err := functiontool.New(functiontool.Config{
		Name:        "research_statistics",
		Description: "Searches for and extracts statistics on a given topic from reputable sources",
	}, ra.researchToolHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create research tool: %w", err)
	}

	// Create the ADK agent
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "statistics_research_agent",
		Model:       model,
		Description: "Finds verifiable statistics from reputable web sources",
		Instruction: `You are a statistics research agent. Your job is to:
1. Search the web for relevant statistics on the given topic
2. Prioritize reputable sources (academic journals, government agencies, established research organizations)
3. Extract numerical values with their context
4. Capture verbatim excerpts that contain the statistic
5. Return well-structured candidate statistics

Reputable sources include:
- Government agencies (CDC, NIH, Census Bureau, etc.)
- Academic institutions and journals
- Established research organizations (Pew Research, Gallup, etc.)
- International organizations (WHO, UN, World Bank, etc.)

Always include the exact URL and a verbatim quote containing the statistic.`,
		Tools: []tool.Tool{researchTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	ra.adkAgent = adkAgent

	return ra, nil
}

// researchToolHandler implements the actual research logic
func (ra *ResearchAgent) researchToolHandler(ctx tool.Context, input ResearchInput) (ResearchOutput, error) {
	log.Printf("Research Agent: Searching for statistics on topic: %s", input.Topic)

	// Use real search if available, otherwise fall back to mock data
	if ra.searchSvc != nil {
		candidates, err := ra.searchForStatistics(ctx, input.Topic, input.MinStatistics, input.MaxStatistics)
		if err != nil {
			log.Printf("Search failed, falling back to mock data: %v", err)
			return ResearchOutput{
				Candidates: ra.generateMockCandidates(input.Topic, input.MinStatistics),
			}, nil
		}
		return ResearchOutput{
			Candidates: candidates,
		}, nil
	}

	// Fall back to mock data if search service not configured
	log.Printf("Using mock data (search service not configured)")
	return ResearchOutput{
		Candidates: ra.generateMockCandidates(input.Topic, input.MinStatistics),
	}, nil
}

// searchForStatistics uses the search service to find real statistics
func (ra *ResearchAgent) searchForStatistics(ctx context.Context, topic string, minStats, maxStats int) ([]models.CandidateStatistic, error) {
	// Determine how many results to request
	numResults := maxStats
	if numResults == 0 {
		numResults = 20 // Default to more results to have options
	}

	// Perform search
	searchResp, err := ra.searchSvc.SearchForStatistics(ctx, topic, numResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	log.Printf("Research Agent: Found %d search results", searchResp.Total)

	// Extract statistics from search results
	// For now, create candidates from search results
	// In a production system, you would use the LLM to analyze each result
	candidates := make([]models.CandidateStatistic, 0)

	for i, result := range searchResp.Results {
		if len(candidates) >= maxStats && maxStats > 0 {
			break
		}

		// Create a candidate from the search result
		// TODO: Use LLM to extract actual statistics from the page content
		candidate := models.CandidateStatistic{
			Name:      fmt.Sprintf("Statistic about %s from %s", topic, result.DisplayLink),
			Value:     float32((i + 1) * 10), // Placeholder value
			Unit:      "%",                    // Placeholder unit
			Source:    extractSource(result.DisplayLink),
			SourceURL: result.URL,
			Excerpt:   result.Snippet,
		}

		candidates = append(candidates, candidate)
	}

	if len(candidates) < minStats {
		log.Printf("Warning: Only found %d candidates, requested minimum %d", len(candidates), minStats)
	}

	return candidates, nil
}

// extractSource extracts a clean source name from a URL
func extractSource(displayLink string) string {
	// Remove www. prefix and clean up
	source := strings.TrimPrefix(displayLink, "www.")
	// Capitalize first letter
	if len(source) > 0 {
		source = strings.ToUpper(source[:1]) + source[1:]
	}
	return source
}

// generateMockCandidates creates mock data for demonstration
func (ra *ResearchAgent) generateMockCandidates(topic string, count int) []models.CandidateStatistic {
	if count < 5 {
		count = 5
	}

	candidates := make([]models.CandidateStatistic, count)
	for i := 0; i < count; i++ {
		candidates[i] = models.CandidateStatistic{
			Name:      fmt.Sprintf("Statistic #%d about %s", i+1, topic),
			Value:     float32((i + 1) * 10),
			Unit:      "%",
			Source:    "Pew Research Center",
			SourceURL: fmt.Sprintf("https://www.pewresearch.org/example-%d", i+1),
			Excerpt:   fmt.Sprintf("According to our latest survey, %d%% of respondents reported...", (i+1)*10),
		}
	}
	return candidates
}

// Research performs research directly
//
//nolint:unparam // error return kept for API consistency, will be used when real implementation replaces mock
func (ra *ResearchAgent) Research(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	log.Printf("Research Agent: Searching for statistics on topic: %s", req.Topic)

	var candidates []models.CandidateStatistic
	var err error

	// Use real search if available
	if ra.searchSvc != nil {
		candidates, err = ra.searchForStatistics(ctx, req.Topic, req.MinStatistics, req.MaxStatistics)
		if err != nil {
			log.Printf("Search failed, falling back to mock data: %v", err)
			candidates = ra.generateMockCandidates(req.Topic, req.MinStatistics)
		}
	} else {
		// Fall back to mock data if search service not configured
		log.Printf("Using mock data (search service not configured)")
		candidates = ra.generateMockCandidates(req.Topic, req.MinStatistics)
	}

	response := &models.ResearchResponse{
		Topic:      req.Topic,
		Candidates: candidates,
		Timestamp:  time.Now(),
	}

	log.Printf("Research Agent: Found %d candidate statistics", len(candidates))
	return response, nil
}

// HandleResearchRequest is the HTTP handler for research requests
func (ra *ResearchAgent) HandleResearchRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MinStatistics == 0 {
		req.MinStatistics = 5
	}
	if req.MaxStatistics == 0 {
		req.MaxStatistics = 10
	}

	resp, err := ra.Research(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Research failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func main() {
	cfg := config.LoadConfig()

	researchAgent, err := NewResearchAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create research agent: %v", err)
	}

	// Start HTTP server with timeout
	server := &http.Server{
		Addr:         ":8001",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	http.HandleFunc("/research", researchAgent.HandleResearchRequest)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	log.Println("Research Agent HTTP server starting on :8001")
	log.Println("(ADK agent initialized for future A2A integration)")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
