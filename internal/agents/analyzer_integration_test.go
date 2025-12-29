//go:build integration
// +build integration

package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
	testHelpers "github.com/user/gendocs/internal/testing"
	"github.com/user/gendocs/internal/tools"
)

// TestAnalyzerAgent_CompleteFlow tests the complete analyzer workflow
// Skip with: go test -short
func TestAnalyzerAgent_CompleteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test repository
	repoPath := testHelpers.CreateTempRepo(t, testHelpers.SampleGoProject())

	// Create mock LLM client with predefined responses
	mockClient := &testHelpers.MockLLMClient{
		Responses: []llm.CompletionResponse{
			// Response 1: Request to list files
			{
				Content: "I need to list the files in the repository.",
				ToolCalls: []llm.ToolCall{
					{
						Name: "list_files",
						Arguments: map[string]interface{}{
							"directory": ".",
						},
					},
				},
				Usage: llm.TokenUsage{InputTokens: 100, OutputTokens: 20},
			},
			// Response 2: Request to read main.go
			{
				Content: "Let me read the main file.",
				ToolCalls: []llm.ToolCall{
					{
						Name: "read_file",
						Arguments: map[string]interface{}{
							"file_path": "main.go",
						},
					},
				},
				Usage: llm.TokenUsage{InputTokens: 150, OutputTokens: 25},
			},
			// Response 3: Request to read go.mod
			{
				Content: "Let me check the dependencies.",
				ToolCalls: []llm.ToolCall{
					{
						Name: "read_file",
						Arguments: map[string]interface{}{
							"file_path": "go.mod",
						},
					},
				},
				Usage: llm.TokenUsage{InputTokens: 200, OutputTokens: 30},
			},
			// Response 4: Final analysis
			{
				Content: testHelpers.SampleAnalysisOutput(),
				Usage:   llm.TokenUsage{InputTokens: 500, OutputTokens: 300},
			},
		},
	}

	// Create logger
	logCfg := &logging.Config{
		LogDir:       t.TempDir(),
		FileLevel:    logging.LevelInfo,
		ConsoleLevel: logging.LevelInfo,
		EnableCaller: false,
	}
	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create tools
	fileReadTool := tools.NewFileReadTool(3)
	listFilesTool := tools.NewListFilesTool(3)

	// Create prompt manager
	// For integration tests, we use simple mock prompts
	promptManager := prompts.NewManagerFromMap(map[string]string{
		"analyzer_system": "You are a code analyzer.",
		"analyzer_user":   "Analyze the repository at {{.RepoPath}}",
	})

	// Create sub-agent (simulating AnalyzerAgent behavior)
	subAgent, err := NewSubAgent(SubAgentConfig{
		Name:         "TestAnalyzer",
		LLMConfig:    config.LLMConfig{},
		RepoPath:     repoPath,
		PromptSuffix: "analyzer",
	}, &llm.Factory{}, promptManager, logger)

	if err != nil {
		t.Fatalf("Failed to create sub-agent: %v", err)
	}

	// Override the LLM client with our mock
	subAgent.BaseAgent.llmClient = mockClient

	// Add tools
	subAgent.BaseAgent.tools = []tools.Tool{fileReadTool, listFilesTool}

	// Run the analyzer
	ctx := context.Background()
	result, err := subAgent.Run(ctx)

	// Verify execution
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify result contains expected content
	if result == "" {
		t.Fatal("Expected non-empty result")
	}

	if len(result) < 100 {
		t.Errorf("Expected result to be substantial, got %d characters", len(result))
	}

	// Verify tool calls happened
	if mockClient.CallCount != 4 {
		t.Errorf("Expected 4 LLM calls (3 tool requests + 1 final), got %d", mockClient.CallCount)
	}

	// Verify result contains analysis content
	expectedPhrases := []string{"Code Structure", "Components", "Architecture"}
	for _, phrase := range expectedPhrases {
		if !containsString(result, phrase) {
			t.Errorf("Expected result to contain '%s'", phrase)
		}
	}
}

// TestSubAgent_ToolCalling tests that sub-agents can call tools correctly
func TestSubAgent_ToolCalling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test repository
	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"test.txt": "test content",
	})

	// Mock client that requests file read
	mockClient := &testHelpers.MockLLMClient{
		Responses: []llm.CompletionResponse{
			{
				Content: "Reading file",
				ToolCalls: []llm.ToolCall{
					{
						Name: "read_file",
						Arguments: map[string]interface{}{
							"file_path": "test.txt",
						},
					},
				},
			},
			{
				Content: "File contains: test content",
			},
		},
	}

	// Minimal setup
	logger, _ := logging.NewLogger(&logging.Config{
		LogDir:       t.TempDir(),
		FileLevel:    logging.LevelInfo,
		ConsoleLevel: logging.LevelError,
	})
	defer logger.Sync()

	promptManager := prompts.NewManagerFromMap(map[string]string{
		"test_system": "Test system",
		"test_user":   "Test user",
	})

	subAgent, err := NewSubAgent(SubAgentConfig{
		Name:         "TestAgent",
		LLMConfig:    config.LLMConfig{},
		RepoPath:     repoPath,
		PromptSuffix: "test",
	}, &llm.Factory{}, promptManager, logger)

	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	subAgent.BaseAgent.llmClient = mockClient
	subAgent.BaseAgent.tools = []tools.Tool{tools.NewFileReadTool(3)}

	// Run
	result, err := subAgent.Run(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify tool was called
	if mockClient.CallCount != 2 {
		t.Errorf("Expected 2 calls, got %d", mockClient.CallCount)
	}

	// Verify result mentions the file content
	if !containsString(result, "test content") {
		t.Error("Expected result to mention file content")
	}
}

// TestSubAgent_ErrorHandling tests error handling in agent execution
func TestSubAgent_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	repoPath := t.TempDir()

	// Mock client that triggers tool error
	mockClient := &testHelpers.MockLLMClient{
		Responses: []llm.CompletionResponse{
			{
				Content: "Reading non-existent file",
				ToolCalls: []llm.ToolCall{
					{
						Name: "read_file",
						Arguments: map[string]interface{}{
							"file_path": "/nonexistent/file.txt",
						},
					},
				},
			},
			{
				Content: "I encountered an error reading the file.",
			},
		},
	}

	logger, _ := logging.NewLogger(&logging.Config{
		LogDir:       t.TempDir(),
		FileLevel:    logging.LevelError,
		ConsoleLevel: logging.LevelError,
	})
	defer logger.Sync()

	promptManager := prompts.NewManagerFromMap(map[string]string{
		"test_system": "Test",
		"test_user":   "Test",
	})

	subAgent, _ := NewSubAgent(SubAgentConfig{
		Name:         "ErrorTest",
		LLMConfig:    config.LLMConfig{},
		RepoPath:     repoPath,
		PromptSuffix: "test",
	}, &llm.Factory{}, promptManager, logger)

	subAgent.BaseAgent.llmClient = mockClient
	subAgent.BaseAgent.tools = []tools.Tool{tools.NewFileReadTool(3)}

	// Run - should handle tool error gracefully
	result, err := subAgent.Run(context.Background())

	// Agent should not crash, but report the error to LLM
	if err != nil {
		t.Fatalf("Agent should handle tool errors gracefully, got: %v", err)
	}

	// Result should still be returned (LLM's response after seeing error)
	if result == "" {
		t.Error("Expected result even after tool error")
	}
}

// TestSubAgent_ContextCancellation tests context cancellation
func TestSubAgent_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	repoPath := t.TempDir()

	// Mock client that never returns (simulated)
	mockClient := &testHelpers.MockLLMClient{
		Responses: []llm.CompletionResponse{
			{Content: "Response"},
		},
	}

	logger, _ := logging.NewLogger(&logging.Config{
		LogDir:       t.TempDir(),
		FileLevel:    logging.LevelError,
		ConsoleLevel: logging.LevelError,
	})
	defer logger.Sync()

	promptManager := prompts.NewManagerFromMap(map[string]string{
		"test_system": "Test",
		"test_user":   "Test",
	})

	subAgent, _ := NewSubAgent(SubAgentConfig{
		Name:         "CancelTest",
		LLMConfig:    config.LLMConfig{},
		RepoPath:     repoPath,
		PromptSuffix: "test",
	}, &llm.Factory{}, promptManager, logger)

	subAgent.BaseAgent.llmClient = mockClient

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Run should respect context
	_, err := subAgent.Run(ctx)

	// Should get context canceled error
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// TestSubAgent_MultipleToolCalls tests multiple sequential tool calls
func TestSubAgent_MultipleToolCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	repoPath := testHelpers.CreateTempRepo(t, map[string]string{
		"file1.txt": "content 1",
		"file2.txt": "content 2",
		"file3.txt": "content 3",
	})

	mockClient := &testHelpers.MockLLMClient{
		Responses: []llm.CompletionResponse{
			{ToolCalls: []llm.ToolCall{{Name: "list_files", Arguments: map[string]interface{}{"directory": "."}}}},
			{ToolCalls: []llm.ToolCall{{Name: "read_file", Arguments: map[string]interface{}{"file_path": "file1.txt"}}}},
			{ToolCalls: []llm.ToolCall{{Name: "read_file", Arguments: map[string]interface{}{"file_path": "file2.txt"}}}},
			{ToolCalls: []llm.ToolCall{{Name: "read_file", Arguments: map[string]interface{}{"file_path": "file3.txt"}}}},
			{Content: "All files read successfully"},
		},
	}

	logger, _ := logging.NewLogger(&logging.Config{
		LogDir:       t.TempDir(),
		FileLevel:    logging.LevelError,
		ConsoleLevel: logging.LevelError,
	})
	defer logger.Sync()

	promptManager := prompts.NewManagerFromMap(map[string]string{
		"test_system": "Test",
		"test_user":   "Test",
	})

	subAgent, _ := NewSubAgent(SubAgentConfig{
		Name:         "MultiToolTest",
		LLMConfig:    config.LLMConfig{},
		RepoPath:     repoPath,
		PromptSuffix: "test",
	}, &llm.Factory{}, promptManager, logger)

	subAgent.BaseAgent.llmClient = mockClient
	subAgent.BaseAgent.tools = []tools.Tool{
		tools.NewFileReadTool(3),
		tools.NewListFilesTool(3),
	}

	result, err := subAgent.Run(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have called LLM 5 times (4 tool calls + 1 final)
	if mockClient.CallCount != 5 {
		t.Errorf("Expected 5 LLM calls, got %d", mockClient.CallCount)
	}

	if !containsString(result, "successfully") {
		t.Error("Expected success message in result")
	}
}

// Helper function
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle || len(needle) == 0 || findSubstring(haystack, needle))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
