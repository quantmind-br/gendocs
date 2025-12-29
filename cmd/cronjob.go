package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/handlers"
	"github.com/user/gendocs/internal/logging"
)

// cronjobCmd represents the cronjob command
var cronjobCmd = &cobra.Command{
	Use:   "cronjob",
	Short: "Automated batch processing for GitLab projects",
	Long: `Process multiple GitLab projects automatically, analyzing each
and creating merge requests with the results.`,
}

var (
	cronjobMaxDays    int
	cronjobWorkingPath string
	cronjobGroupID     int
)

// cronjobAnalyzeCmd represents the cronjob analyze command
var cronjobAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze all applicable GitLab projects in a group",
	Long: `Fetch all projects in a GitLab group, filter them based on
configuration, and run analysis on each applicable project.
Creates branches and merge requests automatically.`,
	RunE: runCronjobAnalyze,
}

func init() {
	rootCmd.AddCommand(cronjobCmd)
	cronjobCmd.AddCommand(cronjobAnalyzeCmd)

	cronjobAnalyzeCmd.Flags().IntVar(&cronjobMaxDays, "max-days-since-last-commit", 14, "Skip projects with no commits in N days")
	cronjobAnalyzeCmd.Flags().StringVar(&cronjobWorkingPath, "working-path", "./work", "Working directory for cloning repos")
	cronjobAnalyzeCmd.Flags().IntVar(&cronjobGroupID, "group-project-id", 0, "GitLab group/project ID to analyze")
	cronjobAnalyzeCmd.MarkFlagRequired("group-project-id")
}

func runCronjobAnalyze(cmd *cobra.Command, args []string) error {
	// Validate required GitLab configuration
	gitLabToken := os.Getenv("GITLAB_OAUTH_TOKEN")
	if gitLabToken == "" {
		return errors.NewMissingEnvVarError("GITLAB_OAUTH_TOKEN", "GitLab API authentication token")
	}

	gitLabURL := os.Getenv("GITLAB_API_URL")
	if gitLabURL == "" {
		gitLabURL = "https://gitlab.com"
	}

	// Build configurations
	cronjobCfg := config.CronjobConfig{
		MaxDaysSinceLastCommit: cronjobMaxDays,
		WorkingPath:            cronjobWorkingPath,
		GroupProjectID:         cronjobGroupID,
	}

	gitLabCfg := config.GitLabConfig{
		APIURL:      gitLabURL,
		OAuthToken:  gitLabToken,
		UserName:    os.Getenv("GITLAB_USER_NAME"),
		UserUsername: os.Getenv("GITLAB_USER_USERNAME"),
		UserEmail:   os.Getenv("GITLAB_USER_EMAIL"),
	}

	// Set defaults for GitLab user info
	if gitLabCfg.UserName == "" {
		gitLabCfg.UserName = "AI Analyzer"
	}
	if gitLabCfg.UserUsername == "" {
		gitLabCfg.UserUsername = "agent_doc"
	}

	// Analyzer configuration (from env vars with defaults for cronjob)
	analyzerCfg := config.AnalyzerConfig{
		BaseConfig: config.BaseConfig{
			Debug: debugFlag,
		},
		LLM: LLMConfigFromEnv("ANALYZER", LLMDefaults{
			Retries:     2,
			Timeout:     180,
			MaxTokens:   8192,
			Temperature: 0.0,
		}),
		MaxWorkers: 0, // Auto-detect
	}

	// Initialize logger (verbose=true to enable console output for cronjob)
	logger, err := InitLogger(cronjobWorkingPath, debugFlag, true)
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Info("Starting cronjob analysis",
		logging.Int("group_id", cronjobGroupID),
		logging.Int("max_days", cronjobMaxDays),
	)

	// Create and run CronjobHandler
	handler := handlers.NewCronjobHandler(cronjobCfg, gitLabCfg, analyzerCfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		return HandleCommandError(err, nil, false)
	}

	logger.Info("Cronjob analysis complete")
	return nil
}
