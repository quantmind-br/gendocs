# Implement response caching for LLM API calls with identical inputs

## Overview

Add an in-memory and disk-based cache layer for LLM API responses. Many analysis calls may be repeated across runs (e.g., re-analyzing unchanged files), but currently every request hits the API.

## Rationale

The documenter agent and AI rules agent both read all 5 analysis files and make LLM calls with them as context. If the analysis files haven't changed, the LLM responses will be nearly identical. By hashing the prompt + system message and caching responses, we can avoid redundant API calls. This would save significant API costs and latency for incremental runs where only some agents need to re-run.

---
*This spec was created from ideation and is pending detailed specification.*
