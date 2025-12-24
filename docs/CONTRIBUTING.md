# Contributing to Gendocs

Thank you for your interest in contributing to Gendocs! This document provides guidelines and instructions for contributing.

---

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Getting Started](#getting-started)
3. [Development Setup](#development-setup)
4. [Development Workflow](#development-workflow)
5. [Testing Requirements](#testing-requirements)
6. [Code Style](#code-style)
7. [Commit Guidelines](#commit-guidelines)
8. [Pull Request Process](#pull-request-process)
9. [Adding Features](#adding-features)

---

## Code of Conduct

- Be respectful and constructive
- Focus on the issue, not the person
- Accept constructive criticism gracefully
- Prioritize project goals over personal preferences

---

## Getting Started

### Prerequisites

- **Go 1.22+** installed
- **Git** configured
- **Make** utility (optional but recommended)
- API key for at least one LLM provider (OpenAI, Anthropic, or Gemini)

### Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/gendocs.git
cd gendocs

# Add upstream remote
git remote add upstream https://github.com/user/gendocs.git
```

---

## Development Setup

### 1. Install Dependencies

```bash
go mod download
go mod verify
```

### 2. Set Up Environment

```bash
# Copy example env file
cp .env.example .env

# Edit with your API keys
nano .env
```

Example `.env`:
```bash
ANALYZER_LLM_PROVIDER=openai
ANALYZER_LLM_MODEL=gpt-4
ANALYZER_LLM_API_KEY=sk-...

DOCUMENTER_LLM_PROVIDER=openai
DOCUMENTER_LLM_MODEL=gpt-4
DOCUMENTER_LLM_API_KEY=sk-...
```

### 3. Build

```bash
make build
# Or: go build -o gendocs .
```

### 4. Verify Installation

```bash
./gendocs --version
./gendocs --help
```

---

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/my-new-feature
# or
git checkout -b fix/bug-description
```

**Branch Naming:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test improvements

### 2. Make Changes

Follow these principles:
- **Small, focused changes** - One feature/fix per branch
- **Test as you go** - Write tests alongside code
- **Document as you go** - Update docs with changes

### 3. Run Tests Frequently

```bash
# Quick check (runs in ~5s)
make test-short

# Full test suite
make test

# With coverage
make test-coverage
```

### 4. Run Linters

```bash
# Install golangci-lint first:
# https://golangci-lint.run/usage/install/

make lint
```

### 5. Commit Changes

```bash
git add .
git commit -m "feat: add new feature"
# See Commit Guidelines below for message format
```

### 6. Keep Branch Updated

```bash
git fetch upstream
git rebase upstream/main
```

### 7. Push and Create PR

```bash
git push origin feature/my-new-feature
```

Then create a Pull Request on GitHub.

---

## Testing Requirements

### Minimum Requirements

All contributions **must** include tests:

1. **New Features** → Unit tests + integration tests (if applicable)
2. **Bug Fixes** → Test that reproduces the bug + fix
3. **Refactoring** → Existing tests must pass
4. **Documentation** → No tests required (but examples should work)

### Coverage Standards

- **New code:** 80%+ coverage required
- **Overall project:** Must not decrease below 60%

### Running Tests

```bash
# Unit tests only (fast)
make test-short

# All tests
make test

# Integration tests
go test -tags integration ./...

# Specific package
go test ./internal/llm/

# Specific test
go test -run TestOpenAIClient_GenerateCompletion_Success ./internal/llm/
```

### Writing Tests

See [TESTING.md](TESTING.md) for detailed guidelines.

**Quick Example:**

```go
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

---

## Code Style

### Go Conventions

Follow standard Go conventions:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run ./...
```

### Style Guidelines

1. **Package Comments**

```go
// Package llm provides abstractions for LLM providers.
//
// This package implements a common interface for OpenAI,
// Anthropic, and Gemini APIs.
package llm
```

2. **Function Comments**

```go
// GenerateCompletion sends a completion request to the LLM.
//
// The function handles tool calling loops automatically,
// executing tools and returning results to the LLM until
// a final response is generated.
func GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // ...
}
```

3. **Variable Names**

```go
// Good - descriptive
var analyzerConfig *config.AnalyzerConfig
var maxWorkers int

// Bad - abbreviations
var cfg *config.AnalyzerConfig
var mw int
```

4. **Error Handling**

```go
// Good - wrap errors with context
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Bad - lose context
if err != nil {
    return err
}
```

5. **Imports Organization**

```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // External packages
    "github.com/spf13/cobra"

    // Internal packages
    "github.com/user/gendocs/internal/config"
    "github.com/user/gendocs/internal/llm"
)
```

---

## Commit Guidelines

### Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions or fixes
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `chore:` - Build/tooling changes

### Examples

```bash
# Feature
feat(llm): add support for Claude 3.5 Sonnet

Implements support for Anthropic's Claude 3.5 Sonnet model
with improved tool calling.

Closes #123

# Bug fix
fix(tools): prevent path traversal in FileReadTool

Adds validation to ensure file paths cannot escape the
repository directory.

# Documentation
docs(readme): update installation instructions

# Test
test(llm): add integration tests for retry logic

# Breaking change
feat(config)!: change config file format to YAML

BREAKING CHANGE: Configuration files now use YAML format
instead of JSON. Users must migrate their configs.
```

### Commit Best Practices

✅ **DO:**
- Write descriptive commit messages
- Keep commits focused and atomic
- Reference issues in commit messages

❌ **DON'T:**
- Commit unrelated changes together
- Use vague messages like "fix bug" or "update code"
- Commit commented-out code or debug statements

---

## Pull Request Process

### Before Creating PR

1. ✅ All tests pass (`make test`)
2. ✅ Linters pass (`make lint`)
3. ✅ Code formatted (`go fmt ./...`)
4. ✅ Branch rebased on latest main
5. ✅ Commit messages follow guidelines
6. ✅ Documentation updated (if applicable)

### PR Title

Use same format as commit messages:

```
feat(llm): add support for Claude 3.5 Sonnet
fix(config): handle missing API key gracefully
docs: add examples for custom prompts
```

### PR Description

Use this template:

```markdown
## Summary
Brief description of what this PR does

## Changes
- Change 1
- Change 2
- Change 3

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Tests pass
- [ ] Linters pass
- [ ] Documentation updated
- [ ] Breaking changes documented

## Related Issues
Closes #123
Relates to #456
```

### PR Review Process

1. **Automated Checks** run on every PR:
   - Tests must pass
   - Linters must pass
   - Coverage must not decrease

2. **Code Review** by maintainer:
   - Architecture fit
   - Code quality
   - Test coverage
   - Documentation

3. **Changes Requested** (if needed):
   - Address feedback
   - Push updates to same branch
   - Request re-review

4. **Merge** when approved:
   - Squash merge for clean history
   - Delete branch after merge

---

## Adding Features

### Adding a New LLM Provider

1. **Implement Interface**

```go
// internal/llm/newprovider.go
type NewProviderClient struct {
    *BaseLLMClient
    apiKey string
    model  string
}

func NewNewProviderClient(cfg config.LLMConfig, retryClient *RetryClient) *NewProviderClient {
    return &NewProviderClient{
        BaseLLMClient: NewBaseLLMClient(retryClient),
        apiKey:        cfg.APIKey,
        model:         cfg.Model,
    }
}

func (c *NewProviderClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // Transform request to provider format
    // Make API call
    // Transform response to unified format
}

func (c *NewProviderClient) SupportsTools() bool {
    return true
}

func (c *NewProviderClient) GetProvider() string {
    return "newprovider"
}
```

2. **Register in Factory**

```go
// internal/llm/factory.go
func (f *Factory) NewLLMClient(cfg config.LLMConfig) (LLMClient, error) {
    switch cfg.Provider {
    case "openai":
        return NewOpenAIClient(cfg, retryClient), nil
    case "anthropic":
        return NewAnthropicClient(cfg, retryClient), nil
    case "gemini":
        return NewGeminiClient(cfg, retryClient), nil
    case "newprovider": // Add here
        return NewNewProviderClient(cfg, retryClient), nil
    }
}
```

3. **Add Tests**

```go
// internal/llm/newprovider_test.go
func TestNewProviderClient_GenerateCompletion_Success(t *testing.T) { ... }
func TestNewProviderClient_GenerateCompletion_WithToolCalls(t *testing.T) { ... }
func TestNewProviderClient_GenerateCompletion_InvalidAPIKey(t *testing.T) { ... }
```

4. **Update Documentation**

- Add to `README.md` supported providers
- Add to `docs/ARCHITECTURE.md` LLM layer
- Add configuration example

### Adding a New Tool

1. **Implement Interface**

```go
// internal/tools/newtool.go
type NewTool struct {
    BaseTool
}

func NewNewTool(maxRetries int) *NewTool {
    return &NewTool{
        BaseTool: NewBaseTool(maxRetries),
    }
}

func (t *NewTool) Name() string {
    return "new_tool"
}

func (t *NewTool) Description() string {
    return "Description of what the tool does"
}

func (t *NewTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *NewTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    return t.RetryableExecute(ctx, func() (interface{}, error) {
        // Implement tool logic
    })
}
```

2. **Add Tests**

```go
// internal/tools/newtool_test.go
func TestNewTool_Execute_Success(t *testing.T) { ... }
func TestNewTool_Execute_InvalidParams(t *testing.T) { ... }
```

3. **Register with Agents**

```go
// internal/agents/analyzer.go
tools := []tools.Tool{
    tools.NewFileReadTool(3),
    tools.NewListFilesTool(3),
    tools.NewNewTool(3), // Add here
}
```

4. **Update Documentation**

### Adding a New Command

1. **Create Command File**

```go
// cmd/newcommand.go
var newCmd = &cobra.Command{
    Use:   "newcommand",
    Short: "Short description",
    Long:  `Long description`,
    RunE:  runNewCommand,
}

func init() {
    rootCmd.AddCommand(newCmd)
    newCmd.Flags().StringVar(&flagVar, "flag", "default", "Flag description")
}

func runNewCommand(cmd *cobra.Command, args []string) error {
    // Implementation
}
```

2. **Create Handler**

```go
// internal/handlers/newhandler.go
type NewHandler struct {
    config config.NewConfig
    logger *logging.Logger
}

func NewNewHandler(cfg config.NewConfig, logger *logging.Logger) *NewHandler {
    return &NewHandler{config: cfg, logger: logger}
}

func (h *NewHandler) Handle(ctx context.Context) error {
    // Implementation
}
```

3. **Add Tests**

4. **Update Documentation**

---

## Questions?

- **Issues:** https://github.com/user/gendocs/issues
- **Discussions:** https://github.com/user/gendocs/discussions
- **Email:** maintainers@gendocs.dev (if exists)

---

**Thank you for contributing!** 🎉

Every contribution, no matter how small, helps make Gendocs better.

---

**Document Version:** 1.0
**Last Updated:** 2025-12-23
