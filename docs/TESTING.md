# Testing Guide

**Project:** Gendocs
**Last Updated:** 2025-12-23

---

## Table of Contents

1. [Overview](#overview)
2. [Test Organization](#test-organization)
3. [Running Tests](#running-tests)
4. [Writing Unit Tests](#writing-unit-tests)
5. [Writing Integration Tests](#writing-integration-tests)
6. [Test Helpers](#test-helpers)
7. [Coverage Requirements](#coverage-requirements)
8. [Best Practices](#best-practices)

---

## Overview

Gendocs employs a comprehensive testing strategy with three test levels:

1. **Unit Tests** - Test individual components in isolation
2. **Integration Tests** - Test component interactions
3. **Manual Tests** - Real LLM API validation (pre-release)

### Current Status

| Component | Unit Tests | Integration Tests | Coverage |
|-----------|------------|-------------------|----------|
| LLM Clients | ✅ 31 tests | N/A | ~85% |
| Tools | ✅ 29 tests | ✅ Included | ~90% |
| Prompts | ✅ 17 tests | N/A | ~85% |
| Config | ✅ 20 tests | N/A | ~80% |
| Validation | ✅ 25 tests | N/A | ~95% |
| Agents | ✅ Basic | ✅ 6 tests | ~70% |
| **TOTAL** | **~147 tests** | **6 tests** | **~82%** |

---

## Test Organization

### File Structure

```
internal/
├── llm/
│   ├── openai.go
│   ├── openai_test.go          # Unit tests
│   ├── anthropic.go
│   └── anthropic_test.go
├── agents/
│   ├── analyzer.go
│   ├── analyzer_test.go        # Unit tests
│   └── analyzer_integration_test.go  # Integration tests
└── testing/
    ├── helpers.go               # Test utilities
    └── fixtures.go              # Sample data
```

### Naming Conventions

- Unit tests: `*_test.go`
- Integration tests: `*_integration_test.go`
- Test functions: `Test<Component>_<Method>_<Scenario>`

**Examples:**
```go
func TestOpenAIClient_GenerateCompletion_Success(t *testing.T)
func TestFileReadTool_Execute_FileNotFound(t *testing.T)
func TestAnalyzerAgent_CompleteFlow(t *testing.T)  // Integration
```

### Build Tags

Integration tests use build tags:

```go
// +build integration

package agents

func TestAnalyzerAgent_CompleteFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // ...
}
```

---

## Running Tests

### All Tests

```bash
# Run all tests (unit + integration)
make test

# With verbose output
make test-verbose

# With coverage report
make test-coverage
```

### Unit Tests Only

```bash
# Fast unit tests only (skip integration)
make test-short

# Or directly
go test -short -race ./...
```

### Integration Tests

```bash
# Run integration tests
go test -tags integration ./...

# Specific package
go test -tags integration ./internal/agents/
```

### Specific Tests

```bash
# Run single test
go test -run TestOpenAIClient_GenerateCompletion_Success ./internal/llm/

# Run test pattern
go test -run TestFileReadTool ./internal/tools/
```

### Coverage

```bash
# Generate coverage report
make test-coverage

# View HTML report
go tool cover -html=coverage/coverage.out

# Check coverage for specific package
go test -cover ./internal/llm/
```

---

## Writing Unit Tests

### Basic Structure

```go
package mypackage

import (
    "testing"
)

func TestMyFunction_Success(t *testing.T) {
    // Arrange
    input := "test"

    // Act
    result, err := MyFunction(input)

    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if result != "expected" {
        t.Errorf("Expected 'expected', got '%s'", result)
    }
}
```

### Table-Driven Tests

```go
func TestMyFunction_Various(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "TEST",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing LLM Clients

```go
func TestOpenAIClient_GenerateCompletion_Success(t *testing.T) {
    // Setup mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate request
        assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

        // Return mock response
        w.Write([]byte(`{
            "choices": [{
                "message": {"content": "test response"}
            }],
            "usage": {"prompt_tokens": 10, "completion_tokens": 5}
        }`))
    }))
    defer server.Close()

    // Create client
    client := NewOpenAIClient(config.LLMConfig{
        APIKey: "test-key",
        BaseURL: server.URL,
    }, nil)

    // Execute
    resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
        Messages: []Message{{Role: "user", Content: "test"}},
    })

    // Verify
    assert.NoError(t, err)
    assert.Equal(t, "test response", resp.Content)
}
```

### Testing Tools

```go
func TestFileReadTool_Execute_Success(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("test content"), 0644)

    // Create tool
    tool := NewFileReadTool(3)

    // Execute
    result, err := tool.Execute(context.Background(), map[string]interface{}{
        "file_path": testFile,
    })

    // Verify
    assert.NoError(t, err)
    assert.Contains(t, result.(map[string]interface{})["content"], "test content")
}
```

---

## Writing Integration Tests

### Agent Flow Tests

```go
// +build integration

func TestAnalyzerAgent_CompleteFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // 1. Create test repository
    repoPath := testing.CreateTempRepo(t, testing.SampleGoProject())

    // 2. Create mock LLM with predefined responses
    mockClient := &testing.MockLLMClient{
        Responses: []llm.CompletionResponse{
            // Tool call to list files
            {ToolCalls: []llm.ToolCall{{Name: "list_files", ...}}},
            // Tool call to read file
            {ToolCalls: []llm.ToolCall{{Name: "read_file", ...}}},
            // Final analysis
            {Content: "# Analysis\n\nThis is a Go project..."},
        },
    }

    // 3. Create agent with dependencies
    agent, _ := NewAnalyzerAgent(...)
    agent.llmClient = mockClient

    // 4. Execute
    result, err := agent.Run(context.Background())

    // 5. Verify
    assert.NoError(t, err)
    assert.Contains(t, result, "Analysis")
    assert.Equal(t, 3, mockClient.CallCount)
}
```

### Multi-Component Tests

```go
func TestHandlerToAgentFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup
    cfg := config.AnalyzerConfig{...}
    handler := handlers.NewAnalyzeHandler(cfg, logger)

    // Execute full flow
    err := handler.Handle(context.Background())

    // Verify outputs
    assert.NoError(t, err)
    testing.AssertFileExists(t, ".ai/docs/code_structure.md")
}
```

---

## Test Helpers

### Location

All test helpers are in `internal/testing/`:

```go
import testHelpers "github.com/user/gendocs/internal/testing"
```

### MockLLMClient

```go
// Create mock with responses
mockClient := testHelpers.NewMockLLMClient(
    llm.CompletionResponse{Content: "Response 1"},
    llm.CompletionResponse{Content: "Response 2"},
)

// Use in tests
agent.llmClient = mockClient
agent.Run(ctx)

// Verify calls
assert.Equal(t, 2, mockClient.CallCount)
assert.Equal(t, "user prompt", mockClient.LastRequest.Messages[0].Content)
```

### CreateTempRepo

```go
// Create temp git repository with files
repoPath := testHelpers.CreateTempRepo(t, map[string]string{
    "main.go": "package main\n...",
    "go.mod": "module test\n...",
    "internal/db.go": "package internal\n...",
})

// Cleanup is automatic (t.TempDir())
```

### Sample Fixtures

```go
// Predefined project structures
files := testHelpers.SampleGoProject()
files := testHelpers.SamplePythonProject()

// Sample outputs
output := testHelpers.SampleAnalysisOutput()
readme := testHelpers.SampleREADME()
```

### Assertions

```go
// File operations
testHelpers.AssertFileExists(t, "README.md")
testHelpers.AssertFileNotExists(t, "invalid.txt")
testHelpers.AssertFileContains(t, "README.md", "## Features")

// YAML helpers
testHelpers.CreateYAML(t, tmpDir, "config.yaml", map[string]string{
    "key": "value",
})
```

---

## Coverage Requirements

### Minimum Coverage Targets

- **Critical Components:** 90%+
  - LLM clients
  - Tools
  - Error handling

- **Business Logic:** 80%+
  - Agents
  - Handlers
  - Configuration

- **Infrastructure:** 70%+
  - Logging
  - Worker pool

### Measuring Coverage

```bash
# Full report
make test-coverage

# Per-package
go test -cover ./internal/llm/
# ok    github.com/user/gendocs/internal/llm    1.234s  coverage: 87.5% of statements

# Detailed function coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Coverage Gates

CI blocks merges if coverage drops below 60%:

```yaml
# .github/workflows/test.yml
- name: Check coverage
  run: |
    go test -coverprofile=coverage.out ./...
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 60" | bc -l) )); then
      echo "Coverage $coverage% is below 60%"
      exit 1
    fi
```

---

## Best Practices

### DO ✅

1. **Use `t.Helper()` in Helper Functions**

```go
func assertNoError(t *testing.T, err error) {
    t.Helper() // Reports correct line number on failure
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
}
```

2. **Use `t.Cleanup()` for Deferred Cleanup**

```go
func TestWithCleanup(t *testing.T) {
    server := httptest.NewServer(...)
    t.Cleanup(func() { server.Close() }) // Runs even if test fails

    // Test code...
}
```

3. **Use Subtests for Organization**

```go
func TestMyFunction(t *testing.T) {
    t.Run("success case", func(t *testing.T) { ... })
    t.Run("error case", func(t *testing.T) { ... })
    t.Run("edge case", func(t *testing.T) { ... })
}
```

4. **Test Error Messages**

```go
if !strings.Contains(err.Error(), "expected phrase") {
    t.Errorf("Error should mention 'expected phrase', got: %v", err)
}
```

5. **Use Table Tests for Multiple Scenarios**

### DON'T ❌

1. **Don't Use `t.Fatal` in Goroutines**

```go
// WRONG
go func() {
    t.Fatal("error") // Causes panic
}()

// RIGHT
go func() {
    if err != nil {
        t.Error("error")
    }
}()
```

2. **Don't Test Implementation Details**

```go
// WRONG - tests internal behavior
assert.Equal(t, 3, client.requestCount)

// RIGHT - tests observable behavior
resp, err := client.GenerateCompletion(...)
assert.NoError(t, err)
assert.NotEmpty(t, resp.Content)
```

3. **Don't Hardcode Paths**

```go
// WRONG
tmpFile := "/tmp/test.txt"

// RIGHT
tmpDir := t.TempDir()
tmpFile := filepath.Join(tmpDir, "test.txt")
```

4. **Don't Ignore Errors**

```go
// WRONG
result, _ := MyFunction()

// RIGHT
result, err := MyFunction()
if err != nil {
    t.Fatalf("Unexpected error: %v", err)
}
```

5. **Don't Share State Between Tests**

```go
// WRONG
var globalState = map[string]string{}

func TestA(t *testing.T) {
    globalState["key"] = "value" // Affects TestB
}

func TestB(t *testing.T) {
    // Depends on TestA running first
}

// RIGHT - each test is independent
func TestA(t *testing.T) {
    state := map[string]string{}
    state["key"] = "value"
}
```

---

## Debugging Tests

### Verbose Output

```bash
# See all test output
go test -v ./...

# See logs from test
go test -v -args -debug
```

### Running Single Test

```bash
# Isolate failing test
go test -run TestMyFunction_SpecificCase ./pkg/
```

### Print Debugging

```go
func TestDebug(t *testing.T) {
    t.Logf("Debug info: %+v", someVar)  // Only shows if test fails or -v
    fmt.Printf("Always prints: %v\n", someVar)  // Always visible
}
```

### Race Detector

```bash
# Detect race conditions
go test -race ./...
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: make test-coverage
      - uses: codecov/codecov-action@v3
        with:
          file: ./coverage/coverage.out
```

### Local Pre-Commit

```bash
#!/bin/bash
# .git/hooks/pre-commit

make test-short || exit 1
make lint || exit 1
```

---

## Troubleshooting

### Tests Hang

- Check for missing `t.Parallel()` with shared resources
- Look for blocking operations without timeouts
- Verify context cancellation is handled

### Flaky Tests

- Race conditions: Run with `-race` flag
- Timing issues: Add explicit synchronization
- External dependencies: Mock or stub

### Coverage Issues

- Use `-coverprofile` to identify uncovered lines
- Check for unreachable code
- Verify test actually exercises the code path

---

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Effective Go - Testing](https://go.dev/doc/effective_go#testing)
- [Testify Assertions](https://github.com/stretchr/testify) (if added)

---

**Document Version:** 1.0
**Maintainers:** Gendocs Team
**Last Review:** 2025-12-23
