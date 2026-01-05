package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MaxLineLength is the maximum length of a single line to prevent memory issues
const MaxLineLength = 10000

// MaxTotalChars is the maximum total characters to return
const MaxTotalChars = 50000

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
	return "Read contents of a source code file. By default reads first 200 lines. Binary files are automatically rejected. Use line_number and line_count for pagination."
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

		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return map[string]interface{}{
					"error":   fmt.Sprintf("File '%s' does not exist", filePath),
					"message": "The requested file was not found. Please verify the path exists and try again with a valid file path from the project root.",
				}, nil
			}
			return nil, &ModelRetryError{Message: fmt.Sprintf("Failed to stat file: %v", err)}
		}

		// Check if it's a directory
		if info.IsDir() {
			return nil, &ModelRetryError{Message: "Path is a directory, not a file"}
		}

		// Check file size - warn if too large
		if info.Size() > 10*1024*1024 { // > 10MB
			return map[string]interface{}{
				"error":   "File too large",
				"message": fmt.Sprintf("File is %d bytes. Consider reading specific sections with line_number and line_count.", info.Size()),
			}, nil
		}

		// Check if file is binary
		if IsBinaryFile(filePath) {
			ext := filepath.Ext(filePath)
			return map[string]interface{}{
				"error":   "Binary file detected",
				"message": fmt.Sprintf("File '%s' appears to be a binary file (extension: %s). Binary files cannot be read as text.", filepath.Base(filePath), ext),
			}, nil
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

		// Cap line count to prevent excessive reads
		if lineCount > 500 {
			lineCount = 500
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil, &ModelRetryError{Message: fmt.Sprintf("Failed to open file: %v", err)}
		}
		defer func() { _ = file.Close() }()

		scanner := bufio.NewScanner(file)
		// Increase scanner buffer for long lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024) // 1MB max line

		var lines []string
		currentLine := 1
		totalChars := 0
		truncatedLines := 0

		for scanner.Scan() {
			if currentLine >= lineNumber && currentLine < lineNumber+lineCount {
				line := scanner.Text()

				// Truncate very long lines
				if len(line) > MaxLineLength {
					line = line[:MaxLineLength] + "... [line truncated]"
					truncatedLines++
				}

				// Check total character limit
				if totalChars+len(line) > MaxTotalChars {
					lines = append(lines, "[OUTPUT TRUNCATED - exceeded character limit]")
					break
				}

				lines = append(lines, line)
				totalChars += len(line)
			}
			currentLine++
			if currentLine >= lineNumber+lineCount {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			// Handle lines that are too long
			if strings.Contains(err.Error(), "token too long") {
				return map[string]interface{}{
					"error":   "File has extremely long lines",
					"message": "This file contains lines that exceed the maximum buffer size. It may be a minified or binary file.",
				}, nil
			}
			return nil, &ModelRetryError{Message: fmt.Sprintf("Error reading file: %v", err)}
		}

		result := map[string]interface{}{
			"content":          lines,
			"start_line":       lineNumber,
			"end_line":         lineNumber + len(lines) - 1,
			"total_lines_read": len(lines),
		}

		if truncatedLines > 0 {
			result["truncated_lines"] = truncatedLines
			result["note"] = fmt.Sprintf("%d lines were truncated due to excessive length", truncatedLines)
		}

		return result, nil
	})
}
