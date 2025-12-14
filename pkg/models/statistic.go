package models

import "time"

// Statistic represents a verified statistic with its source
type Statistic struct {
	Name      string    `json:"name"`       // Name/description of the statistic
	Value     float32   `json:"value"`      // Numerical value
	Unit      string    `json:"unit"`       // Unit of measurement (e.g., "°C", "%", "million")
	Source    string    `json:"source"`     // Name of the source (e.g., "Pew Research Center")
	SourceURL string    `json:"source_url"` // URL to the source
	Excerpt   string    `json:"excerpt"`    // Verbatim quote containing the statistic
	Verified  bool      `json:"verified"`   // Whether this has been verified by verification agent
	DateFound time.Time `json:"date_found"` // When this statistic was found
}

// CandidateStatistic represents an unverified statistic from research
type CandidateStatistic struct {
	Name      string  `json:"name"`
	Value     float32 `json:"value"`
	Unit      string  `json:"unit"`
	Source    string  `json:"source"`
	SourceURL string  `json:"source_url"`
	Excerpt   string  `json:"excerpt"`
}

// VerificationResult represents the result of verifying a statistic
type VerificationResult struct {
	Statistic *Statistic `json:"statistic"`
	Verified  bool       `json:"verified"`
	Reason    string     `json:"reason,omitempty"` // Why verification failed (if applicable)
}

// ResearchRequest represents a request to find statistics
type ResearchRequest struct {
	Topic         string `json:"topic"`
	MinStatistics int    `json:"min_statistics"` // Minimum number of statistics to find
	MaxStatistics int    `json:"max_statistics"` // Maximum number of statistics to find
	ReputableOnly bool   `json:"reputable_only"` // Only search reputable sources
}

// ResearchResponse represents the response from research agent
type ResearchResponse struct {
	Topic      string               `json:"topic"`
	Candidates []CandidateStatistic `json:"candidates"`
	Timestamp  time.Time            `json:"timestamp"`
}

// VerificationRequest represents a request to verify statistics
type VerificationRequest struct {
	Candidates []CandidateStatistic `json:"candidates"`
}

// VerificationResponse represents the response from verification agent
type VerificationResponse struct {
	Results   []VerificationResult `json:"results"`
	Verified  int                  `json:"verified_count"`
	Failed    int                  `json:"failed_count"`
	Timestamp time.Time            `json:"timestamp"`
}

// OrchestrationRequest represents the main request to the orchestrator
type OrchestrationRequest struct {
	Topic            string `json:"topic"`
	MinVerifiedStats int    `json:"min_verified_stats"` // Minimum verified statistics required
	MaxCandidates    int    `json:"max_candidates"`     // Maximum candidates to research
	ReputableOnly    bool   `json:"reputable_only"`
}

// OrchestrationResponse represents the final response
type OrchestrationResponse struct {
	Topic           string      `json:"topic"`
	Statistics      []Statistic `json:"statistics"`
	TotalCandidates int         `json:"total_candidates"`
	VerifiedCount   int         `json:"verified_count"`
	FailedCount     int         `json:"failed_count"`
	Timestamp       time.Time   `json:"timestamp"`
}

// SearchResult represents a source URL from research agent
type SearchResult struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Snippet  string `json:"snippet"`
	Domain   string `json:"domain"`
	Position int    `json:"position,omitempty"`
}

// SynthesisRequest is the request to synthesis agent
type SynthesisRequest struct {
	Topic         string         `json:"topic"`
	SearchResults []SearchResult `json:"search_results"`
	MinStatistics int            `json:"min_statistics"`
	MaxStatistics int            `json:"max_statistics"`
}

// SynthesisResponse is the response from synthesis agent
type SynthesisResponse struct {
	Topic           string               `json:"topic"`
	Candidates      []CandidateStatistic `json:"candidates"`
	SourcesAnalyzed int                  `json:"sources_analyzed"`
	Timestamp       time.Time            `json:"timestamp"`
}
