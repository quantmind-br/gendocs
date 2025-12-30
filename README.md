# Gendocs

Gendocs is a modular, AI-powered CLI application built in Go that automates the analysis of codebases and the generation of comprehensive documentation. By leveraging a multi-agent orchestration model and advanced Large Language Models (LLMs), Gendocs synthesizes deep insights into project structure, dependencies, data flows, and API contracts.

## Features

- **Multi-Agent Orchestration**: A central analyzer coordinates specialized sub-agents to perform focused analysis on structure, dependencies, data flow, request flow, and API definitions.
- **Incremental Analysis**: Uses a two-tier caching system (file-based hashes and LLM response caching) to skip unchanged files, reducing API costs and execution time.
- **Support for Multiple LLM Providers**: Built-in support for Anthropic, OpenAI, and Google Gemini.
- **Automated Documentation**: Generates high-quality `README.md` files, AI assistant rules (`CLAUDE.md`, `.cursor/rules`), and technical documentation.
- **Interactive TUI Dashboard**: A component-based Terminal User Interface for managing configurations and tracking analysis progress in real-time.
- **Enterprise Integration**: Features a GitLab cronjob mode to batch-process entire groups, automatically creating Merge Requests with updated documentation.
- **Flexible Export Formats**: Export generated documentation to Markdown, standalone HTML, or structured JSON.

## Installation

### Prerequisites
- Go 1.21 or higher
- An API key from a supported LLM provider (Anthropic, OpenAI, or Google Gemini)

### Build from Source
```bash
git clone https://github.com/your-repo/gendocs.git
cd gendocs
go build -o gendocs main.go
```

## Quick Start

1. **Initialize Configuration**:
   Launch the interactive TUI to set up your API keys and project preferences.
   ```bash
   gendocs config
   ```
   Alternatively, export your API key:
   ```bash
   export ANTHROPIC_API_KEY=your_key_here
   ```

2. **Analyze Your Project**:
   Run a deep analysis of the current directory. This scans the codebase and generates intermediate analysis files in `.ai/docs/`.
   ```bash
   gendocs analyze
   ```

3. **Generate Documentation**:
   Synthesize the analysis results into a user-friendly README.
   ```bash
   gendocs generate readme
   ```

4. **Generate AI Rules**:
   Create configuration files for AI coding assistants.
   ```bash
   gendocs generate ai-rules
   ```

## Architecture

Gendocs follows a layered architecture designed for extensibility and performance:

- **CLI Layer (`cmd/`)**: Built with Cobra, managing command routing and user input.
- **Handler Layer (`internal/handlers/`)**: Orchestrates the lifecycle of operations between the CLI and internal agents.
- **Agent Layer (`internal/agents/`)**: The core logic layer. It uses an **Orchestrator Pattern** where an `AnalyzerAgent` manages a worker pool of specialized sub-agents.
- **LLM Layer (`internal/llm/`)**: A unified interface for different AI providers, featuring a **Decorator Pattern** to add retry logic and caching without modifying core LLM logic.
- **Tool Layer (`internal/tools/`)**: Defines safe "capabilities" (like file system operations) that agents can invoke during analysis.
- **Cache System**: Manages incremental analysis via file hashing (`.ai/analysis_cache.json`) and LLM response persistence (`.ai/llm_cache.json`).

## Configuration

Gendocs uses a hierarchical configuration system (loaded via Viper) with the following priority:
1.  **CLI Flags** (e.g., `--repo-path`, `--force`)
2.  **Environment Variables** (e.g., `ANTHROPIC_API_KEY`, `GITLAB_OAUTH_TOKEN`)
3.  **Project Config**: `.ai/config.yaml` within the target repository.
4.  **Global Config**: `~/.gendocs.yaml` for system-wide defaults.

### Key Environment Variables
| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | API key for Anthropic Claude models. |
| `OPENAI_API_KEY` | API key for OpenAI GPT models. |
| `GEMINI_API_KEY` | API key for Google Gemini/Vertex AI. |
| `GITLAB_OAUTH_TOKEN` | Required for automated GitLab group analysis. |

## Development

### Running Tests
Execute the test suite using the Go toolchain:
```bash
go test ./...
```

### Linting
Ensure code quality by running your preferred Go linter (e.g., `golangci-lint`):
```bash
golangci-lint run
```

### Project Structure
- `internal/agents/`: Logic for the various analysis sub-agents.
- `internal/llm/`: LLM provider implementations and client decorators.
- `internal/tui/`: Bubble Tea components for the terminal dashboard.
- `internal/cache/`: Incremental analysis and hashing logic.

## Deployment

For automated maintenance of large organizations, Gendocs can be run as a cronjob:
```bash
gendocs cronjob analyze --group-project-id <your-gitlab-group-id>
```
This command will:
1. Fetch projects from the specified GitLab group.
2. Filter for active projects (default within 14 days).
3. Clone and analyze each repository.
4. Commit documentation changes and open Merge Requests.

## License

Refer to the `LICENSE` file in the project root for licensing information.