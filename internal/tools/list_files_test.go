package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListFilesTool_Name(t *testing.T) {
	tool := NewListFilesTool(3)
	if tool.Name() != "list_files" {
		t.Errorf("Expected name 'list_files', got '%s'", tool.Name())
	}
}

func TestListFilesTool_Description(t *testing.T) {
	tool := NewListFilesTool(3)
	desc := tool.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestListFilesTool_Parameters(t *testing.T) {
	tool := NewListFilesTool(3)
	params := tool.Parameters()

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Expected 'required' field in parameters")
	}

	if len(required) != 1 || required[0] != "directory" {
		t.Errorf("Expected required field 'directory', got %v", required)
	}

	// Check properties
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'properties' field in parameters")
	}

	if _, ok := properties["directory"]; !ok {
		t.Error("Expected 'directory' property")
	}
}

func TestListFilesTool_Execute_Success(t *testing.T) {
	// Create temp directory with files
	tmpDir := t.TempDir()

	// Create files
	files := []string{
		"file1.txt",
		"file2.go",
		"subdir/file3.md",
		"subdir/nested/file4.yaml",
	}

	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f)
		_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
		_ = os.WriteFile(fullPath, []byte("content"), 0644)
	}

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be map, got %T", result)
	}

	filesList, ok := resultMap["files"].([]string)
	if !ok {
		t.Fatalf("Expected files to be []string, got %T", resultMap["files"])
	}

	if len(filesList) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(filesList))
	}

	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count to be int, got %T", resultMap["count"])
	}

	if count != len(files) {
		t.Errorf("Expected count %d, got %d", len(files), count)
	}

	// Verify all expected files are in the list
	fileMap := make(map[string]bool)
	for _, f := range filesList {
		fileMap[f] = true
	}

	for _, expectedFile := range files {
		if !fileMap[expectedFile] {
			t.Errorf("Expected file '%s' not found in results", expectedFile)
		}
	}
}

func TestListFilesTool_Execute_EmptyDirectory(t *testing.T) {
	// Create empty directory
	tmpDir := t.TempDir()

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	filesList := resultMap["files"].([]string)

	if len(filesList) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", len(filesList))
	}

	if count := resultMap["count"].(int); count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestListFilesTool_Execute_OnlyDirectories(t *testing.T) {
	// Create directory with only subdirectories (no files)
	tmpDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmpDir, "dir1"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "dir2/subdir"), 0755)

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	filesList := resultMap["files"].([]string)

	if len(filesList) != 0 {
		t.Errorf("Expected 0 files (directories should not be listed), got %d", len(filesList))
	}
}

func TestListFilesTool_Execute_DirectoryNotFound(t *testing.T) {
	tool := NewListFilesTool(3)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": "/nonexistent/directory",
	})

	if err != nil {
		t.Fatalf("Expected no error (graceful handling), got %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if _, hasError := resultMap["error"]; !hasError {
		t.Fatal("Expected result to contain 'error' field for non-existent directory")
	}

	if _, hasMessage := resultMap["message"]; !hasMessage {
		t.Fatal("Expected result to contain 'message' field for non-existent directory")
	}

	filesList := resultMap["files"].([]string)
	if len(filesList) != 0 {
		t.Errorf("Expected 0 files for non-existent directory, got %d", len(filesList))
	}
}

func TestListFilesTool_Execute_MissingDirectory(t *testing.T) {
	tool := NewListFilesTool(3)

	_, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected error for missing directory parameter, got nil")
	}
}

func TestListFilesTool_Execute_InvalidDirectory(t *testing.T) {
	tool := NewListFilesTool(3)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": 123, // Invalid type
	})

	if err == nil {
		t.Fatal("Expected error for invalid directory type, got nil")
	}
}

func TestListFilesTool_Execute_FileAsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("content"), 0644)

	tool := NewListFilesTool(3)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": testFile,
	})

	if err != nil {
		t.Fatalf("Expected no error (graceful handling), got %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if _, hasError := resultMap["error"]; !hasError {
		t.Fatal("Expected result to contain 'error' field when path is a file")
	}

	filesList := resultMap["files"].([]string)
	if len(filesList) != 0 {
		t.Errorf("Expected 0 files when path is a file, got %d", len(filesList))
	}
}

func TestListFilesTool_Execute_DeepNesting(t *testing.T) {
	// Test with deeply nested directories
	tmpDir := t.TempDir()

	// Create deep nesting
	deepPath := tmpDir
	for i := 0; i < 10; i++ {
		deepPath = filepath.Join(deepPath, "level")
		_ = os.MkdirAll(deepPath, 0755)
	}

	// Create file at the deepest level
	deepFile := filepath.Join(deepPath, "deep.txt")
	_ = os.WriteFile(deepFile, []byte("content"), 0644)

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	filesList := resultMap["files"].([]string)

	if len(filesList) != 1 {
		t.Errorf("Expected 1 file in deep nesting, got %d", len(filesList))
	}

	// File should be found with relative path
	expectedRelPath := filepath.Join("level", "level", "level", "level", "level", "level", "level", "level", "level", "level", "deep.txt")
	if filesList[0] != expectedRelPath {
		t.Errorf("Expected file '%s', got '%s'", expectedRelPath, filesList[0])
	}
}

func TestListFilesTool_Execute_SpecialCharactersInFilenames(t *testing.T) {
	// Test files with special characters
	tmpDir := t.TempDir()

	specialFiles := []string{
		"file with spaces.txt",
		"file-with-dashes.go",
		"file_with_underscores.md",
		"file.multiple.dots.yaml",
	}

	for _, f := range specialFiles {
		fullPath := filepath.Join(tmpDir, f)
		_ = os.WriteFile(fullPath, []byte("content"), 0644)
	}

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	filesList := resultMap["files"].([]string)

	if len(filesList) != len(specialFiles) {
		t.Errorf("Expected %d files, got %d", len(specialFiles), len(filesList))
	}

	// Verify all files are found
	fileMap := make(map[string]bool)
	for _, f := range filesList {
		fileMap[f] = true
	}

	for _, expectedFile := range specialFiles {
		if !fileMap[expectedFile] {
			t.Errorf("Expected file '%s' not found in results", expectedFile)
		}
	}
}

func TestListFilesTool_Execute_RelativePaths(t *testing.T) {
	// Test that returned paths are relative to the directory
	tmpDir := t.TempDir()

	// Create file
	file := "testfile.txt"
	_ = os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644)

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	filesList := resultMap["files"].([]string)

	if len(filesList) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(filesList))
	}

	// Path should be relative, not absolute
	if filesList[0] != file {
		t.Errorf("Expected relative path '%s', got '%s'", file, filesList[0])
	}

	// Should NOT start with /
	if filepath.IsAbs(filesList[0]) {
		t.Errorf("Expected relative path, got absolute path '%s'", filesList[0])
	}
}

func TestListFilesTool_Execute_ManyFiles(t *testing.T) {
	// Test with many files
	tmpDir := t.TempDir()

	expectedCount := 100
	for i := 0; i < expectedCount; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i%10))+".txt")
		_ = os.WriteFile(filename, []byte("content"), 0644)
	}

	tool := NewListFilesTool(3)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"directory": tmpDir,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	filesList := resultMap["files"].([]string)

	// Should find all files (note: some may have same name due to modulo)
	if len(filesList) < 10 {
		t.Errorf("Expected at least 10 unique files, got %d", len(filesList))
	}

	if count := resultMap["count"].(int); count != len(filesList) {
		t.Errorf("Expected count to match files length, got count=%d, len=%d", count, len(filesList))
	}
}
