package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

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
	return "List all files in a directory recursively"
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

		var files []string
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(directory, path)
				if err == nil {
					files = append(files, relPath)
				}
			}
			return nil
		})

		if err != nil {
			return nil, &ModelRetryError{Message: fmt.Sprintf("Failed to list files: %v", err)}
		}

		return map[string]interface{}{
			"files": files,
			"count": len(files),
		}, nil
	})
}
