package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	debugFlag bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "gendocs",
	Short: "AI-powered code documentation generator",
	Long: `Generate comprehensive documentation for your codebase using AI.

Gendocs analyzes your codebase structure, dependencies, data flow, and APIs
to generate detailed documentation including README.md, AI assistant configs,
and more.`,
	Version: "2.0.0",
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug mode")
}
