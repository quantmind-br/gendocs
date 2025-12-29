# Duplicated HTTP Request Pattern Analysis

## Overview
This document analyzes the duplicated HTTP request handling pattern across the three LLM client implementations: OpenAI, Anthropic, and Gemini.

## The Duplicated Pattern

All three clients implement the same 8-step pattern in their `GenerateCompletion` methods:

### Step 1: Marshal Request to JSON
```go
jsonData, err := json.Marshal(providerRequest)
if err != nil {
    return CompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
}
```
- **OpenAI**: Line 117
- **Anthropic**: Line 115
- **Gemini**: Line 115

### Step 2: Create HTTP Request
```go
httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
if err != nil {
    return CompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
}
```
- **OpenAI**: Line 124
- **Anthropic**: Line 122
- **Gemini**: Line 127

### Step 3: Set HTTP Headers
```go
httpReq.Header.Set("Content-Type", "application/json")
// Provider-specific headers set here
```
- **OpenAI** (Lines 129-130): Content-Type + Authorization (Bearer token)
- **Anthropic** (Lines 127-129): Content-Type + x-api-key + anthropic-version
- **Gemini** (Line 132): Content-Type only (API key in URL query param)

### Step 4: Execute Request with Retry
```go
resp, err := c.retryClient.Do(httpReq)
if err != nil {
    return CompletionResponse{}, fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()
```
- **OpenAI**: Lines 133-137
- **Anthropic**: Lines 132-136
- **Gemini**: Lines 135-139

### Step 5: Read Response Body
```go
body, err := io.ReadAll(resp.Body)
if err != nil {
    return CompletionResponse{}, fmt.Errorf("failed to read response: %w", err)
}
```
- **OpenAI**: Lines 140-143
- **Anthropic**: Lines 139-142
- **Gemini**: Lines 142-145

### Step 6: Check HTTP Status Code
```go
if resp.StatusCode != http.StatusOK {
    return CompletionResponse{}, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
}
```
- **OpenAI**: Lines 146-148
- **Anthropic**: Lines 145-147
- **Gemini**: Lines 148-150

### Step 7: Parse JSON Response
```go
var providerResp ProviderResponseType
if err := json.Unmarshal(body, &providerResp); err != nil {
    return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
}
```
- **OpenAI**: Lines 151-154 (parse openaiResponse)
- **Anthropic**: Lines 150-153 (parse anthropicResponse)
- **Gemini**: Lines 153-156 (parse geminiResponse)

### Step 8: Check Provider-Specific API Error
```go
if providerResp.Error != nil {
    return CompletionResponse{}, fmt.Errorf("API error: %s", providerResp.Error.Message)
}
```
- **OpenAI**: Lines 157-159 (checks openaiResponse.Error)
- **Anthropic**: Lines 156-158 (checks anthropicResponse.Error)
- **Gemini**: Lines 159-161 (checks geminiResponse.Error)
- **Gemini Additional** (Lines 164-171): Checks for empty candidates and SAFETY blocks

## Code Duplication Metrics

### Lines of Duplicated Code (per file):
- **OpenAI** (lines 117-148): 32 lines
- **Anthropic** (lines 115-147): 33 lines
- **Gemini** (lines 115-150): 36 lines
- **Total**: ~101 lines of nearly identical code

### Error Message Consistency:
All three implementations use **identical** error messages:
- `"failed to marshal request: %w"`
- `"failed to create request: %w"`
- `"request failed: %w"`
- `"failed to read response: %w"`
- `"API error: status %d, body: %s"`
- `"failed to parse response: %w"`

## Provider-Specific Logic (NOT duplicated)

The following logic remains unique to each provider and should stay in the client implementations:

### 1. Request Format Conversion
- **OpenAI**: `convertRequest()` creates `openaiRequest` with OpenAI-specific message format
- **Anthropic**: `convertRequest()` creates `anthropicRequest` with content blocks structure
- **Gemini**: `convertRequest()` creates `geminiRequest` with contents/parts structure

### 2. Response Format Conversion
- **OpenAI**: `convertResponse()` extracts from `openaiResponse.Choices[]`
- **Anthropic**: `convertResponse()` extracts from `anthropicResponse.Content[]`
- **Gemini**: `convertResponse()` extracts from `geminiResponse.Candidates[]`

### 3. API Authentication
- **OpenAI**: `Authorization: Bearer <token>` header
- **Anthropic**: `x-api-key: <key>` header
- **Gemini**: API key in URL query parameter

### 4. URL Construction
- **OpenAI**: `{baseURL}/chat/completions`
- **Anthropic**: `{baseURL}/v1/messages`
- **Gemini**: `{baseURL}/v1beta/{model}:generateContent?key={apiKey}`

### 5. Additional Response Validation
- **OpenAI**: Checks `openaiResponse.Error` field
- **Anthropic**: Checks `anthropicResponse.Error` field
- **Gemini**: Checks `geminiResponse.Error` field + validates candidates + safety blocks

## Proposed Helper Function Signature

```go
// doHTTPRequest executes an HTTP request with retry and standard error handling
// Returns raw response body bytes for provider-specific parsing
func (c *BaseLLMClient) doHTTPRequest(
    ctx context.Context,
    method string,
    url string,
    headers map[string]string,
    body interface{},
) ([]byte, error)
```

This function would handle steps 1-6 (marshaling through status checking), returning the raw response body bytes. Each client would then:
1. Parse the bytes into their provider-specific response type
2. Check provider-specific error fields
3. Call their `convertResponse()` method

## Impact of Extraction

After extraction, each `GenerateCompletion` method would be reduced to approximately:

```go
func (c *ProviderClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // Provider-specific: convert request format
    providerReq := c.convertRequest(req)

    // COMMON: execute HTTP request
    body, err := c.doHTTPRequest(ctx, "POST", url, headers, providerReq)
    if err != nil {
        return CompletionResponse{}, err
    }

    // Provider-specific: parse response
    var providerResp providerResponse
    if err := json.Unmarshal(body, &providerResp); err != nil {
        return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
    }

    // Provider-specific: check API errors
    if providerResp.Error != nil {
        return CompletionResponse{}, fmt.Errorf("API error: %s", providerResp.Error.Message)
    }

    // Provider-specific: convert response format
    return c.convertResponse(providerResp), nil
}
```

This reduces each implementation from ~50 lines to ~15 lines, while maintaining all provider-specific logic.
