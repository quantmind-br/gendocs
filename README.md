# Gendocs

Gendocs is a modular, AI-powered CLI application built in Go that automates the analysis of codebases and the generation of comprehensive documentation. By leveraging a multi-agent orchestration model and advanced Large Language Models (LLMs), Gendocs synthesizes deep insights into project structure, dependencies, data flows, and API contracts.

## Features

- **Multi-Agent Orchestration**: A central analyzer coordinates specialized sub-agents to perform focused analysis on structure, dependencies, data flow, request flow, and API definitions.
- **Incremental Analysis**: Uses a two-tier caching system (file-based hashes and LLM response caching) to skip unchanged files, reducing API costs and execution time.
- **Documentation Drift Detection**: Detect when your code has diverged from the last analysis and get recommendations for keeping documentation fresh.
- **Support for Multiple LLM Providers**: Built-in support for Anthropic, OpenAI, and Google Gemini.
- **Automated Documentation**: Generates high-quality `README.md` files, AI assistant rules (`CLAUDE.md`, `.cursor/rules`), and technical documentation.
- **Interactive TUI Dashboard**: A component-based Terminal User Interface for managing configurations, running analysis, and tracking progress in real-time with cancellation support.
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
   You can run analysis directly from the TUI or via CLI:
   
   **Option A - TUI (Recommended)**:
   ```bash
   gendocs config
   ```
   In the TUI:
   - Navigate to "Analysis Settings" section using ↑/↓
   - Press Tab to enter the content area
   - Configure exclusions and worker settings as needed
   - Tab to the "Run Analysis" button and press Enter
   - Monitor real-time progress with task status updates
   - Press Esc to cancel if needed, or Enter to close after completion

   **Option B - CLI**:
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

5. **Check for Documentation Drift**:
   Detect if your codebase has changed since the last analysis.
   ```bash
   gendocs check
   ```
   Use `--exit-code` flag in CI/CD pipelines to fail builds when documentation is stale.

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

### Using Local LLMs (Ollama, LM Studio)

Gendocs supports local LLM providers for users who prefer to run models locally:

#### Ollama
1. Install [Ollama](https://ollama.ai/) and pull a model:
   ```bash
   ollama pull llama3
   ```
2. In the TUI (`gendocs config`), select "Ollama (Local)" as the provider
3. The Base URL will auto-populate to `http://localhost:11434/v1`
4. Enter your model name (e.g., `llama3`, `codellama`, `mistral`)
5. No API key is required

#### LM Studio
1. Install [LM Studio](https://lmstudio.ai/) and download a model
2. Start the local server in LM Studio
3. In the TUI, select "LM Studio (Local)" as the provider
4. The Base URL will auto-populate to `http://localhost:1234/v1`
5. Enter your model name as shown in LM Studio
6. No API key is required

**Note**: Model availability varies based on your local installation. Ensure your chosen model is downloaded and available before running analysis.

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