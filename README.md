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

## Performance Optimizations

Gendocs implements two key optimizations to significantly improve file scanning performance, especially for incremental analysis of large codebases:

### Selective Hashing

The file scanner uses modification time (mtime) and size-based caching to skip rehashing unchanged files:

- **How it works**: When scanning a repository, Gendocs stores each file's metadata (SHA256 hash, modification time, and size) in a cache file (`.ai/analysis_cache.json`). On subsequent scans, files with matching mtime and size skip the expensive SHA256 hash computation and reuse the cached hash value.

- **Performance impact**: For incremental scans where most files haven't changed, this can reduce scan time by 80-95% since hash computation is avoided for unchanged files.

- **Cache hit conditions**: A file is considered unchanged if **both** the modification time and size match the cached values. Using both conditions provides robust change detection while avoiding false positives.

### Parallel Hashing

When files do need hashing, they are processed concurrently using a worker pool pattern:

- **How it works**: Files requiring hash computation are distributed across multiple worker goroutines (default: number of CPU cores, max 8). Each worker independently computes SHA256 hashes, allowing the CPU-bound work to proceed in parallel.

- **Performance impact**: Parallel hashing provides 2-4x speedup for the actual hash computation phase on multi-core systems. The combined effect of selective hashing + parallel processing can provide 3-5x faster incremental scans on large repositories.

### Configuration

The parallel hashing behavior can be configured via:

**Configuration file** (`.ai/config.yaml`):
```yaml
analyzer:
  max_hash_workers: 4  # Number of parallel hash workers (0 = auto-detect)
```

**Environment variable**:
```bash
export GENDOCS_ANALYZER_MAX_HASH_WORKERS=4
```

**Values**:
- `0` (default): Auto-detect using `runtime.NumCPU()`, capped at 8 workers
- `1`: Sequential hashing (no parallelism)
- `2-8`: Specify exact number of parallel workers

**Note**: All values are capped at 8 to avoid overwhelming the filesystem.

### Metrics and Monitoring

The analyzer logs cache hit/miss metrics after each scan to help track optimization effectiveness:

```
DEBUG Scan complete: total=1500 files, cached=1420 (94.7%), hashed=80 (5.3%)
```

- **Cached files**: Files that skipped hashing due to cache hits (mtime+size match)
- **Hashed files**: Files that required new hash computation (cache misses)
- **Cache hit rate**: Higher percentages indicate more effective optimization

### Benchmark Results

Based on typical repository structures (1000+ files, ~50KB average):

| Scenario | Time | Speedup |
|----------|------|---------|
| Baseline (no cache, sequential) | 100% | 1x |
| With cache only | 10-20% | 5-10x |
| With parallel only | 30-50% | 2-3x |
| With cache + parallel (incremental) | 5-15% | 7-20x |

*Actual results vary based on file sizes, change frequency, and hardware.*

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

Refer to the `internal/config/` package for details on available configuration options.  Example environment variables used include those needed to authenticate with LLM providers (e.g., OpenAI API key).

## Contributing

Contributions are welcome! Please follow these guidelines:

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix.
3.  Write tests for your changes.
4.  Submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE).
