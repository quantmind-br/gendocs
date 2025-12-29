package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/llmcache"
)

var (
	cacheClearRepoPath string
)

// cacheClearCmd represents the cache-clear command
var cacheClearCmd = &cobra.Command{
	Use:   "cache-clear",
	Short: "Clear the LLM response cache",
	Long: `Clear the LLM response cache by removing all cached entries.

This command removes the disk cache file, forcing all LLM requests to be
executed again on the next run. Use this to:
  - Free up disk space
  - Force fresh LLM responses
  - Reset cache statistics`,
	RunE: runCacheClear,
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	return clearCache(cacheClearRepoPath)
}

// clearCache removes the LLM cache file
func clearCache(repoPath string) error {
	cachePath := filepath.Join(repoPath, llmcache.DefaultCacheFileName)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		fmt.Println("ℹ️  Cache file not found.")
		fmt.Printf("   Expected location: %s\n", cachePath)
		fmt.Println("   No action taken.")
		return nil
	}

	// Remove the cache file
	if err := os.Remove(cachePath); err != nil {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	fmt.Println("✅ Cache cleared successfully!")
	fmt.Printf("   Removed: %s\n\n", cachePath)
	return nil
}
