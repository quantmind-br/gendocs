package llm

import (
	"fmt"

	"github.com/user/gendocs/internal/config"
)

// Factory creates LLM clients
type Factory struct {
	retryClient *RetryClient
}

// NewFactory creates a new LLM factory
func NewFactory(retryClient *RetryClient) *Factory {
	return &Factory{
		retryClient: retryClient,
	}
}

// CreateClient creates an LLM client based on the provider configuration
func (f *Factory) CreateClient(cfg config.LLMConfig) (LLMClient, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIClient(cfg, f.retryClient), nil
	case "anthropic":
		return NewAnthropicClient(cfg, f.retryClient), nil
	case "gemini":
		return NewGeminiClient(cfg, f.retryClient), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: openai, anthropic, gemini)", cfg.Provider)
	}
}
