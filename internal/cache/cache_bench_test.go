package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Benchmark setup helper that creates a realistic test repository
func setupBenchmarkRepo(b *testing.B, numFiles int) (string, map[string]FileInfo) {
	b.Helper()

	tmpDir := b.TempDir()

	// Create a realistic file structure similar to a typical project
	// Source files with various extensions and sizes
	createdFiles := make(map[string]FileInfo)

	// Create directories
	dirs := []string{
		"cmd/server",
		"internal/handlers",
		"internal/models",
		"internal/utils",
		"pkg/api",
		"pkg/config",
		"web/assets",
		"scripts",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			b.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create files with realistic content patterns
	filePatterns := []struct {
		path    string
		content string
		count   int
	}{
		{
			path: "cmd/server/main.go",
			content: `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}
`,
			count: 1,
		},
		{
			path: "internal/handlers/user.go",
			content: `package handlers

type User struct {
	ID    int    ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

func GetUser(id int) (*User, error) {
	// Database query logic
	return &User{ID: id, Name: "John Doe", Email: "john@example.com"}, nil
}

func CreateUser(u *User) error {
	// Create user logic
	return nil
}

func UpdateUser(u *User) error {
	// Update user logic
	return nil
}

func DeleteUser(id int) error {
	// Delete user logic
	return nil
}
`,
			count: 5,
		},
		{
			path: "internal/models/models.go",
			content: `package models

type Model struct {
	ID        int       ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}

func (m *Model) Validate() error {
	if m.ID < 0 {
		return fmt.Errorf("invalid ID")
	}
	return nil
}
`,
			count: 3,
		},
		{
			path: "pkg/api/client.go",
			content: `package api

import "net/http"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Get(endpoint string) (*http.Response, error) {
	return c.HTTPClient.Get(c.BaseURL + endpoint)
}

func (c *Client) Post(endpoint string, body interface{}) (*http.Response, error) {
	return c.HTTPClient.Post(c.BaseURL+endpoint, "application/json", nil)
}
`,
			count: 4,
		},
		{
			path: "pkg/config/config.go",
			content: `package config

type Config struct {
	DatabaseURL string
	ServerPort  int
	Debug       bool
	LogLevel    string
}

func Load() (*Config, error) {
	return &Config{
		DatabaseURL: "postgres://localhost/db",
		ServerPort:  8080,
		Debug:       false,
		LogLevel:    "info",
	}, nil
}
`,
			count: 2,
		},
		{
			path: "scripts/build.sh",
			content: `#!/bin/bash
set -e
echo "Building application..."
go build -o bin/app ./cmd/server
echo "Build complete!"
`,
			count: 3,
		},
		{
			path: "README.md",
			content: `# Project README

## Overview
This is a sample project.

## Installation
` + "```bash" + `
go install ./cmd/server
` + "```" + `

## Usage
Run the server with:
` + "```bash" + `
server --port 8080
` + "```" + `
`,
			count: 1,
		},
	}

	// Create files to reach target count
	fileCount := 0
	for _, pattern := range filePatterns {
		for i := 0; i < pattern.count; i++ {
			if fileCount >= numFiles {
				break
			}

			var filePath string
			if i == 0 {
				filePath = pattern.path
			} else {
				// Create variations of the file
				ext := filepath.Ext(pattern.path)
				base := pattern.path[:len(pattern.path)-len(ext)]
				filePath = fmt.Sprintf("%s_%d%s", base, i, ext)
			}

			fullPath := filepath.Join(tmpDir, filePath)
			if err := os.WriteFile(fullPath, []byte(pattern.content), 0644); err != nil {
				b.Fatalf("Failed to create file %s: %v", filePath, err)
			}

			// Store file info for cache
			info, err := os.Stat(fullPath)
			if err != nil {
				b.Fatalf("Failed to stat file %s: %v", filePath, err)
			}

			hash, err := HashFile(fullPath)
			if err != nil {
				b.Fatalf("Failed to hash file %s: %v", filePath, err)
			}

			createdFiles[filePath] = FileInfo{
				Hash:     hash,
				Modified: info.ModTime(),
				Size:     info.Size(),
			}

			fileCount++
		}
		if fileCount >= numFiles {
			break
		}
	}

	return tmpDir, createdFiles
}

// BenchmarkScanFiles_NoCacheSequential benchmarks baseline performance:
// - No cache (all files need hashing)
// - Sequential hashing (single worker)
// This represents the worst-case scenario before optimizations
func BenchmarkScanFiles_NoCacheSequential(b *testing.B) {
	repoPath, _ := setupBenchmarkRepo(b, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, nil, metrics, 1) // maxHashWorkers=1 for sequential
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}
	}
}

// BenchmarkScanFiles_WithCacheSequential benchmarks selective hashing only:
// - With cache (unchanged files use cached hashes)
// - Sequential hashing (single worker)
// This measures the benefit of selective hashing without parallelism
func BenchmarkScanFiles_WithCacheSequential(b *testing.B) {
	repoPath, cachedFiles := setupBenchmarkRepo(b, 100)

	// Create cache with all files (simulating previous scan)
	cache := NewCache()
	cache.Files = cachedFiles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, cache, metrics, 1) // maxHashWorkers=1 for sequential
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}

		// Verify selective hashing is working (all files should be cached)
		if metrics.CachedFiles != len(cachedFiles) {
			b.Errorf("Expected %d cached files, got %d", len(cachedFiles), metrics.CachedFiles)
		}
		if metrics.HashedFiles != 0 {
			b.Errorf("Expected 0 hashed files, got %d", metrics.HashedFiles)
		}
	}
}

// BenchmarkScanFiles_WithCacheParallel benchmarks selective hashing + parallel processing:
// - With cache (unchanged files use cached hashes)
// - Parallel hashing (default worker count)
// This measures the combined benefit of both optimizations
func BenchmarkScanFiles_WithCacheParallel(b *testing.B) {
	repoPath, cachedFiles := setupBenchmarkRepo(b, 100)

	// Create cache with all files (simulating previous scan)
	cache := NewCache()
	cache.Files = cachedFiles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, cache, metrics, 0) // maxHashWorkers=0 for auto-detect
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}

		// Verify selective hashing is working (all files should be cached)
		if metrics.CachedFiles != len(cachedFiles) {
			b.Errorf("Expected %d cached files, got %d", len(cachedFiles), metrics.CachedFiles)
		}
		if metrics.HashedFiles != 0 {
			b.Errorf("Expected 0 hashed files, got %d", metrics.HashedFiles)
		}
	}
}

// BenchmarkScanFiles_PartialCacheParallel benchmarks realistic incremental scan:
// - With partial cache (some files changed, need rehashing)
// - Parallel hashing (default worker count)
// This simulates a typical incremental scan where some files have changed
func BenchmarkScanFiles_PartialCacheParallel(b *testing.B) {
	repoPath, cachedFiles := setupBenchmarkRepo(b, 100)

	// Modify 20% of files to simulate changes
	changedFiles := make(map[string]bool)
	fileCount := 0
	changePercent := 0.2
	numChanged := int(float64(len(cachedFiles)) * changePercent)

	for path := range cachedFiles {
		if fileCount >= numChanged {
			break
		}
		changedFiles[path] = true
		fileCount++
	}

	// Create cache with old modtime for changed files
	cache := NewCache()
	for path, info := range cachedFiles {
		if changedFiles[path] {
			// Set modtime to 1 hour ago to trigger rehashing
			cache.Files[path] = FileInfo{
				Hash:     info.Hash,
				Modified: info.Modified.Add(-1 * time.Hour),
				Size:     info.Size,
			}
		} else {
			cache.Files[path] = info
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, cache, metrics, 0) // maxHashWorkers=0 for auto-detect
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}

		// Verify selective hashing is working correctly
		expectedCached := len(cachedFiles) - numChanged
		if metrics.CachedFiles != expectedCached {
			b.Errorf("Expected %d cached files, got %d", expectedCached, metrics.CachedFiles)
		}
		if metrics.HashedFiles != numChanged {
			b.Errorf("Expected %d hashed files, got %d", numChanged, metrics.HashedFiles)
		}
	}
}

// BenchmarkScanFiles_NoCacheParallel benchmarks parallel hashing without cache:
// - No cache (all files need hashing)
// - Parallel hashing (default worker count)
// This measures the benefit of parallel processing alone
func BenchmarkScanFiles_NoCacheParallel(b *testing.B) {
	repoPath, _ := setupBenchmarkRepo(b, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, nil, metrics, 0) // maxHashWorkers=0 for auto-detect
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}

		// Verify all files were hashed
		if metrics.HashedFiles != metrics.TotalFiles {
			b.Errorf("Expected %d hashed files, got %d", metrics.TotalFiles, metrics.HashedFiles)
		}
	}
}

// BenchmarkScanFiles_LargeRepository benchmarks performance on larger repository
func BenchmarkScanFiles_LargeRepository(b *testing.B) {
	// Test with larger file count to simulate big projects
	repoPath, cachedFiles := setupBenchmarkRepo(b, 500)

	// Create cache with all files
	cache := NewCache()
	cache.Files = cachedFiles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, cache, metrics, 0)
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}
	}
}

// Benchmark helper to calculate throughput statistics
func BenchmarkScanFiles_WithStats(b *testing.B) {
	repoPath, cachedFiles := setupBenchmarkRepo(b, 100)

	// Calculate total size
	var totalSize int64
	for _, info := range cachedFiles {
		totalSize += info.Size
	}

	// Create cache
	cache := NewCache()
	cache.Files = cachedFiles

	b.ResetTimer()

	// Run benchmark and collect metrics
	var totalOps int
	var totalDuration time.Duration

	for i := 0; i < b.N; i++ {
		start := time.Now()
		metrics := &ScanMetrics{}
		_, err := ScanFiles(repoPath, []string{}, cache, metrics, 0)
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}
		duration := time.Since(start)

		totalOps++
		totalDuration += duration

		// Report throughput statistics
		filesPerSec := float64(metrics.TotalFiles) / duration.Seconds()
		mbPerSec := (float64(totalSize) / (1024 * 1024)) / duration.Seconds()

		b.ReportMetric(filesPerSec, "files/sec")
		b.ReportMetric(mbPerSec, "MB/sec")
	}

	// Report average metrics
	avgDuration := totalDuration / time.Duration(totalOps)
	b.ReportMetric(float64(avgDuration.Milliseconds()), "ms/op")
}

// BenchmarkParallelHashWorkers benchmarks different worker counts
func BenchmarkParallelHashWorkers(b *testing.B) {
	repoPath, _ := setupBenchmarkRepo(b, 100)

	workerCounts := []int{1, 2, 4, 8}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				metrics := &ScanMetrics{}
				_, err := ScanFiles(repoPath, []string{}, nil, metrics, workers)
				if err != nil {
					b.Fatalf("ScanFiles failed: %v", err)
				}
			}
		})
	}
}
