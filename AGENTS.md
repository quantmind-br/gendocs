# AGENTS.md - Gendocs Agent Guidelines

> Guidelines for AI coding agents operating in this Go codebase.

## Build / Lint / Test Commands

```bash
# Build
make build                    # Binary to build/gendocs-{os}-{arch}
go build -o gendocs .         # Quick local build

# Test - Full Suite
make test                     # All tests, race detection (5m timeout)
make test-short               # Unit tests only (2m timeout)
make test-coverage            # Tests + coverage report

# Test - Single Test (IMPORTANT)
go test -v -run TestOpenAIClient_GenerateCompletion ./internal/llm/
go test -v -run TestAnalyzer ./internal/agents/
go test -v -run "TestCachedClient_Get.*" ./internal/llm/   # Pattern match

# Test - Single Package
go test -v ./internal/llm/
go test -v ./internal/agents/

# Integration Tests (build tag required)
go test -v -tags integration ./internal/agents/

# Lint
make lint                     # golangci-lint
go fmt ./...                  # Format all
```

## Project Structure

```
cmd/                    # CLI commands (Cobra)
internal/
  agents/               # AI agents (analyzer, documenter, ai_rules_generator)
  cache/                # Analysis result caching
  config/               # Configuration loading (Viper)
  errors/               # Custom error types with exit codes
  export/               # HTML/JSON exporters
  handlers/             # Business logic orchestration
  llm/                  # LLM clients (OpenAI, Anthropic, Gemini)
  llmcache/             # LLM response caching
  prompts/              # YAML prompt management
  testing/              # Test helpers and fixtures
  tools/                # Agent tools (file_read, list_files)
  tui/                  # Terminal UI (Bubbletea)
prompts/                # YAML prompt templates
```

## Code Style

### Imports (3 groups: stdlib, external, internal)
```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/user/gendocs/internal/config"
)
```

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Files | snake_case | `file_read.go`, `cached_client.go` |
| Packages | lowercase | `llm`, `config`, `llmcache` |
| Interfaces | Noun/verb | `LLMClient`, `Tool`, `Agent` |
| Structs | PascalCase | `AnthropicClient`, `BaseAgent` |
| Tests | `Test<Type>_<Method>_<Scenario>` | `TestOpenAIClient_GenerateCompletion_Success` |
| Unexported | camelCase | `anthropicRequest`, `openaiMessage` |

### Error Handling
```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)  // Always wrap with %w
}
return errors.WrapError(err, "LLM API failed", errors.ExitLLMError)  // User-facing
```

### Context Propagation
Always accept `context.Context` as first parameter:
```go
func (c *Client) GenerateCompletion(ctx context.Context, req Request) (Response, error)
```

### Logging (Zap structured)
```go
logger.Info("Starting analysis", logging.String("repo", repoPath), logging.Int("workers", n))
```

## Testing Patterns

### Table-Driven Tests
```go
tests := []struct{ name, input, want string; wantErr bool }{
    {"valid", "test", "TEST", false},
    {"empty", "", "", true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := MyFunction(tt.input)
        if (err != nil) != tt.wantErr { t.Errorf("error = %v, wantErr %v", err, tt.wantErr) }
    })
}
```

### Test Helpers (`internal/testing/`)
```go
repoPath := testHelpers.CreateTempRepo(t, map[string]string{"main.go": "package main"})
mock := testHelpers.NewMockLLMClient(llm.CompletionResponse{Content: "response"})
testHelpers.AssertFileExists(t, path)
testHelpers.AssertFileContains(t, path, "expected")
```

### HTTP Mocking for LLM Tests
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Write([]byte(`data: {"choices":[{"delta":{"content":"Hello"}}]}` + "\n\n"))
}))
defer server.Close()
client := NewOpenAIClient(config.LLMConfig{BaseURL: server.URL}, nil)
```

### Integration Tests
```go
//go:build integration

func TestAnalyzer_Integration(t *testing.T) {
    if testing.Short() { t.Skip("Skipping integration test in short mode") }
    // ...
}
```

## Architecture

- **Handler-Agent Separation**: Handlers orchestrate I/O; Agents handle LLM loops
- **Factory Pattern**: LLM client creation in `internal/llm/factory.go`
- **Tool Interface**: Agents use tools via `Tool.Execute()` interface
- **Worker Pool**: Parallel execution in `internal/worker_pool/`

## Local LLM Support

The application supports local LLM providers:

| Provider | Default BaseURL | API Key Required |
|----------|-----------------|------------------|
| Ollama | http://localhost:11434/v1 | No |
| LM Studio | http://localhost:1234/v1 | No |

Both providers use OpenAI-compatible API format. The LLM factory routes these to the OpenAI client internally.

### Relevant Files
- `internal/tui/dashboard/sections/llm.go` - TUI provider selection and BaseURL auto-population
- `internal/llm/factory.go` - Factory routing for local providers

### Testing Local Providers
```bash
go test -v -run ".*Ollama.*" ./internal/tui/dashboard/sections/
go test -v -run ".*Ollama.*" ./internal/llm/
go test -v -run ".*LMStudio.*" ./internal/llm/
```

## TUI Analysis Runner

The TUI Dashboard (`gendocs config`) supports running analysis directly with real-time progress:

### Key Files
- `internal/tui/dashboard/progress_reporter.go` - Bridges `agents.ProgressReporter` to Bubble Tea messages
- `internal/tui/dashboard/progress_view.go` - Visual progress component with spinner, task states
- `internal/tui/dashboard/analysis_messages.go` - Bubble Tea message types for analysis flow

### Message Flow
1. `RunAnalysisMsg` - Triggers analysis from button press
2. `AnalysisProgressMsg` - Task status updates (added/started/completed/failed/skipped)
3. `AnalysisCompleteMsg` - Final summary with success/failure counts
4. `CancelAnalysisMsg` - User-initiated cancellation via Esc key
5. `AnalysisCancelledMsg` - Confirms cancellation completed

### Key Patterns
- Analysis runs in a goroutine, communicates via `tea.Program.Send()`
- `TUIProgressReporter` implements `agents.ProgressReporter` interface
- Context cancellation enables graceful shutdown
- Progress view overlay replaces main content during analysis
- Spinner animation at 100ms intervals via `TickMsg`

### Testing
- Unit tests: `progress_reporter_test.go`, `progress_view_test.go`
- Integration tests: `analysis_runner_test.go` (build tag: `integration`)
- Run: `go test -tags integration ./internal/tui/dashboard/...`

## Key Dependencies

`cobra` (CLI) | `viper` (Config) | `zap` (Logging) | `bubbletea` (TUI)

## Configuration Precedence

1. CLI flags > 2. `.ai/config.yaml` > 3. `~/.gendocs.yaml` > 4. Env vars > 5. Defaults

## Gotchas

1. **LLM streaming**: Clients parse SSE format with buffered readers
2. **Tool retries**: `BaseTool.RetryableExecute()` handles transient failures
3. **Path validation**: Tools prevent directory traversal outside repo
4. **Prompts**: YAML files with Go template syntax `{{.Variable}}`
5. **Test imports**: Use `testHelpers "github.com/user/gendocs/internal/testing"`

## Coverage Requirements

| Scope | Target |
|-------|--------|
| New code | 80%+ |
| Critical (LLM, tools) | 90%+ |
| Project minimum | 60% |

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
