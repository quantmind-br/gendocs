# Gendocs - Task Completion Checklist

## Before Marking a Task Complete

### 1. Code Quality
- [ ] Code follows project conventions (see `code_style_conventions.md`)
- [ ] Imports are properly grouped (stdlib, external, internal)
- [ ] Errors are wrapped with context using `%w`
- [ ] Context propagation is correct (`ctx` as first param)

### 2. Testing
- [ ] New code has tests (80%+ coverage for new code)
- [ ] Critical code (LLM, tools) has 90%+ coverage
- [ ] Tests follow naming: `Test<Type>_<Method>_<Scenario>`
- [ ] Table-driven tests used where appropriate
- [ ] Run tests: `make test-short` (fast) or `make test` (full)

### 3. Linting
- [ ] Run `go fmt ./...` to format code
- [ ] Run `make lint` (if golangci-lint is installed)
- [ ] No type suppressions (`as any`, `@ts-ignore` equivalents)

### 4. Documentation
- [ ] Public functions have doc comments
- [ ] Complex logic has inline comments
- [ ] README updated if adding new features/commands

### 5. Build Verification
- [ ] `make build` succeeds
- [ ] No new compiler warnings

## Quick Verification Commands
```bash
# Format code
go fmt ./...

# Run short tests
make test-short

# Run full tests with race detection
make test

# Check linting
make lint

# Verify build
make build
```

## Coverage Report
```bash
make test-coverage
go tool cover -html=coverage/coverage.out
```

## Integration Tests
```bash
# Only run when needed (slower)
go test -tags integration ./internal/agents/
```
