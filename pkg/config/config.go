package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	// LLM Configuration
	LLMProvider string
	LLMAPIKey   string
	LLMModel    string

	// Search Configuration
	SearchProvider string
	SearchAPIKey   string

	// Agent Configuration
	ResearchAgentURL       string
	VerificationAgentURL   string
	OrchestratorURL        string
	OrchestratorEinoURL    string

	// A2A Protocol Configuration
	A2AEnabled bool
	A2AAuthType string // "jwt", "apikey", "oauth2"
	A2AAuthToken string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		// LLM settings (default to OpenAI)
		LLMProvider: getEnv("LLM_PROVIDER", "openai"),
		LLMAPIKey:   getEnv("LLM_API_KEY", ""),
		LLMModel:    getEnv("LLM_MODEL", "gpt-4"),

		// Search settings
		SearchProvider: getEnv("SEARCH_PROVIDER", "google"),
		SearchAPIKey:   getEnv("SEARCH_API_KEY", ""),

		// Agent URLs
		ResearchAgentURL:     getEnv("RESEARCH_AGENT_URL", "http://localhost:8001"),
		VerificationAgentURL: getEnv("VERIFICATION_AGENT_URL", "http://localhost:8002"),
		OrchestratorURL:      getEnv("ORCHESTRATOR_URL", "http://localhost:8000"),
		OrchestratorEinoURL:  getEnv("ORCHESTRATOR_EINO_URL", "http://localhost:8003"),

		// A2A Protocol
		A2AEnabled:   getEnv("A2A_ENABLED", "true") == "true",
		A2AAuthType:  getEnv("A2A_AUTH_TYPE", "apikey"),
		A2AAuthToken: getEnv("A2A_AUTH_TOKEN", ""),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
