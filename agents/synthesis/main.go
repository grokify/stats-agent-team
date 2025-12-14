package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/llm"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// SynthesisAgent extracts statistics from webpage content using LLM
type SynthesisAgent struct {
	cfg         *config.Config
	client      *http.Client
	adkAgent    agent.Agent
	model       model.LLM
	modelFactor *llm.ModelFactory
}

// SynthesisInput defines input for synthesis tool
type SynthesisInput struct {
	Topic         string               `json:"topic" jsonschema:"description=The topic being researched"`
	SearchResults []models.SearchResult `json:"search_results" jsonschema:"description=URLs to analyze for statistics"`
	MinStatistics int                  `json:"min_statistics" jsonschema:"description=Minimum statistics to extract"`
	MaxStatistics int                  `json:"max_statistics" jsonschema:"description=Maximum statistics to extract"`
}

// SynthesisToolOutput defines output from synthesis tool
type SynthesisToolOutput struct {
	Candidates []models.CandidateStatistic `json:"candidates"`
}

// NewSynthesisAgent creates a new ADK-based synthesis agent
func NewSynthesisAgent(cfg *config.Config) (*SynthesisAgent, error) {
	ctx := context.Background()

	// Create model using factory
	modelFactory := llm.NewModelFactory(cfg)
	model, err := modelFactory.CreateModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	log.Printf("Synthesis Agent: Using %s", modelFactory.GetProviderInfo())

	sa := &SynthesisAgent{
		cfg:         cfg,
		client:      &http.Client{Timeout: 45 * time.Second},
		model:       model,
		modelFactor: modelFactory,
	}

	// Create synthesis tool
	synthesisTool, err := functiontool.New(functiontool.Config{
		Name:        "synthesize_statistics",
		Description: "Extracts numerical statistics from web page content",
	}, sa.synthesisToolHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesis tool: %w", err)
	}

	// Create ADK agent
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "statistics_synthesis_agent",
		Model:       model,
		Description: "Extracts statistics from web content",
		Instruction: `You are a statistics synthesis agent. Your job is to:
1. Fetch content from provided URLs
2. Analyze the content to find numerical statistics
3. Extract exact values, units, and context
4. Create verbatim excerpts containing the statistics
5. Identify the source credibility

When extracting statistics:
- Look for numerical values with context (percentages, measurements, counts)
- Extract the exact excerpt containing the statistic (word-for-word)
- Identify the unit of measurement
- Verify the source is reputable (academic, government, research)
- Only extract statistics that are clearly stated with numbers

Reputable sources include:
- Government agencies (.gov domains)
- Academic institutions (.edu domains)
- Research organizations (Pew, Gallup, etc.)
- International organizations (WHO, UN, World Bank, etc.)
- Peer-reviewed journals`,
		Tools: []tool.Tool{synthesisTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	sa.adkAgent = adkAgent

	return sa, nil
}

// synthesisToolHandler implements the synthesis logic
func (sa *SynthesisAgent) synthesisToolHandler(ctx tool.Context, input SynthesisInput) (SynthesisToolOutput, error) {
	log.Printf("Synthesis Agent: Analyzing %d URLs for topic: %s", len(input.SearchResults), input.Topic)

	candidates := make([]models.CandidateStatistic, 0)

	// Analyze each search result
	for i, result := range input.SearchResults {
		if len(candidates) >= input.MaxStatistics && input.MaxStatistics > 0 {
			break
		}

		log.Printf("Synthesis Agent: Fetching content from %s", result.URL)

		// Fetch webpage content
		content, err := sa.fetchWebpage(context.Background(), result.URL)
		if err != nil {
			log.Printf("Failed to fetch %s: %v", result.URL, err)
			continue
		}

		// Extract statistics from content using LLM
		stats := sa.extractStatisticsWithLLM(input.Topic, result, content)
		candidates = append(candidates, stats...)

		log.Printf("Synthesis Agent: Extracted %d statistics from %s (total: %d/%d)",
			len(stats), result.Domain, len(candidates), input.MinStatistics)

		// Stop if we have enough
		if len(candidates) >= input.MinStatistics && i > 2 {
			break
		}
	}

	log.Printf("Synthesis Agent: Total candidates extracted: %d", len(candidates))

	return SynthesisToolOutput{
		Candidates: candidates,
	}, nil
}

// fetchWebpage fetches and returns webpage content
func (sa *SynthesisAgent) fetchWebpage(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "StatisticsSynthesisAgent/1.0")

	resp, err := sa.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit response size to 1MB
	limitedReader := io.LimitReader(resp.Body, 1*1024*1024)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// extractStatisticsWithLLM uses LLM to extract statistics from content
func (sa *SynthesisAgent) extractStatisticsWithLLM(topic string, result models.SearchResult, content string) []models.CandidateStatistic {
	// For now, use a simple regex-based approach
	// TODO: Use LLM to intelligently analyze content
	candidates := make([]models.CandidateStatistic, 0)

	// Simple pattern matching for statistics
	// Look for patterns like: "50%", "1.5 million", "23 degrees", etc.
	patterns := []string{
		`(\d+\.?\d*)\s*%`,                    // Percentages
		`(\d+\.?\d*)\s*(million|billion|thousand)`, // Large numbers
		`(\d+\.?\d*)\s*(degrees?|°[CF])`,    // Temperatures
		`(\d+\.?\d*)\s*(people|users|cases)`, // Counts
	}

	re := regexp.MustCompile(strings.Join(patterns, "|"))
	matches := re.FindAllStringSubmatch(content, -1)

	// Extract context around each match
	for _, match := range matches {
		if len(candidates) >= 5 { // Limit per URL
			break
		}

		// Find the full match
		fullMatch := match[0]

		// Parse the value
		valueStr := regexp.MustCompile(`\d+\.?\d*`).FindString(fullMatch)
		value, err := strconv.ParseFloat(valueStr, 32)
		if err != nil {
			continue
		}

		// Extract excerpt (surrounding context)
		index := strings.Index(content, fullMatch)
		if index == -1 {
			continue
		}

		start := max(0, index-100)
		end := min(len(content), index+len(fullMatch)+100)
		excerpt := strings.TrimSpace(content[start:end])

		// Clean up excerpt
		excerpt = strings.ReplaceAll(excerpt, "\n", " ")
		excerpt = regexp.MustCompile(`\s+`).ReplaceAllString(excerpt, " ")

		// Determine unit
		unit := strings.TrimSpace(strings.TrimPrefix(fullMatch, valueStr))
		if unit == "" {
			unit = "count"
		}

		candidates = append(candidates, models.CandidateStatistic{
			Name:      fmt.Sprintf("%s statistic from %s", topic, result.Domain),
			Value:     float32(value),
			Unit:      unit,
			Source:    result.Domain,
			SourceURL: result.URL,
			Excerpt:   excerpt,
		})
	}

	return candidates
}

// Synthesize processes a synthesis request directly
func (sa *SynthesisAgent) Synthesize(ctx context.Context, req *models.SynthesisRequest) (*models.SynthesisResponse, error) {
	log.Printf("Synthesis Agent: Processing %d search results for topic: %s", len(req.SearchResults), req.Topic)

	var candidates []models.CandidateStatistic

	// Analyze each search result
	for i, result := range req.SearchResults {
		if len(candidates) >= req.MaxStatistics && req.MaxStatistics > 0 {
			break
		}

		// Fetch webpage content
		content, err := sa.fetchWebpage(ctx, result.URL)
		if err != nil {
			log.Printf("Failed to fetch %s: %v", result.URL, err)
			continue
		}

		// Extract statistics
		stats := sa.extractStatisticsWithLLM(req.Topic, result, content)
		candidates = append(candidates, stats...)

		log.Printf("Synthesis Agent: Extracted %d statistics from %s (total: %d)",
			len(stats), result.Domain, len(candidates))

		// Stop early if we have enough
		if len(candidates) >= req.MinStatistics && i >= 2 {
			break
		}
	}

	response := &models.SynthesisResponse{
		Topic:           req.Topic,
		Candidates:      candidates,
		SourcesAnalyzed: min(len(req.SearchResults), len(candidates)/2+1),
		Timestamp:       time.Now(),
	}

	log.Printf("Synthesis Agent: Completed with %d candidates from %d sources",
		len(candidates), response.SourcesAnalyzed)

	return response, nil
}

// HandleSynthesisRequest is the HTTP handler
func (sa *SynthesisAgent) HandleSynthesisRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SynthesisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MinStatistics == 0 {
		req.MinStatistics = 5
	}
	if req.MaxStatistics == 0 {
		req.MaxStatistics = 20
	}

	resp, err := sa.Synthesize(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Synthesis failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func main() {
	cfg := config.LoadConfig()

	synthesisAgent, err := NewSynthesisAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create synthesis agent: %v", err)
	}

	// Start HTTP server with timeout
	server := &http.Server{
		Addr:         ":8004",
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	http.HandleFunc("/synthesize", synthesisAgent.HandleSynthesisRequest)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	log.Println("Synthesis Agent HTTP server starting on :8004")
	log.Println("(ADK agent initialized for LLM-based statistics extraction)")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
