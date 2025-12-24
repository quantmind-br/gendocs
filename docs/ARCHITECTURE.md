# Gendocs Architecture

**Version:** 2.0 (Go Implementation)
**Last Updated:** 2025-12-23

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Architectural Layers](#architectural-layers)
3. [Component Responsibilities](#component-responsibilities)
4. [Design Patterns](#design-patterns)
5. [Data Flow](#data-flow)
6. [Configuration System](#configuration-system)
7. [Error Handling Strategy](#error-handling-strategy)
8. [Concurrency Model](#concurrency-model)
9. [Testing Strategy](#testing-strategy)
10. [Extension Points](#extension-points)

---

## System Overview

Gendocs is a CLI tool that analyzes codebases and generates comprehensive documentation using Large Language Models (LLMs). The system employs a multi-agent architecture where specialized AI agents analyze different aspects of the code.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│              (Cobra Commands + Flag Parsing)                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                      Handler Layer                           │
│         (Business Logic Orchestration + I/O)                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                       Agent Layer                            │
│    (AnalyzerAgent + 5 Sub-Agents + 2 Generator Agents)      │
└────────┬────────────────────────────────────┬───────────────┘
         │                                    │
┌────────▼────────┐                  ┌───────▼───────────────┐
│   LLM Layer     │                  │     Tool Layer        │
│  (3 Providers)  │                  │ (File Operations)     │
└─────────────────┘                  └───────────────────────┘
```

### Core Components

- **7 AI Agents:** 1 orchestrator + 5 analyzers + 2 generators
- **3 LLM Providers:** OpenAI, Anthropic Claude, Google Gemini
- **2 Tools:** FileReadTool, ListFilesTool
- **5 Commands:** analyze, generate (readme, ai-rules), cronjob, config

---

## Architectural Layers

### 1. CLI Layer (`cmd/`)

**Responsibility:** Command parsing and user interaction

**Components:**
- `root.go` - Root command and global flags
- `analyze.go` - Code analysis command
- `generate.go` - Documentation generation commands
- `cronjob.go` - Automated GitLab processing
- `config.go` - Interactive configuration wizard

**Key Decisions:**
- Uses Cobra for command routing
- Minimal business logic (delegated to handlers)
- Validates flags but not business rules

### 2. Handler Layer (`internal/handlers/`)

**Responsibility:** Business logic orchestration

**Pattern:** Handler-Agent separation

```go
type Handler interface {
    Handle(ctx context.Context) error
}
```

**Components:**
- `AnalyzeHandler` - Orchestrates code analysis flow
- `READMEHandler` - Generates README.md
- `AIRulesHandler` - Generates CLAUDE.md/AGENTS.md
- `CronjobHandler` - Processes GitLab repositories in batch

**Responsibilities:**
1. Load and validate configuration
2. Initialize agents with proper dependencies
3. Coordinate agent execution
4. Write output files
5. Handle errors and provide user feedback

### 3. Agent Layer (`internal/agents/`)

**Responsibility:** AI-powered analysis and generation

**Architecture:** Orchestrator + Sub-Agents

```
AnalyzerAgent (Orchestrator)
    ├─ StructureAnalyzer
    ├─ DependencyAnalyzer
    ├─ DataFlowAnalyzer
    ├─ RequestFlowAnalyzer
    └─ APIAnalyzer

DocumenterAgent (Standalone)

AIRulesGeneratorAgent (Standalone)
```

**Base Agent Structure:**

```go
type Agent interface {
    Run(ctx context.Context) (string, error)
    Name() string
}

type BaseAgent struct {
    name          string
    llmClient     llm.LLMClient
    tools         []tools.Tool
    promptManager *prompts.Manager
    logger        *logging.Logger
    systemPrompt  string
    maxRetries    int
    maxTokens     int
    temperature   float64
}
```

**Tool Calling Loop:**

```
1. Send prompt + tools to LLM
2. LLM responds with content or tool calls
3. If tool calls:
   a. Execute each tool
   b. Add results to conversation
   c. Return to step 1
4. If no tool calls, return content
```

### 4. LLM Layer (`internal/llm/`)

**Responsibility:** LLM provider abstraction

**Interface:**

```go
type LLMClient interface {
    GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    SupportsTools() bool
    GetProvider() string
}
```

**Providers:**
- `OpenAIClient` - OpenAI API (GPT-4, GPT-3.5)
- `AnthropicClient` - Anthropic Claude API
- `GeminiClient` - Google Gemini API

**Common Features:**
- Retry logic with exponential backoff
- Tool calling support (function calling)
- Token usage tracking
- Error normalization

### 5. Tool Layer (`internal/tools/`)

**Responsibility:** Agent capabilities

**Interface:**

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}
```

**Available Tools:**
- `FileReadTool` - Read file contents with pagination
- `ListFilesTool` - List files recursively

**Safety Features:**
- Path traversal prevention
- Retry logic for transient errors
- Parameter validation

---

## Component Responsibilities

### Configuration (`internal/config/`)

**Precedence Order (highest to lowest):**
1. CLI flags
2. Project `.ai/config.yaml`
3. Global `~/.gendocs.yaml`
4. Environment variables
5. Hardcoded defaults

**Example:**

```yaml
# .ai/config.yaml
analyzer:
  max_workers: 4
  llm:
    provider: anthropic
    model: claude-3-sonnet
    api_key: sk-ant-...
```

**Configuration Loading:**

```go
cfg, err := config.LoadAnalyzerConfig(repoPath, cliOverrides)
// cfg contains merged configuration with proper precedence
```

### Prompt Management (`internal/prompts/`)

**Responsibility:** Template-based prompt generation

**Storage:** YAML files in `prompts/` directory

```yaml
# prompts/analyzer.yaml
structure_analyzer_system: |
  You are a code structure analyst...

structure_analyzer_user: |
  Analyze the repository at {{.RepoPath}}
```

**Rendering:**

```go
prompt, err := promptManager.Render("structure_analyzer_user", map[string]interface{}{
    "RepoPath": "/path/to/repo",
})
```

### Logging (`internal/logging/`)

**Dual Output:**
- **Console:** Colored, human-readable (using zap)
- **File:** Structured JSON for debugging

**Log Levels:**
- DEBUG: Detailed execution flow
- INFO: Major operations
- WARN: Recoverable issues
- ERROR: Failures

**Usage:**

```go
logger.Info("Starting analysis",
    logging.String("repo", repoPath),
    logging.Int("workers", maxWorkers),
)
```

### Error Handling (`internal/errors/`)

**14 Error Types:**

```go
type AIDocGenError struct {
    ErrorType   string // e.g., "MissingEnvVar"
    Message     string // Technical message
    UserMessage string // User-friendly message
    Suggestion  string // How to fix
    Context     map[string]interface{}
}
```

**Examples:**
- `ConfigFileError` - Invalid YAML
- `MissingEnvVarError` - Required env var not set
- `LLMClientError` - LLM API failure
- `ToolExecutionError` - Tool failed
- `PathTraversalError` - Security violation

### Worker Pool (`internal/worker_pool/`)

**Purpose:** Concurrent agent execution

**Implementation:** Semaphore-based

```go
pool := worker_pool.NewWorkerPool(maxWorkers) // 0 = auto-detect CPUs

tasks := []Task{
    func(ctx) { return analyzeStructure() },
    func(ctx) { return analyzeDependencies() },
    func(ctx) { return analyzeDataFlow() },
}

results := pool.Run(ctx, tasks) // Blocks until all complete
```

**Features:**
- Automatic CPU detection (`runtime.NumCPU()`)
- Context-aware cancellation
- Maintains result order

---

## Design Patterns

### 1. Handler-Agent Pattern

**Problem:** Separate CLI concerns from AI orchestration

**Solution:**

```
CLI Command → Handler → Agent → LLM
     ↓           ↓        ↓       ↓
  Flags     Config    Tools   API
```

**Benefits:**
- Testable agents (mock LLM client)
- Clean separation of concerns
- Reusable agents across commands

### 2. Factory Pattern

**Used In:** LLM clients, Agents

```go
func NewLLMClient(cfg config.LLMConfig) (llm.LLMClient, error) {
    switch cfg.Provider {
    case "openai":
        return NewOpenAIClient(cfg), nil
    case "anthropic":
        return NewAnthropicClient(cfg), nil
    case "gemini":
        return NewGeminiClient(cfg), nil
    }
}
```

**Benefits:**
- Decouples creation from usage
- Enables testing with mocks
- Centralized provider logic

### 3. Tool Interface Pattern

**Purpose:** Extensible agent capabilities

**Adding New Tool:**

```go
type GrepTool struct {
    BaseTool
}

func (g *GrepTool) Name() string { return "grep" }
func (g *GrepTool) Description() string { return "Search for patterns" }
func (g *GrepTool) Parameters() map[string]interface{} { ... }
func (g *GrepTool) Execute(ctx, params) (interface{}, error) { ... }

// Register with agent
agent.tools = append(agent.tools, NewGrepTool())
```

### 4. Strategy Pattern (LLM Providers)

**Problem:** Different LLM APIs have different formats

**Solution:** Common interface, provider-specific implementations

```go
// OpenAI format
{"messages": [{"role": "user", "content": "..."}]}

// Anthropic format
{"messages": [{"role": "user", "content": [{"type": "text", "text": "..."}]}]}

// Gemini format
{"contents": [{"parts": [{"text": "..."}], "role": "user"}]}
```

All converted to/from unified `CompletionRequest/Response`.

---

## Data Flow

### Analyze Command Flow

```
1. CLI: gendocs analyze --repo-path .
   ↓
2. cmd/analyze.go: Parse flags, build config
   ↓
3. handlers.AnalyzeHandler: Load configuration
   ↓
4. Create AnalyzerAgent with 5 sub-agents
   ↓
5. Run sub-agents in parallel (worker pool)
   ↓
   ├─ StructureAnalyzer
   │   ├─ List files (tool)
   │   ├─ Read key files (tool)
   │   └─ LLM analysis → code_structure.md
   │
   ├─ DependencyAnalyzer → dependencies.md
   ├─ DataFlowAnalyzer → data_flow.md
   ├─ RequestFlowAnalyzer → request_flow.md
   └─ APIAnalyzer → api_analysis.md
   ↓
6. Collect results
   ↓
7. Write to .ai/docs/
   ↓
8. Log completion
```

### Tool Calling Flow

```
Agent:
  "I need to read main.go"
  ToolCall: {name: "read_file", args: {path: "main.go"}}
    ↓
Tool:
  - Validate path (no traversal)
  - Execute: os.ReadFile("main.go")
  - Return: {content: [...], line_count: 50}
    ↓
Agent:
  "The file contains..."`
  (continues analysis)
```

---

## Configuration System

### Multi-Source Loading

**Implementation:** Viper library

```go
type Loader struct {
    v *viper.Viper
}

func (l *Loader) LoadForAgent(repoPath, section string) (*viper.Viper, error) {
    // 1. Load ~/.gendocs.yaml
    l.loadGlobalConfig()

    // 2. Load .ai/config.yaml
    l.loadProjectConfig(repoPath)

    // 3. Merge with CLI overrides
    l.applyCLIOverrides(overrides)

    // 4. Substitute environment variables
    l.v.AutomaticEnv()

    return l.v, nil
}
```

### Validation

```go
func validateLLMConfig(cfg *LLMConfig) error {
    if cfg.APIKey == "" {
        return NewMissingEnvVarError("API_KEY", "Required for LLM provider")
    }

    validProviders := map[string]bool{
        "openai": true, "anthropic": true, "gemini": true,
    }

    if !validProviders[cfg.Provider] {
        return NewInvalidEnvVarError("PROVIDER", cfg.Provider, "Must be: openai, anthropic, or gemini")
    }

    return nil
}
```

---

## Error Handling Strategy

### Error Flow

```
Tool Error → ModelRetryError (retryable)
    ↓
BaseAgent catches → Formats for LLM
    ↓
LLM sees error → Adjusts strategy or reports
    ↓
Final result includes error context
```

### Error Context

```go
err := &AIDocGenError{
    ErrorType: "LLMClientError",
    Message:   "API request failed: 429 Too Many Requests",
    UserMessage: `
Rate limit exceeded. You can:
1. Wait a few minutes and retry
2. Reduce --max-workers flag
3. Upgrade your API tier
    `,
    Suggestion: "Try: gendocs analyze --max-workers 2",
    Context: map[string]interface{}{
        "provider": "openai",
        "model":    "gpt-4",
        "attempt":  3,
    },
}
```

### User-Facing Errors

```bash
$ gendocs analyze
Error: Missing required configuration: ANALYZER_LLM_API_KEY

ANALYZER_LLM_API_KEY is required but not set. You can fix this by:

1. Setting the environment variable:
   export ANALYZER_LLM_API_KEY="your-key"

2. Adding to .ai/config.yaml:
   analyzer:
     llm:
       api_key: "your-key"

3. Running the configuration wizard:
   gendocs config
```

---

## Concurrency Model

### Worker Pool Implementation

```go
type WorkerPool struct {
    maxWorkers int
    semaphore  chan struct{}
}

func (wp *WorkerPool) Run(ctx context.Context, tasks []Task) []Result {
    results := make([]Result, len(tasks))
    var wg sync.WaitGroup

    for i, task := range tasks {
        wg.Add(1)
        go func(index int, t Task) {
            defer wg.Done()

            // Acquire semaphore (blocks if max workers reached)
            wp.semaphore <- struct{}{}
            defer func() { <-wp.semaphore }()

            // Execute task
            results[index] = t(ctx)
        }(i, task)
    }

    wg.Wait()
    return results
}
```

### Parallelism Strategy

**Parallel:**
- Sub-agent execution (5 analyzers run concurrently)
- File operations (if safe)

**Sequential:**
- LLM API calls per agent (tool calling loop)
- File writes (prevent corruption)

**Configurable:**
```bash
# Auto-detect CPUs (default)
gendocs analyze

# Limit for rate limits
gendocs analyze --max-workers 2

# Maximize throughput
gendocs analyze --max-workers 8
```

---

## Testing Strategy

### Test Types

1. **Unit Tests** (`*_test.go`)
   - LLM clients with mock HTTP servers
   - Tools with temp directories
   - Prompt manager with mock prompts
   - Config loader with temp files

2. **Integration Tests** (`*_integration_test.go`)
   - Agent flows with mock LLM clients
   - End-to-end analyze workflow
   - Tool calling sequences

3. **Build Tag Separation:**
   ```go
   // +build integration
   ```

   Run: `go test -tags integration ./...`

### Mock Infrastructure

**MockLLMClient:**

```go
mockClient := &MockLLMClient{
    Responses: []llm.CompletionResponse{
        {ToolCalls: ...}, // Response 1
        {Content: ...},   // Response 2
    },
}

agent.llmClient = mockClient
agent.Run(ctx) // Uses mock responses
```

**Test Helpers:**

```go
// Create temp git repo
repoPath := testing.CreateTempRepo(t, map[string]string{
    "main.go": "package main...",
    "go.mod":  "module test...",
})

// Assert file operations
testing.AssertFileExists(t, "README.md")
testing.AssertFileContains(t, "README.md", "## Features")
```

### Coverage Goals

- **Unit Tests:** 80%+ coverage
- **Integration Tests:** Critical paths
- **Manual Tests:** Real LLM API calls (pre-release)

---

## Extension Points

### Adding a New LLM Provider

1. **Implement Interface:**

```go
// internal/llm/newprovider.go
type NewProviderClient struct {
    *BaseLLMClient
    apiKey string
}

func (c *NewProviderClient) GenerateCompletion(...) (CompletionResponse, error) {
    // Transform request to provider format
    // Make API call
    // Transform response to unified format
}
```

2. **Register in Factory:**

```go
// internal/llm/factory.go
case "newprovider":
    return NewNewProviderClient(cfg, retryClient), nil
```

3. **Add Tests:**

```go
// internal/llm/newprovider_test.go
func TestNewProviderClient_GenerateCompletion_Success(t *testing.T) { ... }
```

### Adding a New Tool

1. **Implement Interface:**

```go
// internal/tools/newtool.go
type NewTool struct {
    BaseTool
}

func (n *NewTool) Name() string { return "new_tool" }
func (n *NewTool) Execute(ctx, params) (interface{}, error) {
    // Implement tool logic
}
```

2. **Register with Agent:**

```go
// internal/agents/analyzer.go
tools := []tools.Tool{
    tools.NewFileReadTool(3),
    tools.NewListFilesTool(3),
    tools.NewNewTool(3), // Add here
}
```

### Adding a New Sub-Agent

1. **Create Factory Function:**

```go
// internal/agents/factory.go
func CreateNewAnalyzer(...) (*SubAgent, error) {
    return NewSubAgent(SubAgentConfig{
        Name: "NewAnalyzer",
        PromptSuffix: "new_analyzer",
    }, ...)
}
```

2. **Add Prompts:**

```yaml
# prompts/analyzer.yaml
new_analyzer_system: |
  You are a specialized analyzer for...

new_analyzer_user: |
  Analyze {{.RepoPath}} for...
```

3. **Integrate in Orchestrator:**

```go
// internal/handlers/analyze.go
newAnalyzer, _ := CreateNewAnalyzer(...)
results := pool.Run(ctx, []Task{
    structureAnalyzer.Run,
    newAnalyzer.Run, // Add here
})
```

---

## Performance Considerations

### Bottlenecks

1. **LLM API Latency**
   - Mitigation: Parallel sub-agent execution
   - Configurable: `--max-workers`

2. **File I/O**
   - Mitigation: Efficient streaming for large files
   - Tool: FileReadTool supports pagination

3. **Memory**
   - Large codebases: Tools return bounded data
   - LLM context: Agents work on subsets

### Optimization Strategies

**Implemented:**
- Worker pool for parallelism
- HTTP keep-alive connections
- Retry with exponential backoff

**Future (Phase 4):**
- Incremental analysis cache
- Dependency graph-based invalidation
- Streaming responses from LLMs

---

## Security Considerations

### Path Traversal Prevention

```go
// tools/file_read.go
func validatePath(repoPath, filePath string) error {
    absRepo, _ := filepath.Abs(repoPath)
    absFile, _ := filepath.Abs(filepath.Join(repoPath, filePath))

    if !strings.HasPrefix(absFile, absRepo) {
        return &PathTraversalError{...}
    }
    return nil
}
```

### API Key Handling

- Never logged or displayed
- Loaded from secure sources (env vars, config files with restricted permissions)
- Not passed in CLI arguments (shell history risk)

### LLM Output Validation

- Markdown validator prevents broken output
- No shell command execution from LLM outputs
- Tools have strict parameter validation

---

## Deployment

### Binary Distribution

```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o gendocs-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o gendocs-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o gendocs-windows-amd64.exe
```

### Docker (Future)

```dockerfile
FROM golang:1.22-alpine
WORKDIR /app
COPY . .
RUN go build -o gendocs
ENTRYPOINT ["./gendocs"]
```

---

## Future Architecture Evolution

### Planned Changes (Phase 4)

1. **Cache Layer:**
   ```
   Agent → Cache Check → (miss) → LLM → Cache Store
   ```

2. **GitHub Support:**
   ```
   GitProvider Interface
       ├─ GitLabProvider
       └─ GitHubProvider
   ```

3. **Plugin System:**
   - **Rejected:** Dynamic plugins (security risk)
   - **Accepted:** Static tool registration

---

## Conclusion

Gendocs architecture prioritizes:
- **Modularity:** Clear layer separation
- **Testability:** Mockable dependencies
- **Extensibility:** Well-defined interfaces
- **Security:** Input validation and sandboxing
- **Performance:** Concurrent execution where safe

The Handler-Agent pattern provides clean separation between orchestration and AI logic, making the system maintainable and testable.

---

**Document Version:** 2.0
**Maintainers:** Gendocs Team
**Last Review:** 2025-12-23
