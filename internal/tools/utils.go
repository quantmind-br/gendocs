package tools

import (
	"os"
	"path/filepath"
	"strings"
)

// MaxToolResponseSize is the maximum size of a tool response in bytes
const MaxToolResponseSize = 50000 // ~50KB

// DefaultIgnorePatterns are patterns always ignored when listing files
var DefaultIgnorePatterns = []string{
	// Version control
	".git",
	".git/**",
	".svn",
	".hg",

	// Dependencies
	"node_modules",
	"node_modules/**",
	"vendor",
	"vendor/**",
	".venv",
	"venv",
	"__pycache__",
	"__pycache__/**",

	// Build outputs
	"dist",
	"dist/**",
	"build",
	"build/**",
	"out",
	"out/**",
	"target",
	"target/**",
	"bin",
	"*.exe",
	"*.dll",
	"*.so",
	"*.dylib",
	"*.o",
	"*.a",
	"*.obj",

	// IDE/Editor
	".idea",
	".idea/**",
	".vscode",
	".vscode/**",
	"*.swp",
	"*.swo",
	"*~",

	// Logs and temp files
	"*.log",
	"*.tmp",
	"*.temp",
	"*.cache",

	// Media and binary files
	"*.png",
	"*.jpg",
	"*.jpeg",
	"*.gif",
	"*.ico",
	"*.bmp",
	"*.webp",
	"*.svg",
	"*.mp3",
	"*.mp4",
	"*.avi",
	"*.mov",
	"*.wav",
	"*.flac",
	"*.pdf",
	"*.doc",
	"*.docx",
	"*.xls",
	"*.xlsx",
	"*.ppt",
	"*.pptx",
	"*.zip",
	"*.tar",
	"*.gz",
	"*.rar",
	"*.7z",
	"*.woff",
	"*.woff2",
	"*.ttf",
	"*.eot",
	"*.otf",

	// Minified files
	"*.min.js",
	"*.min.css",

	// Lock files (large)
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"Cargo.lock",
	"poetry.lock",
	"Gemfile.lock",
	"go.sum",
}

// BinaryExtensions are file extensions that indicate binary files
var BinaryExtensions = map[string]bool{
	// Executables
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".o": true, ".a": true, ".obj": true,
	".com": true, ".msi": true, ".app": true,

	// Images
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".ico": true, ".bmp": true, ".webp": true, ".tiff": true,
	".psd": true, ".raw": true, ".heic": true,

	// Audio/Video
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	".wav": true, ".flac": true, ".ogg": true, ".mkv": true,
	".wmv": true, ".m4a": true, ".aac": true,

	// Archives
	".zip": true, ".tar": true, ".gz": true, ".rar": true,
	".7z": true, ".bz2": true, ".xz": true, ".iso": true,

	// Documents (binary)
	".pdf": true, ".doc": true, ".docx": true,
	".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".odt": true, ".ods": true, ".odp": true,

	// Fonts
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,

	// Database
	".db": true, ".sqlite": true, ".sqlite3": true,

	// Other binary
	".class": true, ".pyc": true, ".pyo": true,
	".wasm": true, ".deb": true, ".rpm": true,
}

// IsBinaryFile checks if a file is binary based on extension and content
func IsBinaryFile(path string) bool {
	// Check extension first (fast path)
	ext := strings.ToLower(filepath.Ext(path))
	if BinaryExtensions[ext] {
		return true
	}

	// Check if file has no extension but is likely a binary (e.g., compiled Go binary)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Skip directories
	if info.IsDir() {
		return false
	}

	// Large files without extension are likely binaries
	if ext == "" && info.Size() > 1024*1024 { // > 1MB
		return true
	}

	// Check file content for binary characteristics
	return isBinaryContent(path)
}

// isBinaryContent checks if file content appears to be binary
func isBinaryContent(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	// Read first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	// Count null bytes and non-printable characters
	nullCount := 0
	nonPrintable := 0

	for i := 0; i < n; i++ {
		b := buf[i]
		if b == 0 {
			nullCount++
		}
		// Non-printable: not tab, newline, carriage return, or printable ASCII
		if b != 9 && b != 10 && b != 13 && (b < 32 || b > 126) && b < 128 {
			nonPrintable++
		}
	}

	// If more than 10% null bytes or 30% non-printable, it's binary
	return nullCount > n/10 || nonPrintable > n*3/10
}

// TruncateString truncates a string to maxLen and adds a truncation notice
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	truncated := s[:maxLen]
	return truncated + "\n\n[TRUNCATED - Response exceeded limit. Showing first " +
		strings.TrimRight(strings.TrimRight(formatBytes(maxLen), "0"), ".") + "]"
}

// formatBytes formats bytes as human readable string
func formatBytes(bytes int) string {
	if bytes < 1024 {
		return string(rune(bytes)) + "B"
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return strings.TrimRight(strings.TrimRight(
			strings.Replace(string(rune(int(kb*10)/10))+"KB", ".", "", 1), "0"), ".") + "KB"
	}
	return "~" + string(rune(int(kb/1024))) + "MB"
}

// LoadGitignorePatterns loads patterns from .gitignore file and combines with defaults
func LoadGitignorePatterns(repoPath string) []string {
	patterns := make([]string, len(DefaultIgnorePatterns))
	copy(patterns, DefaultIgnorePatterns)

	// Try to load .gitignore
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return patterns
	}

	// Parse .gitignore
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns
}

// ShouldIgnore checks if a path should be ignored based on patterns
func ShouldIgnore(path string, patterns []string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)
	baseName := filepath.Base(path)

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Handle negation patterns (not fully supported, just skip)
		if strings.HasPrefix(pattern, "!") {
			continue
		}

		// Remove leading slash for matching
		pattern = strings.TrimPrefix(pattern, "/")

		// Check various match scenarios
		matched := false

		// Direct match with base name
		if matchPattern(baseName, pattern) {
			matched = true
		}

		// Match against full path
		if !matched && matchPattern(normalizedPath, pattern) {
			matched = true
		}

		// Match against path components
		if !matched {
			parts := strings.Split(normalizedPath, "/")
			for _, part := range parts {
				if matchPattern(part, pattern) {
					matched = true
					break
				}
			}
		}

		if matched {
			return true
		}
	}

	return false
}

// matchPattern performs simple glob-style pattern matching
func matchPattern(name, pattern string) bool {
	// Handle ** (matches everything including /)
	if strings.Contains(pattern, "**") {
		// Convert ** to regex-like matching
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			if prefix != "" && !strings.HasPrefix(name, prefix) {
				return false
			}
			if suffix != "" && !strings.HasSuffix(name, suffix) {
				return false
			}
			if prefix != "" || suffix != "" {
				return true
			}
		}
	}

	// Handle * (matches everything except /)
	if strings.Contains(pattern, "*") && !strings.Contains(pattern, "**") {
		return simpleGlobMatch(name, pattern)
	}

	// Exact match
	return name == pattern
}

// simpleGlobMatch performs simple * glob matching
func simpleGlobMatch(name, pattern string) bool {
	// Handle patterns like *.go, test_*, etc.
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(name, suffix)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}

	// Handle patterns with * in the middle
	if strings.Contains(pattern, "*") {
		parts := strings.SplitN(pattern, "*", 2)
		return strings.HasPrefix(name, parts[0]) && strings.HasSuffix(name, parts[1])
	}

	return name == pattern
}

// EstimateTokens estimates the number of tokens in a string
// Approximate rule: 1 token ≈ 4 characters for English text
func EstimateTokens(text string) int {
	return len(text) / 4
}
