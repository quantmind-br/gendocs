# Gendocs Implementation Plan
**Version:** 2.0 (Critical Review Edition)
**Created:** 2025-12-23
**Status:** Active Development

---

## 📋 Executive Summary

This plan provides a comprehensive roadmap for Gendocs evolution based on critical analysis of proposed features. The plan prioritizes **stability, testability, and security** over feature velocity.

### Critical Findings from Analysis:
- ✅ **Current State:** ~95% implementation complete, 0% test coverage
- ❌ **Previous Proposals:** 4/6 features rejected or significantly modified
- 🎯 **New Focus:** Foundation before features

### Strategic Priorities:
1. **PHASE 0 (BLOCKER):** Establish testing foundation (0% → 60% coverage)
2. **PHASE 1:** Low-risk quick wins
3. **PHASE 2:** Custom prompt system (high-value, low-risk)
4. **PHASE 3:** HTML export (modified scope)
5. **PHASE 4:** Future features (post-stabilization)

---

## 🏗️ Current State Assessment

### Architecture Overview
```
gendocs/
├── cmd/                     # 5 CLI commands (Cobra)
├── internal/
│   ├── agents/             # 7 agents (1 orchestrator + 5 sub + 2 generators)
│   ├── config/             # Multi-source config (Viper)
│   ├── errors/             # 14 error types
│   ├── gitlab/             # GitLab client
│   ├── handlers/           # 5 command handlers
│   ├── llm/                # 3 providers (OpenAI, Anthropic, Gemini)
│   ├── logging/            # Structured logging (zap)
│   ├── prompts/            # YAML template system
│   ├── tools/              # 2 agent tools (file_read, list_files)
│   ├── tui/                # Bubble Tea config wizard
│   └── worker_pool/        # Semaphore-based concurrency
└── prompts/                # 3 YAML templates
```

### Quality Metrics
| Metric | Current | Target (Phase 0) | Target (Phase 2) |
|--------|---------|------------------|------------------|
| Test Coverage | 0% | 60% | 80% |
| Unit Tests | 0 | 45+ | 80+ |
| Integration Tests | 0 | 3 | 8 |
| Documentation | Basic README | Architecture docs | Complete |
| Security Audit | None | Basic | Full |

### Known Issues
1. **No validation of LLM-generated Markdown** → Can break IDE parsers
2. **Prompts hardcoded in binary** → No user customization
3. **No incremental analysis** → Expensive reruns
4. **GitLab-only automation** → Limits adoption
5. **No output format options** → Markdown only

---

## 🎯 Guiding Principles

### 1. Test-Driven Stability
> **"No new feature without tests for existing code"**

Before adding features, ensure the foundation is solid:
- All LLM clients must have unit tests
- All tools must have unit tests
- Core flows must have integration tests

### 2. Security by Default
> **"LLMs are untrusted output sources"**

- Validate all LLM-generated content
- Sandbox any shell execution
- Never trust LLM outputs without validation

### 3. Simplicity Over Flexibility
> **"Solve today's problem, not tomorrow's hypothetical"**

- Reject plugin systems in favor of static tools
- Reject dynamic configuration in favor of clear precedence
- Reject complexity without proven need

### 4. User Experience First
> **"Configuration should be obvious, not magical"**

- Clear error messages with actionable suggestions
- Predictable configuration precedence
- Fail fast with helpful context

---

## 🚀 PHASE 0: Foundation (BLOCKER)

**Goal:** Establish testing infrastructure and validate existing implementation
**Duration:** 2-3 weeks
**Complexity:** Large
**Blocking:** All other phases

### Objectives
- [ ] Achieve 60% test coverage on critical paths
- [ ] Document architecture and design decisions
- [ ] Identify and fix bugs discovered during testing
- [ ] Establish CI/CD pipeline basics

### Pre-requisites
- None (this is the foundation)

---

### Task 0.1: Test Infrastructure Setup

**Files Created:**
- `internal/testing/helpers.go` - Test utilities and mocks
- `internal/testing/fixtures.go` - Sample data for tests
- `.github/workflows/test.yml` - GitHub Actions CI
- `Makefile` - Test and build automation

**Implementation:**

```go
// internal/testing/helpers.go
package testing

import (
    "context"
    "github.com/user/gendocs/internal/llm"
)

// MockLLMClient implements llm.LLMClient for testing
type MockLLMClient struct {
    Responses []llm.CompletionResponse
    CallCount int
    LastRequest llm.CompletionRequest
}

func NewMockLLMClient(responses ...llm.CompletionResponse) *MockLLMClient {
    return &MockLLMClient{Responses: responses}
}

func (m *MockLLMClient) GenerateCompletion(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
    m.LastRequest = req
    if m.CallCount >= len(m.Responses) {
        return m.Responses[len(m.Responses)-1], nil
    }
    resp := m.Responses[m.CallCount]
    m.CallCount++
    return resp, nil
}

func (m *MockLLMClient) SupportsTools() bool { return true }
func (m *MockLLMClient) GetProvider() string { return "mock" }

// CreateTempRepo creates a temporary repository for testing
func CreateTempRepo(t *testing.T, files map[string]string) string {
    // Implementation...
}
```

**Acceptance Criteria:**
- ✅ Mock LLM client supports all interface methods
- ✅ Temp repo helper creates valid git repositories
- ✅ CI runs on every push to main
- ✅ `make test` runs all tests with coverage report

**Risks:**
- Mocks may not reflect real LLM behavior → Mitigate with integration tests

---

### Task 0.2: LLM Client Unit Tests

**Files Created:**
- `internal/llm/openai_test.go`
- `internal/llm/anthropic_test.go`
- `internal/llm/gemini_test.go`
- `internal/llm/factory_test.go`
- `internal/llm/retry_client_test.go`

**Test Coverage Targets:**
- `openai.go`: 80% (focus on request/response transformation)
- `anthropic.go`: 80% (focus on tool calling format)
- `gemini.go`: 80% (focus on safety settings handling)
- `factory.go`: 90% (all provider creation paths)
- `retry_client.go`: 95% (all retry scenarios)

**Key Test Cases:**

```go
// internal/llm/openai_test.go
package llm

import (
    "testing"
    "context"
    "net/http"
    "net/http/httptest"
)

func TestOpenAIClient_GenerateCompletion_Success(t *testing.T) {
    // Setup mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate request format
        assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

        // Return mock response
        w.Write([]byte(`{
            "choices": [{
                "message": {
                    "content": "test response",
                    "tool_calls": null
                }
            }],
            "usage": {"prompt_tokens": 10, "completion_tokens": 5}
        }`))
    }))
    defer server.Close()

    client := NewOpenAIClient(config.LLMConfig{
        APIKey: "test-key",
        BaseURL: server.URL,
    }, nil)

    resp, err := client.GenerateCompletion(context.Background(), CompletionRequest{
        SystemPrompt: "test",
        Messages: []Message{{Role: "user", Content: "hello"}},
    })

    assert.NoError(t, err)
    assert.Equal(t, "test response", resp.Content)
    assert.Equal(t, 10, resp.Usage.InputTokens)
}

func TestOpenAIClient_GenerateCompletion_WithToolCalls(t *testing.T) {
    // Test tool calling format conversion
}

func TestOpenAIClient_GenerateCompletion_RateLimitRetry(t *testing.T) {
    // Test retry on 429 status
}

func TestOpenAIClient_GenerateCompletion_InvalidAPIKey(t *testing.T) {
    // Test 401 error handling
}
```

**Acceptance Criteria:**
- ✅ All providers tested with mock HTTP servers
- ✅ Tool calling format validated for each provider
- ✅ Retry logic tested with simulated failures
- ✅ Error cases return appropriate error types

---

### Task 0.3: Tools Unit Tests

**Files Created:**
- `internal/tools/file_read_test.go`
- `internal/tools/list_files_test.go`
- `internal/tools/base_test.go`

**Test Coverage Target:** 90%

**Key Test Cases:**

```go
// internal/tools/file_read_test.go
package tools

func TestFileReadTool_Execute_Success(t *testing.T) {
    // Create temp file
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.go")
    content := "package main\n\nfunc main() {}\n"
    os.WriteFile(testFile, []byte(content), 0644)

    tool := NewFileReadTool(tmpDir, 3)
    result, err := tool.Execute(context.Background(), map[string]interface{}{
        "file_path": "test.go",
    })

    assert.NoError(t, err)
    assert.Contains(t, result.(string), "package main")
}

func TestFileReadTool_Execute_PathTraversal(t *testing.T) {
    tmpDir := t.TempDir()
    tool := NewFileReadTool(tmpDir, 3)

    // Attempt directory traversal
    _, err := tool.Execute(context.Background(), map[string]interface{}{
        "file_path": "../../../etc/passwd",
    })

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "path traversal")
}

func TestFileReadTool_Execute_RetryOnTransientError(t *testing.T) {
    // Test ModelRetryError behavior
}
```

**Acceptance Criteria:**
- ✅ Path traversal attempts are rejected
- ✅ Retry logic works for transient errors
- ✅ Large files are handled correctly
- ✅ Non-existent files return clear errors

---

### Task 0.4: Prompt Manager Unit Tests

**Files Created:**
- `internal/prompts/manager_test.go`

**Test Coverage Target:** 85%

**Key Test Cases:**

```go
// internal/prompts/manager_test.go
package prompts

func TestManager_Get_Success(t *testing.T) {
    mgr := NewManagerFromMap(map[string]string{
        "test_system": "You are a test assistant",
        "test_user": "Analyze: {{.RepoPath}}",
    })

    prompt, err := mgr.Get("test_system")
    assert.NoError(t, err)
    assert.Equal(t, "You are a test assistant", prompt)
}

func TestManager_Render_WithVariables(t *testing.T) {
    mgr := NewManagerFromMap(map[string]string{
        "template": "Path: {{.RepoPath}}, Workers: {{.Workers}}",
    })

    result, err := mgr.Render("template", map[string]interface{}{
        "RepoPath": "/test/path",
        "Workers": 4,
    })

    assert.NoError(t, err)
    assert.Equal(t, "Path: /test/path, Workers: 4", result)
}

func TestManager_Render_MissingVariable(t *testing.T) {
    // Should error on missing template variables
}
```

**Acceptance Criteria:**
- ✅ Template rendering works with all variable types
- ✅ Missing variables cause clear errors
- ✅ YAML parsing errors are surfaced properly

---

### Task 0.5: Agent Integration Tests

**Files Created:**
- `internal/agents/analyzer_integration_test.go`
- `internal/agents/documenter_integration_test.go`
- `internal/agents/base_integration_test.go`

**Test Coverage Target:** Basic (3 critical flows)

**Key Test Cases:**

```go
// internal/agents/analyzer_integration_test.go
package agents

func TestAnalyzerAgent_Run_CompleteFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Create test repo
    repoPath := testing.CreateTempRepo(t, map[string]string{
        "main.go": "package main\n\nfunc main() {}\n",
        "go.mod": "module test\n\ngo 1.22\n",
    })

    // Setup mock LLM
    mockClient := testing.NewMockLLMClient(
        // Response 1: Tool call to list_files
        llm.CompletionResponse{
            ToolCalls: []llm.ToolCall{{
                Name: "list_files",
                Arguments: map[string]interface{}{"path": "."},
            }},
        },
        // Response 2: Tool call to read_file
        llm.CompletionResponse{
            ToolCalls: []llm.ToolCall{{
                Name: "read_file",
                Arguments: map[string]interface{}{"file_path": "main.go"},
            }},
        },
        // Response 3: Final analysis
        llm.CompletionResponse{
            Content: "# Code Structure\n\nSimple Go application...",
        },
    )

    // Create analyzer
    cfg := config.AnalyzerConfig{
        BaseConfig: config.BaseConfig{RepoPath: repoPath},
        MaxWorkers: 1,
    }

    analyzer, err := NewAnalyzerAgent(cfg, mockClient, logger)
    assert.NoError(t, err)

    // Run analysis
    result, err := analyzer.Run(context.Background())
    assert.NoError(t, err)
    assert.Contains(t, result, "Code Structure")

    // Verify tool calls happened
    assert.Equal(t, 3, mockClient.CallCount)
}
```

**Acceptance Criteria:**
- ✅ Complete analyze → generate flow works end-to-end
- ✅ Tool calling loop executes correctly
- ✅ Output files are created in .ai/docs/
- ✅ Mock LLM receives correct prompts and context

---

### Task 0.6: Configuration Loading Tests

**Files Created:**
- `internal/config/loader_test.go`
- `internal/config/types_test.go`

**Test Coverage Target:** 85%

**Key Test Cases:**

```go
// internal/config/loader_test.go
package config

func TestLoader_Precedence_CLIOverridesAll(t *testing.T) {
    // Setup: Create temp .ai/config.yaml with values
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, ".ai", "config.yaml")
    os.MkdirAll(filepath.Dir(configPath), 0755)
    os.WriteFile(configPath, []byte(`
analyzer:
  max_workers: 4
  llm:
    provider: openai
`), 0644)

    // Setup: Create ~/.gendocs.yaml
    home := t.TempDir()
    t.Setenv("HOME", home)
    globalConfig := filepath.Join(home, ".gendocs.yaml")
    os.WriteFile(globalConfig, []byte(`
analyzer:
  max_workers: 2
`), 0644)

    // Setup: Environment variable
    t.Setenv("ANALYZER_MAX_WORKERS", "8")

    // Act: Load with CLI override
    cfg, err := LoadAnalyzerConfig(tmpDir, map[string]interface{}{
        "max_workers": 16,
    })

    // Assert: CLI wins (precedence: CLI > project > global > env > default)
    assert.NoError(t, err)
    assert.Equal(t, 16, cfg.MaxWorkers)
}

func TestLoader_EnvVarSubstitution(t *testing.T) {
    // Test that environment variables are properly loaded
}

func TestLoader_MissingAPIKey_ReturnsError(t *testing.T) {
    // Should fail validation if API key missing
}
```

**Acceptance Criteria:**
- ✅ Precedence order is correct and documented
- ✅ Missing required fields cause validation errors
- ✅ Environment variables properly substitute into config
- ✅ Both YAML formats (nested and flat) work

---

### Task 0.7: Error Handling Tests

**Files Created:**
- `internal/errors/errors_test.go`

**Test Coverage Target:** 90%

**Test All 14 Error Types:**
- ConfigFileError
- MissingEnvVarError
- InvalidEnvVarError
- LLMClientError
- ToolExecutionError
- FileNotFoundError
- PathTraversalError
- MarkdownValidationError (new)
- ... (rest)

**Acceptance Criteria:**
- ✅ All error types have user-friendly messages
- ✅ Error wrapping preserves stack traces
- ✅ Error types can be distinguished with errors.As()

---

### Task 0.8: Architecture Documentation

**Files Created:**
- `docs/ARCHITECTURE.md` - System design and patterns
- `docs/TESTING.md` - Testing strategy and guidelines
- `docs/CONTRIBUTING.md` - Developer guide

**Content Structure:**

```markdown
# docs/ARCHITECTURE.md

## System Overview
[High-level architecture diagram]

## Component Responsibilities

### CLI Layer (cmd/)
- Command parsing and flag handling
- Configuration aggregation
- Handler delegation

### Handler Layer (internal/handlers/)
- Business logic orchestration
- Agent lifecycle management
- Error handling and user feedback

### Agent Layer (internal/agents/)
- LLM interaction loops
- Tool calling orchestration
- Analysis coordination

### LLM Layer (internal/llm/)
- Provider abstraction
- Request/response transformation
- Retry and error handling

### Tool Layer (internal/tools/)
- File system operations
- Safety validations
- Retry logic for transient errors

## Design Patterns

### Handler-Agent Pattern
**Problem:** Separate CLI concerns from AI orchestration
**Solution:** Handlers manage configuration and I/O, agents manage LLM interactions

### Factory Pattern
**Used In:** LLM clients, agents
**Benefit:** Decouples creation from usage, enables testing

### Tool Interface
**Purpose:** Extensible agent capabilities without modifying agent code
**Contract:** Execute(ctx, params) -> (result, error)

## Configuration Precedence
1. CLI flags (highest)
2. Project .ai/config.yaml
3. Global ~/.gendocs.yaml
4. Environment variables
5. Hardcoded defaults (lowest)

## Error Handling Strategy
- Errors bubble up with context wrapping
- User-facing errors have clear messages and suggestions
- Technical errors logged with full context
```

**Acceptance Criteria:**
- ✅ Architecture doc explains all major components
- ✅ Design decisions are justified
- ✅ Diagrams show data flow and dependencies
- ✅ Contributing guide explains test requirements

---

### Task 0.9: Markdown Validation

**Files Created:**
- `internal/validation/markdown.go`
- `internal/validation/markdown_test.go`

**Purpose:** Validate LLM-generated Markdown doesn't break IDE parsers

**Implementation:**

```go
// internal/validation/markdown.go
package validation

import (
    "strings"
    "fmt"
)

type MarkdownValidator struct{}

func NewMarkdownValidator() *MarkdownValidator {
    return &MarkdownValidator{}
}

// Validate checks Markdown for common issues
func (v *MarkdownValidator) Validate(content string) error {
    var errors []string

    // Check for unclosed code blocks
    if err := v.checkCodeBlocks(content); err != nil {
        errors = append(errors, err.Error())
    }

    // Check for malformed headers
    if err := v.checkHeaders(content); err != nil {
        errors = append(errors, err.Error())
    }

    // Check for minimum structure
    if err := v.checkMinimumStructure(content); err != nil {
        errors = append(errors, err.Error())
    }

    if len(errors) > 0 {
        return fmt.Errorf("markdown validation failed: %s", strings.Join(errors, "; "))
    }

    return nil
}

func (v *MarkdownValidator) checkCodeBlocks(content string) error {
    openCount := strings.Count(content, "```")
    if openCount%2 != 0 {
        return fmt.Errorf("unclosed code block detected (%d ``` markers)", openCount)
    }
    return nil
}

func (v *MarkdownValidator) checkHeaders(content string) error {
    lines := strings.Split(content, "\n")
    for i, line := range lines {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "#") {
            // Check for space after #
            if len(trimmed) > 1 && trimmed[1] != ' ' && trimmed[1] != '#' {
                return fmt.Errorf("line %d: malformed header (missing space after #)", i+1)
            }
        }
    }
    return nil
}

func (v *MarkdownValidator) checkMinimumStructure(content string) error {
    if len(strings.TrimSpace(content)) < 50 {
        return fmt.Errorf("content too short (less than 50 characters)")
    }
    if !strings.Contains(content, "#") {
        return fmt.Errorf("no headers found in content")
    }
    return nil
}
```

**Integration Points:**
- `internal/agents/base.go:106` - Add validation before returning content
- `internal/handlers/readme.go` - Validate before writing file
- `internal/handlers/ai_rules.go` - Validate before writing file

**Acceptance Criteria:**
- ✅ Unclosed code blocks detected
- ✅ Malformed headers detected
- ✅ Minimum content requirements enforced
- ✅ Clear error messages guide LLM prompt fixes

---

### Phase 0 Summary

**Deliverables:**
- ✅ 60%+ test coverage achieved
- ✅ All LLM clients have unit tests
- ✅ All tools have unit tests
- ✅ 3 integration tests cover critical flows
- ✅ Architecture documented
- ✅ Markdown validation prevents broken output
- ✅ CI/CD pipeline established

**Exit Criteria:**
- `make test` passes with 60%+ coverage
- All tests run in CI on every commit
- Architecture documentation reviewed and approved
- Zero critical bugs in existing functionality

**Estimated Duration:** 2-3 weeks
**Estimated LOC:** ~2,500 (tests) + 500 (validation) + 1,000 (docs)

---

## 🎯 PHASE 1: Quick Wins

**Goal:** Deliver immediate value improvements with minimal risk
**Duration:** 1 week
**Complexity:** Small
**Depends On:** Phase 0 complete

---

### Task 1.1: TUI Environment Variable Detection

**Problem:** Users must manually type API keys even when already in environment

**Solution:** Detect and suggest environment variables in config wizard

**Files Modified:**
- `internal/tui/config.go`

**Implementation:**

```go
// internal/tui/config.go - Add to model struct
type model struct {
    // ... existing fields ...
    detectedEnvVars map[string]string // NEW
}

// Add detection function
func detectEnvironmentVariables() map[string]string {
    detected := make(map[string]string)

    envVars := []string{
        "ANALYZER_LLM_PROVIDER",
        "ANALYZER_LLM_MODEL",
        "ANALYZER_LLM_API_KEY",
        "DOCUMENTER_LLM_PROVIDER",
        "DOCUMENTER_LLM_MODEL",
        "DOCUMENTER_LLM_API_KEY",
        "GITLAB_OAUTH_TOKEN",
    }

    for _, key := range envVars {
        if val := os.Getenv(key); val != "" {
            // Mask API keys for display
            if strings.Contains(key, "API_KEY") || strings.Contains(key, "TOKEN") {
                detected[key] = maskSecret(val)
            } else {
                detected[key] = val
            }
        }
    }

    return detected
}

func maskSecret(s string) string {
    if len(s) <= 8 {
        return "***"
    }
    return s[:4] + "..." + s[len(s)-4:]
}

// Update Init() to show detected values
func (m model) Init() tea.Cmd {
    m.detectedEnvVars = detectEnvironmentVariables()
    return textinput.Blink
}

// Update View() to show suggestions
func (m model) View() string {
    // ... existing code ...

    // If env var exists for current field, show hint
    if detected, exists := m.detectedEnvVars[currentEnvVar]; exists {
        hint := fmt.Sprintf("\n  💡 Detected in environment: %s", detected)
        hint += "\n  Press Enter to use, or type to override"
        return mainView + hint
    }

    return mainView
}
```

**Test Cases:**

```go
// internal/tui/config_test.go
func TestDetectEnvironmentVariables_Found(t *testing.T) {
    t.Setenv("ANALYZER_LLM_API_KEY", "sk-test123456789")

    detected := detectEnvironmentVariables()

    assert.Contains(t, detected, "ANALYZER_LLM_API_KEY")
    assert.Equal(t, "sk-t...789", detected["ANALYZER_LLM_API_KEY"]) // Masked
}

func TestMaskSecret_Short(t *testing.T) {
    assert.Equal(t, "***", maskSecret("short"))
}

func TestMaskSecret_Long(t *testing.T) {
    assert.Equal(t, "sk-t...6789", maskSecret("sk-test123456789"))
}
```

**Acceptance Criteria:**
- ✅ Env vars detected on TUI start
- ✅ API keys masked in display (show first 4 and last 4 chars)
- ✅ User can press Enter to accept or type to override
- ✅ Works for all provider API keys

**Estimated Duration:** 1 day
**Risk:** Low - Purely UI enhancement

---

### Task 1.2: Improved Error Messages

**Problem:** Validation errors are technical and not actionable

**Solution:** Add user-friendly error messages with suggestions

**Files Modified:**
- `internal/errors/errors.go`
- `internal/config/loader.go`

**Implementation:**

```go
// internal/errors/errors.go - Enhance existing error types

func NewMissingEnvVarError(name, description string) *AIDocGenError {
    return &AIDocGenError{
        ErrorType: "MissingEnvVar",
        Message:   fmt.Sprintf("Environment variable %s is not set", name),
        UserMessage: fmt.Sprintf(`Missing required configuration: %s

%s is required but not set. You can fix this by:

1. Setting the environment variable:
   export %s="your-value"

2. Adding to .ai/config.yaml:
   analyzer:
     llm:
       %s: "your-value"

3. Running the configuration wizard:
   gendocs config

Description: %s
`, name, name, name, convertEnvToYAMLKey(name), description),
        Suggestion: fmt.Sprintf("Run 'gendocs config' to set up configuration interactively"),
    }
}

func convertEnvToYAMLKey(envVar string) string {
    // ANALYZER_LLM_API_KEY -> api_key
    parts := strings.Split(envVar, "_")
    if len(parts) > 2 {
        return strings.ToLower(strings.Join(parts[2:], "_"))
    }
    return strings.ToLower(envVar)
}
```

**Test Cases:**

```go
func TestMissingEnvVarError_UserMessage(t *testing.T) {
    err := NewMissingEnvVarError("ANALYZER_LLM_API_KEY", "API key for LLM provider")

    assert.Contains(t, err.UserMessage, "export ANALYZER_LLM_API_KEY")
    assert.Contains(t, err.UserMessage, "gendocs config")
    assert.Contains(t, err.UserMessage, "api_key:")
}
```

**Acceptance Criteria:**
- ✅ Error messages include 3 resolution paths
- ✅ Env var name correctly converts to YAML key
- ✅ Suggestion points to `gendocs config`
- ✅ All existing error types enhanced

**Estimated Duration:** 1 day
**Risk:** Low - Cosmetic improvement

---

### Task 1.3: Max Workers Configuration Flag

**Problem:** Worker pool auto-detection doesn't account for API rate limits

**Solution:** Add explicit `--max-workers` flag (already exists!) and document it

**Files Modified:**
- `README.md` - Add usage examples
- `docs/CONFIGURATION.md` - Document flag and rationale

**Documentation Addition:**

```markdown
## Performance Tuning

### Max Workers Configuration

The `--max-workers` flag controls concurrent agent execution:

```bash
# Auto-detect (uses CPU count)
gendocs analyze --repo-path ./my-project

# Limit to 2 workers (for free-tier API limits)
gendocs analyze --repo-path ./my-project --max-workers 2

# Maximize throughput (for high-tier accounts)
gendocs analyze --repo-path ./my-project --max-workers 8
```

**When to tune:**
- **Free tier APIs:** Use 1-2 workers to avoid rate limits
- **Standard tier:** Use 4-6 workers (default auto-detect)
- **Enterprise tier:** Use 8+ workers for large repos

**Provider-specific recommendations:**
- **OpenAI Free:** 1-2 workers (60 requests/min limit)
- **OpenAI Paid:** 4-8 workers
- **Anthropic Claude:** 2-4 workers (lower rate limits)
- **Gemini:** 4-6 workers
```

**Acceptance Criteria:**
- ✅ Flag documented in README
- ✅ Provider-specific recommendations added
- ✅ Examples show common use cases

**Estimated Duration:** 2 hours
**Risk:** None - Documentation only

---

### Phase 1 Summary

**Deliverables:**
- ✅ TUI detects and suggests environment variables
- ✅ Error messages are actionable and user-friendly
- ✅ Worker configuration documented with guidance

**Exit Criteria:**
- TUI shows detected env vars correctly
- Error messages tested manually and in CI
- Documentation reviewed for clarity

**Estimated Duration:** 1 week
**Estimated LOC:** ~200 (code) + 300 (docs)

---

## 🔧 PHASE 2: Custom Prompts System

**Goal:** Allow users to customize AI behavior without modifying binary
**Duration:** 2 weeks
**Complexity:** Medium
**Depends On:** Phase 0 complete

**Strategic Value:**
- High user demand (enterprise customization)
- Low implementation risk (extends existing system)
- Enables domain-specific documentation styles
- No breaking changes to existing users

---

### Task 2.1: Multi-Directory Prompt Manager

**Problem:** Prompts are loaded from single hardcoded directory

**Solution:** Load from system directory first, then project directory (with override)

**Files Modified:**
- `internal/prompts/manager.go`

**Implementation:**

```go
// internal/prompts/manager.go

type Manager struct {
    prompts map[string]string
    sources map[string]string // Track which file provided each prompt (for debugging)
}

// NewManager creates manager from ONLY system prompts
func NewManager(promptsDir string) (*Manager, error) {
    // Keep existing single-directory implementation
    // This is for backward compatibility
}

// NewManagerWithOverrides creates manager with system + project overrides
func NewManagerWithOverrides(systemDir, projectDir string) (*Manager, error) {
    pm := &Manager{
        prompts: make(map[string]string),
        sources: make(map[string]string),
    }

    // 1. Load system prompts first (baseline)
    if err := pm.loadDirectory(systemDir, "system"); err != nil {
        return nil, fmt.Errorf("failed to load system prompts: %w", err)
    }

    // 2. Load project prompts (overrides)
    if projectDir != "" {
        if _, err := os.Stat(projectDir); err == nil {
            if err := pm.loadDirectory(projectDir, "project"); err != nil {
                return nil, fmt.Errorf("failed to load project prompts: %w", err)
            }
        }
        // If project dir doesn't exist, that's OK - no overrides
    }

    // 3. Validate required prompts exist
    if err := pm.validateRequiredPrompts(); err != nil {
        return nil, err
    }

    return pm, nil
}

// loadDirectory loads all YAML files from a directory
func (pm *Manager) loadDirectory(dir, source string) error {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return fmt.Errorf("failed to read directory %s: %w", dir, err)
    }

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        ext := filepath.Ext(entry.Name())
        if ext != ".yaml" && ext != ".yml" {
            continue
        }

        filePath := filepath.Join(dir, entry.Name())
        data, err := os.ReadFile(filePath)
        if err != nil {
            return fmt.Errorf("failed to read %s: %w", filePath, err)
        }

        var prompts map[string]string
        if err := yaml.Unmarshal(data, &prompts); err != nil {
            return fmt.Errorf("failed to parse %s: %w", filePath, err)
        }

        // Merge into main map (later loads override earlier)
        for key, value := range prompts {
            pm.prompts[key] = value
            pm.sources[key] = fmt.Sprintf("%s:%s", source, entry.Name())
        }
    }

    return nil
}

// validateRequiredPrompts ensures critical prompts exist
func (pm *Manager) validateRequiredPrompts() error {
    required := []string{
        "structure_analyzer_system",
        "structure_analyzer_user",
        "dependency_analyzer_system",
        "dependency_analyzer_user",
        "data_flow_analyzer_system",
        "data_flow_analyzer_user",
        "request_flow_analyzer_system",
        "request_flow_analyzer_user",
        "api_analyzer_system",
        "api_analyzer_user",
        "documenter_system_prompt",
        "documenter_user_prompt",
        "ai_rules_system_prompt",
        "ai_rules_user_prompt",
    }

    var missing []string
    for _, key := range required {
        if _, ok := pm.prompts[key]; !ok {
            missing = append(missing, key)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("missing required prompts: %v", missing)
    }

    return nil
}

// GetSource returns which file provided a prompt (for debugging)
func (pm *Manager) GetSource(name string) string {
    return pm.sources[name]
}

// ListOverrides returns all prompts that were overridden from project
func (pm *Manager) ListOverrides() []string {
    var overrides []string
    for key, source := range pm.sources {
        if strings.HasPrefix(source, "project:") {
            overrides = append(overrides, key)
        }
    }
    return overrides
}
```

**Usage Changes:**

```go
// internal/handlers/analyze.go - Update to use new manager

// OLD:
promptManager, err := prompts.NewManager("prompts/")

// NEW:
systemPromptsDir := "prompts/" // Embedded in binary
projectPromptsDir := filepath.Join(cfg.RepoPath, ".ai/prompts")

promptManager, err := prompts.NewManagerWithOverrides(systemPromptsDir, projectPromptsDir)
if err != nil {
    return fmt.Errorf("failed to load prompts: %w", err)
}

// Log overrides for transparency
if overrides := promptManager.ListOverrides(); len(overrides) > 0 {
    logger.Info("Using custom prompts from project",
        logging.Strings("overrides", overrides),
    )
}
```

**Test Cases:**

```go
// internal/prompts/manager_test.go

func TestNewManagerWithOverrides_SystemOnly(t *testing.T) {
    systemDir := t.TempDir()
    createYAML(t, systemDir, "analyzer.yaml", map[string]string{
        "test_system": "System value",
    })

    mgr, err := NewManagerWithOverrides(systemDir, "")
    assert.NoError(t, err)

    val, _ := mgr.Get("test_system")
    assert.Equal(t, "System value", val)
    assert.Equal(t, "system:analyzer.yaml", mgr.GetSource("test_system"))
}

func TestNewManagerWithOverrides_ProjectOverrides(t *testing.T) {
    systemDir := t.TempDir()
    projectDir := t.TempDir()

    createYAML(t, systemDir, "analyzer.yaml", map[string]string{
        "test_system": "System value",
        "other_prompt": "Not overridden",
    })

    createYAML(t, projectDir, "analyzer.yaml", map[string]string{
        "test_system": "PROJECT OVERRIDE",
    })

    mgr, err := NewManagerWithOverrides(systemDir, projectDir)
    assert.NoError(t, err)

    // Should get project override
    val, _ := mgr.Get("test_system")
    assert.Equal(t, "PROJECT OVERRIDE", val)
    assert.Equal(t, "project:analyzer.yaml", mgr.GetSource("test_system"))

    // Should get system value for non-overridden
    val, _ = mgr.Get("other_prompt")
    assert.Equal(t, "Not overridden", val)
    assert.Equal(t, "system:analyzer.yaml", mgr.GetSource("other_prompt"))

    // Verify override tracking
    overrides := mgr.ListOverrides()
    assert.Contains(t, overrides, "test_system")
    assert.NotContains(t, overrides, "other_prompt")
}

func TestNewManagerWithOverrides_MissingRequired(t *testing.T) {
    systemDir := t.TempDir()
    // Don't create any files - all required prompts missing

    _, err := NewManagerWithOverrides(systemDir, "")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "missing required prompts")
}

func TestNewManagerWithOverrides_ProjectDirNotExists(t *testing.T) {
    systemDir := t.TempDir()
    createYAML(t, systemDir, "analyzer.yaml", map[string]string{
        "test_system": "System value",
    })

    // Project dir doesn't exist - should not error
    mgr, err := NewManagerWithOverrides(systemDir, "/nonexistent")
    assert.NoError(t, err)

    val, _ := mgr.Get("test_system")
    assert.Equal(t, "System value", val)
}

func createYAML(t *testing.T, dir, filename string, data map[string]string) {
    t.Helper()
    content, _ := yaml.Marshal(data)
    os.WriteFile(filepath.Join(dir, filename), content, 0644)
}
```

**Acceptance Criteria:**
- ✅ System prompts loaded from binary directory
- ✅ Project prompts override system prompts
- ✅ Missing project directory is not an error
- ✅ Missing required prompts cause clear error
- ✅ Override tracking works for debugging
- ✅ Backward compatibility maintained (NewManager still works)

---

### Task 2.2: Documentation and Examples

**Files Created:**
- `docs/CUSTOM_PROMPTS.md` - User guide for customization
- `.ai/prompts/README.md` - Template with examples
- `examples/custom-prompts/` - Example overrides

**Content:**

```markdown
# docs/CUSTOM_PROMPTS.md

# Custom Prompts Guide

## Overview

Gendocs allows you to customize AI analysis behavior by overriding system prompts. This enables:
- Domain-specific documentation (e.g., financial, medical, gaming)
- Company-specific conventions
- Language/tone customization
- Focus on specific architectural concerns

## Precedence

Prompts are loaded in this order:
1. **System prompts** (embedded in `prompts/` directory) - Baseline
2. **Project prompts** (`.ai/prompts/` in your repo) - Override system

## Quick Start

### 1. Create Prompts Directory

```bash
mkdir -p .ai/prompts
```

### 2. Create Override File

Create `.ai/prompts/analyzer.yaml`:

```yaml
# Override the structure analyzer system prompt
structure_analyzer_system: |
  You are a code structure analyst specializing in microservices architecture.

  CRITICAL: This codebase follows Domain-Driven Design (DDD) principles.
  Focus heavily on:
  - Bounded contexts and their boundaries
  - Aggregate roots and entities
  - Domain events and event sourcing

  Your goal is to produce a comprehensive DDD-focused analysis.

# Keep the user prompt unchanged (don't include it to use system default)
```

### 3. Run Analysis

```bash
gendocs analyze --repo-path .
```

Gendocs will log which prompts were overridden:
```
INFO Using custom prompts from project overrides=["structure_analyzer_system"]
```

## Available Prompts

### Structure Analyzer
- `structure_analyzer_system` - How to analyze code structure
- `structure_analyzer_user` - What to look for (usually don't override)

### Dependency Analyzer
- `dependency_analyzer_system` - How to analyze dependencies
- `dependency_analyzer_user` - What to document

### Data Flow Analyzer
- `data_flow_analyzer_system` - How to trace data flow
- `data_flow_analyzer_user` - What to track

### Request Flow Analyzer
- `request_flow_analyzer_system` - How to trace requests
- `request_flow_analyzer_user` - What to document

### API Analyzer
- `api_analyzer_system` - How to analyze APIs
- `api_analyzer_user` - What to document

### Documenter (README Generator)
- `documenter_system_prompt` - How to write documentation
- `documenter_user_prompt` - What to include

### AI Rules Generator
- `ai_rules_system_prompt` - How to generate AI assistant rules
- `ai_rules_user_prompt` - What to include

## Examples

### Example 1: Focus on Security

```yaml
# .ai/prompts/analyzer.yaml
structure_analyzer_system: |
  You are a security-focused code structure analyst.

  PRIORITY: Identify and document security-relevant components:
  - Authentication and authorization mechanisms
  - Input validation and sanitization
  - Encryption and key management
  - Security boundaries and trust zones

  Flag any concerning patterns immediately.
```

### Example 2: Corporate Tone

```yaml
# .ai/prompts/documenter.yaml
documenter_system_prompt: |
  You are a technical writer for ACME Corp.

  Writing style:
  - Professional and formal tone
  - Use "we" instead of "you"
  - Include compliance references (SOC2, HIPAA)
  - Executive summaries at the top

  Format all documentation following ACME Corp style guide.
```

### Example 3: Framework-Specific

```yaml
# .ai/prompts/analyzer.yaml
structure_analyzer_system: |
  You are a Django web framework specialist.

  Focus on Django-specific patterns:
  - Apps and their purposes
  - Models and ORM relationships
  - Views (function-based vs class-based)
  - URL routing patterns
  - Middleware stack
  - Custom management commands

  Use Django terminology throughout.
```

## Best Practices

### DO:
✅ Start with small overrides and test
✅ Keep system prompt style (clear instructions)
✅ Focus on "how to think" not "what to output"
✅ Version control `.ai/prompts/` with your code
✅ Document why you're overriding in comments

### DON'T:
❌ Override all prompts at once
❌ Make prompts too prescriptive (limits AI creativity)
❌ Remove critical instructions from system prompts
❌ Use overly complex templating
❌ Expect LLM to follow 100% of instructions

## Troubleshooting

### Changes Not Applied

1. Check gendocs logs for "Using custom prompts" message
2. Verify YAML syntax: `yamllint .ai/prompts/*.yaml`
3. Check prompt key names match exactly (case-sensitive)
4. Clear `.ai/docs/` and re-run analysis

### Invalid YAML Error

```bash
# Validate syntax
yamllint .ai/prompts/analyzer.yaml

# Common issues:
# - Unescaped special characters
# - Incorrect indentation (use 2 spaces)
# - Missing | after key for multiline strings
```

### Prompts Too Long

LLMs have context limits. If overrides are very long:
- Keep system prompts under 1000 words
- Focus on high-level instructions, not examples
- Use user prompts for specific examples

## Reference

See system prompts in binary:
- `prompts/analyzer.yaml`
- `prompts/documenter.yaml`
- `prompts/ai_rules_generator.yaml`

Or view online: https://github.com/user/gendocs/tree/main/prompts
```

**Acceptance Criteria:**
- ✅ Documentation explains precedence clearly
- ✅ 3+ real-world examples provided
- ✅ Best practices and troubleshooting included
- ✅ All available prompts listed with descriptions

---

### Task 2.3: Validation and Error Handling

**Problem:** Invalid project prompts could break analysis

**Solution:** Validate project prompts and provide clear errors

**Files Modified:**
- `internal/prompts/manager.go`

**Implementation:**

```go
// Add to Manager

// ValidatePromptContent checks if a prompt is well-formed
func (pm *Manager) ValidatePromptContent(key, content string) error {
    // Check length
    if len(content) < 20 {
        return fmt.Errorf("prompt '%s' is too short (minimum 20 characters)", key)
    }

    if len(content) > 10000 {
        return fmt.Errorf("prompt '%s' is too long (maximum 10,000 characters)", key)
    }

    // Check for template syntax errors (if it uses Go templates)
    if strings.Contains(content, "{{") {
        _, err := textTemplate.New("test").Parse(content)
        if err != nil {
            return fmt.Errorf("prompt '%s' has invalid template syntax: %w", key, err)
        }
    }

    // Check for common mistakes
    if strings.Count(content, "{{") != strings.Count(content, "}}") {
        return fmt.Errorf("prompt '%s' has unmatched template braces", key)
    }

    return nil
}

// Update loadDirectory to validate
func (pm *Manager) loadDirectory(dir, source string) error {
    // ... existing loading code ...

    for key, value := range prompts {
        // Validate content
        if err := pm.ValidatePromptContent(key, value); err != nil {
            if source == "project" {
                // Project prompts: error out
                return fmt.Errorf("invalid project prompt in %s: %w", filePath, err)
            } else {
                // System prompts: log warning and skip
                // (shouldn't happen, but defensive)
                fmt.Fprintf(os.Stderr, "WARNING: %v\n", err)
                continue
            }
        }

        pm.prompts[key] = value
        pm.sources[key] = fmt.Sprintf("%s:%s", source, entry.Name())
    }

    return nil
}
```

**Test Cases:**

```go
func TestValidatePromptContent_TooShort(t *testing.T) {
    mgr := &Manager{}
    err := mgr.ValidatePromptContent("test", "short")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "too short")
}

func TestValidatePromptContent_TooLong(t *testing.T) {
    mgr := &Manager{}
    longPrompt := strings.Repeat("a", 10001)
    err := mgr.ValidatePromptContent("test", longPrompt)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "too long")
}

func TestValidatePromptContent_InvalidTemplate(t *testing.T) {
    mgr := &Manager{}
    err := mgr.ValidatePromptContent("test", "Hello {{.InvalidSyntax")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid template syntax")
}

func TestLoadDirectory_InvalidProjectPrompt_Errors(t *testing.T) {
    projectDir := t.TempDir()
    createYAML(t, projectDir, "bad.yaml", map[string]string{
        "test": "too short",
    })

    mgr := &Manager{prompts: make(map[string]string), sources: make(map[string]string)}
    err := mgr.loadDirectory(projectDir, "project")

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid project prompt")
}
```

**Acceptance Criteria:**
- ✅ Invalid project prompts cause clear errors
- ✅ Template syntax validated before use
- ✅ Length limits prevent abuse
- ✅ Error messages include file and key name

---

### Task 2.4: Logging and Observability

**Problem:** Users don't know if overrides are working

**Solution:** Log override activity with details

**Files Modified:**
- `internal/handlers/analyze.go`
- `internal/handlers/readme.go`
- `internal/handlers/ai_rules.go`

**Implementation:**

```go
// internal/handlers/analyze.go

func (h *AnalyzeHandler) Handle(ctx context.Context) error {
    // ... existing setup ...

    // Load prompts with overrides
    systemPromptsDir := "prompts/"
    projectPromptsDir := filepath.Join(h.config.RepoPath, ".ai/prompts")

    promptManager, err := prompts.NewManagerWithOverrides(systemPromptsDir, projectPromptsDir)
    if err != nil {
        return fmt.Errorf("failed to load prompts: %w", err)
    }

    // Log override information
    overrides := promptManager.ListOverrides()
    if len(overrides) > 0 {
        h.logger.Info("Using custom prompts from project",
            logging.Int("count", len(overrides)),
            logging.Strings("prompts", overrides),
        )

        // Detailed logging in debug mode
        if h.config.Debug {
            for _, key := range overrides {
                source := promptManager.GetSource(key)
                h.logger.Debug("Prompt override detail",
                    logging.String("key", key),
                    logging.String("source", source),
                )
            }
        }
    } else {
        h.logger.Info("Using system prompts (no project overrides found)")
    }

    // ... continue with analysis ...
}
```

**Console Output Examples:**

```bash
# No overrides
$ gendocs analyze --repo-path .
INFO Using system prompts (no project overrides found)

# With overrides
$ gendocs analyze --repo-path .
INFO Using custom prompts from project count=2 prompts=[structure_analyzer_system, documenter_system_prompt]

# Debug mode shows details
$ gendocs analyze --repo-path . --debug
INFO Using custom prompts from project count=2 prompts=[...]
DEBUG Prompt override detail key=structure_analyzer_system source=project:analyzer.yaml
DEBUG Prompt override detail key=documenter_system_prompt source=project:documenter.yaml
```

**Acceptance Criteria:**
- ✅ Override count logged on every run
- ✅ Override list logged in info mode
- ✅ Detailed sources logged in debug mode
- ✅ No overrides case handled gracefully

---

### Phase 2 Summary

**Deliverables:**
- ✅ Multi-directory prompt manager with override support
- ✅ Comprehensive documentation with examples
- ✅ Validation prevents invalid prompts
- ✅ Logging provides transparency
- ✅ Backward compatibility maintained

**Exit Criteria:**
- Tests pass with 85%+ coverage for prompt system
- Documentation reviewed by 2+ team members
- Integration tests cover override scenarios
- Example overrides tested in real analysis

**Estimated Duration:** 2 weeks
**Estimated LOC:** ~600 (code) + ~1,200 (docs + examples)

---

## 📄 PHASE 3: HTML Export

**Goal:** Generate HTML documentation from Markdown
**Duration:** 1-2 weeks
**Complexity:** Medium
**Depends On:** Phase 0 complete

**Scope Clarification:**
- ✅ Basic HTML generation from README.md
- ✅ Simple CSS for readability
- ✅ Syntax highlighting for code blocks
- ❌ NOT: Multi-page site generation (future)
- ❌ NOT: PDF export (requires headless browser)
- ❌ NOT: Wiki integration (too specific)

---

### Task 3.1: HTML Renderer Implementation

**Files Created:**
- `internal/export/html.go`
- `internal/export/html_test.go`
- `internal/export/templates/` - HTML templates
- `internal/export/assets/` - CSS files

**Dependencies:**
```bash
go get github.com/yuin/goldmark
go get github.com/alecthomas/chroma/v2
go get github.com/yuin/goldmark-highlighting/v2
```

**Implementation:**

```go
// internal/export/html.go
package export

import (
    "bytes"
    "fmt"
    "html/template"
    "os"
    "path/filepath"

    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark-highlighting/v2"
    "github.com/yuin/goldmark/extension"
    "github.com/yuin/goldmark/renderer/html"
)

type HTMLExporter struct {
    markdown     goldmark.Markdown
    htmlTemplate *template.Template
}

type HTMLDocument struct {
    Title   string
    Content template.HTML
    CSS     string
}

func NewHTMLExporter() (*HTMLExporter, error) {
    // Configure Goldmark with extensions
    md := goldmark.New(
        goldmark.WithExtensions(
            extension.GFM,
            extension.Table,
            extension.Strikethrough,
            extension.TaskList,
            highlighting.NewHighlighting(
                highlighting.WithStyle("monokai"),
            ),
        ),
        goldmark.WithRendererOptions(
            html.WithHardWraps(),
            html.WithXHTML(),
            html.WithUnsafe(), // Allow raw HTML in Markdown
        ),
    )

    // Load HTML template
    tmpl, err := loadHTMLTemplate()
    if err != nil {
        return nil, fmt.Errorf("failed to load HTML template: %w", err)
    }

    return &HTMLExporter{
        markdown:     md,
        htmlTemplate: tmpl,
    }, nil
}

// ExportToHTML converts Markdown file to standalone HTML
func (e *HTMLExporter) ExportToHTML(markdownPath, outputPath string) error {
    // Read Markdown
    mdContent, err := os.ReadFile(markdownPath)
    if err != nil {
        return fmt.Errorf("failed to read markdown: %w", err)
    }

    // Convert to HTML
    var buf bytes.Buffer
    if err := e.markdown.Convert(mdContent, &buf); err != nil {
        return fmt.Errorf("failed to convert markdown: %w", err)
    }

    // Extract title (first H1)
    title := extractTitle(string(mdContent))

    // Render full HTML document
    doc := HTMLDocument{
        Title:   title,
        Content: template.HTML(buf.String()),
        CSS:     getDefaultCSS(),
    }

    var htmlBuf bytes.Buffer
    if err := e.htmlTemplate.Execute(&htmlBuf, doc); err != nil {
        return fmt.Errorf("failed to execute template: %w", err)
    }

    // Write output
    if err := os.WriteFile(outputPath, htmlBuf.Bytes(), 0644); err != nil {
        return fmt.Errorf("failed to write HTML: %w", err)
    }

    return nil
}

func loadHTMLTemplate() (*template.Template, error) {
    const tmpl = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="generator" content="Gendocs">
    <title>{{.Title}}</title>
    <style>
        {{.CSS}}
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="generator-badge">Generated with Gendocs</div>
        </header>
        <main>
            {{.Content}}
        </main>
        <footer>
            <p>Generated on {{now}} by <a href="https://github.com/user/gendocs">Gendocs</a></p>
        </footer>
    </div>
</body>
</html>`

    return template.New("html").Funcs(template.FuncMap{
        "now": func() string {
            return time.Now().Format("2006-01-02 15:04:05")
        },
    }).Parse(tmpl)
}

func extractTitle(markdown string) string {
    lines := strings.Split(markdown, "\n")
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "# ") {
            return strings.TrimPrefix(trimmed, "# ")
        }
    }
    return "Documentation"
}

func getDefaultCSS() string {
    return `
        * {
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: #24292f;
            background-color: #ffffff;
            margin: 0;
            padding: 0;
        }

        .container {
            max-width: 980px;
            margin: 0 auto;
            padding: 45px;
        }

        header {
            border-bottom: 1px solid #d0d7de;
            margin-bottom: 30px;
            padding-bottom: 10px;
        }

        .generator-badge {
            font-size: 12px;
            color: #57606a;
            text-align: right;
        }

        main {
            margin-bottom: 60px;
        }

        h1, h2, h3, h4, h5, h6 {
            margin-top: 24px;
            margin-bottom: 16px;
            font-weight: 600;
            line-height: 1.25;
        }

        h1 {
            font-size: 2em;
            border-bottom: 1px solid #d0d7de;
            padding-bottom: 0.3em;
        }

        h2 {
            font-size: 1.5em;
            border-bottom: 1px solid #d0d7de;
            padding-bottom: 0.3em;
        }

        code {
            background-color: rgba(175, 184, 193, 0.2);
            border-radius: 6px;
            font-size: 85%;
            margin: 0;
            padding: 0.2em 0.4em;
            font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace;
        }

        pre {
            background-color: #f6f8fa;
            border-radius: 6px;
            font-size: 85%;
            line-height: 1.45;
            overflow: auto;
            padding: 16px;
        }

        pre code {
            background-color: transparent;
            border: 0;
            display: inline;
            line-height: inherit;
            margin: 0;
            overflow: visible;
            padding: 0;
            word-wrap: normal;
        }

        table {
            border-collapse: collapse;
            border-spacing: 0;
            width: 100%;
            margin-bottom: 16px;
        }

        table th {
            font-weight: 600;
            background-color: #f6f8fa;
        }

        table th, table td {
            padding: 6px 13px;
            border: 1px solid #d0d7de;
        }

        table tr:nth-child(2n) {
            background-color: #f6f8fa;
        }

        a {
            color: #0969da;
            text-decoration: none;
        }

        a:hover {
            text-decoration: underline;
        }

        blockquote {
            padding: 0 1em;
            color: #57606a;
            border-left: 0.25em solid #d0d7de;
            margin: 0 0 16px;
        }

        footer {
            border-top: 1px solid #d0d7de;
            padding-top: 20px;
            text-align: center;
            font-size: 14px;
            color: #57606a;
        }

        @media (max-width: 768px) {
            .container {
                padding: 15px;
            }
        }
    `
}
```

**Test Cases:**

```go
// internal/export/html_test.go
package export

func TestHTMLExporter_ExportToHTML_Success(t *testing.T) {
    exporter, err := NewHTMLExporter()
    assert.NoError(t, err)

    // Create temp Markdown file
    tmpDir := t.TempDir()
    mdFile := filepath.Join(tmpDir, "test.md")
    htmlFile := filepath.Join(tmpDir, "test.html")

    markdown := `# Test Document

This is a **test** with code:

` + "```go\nfunc main() {}\n```" + `

## Section 2

- Item 1
- Item 2
`

    os.WriteFile(mdFile, []byte(markdown), 0644)

    // Export
    err = exporter.ExportToHTML(mdFile, htmlFile)
    assert.NoError(t, err)

    // Verify output
    html, err := os.ReadFile(htmlFile)
    assert.NoError(t, err)
    assert.Contains(t, string(html), "<h1>Test Document</h1>")
    assert.Contains(t, string(html), "<strong>test</strong>")
    assert.Contains(t, string(html), "<code")
    assert.Contains(t, string(html), "func main")
}

func TestExtractTitle(t *testing.T) {
    tests := []struct {
        markdown string
        expected string
    }{
        {"# My Title\n\nContent", "My Title"},
        {"Some content\n# Title", "Title"},
        {"No title here", "Documentation"},
        {"", "Documentation"},
    }

    for _, tt := range tests {
        assert.Equal(t, tt.expected, extractTitle(tt.markdown))
    }
}
```

**Acceptance Criteria:**
- ✅ Markdown converts to valid HTML5
- ✅ Code blocks have syntax highlighting
- ✅ Tables render correctly
- ✅ Output is responsive (mobile-friendly)
- ✅ CSS is embedded (single-file output)

---

### Task 3.2: Export Command Implementation

**Files Modified:**
- `cmd/generate.go` - Add `export` subcommand

**Implementation:**

```go
// cmd/generate.go - Add to existing file

var exportFormat string
var exportOutput string

var exportCmd = &cobra.Command{
    Use:   "export",
    Short: "Export documentation to different formats",
    Long: `Export generated documentation to formats like HTML for easier sharing.

Supported formats:
  - html: Standalone HTML file with embedded CSS and syntax highlighting

Examples:
  # Export README.md to HTML
  gendocs generate export --repo-path . --format html --output docs.html

  # Export specific file
  gendocs generate export --repo-path . --input .ai/docs/code_structure.md --format html
`,
    RunE: runExport,
}

func init() {
    generateCmd.AddCommand(exportCmd)

    exportCmd.Flags().StringVar(&exportFormat, "format", "html", "Export format (html)")
    exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: input.html)")
    exportCmd.Flags().StringVar(&repoPath, "repo-path", ".", "Path to repository")
    exportCmd.Flags().StringVar(&exportInput, "input", "README.md", "Input markdown file")
}

var exportInput string

func runExport(cmd *cobra.Command, args []string) error {
    // Determine input file
    inputPath := exportInput
    if !filepath.IsAbs(inputPath) {
        inputPath = filepath.Join(repoPath, inputPath)
    }

    // Check input exists
    if _, err := os.Stat(inputPath); err != nil {
        return fmt.Errorf("input file not found: %s", inputPath)
    }

    // Determine output file
    outputPath := exportOutput
    if outputPath == "" {
        ext := filepath.Ext(inputPath)
        outputPath = strings.TrimSuffix(inputPath, ext) + ".html"
    }

    // Export based on format
    switch exportFormat {
    case "html":
        return exportToHTML(inputPath, outputPath)
    default:
        return fmt.Errorf("unsupported format: %s (supported: html)", exportFormat)
    }
}

func exportToHTML(inputPath, outputPath string) error {
    fmt.Printf("Exporting %s to %s...\n", inputPath, outputPath)

    exporter, err := export.NewHTMLExporter()
    if err != nil {
        return fmt.Errorf("failed to create exporter: %w", err)
    }

    if err := exporter.ExportToHTML(inputPath, outputPath); err != nil {
        return fmt.Errorf("export failed: %w", err)
    }

    fmt.Printf("✓ HTML exported to %s\n", outputPath)
    return nil
}
```

**Acceptance Criteria:**
- ✅ `gendocs generate export` command works
- ✅ Defaults to README.md → README.html
- ✅ Custom input/output paths supported
- ✅ Clear error if input file missing
- ✅ Success message shows output location

---

### Task 3.3: Integration with Generate Workflow

**Problem:** Users might want HTML generated automatically after README

**Solution:** Add optional `--export-html` flag to generate command

**Files Modified:**
- `cmd/generate.go`

**Implementation:**

```go
// Add to generateReadmeCmd
var autoExportHTML bool

func init() {
    // ... existing flags ...
    generateReadmeCmd.Flags().BoolVar(&autoExportHTML, "export-html", false, "Also export to HTML after generation")
}

func runGenerateReadme(cmd *cobra.Command, args []string) error {
    // ... existing README generation ...

    if autoExportHTML {
        readmePath := filepath.Join(repoPath, "README.md")
        htmlPath := filepath.Join(repoPath, "README.html")

        fmt.Println("\nExporting to HTML...")
        if err := exportToHTML(readmePath, htmlPath); err != nil {
            // Don't fail the whole command, just warn
            fmt.Fprintf(os.Stderr, "Warning: HTML export failed: %v\n", err)
        }
    }

    return nil
}
```

**Acceptance Criteria:**
- ✅ `--export-html` flag generates both MD and HTML
- ✅ HTML export failure doesn't fail entire command
- ✅ Both files created in repo root

---

### Task 3.4: Documentation

**Files Modified:**
- `README.md` - Add export examples
- `docs/EXPORT.md` - Detailed export guide

**Content:**

```markdown
# docs/EXPORT.md

# Documentation Export

## HTML Export

Generate standalone HTML documentation for easy sharing.

### Basic Usage

```bash
# Export README.md to HTML
gendocs generate export --format html

# Export specific file
gendocs generate export --input .ai/docs/code_structure.md --output structure.html

# Generate README and export in one command
gendocs generate readme --export-html
```

### Features

- ✅ **Standalone**: Single HTML file with embedded CSS
- ✅ **Syntax Highlighting**: Code blocks with Monokai theme
- ✅ **Responsive**: Mobile-friendly design
- ✅ **GitHub-Style**: Familiar Markdown rendering
- ✅ **Tables & Lists**: Full GFM support

### Use Cases

**Sharing with Non-Technical Stakeholders**
```bash
gendocs generate readme --export-html
# Send README.html to executives
```

**Internal Wiki/Confluence**
```bash
gendocs generate export --input README.md --output docs.html
# Upload docs.html to wiki
```

**Offline Documentation**
```bash
gendocs generate export --format html
# Open README.html in browser (works offline)
```

### Customization

#### Custom CSS (Future)

Currently, CSS is embedded and not customizable. For custom styling:

1. Generate HTML with default CSS
2. Edit HTML file to replace `<style>` block
3. Or use external CSS processor

### Limitations

- Single file only (no multi-page generation)
- No PDF export (requires headless browser)
- No custom themes (default GitHub style)
- Images must be embedded or use absolute URLs

### Future Enhancements

Planned for Phase 4:
- Multi-page site generation
- Custom CSS themes
- PDF export via headless Chrome
- Search functionality
```

**Acceptance Criteria:**
- ✅ Documentation explains all use cases
- ✅ Examples are tested and work
- ✅ Limitations clearly stated
- ✅ Future enhancements listed

---

### Phase 3 Summary

**Deliverables:**
- ✅ HTML exporter with Goldmark + syntax highlighting
- ✅ `gendocs generate export` command
- ✅ Optional `--export-html` flag for README generation
- ✅ Comprehensive documentation

**Exit Criteria:**
- HTML exports render correctly in all major browsers
- Syntax highlighting works for 10+ languages
- Mobile rendering tested on real devices
- Export tested with all analyzer output files

**Estimated Duration:** 1-2 weeks
**Estimated LOC:** ~500 (code) + ~400 (docs + templates)

---

## 🔮 PHASE 4: Future Enhancements

**Status:** Planned, not committed
**Trigger:** After Phase 0-3 complete AND user demand validated

This phase contains features that were analyzed and either:
- Deferred due to complexity
- Require more validation
- Depend on infrastructure not yet built

---

### Feature 4.1: Incremental Analysis Cache

**Acceptance Criteria for Starting:**
- [ ] Dependency graph analysis implemented (new feature)
- [ ] Cache invalidation strategy designed and reviewed
- [ ] Performance benchmarks show >50% time savings on large repos

**Estimated Complexity:** Large (3-4 weeks)

**Rough Plan:**
1. Build file dependency graph analyzer
2. Implement hash-based change detection
3. Design cascade invalidation algorithm
4. Implement cache storage (SQLite or JSON)
5. Add cache management CLI commands
6. Extensive testing on real repos

---

### Feature 4.2: GitHub Support in Cronjob

**Acceptance Criteria for Starting:**
- [ ] GitLab client refactored to interface abstraction
- [ ] Integration tests for GitLab client at 80%
- [ ] GitHub API client designed (OAuth vs GitHub Apps decided)

**Estimated Complexity:** Large (3-4 weeks)

**Rough Plan:**
1. Create `internal/git/provider.go` interface
2. Refactor `internal/gitlab/client.go` to implement interface
3. Implement `internal/github/client.go`
4. Add GitHub App authentication support
5. Update cronjob handler for multi-provider
6. Add provider selection CLI flag

---

### Feature 4.3: Security Tool Integration

**Acceptance Criteria for Starting:**
- [ ] User survey shows demand for security analysis
- [ ] Chosen tools (gosec, nancy, trivy) evaluated
- [ ] Integration approach designed (direct vs subprocess)

**Estimated Complexity:** Medium (2-3 weeks)

**Rough Plan:**
1. Create `internal/security/scanner.go`
2. Integrate gosec for static analysis
3. Integrate nancy for dependency scanning
4. Integrate trivy for secret detection
5. Generate `security_analysis.md` output
6. Add `--exclude-security` flag

**Note:** Replaces rejected "LLM-based security analyzer" with real tools.

---

### Feature 4.4: Multi-Page HTML Site Generation

**Acceptance Criteria for Starting:**
- [ ] Basic HTML export (Phase 3) used by 10+ users
- [ ] Feedback shows need for multi-page navigation
- [ ] Static site generator chosen (Hugo, Jekyll, or custom)

**Estimated Complexity:** Medium (2 weeks)

**Rough Plan:**
1. Design site structure (index + per-analyzer page)
2. Create navigation templates
3. Implement cross-linking between pages
4. Add search functionality (lunr.js)
5. Generate `site/` directory structure

---

### Feature 4.5: PDF Export

**Acceptance Criteria for Starting:**
- [ ] HTML export (Phase 3) validated as high-quality
- [ ] PDF generation approach decided (wkhtmltopdf vs Chrome headless)
- [ ] User demand validated (10+ requests)

**Estimated Complexity:** Small-Medium (1-2 weeks)

**Rough Plan:**
1. Integrate Chrome DevTools Protocol or wkhtmltopdf
2. Design PDF layout (page breaks, headers, footers)
3. Add `--format pdf` to export command
4. Handle images and styling for print
5. Test across different PDF readers

---

### Phase 4 Decision Framework

**Before starting ANY Phase 4 feature, answer:**

1. **User Demand:** Do we have 10+ user requests?
2. **Dependencies:** Are all prerequisites complete?
3. **Risk:** Can we rollback if it fails?
4. **Test Coverage:** Is foundation at 80%+?
5. **Maintenance:** Can we support it long-term?

**If any answer is NO, defer the feature.**

---

## 📊 Success Metrics

### Phase 0 (Foundation)
- ✅ Test coverage ≥ 60%
- ✅ CI build time < 5 minutes
- ✅ Zero P0 bugs in existing features
- ✅ Architecture documented

### Phase 1 (Quick Wins)
- ✅ TUI env detection used by 100% of new users
- ✅ Error messages result in <30% support tickets
- ✅ Positive feedback on UX improvements

### Phase 2 (Custom Prompts)
- ✅ 10+ users create custom prompts
- ✅ Zero prompt-related crashes
- ✅ Documentation rated 4+/5 stars
- ✅ 3+ community-shared prompt examples

### Phase 3 (HTML Export)
- ✅ HTML exports render in Chrome, Firefox, Safari
- ✅ Mobile rendering works on iOS & Android
- ✅ Export used by 20%+ of users
- ✅ Zero HTML validation errors

### Phase 4 (Future)
- ✅ Each feature shows clear user demand
- ✅ Each feature maintains 80%+ test coverage
- ✅ No feature increases build time >20%

---

## 🎯 Implementation Timeline

### Q1 2025
- **Weeks 1-3:** Phase 0 (Foundation)
- **Week 4:** Phase 1 (Quick Wins)

### Q2 2025
- **Weeks 1-2:** Phase 2 (Custom Prompts)
- **Weeks 3-4:** Phase 3 (HTML Export)

### Q3+ 2025
- **As Needed:** Phase 4 features based on demand

---

## 🚨 Risk Management

### High-Risk Areas

**1. LLM API Changes**
- **Risk:** Providers change API without notice
- **Mitigation:** Integration tests detect breakage; pin API versions

**2. Test Coverage Slips**
- **Risk:** New features added without tests
- **Mitigation:** CI blocks merge if coverage drops below 60%

**3. Scope Creep**
- **Risk:** Phase 4 features pulled into earlier phases
- **Mitigation:** Strict phase gating; decision framework

**4. User Prompt Injection**
- **Risk:** Malicious users craft prompts to exploit LLMs
- **Mitigation:** Prompt validation; LLM output validation

### Mitigation Strategies

**For Each Phase:**
- Daily standups during active development
- Weekly code reviews
- Bi-weekly demo to stakeholders
- Monthly user feedback sessions

**Quality Gates:**
- No merge without tests
- No merge without documentation
- No merge with failing CI
- No merge with <60% coverage

---

## 📝 Appendix

### Rejected Features (Do Not Implement)

1. **Dynamic Plugin System for Tools**
   - Reason: Security risk outweighs benefits
   - Alternative: Static tool additions

2. **LLM-Based Security Analysis**
   - Reason: High false positive rate
   - Alternative: gosec/nancy/trivy integration

3. **Wiki Format Export**
   - Reason: Too many wiki formats, unclear demand
   - Alternative: HTML export covers most use cases

### Deferred Features (Maybe Later)

1. **Bitbucket Support**
   - Defer Until: GitHub support complete and validated

2. **Self-Hosted LLM Support (Ollama)**
   - Defer Until: 50+ user requests

3. **VSCode Extension**
   - Defer Until: CLI usage at 1000+ users

---

## 🔄 Plan Maintenance

**This plan is a living document.**

### Review Cadence
- **Weekly:** Progress tracking during active phases
- **Monthly:** Metrics review and priority adjustment
- **Quarterly:** Strategic review and Phase 4 evaluation

### Change Process
1. Propose change in GitHub issue
2. Discuss in team meeting
3. Update PLAN.md via PR
4. Increment version number
5. Announce in changelog

### Version History
- **v2.0** (2025-12-23): Critical review edition (this version)
- **v1.0** (2025-12-20): Original shotgun prompt proposals (rejected)

---

**END OF PLAN**

---

## Quick Reference

### Phase Checklist

- [ ] **Phase 0:** Foundation & Testing
  - [ ] Test infrastructure
  - [ ] LLM client tests
  - [ ] Tool tests
  - [ ] Prompt manager tests
  - [ ] Integration tests
  - [ ] Config tests
  - [ ] Error handling tests
  - [ ] Architecture docs
  - [ ] Markdown validation

- [ ] **Phase 1:** Quick Wins
  - [ ] TUI env detection
  - [ ] Enhanced errors
  - [ ] Config documentation

- [ ] **Phase 2:** Custom Prompts
  - [ ] Multi-directory manager
  - [ ] Documentation
  - [ ] Validation
  - [ ] Logging

- [ ] **Phase 3:** HTML Export
  - [ ] HTML renderer
  - [ ] Export command
  - [ ] Auto-export flag
  - [ ] Documentation

- [ ] **Phase 4:** Future (Conditional)
  - TBD based on Phase 0-3 outcomes

### File Tree After Phase 3

```
gendocs/
├── cmd/
│   ├── analyze.go              [Modified: logging]
│   ├── config.go               [Modified: env detection]
│   ├── generate.go             [Modified: export command]
│   └── ...
├── internal/
│   ├── agents/
│   │   ├── analyzer_integration_test.go     [NEW]
│   │   └── ...
│   ├── config/
│   │   ├── loader_test.go                   [NEW]
│   │   └── ...
│   ├── errors/
│   │   ├── errors_test.go                   [NEW]
│   │   └── errors.go                        [Modified]
│   ├── export/                              [NEW]
│   │   ├── html.go
│   │   ├── html_test.go
│   │   └── templates/
│   ├── llm/
│   │   ├── openai_test.go                   [NEW]
│   │   ├── anthropic_test.go                [NEW]
│   │   ├── gemini_test.go                   [NEW]
│   │   └── ...
│   ├── prompts/
│   │   ├── manager.go                       [Modified: overrides]
│   │   └── manager_test.go                  [NEW]
│   ├── testing/                             [NEW]
│   │   ├── helpers.go
│   │   └── fixtures.go
│   ├── tools/
│   │   ├── file_read_test.go                [NEW]
│   │   ├── list_files_test.go               [NEW]
│   │   └── ...
│   ├── tui/
│   │   ├── config.go                        [Modified: env detection]
│   │   └── config_test.go                   [NEW]
│   └── validation/                          [NEW]
│       ├── markdown.go
│       └── markdown_test.go
├── docs/                                    [NEW]
│   ├── ARCHITECTURE.md
│   ├── TESTING.md
│   ├── CONTRIBUTING.md
│   ├── CUSTOM_PROMPTS.md
│   └── EXPORT.md
├── examples/                                [NEW]
│   └── custom-prompts/
│       ├── ddd-focus.yaml
│       ├── security-focus.yaml
│       └── django-focus.yaml
├── .github/
│   └── workflows/
│       └── test.yml                         [NEW]
├── Makefile                                 [NEW]
├── PLAN.md                                  [This file]
└── README.md                                [Modified: examples]
```

---

**Total Estimated Effort:**
- Phase 0: 2-3 weeks
- Phase 1: 1 week
- Phase 2: 2 weeks
- Phase 3: 1-2 weeks
- **Total: 6-8 weeks** (with 1 developer)

**Total Estimated LOC:**
- Tests: ~2,500
- New Features: ~1,300
- Documentation: ~2,200
- **Total: ~6,000 LOC**
