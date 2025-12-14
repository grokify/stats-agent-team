package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/compose"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/httpclient"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// EinoOrchestrationAgent uses Eino framework for deterministic orchestration
type EinoOrchestrationAgent struct {
	cfg    *config.Config
	client *http.Client
	graph  *compose.Graph[*models.OrchestrationRequest, *models.OrchestrationResponse]
}

// NewEinoOrchestrationAgent creates a new Eino-based orchestration agent
func NewEinoOrchestrationAgent(cfg *config.Config) *EinoOrchestrationAgent {
	oa := &EinoOrchestrationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 60 * time.Second},
	}

	// Build the deterministic workflow graph
	oa.graph = oa.buildWorkflowGraph()

	return oa
}

// buildWorkflowGraph creates a deterministic Eino graph for the workflow
func (oa *EinoOrchestrationAgent) buildWorkflowGraph() *compose.Graph[*models.OrchestrationRequest, *models.OrchestrationResponse] {
	// Create a new graph with typed input/output
	g := compose.NewGraph[*models.OrchestrationRequest, *models.OrchestrationResponse]()

	// Node names
	const (
		nodeValidateInput  = "validate_input"
		nodeResearch       = "research"
		nodeSynthesis      = "synthesis"
		nodeVerification   = "verification"
		nodeCheckQuality   = "check_quality"
		nodeRetryResearch  = "retry_research"
		nodeFormatResponse = "format_response"
	)

	// Add Lambda nodes for each step in the workflow

	// 1. Validate Input Node
	validateInputLambda := compose.InvokableLambda(func(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationRequest, error) {
		log.Printf("[Eino] Validating input for topic: %s", req.Topic)

		// Set defaults
		if req.MinVerifiedStats == 0 {
			req.MinVerifiedStats = 10
		}
		if req.MaxCandidates == 0 {
			req.MaxCandidates = 30
		}

		return req, nil
	})
	if err := g.AddLambdaNode(nodeValidateInput, validateInputLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add validate input node: %v", err)
	}

	// 2. Research Node - calls research agent to find sources (URLs)
	researchLambda := compose.InvokableLambda(func(ctx context.Context, req *models.OrchestrationRequest) (*ResearchState, error) {
		log.Printf("[Eino] Executing research for topic: %s", req.Topic)

		researchReq := &models.ResearchRequest{
			Topic:         req.Topic,
			MinStatistics: req.MinVerifiedStats,
			MaxStatistics: req.MaxCandidates,
			ReputableOnly: req.ReputableOnly,
		}

		resp, err := oa.callResearchAgent(ctx, researchReq)
		if err != nil {
			return nil, fmt.Errorf("research failed: %w", err)
		}

		// Convert candidates to search results
		searchResults := make([]models.SearchResult, 0, len(resp.Candidates))
		for _, cand := range resp.Candidates {
			searchResults = append(searchResults, models.SearchResult{
				URL:     cand.SourceURL,
				Title:   cand.Name,
				Snippet: cand.Excerpt,
				Domain:  cand.Source,
			})
		}

		log.Printf("[Eino] Research found %d sources", len(searchResults))

		return &ResearchState{
			Request:       req,
			SearchResults: searchResults,
		}, nil
	})
	if err := g.AddLambdaNode(nodeResearch, researchLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add research node: %v", err)
	}

	// 3. Synthesis Node - calls synthesis agent to extract statistics
	synthesisLambda := compose.InvokableLambda(func(ctx context.Context, state *ResearchState) (*SynthesisState, error) {
		log.Printf("[Eino] Synthesizing statistics from %d sources", len(state.SearchResults))

		synthesisReq := &models.SynthesisRequest{
			Topic:         state.Request.Topic,
			SearchResults: state.SearchResults,
			MinStatistics: state.Request.MinVerifiedStats,
			MaxStatistics: state.Request.MaxCandidates,
		}

		resp, err := oa.callSynthesisAgent(ctx, synthesisReq)
		if err != nil {
			return nil, fmt.Errorf("synthesis failed: %w", err)
		}

		log.Printf("[Eino] Synthesis extracted %d candidate statistics", len(resp.Candidates))

		return &SynthesisState{
			Request:       state.Request,
			SearchResults: state.SearchResults,
			Candidates:    resp.Candidates,
		}, nil
	})
	if err := g.AddLambdaNode(nodeSynthesis, synthesisLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add synthesis node: %v", err)
	}

	// 4. Verification Node - calls verification agent
	verificationLambda := compose.InvokableLambda(func(ctx context.Context, state *SynthesisState) (*VerificationState, error) {
		log.Printf("[Eino] Verifying %d candidates", len(state.Candidates))

		verifyReq := &models.VerificationRequest{
			Candidates: state.Candidates,
		}

		resp, err := oa.callVerificationAgent(ctx, verifyReq)
		if err != nil {
			return nil, fmt.Errorf("verification failed: %w", err)
		}

		// Extract verified statistics
		var verifiedStats []models.Statistic
		for _, result := range resp.Results {
			if result.Verified {
				verifiedStats = append(verifiedStats, *result.Statistic)
			}
		}

		return &VerificationState{
			Request:       state.Request,
			AllCandidates: state.Candidates,
			Verified:      verifiedStats,
			Failed:        resp.Failed,
		}, nil
	})
	if err := g.AddLambdaNode(nodeVerification, verificationLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add verification node: %v", err)
	}

	// 5. Quality Check Node - deterministic decision
	qualityCheckLambda := compose.InvokableLambda(func(ctx context.Context, state *VerificationState) (*QualityDecision, error) {
		verified := len(state.Verified)
		target := state.Request.MinVerifiedStats

		log.Printf("[Eino] Quality check: %d verified (target: %d)", verified, target)

		decision := &QualityDecision{
			State:     state,
			NeedMore:  verified < target,
			Shortfall: target - verified,
		}

		if decision.NeedMore {
			log.Printf("[Eino] Need %d more verified statistics", decision.Shortfall)
		} else {
			log.Printf("[Eino] Quality target met")
		}

		return decision, nil
	})
	if err := g.AddLambdaNode(nodeCheckQuality, qualityCheckLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add quality check node: %v", err)
	}

	// 6. Retry Research Node (if needed) - NOT IMPLEMENTED YET in 4-agent architecture
	// TODO: Implement retry logic for 4-agent workflow
	retryResearchLambda := compose.InvokableLambda(func(ctx context.Context, decision *QualityDecision) (*VerificationState, error) {
		if !decision.NeedMore {
			// No retry needed, return existing state
			return decision.State, nil
		}

		log.Printf("[Eino] Retry logic not yet implemented for 4-agent architecture")
		log.Printf("[Eino] Would retry for %d more candidates", decision.Shortfall)

		// For now, just return the existing state
		// TODO: Implement: Research → Synthesis → Verification loop
		return decision.State, nil
	})
	if err := g.AddLambdaNode(nodeRetryResearch, retryResearchLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add retry research node: %v", err)
	}

	// 7. Format Response Node
	formatResponseLambda := compose.InvokableLambda(func(ctx context.Context, state *VerificationState) (*models.OrchestrationResponse, error) {
		log.Printf("[Eino] Formatting response with %d verified statistics", len(state.Verified))

		return &models.OrchestrationResponse{
			Topic:           state.Request.Topic,
			Statistics:      state.Verified,
			TotalCandidates: len(state.AllCandidates),
			VerifiedCount:   len(state.Verified),
			FailedCount:     state.Failed,
			Timestamp:       time.Now(),
		}, nil
	})
	if err := g.AddLambdaNode(nodeFormatResponse, formatResponseLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add format response node: %v", err)
	}

	// Add edges to define the workflow
	_ = g.AddEdge(compose.START, nodeValidateInput)
	_ = g.AddEdge(nodeValidateInput, nodeResearch)
	_ = g.AddEdge(nodeResearch, nodeSynthesis)      // NEW: Research → Synthesis
	_ = g.AddEdge(nodeSynthesis, nodeVerification)  // NEW: Synthesis → Verification
	_ = g.AddEdge(nodeVerification, nodeCheckQuality)

	// Conditional branching based on quality check
	_ = g.AddEdge(nodeCheckQuality, nodeRetryResearch)
	_ = g.AddEdge(nodeRetryResearch, nodeFormatResponse)
	_ = g.AddEdge(nodeFormatResponse, compose.END)

	log.Printf("[Eino] Workflow graph built: ValidateInput → Research → Synthesis → Verification → QualityCheck → Format")

	return g
}

// Orchestrate executes the deterministic Eino workflow
func (oa *EinoOrchestrationAgent) Orchestrate(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	log.Printf("[Eino Orchestrator] Starting deterministic workflow for topic: %s", req.Topic)

	// Compile the graph
	compiledGraph, err := oa.graph.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %w", err)
	}

	// Execute the graph
	result, err := compiledGraph.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("[Eino Orchestrator] Workflow completed successfully")
	return result, nil
}

// Helper methods to call research and verification agents

func (oa *EinoOrchestrationAgent) callResearchAgent(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	var resp models.ResearchResponse
	url := fmt.Sprintf("%s/research", oa.cfg.ResearchAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (oa *EinoOrchestrationAgent) callSynthesisAgent(ctx context.Context, req *models.SynthesisRequest) (*models.SynthesisResponse, error) {
	var resp models.SynthesisResponse
	url := fmt.Sprintf("%s/synthesize", oa.cfg.SynthesisAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (oa *EinoOrchestrationAgent) callVerificationAgent(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	var resp models.VerificationResponse
	url := fmt.Sprintf("%s/verify", oa.cfg.VerificationAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// HTTP Handler
func (oa *EinoOrchestrationAgent) HandleOrchestrationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.OrchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
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

// State types for the workflow
type ResearchState struct {
	Request       *models.OrchestrationRequest
	SearchResults []models.SearchResult
}

type SynthesisState struct {
	Request       *models.OrchestrationRequest
	SearchResults []models.SearchResult
	Candidates    []models.CandidateStatistic
}

type VerificationState struct {
	Request       *models.OrchestrationRequest
	AllCandidates []models.CandidateStatistic
	Verified      []models.Statistic
	Failed        int
}

type QualityDecision struct {
	State     *VerificationState
	NeedMore  bool
	Shortfall int
}
