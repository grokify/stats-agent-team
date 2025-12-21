package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	// LLM Configuration
	LLMProvider string // "gemini", "claude", "openai", "ollama", "xai"
	LLMAPIKey   string
	LLMModel    string
	LLMBaseURL  string // For Ollama or custom endpoints

	// Provider-specific API keys
	GeminiAPIKey string
	ClaudeAPIKey string
	OpenAIAPIKey string
	XAIAPIKey    string
	OllamaURL    string

	// Search Configuration
	SearchProvider string // "serper", "serpapi"
	SerperAPIKey   string
	SerpAPIKey     string

	// Agent Configuration
	ResearchAgentURL     string
	SynthesisAgentURL    string
	VerificationAgentURL string
	OrchestratorURL      string
	OrchestratorEinoURL  string

	// A2A Protocol Configuration
	A2AEnabled   bool
	A2AAuthType  string // "jwt", "apikey", "oauth2"
	A2AAuthToken string

	// Observability Configuration
	ObservabilityEnabled  bool   // Enable LLM observability
	ObservabilityProvider string // "opik", "langfuse", "phoenix"
	ObservabilityAPIKey   string
	ObservabilityEndpoint string // Custom endpoint (optional)
	ObservabilityProject  string // Project name for grouping traces
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	provider := getEnv("LLM_PROVIDER", "gemini")

	cfg := &Config{
		// LLM settings
		LLMProvider: provider,
		LLMAPIKey:   getEnv("LLM_API_KEY", ""),
		LLMModel:    getEnv("LLM_MODEL", getDefaultModel(provider)),
		LLMBaseURL:  getEnv("LLM_BASE_URL", ""),

		// Provider-specific API keys
		GeminiAPIKey: getEnv("GEMINI_API_KEY", getEnv("GOOGLE_API_KEY", "")),
		ClaudeAPIKey: getEnv("CLAUDE_API_KEY", getEnv("ANTHROPIC_API_KEY", "")),
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		XAIAPIKey:    getEnv("XAI_API_KEY", ""),
		OllamaURL:    getEnv("OLLAMA_URL", "http://localhost:11434"),

		// Search settings
		SearchProvider: getEnv("SEARCH_PROVIDER", "serper"),
		SerperAPIKey:   getEnv("SERPER_API_KEY", ""),
		SerpAPIKey:     getEnv("SERPAPI_API_KEY", ""),

		// Agent URLs
		ResearchAgentURL:     getEnv("RESEARCH_AGENT_URL", "http://localhost:8001"),
		SynthesisAgentURL:    getEnv("SYNTHESIS_AGENT_URL", "http://localhost:8004"),
		VerificationAgentURL: getEnv("VERIFICATION_AGENT_URL", "http://localhost:8002"),
		OrchestratorURL:      getEnv("ORCHESTRATOR_URL", "http://localhost:8000"),
		OrchestratorEinoURL:  getEnv("ORCHESTRATOR_EINO_URL", "http://localhost:8000"), // Same port as ADK

		// A2A Protocol
		A2AEnabled:   getEnv("A2A_ENABLED", "true") == "true",
		A2AAuthType:  getEnv("A2A_AUTH_TYPE", "apikey"),
		A2AAuthToken: getEnv("A2A_AUTH_TOKEN", ""),

		// Observability
		ObservabilityEnabled:  getEnv("OBSERVABILITY_ENABLED", "false") == "true",
		ObservabilityProvider: getEnv("OBSERVABILITY_PROVIDER", "opik"),
		ObservabilityAPIKey:   getEnv("OBSERVABILITY_API_KEY", getEnv("OPIK_API_KEY", "")),
		ObservabilityEndpoint: getEnv("OBSERVABILITY_ENDPOINT", ""),
		ObservabilityProject:  getEnv("OBSERVABILITY_PROJECT", "stats-agent-team"),
	}

	// Set LLMAPIKey based on provider if not explicitly set
	if cfg.LLMAPIKey == "" {
		switch provider {
		case "gemini":
			cfg.LLMAPIKey = cfg.GeminiAPIKey
		case "claude":
			cfg.LLMAPIKey = cfg.ClaudeAPIKey
		case "openai":
			cfg.LLMAPIKey = cfg.OpenAIAPIKey
		case "xai":
			cfg.LLMAPIKey = cfg.XAIAPIKey
		}
	}

	// Set LLMBaseURL for Ollama if not explicitly set
	if cfg.LLMBaseURL == "" && provider == "ollama" {
		cfg.LLMBaseURL = cfg.OllamaURL
	}

	return cfg
}

// getDefaultModel returns the default model for a given provider
func getDefaultModel(provider string) string {
	switch provider {
	case "gemini":
		return "gemini-2.0-flash-exp"
	case "claude":
		return "claude-3-5-sonnet-20241022" // Latest Claude 3.5 Sonnet (Oct 2024)
	case "openai":
		return "gpt-4"
	case "xai":
		return "grok-3" // Latest stable Grok model
	case "ollama":
		return "llama3:latest"
	default:
		return "gemini-2.0-flash-exp"
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
