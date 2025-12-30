# Gendocs - Suggested Commands

## Build Commands
```bash
make build                    # Compile binary to build/gendocs-{os}-{arch}
go build -o gendocs .         # Quick local build
make install                  # Install to ~/.local/bin
```

## Test Commands
```bash
make test                     # All tests with race detection (5m timeout)
make test-short               # Unit tests only (2m timeout), skips integration
make test-coverage            # Tests + coverage report to coverage/coverage.out
make test-verbose             # Verbose output

# Single test/package
go test -run TestOpenAIClient_GenerateCompletion ./internal/llm/
go test -v ./internal/agents/

# Integration tests (requires build tag)
go test -tags integration ./internal/agents/

# View coverage report
go tool cover -html=coverage/coverage.out
```

## Lint Commands
```bash
make lint                     # golangci-lint (must be installed)
go fmt ./...                  # Format all code
```

## Clean Commands
```bash
make clean                    # Remove build/ and coverage/ directories
```

## Application Commands
```bash
# Analyze a codebase
gendocs analyze --path /path/to/repo

# Generate documentation
gendocs generate readme --output README.md
gendocs generate ai_rules --output CLAUDE.md

# Export formats
gendocs generate export --input README.md --output docs.html --format html
gendocs generate export --input README.md --output docs.json --format json

# Cache management
gendocs cache-stats           # View cache statistics
gendocs cache-clear           # Clear LLM response cache

# Configuration
gendocs config                # Interactive config wizard
```

## System Utilities (Linux)
```bash
git status / git diff / git log
ls -la
find . -name "*.go"
grep -r "pattern" .
```
