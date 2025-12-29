package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_FullScanCycle verifies the complete workflow: load cache -> scan -> detect changes -> save
func TestIntegration_FullScanCycle(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create initial test files
	files := map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"utils.go":  "package main\n\nfunc Utils() {}\n",
		"README.md": "# Test Project\n",
		"go.mod":    "module test\n\ngo 1.21\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// First scan: no cache exists
	cache1, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if cache1.Version != CacheVersion {
		t.Errorf("Expected cache version %d, got %d", CacheVersion, cache1.Version)
	}

	if len(cache1.Files) != 0 {
		t.Errorf("Expected empty cache on first load, got %d files", len(cache1.Files))
	}

	// Scan files
	metrics1 := &ScanMetrics{}
	currentFiles1, err := ScanFiles(tmpDir, []string{}, cache1, metrics1, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify first scan results
	if metrics1.TotalFiles != 4 {
		t.Errorf("Expected 4 total files, got %d", metrics1.TotalFiles)
	}

	if metrics1.CachedFiles != 0 {
		t.Errorf("Expected 0 cached files on first scan, got %d", metrics1.CachedFiles)
	}

	if metrics1.HashedFiles != 4 {
		t.Errorf("Expected 4 hashed files on first scan, got %d", metrics1.HashedFiles)
	}

	// Detect changes (should detect all as new on first run)
	report1 := cache1.DetectChanges(tmpDir, currentFiles1)
	if !report1.IsFirstRun {
		t.Error("Expected IsFirstRun to be true")
	}

	if !report1.HasChanges {
		t.Error("Expected HasChanges to be true on first run")
	}

	if len(report1.NewFiles) != 4 {
		t.Errorf("Expected 4 new files, got %d", len(report1.NewFiles))
	}

	// Update cache after analysis
	agentResults1 := map[string]bool{
		"structure_analyzer":    true,
		"dependency_analyzer":   true,
		"data_flow_analyzer":    true,
		"request_flow_analyzer": true,
		"api_analyzer":          true,
	}
	cache1.UpdateAfterAnalysis(tmpDir, currentFiles1, agentResults1)

	// Save cache
	if err := cache1.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, CacheFileName)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Second scan: load existing cache
	cache2, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	// Verify cache was loaded correctly
	if len(cache2.Files) != 4 {
		t.Errorf("Expected 4 files in loaded cache, got %d", len(cache2.Files))
	}

	if cache2.LastAnalysis.IsZero() {
		t.Errorf("Expected LastAnalysis to be set")
	}

	// Scan files again (should use cache)
	metrics2 := &ScanMetrics{}
	currentFiles2, err := ScanFiles(tmpDir, []string{}, cache2, metrics2, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify cache was used
	if metrics2.TotalFiles != 4 {
		t.Errorf("Expected 4 total files, got %d", metrics2.TotalFiles)
	}

	if metrics2.CachedFiles != 4 {
		t.Errorf("Expected 4 cached files, got %d", metrics2.CachedFiles)
	}

	if metrics2.HashedFiles != 0 {
		t.Errorf("Expected 0 hashed files (all cached), got %d", metrics2.HashedFiles)
	}

	// Detect changes (should detect no changes)
	report2 := cache2.DetectChanges(tmpDir, currentFiles2)
	if report2.IsFirstRun {
		t.Error("Expected IsFirstRun to be false on second run")
	}

	if report2.HasChanges {
		t.Error("Expected HasChanges to be false when nothing changed")
	}

	if len(report2.NewFiles) != 0 {
		t.Errorf("Expected 0 new files, got %d", len(report2.NewFiles))
	}

	if len(report2.ModifiedFiles) != 0 {
		t.Errorf("Expected 0 modified files, got %d", len(report2.ModifiedFiles))
	}

	// Verify file hashes match between scans
	for path, file1 := range currentFiles1 {
		file2, exists := currentFiles2[path]
		if !exists {
			t.Errorf("File %s missing from second scan", path)
			continue
		}

		if file1.Hash != file2.Hash {
			t.Errorf("Hash mismatch for %s: %s != %s", path, file1.Hash, file2.Hash)
		}
	}
}

// TestIntegration_IncrementalScanWithChanges verifies incremental scanning when files are modified
func TestIntegration_IncrementalScanWithChanges(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create initial files
	initialFiles := map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"utils.go":  "package main\n\nfunc Utils() {}\n",
		"README.md": "# Test Project\n",
	}

	for name, content := range initialFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// First scan: create initial cache
	cache1 := NewCache()
	metrics1 := &ScanMetrics{}
	currentFiles1, err := ScanFiles(tmpDir, []string{}, cache1, metrics1, 0)
	if err != nil {
		t.Fatalf("First ScanFiles failed: %v", err)
	}

	cache1.UpdateAfterAnalysis(tmpDir, currentFiles1, map[string]bool{"structure_analyzer": true})
	if err := cache1.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Wait a bit to ensure mtime changes
	time.Sleep(10 * time.Millisecond)

	// Modify some files and add new ones
	// Note: We don't rewrite utils.go because WriteFile would change its mtime,
	// making the cache detect it as modified even though content is the same
	modifiedFiles := map[string]string{
		"main.go":   "package main\n\nfunc main() {\n\tprintln(\"changed\")\n}\n", // Modified
		"README.md": "# Test Project\n\nUpdated documentation\n",                  // Modified
		"new.go":    "package main\n\nfunc New() {}\n",                            // New file
	}

	for name, content := range modifiedFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}

	// Second scan: load cache and detect changes
	cache2, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	metrics2 := &ScanMetrics{}
	currentFiles2, err := ScanFiles(tmpDir, []string{}, cache2, metrics2, 0)
	if err != nil {
		t.Fatalf("Second ScanFiles failed: %v", err)
	}

	// Verify scan metrics
	if metrics2.TotalFiles != 4 {
		t.Errorf("Expected 4 total files, got %d", metrics2.TotalFiles)
	}

	if metrics2.CachedFiles != 1 {
		t.Errorf("Expected 1 cached file (utils.go), got %d", metrics2.CachedFiles)
	}

	if metrics2.HashedFiles != 3 {
		t.Errorf("Expected 3 hashed files (main.go, README.md, new.go), got %d", metrics2.HashedFiles)
	}

	// Detect changes
	report2 := cache2.DetectChanges(tmpDir, currentFiles2)
	if !report2.HasChanges {
		t.Error("Expected HasChanges to be true when files changed")
	}

	if len(report2.NewFiles) != 1 {
		t.Errorf("Expected 1 new file (new.go), got %d", len(report2.NewFiles))
	}

	if report2.NewFiles[0] != "new.go" {
		t.Errorf("Expected new.go to be new, got %s", report2.NewFiles[0])
	}

	if len(report2.ModifiedFiles) != 2 {
		t.Errorf("Expected 2 modified files (main.go, README.md), got %d", len(report2.ModifiedFiles))
	}

	// Verify modified files are detected (order may vary)
	modifiedMap := make(map[string]bool)
	for _, file := range report2.ModifiedFiles {
		modifiedMap[file] = true
	}

	if !modifiedMap["main.go"] {
		t.Error("Expected main.go to be in modified files")
	}

	if !modifiedMap["README.md"] {
		t.Error("Expected README.md to be in modified files")
	}

	if modifiedMap["utils.go"] {
		t.Error("Expected utils.go to NOT be in modified files (unchanged)")
	}

	if len(report2.DeletedFiles) != 0 {
		t.Errorf("Expected 0 deleted files, got %d", len(report2.DeletedFiles))
	}
}

// TestIntegration_IncrementalScanWithDeletions verifies handling of deleted files
func TestIntegration_IncrementalScanWithDeletions(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create initial files
	initialFiles := map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"utils.go":  "package main\n\nfunc Utils() {}\n",
		"config.go": "package main\n\nfunc Config() {}\n",
	}

	for name, content := range initialFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// First scan: create initial cache
	cache1 := NewCache()
	metrics1 := &ScanMetrics{}
	currentFiles1, err := ScanFiles(tmpDir, []string{}, cache1, metrics1, 0)
	if err != nil {
		t.Fatalf("First ScanFiles failed: %v", err)
	}

	cache1.UpdateAfterAnalysis(tmpDir, currentFiles1, map[string]bool{"structure_analyzer": true})
	if err := cache1.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Delete some files
	_ = os.Remove(filepath.Join(tmpDir, "utils.go"))
	_ = os.Remove(filepath.Join(tmpDir, "config.go"))

	// Modify remaining file
	time.Sleep(10 * time.Millisecond)
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {\n}\n"), 0644)

	// Second scan: load cache and detect deletions
	cache2, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	metrics2 := &ScanMetrics{}
	currentFiles2, err := ScanFiles(tmpDir, []string{}, cache2, metrics2, 0)
	if err != nil {
		t.Fatalf("Second ScanFiles failed: %v", err)
	}

	// Verify only main.go is in current files
	if len(currentFiles2) != 1 {
		t.Errorf("Expected 1 current file, got %d", len(currentFiles2))
	}

	if _, exists := currentFiles2["main.go"]; !exists {
		t.Error("Expected main.go to be in current files")
	}

	// Detect changes
	report2 := cache2.DetectChanges(tmpDir, currentFiles2)
	if !report2.HasChanges {
		t.Error("Expected HasChanges to be true when files deleted")
	}

	if len(report2.DeletedFiles) != 2 {
		t.Errorf("Expected 2 deleted files, got %d", len(report2.DeletedFiles))
	}

	// Verify deleted files (order may vary)
	deletedMap := make(map[string]bool)
	for _, file := range report2.DeletedFiles {
		deletedMap[file] = true
	}

	if !deletedMap["utils.go"] {
		t.Error("Expected utils.go to be in deleted files")
	}

	if !deletedMap["config.go"] {
		t.Error("Expected config.go to be in deleted files")
	}

	if deletedMap["main.go"] {
		t.Error("Expected main.go to NOT be in deleted files")
	}
}

// TestIntegration_CachePersistenceAcrossMultipleCycles verifies cache persistence across many scan cycles
func TestIntegration_CachePersistenceAcrossMultipleCycles(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create initial files
	files := map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// Perform multiple scan cycles
	numCycles := 5
	var prevHash string

	for i := 0; i < numCycles; i++ {
		// Load cache
		cache, err := LoadCache(tmpDir)
		if err != nil {
			t.Fatalf("Cycle %d: LoadCache failed: %v", i, err)
		}

		// Scan files
		metrics := &ScanMetrics{}
		currentFiles, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
		if err != nil {
			t.Fatalf("Cycle %d: ScanFiles failed: %v", i, err)
		}

		// On first cycle, all files should be hashed
		if i == 0 {
			if metrics.HashedFiles != 1 {
				t.Errorf("Cycle %d: Expected 1 hashed file, got %d", i, metrics.HashedFiles)
			}
			if metrics.CachedFiles != 0 {
				t.Errorf("Cycle %d: Expected 0 cached files, got %d", i, metrics.CachedFiles)
			}
		} else {
			// On subsequent cycles, all files should be cached
			if metrics.HashedFiles != 0 {
				t.Errorf("Cycle %d: Expected 0 hashed files, got %d", i, metrics.HashedFiles)
			}
			if metrics.CachedFiles != 1 {
				t.Errorf("Cycle %d: Expected 1 cached file, got %d", i, metrics.CachedFiles)
			}
		}

		// Update and save cache
		cache.UpdateAfterAnalysis(tmpDir, currentFiles, map[string]bool{"structure_analyzer": true})
		if err := cache.Save(tmpDir); err != nil {
			t.Fatalf("Cycle %d: Save failed: %v", i, err)
		}

		// Verify hash remains consistent across cycles
		currentHash := currentFiles["main.go"].Hash
		if prevHash != "" && currentHash != prevHash {
			t.Errorf("Cycle %d: Hash changed unexpectedly: %s -> %s", i, prevHash, currentHash)
		}
		prevHash = currentHash

		// Small delay to ensure mtime changes if we were to modify files
		time.Sleep(5 * time.Millisecond)
	}

	// Final verification: load cache one more time and verify contents
	finalCache, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("Final LoadCache failed: %v", err)
	}

	if len(finalCache.Files) != 1 {
		t.Errorf("Expected 1 file in final cache, got %d", len(finalCache.Files))
	}

	if finalCache.Files["main.go"].Hash != prevHash {
		t.Errorf("Final cache hash mismatch: %s != %s", finalCache.Files["main.go"].Hash, prevHash)
	}

	if len(finalCache.Agents) != 1 {
		t.Errorf("Expected 1 agent in final cache, got %d", len(finalCache.Agents))
	}
}

// TestIntegration_CacheWithIgnorePatterns verifies cache works correctly with ignore patterns
func TestIntegration_CacheWithIgnorePatterns(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create files including some that should be ignored
	files := map[string]string{
		"main.go":         "package main\n\nfunc main() {}\n",
		"vendor/lib.go":   "package vendor\n",
		"test_test.go":    "package test\n",
		"README.md":       "# Test\n",
		".ai/config.yaml": "config: true\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// First scan with ignore patterns
	ignorePatterns := []string{"vendor", "*_test.go"}
	cache1 := NewCache()
	metrics1 := &ScanMetrics{}
	currentFiles1, err := ScanFiles(tmpDir, ignorePatterns, cache1, metrics1, 0)
	if err != nil {
		t.Fatalf("First ScanFiles failed: %v", err)
	}

	// Verify only non-ignored files were scanned
	if metrics1.TotalFiles != 2 {
		t.Errorf("Expected 2 total files (main.go, README.md), got %d", metrics1.TotalFiles)
	}

	if len(currentFiles1) != 2 {
		t.Errorf("Expected 2 files in results, got %d", len(currentFiles1))
	}

	// Save cache
	cache1.UpdateAfterAnalysis(tmpDir, currentFiles1, map[string]bool{"structure_analyzer": true})
	if err := cache1.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Second scan with same ignore patterns
	cache2, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	metrics2 := &ScanMetrics{}
	_, err = ScanFiles(tmpDir, ignorePatterns, cache2, metrics2, 0)
	if err != nil {
		t.Fatalf("Second ScanFiles failed: %v", err)
	}

	// Verify cache was used correctly
	if metrics2.CachedFiles != 2 {
		t.Errorf("Expected 2 cached files, got %d", metrics2.CachedFiles)
	}

	if metrics2.HashedFiles != 0 {
		t.Errorf("Expected 0 hashed files, got %d", metrics2.HashedFiles)
	}

	// Verify ignored files are not in cache
	if _, exists := cache2.Files["vendor/lib.go"]; exists {
		t.Error("vendor/lib.go should not be in cache (ignored)")
	}

	if _, exists := cache2.Files["test_test.go"]; exists {
		t.Error("test_test.go should not be in cache (ignored)")
	}

	if _, exists := cache2.Files[".ai/config.yaml"]; exists {
		t.Error(".ai/config.yaml should not be in cache (ignored)")
	}
}

// TestIntegration_CacheCorruptionHandling verifies handling of corrupted cache files
func TestIntegration_CacheCorruptionHandling(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create a corrupted cache file
	cachePath := filepath.Join(tmpDir, CacheFileName)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Write invalid JSON
	corruptedData := []byte("{ invalid json content")
	if err := os.WriteFile(cachePath, corruptedData, 0644); err != nil {
		t.Fatalf("Failed to write corrupted cache: %v", err)
	}

	// Load cache should handle corruption gracefully and return new cache
	cache, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache should not fail on corrupted cache, got: %v", err)
	}

	// Should return a fresh cache
	if cache == nil {
		t.Fatal("Expected non-nil cache after corruption")
	}

	if len(cache.Files) != 0 {
		t.Errorf("Expected empty cache after corruption, got %d files", len(cache.Files))
	}

	if cache.Version != CacheVersion {
		t.Errorf("Expected cache version %d after corruption, got %d", CacheVersion, cache.Version)
	}

	// Verify we can still scan files normally
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	metrics := &ScanMetrics{}
	files, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if metrics.HashedFiles != 1 {
		t.Errorf("Expected 1 hashed file, got %d", metrics.HashedFiles)
	}
}

// TestIntegration_VersionMismatchHandling verifies handling of cache version mismatches
func TestIntegration_VersionMismatchHandling(t *testing.T) {
	// Setup: Create temporary repository directory
	tmpDir := t.TempDir()

	// Create cache file with different version
	cachePath := filepath.Join(tmpDir, CacheFileName)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// oldCache struct is intentionally unused - we use raw JSON instead to simulate
	// a cache file with a different version that we don't know how to deserialize

	data := []byte(`{"version":999,"last_analysis":"0001-01-01T00:00:00Z","git_commit":"","files":{},"agents":{}}`)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write old cache: %v", err)
	}

	// Load cache should handle version mismatch and return new cache
	cache, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache should not fail on version mismatch, got: %v", err)
	}

	// Should return a fresh cache with current version
	if cache.Version != CacheVersion {
		t.Errorf("Expected cache version %d, got %d", CacheVersion, cache.Version)
	}

	if len(cache.Files) != 0 {
		t.Errorf("Expected empty cache after version mismatch, got %d files", len(cache.Files))
	}
}
