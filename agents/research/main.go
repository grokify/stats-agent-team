package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/models"
	"github.com/grokify/stats-agent-team/pkg/search"
)

// ResearchAgent finds relevant sources using web search
// Note: This agent now focuses ONLY on search - no LLM analysis
// Statistics extraction is handled by the Synthesis Agent
type ResearchAgent struct {
	cfg       *config.Config
	client    *http.Client
	searchSvc *search.Service
}

// ResearchInput defines the input for the research tool
type ResearchInput struct {
	Topic         string `json:"topic" jsonschema:"description=The topic to research statistics for"`
	NumResults    int    `json:"num_results" jsonschema:"description=Number of search results to return"`
	ReputableOnly bool   `json:"reputable_only" jsonschema:"description=Only return reputable sources"`
}

// ResearchOutput defines the output from the research tool
type ResearchOutput struct {
	SearchResults []models.SearchResult `json:"search_results"`
}

// NewResearchAgent creates a new search-focused research agent
func NewResearchAgent(cfg *config.Config) (*ResearchAgent, error) {
	// Create search service
	searchSvc, err := search.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("search service required: %w", err)
	}

	log.Printf("Research Agent: Using %s search provider", cfg.SearchProvider)
	log.Printf("Research Agent: Focuses on finding relevant sources (no LLM analysis)")

	ra := &ResearchAgent{
		cfg:       cfg,
		client:    &http.Client{Timeout: 30 * time.Second},
		searchSvc: searchSvc,
	}

	return ra, nil
}

// findSources performs web search and returns relevant URLs
func (ra *ResearchAgent) findSources(ctx context.Context, topic string, numResults int, reputableOnly bool) ([]models.SearchResult, error) {
	log.Printf("Research Agent: Searching for sources on topic: %s", topic)

	if numResults <= 0 {
		numResults = 10
	}

	// Perform search
	searchResp, err := ra.searchSvc.SearchForStatistics(ctx, topic, numResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	log.Printf("Research Agent: Found %d search results", searchResp.Total)

	// Convert search results to our model format
	results := make([]models.SearchResult, 0, len(searchResp.Results))
	for i, result := range searchResp.Results {
		// Filter for reputable sources if requested
		if reputableOnly && !isReputableSource(result.DisplayLink) {
			log.Printf("Filtering out non-reputable source: %s", result.DisplayLink)
			continue
		}

		results = append(results, models.SearchResult{
			URL:      result.URL,
			Title:    result.Title,
			Snippet:  result.Snippet,
			Domain:   result.DisplayLink,
			Position: i + 1,
		})
	}

	log.Printf("Research Agent: Returning %d sources", len(results))
	return results, nil
}

// isReputableSource checks if a domain is from a reputable source
func isReputableSource(domain string) bool {
	reputableDomains := []string{
		".gov", ".edu", // Government and education
		"who.int", "un.org", "worldbank.org", // International orgs
		"pewresearch.org", "gallup.com", // Research organizations
		"nature.com", "science.org", "nejm.org", // Journals
	}

	domainLower := strings.ToLower(domain)
	for _, rep := range reputableDomains {
		if strings.Contains(domainLower, rep) {
			return true
		}
	}
	return false
}

// Research finds sources for a given topic (returns URLs, not statistics)
func (ra *ResearchAgent) Research(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	log.Printf("Research Agent: Finding sources for topic: %s", req.Topic)

	// Determine number of results to fetch
	numResults := req.MaxStatistics
	if numResults == 0 {
		numResults = 20 // Default
	}

	// Find sources
	searchResults, err := ra.findSources(ctx, req.Topic, numResults, req.ReputableOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to find sources: %w", err)
	}

	// Note: We now return SearchResults, which will be analyzed by Synthesis Agent
	// Convert to old format for backward compatibility (temporary)
	candidates := make([]models.CandidateStatistic, 0, len(searchResults))
	for i, result := range searchResults {
		if i >= req.MaxStatistics && req.MaxStatistics > 0 {
			break
		}
		// Placeholder candidate - real extraction happens in Synthesis Agent
		candidates = append(candidates, models.CandidateStatistic{
			Name:      fmt.Sprintf("Source from %s", result.Domain),
			Value:     0, // Will be extracted by Synthesis Agent
			Unit:      "",
			Source:    result.Domain,
			SourceURL: result.URL,
			Excerpt:   result.Snippet,
		})
	}

	response := &models.ResearchResponse{
		Topic:      req.Topic,
		Candidates: candidates,
		Timestamp:  time.Now(),
	}

	log.Printf("Research Agent: Found %d sources", len(searchResults))
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
	log.Println("Role: Find relevant sources via web search (no LLM)")
	log.Println("Next step: Synthesis Agent extracts statistics from these sources")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
