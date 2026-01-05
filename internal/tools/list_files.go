package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// MaxFilesToList is the maximum number of files to return
const MaxFilesToList = 500

// ListFilesTool lists files in a directory recursively
type ListFilesTool struct {
	BaseTool
}

// NewListFilesTool creates a new list files tool
func NewListFilesTool(maxRetries int) *ListFilesTool {
	return &ListFilesTool{
		BaseTool: NewBaseTool(maxRetries),
	}
}

// Name returns the tool name
func (lft *ListFilesTool) Name() string {
	return "list_files"
}

// Description returns the tool description
func (lft *ListFilesTool) Description() string {
	return "List source code files in a directory recursively. Automatically filters out binary files, build outputs, dependencies (node_modules, vendor), and files matching .gitignore patterns. Returns up to 500 files."
}

// Parameters returns the JSON schema for the tool parameters
func (lft *ListFilesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"directory": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list files from",
			},
		},
		"required": []string{"directory"},
	}
}

// Execute lists files in the directory
func (lft *ListFilesTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return lft.RetryableExecute(ctx, func() (interface{}, error) {
		directory, ok := params["directory"].(string)
		if !ok {
			return nil, fmt.Errorf("directory must be a string")
		}

		info, statErr := os.Stat(directory)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return map[string]interface{}{
					"files":   []string{},
					"count":   0,
					"error":   fmt.Sprintf("Directory '%s' does not exist", directory),
					"message": "The requested directory was not found. Please verify the path exists and try again with a valid directory from the project root.",
				}, nil
			}
			return nil, &ModelRetryError{Message: fmt.Sprintf("Failed to access directory: %v", statErr)}
		}

		if !info.IsDir() {
			return map[string]interface{}{
				"files":   []string{},
				"count":   0,
				"error":   fmt.Sprintf("Path '%s' is not a directory", directory),
				"message": "The specified path is a file, not a directory. Use read_file tool to read file contents.",
			}, nil
		}

		ignorePatterns := LoadGitignorePatterns(directory)

		var files []string
		var skippedCount int
		var truncated bool

		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip permission errors
				if os.IsPermission(err) {
					return nil
				}
				return err
			}

			// Get relative path for pattern matching
			relPath, relErr := filepath.Rel(directory, path)
			if relErr != nil {
				relPath = path
			}

			// Skip directories that match ignore patterns
			if info.IsDir() {
				if ShouldIgnore(relPath, ignorePatterns) || ShouldIgnore(info.Name(), ignorePatterns) {
					skippedCount++
					return filepath.SkipDir
				}
				return nil
			}

			// Skip files that match ignore patterns
			if ShouldIgnore(relPath, ignorePatterns) || ShouldIgnore(info.Name(), ignorePatterns) {
				skippedCount++
				return nil
			}

			// Skip binary files
			if IsBinaryFile(path) {
				skippedCount++
				return nil
			}

			// Enforce maximum file limit
			if len(files) >= MaxFilesToList {
				truncated = true
				return filepath.SkipAll
			}

			files = append(files, relPath)
			return nil
		})

		if err != nil && err != filepath.SkipAll {
			return nil, &ModelRetryError{Message: fmt.Sprintf("Failed to list files: %v", err)}
		}

		result := map[string]interface{}{
			"files":   files,
			"count":   len(files),
			"skipped": skippedCount,
		}

		if truncated {
			result["truncated"] = true
			result["message"] = fmt.Sprintf("Results truncated to %d files. Use more specific directory paths for complete listings.", MaxFilesToList)
		}

		return result, nil
	})
}
