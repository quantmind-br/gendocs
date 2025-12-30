# Gendocs - Testing Guidelines

## Test Organization

### File Locations
- Unit tests: co-located with source (e.g., `openai_test.go`)
- Integration tests: `*_integration_test.go` with `//go:build integration` tag
- Benchmarks: `*_bench_test.go`

### Naming Convention
```go
func Test<Type>_<Method>_<Scenario>(t *testing.T)

// Examples:
func TestOpenAIClient_GenerateCompletion_Success(t *testing.T)
func TestAnalyzer_Run_EmptyRepository(t *testing.T)
func TestCachedClient_Get_CacheHit(t *testing.T)
```

## Testing Utilities (`internal/testing/`)

### MockLLMClient
```go
mock := testing.NewMockLLMClient(
    &llmtypes.CompletionResponse{Content: "response1"},
    &llmtypes.CompletionResponse{Content: "response2"},
)
// Tracks: mock.CallCount, mock.RequestHistory
```

### CreateTempRepo
```go
repoPath := testing.CreateTempRepo(t, map[string]string{
    "main.go": "package main\nfunc main() {}",
    "go.mod":  "module example\n\ngo 1.21",
})
// Creates a real git repo with initial commit
```

### Fixtures
```go
files := testing.SampleGoProject()   // Returns map[string]string
files := testing.SamplePythonProject()
```

### Assertions
```go
testing.AssertFileExists(t, "/path/to/file")
testing.AssertFileNotExists(t, "/path/to/file")
testing.AssertFileContains(t, "/path/to/file", "expected content")
```

## Table-Driven Tests Pattern
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## HTTP Mocking for LLM Tests
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Mock SSE streaming response
    w.Header().Set("Content-Type", "text/event-stream")
    w.Write([]byte(`data: {"choices":[{"delta":{"content":"Hello"}}]}` + "\n\n"))
    w.Write([]byte("data: [DONE]\n\n"))
}))
defer server.Close()

client := NewOpenAIClient(config.LLMConfig{BaseURL: server.URL}, nil)
```

## Integration Tests
```go
//go:build integration

package agents_test

func TestAnalyzer_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // ... full agent test with mocked LLM
}
```

## Coverage Requirements
- New code: 80%+
- Critical paths (LLM, tools): 90%+
- Project minimum: 60%
