package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchFilesTool_Name(t *testing.T) {
	tool := NewSearchFilesTool("/tmp", 3)
	if tool.Name() != "search_files" {
		t.Errorf("Expected name 'search_files', got '%s'", tool.Name())
	}
}

func TestSearchFilesTool_Description(t *testing.T) {
	tool := NewSearchFilesTool("/tmp", 3)
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestSearchFilesTool_Parameters(t *testing.T) {
	tool := NewSearchFilesTool("/tmp", 3)
	params := tool.Parameters()

	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "pattern" {
		t.Error("Expected required field 'pattern'")
	}
}

func TestSearchFilesTool_Execute(t *testing.T) {
	repoPath := t.TempDir()

	files := map[string]string{
		"main.go":         "package main\nfunc main() {\n\tprintln(\"hello world\")\n}",
		"utils.go":        "package main\nfunc helper() string {\n\treturn \"hello\"\n}",
		"README.md":       "# Hello Project\nThis is a hello world project.",
		"internal/api.go": "package api\n// handles hello api",
		"vendor/dep.go":   "package dep\nfunc dep() { println(\"hello\") }",
		".gitignore":      "vendor/\nignored.txt",
		"ignored.txt":     "hello ignored",
	}

	for path, content := range files {
		fullPath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tool := NewSearchFilesTool(repoPath, 3)

	t.Run("Basic Search", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern": "hello",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		res := result.(map[string]interface{})
		count := res["matches_count"].(int)
		matches := res["results"].([]string)

		if count != 4 {
			t.Errorf("Expected 4 matches, got %d", count)
			for _, m := range matches {
				t.Log(m)
			}
		}
	})

	t.Run("Extension Filter", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern":    "hello",
			"extensions": []interface{}{".go"},
		})
		if err != nil {
			t.Fatal(err)
		}

		res := result.(map[string]interface{})
		count := res["matches_count"].(int)

		if count != 3 {
			t.Errorf("Expected 3 matches, got %d", count)
		}
	})

	t.Run("Subdirectory Scope", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern": "hello",
			"path":    "internal",
		})
		if err != nil {
			t.Fatal(err)
		}

		res := result.(map[string]interface{})
		count := res["matches_count"].(int)

		if count != 1 {
			t.Errorf("Expected 1 match, got %d", count)
		}
	})

	t.Run("Binary Skip", func(t *testing.T) {
		binPath := filepath.Join(repoPath, "test.bin")
		content := append([]byte{0, 0, 0, 0}, []byte("hello binary")...)
		os.WriteFile(binPath, content, 0644)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern": "hello binary",
		})
		if err != nil {
			t.Fatal(err)
		}

		res := result.(map[string]interface{})
		count := res["matches_count"].(int)

		if count != 0 {
			t.Error("Expected binary file to be skipped")
		}
	})

	t.Run("Truncation", func(t *testing.T) {
		largePath := filepath.Join(repoPath, "large.txt")
		f, _ := os.Create(largePath)
		pattern := "match me please "
		for i := 0; i < 5000; i++ {
			f.WriteString(pattern + "\n")
		}
		f.Close()

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern": "match me please",
		})
		if err != nil {
			t.Fatal(err)
		}

		res := result.(map[string]interface{})
		if _, ok := res["warning"]; !ok {
			t.Error("Expected truncation warning")
		}
	})

	t.Run("Case Sensitive", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern": "HELLO",
		})
		if err != nil {
			t.Fatal(err)
		}

		res := result.(map[string]interface{})
		count := res["matches_count"].(int)

		if count != 0 {
			t.Errorf("Expected 0 matches for case sensitive search, got %d", count)
		}
	})
}
