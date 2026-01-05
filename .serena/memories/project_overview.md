# Gendocs - Project Overview

## Purpose
Gendocs is a CLI application that uses LLMs to analyze codebases and automatically generate documentation (README files, AI assistant rules, etc.).

## Tech Stack
- **Language**: Go 1.25.5
- **CLI Framework**: Cobra
- **Configuration**: Viper (YAML files, env vars)
- **Logging**: Zap (structured logging)
- **TUI**: Bubbletea + Lipgloss
- **Markdown**: Goldmark with Chroma syntax highlighting

## Project Structure
```
cmd/                    # CLI commands (Cobra)
  analyze.go            # Analyze codebase command
  generate.go           # Generate docs command
  config.go             # Config wizard command
  cache.go              # Cache management commands
  cronjob.go            # GitLab CI/CD automation
  root.go               # Root command setup

internal/
  agents/               # AI agents (analyzer, documenter, ai_rules_generator)
  cache/                # Analysis result caching (file hashing)
  config/               # Configuration loading (Viper)
  errors/               # Custom error types with exit codes
  export/               # HTML/JSON exporters
  handlers/             # Business logic orchestration
  llm/                  # LLM clients (OpenAI, Anthropic, Gemini)
  llmcache/             # LLM response caching
  llmtypes/             # Shared LLM types
  logging/              # Zap logger setup
  prompts/              # YAML prompt management with Go templates
  testing/              # Test helpers and fixtures
  tools/                # Agent tools (file_read, list_files)
  tui/                  # Terminal UI components
  validation/           # Markdown validation
  worker_pool/          # Parallel task execution

prompts/                # YAML prompt templates
```

## Architecture Pattern
Clean Architecture with Handler-Agent separation:
1. **CLI (cmd/)** → parses commands, calls handlers
2. **Handlers** → orchestrate I/O, config, file writing
3. **Agents** → LLM interaction, tool-calling loops
4. **Tools** → atomic capabilities (file read, list files)

## Key Interfaces
- `Agent` - `Run(ctx)` and `Name()` methods
- `LLMClient` - `GenerateCompletion()`, `SupportsTools()`, `GetProvider()`
- `Tool` - `Name()`, `Description()`, `Parameters()`, `Execute()`

## LLM Providers Supported
- OpenAI (GPT-4, etc.)
- Anthropic (Claude)
- Google Gemini (including Vertex AI)
- Ollama (Local)
- LM Studio (Local)

## Configuration Precedence
1. CLI flags
2. `.ai/config.yaml` (project-level)
3. `~/.gendocs.yaml` (user-level)
4. Environment variables
5. Defaults
