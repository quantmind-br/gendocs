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

// AnalysisCache holds information about previous analysis runs
type AnalysisCache struct {
	Version      int                     `json:"version"`
	LastAnalysis time.Time               `json:"last_analysis"`
	GitCommit    string                  `json:"git_commit"`
	Files        map[string]FileInfo     `json:"files"`
	Agents       map[string]AgentStatus  `json:"agents"`
}

// FileInfo holds information about a file
type FileInfo struct {
	Hash     string    `json:"hash"`
	Modified time.Time `json:"modified"`
	Size     int64     `json:"size"`
}

// AgentStatus holds information about an agent's last run
type AgentStatus struct {
	LastRun       time.Time `json:"last_run"`
	Success       bool      `json:"success"`
	FilesAnalyzed []string  `json:"files_analyzed,omitempty"`
}

// ScanMetrics holds statistics about file scanning operations
type ScanMetrics struct {
	TotalFiles  int // Total number of files scanned
	CachedFiles int // Number of files that reused cached hashes
	HashedFiles int // Number of files that required new hash computation
}

// hashFileJob represents a file hashing job with its path and result
type hashFileJob struct {
	relPath string // Relative path of the file to hash
	fullPath string // Absolute path of the file to hash
}

// hashFileResult holds the result of hashing a single file
type hashFileResult struct {
	relPath string // Relative path of the file
	hash    string // Computed hash
	err     error  // Error if hashing failed
}

// DefaultMaxHashWorkers is the default maximum number of parallel hash workers
const DefaultMaxHashWorkers = 8

// getMaxHashWorkers returns the optimal number of hash workers based on CPU count
func getMaxHashWorkers() int {
	numCPU := runtime.NumCPU()
	// Limit to 8 workers to avoid overwhelming the filesystem
	if numCPU > DefaultMaxHashWorkers {
		return DefaultMaxHashWorkers
	}
	return numCPU
}

// parallelHashFiles computes hashes for multiple files concurrently using a worker pool
func parallelHashFiles(jobs []hashFileJob) map[string]string {
	if len(jobs) == 0 {
		return make(map[string]string)
	}

	numWorkers := getMaxHashWorkers()
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

	data, err := json.MarshalIndent(c, "", "  ")
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

// ScanFiles scans repository files and returns their info
// If cache is provided, it will skip hashing files whose mtime and size haven't changed
// If metrics is provided, it will populate statistics about cache hits and hash computations
func ScanFiles(repoPath string, ignorePatterns []string, cache *AnalysisCache, metrics *ScanMetrics) (map[string]FileInfo, error) {
	// Phase 1: Walk directory tree and collect file metadata (no hashing yet)
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
	files := make(map[string]FileInfo)
	var filesToHash []hashFileJob

	for _, meta := range filesToProcess {
		// Check if we can reuse cached hash
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
		if !cached {
			filesToHash = append(filesToHash, hashFileJob{
				relPath:  meta.relPath,
				fullPath: meta.fullPath,
			})
		}

		// Initialize file entry (hash will be filled in after parallel hashing)
		files[meta.relPath] = FileInfo{
			Hash:     hash,
			Modified: meta.modTime,
			Size:     meta.size,
		}
	}

	// Phase 3: Batch hash all files that need it in parallel
	if len(filesToHash) > 0 {
		hashResults := parallelHashFiles(filesToHash)

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
