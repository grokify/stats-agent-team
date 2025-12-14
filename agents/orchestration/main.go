package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/httpclient"
	"github.com/grokify/stats-agent-team/pkg/llm"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// OrchestrationAgent uses ADK to coordinate research and verification agents
type OrchestrationAgent struct {
	cfg      *config.Config
	client   *http.Client
	adkAgent agent.Agent
}

// OrchestrationInput defines input for orchestration tool
type OrchestrationInput struct {
	Topic            string `json:"topic" jsonschema:"description=The topic to research statistics for"`
	MinVerifiedStats int    `json:"min_verified_stats" jsonschema:"description=Minimum number of verified statistics required"`
	MaxCandidates    int    `json:"max_candidates" jsonschema:"description=Maximum number of candidate statistics to gather"`
	ReputableOnly    bool   `json:"reputable_only" jsonschema:"description=Only use reputable sources"`
}

// OrchestrationToolOutput defines output from orchestration tool
type OrchestrationToolOutput struct {
	Response *models.OrchestrationResponse `json:"response"`
}

// NewOrchestrationAgent creates a new ADK-based orchestration agent
func NewOrchestrationAgent(cfg *config.Config) (*OrchestrationAgent, error) {
	ctx := context.Background()

	// Create model using factory
	modelFactory := llm.NewModelFactory(cfg)
	model, err := modelFactory.CreateModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	log.Printf("Orchestration Agent: Using %s", modelFactory.GetProviderInfo())

	oa := &OrchestrationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 60 * time.Second},
	}

	// Create orchestration tool
	orchestrationTool, err := functiontool.New(functiontool.Config{
		Name:        "orchestrate_statistics_workflow",
		Description: "Coordinates research and verification agents to find verified statistics on a topic",
	}, oa.orchestrationToolHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestration tool: %w", err)
	}

	// Create ADK agent
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "statistics_orchestration_agent",
		Model:       model,
		Description: "Orchestrates multi-agent workflow to find and verify statistics",
		Instruction: `You are a statistics orchestration agent. Your job is to:
1. Coordinate the research agent to find candidate statistics
2. Send candidates to the verification agent for validation
3. Retry if needed to meet the target number of verified statistics
4. Return a final set of verified statistics with sources

Workflow:
- Request statistics from research agent based on topic
- Send candidates to verification agent
- Collect verified statistics
- If target not met and retries available, request more candidates
- Build final response with all verified statistics`,
		Tools: []tool.Tool{orchestrationTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	oa.adkAgent = adkAgent

	return oa, nil
}

// orchestrationToolHandler implements the orchestration logic
func (oa *OrchestrationAgent) orchestrationToolHandler(ctx tool.Context, input OrchestrationInput) (OrchestrationToolOutput, error) {
	log.Printf("Orchestration Agent: Starting orchestration for topic: %s", input.Topic)
	log.Printf("Target: %d verified statistics, max %d candidates", input.MinVerifiedStats, input.MaxCandidates)

	req := &models.OrchestrationRequest{
		Topic:            input.Topic,
		MinVerifiedStats: input.MinVerifiedStats,
		MaxCandidates:    input.MaxCandidates,
		ReputableOnly:    input.ReputableOnly,
	}

	// Use background context since tool.Context is different
	bgCtx := context.Background()
	response, err := oa.orchestrate(bgCtx, req)
	if err != nil {
		return OrchestrationToolOutput{}, fmt.Errorf("orchestration failed: %w", err)
	}

	return OrchestrationToolOutput{
		Response: response,
	}, nil
}

// orchestrate coordinates the workflow to find verified statistics
func (oa *OrchestrationAgent) orchestrate(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	var allCandidates []models.CandidateStatistic
	var verifiedStatistics []models.Statistic
	totalVerified := 0
	totalFailed := 0
	maxRetries := 3
	retry := 0

	for retry < maxRetries && totalVerified < req.MinVerifiedStats {
		// Calculate how many more candidates we need
		candidatesNeeded := req.MinVerifiedStats - totalVerified
		if candidatesNeeded < 5 {
			candidatesNeeded = 5 // Always request at least 5 for buffer
		}

		// Don't exceed max candidates
		candidatesLeft := req.MaxCandidates - len(allCandidates)
		if candidatesLeft <= 0 {
			log.Printf("Reached maximum candidates limit (%d)", req.MaxCandidates)
			break
		}
		if candidatesNeeded > candidatesLeft {
			candidatesNeeded = candidatesLeft
		}

		// Step 1: Request sources from research agent
		researchReq := &models.ResearchRequest{
			Topic:         req.Topic,
			MinStatistics: candidatesNeeded,
			MaxStatistics: candidatesNeeded + 5,
			ReputableOnly: req.ReputableOnly,
		}

		log.Printf("Orchestration: Requesting %d sources from research agent (attempt %d/%d)",
			candidatesNeeded, retry+1, maxRetries)

		researchResp, err := oa.callResearchAgent(ctx, researchReq)
		if err != nil {
			log.Printf("Research agent failed: %v", err)
			retry++
			continue
		}

		// Convert candidates to search results (research agent returns placeholder candidates now)
		searchResults := make([]models.SearchResult, 0, len(researchResp.Candidates))
		for _, cand := range researchResp.Candidates {
			searchResults = append(searchResults, models.SearchResult{
				URL:     cand.SourceURL,
				Title:   cand.Name,
				Snippet: cand.Excerpt,
				Domain:  cand.Source,
			})
		}

		log.Printf("Orchestration: Received %d sources from research agent", len(searchResults))

		// Step 2: Send sources to synthesis agent to extract statistics
		synthesisReq := &models.SynthesisRequest{
			Topic:         req.Topic,
			SearchResults: searchResults,
			MinStatistics: candidatesNeeded,
			MaxStatistics: candidatesNeeded + 5,
		}

		log.Printf("Orchestration: Sending %d sources to synthesis agent", len(searchResults))

		synthesisResp, err := oa.callSynthesisAgent(ctx, synthesisReq)
		if err != nil {
			log.Printf("Synthesis agent failed: %v", err)
			retry++
			continue
		}

		log.Printf("Orchestration: Synthesis extracted %d candidates", len(synthesisResp.Candidates))
		allCandidates = append(allCandidates, synthesisResp.Candidates...)

		// Step 3: Send candidates to verification agent
		verifyReq := &models.VerificationRequest{
			Candidates: synthesisResp.Candidates,
		}

		log.Printf("Orchestration: Sending %d candidates to verification agent", len(verifyReq.Candidates))

		verifyResp, err := oa.callVerificationAgent(ctx, verifyReq)
		if err != nil {
			log.Printf("Verification agent failed: %v", err)
			retry++
			continue
		}

		log.Printf("Orchestration: Verification complete - %d verified, %d failed",
			verifyResp.Verified, verifyResp.Failed)

		// Step 3: Collect verified statistics
		for _, result := range verifyResp.Results {
			if result.Verified {
				verifiedStatistics = append(verifiedStatistics, *result.Statistic)
				totalVerified++
			} else {
				totalFailed++
				log.Printf("Statistic failed verification: %s - %s", result.Statistic.Name, result.Reason)
			}
		}

		log.Printf("Orchestration: Current progress - %d/%d verified statistics",
			totalVerified, req.MinVerifiedStats)

		// Check if we have enough verified statistics
		if totalVerified >= req.MinVerifiedStats {
			log.Printf("Orchestration: Target reached with %d verified statistics", totalVerified)
			break
		}

		retry++
	}

	// Build final response
	response := &models.OrchestrationResponse{
		Topic:           req.Topic,
		Statistics:      verifiedStatistics,
		TotalCandidates: len(allCandidates),
		VerifiedCount:   totalVerified,
		FailedCount:     totalFailed,
		Timestamp:       time.Now(),
	}

	if totalVerified < req.MinVerifiedStats {
		log.Printf("Warning: Only found %d verified statistics (target: %d)",
			totalVerified, req.MinVerifiedStats)
	} else {
		log.Printf("Orchestration: Successfully completed with %d verified statistics", totalVerified)
	}

	return response, nil
}

// callResearchAgent calls the research agent via HTTP
func (oa *OrchestrationAgent) callResearchAgent(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	var resp models.ResearchResponse
	url := fmt.Sprintf("%s/research", oa.cfg.ResearchAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// callSynthesisAgent calls the synthesis agent via HTTP
func (oa *OrchestrationAgent) callSynthesisAgent(ctx context.Context, req *models.SynthesisRequest) (*models.SynthesisResponse, error) {
	var resp models.SynthesisResponse
	url := fmt.Sprintf("%s/synthesize", oa.cfg.SynthesisAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// callVerificationAgent calls the verification agent via HTTP
func (oa *OrchestrationAgent) callVerificationAgent(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	var resp models.VerificationResponse
	url := fmt.Sprintf("%s/verify", oa.cfg.VerificationAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Orchestrate is the public method for orchestrating the workflow
func (oa *OrchestrationAgent) Orchestrate(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	return oa.orchestrate(ctx, req)
}

// HandleOrchestrationRequest is the HTTP handler for orchestration requests
func (oa *OrchestrationAgent) HandleOrchestrationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.OrchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MinVerifiedStats == 0 {
		req.MinVerifiedStats = 10
	}
	if req.MaxCandidates == 0 {
		req.MaxCandidates = 30
	}

	resp, err := oa.Orchestrate(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Orchestration failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func main() {
	cfg := config.LoadConfig()

	orchestrationAgent, err := NewOrchestrationAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create orchestration agent: %v", err)
	}

	// Start HTTP server with timeout
	server := &http.Server{
		Addr:         ":8000",
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	http.HandleFunc("/orchestrate", orchestrationAgent.HandleOrchestrationRequest)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	log.Println("Orchestration Agent HTTP server starting on :8000")
	log.Println("(ADK agent initialized for future A2A integration)")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
