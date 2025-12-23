package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/handlers"
	"github.com/user/gendocs/internal/logging"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate documentation from analysis results",
	Long:  `Generate documentation files (README.md, AI rules) from existing analysis results.`,
}

var (
	readmeRepoPath string
)

// readmeCmd represents the generate readme command
var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Generate README.md from analysis results",
	Long: `Generate a comprehensive README.md file based on existing analysis documents
in .ai/docs/. This synthesizes information from structure, dependency, data flow,
request flow, and API analyses into a user-friendly README.`,
	RunE: runReadme,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(readmeCmd)

	readmeCmd.Flags().StringVar(&readmeRepoPath, "repo-path", ".", "Path to repository")
}

func runReadme(cmd *cobra.Command, args []string) error {
	// Build configuration
	cfg := config.DocumenterConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: readmeRepoPath,
			Debug:    debugFlag,
		},
		LLM: config.LLMConfig{
			Provider:    os.Getenv("DOCUMENTER_LLM_PROVIDER"),
			Model:       os.Getenv("DOCUMENTER_LLM_MODEL"),
			APIKey:      os.Getenv("DOCUMENTER_LLM_API_KEY"),
			BaseURL:     os.Getenv("DOCUMENTER_LLM_BASE_URL"),
			Retries:     2,
			Timeout:     180,
			MaxTokens:   8192,
			Temperature: 0.0,
		},
	}

	// Set defaults from environment if not set
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = os.Getenv("ANALYZER_LLM_PROVIDER")
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = os.Getenv("ANALYZER_LLM_MODEL")
	}
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = os.Getenv("ANALYZER_LLM_API_KEY")
	}

	// Initialize logger
	logDir := ".ai/logs"
	if readmeRepoPath != "." {
		logDir = readmeRepoPath + "/.ai/logs"
	}
	logCfg := &logging.Config{
		LogDir:       logDir,
		FileLevel:    logging.LevelFromString("info"),
		ConsoleLevel: logging.LevelFromString("debug"),
		EnableCaller: debugFlag,
	}

	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting README generation",
		logging.String("repo_path", readmeRepoPath),
	)

	// Create and run ReadmeHandler
	handler := handlers.NewReadmeHandler(cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			return docErr
		}
		return err
	}

	logger.Info("README.md generation complete")
	return nil
}

// aiRulesCmd represents the generate ai-rules command
var aiRulesCmd = &cobra.Command{
	Use:   "ai-rules",
	Short: "Generate AI assistant configuration files",
	Long: `Generate AI assistant configuration files (CLAUDE.md, AGENTS.md, .cursor/rules/)
from existing analysis results. These files help AI coding assistants understand the project.`,
	RunE: runAIRules,
}

func init() {
	generateCmd.AddCommand(aiRulesCmd)
	aiRulesCmd.Flags().StringVar(&readmeRepoPath, "repo-path", ".", "Path to repository")
}

func runAIRules(cmd *cobra.Command, args []string) error {
	// Build configuration
	cfg := config.AIRulesConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: readmeRepoPath,
			Debug:    debugFlag,
		},
		LLM: config.LLMConfig{
			Provider:    os.Getenv("AI_RULES_LLM_PROVIDER"),
			Model:       os.Getenv("AI_RULES_LLM_MODEL"),
			APIKey:      os.Getenv("AI_RULES_LLM_API_KEY"),
			BaseURL:     os.Getenv("AI_RULES_LLM_BASE_URL"),
			Retries:     2,
			Timeout:     240,
			MaxTokens:   8192,
			Temperature: 0.0,
		},
	}

	// Set defaults from environment if not set
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = os.Getenv("ANALYZER_LLM_PROVIDER")
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = os.Getenv("ANALYZER_LLM_MODEL")
	}
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = os.Getenv("ANALYZER_LLM_API_KEY")
	}

	// Initialize logger
	logDir := ".ai/logs"
	if readmeRepoPath != "." {
		logDir = readmeRepoPath + "/.ai/logs"
	}
	logCfg := &logging.Config{
		LogDir:       logDir,
		FileLevel:    logging.LevelFromString("info"),
		ConsoleLevel: logging.LevelFromString("debug"),
		EnableCaller: debugFlag,
	}

	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting AI rules generation",
		logging.String("repo_path", readmeRepoPath),
	)

	// Create and run AIRulesHandler
	handler := handlers.NewAIRulesHandler(cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			return docErr
		}
		return err
	}

	logger.Info("AI rules generation complete")
	return nil
}
