# Gendocs - Automated Documentation Generation

Gendocs is a CLI application that leverages Large Language Models (LLMs) to analyze codebases and automatically generate documentation. It streamlines the process of creating and maintaining up-to-date documentation for your projects.

## Features

*   **Codebase Analysis:** Analyzes code structure, dependencies, data flow, and API definitions.
*   **Automated Documentation:** Generates README files and AI assistant rules.
*   **LLM Integration:** Supports multiple LLM providers (Anthropic, Gemini, OpenAI).
*   **Customizable Configuration:** Configure LLM providers, agents, and output formats.
*   **GitLab Integration:** Supports automated documentation updates via cronjobs.
*   **HTML Export:** Converts Markdown documentation to HTML with syntax highlighting.

## Installation

1.  **Clone the repository:**
    ```bash
    git clone <repository_url>
    cd <repository_directory>
    ```
2.  **Install dependencies:**

    ```bash
    go mod download
    go mod vendor
    ```

## Quick Start

1.  **Analyze your codebase:**

    ```bash
    go run cmd/gendocs/main.go analyze --path <path_to_codebase>
    ```

    This command analyzes the codebase at the specified path and saves the analysis results in the `.ai/docs/` directory.
2.  **Generate a README file:**

    ```bash
    go run cmd/gendocs/main.go generate readme --output README.md
    ```

    This command generates a `README.md` file based on the analysis results.
3.  **(Optional) Generate AI rules:**

    ```bash
    go run cmd/gendocs/main.go generate ai_rules --output ai_rules.yaml
    ```

    This command generates an `ai_rules.yaml` file.
4. **(Optional) Export to HTML:**

    ```bash
    go run cmd/gendocs/main.go export html --input README.md --output docs/index.html
    ```

    This command converts the README.md file to HTML format with syntax highlighting.

## Architecture

Gendocs follows a clean architecture pattern, separating concerns into distinct layers:

*   **Command Layer (`cmd/`)**: Handles CLI command parsing and execution.
*   **Agent System (`internal/agents/`)**: Coordinates analysis and documentation generation using specialized agents.
*   **LLM Integration (`internal/llm/`)**: Provides a unified interface for interacting with different LLM providers.
*   **Configuration (`internal/config/`)**: Loads and manages application configuration.
*   **Handlers (`internal/handlers/`)**: Orchestrates the overall workflow for each command.

The CLI commands delegate to handlers, which in turn create and run agents. Agents use LLM clients and tools to perform their tasks. All components share configuration and logging.

## Development

### Running Tests

```bash
go test ./...
```

### Linting

```bash
# Install golangci-lint (if not already installed)
# example:  brew install golangci/tap/golangci-lint
golangci-lint run
```

### Building

```bash
go build -o gendocs cmd/gendocs/main.go
```

This command builds an executable file named `gendocs` in the current directory.

## Configuration

Gendocs uses environment variables and configuration files to manage its settings.

*   **Configuration File:**  The application uses `viper` to load configurations. Configuration files can be in YAML format.
*   **Environment Variables:** Environment variables can override settings defined in the configuration file.

### Cache Configuration

Gendocs supports caching LLM API responses to reduce API costs and improve performance on repeated analyses. Cache settings are configured under the `llm.cache` section in your configuration file (`.ai/config.yaml`):

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable or disable LLM response caching. When enabled, identical LLM requests will return cached responses instead of making new API calls. |
| `max_size` | integer | `1000` | Maximum number of entries to store in the in-memory cache. When this limit is reached, least-recently-used entries are automatically evicted. |
| `ttl` | integer | `7` | Time-to-live for cache entries in days. After this period, cached responses are considered stale and will not be used. |
| `cache_path` | string | `.ai/llm_cache.json` | Path to the persistent disk cache file. This file survives program restarts and allows cache entries to be reused across multiple runs. |

#### Example Configuration

```yaml
llm:
  provider: openai
  model: gpt-4
  api_key: your-api-key
  cache:
    enabled: true
    max_size: 1000
    ttl: 7
    cache_path: .ai/llm_cache.json
```

#### Cache Management Commands

Gendocs provides CLI commands to manage the LLM response cache:

*   **View cache statistics:**
    ```bash
    gendocs cache-stats
    ```
    Displays cache hit/miss rates, number of entries, storage size, and other metrics.

*   **Clear the cache:**
    ```bash
    gendocs cache-clear
    ```
    Removes all cached responses, forcing the next run to make fresh API calls.

*   **Show cache stats after analysis:**
    ```bash
    gendocs analyze --show-cache-stats
    ```
    Displays cache statistics immediately after completing an analysis.

#### Benefits of Caching

*   **Cost Savings:** Avoid redundant API calls for identical requests, significantly reducing LLM API costs.
*   **Faster Iteration:** Repeated analyses with unchanged code complete much faster by serving responses from cache.
*   **Offline Capability:** Cached responses allow some operations to complete even without API access.

For additional configuration options, refer to the `internal/config/` package. Example environment variables used include those needed to authenticate with LLM providers (e.g., OpenAI API key).

## Contributing

Contributions are welcome! Please follow these guidelines:

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix.
3.  Write tests for your changes.
4.  Submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE).
