package handlers

import (
	"context"
	"fmt"

	"github.com/user/gendocs/internal/agents"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
)

// AnalyzeHandler handles the analyze command
type AnalyzeHandler struct {
	*BaseHandler
	config config.AnalyzerConfig
}

// NewAnalyzeHandler creates a new analyze handler
func NewAnalyzeHandler(cfg config.AnalyzerConfig, logger *logging.Logger) *AnalyzeHandler {
	return &AnalyzeHandler{
		BaseHandler: &BaseHandler{
			Config: cfg.BaseConfig,
			Logger: logger,
		},
		config: cfg,
	}
}

// Handle executes the analysis
func (h *AnalyzeHandler) Handle(ctx context.Context) error {
	h.Logger.Info("Starting analyze handler",
		logging.String("repo_path", h.config.RepoPath),
	)

	// Load prompts
	// Try to find prompts directory - check relative to binary or repo
	promptManager, err := prompts.NewManager("./prompts")
	if err != nil {
		// Try relative to repo path
		repoPromptsDir := fmt.Sprintf("%s/../gendocs/prompts", h.config.RepoPath)
		promptManager, err = prompts.NewManager(repoPromptsDir)
		if err != nil {
			return errors.NewConfigurationError(fmt.Sprintf("failed to load prompts: %v", err))
		}
	}

	// Create analyzer agent
	analyzerAgent := agents.NewAnalyzerAgent(h.config, promptManager, h.Logger)

	// Run analysis
	result, err := analyzerAgent.Run(ctx)
	if err != nil {
		return errors.NewAnalysisError("analysis execution failed", err)
	}

	// Log results
	h.Logger.Info(fmt.Sprintf("Analysis complete: %d/%d successful",
		len(result.Successful), len(result.Successful)+len(result.Failed)))

	// Determine exit code
	if len(result.Failed) > 0 && len(result.Successful) == 0 {
		return errors.NewAnalysisError("all analyses failed", fmt.Errorf("no successful analyses"))
	}

	if len(result.Failed) > 0 {
		h.Logger.Warn(fmt.Sprintf("Partial success: %d analyses failed", len(result.Failed)))
		for _, failed := range result.Failed {
			h.Logger.Error(fmt.Sprintf("  - %s: %v", failed.Name, failed.Error))
		}
	}

	return nil
}
