# Gendocs Go Implementation

This is the Go port of the AI Documentation Generator. The Go version aims to provide better performance, easier distribution, and simplified deployment while maintaining complete feature parity with the Python version.

## Current Status

**Phase: Foundation Complete** 🏗️

The core infrastructure has been implemented and the CLI compiles successfully. This is a work in progress following the detailed implementation plan in `../PLAN.md`.

### What's Implemented

✅ **Phase 1: Foundation**
- Project structure with Go modules
- Error handling system (14 exception types, rich error context)
- Structured logging (zap with file + console output)
- Configuration system (multi-source: CLI > YAML > env > defaults)

✅ **Phase 2: LLM Integration**
- HTTP retry client with exponential backoff
- LLM provider interface
- OpenAI client implementation (including OpenAI-compatible APIs)

✅ **Phase 3: Tools & Concurrency**
- FileReadTool (with pagination support)
- ListFilesTool (recursive directory listing)
- Worker pool (semaphore-based concurrency control)

✅ **Phase 5: CLI**
- Cobra CLI framework
- `gendocs analyze` command with all flags
- Help system

### What's Still Needed

🔄 **Phase 4: Agents**
- Prompt system (Jinja2 → Go template conversion)
- Base agent with tool calling loop
- 5 sub-agents (Structure, Dependency, DataFlow, RequestFlow, API)
- DocumenterAgent (README generation)
- AIRulesGeneratorAgent (CLAUDE.md, AGENTS.md, .cursor/rules/)

🔄 **Phase 5: Handlers**
- AnalyzeHandler
- ReadmeHandler
- AIRulesHandler
- CronjobHandler

🔄 **Phase 6: Remaining LLM Providers**
- Anthropic Claude client
- Google Gemini client (standard API)
- Google Gemini via Vertex AI

🔄 **Phase 6: GitLab Integration**
- GitLab API client
- Project filtering logic
- Repository cloning
- Branch creation, commit, push
- Merge request creation

🔄 **Phase 7: TUI Config Wizard**
- Bubble Tea configuration UI
- Provider selection
- API key input (masked)
- Save to ~/.gendocs.yaml

## Building

```bash
cd gendocs
go build -o gendocs .
```

## Running

```bash
# Show help
./gendocs --help

# Show analyze command help
./gendocs analyze --help

# Analyze a codebase (not yet fully functional)
./gendocs analyze --repo-path ../
```

## Configuration

The Go version supports the same configuration sources as the Python version:

1. **CLI arguments** (highest priority)
2. **`.ai/config.yaml`** (project-specific)
3. **`~/.gendocs.yaml`** (global user config, from TUI)
4. **Environment variables**
5. **Defaults** (lowest priority)

### Environment Variables

```bash
# Analyzer configuration
export ANALYZER_LLM_PROVIDER="openai"  # openai, anthropic, gemini
export ANALYZER_LLM_MODEL="gpt-4o"
export ANALYZER_LLM_API_KEY="sk-..."
export ANALYZER_LLM_BASE_URL="https://api.openai.com/v1"  # optional
export ANALYZER_AGENT_RETRIES=2
export ANALYZER_LLM_TIMEOUT=180
export ANALYZER_LLM_MAX_TOKENS=8192
export ANALYZER_LLM_TEMPERATURE=0.0
export ANALYZER_MAX_WORKERS=0  # 0 = auto-detect CPU count
```

See `../PLAN.md` for the complete list of 40+ environment variables.

## Project Structure

```
gendocs/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command
│   └── analyze.go         # Analyze subcommand
├── internal/
│   ├── agents/            # AI agents (not yet implemented)
│   ├── config/            # Configuration loading
│   ├── errors/            # Error handling (14 exception types)
│   ├── logging/           # Structured logging (zap)
│   ├── llm/               # LLM provider abstraction
│   │   ├── client.go      # LLM client interface
│   │   ├── retry_client.go # HTTP with retry logic
│   │   └── openai.go      # OpenAI implementation
│   ├── tools/             # Agent tools
│   │   ├── base.go        # Tool base with retry
│   │   ├── file_read.go   # File reading tool
│   │   └── list_files.go  # Directory listing tool
│   ├── worker_pool/       # Concurrent task execution
│   │   └── pool.go        # Semaphore-based pool
│   └── tui/               # TUI config wizard (not yet implemented)
├── prompts/               # YAML prompt templates (not yet added)
├── main.go
├── go.mod
└── README.md
```

## Architecture

The Go implementation follows the same **Handler-Agent Architecture** as the Python version:

```
CLI Layer (Cobra)
    ↓
Handler Layer (orchestration)
    ↓
Agent Layer (AI logic)
    ↓
Tools (file system access)
    ↓
Infrastructure (logging, retry, worker pool)
```

## Implementation Plan

See `../PLAN.md` for the complete implementation roadmap:

1. **Phase 1-3**: ✅ Foundation (Complete)
2. **Phase 4**: Agents (In Progress - foundation laid)
3. **Phase 5**: Handlers (In Progress - CLI working)
4. **Phase 6**: GitLab Integration (Pending)
5. **Phase 7**: TUI Config (Pending)
6. **Phase 8**: Testing & Validation (Pending)

**Estimated Timeline**: 13 weeks total (3.25 months)

**Current Progress**: ~40% (Phases 1-3 complete, Phase 5 started)

## Development

### Prerequisites

- Go 1.22 or later
- Access to LLM provider API (OpenAI recommended during development)

### Running Tests

```bash
go test ./...
```

### Linting

```bash
go fmt ./...
go vet ./...
```

## License

Same as the parent project.

## Contributing

This is a work in progress. Refer to `../PLAN.md` for guidance on contributing to specific phases.
