# Extract duplicated LLM client HTTP request handling

## Overview

The LLM client implementations (openai.go, anthropic.go, gemini.go) contain nearly identical HTTP request handling logic in their GenerateCompletion methods. Each implements the same pattern: marshal JSON, create HTTP request, execute with retry, read response, check status code, parse response, check for API errors.

## Rationale

Code duplication leads to bugs when fixes are applied inconsistently. If retry logic or error handling needs improvement, it must be updated in 3+ places. This increases maintenance burden and risk of inconsistencies.

---
*This spec was created from ideation and is pending detailed specification.*
