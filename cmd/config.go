package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/tui/dashboard"
	"github.com/user/gendocs/internal/tui/dashboard/sections"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure gendocs settings",
	Long: `Launch an interactive configuration dashboard to manage all gendocs settings.

The dashboard provides a multi-section interface to configure:
  - LLM provider settings (API keys, models, parameters)
  - Response caching options
  - Analysis behavior and exclusions
  - Retry policies for API calls
  - Gemini/Vertex AI specific settings
  - GitLab integration
  - Cronjob scheduling
  - Logging configuration

Configuration can be saved to:
  - Global: ~/.gendocs.yaml (applies to all projects)
  - Project: .ai/config.yaml (project-specific overrides)

Use Ctrl+T to toggle between global and project scope.`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	model := dashboard.NewDashboard()

	model.RegisterSection("llm", sections.NewLLMSection())
	model.RegisterSection("documenter_llm", sections.NewDocumenterLLMSection())
	model.RegisterSection("ai_rules_llm", sections.NewAIRulesLLMSection())
	model.RegisterSection("cache", sections.NewCacheSection())
	model.RegisterSection("analysis", sections.NewAnalysisSection())
	model.RegisterSection("retry", sections.NewRetrySection())
	model.RegisterSection("gemini", sections.NewGeminiSection())
	model.RegisterSection("gitlab", sections.NewGitLabSection())
	model.RegisterSection("cronjob", sections.NewCronjobSection())
	model.RegisterSection("logging", sections.NewLoggingSection())

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running config dashboard: %w", err)
	}

	m, ok := finalModel.(dashboard.DashboardModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if m.HasError() {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", m.Error())
		return m.Error()
	}

	return nil
}
