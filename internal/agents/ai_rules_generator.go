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
)

// AIRulesGeneratorAgent generates AI assistant config files
type AIRulesGeneratorAgent struct {
	config        config.AIRulesConfig
	promptManager *prompts.Manager
	logger        *logging.Logger
}

// NewAIRulesGeneratorAgent creates a new AI rules generator agent
func NewAIRulesGeneratorAgent(cfg config.AIRulesConfig, promptManager *prompts.Manager, logger *logging.Logger) *AIRulesGeneratorAgent {
	return &AIRulesGeneratorAgent{
		config:        cfg,
		promptManager: promptManager,
		logger:        logger,
	}
}

// Run generates AI rules files
func (aa *AIRulesGeneratorAgent) Run(ctx context.Context) error {
	// Pre-load all analysis documents
	analysisFiles := []string{
		"structure_analysis.md",
		"dependency_analysis.md",
		"data_flow_analysis.md",
		"request_flow_analysis.md",
		"api_analysis.md",
	}

	analysisContent := make(map[string]string)
	docsDir := filepath.Join(aa.config.RepoPath, ".ai/docs")

	for _, filename := range analysisFiles {
		filePath := filepath.Join(docsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			aa.logger.Warn(fmt.Sprintf("Could not read %s: %v", filename, err))
			continue
		}
		analysisContent[filename] = string(content)
	}

	retryClient := llm.NewRetryClient(llm.DefaultRetryConfig())
	factory := llm.NewFactory(retryClient, nil, nil, false, 0)

	// For now, generate CLAUDE.md
	agent, err := CreateAIRulesGeneratorAgent(aa.config.LLM, aa.config.RepoPath, factory, aa.promptManager, aa.logger)
	if err != nil {
		return fmt.Errorf("failed to create AI rules agent: %w", err)
	}

	// Render user prompt with analysis content embedded
	promptData := map[string]interface{}{
		"RepoPath":        aa.config.RepoPath,
		"AnalysisContent": analysisContent,
	}
	userPrompt, err := aa.promptManager.Render("ai_rules_user", promptData)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// Run agent with custom user prompt
	output, err := agent.RunOnce(ctx, userPrompt)
	if err != nil {
		return fmt.Errorf("AI rules agent failed: %w", err)
	}

	// Save to CLAUDE.md
	outputPath := filepath.Join(aa.config.RepoPath, "CLAUDE.md")
	if err := agent.SaveOutput(output, outputPath); err != nil {
		return err
	}

	aa.logger.Info(fmt.Sprintf("CLAUDE.md generated at %s", outputPath))
	return nil
}
