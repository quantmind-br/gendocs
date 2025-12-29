// Package cache provides file scanning and caching functionality for code analysis.
//
// The cache package implements an optimized file scanning system that uses two key optimizations:
//
// 1. Selective Hashing: Files are only re-hashed if their modification time (mtime) and size
//    have changed since the last scan. This significantly reduces I/O and CPU overhead for
//    incremental analysis where most files haven't changed.
//
// 2. Parallel Hashing: When files do need hashing, they are processed concurrently using a
//    worker pool pattern. This takes advantage of multi-core CPUs to speed up the CPU-bound
//    hash computation.
//
// The combination of these optimizations can provide 3-5x speedup for incremental scans on
// large repositories with many unchanged files.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// CacheVersion is the current cache format version
const CacheVersion = 1

// CacheFileName is the name of the cache file
const CacheFileName = ".ai/analysis_cache.json"

// AnalysisCache holds information about previous analysis runs.
//
// The cache stores file metadata (hash, modification time, size) to enable selective hashing.
// On subsequent scans, files with matching mtime and size can skip re-computation of their SHA256 hash.
// This is particularly effective for incremental analysis where only a small subset of files change.
type AnalysisCache struct {
	Version      int                     `json:"version"`
	LastAnalysis time.Time               `json:"last_analysis"`
	GitCommit    string                  `json:"git_commit"`
	Files        map[string]FileInfo     `json:"files"`
	Agents       map[string]AgentStatus  `json:"agents"`
}

// FileInfo holds metadata about a file for change detection.
//
// The combination of Hash, Modified (mtime), and Size allows us to detect file changes
// without re-reading file contents. A file is considered unchanged if both Modified and Size
// match the cached values, allowing us to skip expensive hash computation.
type FileInfo struct {
	Hash     string    `json:"hash"`     // SHA256 hash of file contents
	Modified time.Time `json:"modified"` // File modification time (mtime)
	Size     int64     `json:"size"`     // File size in bytes
}

// AgentStatus holds information about an agent's last run.
//
// This is used to track which agents have successfully completed and which need to run
// when changes are detected.
type AgentStatus struct {
	LastRun       time.Time `json:"last_run"`
	Success       bool      `json:"success"`
	FilesAnalyzed []string  `json:"files_analyzed,omitempty"`
}

// ScanMetrics holds statistics about file scanning operations.
//
// These metrics help track the effectiveness of selective hashing optimization:
// - CachedFiles: Files that skipped hashing due to cache hits (mtime+size match)
// - HashedFiles: Files that required new hash computation (cache misses)
// - A high cache hit rate (CachedFiles / TotalFiles) indicates effective optimization
type ScanMetrics struct {
	TotalFiles  int // Total number of files scanned
	CachedFiles int // Number of files that reused cached hashes (cache hits)
	HashedFiles int // Number of files that required new hash computation (cache misses)
}

// hashFileJob represents a file hashing job in the parallel processing system.
//
// This struct holds the file path information needed to compute a SHA256 hash.
// Jobs are distributed to worker goroutines via a buffered channel.
type hashFileJob struct {
	relPath  string // Relative path of the file to hash (used as the result map key)
	fullPath string // Absolute path of the file to hash (used for file I/O)
}

// hashFileResult holds the result of hashing a single file.
//
// Workers send results through this struct to a results channel, allowing the main
// goroutine to collect all hash computations without blocking.
type hashFileResult struct {
	relPath string // Relative path of the file (for mapping back to the file list)
	hash    string // Computed SHA256 hash (empty if error occurred)
	err     error  // Error if hashing failed (nil on success)
}

// fileMetadata holds file path and metadata collected during directory traversal.
//
// This struct is used internally by ScanFiles to collect file information before
// deciding which files need hash computation.
type fileMetadata struct {
	relPath  string    // Relative path from repository root
	fullPath string    // Absolute path to the file
	modTime  time.Time // File modification time
	size     int64     // File size in bytes
}

// DefaultMaxHashWorkers is the default maximum number of parallel hash workers.
//
// This constant provides a safety cap to prevent overwhelming the filesystem with
// too many concurrent I/O operations. Even on systems with many CPU cores, we limit
// parallelism to avoid excessive disk contention.
const DefaultMaxHashWorkers = 8

// getMaxHashWorkers returns the optimal number of hash workers based on CPU count and configured limit.
//
// This function implements a smart worker count strategy:
// - If maxHashWorkers is 0: Use CPU count, capped at DefaultMaxHashWorkers (auto-detect mode)
// - If maxHashWorkers > 0: Use configured value, capped at DefaultMaxHashWorkers (explicit mode)
//
// The cap ensures we don't overwhelm the filesystem with too many parallel reads,
// which could actually degrade performance due to I/O contention.
//
// Parameters:
//   - maxHashWorkers: Configured maximum (0 for auto-detect, >0 for explicit)
//
// Returns:
//   - The optimal number of workers to use (capped at DefaultMaxHashWorkers)
func getMaxHashWorkers(maxHashWorkers int) int {
	numCPU := runtime.NumCPU()

	// If explicit configuration is provided, use it (with safety cap)
	if maxHashWorkers > 0 {
		if maxHashWorkers > DefaultMaxHashWorkers {
			return DefaultMaxHashWorkers
		}
		return maxHashWorkers
	}

	// Default: use CPU count, capped at DefaultMaxHashWorkers
	if numCPU > DefaultMaxHashWorkers {
		return DefaultMaxHashWorkers
	}
	return numCPU
}

// parallelHashFiles computes hashes for multiple files concurrently using a worker pool pattern.
//
// This function implements the parallel hashing optimization that significantly speeds up
// file scanning on multi-core systems. The worker pool design ensures:
//
// 1. Concurrent Execution: Multiple goroutines compute hashes in parallel, utilizing all CPU cores
// 2. Bounded Parallelism: Worker count is capped to avoid overwhelming the filesystem with I/O
// 3. Clean Shutdown: Uses sync.WaitGroup to ensure all workers complete before returning
// 4. Non-Blocking: Uses buffered channels for job distribution and result collection
//
// Architecture:
//   - Worker Pool: Fixed number of goroutines that process jobs from a shared queue
//   - Job Queue: Buffered channel holding hashFileJob objects (one per file to hash)
//   - Results Channel: Buffered channel collecting hashFileResult objects as workers finish
//
// Workflow:
//   1. Create worker pool (determined by getMaxHashWorkers())
//   2. Dispatch all jobs to the job queue
//   3. Workers pull jobs, compute hashes, send results
//   4. Close job queue and wait for all workers to finish
//   5. Collect results from the results channel into a map
//
// Parameters:
//   - jobs: Slice of file hashing jobs to process
//   - maxHashWorkers: Maximum number of workers (0 = auto-detect CPU count, capped at 8)
//
// Returns:
//   - Map of relative file paths to their SHA256 hashes (excludes files that errored)
func parallelHashFiles(jobs []hashFileJob, maxHashWorkers int) map[string]string {
	if len(jobs) == 0 {
		return make(map[string]string)
	}

	numWorkers := getMaxHashWorkers(maxHashWorkers)
	results := make(map[string]string)
	resultsChan := make(chan hashFileResult, len(jobs))
	var wg sync.WaitGroup

	// Create worker pool
	jobQueue := make(chan hashFileJob, len(jobs))

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobQueue {
				hash, err := HashFile(job.fullPath)
				resultsChan <- hashFileResult{
					relPath: job.relPath,
					hash:    hash,
					err:     err,
				}
			}
		}()
	}

	// Dispatch jobs
	for _, job := range jobs {
		jobQueue <- job
	}
	close(jobQueue)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		if result.err == nil {
			results[result.relPath] = result.hash
		}
	}

	return results
}

// ChangeReport describes what changed since last analysis
type ChangeReport struct {
	HasChanges       bool
	NewFiles         []string
	ModifiedFiles    []string
	DeletedFiles     []string
	AgentsToRun      []string
	AgentsToSkip     []string
	Reason           string
	IsFirstRun       bool
	GitCommitChanged bool
}

// AgentFilePatterns maps agents to file patterns they care about
var AgentFilePatterns = map[string][]string{
	"structure_analyzer": {
		"*.go", "*.py", "*.js", "*.ts", "*.jsx", "*.tsx",
		"*.java", "*.rs", "*.c", "*.cpp", "*.h", "*.hpp",
		"go.mod", "package.json", "Cargo.toml", "pom.xml",
	},
	"dependency_analyzer": {
		"go.mod", "go.sum", "package.json", "package-lock.json",
		"yarn.lock", "pnpm-lock.yaml", "Cargo.toml", "Cargo.lock",
		"requirements.txt", "pyproject.toml", "pom.xml", "build.gradle",
	},
	"data_flow_analyzer": {
		"*.go", "*.py", "*.js", "*.ts", "*.jsx", "*.tsx",
		"*.java", "*.rs",
	},
	"request_flow_analyzer": {
		"*handler*.go", "*controller*.go", "*route*.go", "*api*.go",
		"*handler*.py", "*view*.py", "*route*.py",
		"*controller*.js", "*route*.js", "*api*.js",
		"*controller*.ts", "*route*.ts", "*api*.ts",
	},
	"api_analyzer": {
		"*handler*.go", "*controller*.go", "*route*.go", "*api*.go",
		"*handler*.py", "*view*.py", "*route*.py",
		"*controller*.js", "*route*.js", "*api*.js",
		"*controller*.ts", "*route*.ts", "*api*.ts",
		"openapi*.yaml", "swagger*.yaml", "*.proto",
	},
}

// NewCache creates a new empty cache
func NewCache() *AnalysisCache {
	return &AnalysisCache{
		Version: CacheVersion,
		Files:   make(map[string]FileInfo),
		Agents:  make(map[string]AgentStatus),
	}
}

// LoadCache loads the cache from disk
func LoadCache(repoPath string) (*AnalysisCache, error) {
	cachePath := filepath.Join(repoPath, CacheFileName)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCache(), nil
		}
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var cache AnalysisCache
	if err := json.Unmarshal(data, &cache); err != nil {
		// Cache is corrupted, return fresh cache
		return NewCache(), nil
	}

	// Check version compatibility
	if cache.Version != CacheVersion {
		// Version mismatch, return fresh cache
		return NewCache(), nil
	}

	return &cache, nil
}

// Save saves the cache to disk
func (c *AnalysisCache) Save(repoPath string) error {
	cachePath := filepath.Join(repoPath, CacheFileName)

	// Ensure directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Use json.Marshal instead of json.MarshalIndent for better performance
	// The cache file is machine-readable, so pretty-printing adds unnecessary
	// file size and encoding overhead. json.Unmarshal is indentation-agnostic.
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// GetCurrentGitCommit returns the current git commit hash
func GetCurrentGitCommit(repoPath string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// HashFile calculates SHA256 hash of a file
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ScanFiles scans repository files and returns their information using selective hashing and parallel processing.
//
// This function is the core of the optimized file scanning system, combining two key optimizations:
//
// 1. SELECTIVE HASHING (Cache Hit Detection):
//    - For each file, compare its mtime and size against the cached values
//    - If both match: Skip hash computation, reuse cached hash (cache HIT)
//    - If either differs: Mark file for hashing (cache MISS)
//    - This dramatically reduces I/O for incremental scans where most files are unchanged
//
// 2. PARALLEL HASHING (Worker Pool):
//    - All files marked as cache misses are hashed concurrently
//    - Uses a worker pool pattern with bounded parallelism
//    - Separates I/O-bound directory walking from CPU-bound hash computation
//
// Three-Phase Architecture:
//
//   Phase 1 - File Discovery (I/O Bound):
//     - Walk the directory tree using filepath.Walk()
//     - Collect file metadata (path, mtime, size) for all files
//     - No hash computation yet, just metadata gathering
//     - This allows us to make cache hit/miss decisions for all files upfront
//
//   Phase 2 - Cache Classification (In-Memory):
//     - For each file, check if cached metadata exists
//     - Compare mtime and size to determine cache hit vs miss
//     - Separate files into two groups:
//       * Cached: Reuse hash, update metrics
//       * Needs Hashing: Add to parallel hashing batch
//     - Build initial results map with cached hashes and placeholders
//
//   Phase 3 - Parallel Hashing (CPU Bound):
//     - Batch hash all "needs hashing" files using parallelHashFiles()
//     - Worker pool computes multiple hashes concurrently
//     - Update the results map with computed hashes
//     - This is where the CPU-intensive work happens in parallel
//
// Performance Characteristics:
//   - Best Case (all files cached): O(n) directory walk, no hash computations
//   - Worst Case (all files changed): O(n) directory walk + parallel hash computation
//   - Typical Case (mix): O(n) walk + parallel hash for subset of files
//   - Parallel hashing provides near-linear speedup with CPU cores
//
// Parameters:
//   - repoPath: Root directory of the repository to scan
//   - ignorePatterns: File/directory patterns to skip (e.g., ".git", "node_modules")
//   - cache: Optional analysis cache (nil = no cache, compute all hashes)
//   - metrics: Optional metrics tracker (nil = don't track statistics)
//   - maxHashWorkers: Maximum parallel hash workers (0 = auto-detect CPU count, capped at 8)
//
// Returns:
//   - map[string]FileInfo: Map of relative file paths to their metadata (hash, mtime, size)
//   - error: Error if directory walk fails
//
// Example Usage:
//
//   // With cache and metrics
//   cache, _ := LoadCache(repoPath)
//   var metrics ScanMetrics
//   files, err := ScanFiles(repoPath, nil, cache, &metrics, 0)
//   fmt.Printf("Cache hit rate: %.1f%%\n",
//       float64(metrics.CachedFiles)/float64(metrics.TotalFiles)*100)
//
//   // Without cache (backward compatible)
//   files, err := ScanFiles(repoPath, nil, nil, nil, 0)
func ScanFiles(repoPath string, ignorePatterns []string, cache *AnalysisCache, metrics *ScanMetrics, maxHashWorkers int) (map[string]FileInfo, error) {
	// Phase 1: Walk directory tree and collect file metadata (no hashing yet)
	//
	// This phase is I/O bound as we read directory entries and file metadata from disk.
	// We deliberately avoid any hash computation here to minimize I/O overhead.
	// The fileMetadata struct allows us to collect all info needed for cache decisions.
	type fileMetadata struct {
		relPath  string
		fullPath string
		modTime  time.Time
		size     int64
	}

	var filesToProcess []fileMetadata
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}

		// Skip directories and apply ignore patterns
		if info.IsDir() {
			if shouldIgnore(relPath, info.Name(), ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored files
		if shouldIgnore(relPath, info.Name(), ignorePatterns) {
			return nil
		}

		// Skip binary files (quick check by extension)
		if isBinaryExtension(filepath.Ext(path)) {
			return nil
		}

		// Track total files
		if metrics != nil {
			metrics.TotalFiles++
		}

		// Collect file metadata for processing
		filesToProcess = append(filesToProcess, fileMetadata{
			relPath:  relPath,
			fullPath: path,
			modTime:  info.ModTime(),
			size:     info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Phase 2: Separate files into cached and needs-hashing groups
	//
	// This phase examines each file discovered in Phase 1 and determines whether we can
	// reuse the cached hash or need to compute a new one. The decision is based on:
	//   1. Does the file exist in the cache?
	//   2. If yes, do both mtime AND size match exactly?
	//
	// Files that pass both tests are cache hits and skip hash computation.
	// Files that fail either test are cache misses and are added to the parallel hashing batch.
	//
	// We build the results map incrementally:
	//   - Cache hits: Add immediately with their cached hash
	//   - Cache misses: Add now with empty hash, will be filled in Phase 3
	files := make(map[string]FileInfo)
	var filesToHash []hashFileJob

	for _, meta := range filesToProcess {
		// Check if we can reuse cached hash
		//
		// Cache hit condition (both must be true):
		//   1. File exists in cache
		//   2. Cached mtime equals current mtime (using Equal() for time.Time comparison)
		//   3. Cached size equals current size
		//
		// We require BOTH mtime and size to match because:
		//   - mtime alone can have false positives (e.g., file touched without modification)
		//   - size alone can have false positives (e.g., file changed but same size)
		//   - Combined, they provide a very reliable change detection heuristic
		var hash string
		cached := false
		if cache != nil {
			if cachedFile, exists := cache.Files[meta.relPath]; exists {
				// If mtime and size match, reuse the cached hash
				if cachedFile.Modified.Equal(meta.modTime) && cachedFile.Size == meta.size {
					hash = cachedFile.Hash
					cached = true
				}
			}
		}

		// Track metrics for cached files
		if metrics != nil && cached {
			metrics.CachedFiles++
		}

		// If not cached, add to parallel hashing batch
		//
		// Files that fail the cache hit test are collected into a batch for parallel hashing.
		// This batching allows us to compute multiple hashes concurrently, significantly
		// speeding up the process on multi-core systems.
		if !cached {
			filesToHash = append(filesToHash, hashFileJob{
				relPath:  meta.relPath,
				fullPath: meta.fullPath,
			})
		}

		// Initialize file entry (hash will be filled in after parallel hashing)
		//
		// We build the final results map in Phase 2, but cache misses get an empty hash
		// that will be filled in during Phase 3. This allows us to have a single results
		// map that gets progressively filled in, rather than merging multiple maps.
		files[meta.relPath] = FileInfo{
			Hash:     hash,
			Modified: meta.modTime,
			Size:     meta.size,
		}
	}

	// Phase 3: Batch hash all files that need it in parallel
	//
	// This is the CPU-intensive phase where we compute SHA256 hashes for all cache misses.
	// The parallelHashFiles() function implements a worker pool that:
	//   - Spawns multiple worker goroutines (limited by maxHashWorkers)
	//   - Distributes file hashing jobs across workers
	//   - Collects results into a map as workers complete
	//
	// After parallel hashing completes, we update the files map with the computed hashes.
	// For any hash that failed (error), we leave it as empty string - that file will be
	// treated as missing/errored in subsequent analysis.
	//
	// The batching approach ensures that we maximize parallelism while avoiding
	// overwhelming the filesystem with too many concurrent reads.
	if len(filesToHash) > 0 {
		hashResults := parallelHashFiles(filesToHash, maxHashWorkers)

		// Update file entries with computed hashes
		for relPath, hash := range hashResults {
			if file, exists := files[relPath]; exists {
				file.Hash = hash
				files[relPath] = file
			}

			// Track metrics for hashed files
			if metrics != nil {
				metrics.HashedFiles++
			}
		}
	}

	return files, nil
}

// DetectChanges compares current files with cached files
func (c *AnalysisCache) DetectChanges(repoPath string, currentFiles map[string]FileInfo) *ChangeReport {
	report := &ChangeReport{
		NewFiles:      []string{},
		ModifiedFiles: []string{},
		DeletedFiles:  []string{},
		AgentsToRun:   []string{},
		AgentsToSkip:  []string{},
	}

	// Check if this is first run
	if len(c.Files) == 0 || c.LastAnalysis.IsZero() {
		report.IsFirstRun = true
		report.HasChanges = true
		report.Reason = "First analysis run"
		report.AgentsToRun = getAllAgents()
		for path := range currentFiles {
			report.NewFiles = append(report.NewFiles, path)
		}
		return report
	}

	// Check git commit
	currentCommit := GetCurrentGitCommit(repoPath)
	if currentCommit != "" && c.GitCommit != "" && currentCommit != c.GitCommit {
		report.GitCommitChanged = true
	}

	// Find new and modified files
	for path, info := range currentFiles {
		cached, exists := c.Files[path]
		if !exists {
			report.NewFiles = append(report.NewFiles, path)
		} else if cached.Hash != info.Hash {
			report.ModifiedFiles = append(report.ModifiedFiles, path)
		}
	}

	// Find deleted files
	for path := range c.Files {
		if _, exists := currentFiles[path]; !exists {
			report.DeletedFiles = append(report.DeletedFiles, path)
		}
	}

	// Determine which agents need to run
	changedFiles := append(report.NewFiles, report.ModifiedFiles...)
	changedFiles = append(changedFiles, report.DeletedFiles...)

	report.HasChanges = len(changedFiles) > 0

	if !report.HasChanges {
		report.Reason = "No files changed since last analysis"
		report.AgentsToSkip = getAllAgents()
		return report
	}

	// Determine which agents are affected by the changes
	for agent, patterns := range AgentFilePatterns {
		if agentNeedsRun(changedFiles, patterns, c.Agents[agent]) {
			report.AgentsToRun = append(report.AgentsToRun, agent)
		} else {
			report.AgentsToSkip = append(report.AgentsToSkip, agent)
		}
	}

	// If no specific agents matched, run all (safety fallback)
	if len(report.AgentsToRun) == 0 && report.HasChanges {
		report.AgentsToRun = getAllAgents()
		report.AgentsToSkip = []string{}
		report.Reason = "Changes detected but no specific agent patterns matched"
	} else {
		report.Reason = fmt.Sprintf("%d files changed, %d agents need re-run",
			len(changedFiles), len(report.AgentsToRun))
	}

	return report
}

// UpdateAfterAnalysis updates the cache after a successful analysis
func (c *AnalysisCache) UpdateAfterAnalysis(repoPath string, currentFiles map[string]FileInfo, agentResults map[string]bool) {
	c.LastAnalysis = time.Now()
	c.GitCommit = GetCurrentGitCommit(repoPath)
	c.Files = currentFiles

	for agent, success := range agentResults {
		c.Agents[agent] = AgentStatus{
			LastRun: time.Now(),
			Success: success,
		}
	}
}

// Helper functions

func getAllAgents() []string {
	return []string{
		"structure_analyzer",
		"dependency_analyzer",
		"data_flow_analyzer",
		"request_flow_analyzer",
		"api_analyzer",
	}
}

func agentNeedsRun(changedFiles []string, patterns []string, lastStatus AgentStatus) bool {
	// If agent never ran successfully, it needs to run
	if lastStatus.LastRun.IsZero() || !lastStatus.Success {
		return true
	}

	// Check if any changed file matches agent's patterns
	for _, file := range changedFiles {
		for _, pattern := range patterns {
			if matchPattern(file, pattern) {
				return true
			}
		}
	}

	return false
}

func matchPattern(filename, pattern string) bool {
	// Handle patterns like "*handler*.go"
	if strings.Contains(pattern, "*") {
		// Simple glob matching
		pattern = strings.ToLower(pattern)
		filename = strings.ToLower(filepath.Base(filename))

		// Handle *.ext patterns
		if strings.HasPrefix(pattern, "*.") {
			ext := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(filename, ext)
		}

		// Handle *keyword*.ext patterns
		if strings.HasPrefix(pattern, "*") && strings.Contains(pattern[1:], "*") {
			parts := strings.Split(pattern, "*")
			for _, part := range parts {
				if part != "" && !strings.Contains(filename, part) {
					return false
				}
			}
			return true
		}

		// Handle *keyword* patterns
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			keyword := strings.Trim(pattern, "*")
			return strings.Contains(filename, keyword)
		}
	}

	// Exact match (for files like go.mod)
	return strings.ToLower(filepath.Base(filename)) == strings.ToLower(pattern)
}

func shouldIgnore(relPath, name string, patterns []string) bool {
	// Default ignore patterns
	defaultIgnore := []string{
		".git", "node_modules", "vendor", ".venv", "venv",
		"__pycache__", "dist", "build", ".ai",
	}

	for _, pattern := range append(defaultIgnore, patterns...) {
		if name == pattern || strings.HasPrefix(relPath, pattern+"/") {
			return true
		}
	}

	return false
}

var binaryExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".o": true, ".a": true, ".obj": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".ico": true, ".bmp": true, ".webp": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true,
	".pdf": true, ".doc": true, ".docx": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
}

func isBinaryExtension(ext string) bool {
	return binaryExts[strings.ToLower(ext)]
}