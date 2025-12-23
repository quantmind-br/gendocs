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

// AIRulesHandler handles the generate ai-rules command
type AIRulesHandler struct {
	*BaseHandler
	config config.AIRulesConfig
}

// NewAIRulesHandler creates a new AI rules handler
func NewAIRulesHandler(cfg config.AIRulesConfig, logger *logging.Logger) *AIRulesHandler {
	return &AIRulesHandler{
		BaseHandler: &BaseHandler{
			Config: cfg.BaseConfig,
			Logger: logger,
		},
		config: cfg,
	}
}

// Handle generates AI rules files
func (h *AIRulesHandler) Handle(ctx context.Context) error {
	h.Logger.Info("Starting AI rules generation",
		logging.String("repo_path", h.config.RepoPath),
	)

	// Load prompts
	promptManager, err := prompts.NewManager("./prompts")
	if err != nil {
		repoPromptsDir := fmt.Sprintf("%s/../gendocs/prompts", h.config.RepoPath)
		promptManager, err = prompts.NewManager(repoPromptsDir)
		if err != nil {
			return errors.NewConfigurationError(fmt.Sprintf("failed to load prompts: %v", err))
		}
	}

	// Create AI rules generator agent
	aiRulesAgent := agents.NewAIRulesGeneratorAgent(h.config, promptManager, h.Logger)

	// Run generation
	if err := aiRulesAgent.Run(ctx); err != nil {
		return errors.NewDocumentationError("AI rules", err)
	}

	h.Logger.Info("AI rules files generated successfully")
	return nil
}
