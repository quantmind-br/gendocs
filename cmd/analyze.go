package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/handlers"
	"github.com/user/gendocs/internal/llmcache"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/tui"
)

type analyzeOptions struct {
	repoPath         string
	excludeStructure bool
	excludeDataFlow  bool
	excludeDeps      bool
	excludeReqFlow   bool
	excludeAPI       bool
	maxWorkers       int
	forceAnalysis    bool
	showCacheStats   bool
}

func newAnalyzeCmd() *cobra.Command {
	opts := &analyzeOptions{}

	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.repoPath, "repo-path", ".", "Path to repository")
	cmd.Flags().BoolVar(&opts.excludeStructure, "exclude-code-structure", false, "Exclude structure analysis")
	cmd.Flags().BoolVar(&opts.excludeDataFlow, "exclude-data-flow", false, "Exclude data flow analysis")
	cmd.Flags().BoolVar(&opts.excludeDeps, "exclude-dependencies", false, "Exclude dependency analysis")
	cmd.Flags().BoolVar(&opts.excludeReqFlow, "exclude-request-flow", false, "Exclude request flow analysis")
	cmd.Flags().BoolVar(&opts.excludeAPI, "exclude-api-analysis", false, "Exclude API analysis")
	cmd.Flags().IntVar(&opts.maxWorkers, "max-workers", 0, "Maximum concurrent workers (0=auto)")
	cmd.Flags().BoolVarP(&opts.forceAnalysis, "force", "f", false, "Force full re-analysis, ignoring cache")
	cmd.Flags().BoolVar(&opts.showCacheStats, "show-cache-stats", false, "Show LLM cache statistics after analysis")

	return cmd
}

func init() {
	rootCmd.AddCommand(newAnalyzeCmd())
}

func runAnalyze(cmd *cobra.Command, opts *analyzeOptions) error {
	cliOverrides := map[string]interface{}{
		"repo_path": opts.repoPath,
		"debug":     debugFlag,
	}

	if cmd.Flags().Changed("exclude-code-structure") {
		cliOverrides["exclude_code_structure"] = opts.excludeStructure
	}
	if cmd.Flags().Changed("exclude-data-flow") {
		cliOverrides["exclude_data_flow"] = opts.excludeDataFlow
	}
	if cmd.Flags().Changed("exclude-dependencies") {
		cliOverrides["exclude_dependencies"] = opts.excludeDeps
	}
	if cmd.Flags().Changed("exclude-request-flow") {
		cliOverrides["exclude_request_flow"] = opts.excludeReqFlow
	}
	if cmd.Flags().Changed("exclude-api-analysis") {
		cliOverrides["exclude_api_analysis"] = opts.excludeAPI
	}
	if cmd.Flags().Changed("max-workers") {
		cliOverrides["max_workers"] = opts.maxWorkers
	}
	if cmd.Flags().Changed("force") {
		cliOverrides["force"] = opts.forceAnalysis
	}

	cfg, err := config.LoadAnalyzerConfig(opts.repoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger using helper
	logger, err := InitLogger(cfg.RepoPath, debugFlag, verboseFlag)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	showProgress := !verboseFlag

	logger.Info("Starting gendocs analyze",
		logging.String("repo_path", cfg.RepoPath),
		logging.Int("max_workers", cfg.MaxWorkers),
	)

	handler := handlers.NewAnalyzeHandler(*cfg, logger)

	var progress *tui.Progress
	if showProgress {
		progress = tui.NewProgress("Gendocs Analyze")
		progress.SetSubtitle(fmt.Sprintf("Repository: %s", cfg.RepoPath))
		handler.SetProgressReporter(progress)
		progress.Start()
	}

	err = handler.Handle(cmd.Context())

	if showProgress {
		progress.Stop()
		progress.PrintSummary()
	}

	if err != nil {
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			if !showProgress {
				fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			}
			return docErr
		}
		return err
	}

	if !showProgress {
		logger.Info("Analysis complete")
	}

	// Show cache statistics if requested
	if opts.showCacheStats {
		displayCacheStats(opts.repoPath)
	}

	return nil
}

// displayCacheStats loads and displays cache statistics from the disk cache
func displayCacheStats(repoPath string) {
	cachePath := filepath.Join(repoPath, llmcache.DefaultCacheFileName)

	// Check if cache file exists
	fileInfo, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		fmt.Println("\n📊 LLM Cache Statistics")
		fmt.Println("   Cache file not found. Run analysis with caching enabled first.")
		fmt.Printf("   Expected location: %s\n\n", cachePath)
		return
	}

	// Get actual file size on disk
	actualFileSize := fileInfo.Size()

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		fmt.Printf("\n❌ Failed to read cache file: %v\n\n", err)
		return
	}

	// Parse cache data
	var cacheData llmcache.DiskCacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		fmt.Printf("\n❌ Failed to parse cache file: %v\n\n", err)
		return
	}

	// Display statistics
	fmt.Println("\n📊 LLM Cache Statistics")
	fmt.Println("======================")
	fmt.Printf("Cache File: %s\n", cachePath)
	fmt.Printf("Version: %d\n", cacheData.Version)
	fmt.Printf("Created: %s\n", cacheData.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Last Updated: %s\n\n", cacheData.UpdatedAt.Format("2006-01-02 15:04:05"))

	stats := cacheData.Stats
	fmt.Println("Entries:")
	fmt.Printf("  Total Entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Expired Entries: %d\n", stats.ExpiredEntries)
	fmt.Printf("  Active Entries: %d\n\n", stats.TotalEntries-stats.ExpiredEntries)

	fmt.Println("Performance:")
	fmt.Printf("  Cache Hits: %d\n", stats.Hits)
	fmt.Printf("  Cache Misses: %d\n", stats.Misses)
	fmt.Printf("  Hit Rate: %.2f%%\n\n", stats.HitRate*100)

	fmt.Println("Disk Usage:")
	fmt.Printf("  Actual File Size: %.2f MB\n", float64(actualFileSize)/(1024*1024))
	fmt.Printf("  Logical Data Size: %.2f MB\n", float64(stats.TotalSizeBytes)/(1024*1024))
	if stats.TotalSizeBytes > 0 {
		efficiency := float64(stats.TotalSizeBytes) / float64(actualFileSize) * 100
		fmt.Printf("  Storage Efficiency: %.1f%% (data size / file size)\n", efficiency)
	}
	fmt.Printf("  Evictions: %d\n\n", stats.Evictions)

	fmt.Println("======================")
}
