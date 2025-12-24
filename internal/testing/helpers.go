package testing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/user/gendocs/internal/llm"
)

// MockLLMClient implements llm.LLMClient for testing
type MockLLMClient struct {
	Responses      []llm.CompletionResponse
	CallCount      int
	LastRequest    llm.CompletionRequest
	ShouldError    bool
	ErrorToReturn  error
	RequestHistory []llm.CompletionRequest
}

// NewMockLLMClient creates a new mock LLM client with predefined responses
func NewMockLLMClient(responses ...llm.CompletionResponse) *MockLLMClient {
	return &MockLLMClient{
		Responses:      responses,
		RequestHistory: make([]llm.CompletionRequest, 0),
	}
}

// GenerateCompletion implements llm.LLMClient
func (m *MockLLMClient) GenerateCompletion(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	m.LastRequest = req
	m.RequestHistory = append(m.RequestHistory, req)

	if m.ShouldError {
		m.CallCount++
		return llm.CompletionResponse{}, m.ErrorToReturn
	}

	if m.CallCount >= len(m.Responses) {
		// Return last response if we've exhausted the list
		if len(m.Responses) > 0 {
			resp := m.Responses[len(m.Responses)-1]
			m.CallCount++
			return resp, nil
		}
		return llm.CompletionResponse{}, fmt.Errorf("no responses configured")
	}

	resp := m.Responses[m.CallCount]
	m.CallCount++
	return resp, nil
}

// SupportsTools implements llm.LLMClient
func (m *MockLLMClient) SupportsTools() bool {
	return true
}

// GetProvider implements llm.LLMClient
func (m *MockLLMClient) GetProvider() string {
	return "mock"
}

// Reset resets the mock state
func (m *MockLLMClient) Reset() {
	m.CallCount = 0
	m.LastRequest = llm.CompletionRequest{}
	m.RequestHistory = make([]llm.CompletionRequest, 0)
	m.ShouldError = false
	m.ErrorToReturn = nil
}

// SetError configures the mock to return an error
func (m *MockLLMClient) SetError(err error) {
	m.ShouldError = true
	m.ErrorToReturn = err
}

// CreateTempRepo creates a temporary git repository for testing
func CreateTempRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	// Create temp directory
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	configUser := exec.Command("git", "config", "user.email", "test@example.com")
	configUser.Dir = tmpDir
	_ = configUser.Run()

	configName := exec.Command("git", "config", "user.name", "Test User")
	configName.Dir = tmpDir
	_ = configName.Run()

	// Create files
	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)

		// Create parent directories
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", parentDir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	// Make initial commit
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = tmpDir
	_ = addCmd.Run()

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = tmpDir
	_ = commitCmd.Run()

	return tmpDir
}

// AssertFileExists checks if a file exists at the given path
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks if a file does not exist at the given path
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expected file to not exist: %s", path)
	}
}

// AssertFileContains checks if a file contains the expected content
func AssertFileContains(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !containsString(string(content), expected) {
		t.Errorf("File %s does not contain expected content.\nExpected substring: %s\nActual content:\n%s",
			path, expected, string(content))
	}
}

// containsString checks if haystack contains needle
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle || len(needle) == 0 ||
		 findSubstring(haystack, needle))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CreateYAML creates a YAML file with the given data
func CreateYAML(t *testing.T, dir, filename string, data map[string]string) {
	t.Helper()

	content := ""
	for key, value := range data {
		// Simple YAML formatting (works for basic cases)
		content += fmt.Sprintf("%s: |\n  %s\n\n", key, value)
	}

	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write YAML file %s: %v", fullPath, err)
	}
}
