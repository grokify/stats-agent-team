package adapters

import (
	"context"
	"fmt"
	"iter"

	"github.com/grokify/fluxllm"
	"github.com/grokify/fluxllm/provider"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// FluxLLMAdapterConfig holds configuration for creating a FluxLLM adapter
type FluxLLMAdapterConfig struct {
	ProviderName      string
	APIKey            string
	ModelName         string
	ObservabilityHook fluxllm.ObservabilityHook
}

// FluxLLMAdapter adapts FluxLLM ChatClient to ADK's LLM interface
type FluxLLMAdapter struct {
	client *fluxllm.ChatClient
	model  string
}

// NewFluxLLMAdapter creates a new FluxLLM adapter
func NewFluxLLMAdapter(providerName, apiKey, modelName string) (*FluxLLMAdapter, error) {
	return NewFluxLLMAdapterWithConfig(FluxLLMAdapterConfig{
		ProviderName: providerName,
		APIKey:       apiKey,
		ModelName:    modelName,
	})
}

// NewFluxLLMAdapterWithConfig creates a new FluxLLM adapter with full configuration
func NewFluxLLMAdapterWithConfig(cfg FluxLLMAdapterConfig) (*FluxLLMAdapter, error) {
	// For ollama, API key is optional
	if cfg.ProviderName != "ollama" && cfg.APIKey == "" {
		return nil, fmt.Errorf("%s API key is required", cfg.ProviderName)
	}

	// Create FluxLLM config
	config := fluxllm.ClientConfig{
		Provider:          fluxllm.ProviderName(cfg.ProviderName),
		APIKey:            cfg.APIKey,
		ObservabilityHook: cfg.ObservabilityHook,
	}

	client, err := fluxllm.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create FluxLLM client: %w", err)
	}

	return &FluxLLMAdapter{
		client: client,
		model:  cfg.ModelName,
	}, nil
}

// Name returns the model name
func (f *FluxLLMAdapter) Name() string {
	return f.model
}

// GenerateContent implements the LLM interface
func (f *FluxLLMAdapter) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to FluxLLM request
		messages := make([]provider.Message, 0)

		for _, content := range req.Contents {
			var text string
			for _, part := range content.Parts {
				text += part.Text
			}

			role := provider.RoleUser
			if content.Role == "model" || content.Role == "assistant" {
				role = provider.RoleAssistant
			} else if content.Role == "system" {
				role = provider.RoleSystem
			}

			messages = append(messages, provider.Message{
				Role:    role,
				Content: text,
			})
		}

		// Create FluxLLM request
		fluxReq := &provider.ChatCompletionRequest{
			Model:    f.model,
			Messages: messages,
		}

		// Call FluxLLM API
		resp, err := f.client.CreateChatCompletion(ctx, fluxReq)
		if err != nil {
			yield(nil, fmt.Errorf("FluxLLM API error: %w", err))
			return
		}

		// Convert FluxLLM response to ADK response
		if len(resp.Choices) > 0 {
			adkResp := &model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: resp.Choices[0].Message.Content},
					},
				},
			}
			yield(adkResp, nil)
		}
	}
}
