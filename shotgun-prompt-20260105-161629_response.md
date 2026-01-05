```json
{
  "code_quality": [
    {
      "id": "cq-001",
      "title": "Refactor monolithic JSON exporter",
      "description": "The file `internal/export/json.go` exceeds 1,100 lines and handles data models, AST traversal, node processing, and JSON marshalling.",
      "rationale": "The file mixes data definitions with complex logic. Modifying the export format requires navigating AST traversal logic, and vice versa. It violates the Single Responsibility Principle.",
      "category": "large_files",
      "severity": "major",
      "affected_files": [
        "internal/export/json.go"
      ],
      "current_state": "Single file containing `JSONDocument` struct definitions, `JSONExporter` logic, and 15+ `process*` and `extract*` methods.",
      "proposed_change": "Split into 3 files: `models.go` (structs), `extractor.go` (AST traversal/processing), and `json.go` (entry point/marshalling).",
      "code_example": "// internal/export/models.go\ntype JSONDocument struct { ... }\n\n// internal/export/extractor.go\nfunc (e *JSONExporter) traverseAST(doc ast.Node) ...",
      "best_practice": "Separation of Concerns",
      "estimated_effort": "medium",
      "breaking_change": false,
      "prerequisites": []
    },
    {
      "id": "cq-002",
      "title": "Reduce duplication in LLM client tests",
      "description": "The files `internal/llm/openai_test.go`, `anthropic_test.go`, and `gemini_test.go` contain massive code duplication for setting up mock HTTP servers and validating responses.",
      "rationale": "Maintaining tests is difficult; a change in testing strategy requires updating three separate files with near-identical logic. The total lines of test code are bloated.",
      "category": "duplication",
      "severity": "major",
      "affected_files": [
        "internal/llm/openai_test.go",
        "internal/llm/anthropic_test.go",
        "internal/llm/gemini_test.go"
      ],
      "current_state": "Each test function manually instantiates `httptest.NewServer`, sets headers, and defines raw JSON strings for SSE/REST responses.",
      "proposed_change": "Create a `test_utils.go` in the `llm` package with helpers like `setupMockServer(handler http.HandlerFunc)` and shared assertion helpers.",
      "code_example": "// internal/llm/test_utils.go\nfunc newMockStreamServer(t *testing.T, events []string) *httptest.Server { ... }\n\n// internal/llm/openai_test.go\nserver := newMockStreamServer(t, openaiEvents)",
      "best_practice": "Don't Repeat Yourself (DRY) in Tests",
      "estimated_effort": "medium",
      "breaking_change": false,
      "prerequisites": []
    },
    {
      "id": "cq-003",
      "title": "Simplify complex File Scanning logic",
      "description": "The function `ScanFiles` in `internal/cache/cache.go` (lines 336-486) is overly complex, handling directory walking, ignore patterns, binary checking, and cache logic simultaneously.",
      "rationale": "The function has high cyclomatic complexity. Adding new filtering logic or changing the hashing strategy is risky due to the intermingled concerns.",
      "category": "complexity",
      "severity": "major",
      "affected_files": [
        "internal/cache/cache.go"
      ],
      "current_state": "A 150-line function that defines internal structs, walks directories, checks ignores, checks binary, and orchestrates parallel hashing.",
      "proposed_change": "Extract logic into helper functions: `collectFiles` (walking + filtering), `filterCachedFiles`, and keep `parallelHashFiles` separate.",
      "code_example": "files := collectFiles(repoPath, ignorePatterns)\ntoHash, cached := filterCached(files, cache)\nhashes := parallelHashFiles(toHash, workers)",
      "best_practice": "Function Composition",
      "estimated_effort": "small",
      "breaking_change": false,
      "prerequisites": []
    },
    {
      "id": "cq-004",
      "title": "Remove magic strings in TUI config mapping",
      "description": "The TUI dashboard sections (e.g., `internal/tui/dashboard/sections/llm.go`) use manual string keys to map values to/from the configuration struct.",
      "rationale": "Using string literals like `\"provider\"`, `\"ai_rules_api_key\"` is error-prone. A typo in a key string will result in silent failure (value not saved) or a runtime panic during type assertion.",
      "category": "type_safety",
      "severity": "minor",
      "affected_files": [
        "internal/tui/dashboard/sections/llm.go",
        "internal/tui/dashboard/sections/analysis.go",
        "internal/tui/dashboard/sections/cache.go"
      ],
      "current_state": "Manual mapping: `if v, ok := values[\"documenter_provider\"].(string); ok ...`",
      "proposed_change": "Define constants for configuration keys or use the `mapstructure` library to decode map values directly into partial config structs.",
      "code_example": "const KeyProvider = \"provider\"\n// OR\nvar input LLMSectionInput\nmapstructure.Decode(values, &input)",
      "best_practice": "Type Safety / Constant Definitions",
      "estimated_effort": "medium",
      "breaking_change": false,
      "prerequisites": []
    },
    {
      "id": "cq-005",
      "title": "Consolidate LLM Client Boilerplate",
      "description": "The `GenerateCompletion` methods in `anthropic.go`, `openai.go`, and `gemini.go` share significant boilerplate for HTTP request creation and error handling.",
      "rationale": "While payloads differ, the mechanics of context handling, header setting, and basic error checking are repetitive. `BaseLLMClient` exists but is underutilized.",
      "category": "duplication",
      "severity": "minor",
      "affected_files": [
        "internal/llm/anthropic.go",
        "internal/llm/openai.go",
        "internal/llm/gemini.go"
      ],
      "current_state": "Each client implements full HTTP request lifecycle including marshaling, request creation, header setting, and response checking.",
      "proposed_change": "Move common HTTP execution logic to `BaseLLMClient` or a helper `doRequest(ctx, method, url, payload, headers)`. Clients only handle payload conversion.",
      "code_example": "// internal/llm/client.go\nfunc (b *BaseLLMClient) postJSON(ctx context.Context, url string, headers map[string]string, body interface{}) (*http.Response, error) { ... }",
      "best_practice": "Abstraction of Common Logic",
      "estimated_effort": "small",
      "breaking_change": false,
      "prerequisites": []
    },
    {
      "id": "cq-006",
      "title": "Improve Config Loader Type Safety",
      "description": "The `LoadAnalyzerConfig` function in `internal/config/loader.go` manually extracts values from a generic map using helper functions (`getString`, `getInt`).",
      "rationale": "This manual extraction is tedious and brittle. It re-implements functionality that Viper already provides via struct unmarshalling.",
      "category": "code_smells",
      "severity": "minor",
      "affected_files": [
        "internal/config/loader.go"
      ],
      "current_state": "Manual extraction: `cfg.LLM.Provider = getString(configMap, \"llm.provider\", ...)`",
      "proposed_change": "Use Viper's `Unmarshal` capability with structure tags to decode directly into the `AnalyzerConfig` struct, applying overrides via Viper's API before unmarshalling.",
      "code_example": "v.Set(\"analyzer.llm.provider\", overrideValue)\nvar cfg AnalyzerConfig\nv.UnmarshalKey(\"analyzer\", &cfg)",
      "best_practice": "Leverage Library Capabilities",
      "estimated_effort": "small",
      "breaking_change": false,
      "prerequisites": []
    }
  ],
  "summary": {
    "files_analyzed": 59,
    "issues_by_severity": {
      "critical": 0,
      "major": 3,
      "minor": 3,
      "suggestion": 0
    },
    "issues_by_category": {
      "large_files": 1,
      "duplication": 2,
      "complexity": 1,
      "type_safety": 1,
      "code_smells": 1
    }
  }
}
```