package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/tui"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure gendocs settings",
	Long: `Launch an interactive configuration wizard to set up your LLM provider,
API key, and model preferences. Configuration is saved to ~/.gendocs.yaml.`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Initialize Bubble Tea model
	model := tui.Model{
		Step:     0,
		Provider: "",
		Model:    "",
		BaseURL:  "",
		Quitting: false,
	}

	// Start Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use full screen mode
		tea.WithMouseCellMotion(), // Enable mouse motion
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running config wizard: %w", err)
	}

	// Type assertion to get our model back
	m, ok := finalModel.(tui.Model)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	// Show final status
	if m.Err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", m.Err)
		return m.Err
	}

	if m.SavedConfig {
		fmt.Printf("\nConfiguration saved to: %s\n", m.GetConfigPath())
		fmt.Println("\nYou can now run:")
		fmt.Println("  gendocs analyze --repo-path .")
	}

	return nil
}
