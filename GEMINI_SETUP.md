# Gemini Configuration Guide

## ✅ Configuration Complete!

Your Gemini API is configured and working correctly.

---

## 📝 Configuration Files Created

### 1. Project Configuration: `.ai/config.yaml`

```yaml
analyzer:
  llm:
    provider: gemini
    model: gemini-2.0-flash-exp
    api_key: YOUR_GEMINI_API_KEY_HERE
    base_url: https://generativelanguage.googleapis.com
    max_tokens: 8192
    temperature: 0.0
    timeout: 180
    retries: 3
```

### 2. Environment Variables Script: `/tmp/gendocs-env.sh`

To use environment variables instead of config file:

```bash
source /tmp/gendocs-env.sh
```

This sets:
- `ANALYZER_LLM_PROVIDER=gemini`
- `ANALYZER_LLM_MODEL=gemini-2.0-flash-exp`
- `ANALYZER_LLM_API_KEY=...`
- And more...

---

## 🚀 Quick Start

### Option 1: Using Config File (Recommended)

The config file `.ai/config.yaml` is already created. Just run:

```bash
# Analyze your codebase
./gendocs analyze --repo-path .

# Generate README
./gendocs generate readme --repo-path .

# Generate AI rules
./gendocs generate ai-rules --repo-path .
```

### Option 2: Using Environment Variables

```bash
# Load environment
source /tmp/gendocs-env.sh

# Run commands
./gendocs analyze --repo-path .
```

---

## 🧪 Test Results

**API Connection:** ✅ SUCCESS

```
Provider: gemini
Model: gemini-2.0-flash-exp
Response: Hello from Gemini!
Tokens - Input: 32, Output: 5
```

---

## 📊 Configuration Details

### Model: gemini-2.0-flash-exp

**Capabilities:**
- Fast inference
- Low cost
- Good quality for code analysis
- 1M token context window

**Limits:**
- Max tokens per request: 8,192 (configured)
- Temperature: 0.0 (deterministic)
- Timeout: 180s for analysis, 240s for generation
- Retries: 3 attempts with exponential backoff

### API Costs (approximate)

For Gemini 2.0 Flash:
- Input: ~$0.10 / 1M tokens
- Output: ~$0.30 / 1M tokens

**Example:** Analyzing a 50-file project:
- ~50K input tokens
- ~30K output tokens
- Cost: ~$0.01 per analysis

---

## 🛠️ Advanced Configuration

### Custom Model

To use a different Gemini model, edit `.ai/config.yaml`:

```yaml
analyzer:
  llm:
    model: gemini-1.5-pro  # More powerful but slower
```

Available models:
- `gemini-2.0-flash-exp` - Fast, experimental (default)
- `gemini-1.5-flash` - Stable, fast
- `gemini-1.5-pro` - Most capable, slower

### Tune Performance

```yaml
analyzer:
  max_workers: 4  # Control parallelism
  llm:
    max_tokens: 4096  # Reduce for faster responses
    timeout: 120      # Shorter timeout
```

### Different Models for Different Tasks

```yaml
analyzer:
  llm:
    model: gemini-2.0-flash-exp  # Fast for analysis

documenter:
  llm:
    model: gemini-1.5-pro  # More capable for writing
```

---

## 🔍 Testing Your Setup

### Quick Test

```bash
./gendocs analyze --repo-path . --max-workers 1
```

This will:
1. Read your codebase
2. Run 5 analysis agents (structure, dependencies, data flow, request flow, API)
3. Generate analysis documents in `.ai/docs/`

### Check Results

```bash
ls -la .ai/docs/
cat .ai/docs/code_structure.md
```

### Generate Documentation

```bash
# Generate README from analysis
./gendocs generate readme --repo-path .

# Export to HTML
./gendocs generate export --input README.md --output README.html
```

---

## 🐛 Troubleshooting

### API Key Issues

**Error:** `401 Unauthorized`

**Solution:** Verify API key in `.ai/config.yaml` or environment variables

```bash
# Check current config
cat .ai/config.yaml | grep api_key

# Re-export env vars
source /tmp/gendocs-env.sh
```

### Rate Limiting

**Error:** `429 Too Many Requests`

**Solution:** Reduce `max_workers`:

```bash
./gendocs analyze --repo-path . --max-workers 1
```

Or edit `.ai/config.yaml`:

```yaml
analyzer:
  max_workers: 1  # More conservative
```

### Timeout Issues

**Error:** `context deadline exceeded`

**Solution:** Increase timeout in `.ai/config.yaml`:

```yaml
analyzer:
  llm:
    timeout: 300  # 5 minutes instead of 3
```

---

## 📚 Next Steps

1. **Analyze a Real Project**
   ```bash
   ./gendocs analyze --repo-path ../your-project
   ```

2. **Generate Documentation**
   ```bash
   ./gendocs generate readme --repo-path ../your-project
   ./gendocs generate ai-rules --repo-path ../your-project
   ```

3. **Export to HTML**
   ```bash
   ./gendocs generate export --repo-path ../your-project
   ```

4. **Use Custom Prompts**
   ```bash
   mkdir -p ../your-project/.ai/prompts
   cp examples/custom-prompts/basic-override.yaml ../your-project/.ai/prompts/
   ```

---

## 🔐 Security Notes

**⚠️ IMPORTANT:** Your API key is stored in:
- `.ai/config.yaml` (this file is in `.gitignore`)
- `/tmp/gendocs-env.sh` (temporary, cleaned on reboot)

**Never commit API keys to git!**

The `.gitignore` already excludes:
- `.ai/config.yaml`
- `.env`
- `*.key`

---

## 📞 Support

- **Documentation:** See `README.md` and `docs/EXPORT.md`
- **Examples:** Check `examples/custom-prompts/`
- **Issues:** https://github.com/user/gendocs/issues
