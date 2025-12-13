package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"

	"github.com/grokify/stats-agent/pkg/config"
	"github.com/grokify/stats-agent/pkg/models"
)

// VerificationAgent uses ADK for validating statistics
type VerificationAgent struct {
	cfg      *config.Config
	client   *http.Client
	adkAgent agent.Agent
}

// VerificationInput defines input for verification tool
type VerificationInput struct {
	Candidates []models.CandidateStatistic `json:"candidates" jsonschema:"description=Statistics candidates to verify"`
}

// VerificationToolOutput defines output from verification tool
type VerificationToolOutput struct {
	Results []models.VerificationResult `json:"results"`
}

// NewVerificationAgent creates a new ADK-based verification agent
func NewVerificationAgent(cfg *config.Config) (*VerificationAgent, error) {
	ctx := context.Background()

	// Create Gemini model
	model, err := gemini.NewModel(ctx, "gemini-2.0-flash-exp", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	va := &VerificationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}

	// Create verification tool
	verifyTool, err := functiontool.New(functiontool.Config{
		Name:        "verify_statistics",
		Description: "Verifies that statistics actually exist in their claimed sources by fetching and checking URLs",
	}, va.verifyToolHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification tool: %w", err)
	}

	// Create ADK agent
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "statistics_verification_agent",
		Model:       model,
		Description: "Verifies that statistics actually exist in their claimed sources",
		Instruction: `You are a statistics verification agent. Your job is to:
1. Fetch the content from the provided source URL
2. Search for the verbatim excerpt in the source content
3. Verify the numerical value matches exactly
4. Check if the source is reputable
5. Flag any discrepancies, hallucinations, or mismatches

Verification criteria:
- The exact excerpt must be present in the source
- The numerical value must match (allowing for reasonable formatting differences)
- The source must be accessible and legitimate
- The context must support the claimed statistic`,
		Tools: []tool.Tool{verifyTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	va.adkAgent = adkAgent

	return va, nil
}

// verifyToolHandler implements the verification logic
func (va *VerificationAgent) verifyToolHandler(ctx tool.Context, input VerificationInput) (VerificationToolOutput, error) {
	log.Printf("Verification Agent: Verifying %d candidates", len(input.Candidates))

	results := make([]models.VerificationResult, 0, len(input.Candidates))

	for _, candidate := range input.Candidates {
		result := va.verifyStatistic(ctx, candidate)
		results = append(results, result)
	}

	return VerificationToolOutput{
		Results: results,
	}, nil
}

// verifyStatistic verifies a single candidate
func (va *VerificationAgent) verifyStatistic(ctx context.Context, candidate models.CandidateStatistic) models.VerificationResult {
	log.Printf("Verification Agent: Verifying statistic from %s", candidate.SourceURL)

	// Fetch source content
	sourceContent, err := va.fetchSourceContent(ctx, candidate.SourceURL)
	if err != nil {
		log.Printf("Failed to fetch source: %v", err)
		return models.VerificationResult{
			Statistic: &models.Statistic{
				Name:      candidate.Name,
				Value:     candidate.Value,
				Source:    candidate.Source,
				SourceURL: candidate.SourceURL,
				Excerpt:   candidate.Excerpt,
				Verified:  false,
				DateFound: time.Now(),
			},
			Verified: false,
			Reason:   fmt.Sprintf("Failed to fetch source: %v", err),
		}
	}

	// Simple verification: check if excerpt appears in source
	verified := strings.Contains(sourceContent, candidate.Excerpt)
	reason := ""
	if !verified {
		reason = "Excerpt not found in source content"
	}

	stat := &models.Statistic{
		Name:      candidate.Name,
		Value:     candidate.Value,
		Source:    candidate.Source,
		SourceURL: candidate.SourceURL,
		Excerpt:   candidate.Excerpt,
		Verified:  verified,
		DateFound: time.Now(),
	}

	return models.VerificationResult{
		Statistic: stat,
		Verified:  verified,
		Reason:    reason,
	}
}

// fetchSourceContent fetches content from a URL
func (va *VerificationAgent) fetchSourceContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "StatisticsVerificationAgent/1.0")

	resp, err := va.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// Verify processes a verification request
func (va *VerificationAgent) Verify(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	log.Printf("Verification Agent: Verifying %d candidates", len(req.Candidates))

	results := make([]models.VerificationResult, 0, len(req.Candidates))
	verifiedCount := 0
	failedCount := 0

	for _, candidate := range req.Candidates {
		result := va.verifyStatistic(ctx, candidate)
		results = append(results, result)

		if result.Verified {
			verifiedCount++
		} else {
			failedCount++
		}
	}

	response := &models.VerificationResponse{
		Results:   results,
		Verified:  verifiedCount,
		Failed:    failedCount,
		Timestamp: time.Now(),
	}

	log.Printf("Verification Agent: %d verified, %d failed", verifiedCount, failedCount)
	return response, nil
}

// HandleVerificationRequest is the HTTP handler
func (va *VerificationAgent) HandleVerificationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.VerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := va.Verify(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Verification failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	cfg := config.LoadConfig()

	verificationAgent, err := NewVerificationAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create verification agent: %v", err)
	}

	// Start HTTP server
	http.HandleFunc("/verify", verificationAgent.HandleVerificationRequest)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Println("Verification Agent HTTP server starting on :8002")
	log.Println("(ADK agent initialized for future A2A integration)")
	if err := http.ListenAndServe(":8002", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
