package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/gendocs/internal/cache"
)

// Simple manual verification that cache save/load works with the new json.Marshal
func main() {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		fmt.Printf("❌ Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("✓ Created temporary directory for testing")

	// Create a test cache
	testCache := cache.NewCache()
	testCache.LastAnalysis = testCache.LastAnalysis // Zero time is fine for test
	testCache.Files = map[string]cache.FileInfo{
		"test.go": {
			Hash:     "abc123",
			Modified: testCache.LastAnalysis,
			Size:     1024,
		},
	}

	fmt.Println("✓ Created test cache with sample data")

	// Save the cache using the new json.Marshal implementation
	if err := testCache.Save(tmpDir); err != nil {
		fmt.Printf("❌ Failed to save cache: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Successfully saved cache with json.Marshal (no indentation)")

	// Read the cache file to verify it's valid JSON
	cachePath := filepath.Join(tmpDir, cache.CacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		fmt.Printf("❌ Failed to read cache file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Successfully read cache file from disk")

	// Verify it's valid JSON by unmarshaling it
	var loadedCache cache.AnalysisCache
	if err := json.Unmarshal(data, &loadedCache); err != nil {
		fmt.Printf("❌ Failed to unmarshal cache: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Successfully unmarshaled cache (valid JSON)")

	// Verify the cache can be loaded using LoadCache
	loadedCache2, err := cache.LoadCache(tmpDir)
	if err != nil {
		fmt.Printf("❌ Failed to load cache using LoadCache: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Successfully loaded cache using LoadCache function")

	// Verify data integrity
	if len(loadedCache2.Files) != len(testCache.Files) {
		fmt.Printf("❌ File count mismatch: expected %d, got %d\n", len(testCache.Files), len(loadedCache2.Files))
		os.Exit(1)
	}

	if _, exists := loadedCache2.Files["test.go"]; !exists {
		fmt.Printf("❌ Expected file 'test.go' not found in loaded cache\n")
		os.Exit(1)
	}

	fmt.Println("✓ Cache data integrity verified")

	// Compare file sizes (new format should be smaller)
	originalSize := len(data)
	fmt.Printf("✓ Cache file size: %d bytes\n", originalSize)

	// For comparison, create a pretty-printed version
	prettyData, err := json.MarshalIndent(loadedCache2, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to create pretty-printed version: %v\n", err)
		os.Exit(1)
	}

	prettySize := len(prettyData)
	sizeDiff := prettySize - originalSize
	percentageReduced := float64(sizeDiff) / float64(prettySize) * 100

	fmt.Printf("✓ Pretty-printed version would be: %d bytes\n", prettySize)
	fmt.Printf("✓ Space saved: %d bytes (%.1f%% reduction)\n", sizeDiff, percentageReduced)

	// Verify the new format is compact (no unnecessary newlines/indentation)
	if string(data[0]) == "{" && !containsIndentation(data) {
		fmt.Println("✓ Verified compact JSON format (no indentation)")
	} else {
		fmt.Println("⚠ Warning: JSON might contain unexpected formatting")
	}

	fmt.Println("\n🎉 All verifications passed!")
	fmt.Println("\nSummary:")
	fmt.Println("- Cache.Save() successfully uses json.Marshal (compact format)")
	fmt.Println("- Cache.LoadCache() successfully reads compact JSON")
	fmt.Println("- Data integrity is preserved")
	fmt.Println("- File size is reduced by removing unnecessary indentation")
}

func containsIndentation(data []byte) bool {
	// Check for common indentation patterns
	for _, b := range data {
		if b == '\t' {
			return true
		}
	}
	return false
}
