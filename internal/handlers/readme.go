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

// ReadmeHandler handles the generate readme command
type ReadmeHandler struct {
	*BaseHandler
	config config.DocumenterConfig
}

// NewReadmeHandler creates a new readme handler
func NewReadmeHandler(cfg config.DocumenterConfig, logger *logging.Logger) *ReadmeHandler {
	return &ReadmeHandler{
		BaseHandler: &BaseHandler{
			Config: cfg.BaseConfig,
			Logger: logger,
		},
		config: cfg,
	}
}

// Handle generates the README
func (h *ReadmeHandler) Handle(ctx context.Context) error {
	h.Logger.Info("Starting readme generation",
		logging.String("repo_path", h.config.RepoPath),
	)

	// Load prompts with override support
	// System prompts: try "./prompts" first, fallback to repo-relative path
	systemPromptsDir := "./prompts"
	if _, err := prompts.NewManager(systemPromptsDir); err != nil {
		// Try relative to repo path
		systemPromptsDir = fmt.Sprintf("%s/../gendocs/prompts", h.config.RepoPath)
	}

	// Project prompts: .ai/prompts/ in the repository
	projectPromptsDir := fmt.Sprintf("%s/.ai/prompts", h.config.RepoPath)

	// Load with override support
	promptManager, err := prompts.NewManagerWithOverrides(systemPromptsDir, projectPromptsDir)
	if err != nil {
		return errors.NewConfigurationError(fmt.Sprintf("failed to load prompts: %v", err))
	}

	// Create documenter agent
	documenterAgent := agents.NewDocumenterAgent(h.config, promptManager, h.Logger)

	// Run generation
	if err := documenterAgent.Run(ctx); err != nil {
		return errors.NewDocumentationError("README", err)
	}

	h.Logger.Info("README.md generated successfully")
	return nil
}
