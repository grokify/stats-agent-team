package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/direct"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// DirectAgent provides HTTP API for direct LLM search
type DirectAgent struct {
	cfg       *config.Config
	directSvc *direct.LLMSearchService
}

// NewDirectAgent creates a new direct search agent
func NewDirectAgent(cfg *config.Config) (*DirectAgent, error) {
	directSvc, err := direct.NewLLMSearchService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct search service: %w", err)
	}

	return &DirectAgent{
		cfg:       cfg,
		directSvc: directSvc,
	}, nil
}

// DirectSearchInput represents the input for direct search
type DirectSearchInput struct {
	Body struct {
		Topic         string `json:"topic" minLength:"1" maxLength:"500" example:"climate change" doc:"Topic to search for statistics"`
		MinStats      int    `json:"min_stats,omitempty" minimum:"1" maximum:"100" default:"10" example:"10" doc:"Minimum number of statistics to find"`
		VerifyWithWeb bool   `json:"verify_with_web,omitempty" default:"false" example:"false" doc:"If true, verifies LLM claims with verification agent (requires verification agent running on port 8002)"`
	}
}

// DirectSearchOutput represents the output from direct search
type DirectSearchOutput struct {
	Body *models.OrchestrationResponse
}

// ErrorOutput represents an error response
type ErrorOutput struct {
	Body struct {
		Error   string `json:"error" example:"Invalid topic" doc:"Error message"`
		Message string `json:"message" example:"Topic must be at least 1 character" doc:"Detailed error message"`
	}
}

func main() {
	cfg := config.LoadConfig()

	directAgent, err := NewDirectAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create direct agent: %v", err)
	}

	// Create Chi router
	router := chi.NewMux()

	// Create Huma API
	api := humachi.New(router, huma.DefaultConfig("Statistics Direct Search API", "1.0.0"))

	// Configure API metadata
	api.OpenAPI().Info.Description = `Direct LLM-based statistics search service.

This service provides two modes:
1. **Direct Mode** (verify_with_web: false): Fast LLM search that returns statistics with source URLs
2. **Hybrid Mode** (verify_with_web: true): LLM search + web verification for accuracy

The service uses server-side LLM configuration, so clients don't need API keys.`

	api.OpenAPI().Info.Contact = &huma.Contact{
		Name: "Stats Agent Team",
		URL:  "https://github.com/grokify/stats-agent-team",
	}

	// Add server information
	api.OpenAPI().Servers = []*huma.Server{
		{URL: "http://localhost:8005", Description: "Local development server"},
	}

	// Register the search operation
	huma.Register(api, huma.Operation{
		OperationID:   "search-statistics",
		Method:        http.MethodPost,
		Path:          "/search",
		Summary:       "Search for statistics on a topic",
		Description:   "Performs direct LLM search for statistics, optionally verifying claims with web scraping",
		Tags:          []string{"Statistics"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *DirectSearchInput) (*DirectSearchOutput, error) {
		// Set defaults
		minStats := input.Body.MinStats
		if minStats == 0 {
			minStats = 10
		}

		log.Printf("[Direct Agent] Processing request for topic '%s' (min_stats: %d, verify: %v)",
			input.Body.Topic, minStats, input.Body.VerifyWithWeb)

		// Call direct search service
		resp, err := directAgent.directSvc.SearchStatisticsWithVerification(
			ctx,
			input.Body.Topic,
			minStats,
			input.Body.VerifyWithWeb,
		)
		if err != nil {
			log.Printf("[Direct Agent] Search failed: %v", err)
			return nil, huma.Error500InternalServerError(fmt.Sprintf("Search failed: %v", err))
		}

		log.Printf("[Direct Agent] Search completed: %d verified statistics (partial: %v)",
			resp.VerifiedCount, resp.Partial)

		return &DirectSearchOutput{Body: resp}, nil
	})

	// Add health check endpoint
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check endpoint",
		Description: "Returns OK if the service is healthy",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*struct {
		Body struct {
			Status string `json:"status" example:"OK" doc:"Service status"`
		}
	}, error) {
		return &struct {
			Body struct {
				Status string `json:"status" example:"OK" doc:"Service status"`
			}
		}{
			Body: struct {
				Status string `json:"status" example:"OK" doc:"Service status"`
			}{
				Status: "OK",
			},
		}, nil
	})

	log.Println("===========================================")
	log.Println("Direct Agent HTTP server starting on :8005")
	log.Println("===========================================")
	log.Printf("LLM Provider: %s", cfg.LLMProvider)
	log.Printf("LLM Model: %s", cfg.LLMModel)
	log.Println()
	log.Println("Endpoints:")
	log.Println("  POST /search         - Direct LLM search (optionally with verification)")
	log.Println("  GET  /health         - Health check")
	log.Println("  GET  /docs           - OpenAPI documentation (Swagger UI)")
	log.Println("  GET  /openapi.json   - OpenAPI 3.1 specification")
	log.Println("  GET  /openapi.yaml   - OpenAPI 3.1 specification (YAML)")
	log.Println()
	log.Println("Documentation available at: http://localhost:8005/docs")
	log.Println("===========================================")

	if err := http.ListenAndServe(":8005", router); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
