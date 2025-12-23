package tools

import (
	"context"
	"bufio"
	"fmt"
	"os"
	"strconv"
)

// FileReadTool reads file contents with optional pagination
type FileReadTool struct {
	BaseTool
}

// NewFileReadTool creates a new file read tool
func NewFileReadTool(maxRetries int) *FileReadTool {
	return &FileReadTool{
		BaseTool: NewBaseTool(maxRetries),
	}
}

// Name returns the tool name
func (frt *FileReadTool) Name() string {
	return "read_file"
}

// Description returns the tool description
func (frt *FileReadTool) Description() string {
	return "Read contents of a file. By default reads first 200 lines. Use line_number and line_count for pagination."
}

// Parameters returns the JSON schema for the tool parameters
func (frt *FileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
			"line_number": map[string]interface{}{
				"type":        "integer",
				"description": "Starting line number (1-indexed). Default: 1",
			},
			"line_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of lines to read. Default: 200",
			},
		},
		"required": []string{"file_path"},
	}
}

// Execute reads the file contents
func (frt *FileReadTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return frt.RetryableExecute(ctx, func() (interface{}, error) {
		filePath, ok := params["file_path"].(string)
		if !ok {
			return nil, fmt.Errorf("file_path must be a string")
		}

		lineNumber := 1
		if ln, ok := params["line_number"]; ok {
			switch v := ln.(type) {
			case int:
				lineNumber = v
			case float64:
				lineNumber = int(v)
			case string:
				if i, err := strconv.Atoi(v); err == nil {
					lineNumber = i
				}
			}
		}

		lineCount := 200
		if lc, ok := params["line_count"]; ok {
			switch v := lc.(type) {
			case int:
				lineCount = v
			case float64:
				lineCount = int(v)
			case string:
				if i, err := strconv.Atoi(v); err == nil {
					lineCount = i
				}
			}
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil, &ModelRetryError{Message: fmt.Sprintf("Failed to open file: %v", err)}
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var lines []string
		currentLine := 1

		for scanner.Scan() {
			if currentLine >= lineNumber && currentLine < lineNumber+lineCount {
				lines = append(lines, scanner.Text())
			}
			currentLine++
			if currentLine >= lineNumber+lineCount {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, &ModelRetryError{Message: fmt.Sprintf("Error reading file: %v", err)}
		}

		return map[string]interface{}{
			"content":         lines,
			"start_line":      lineNumber,
			"end_line":        lineNumber + len(lines) - 1,
			"total_lines_read": len(lines),
		}, nil
	})
}
