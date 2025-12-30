# Gendocs - Code Style & Conventions

## File Naming
- Use `snake_case` for file names: `file_read.go`, `cached_client.go`
- Test files: `*_test.go`
- Integration tests: `*_integration_test.go`
- Benchmarks: `*_bench_test.go`

## Package Naming
- Use lowercase single words: `llm`, `config`, `agents`

## Type Naming
| Element | Convention | Example |
|---------|------------|---------|
| Interfaces | Noun/verb | `LLMClient`, `Tool`, `Agent` |
| Structs | PascalCase | `AnthropicClient`, `BaseAgent` |
| Constants | PascalCase | `MaxConversationTokens` |

## Import Groups (3 groups, separated by blank lines)
```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/user/gendocs/internal/config"
)
```

## Error Handling
```go
// Always wrap with context using %w
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// For user-facing errors, use custom types
return errors.WrapError(err, "LLM API call failed", errors.ExitLLMError)
```

## Context Propagation
Always accept `context.Context` as first parameter:
```go
func (h *Handler) Handle(ctx context.Context) error {
    return h.agent.Run(ctx)
}
```

## Logging (Zap structured)
```go
logger.Info("Starting analysis",
    logging.String("repo", repoPath),
    logging.Int("workers", maxWorkers),
)
```

## Test Naming
Follow pattern: `Test<Type>_<Method>_<Scenario>`
```go
func TestOpenAIClient_GenerateCompletion_Success(t *testing.T)
func TestAnthropicClient_ParseSSE_InvalidJSON(t *testing.T)
```

## Patterns Used
- **Factory Pattern**: LLM client creation, agent creation
- **Interface-based Design**: All major components have interfaces
- **Worker Pool**: Parallel file hashing, sub-agent execution
- **Tool-calling Loop**: Agents iterate LLM calls with tool execution
- **Retry with Backoff**: `RetryClient` wraps HTTP calls

## Prompts
- YAML files in `prompts/` directory
- Go template syntax: `{{.Variable}}`
- Support custom overrides via configuration
