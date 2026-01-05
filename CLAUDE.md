# CLAUDE.md

## Project Overview

Gendocs is a CLI application written in Go that uses Large Language Models (LLMs) to analyze codebases and generate documentation. It provides commands to analyze code, generate README files, create AI assistant rules, and export documentation to HTML.

## Common Commands

*   **Run the application:** `go run cmd/root.go` (This might require specifying a subcommand, e.g., `go run cmd/root.go analyze`)
*   **Analyze the codebase:** `go run cmd/root.go analyze` (This command likely initiates the analysis process, storing results in `.ai/docs/`).
*   **Generate a README:** `go run cmd/root.go generate readme` (This command creates a `README.md` file based on the analysis).
*   **Generate AI rules:** `go run cmd/root.go generate ai_rules` (This command creates configuration files for AI assistants).
*   **Export to HTML:** `go run cmd/root.go generate export` (This command converts markdown documentation to HTML).
*   **Run a cronjob (GitLab integration):** `go run cmd/root.go cronjob`
*   **Load configuration:** `go run cmd/root.go config`

## Architecture

Gendocs follows a clean architecture with these key patterns:

*   **CLI Layer:** Uses `github.com/spf13/cobra` for command-line interface.
*   **Handlers:** Orchestrate the core logic (analysis, generation, export). Located in `internal/handlers/`.
*   **Agents:** Perform specialized tasks using LLMs. Located in `internal/agents/`.
*   **LLM Clients:** Integrate with different LLM providers (Anthropic, Gemini, OpenAI). Located in `internal/llm/`. Uses a Factory pattern (`internal/llm/factory.go`).
*   **Configuration:** Loads settings from files and environment variables using `github.com/spf13/viper` in `internal/config/`.
*   **Factory Pattern:** Used for creating LLM clients and agents, allowing for easy swapping of implementations.
*   **Strategy Pattern:**  Different LLM providers implement the same `LLMClient` interface.
*   **Worker Pool:** Employs parallel agent execution.

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
```

## Code Conventions

*   **Configuration:** Defined in `internal/config/models.go` and loaded using `internal/config/loader.go`.
*   **Logging:** Uses `go.uber.org/zap` for structured logging.
*   **Error Handling:** While not explicitly detailed, assume standard Go error handling patterns are used.

## Key Files

*   **`cmd/root.go`:** The main entry point for the CLI application. Defines the root command and global flags.
*   **`cmd/analyze.go`:** Defines the `analyze` command, triggering the codebase analysis.
*   **`cmd/generate.go`:** Defines the `generate` command with subcommands for README, AI rules, and export.
*   **`internal/agents/analyzer.go`:** The main analyzer agent that coordinates sub-agents.
*   **`internal/llm/client.go`:** Defines the `LLMClient` interface for interacting with LLMs.
*   **`internal/llm/anthropic.go`, `internal/llm/gemini.go`, `internal/llm/openai.go`:** Implementations of the `LLMClient` interface for different LLM providers.
*   **`internal/handlers/analyze.go`:** Orchestrates the codebase analysis process.
*   **`internal/handlers/readme.go`:** Handles the generation of the README file.
*   **`go.mod`:**  Lists project dependencies.

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
- Run integration tests: `go test -tags integration ./internal/tui/dashboard/...`

## Testing

There is no explicit information about tests provided. Look for files named `*_test.go` to locate tests. Assume standard Go testing practices are used (e.g., `go test`).

## Troubleshooting

*   **Configuration Errors:** Ensure that the configuration file is correctly formatted and that all required environment variables are set.
*   **LLM API Errors:** Check API keys and rate limits for the configured LLM provider.
*   **File Access Errors:** Verify that the application has the necessary permissions to read and write files in the specified directories.
*   **Dependency Issues:** Run `go mod tidy` and `go mod vendor` to resolve dependency conflicts.
