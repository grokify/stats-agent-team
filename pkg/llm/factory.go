package llm

import (
	"context"
	"fmt"

	"github.com/grokify/fluxllm"
	fluxllmhook "github.com/grokify/observai/integrations/fluxllm"
	"github.com/grokify/observai/llmops"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/llm/adapters"

	// Import observability providers (driver registration via init())
	_ "github.com/grokify/observai/llmops/langfuse"
	_ "github.com/grokify/observai/llmops/opik"
	_ "github.com/grokify/observai/llmops/phoenix"
)

// ModelFactory creates LLM models based on configuration
type ModelFactory struct {
	cfg      *config.Config
	obsHook  fluxllm.ObservabilityHook
	obsClose func() error
}

// NewModelFactory creates a new model factory
func NewModelFactory(cfg *config.Config) *ModelFactory {
	mf := &ModelFactory{cfg: cfg}

	// Initialize observability if enabled
	if cfg.ObservabilityEnabled && cfg.ObservabilityProvider != "" {
		hook, closeFn := mf.initObservability()
		mf.obsHook = hook
		mf.obsClose = closeFn
	}

	return mf
}

// initObservability initializes the observability provider and returns a hook
func (mf *ModelFactory) initObservability() (fluxllm.ObservabilityHook, func() error) {
	opts := []llmops.ClientOption{
		llmops.WithProjectName(mf.cfg.ObservabilityProject),
	}

	if mf.cfg.ObservabilityAPIKey != "" {
		opts = append(opts, llmops.WithAPIKey(mf.cfg.ObservabilityAPIKey))
	}

	if mf.cfg.ObservabilityEndpoint != "" {
		opts = append(opts, llmops.WithEndpoint(mf.cfg.ObservabilityEndpoint))
	}

	provider, err := llmops.Open(mf.cfg.ObservabilityProvider, opts...)
	if err != nil {
		// Log error but don't fail - observability is optional
		fmt.Printf("Warning: failed to initialize observability provider %s: %v\n", mf.cfg.ObservabilityProvider, err)
		return nil, nil
	}

	return fluxllmhook.NewHook(provider), provider.Close
}

// Close cleans up resources (call when factory is no longer needed)
func (mf *ModelFactory) Close() error {
	if mf.obsClose != nil {
		return mf.obsClose()
	}
	return nil
}

// CreateModel creates an LLM model based on the configured provider
func (mf *ModelFactory) CreateModel(ctx context.Context) (model.LLM, error) {
	switch mf.cfg.LLMProvider {
	case "gemini", "":
		return mf.createGeminiModel(ctx)
	case "claude":
		return mf.createClaudeModel()
	case "openai":
		return mf.createOpenAIModel()
	case "xai":
		return mf.createXAIModel()
	case "ollama":
		return mf.createOllamaModel()
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: gemini, claude, openai, xai, ollama)", mf.cfg.LLMProvider)
	}
}

// createGeminiModel creates a Gemini model
func (mf *ModelFactory) createGeminiModel(ctx context.Context) (model.LLM, error) {
	apiKey := mf.cfg.GeminiAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key not set - please set GOOGLE_API_KEY or GEMINI_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp"
	}

	return gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		APIKey: apiKey,
	})
}

// createClaudeModel creates a Claude model using FluxLLM
func (mf *ModelFactory) createClaudeModel() (model.LLM, error) {
	apiKey := mf.cfg.ClaudeAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("claude API key not set - please set CLAUDE_API_KEY or ANTHROPIC_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}

	return adapters.NewFluxLLMAdapterWithConfig(adapters.FluxLLMAdapterConfig{
		ProviderName:      "anthropic",
		APIKey:            apiKey,
		ModelName:         modelName,
		ObservabilityHook: mf.obsHook,
	})
}

// createOpenAIModel creates an OpenAI model using FluxLLM
func (mf *ModelFactory) createOpenAIModel() (model.LLM, error) {
	apiKey := mf.cfg.OpenAIAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("openai API key not set - please set OPENAI_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "gpt-4o-mini" // Use mini for cost efficiency
	}

	return adapters.NewFluxLLMAdapterWithConfig(adapters.FluxLLMAdapterConfig{
		ProviderName:      "openai",
		APIKey:            apiKey,
		ModelName:         modelName,
		ObservabilityHook: mf.obsHook,
	})
}

// createXAIModel creates an xAI Grok model using FluxLLM
func (mf *ModelFactory) createXAIModel() (model.LLM, error) {
	apiKey := mf.cfg.XAIAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("xAI API key not set - please set XAI_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "grok-3"
	}

	return adapters.NewFluxLLMAdapterWithConfig(adapters.FluxLLMAdapterConfig{
		ProviderName:      "xai",
		APIKey:            apiKey,
		ModelName:         modelName,
		ObservabilityHook: mf.obsHook,
	})
}

// createOllamaModel creates an Ollama model using FluxLLM
func (mf *ModelFactory) createOllamaModel() (model.LLM, error) {
	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "llama3.2"
	}

	// Ollama doesn't need an API key for local instances
	// FluxLLM will use the base URL from environment or default to localhost
	return adapters.NewFluxLLMAdapterWithConfig(adapters.FluxLLMAdapterConfig{
		ProviderName:      "ollama",
		APIKey:            "",
		ModelName:         modelName,
		ObservabilityHook: mf.obsHook,
	})
}

// GetProviderInfo returns information about the current provider
func (mf *ModelFactory) GetProviderInfo() string {
	return fmt.Sprintf("Provider: %s, Model: %s", mf.cfg.LLMProvider, mf.cfg.LLMModel)
}
