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

## Testing

There is no explicit information about tests provided. Look for files named `*_test.go` to locate tests. Assume standard Go testing practices are used (e.g., `go test`).

## Troubleshooting

*   **Configuration Errors:** Ensure that the configuration file is correctly formatted and that all required environment variables are set.
*   **LLM API Errors:** Check API keys and rate limits for the configured LLM provider.
*   **File Access Errors:** Verify that the application has the necessary permissions to read and write files in the specified directories.
*   **Dependency Issues:** Run `go mod tidy` and `go mod vendor` to resolve dependency conflicts.
