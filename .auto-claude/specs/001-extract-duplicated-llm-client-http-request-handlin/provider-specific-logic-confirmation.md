# Provider-Specific Logic Confirmation

## Overview
This document confirms the provider-specific logic that **must remain** in each LLM client implementation after extracting the common HTTP request handling into `doHTTPRequest`.

## Provider-Specific Logic Categories

### 1. Request Format Conversion (`convertRequest` methods)

Each provider has a unique request format that must remain provider-specific:

#### OpenAI (lines 174-217 in openai.go)
- Creates `openaiRequest` struct
- Message format: simple array of messages with role/content
- Tool format: tools array at request level with function definitions
- System prompt: separate message with "system" role
- Code location: `func (c *OpenAIClient) convertRequest(req CompletionRequest) openaiRequest`

#### Anthropic (lines 173-237 in anthropic.go)
- Creates `anthropicRequest` struct
- Message format: content blocks array (text/tool_use/tool_result)
- Tool format: tools array with input_schema
- System prompt: separate system field
- Code location: `func (c *AnthropicClient) convertRequest(req CompletionRequest) anthropicRequest`

#### Gemini (lines 186-286 in gemini.go)
- Creates `geminiRequest` struct
- Message format: contents/parts structure
- Tool format: single tools object with functionDeclarations array
- System prompt: systemInstruction field
- Code location: `func (c *GeminiClient) convertRequest(req CompletionRequest) geminiRequest`

**✅ CONFIRMED:** All `convertRequest` methods remain in their respective client files.

---

### 2. Response Format Conversion (`convertResponse` methods)

Each provider parses responses differently:

#### OpenAI (lines 219-259 in openai.go)
- Extracts from `openaiResponse.Choices[]`
- Handles tool_calls from message structure
- Token usage: prompt_tokens, completion_tokens, total_tokens
- Code location: `func (c *OpenAIClient) convertResponse(resp openaiResponse) CompletionResponse`

#### Anthropic (lines 239-268 in anthropic.go)
- Extracts from `anthropicResponse.Content[]` (content blocks)
- Handles tool_use blocks with name/input fields
- Token usage: input_tokens, output_tokens (sums for total)
- Code location: `func (c *AnthropicClient) convertResponse(resp anthropicResponse) CompletionResponse`

#### Gemini (lines 288-322 in gemini.go)
- Extracts from `geminiResponse.Candidates[].Content.Parts[]`
- Handles FunctionCall objects in parts array
- Token usage: promptTokenCount, candidatesTokenCount, totalTokenCount
- Code location: `func (c *GeminiClient) convertResponse(resp geminiResponse) CompletionResponse`

**✅ CONFIRMED:** All `convertResponse` methods remain in their respective client files.

---

### 3. API Authentication

Each provider uses a different authentication mechanism:

#### OpenAI (line 130 in openai.go)
```go
httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
```
- Method: Bearer token in Authorization header
- Location: Set before calling HTTP request

#### Anthropic (lines 128-129 in anthropic.go)
```go
httpReq.Header.Set("x-api-key", c.apiKey)
httpReq.Header.Set("anthropic-version", "2023-06-01")
```
- Method: API key in x-api-key header
- Additional: anthropic-version header required
- Location: Set before calling HTTP request

#### Gemini (line 126 in gemini.go)
```go
url := fmt.Sprintf("%s/v1beta/%s:generateContent?key=%s", c.baseURL, modelName, c.apiKey)
```
- Method: API key in URL query parameter
- Location: Embedded in URL construction

**✅ CONFIRMED:** Authentication logic remains in each client, passed via headers map or URL to `doHTTPRequest`.

---

### 4. URL Construction

Each provider has a different endpoint structure:

#### OpenAI (line 123 in openai.go)
```go
url := fmt.Sprintf("%s/chat/completions", c.baseURL)
```
- Format: `{baseURL}/chat/completions`
- Default baseURL: `https://api.openai.com/v1`

#### Anthropic (line 121 in anthropic.go)
```go
url := c.baseURL + "/v1/messages"
```
- Format: `{baseURL}/v1/messages`
- Default baseURL: `https://api.anthropic.com`

#### Gemini (line 126 in gemini.go)
```go
url := fmt.Sprintf("%s/v1beta/%s:generateContent?key=%s", c.baseURL, modelName, c.apiKey)
```
- Format: `{baseURL}/v1beta/{model}:generateContent?key={apiKey}`
- Default baseURL: `https://generativelanguage.googleapis.com`
- Includes: Model name and API key in path

**✅ CONFIRMED:** URL construction remains in each client.

---

### 5. Additional Response Validation

Each provider has unique error handling beyond HTTP status codes:

#### OpenAI (lines 157-159 in openai.go)
```go
if oaResp.Error != nil {
    return CompletionResponse{}, fmt.Errorf("API error: %s", oaResp.Error.Message)
}
```
- Checks: `openaiResponse.Error` field
- Error type: `*openaiErrorDetail` with Message, Type, Code

#### Anthropic (lines 156-158 in anthropic.go)
```go
if anResp.Error != nil {
    return CompletionResponse{}, fmt.Errorf("API error: %s", anResp.Error.Message)
}
```
- Checks: `anthropicResponse.Error` field
- Error type: `*anthropicError` with Type, Message

#### Gemini (lines 159-171 in gemini.go)
```go
if gemResp.Error != nil {
    return CompletionResponse{}, fmt.Errorf("API error: %s", gemResp.Error.Message)
}

if len(gemResp.Candidates) == 0 {
    return CompletionResponse{}, fmt.Errorf("no candidates returned by model")
}

if len(gemResp.Candidates) > 0 && gemResp.Candidates[0].FinishReason == "SAFETY" {
    return CompletionResponse{}, fmt.Errorf("response blocked for safety reasons")
}
```
- Checks: `geminiResponse.Error` field
- Additional: Empty candidates validation
- Additional: Safety block checking (finishReason == "SAFETY")
- Error type: `*geminiError` with Code, Message, Status

**✅ CONFIRMED:** All provider-specific error checking remains in each client.

---

## What Will Be Extracted to `doHTTPRequest`

The following logic will be centralized in `doHTTPRequest` (steps 1-6 from the pattern analysis):

1. ✅ **JSON marshaling** of request body
2. ✅ **HTTP request creation** with context
3. ✅ **Header setting** from provided map
4. ✅ **Request execution** with retry client
5. ✅ **Response body reading**
6. ✅ **Status code checking** (200 OK validation)

The function returns raw response bytes `[]byte` to allow provider-specific parsing.

---

## After Refactoring: Typical GenerateCompletion Flow

```go
func (c *ProviderClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // 1. Provider-specific: Convert request format
    providerReq := c.convertRequest(req)

    // 2. Provider-specific: Build URL
    url := c.buildURL()

    // 3. Provider-specific: Build headers
    headers := map[string]string{
        "Content-Type": "application/json",
        "Authorization": fmt.Sprintf("Bearer %s", c.apiKey), // or x-api-key, or none
    }

    // 4. COMMON: Execute HTTP request (extracted)
    body, err := c.doHTTPRequest(ctx, "POST", url, headers, providerReq)
    if err != nil {
        return CompletionResponse{}, err
    }

    // 5. Provider-specific: Parse response
    var providerResp providerResponseType
    if err := json.Unmarshal(body, &providerResp); err != nil {
        return CompletionResponse{}, fmt.Errorf("failed to parse response: %w", err)
    }

    // 6. Provider-specific: Check API errors
    if providerResp.Error != nil {
        return CompletionResponse{}, fmt.Errorf("API error: %s", providerResp.Error.Message)
    }

    // 7. Provider-specific: Convert response format
    return c.convertResponse(providerResp), nil
}
```

---

## Summary Table

| Logic Category | OpenAI | Anthropic | Gemini | Action |
|----------------|--------|-----------|--------|--------|
| convertRequest | ✅ Keep | ✅ Keep | ✅ Keep | Remains in client |
| convertResponse | ✅ Keep | ✅ Keep | ✅ Keep | Remains in client |
| Auth (Bearer header) | ✅ Keep | ❌ N/A | ❌ N/A | Remains in client |
| Auth (x-api-key) | ❌ N/A | ✅ Keep | ❌ N/A | Remains in client |
| Auth (URL param) | ❌ N/A | ❌ N/A | ✅ Keep | Remains in client |
| URL construction | ✅ Keep | ✅ Keep | ✅ Keep | Remains in client |
| API error check | ✅ Keep | ✅ Keep | ✅ Keep | Remains in client |
| Safety validation | ❌ N/A | ❌ N/A | ✅ Keep | Remains in client |
| JSON marshaling | ❌ Extract | ❌ Extract | ❌ Extract | To doHTTPRequest |
| HTTP request creation | ❌ Extract | ❌ Extract | ❌ Extract | To doHTTPRequest |
| Request execution | ❌ Extract | ❌ Extract | ❌ Extract | To doHTTPRequest |
| Response reading | ❌ Extract | ❌ Extract | ❌ Extract | To doHTTPRequest |
| Status checking | ❌ Extract | ❌ Extract | ❌ Extract | To doHTTPRequest |

---

## Verification

This confirmation ensures that:

1. ✅ All provider-specific request/response conversion logic remains intact
2. ✅ Provider authentication mechanisms are preserved
3. ✅ Provider-specific error checking remains in place
4. ✅ Only truly duplicated HTTP handling is extracted
5. ✅ Each provider can evolve independently as APIs change
6. ✅ No breaking changes to public interfaces
7. ✅ Test compatibility is maintained

The extraction of `doHTTPRequest` will reduce duplication while maintaining necessary provider-specific behavior.
