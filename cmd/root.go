package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	debugFlag          bool
	verboseFlag        bool
	cacheStatsRepoPath string
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
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed log output instead of progress UI")

	// Add cache-stats command
	rootCmd.AddCommand(cacheStatsCmd)
	cacheStatsCmd.Flags().StringVar(&cacheStatsRepoPath, "repo-path", ".", "Path to repository")

	// Add cache-clear command
	rootCmd.AddCommand(cacheClearCmd)
	cacheClearCmd.Flags().StringVar(&cacheClearRepoPath, "repo-path", ".", "Path to repository")
}

// cacheStatsCmd represents the cache-stats command
var cacheStatsCmd = &cobra.Command{
	Use:   "cache-stats",
	Short: "Display LLM cache statistics",
	Long: `Display statistics about the LLM response cache, including:
  - Total entries and expired entries
  - Cache hits, misses, and hit rate
  - Storage size and evictions

This command shows statistics from the disk cache file without running analysis.`,
	RunE: runCacheStats,
}

func runCacheStats(cmd *cobra.Command, args []string) error {
	displayCacheStats(cacheStatsRepoPath)
	return nil
}
