package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SearchFilesTool struct {
	BaseTool
	repoPath string
}

func NewSearchFilesTool(repoPath string, maxRetries int) *SearchFilesTool {
	return &SearchFilesTool{
		BaseTool: NewBaseTool(maxRetries),
		repoPath: repoPath,
	}
}

func (st *SearchFilesTool) Name() string {
	return "search_files"
}

func (st *SearchFilesTool) Description() string {
	return "Search for a string pattern in files within the repository. Returns matching lines with file paths and line numbers. Automatically skips binary files and ignored directories."
}

func (st *SearchFilesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The exact string pattern to search for (case-sensitive)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional relative path to limit the search scope (default: root)",
			},
			"extensions": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional list of file extensions to include (e.g., ['.go', '.md'])",
			},
		},
		"required": []string{"pattern"},
	}
}

func (st *SearchFilesTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return st.RetryableExecute(ctx, func() (interface{}, error) {
		pattern, ok := params["pattern"].(string)
		if !ok || pattern == "" {
			return nil, fmt.Errorf("pattern is required and must be a non-empty string")
		}

		searchPath := st.repoPath
		if p, ok := params["path"].(string); ok && p != "" {
			cleanPath := filepath.Clean(p)
			if strings.Contains(cleanPath, "..") {
				return nil, fmt.Errorf("invalid path: cannot contain '..'")
			}
			searchPath = filepath.Join(st.repoPath, cleanPath)
		}

		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			return map[string]interface{}{
				"error":   fmt.Sprintf("Path '%s' does not exist", searchPath),
				"message": "The requested search path was not found.",
			}, nil
		}

		var allowedExts map[string]bool
		if extList, ok := params["extensions"].([]interface{}); ok && len(extList) > 0 {
			allowedExts = make(map[string]bool)
			for _, v := range extList {
				if s, ok := v.(string); ok {
					if !strings.HasPrefix(s, ".") {
						s = "." + s
					}
					allowedExts[s] = true
				}
			}
		}

		ignorePatterns := LoadGitignorePatterns(st.repoPath)

		var results []string
		totalBytes := 0
		truncated := false
		matchesCount := 0

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			relPath, err := filepath.Rel(st.repoPath, path)
			if err != nil {
				relPath = path
			}

			if info.IsDir() {
				if ShouldIgnore(relPath, ignorePatterns) {
					return filepath.SkipDir
				}
				return nil
			}

			if ShouldIgnore(relPath, ignorePatterns) {
				return nil
			}

			if allowedExts != nil {
				ext := filepath.Ext(path)
				if !allowedExts[ext] {
					return nil
				}
			}

			if IsBinaryFile(path) {
				return nil
			}

			fileMatches, bytesAdded, err := searchInFile(path, relPath, pattern)
			if err != nil {
				return nil
			}

			if len(fileMatches) > 0 {
				if totalBytes+bytesAdded > MaxToolResponseSize {
					truncated = true
					for _, m := range fileMatches {
						if totalBytes+len(m) > MaxToolResponseSize {
							break
						}
						results = append(results, m)
						totalBytes += len(m)
						matchesCount++
					}
					return filepath.SkipAll
				}

				results = append(results, fileMatches...)
				totalBytes += bytesAdded
				matchesCount += len(fileMatches)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("error walking directory: %v", err)
		}

		response := map[string]interface{}{
			"matches_count": matchesCount,
			"results":       results,
		}

		if truncated {
			response["warning"] = "Output truncated due to size limit. Try a more specific path or pattern."
		}

		if matchesCount == 0 {
			response["message"] = fmt.Sprintf("No matches found for pattern '%s'", pattern)
		}

		return response, nil
	})
}

func searchInFile(fullPath, relPath, pattern string) ([]string, int, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var matches []string
	totalBytes := 0
	scanner := bufio.NewScanner(file)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.Contains(line, pattern) {
			if len(line) > 300 {
				line = line[:300] + "..."
			}

			match := fmt.Sprintf("%s:%d: %s", relPath, lineNum, strings.TrimSpace(line))
			matches = append(matches, match)
			totalBytes += len(match)
		}
	}

	return matches, totalBytes, scanner.Err()
}
