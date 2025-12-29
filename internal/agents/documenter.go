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

// DocumenterAgent generates README.md
type DocumenterAgent struct {
	config        config.DocumenterConfig
	promptManager *prompts.Manager
	logger        *logging.Logger
}

// NewDocumenterAgent creates a new documenter agent
func NewDocumenterAgent(cfg config.DocumenterConfig, promptManager *prompts.Manager, logger *logging.Logger) *DocumenterAgent {
	return &DocumenterAgent{
		config:        cfg,
		promptManager: promptManager,
		logger:        logger,
	}
}

// Run generates the README
func (da *DocumenterAgent) Run(ctx context.Context) error {
	// Pre-load all analysis documents
	analysisFiles := []string{
		"structure_analysis.md",
		"dependency_analysis.md",
		"data_flow_analysis.md",
		"request_flow_analysis.md",
		"api_analysis.md",
	}

	analysisContent := make(map[string]string)
	docsDir := filepath.Join(da.config.RepoPath, ".ai/docs")

	for _, filename := range analysisFiles {
		filePath := filepath.Join(docsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			da.logger.Warn(fmt.Sprintf("Could not read %s: %v", filename, err))
			continue
		}
		analysisContent[filename] = string(content)
	}

	retryClient := llm.NewRetryClient(llm.DefaultRetryConfig())
	factory := llm.NewFactory(retryClient)

	// Create documenter agent
	agent, err := CreateDocumenterAgent(da.config.LLM, da.config.RepoPath, factory, da.promptManager, da.logger)
	if err != nil {
		return fmt.Errorf("failed to create documenter agent: %w", err)
	}

	// Render user prompt with analysis content embedded
	promptData := map[string]interface{}{
		"RepoPath":        da.config.RepoPath,
		"AnalysisContent": analysisContent,
	}
	userPrompt, err := da.promptManager.Render("documenter_user", promptData)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// Run agent with custom user prompt
	output, err := agent.RunOnce(ctx, userPrompt)
	if err != nil {
		return fmt.Errorf("documenter agent failed: %w", err)
	}

	// Save to README.md
	outputPath := filepath.Join(da.config.RepoPath, "README.md")
	if err := agent.SaveOutput(output, outputPath); err != nil {
		return err
	}

	da.logger.Info(fmt.Sprintf("README.md generated at %s", outputPath))
	return nil
}
