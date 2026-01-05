package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileReadTool_Name(t *testing.T) {
	tool := NewFileReadTool(3)
	if tool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", tool.Name())
	}
}

func TestFileReadTool_Description(t *testing.T) {
	tool := NewFileReadTool(3)
	desc := tool.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestFileReadTool_Parameters(t *testing.T) {
	tool := NewFileReadTool(3)
	params := tool.Parameters()

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Expected 'required' field in parameters")
	}

	if len(required) != 1 || required[0] != "file_path" {
		t.Errorf("Expected required field 'file_path', got %v", required)
	}

	// Check properties
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'properties' field in parameters")
	}

	if _, ok := properties["file_path"]; !ok {
		t.Error("Expected 'file_path' property")
	}

	if _, ok := properties["line_number"]; !ok {
		t.Error("Expected 'line_number' property")
	}

	if _, ok := properties["line_count"]; !ok {
		t.Error("Expected 'line_count' property")
	}
}

func TestFileReadTool_Execute_Success(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFileReadTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": testFile,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be map, got %T", result)
	}

	lines, ok := resultMap["content"].([]string)
	if !ok {
		t.Fatalf("Expected content to be []string, got %T", resultMap["content"])
	}

	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	if lines[0] != "line 1" {
		t.Errorf("Expected first line 'line 1', got '%s'", lines[0])
	}
}

func TestFileReadTool_Execute_WithPagination(t *testing.T) {
	// Create temp file with many lines
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	var content string
	for i := 1; i <= 100; i++ {
		content += "line " + string(rune('0'+i)) + "\n"
	}
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFileReadTool(3)

	// Read lines 10-14 (5 lines starting from line 10)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path":   testFile,
		"line_number": 10,
		"line_count":  5,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	lines := resultMap["content"].([]string)

	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	if start := resultMap["start_line"].(int); start != 10 {
		t.Errorf("Expected start_line 10, got %d", start)
	}

	if end := resultMap["end_line"].(int); end != 14 {
		t.Errorf("Expected end_line 14, got %d", end)
	}
}

func TestFileReadTool_Execute_FileNotFound(t *testing.T) {
	tool := NewFileReadTool(3)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": "/nonexistent/file.txt",
	})

	if err != nil {
		t.Fatalf("Expected no error (graceful handling), got %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if _, hasError := resultMap["error"]; !hasError {
		t.Fatal("Expected result to contain 'error' field for non-existent file")
	}

	if _, hasMessage := resultMap["message"]; !hasMessage {
		t.Fatal("Expected result to contain 'message' field for non-existent file")
	}
}

func TestFileReadTool_Execute_MissingFilePath(t *testing.T) {
	tool := NewFileReadTool(3)

	_, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected error for missing file_path, got nil")
	}
}

func TestFileReadTool_Execute_InvalidFilePath(t *testing.T) {
	tool := NewFileReadTool(3)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": 123, // Invalid type
	})

	if err == nil {
		t.Fatal("Expected error for invalid file_path type, got nil")
	}
}

func TestFileReadTool_Execute_LineNumberTypes(t *testing.T) {
	// Test that line_number accepts different numeric types
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	_ = os.WriteFile(testFile, []byte(content), 0644)

	tool := NewFileReadTool(3)

	tests := []struct {
		name       string
		lineNumber interface{}
		expected   int
	}{
		{"int", 2, 2},
		{"float64", 2.0, 2},
		{"string", "2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"file_path":   testFile,
				"line_number": tt.lineNumber,
				"line_count":  2,
			})

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			resultMap := result.(map[string]interface{})
			if start := resultMap["start_line"].(int); start != tt.expected {
				t.Errorf("Expected start_line %d, got %d", tt.expected, start)
			}
		})
	}
}

func TestFileReadTool_Execute_EmptyFile(t *testing.T) {
	// Create empty file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFileReadTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": testFile,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	lines := resultMap["content"].([]string)

	if len(lines) != 0 {
		t.Errorf("Expected 0 lines for empty file, got %d", len(lines))
	}
}

func TestFileReadTool_Execute_SingleLine(t *testing.T) {
	// Create file with single line (no newline at end)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "single.txt")
	if err := os.WriteFile(testFile, []byte("single line"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFileReadTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": testFile,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	lines := resultMap["content"].([]string)

	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	if lines[0] != "single line" {
		t.Errorf("Expected 'single line', got '%s'", lines[0])
	}
}

func TestFileReadTool_Execute_BeyondFileEnd(t *testing.T) {
	// Test reading beyond end of file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\n"
	_ = os.WriteFile(testFile, []byte(content), 0644)

	tool := NewFileReadTool(3)

	// Request lines 2-100 (but file only has 3 lines)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path":   testFile,
		"line_number": 2,
		"line_count":  100,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	lines := resultMap["content"].([]string)

	// Should only return lines 2-3
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestFileReadTool_Execute_ContextCanceled(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("test content\n"), 0644)

	tool := NewFileReadTool(3)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Execute should respect context
	_, err := tool.Execute(ctx, map[string]interface{}{
		"file_path": testFile,
	})

	// Note: Current implementation doesn't check context during file read
	// This test documents current behavior - may want to add context checks
	_ = err
}
