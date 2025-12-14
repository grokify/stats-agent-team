package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "search":
		handleSearch()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleSearch() {
	if len(os.Args) < 3 {
		fmt.Println("Error: topic required")
		fmt.Println("Usage: stats-agent search <topic> [options]")
		os.Exit(1)
	}

	topic := os.Args[2]

	// Parse optional flags
	minStats := 10
	maxCandidates := 30
	reputableOnly := true

	// TODO: Add flag parsing for customization

	cfg := config.LoadConfig()

	// Create orchestration request
	req := &models.OrchestrationRequest{
		Topic:            topic,
		MinVerifiedStats: minStats,
		MaxCandidates:    maxCandidates,
		ReputableOnly:    reputableOnly,
	}

	fmt.Printf("Searching for statistics about: %s\n", topic)
	fmt.Printf("Target: %d verified statistics from reputable sources\n\n", minStats)

	// Call orchestration agent
	resp, err := callOrchestrator(cfg, req)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	// Print results
	printResults(resp)
}

func callOrchestrator(cfg *config.Config, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/orchestrate", cfg.OrchestratorURL)

	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, httpResp.Status)
	}

	var resp models.OrchestrationResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

func printResults(resp *models.OrchestrationResponse) {
	fmt.Printf("=== Statistics Search Results ===\n\n")
	fmt.Printf("Topic: %s\n", resp.Topic)
	fmt.Printf("Found: %d verified statistics (from %d candidates)\n", resp.VerifiedCount, resp.TotalCandidates)
	fmt.Printf("Failed verification: %d\n", resp.FailedCount)
	fmt.Printf("Timestamp: %s\n\n", resp.Timestamp.Format("2006-01-02 15:04:05"))

	if len(resp.Statistics) == 0 {
		fmt.Println("No verified statistics found.")
		return
	}

	// Print as JSON
	fmt.Println("=== Verified Statistics (JSON) ===")
	fmt.Println()
	jsonData, err := json.MarshalIndent(resp.Statistics, "", "  ")
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	fmt.Println(string(jsonData))

	// Also print human-readable format
	fmt.Println()
	fmt.Println("=== Human-Readable Format ===")
	fmt.Println()
	for i, stat := range resp.Statistics {
		fmt.Printf("%d. %s\n", i+1, stat.Name)
		fmt.Printf("   Value: %v %s\n", stat.Value, stat.Unit)
		fmt.Printf("   Source: %s\n", stat.Source)
		fmt.Printf("   URL: %s\n", stat.SourceURL)
		fmt.Printf("   Excerpt: \"%s\"\n", stat.Excerpt)
		fmt.Printf("   Verified: ✓\n")
		fmt.Printf("   Date Found: %s\n\n", stat.DateFound.Format("2006-01-02"))
	}
}

func printUsage() {
	usage := `Statistics Agent - Multi-Agent System for Finding Verified Statistics

USAGE:
    stats-agent <command> [arguments]

COMMANDS:
    search <topic>     Search for verified statistics on a topic
    help               Show this help message

EXAMPLES:
    stats-agent search "climate change"
    stats-agent search "artificial intelligence adoption rates"
    stats-agent search "cybersecurity threats 2024"

ENVIRONMENT VARIABLES:
    LLM_PROVIDER           LLM provider (default: openai)
    LLM_API_KEY           API key for LLM provider
    LLM_MODEL             LLM model to use (default: gpt-4)
    SEARCH_PROVIDER       Search provider (default: google)
    SEARCH_API_KEY        API key for search provider
    ORCHESTRATOR_URL      Orchestrator agent URL (default: http://localhost:8000)
    A2A_ENABLED           Enable A2A protocol (default: true)

ARCHITECTURE:
    This system uses a 3-agent architecture:

    1. Research Agent (port 8001/9001)
       - Searches for statistics from web sources
       - Prioritizes reputable publishers
       - Extracts candidate statistics

    2. Verification Agent (port 8002/9002)
       - Validates statistics in their sources
       - Checks for exact excerpts and values
       - Flags hallucinations and mismatches

    3. Orchestration Agent (port 8000/9000)
       - Coordinates the workflow
       - Manages retry logic
       - Ensures quality standards

OUTPUT FORMAT:
    JSON array with fields:
    - name: Description of the statistic
    - value: Numerical value (number or percentage)
    - source: Name of source organization
    - source_url: URL to the source
    - excerpt: Verbatim quote containing the statistic
    - verified: Verification status (always true in results)
    - date_found: When the statistic was found
`
	fmt.Println(usage)
}
