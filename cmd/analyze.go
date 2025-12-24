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

var (
	repoPath            string
	excludeStructure    bool
	excludeDataFlow     bool
	excludeDeps         bool
	excludeReqFlow      bool
	excludeAPI          bool
	maxWorkers          int
	forceAnalysis       bool
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze codebase structure and dependencies",
	Long: `Analyze the codebase to generate detailed documentation about:
  - Code structure and architecture
  - Dependencies and imports
  - Data flow through the system
  - Request/response flow
  - API endpoints and contracts

Results are written to .ai/docs/ directory.

By default, incremental analysis is used which only re-analyzes files
that have changed since the last run. Use --force to perform a full
re-analysis ignoring the cache.`,
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().StringVar(&repoPath, "repo-path", ".", "Path to repository")
	analyzeCmd.Flags().BoolVar(&excludeStructure, "exclude-code-structure", false, "Exclude structure analysis")
	analyzeCmd.Flags().BoolVar(&excludeDataFlow, "exclude-data-flow", false, "Exclude data flow analysis")
	analyzeCmd.Flags().BoolVar(&excludeDeps, "exclude-dependencies", false, "Exclude dependency analysis")
	analyzeCmd.Flags().BoolVar(&excludeReqFlow, "exclude-request-flow", false, "Exclude request flow analysis")
	analyzeCmd.Flags().BoolVar(&excludeAPI, "exclude-api-analysis", false, "Exclude API analysis")
	analyzeCmd.Flags().IntVar(&maxWorkers, "max-workers", 0, "Maximum concurrent workers (0=auto)")
	analyzeCmd.Flags().BoolVarP(&forceAnalysis, "force", "f", false, "Force full re-analysis, ignoring cache")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Build CLI overrides map
	cliOverrides := map[string]interface{}{
		"repo_path":              repoPath,
		"exclude_code_structure": excludeStructure,
		"exclude_data_flow":      excludeDataFlow,
		"exclude_dependencies":   excludeDeps,
		"exclude_request_flow":   excludeReqFlow,
		"exclude_api_analysis":   excludeAPI,
		"max_workers":            maxWorkers,
		"debug":                  debugFlag,
		"force":                  forceAnalysis,
	}

	// Load configuration
	cfg, err := config.LoadAnalyzerConfig(repoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	logDir := ".ai/logs"
	if cfg.RepoPath != "." {
		logDir = cfg.RepoPath + "/.ai/logs"
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

	logger.Info("Starting gendocs analyze",
		logging.String("repo_path", cfg.RepoPath),
		logging.Int("max_workers", cfg.MaxWorkers),
	)

	// Create and run AnalyzeHandler
	handler := handlers.NewAnalyzeHandler(*cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		// Handle error with proper exit code
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			return docErr
		}
		return err
	}

	logger.Info("Analysis complete")
	return nil
}
