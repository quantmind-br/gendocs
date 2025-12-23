package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
	"github.com/user/gendocs/internal/tools"
)

// SubAgentConfig holds configuration for sub-agents
type SubAgentConfig struct {
	Name         string
	LLMConfig    config.LLMConfig
	RepoPath     string
	PromptSuffix string // e.g., "structure_analyzer"
}

// SubAgent is a specialized analysis agent
type SubAgent struct {
	*BaseAgent
	config SubAgentConfig
}

// NewSubAgent creates a new sub-agent
func NewSubAgent(cfg SubAgentConfig, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	// Create LLM client
	llmClient, err := llmFactory.CreateClient(cfg.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Create tools
	toolList := []tools.Tool{
		tools.NewFileReadTool(2),
		tools.NewListFilesTool(2),
	}

	// Load system prompt
	systemPrompt, err := promptManager.Get(cfg.PromptSuffix + "_system")
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	baseAgent := NewBaseAgent(
		cfg.Name,
		llmClient,
		toolList,
		promptManager,
		logger,
		systemPrompt,
		cfg.LLMConfig.GetRetries(),
	)

	return &SubAgent{
		BaseAgent: baseAgent,
		config:    cfg,
	}, nil
}

// Run executes the sub-agent
func (sa *SubAgent) Run(ctx context.Context) (string, error) {
	// Render user prompt with variables
	userPrompt, err := sa.promptManager.Render(sa.config.PromptSuffix+"_user", map[string]interface{}{
		"RepoPath": sa.config.RepoPath,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render user prompt: %w", err)
	}

	// Run with retry logic
	var lastErr error
	for attempt := 0; attempt < sa.maxRetries; attempt++ {
		sa.logger.Debug(fmt.Sprintf("Running sub-agent %s (attempt %d/%d)", sa.config.Name, attempt+1, sa.maxRetries))

		result, err := sa.RunOnce(ctx, userPrompt)
		if err == nil {
			sa.logger.Info(fmt.Sprintf("Sub-agent %s completed successfully", sa.config.Name))
			return result, nil
		}

		lastErr = err
		sa.logger.Warn(fmt.Sprintf("Sub-agent %s attempt %d failed: %v", sa.config.Name, attempt+1, err))
	}

	return "", fmt.Errorf("sub-agent %s failed after %d retries: %w", sa.config.Name, sa.maxRetries, lastErr)
}

// SaveOutput saves the agent output to a file
func (sa *SubAgent) SaveOutput(output, outputPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	sa.logger.Info(fmt.Sprintf("Output saved to %s", outputPath))
	return nil
}
