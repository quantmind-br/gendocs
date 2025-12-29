package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestScanFiles_CacheHit verifies that files with unchanged mtime and size reuse cached hashes
func TestScanFiles_CacheHit(t *testing.T) {
	// Setup: Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create test file with known content
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get file metadata
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	modTime := info.ModTime()
	fileSize := info.Size()

	// Compute expected hash
	expectedHash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Create cache with the file entry
	cache := NewCache()
	cache.Files["test.go"] = FileInfo{
		Hash:     expectedHash,
		Modified: modTime,
		Size:     fileSize,
	}

	// Scan with cache
	metrics := &ScanMetrics{}
	files, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify file info matches
	fileInfo, exists := files["test.go"]
	if !exists {
		t.Fatal("File not found in scan results")
	}

	if fileInfo.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, fileInfo.Hash)
	}

	if !fileInfo.Modified.Equal(modTime) {
		t.Errorf("Expected modTime %v, got %v", modTime, fileInfo.Modified)
	}

	if fileInfo.Size != fileSize {
		t.Errorf("Expected size %d, got %d", fileSize, fileInfo.Size)
	}

	// Verify cache was used (file should be cached, not hashed)
	if metrics.CachedFiles != 1 {
		t.Errorf("Expected 1 cached file, got %d", metrics.CachedFiles)
	}

	if metrics.HashedFiles != 0 {
		t.Errorf("Expected 0 hashed files, got %d", metrics.HashedFiles)
	}

	if metrics.TotalFiles != 1 {
		t.Errorf("Expected 1 total file, got %d", metrics.TotalFiles)
	}
}

// TestScanFiles_CacheMissDifferentMTime verifies that files with different mtime get rehashed
func TestScanFiles_CacheMissDifferentMTime(t *testing.T) {
	// Setup: Create temporary directory with test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get file metadata
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	fileSize := info.Size()

	// Compute current hash
	expectedHash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Create cache with OLD modtime (1 hour in the past)
	oldModTime := info.ModTime().Add(-1 * time.Hour)
	cache := NewCache()
	cache.Files["test.go"] = FileInfo{
		Hash:     "old-hash",
		Modified: oldModTime,
		Size:     fileSize,
	}

	// Scan with cache
	metrics := &ScanMetrics{}
	files, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify file was rehashed
	fileInfo, exists := files["test.go"]
	if !exists {
		t.Fatal("File not found in scan results")
	}

	if fileInfo.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, fileInfo.Hash)
	}

	if fileInfo.Modified.Equal(oldModTime) {
		t.Error("Expected modTime to be updated to current time")
	}

	// Verify cache was NOT used (file should be rehashed)
	if metrics.CachedFiles != 0 {
		t.Errorf("Expected 0 cached files, got %d", metrics.CachedFiles)
	}

	if metrics.HashedFiles != 1 {
		t.Errorf("Expected 1 hashed file, got %d", metrics.HashedFiles)
	}
}

// TestScanFiles_CacheMissDifferentSize verifies that files with different size get rehashed
func TestScanFiles_CacheMissDifferentSize(t *testing.T) {
	// Setup: Create temporary directory with test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get file metadata
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	modTime := info.ModTime()

	// Compute current hash
	expectedHash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Create cache with WRONG size (different from actual)
	wrongSize := info.Size() + 100
	cache := NewCache()
	cache.Files["test.go"] = FileInfo{
		Hash:     "old-hash",
		Modified: modTime,
		Size:     wrongSize,
	}

	// Scan with cache
	metrics := &ScanMetrics{}
	files, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify file was rehashed
	fileInfo, exists := files["test.go"]
	if !exists {
		t.Fatal("File not found in scan results")
	}

	if fileInfo.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, fileInfo.Hash)
	}

	if fileInfo.Size != info.Size() {
		t.Errorf("Expected size %d, got %d", info.Size(), fileInfo.Size)
	}

	// Verify cache was NOT used (file should be rehashed)
	if metrics.CachedFiles != 0 {
		t.Errorf("Expected 0 cached files, got %d", metrics.CachedFiles)
	}

	if metrics.HashedFiles != 1 {
		t.Errorf("Expected 1 hashed file, got %d", metrics.HashedFiles)
	}
}

// TestScanFiles_NewFiles verifies that files not in cache get hashed
func TestScanFiles_NewFiles(t *testing.T) {
	// Setup: Create temporary directory with test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute expected hash
	expectedHash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Create empty cache (no files)
	cache := NewCache()

	// Scan with cache
	metrics := &ScanMetrics{}
	files, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify file was hashed
	fileInfo, exists := files["test.go"]
	if !exists {
		t.Fatal("File not found in scan results")
	}

	if fileInfo.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, fileInfo.Hash)
	}

	// Verify file was hashed (not cached)
	if metrics.CachedFiles != 0 {
		t.Errorf("Expected 0 cached files, got %d", metrics.CachedFiles)
	}

	if metrics.HashedFiles != 1 {
		t.Errorf("Expected 1 hashed file, got %d", metrics.HashedFiles)
	}
}

// TestScanFiles_DeletedFiles verifies that files in cache but not on disk are handled correctly
func TestScanFiles_DeletedFiles(t *testing.T) {
	// Setup: Create temporary directory
	tmpDir := t.TempDir()

	// Create cache with a file that doesn't exist on disk
	cache := NewCache()
	cache.Files["deleted.go"] = FileInfo{
		Hash:     "some-hash",
		Modified: time.Now(),
		Size:     100,
	}

	// Scan with cache (empty directory)
	metrics := &ScanMetrics{}
	files, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify deleted file is not in scan results
	if _, exists := files["deleted.go"]; exists {
		t.Error("Deleted file should not be in scan results")
	}

	// Verify no files were processed
	if metrics.TotalFiles != 0 {
		t.Errorf("Expected 0 total files, got %d", metrics.TotalFiles)
	}

	if metrics.CachedFiles != 0 {
		t.Errorf("Expected 0 cached files, got %d", metrics.CachedFiles)
	}

	if metrics.HashedFiles != 0 {
		t.Errorf("Expected 0 hashed files, got %d", metrics.HashedFiles)
	}
}

// TestScanFiles_MultipleFilesMixedCache verifies selective hashing with multiple files
func TestScanFiles_MultipleFilesMixedCache(t *testing.T) {
	// Setup: Create temporary directory with multiple test files
	tmpDir := t.TempDir()

	// Create three test files
	files := map[string]string{
		"cached.go":  "package cached\n",
		"changed.go": "package changed\n",
		"newfile.go": "package newfile\n",
	}

	var expectedHashes map[string]string = make(map[string]string)
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}

		hash, err := HashFile(path)
		if err != nil {
			t.Fatalf("Failed to hash %s: %v", name, err)
		}
		expectedHashes[name] = hash
	}

	// Get file metadata
	cachedInfo, _ := os.Stat(filepath.Join(tmpDir, "cached.go"))
	changedInfo, _ := os.Stat(filepath.Join(tmpDir, "changed.go"))

	// Create cache:
	// - cached.go: same mtime and size (should use cache)
	// - changed.go: different mtime (should rehash)
	// - newfile.go: not in cache (should hash)
	cache := NewCache()
	cache.Files["cached.go"] = FileInfo{
		Hash:     expectedHashes["cached.go"],
		Modified: cachedInfo.ModTime(),
		Size:     cachedInfo.Size(),
	}
	cache.Files["changed.go"] = FileInfo{
		Hash:     "old-hash",
		Modified: changedInfo.ModTime().Add(-1 * time.Hour),
		Size:     changedInfo.Size(),
	}
	// newfile.go is not in cache

	// Scan with cache
	metrics := &ScanMetrics{}
	scanResults, err := ScanFiles(tmpDir, []string{}, cache, metrics, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify all files are in results
	if len(scanResults) != 3 {
		t.Errorf("Expected 3 files, got %d", len(scanResults))
	}

	// Verify cached.go used cached hash
	if scanResults["cached.go"].Hash != expectedHashes["cached.go"] {
		t.Errorf("cached.go: expected hash %s, got %s",
			expectedHashes["cached.go"], scanResults["cached.go"].Hash)
	}

	// Verify changed.go was rehashed
	if scanResults["changed.go"].Hash != expectedHashes["changed.go"] {
		t.Errorf("changed.go: expected hash %s, got %s",
			expectedHashes["changed.go"], scanResults["changed.go"].Hash)
	}

	// Verify newfile.go was hashed
	if scanResults["newfile.go"].Hash != expectedHashes["newfile.go"] {
		t.Errorf("newfile.go: expected hash %s, got %s",
			expectedHashes["newfile.go"], scanResults["newfile.go"].Hash)
	}

	// Verify metrics
	if metrics.TotalFiles != 3 {
		t.Errorf("Expected 3 total files, got %d", metrics.TotalFiles)
	}

	if metrics.CachedFiles != 1 {
		t.Errorf("Expected 1 cached file, got %d", metrics.CachedFiles)
	}

	if metrics.HashedFiles != 2 {
		t.Errorf("Expected 2 hashed files, got %d", metrics.HashedFiles)
	}
}

// TestScanFiles_NoCache verifies that ScanFiles works correctly without cache
func TestScanFiles_NoCache(t *testing.T) {
	// Setup: Create temporary directory with test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute expected hash
	expectedHash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Scan WITHOUT cache (pass nil)
	files, err := ScanFiles(tmpDir, []string{}, nil, nil, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify file was hashed
	fileInfo, exists := files["test.go"]
	if !exists {
		t.Fatal("File not found in scan results")
	}

	if fileInfo.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, fileInfo.Hash)
	}
}

// TestScanFiles_WithIgnorePatterns verifies that ignore patterns are respected
func TestScanFiles_WithIgnorePatterns(t *testing.T) {
	// Setup: Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create files
	files := map[string]string{
		"main.go":       "package main\n",
		"vendor/lib.go": "package vendor\n",
		"test_test.go":  "package test\n",
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

	// Scan with ignore patterns
	ignorePatterns := []string{"vendor", "*_test.go"}
	scannedFiles, err := ScanFiles(tmpDir, ignorePatterns, nil, nil, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Verify only main.go is in results
	if len(scannedFiles) != 1 {
		t.Errorf("Expected 1 file, got %d", len(scannedFiles))
	}

	if _, exists := scannedFiles["main.go"]; !exists {
		t.Error("main.go should be in results")
	}

	if _, exists := scannedFiles["vendor/lib.go"]; exists {
		t.Error("vendor/lib.go should be ignored")
	}

	if _, exists := scannedFiles["test_test.go"]; exists {
		t.Error("test_test.go should be ignored")
	}
}

// TestScanFiles_MetricsNil verifies that ScanFiles works when metrics is nil
func TestScanFiles_MetricsNil(t *testing.T) {
	// Setup: Create temporary directory with test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create cache
	cache := NewCache()

	// Scan with nil metrics (should not panic)
	files, err := ScanFiles(tmpDir, []string{}, cache, nil, 0)
	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

// TestHashFile verifies that HashFile computes correct SHA256 hash
func TestHashFile(t *testing.T) {
	// Setup: Create temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!\n")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Expected SHA256 hash of "Hello, World!\n"
	expectedHash := "d9014c4624844aa5bac3147735bad58b4084bf2e074596a9f7e4e22a348a102b"

	hash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hash)
	}
}

// TestHashFile_NonExistent verifies that HashFile handles non-existent files
func TestHashFile_NonExistent(t *testing.T) {
	nonExistentFile := "/tmp/this-file-does-not-exist-12345.txt"

	_, err := HashFile(nonExistentFile)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
