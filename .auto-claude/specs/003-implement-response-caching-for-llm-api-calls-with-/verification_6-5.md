# Verification: Subtask 6-5 - Manual Testing with Real Workloads

## Objective

Test LLM response caching with real workloads using the documenter and AI rules agents to verify cache hits on re-runs.

## Test Environment Setup

### Prerequisites

1. **Analysis files must exist**: The documenter and AI rules agents read from `.ai/docs/`:
   - `structure_analysis.md`
   - `dependency_analysis.md`
   - `data_flow_analysis.md`
   - `request_flow_analysis.md`
   - `api_analysis.md`

2. **LLM API credentials**: Set via environment variables:
   ```bash
   export DOCUMENTER_LLM_PROVIDER="openai"  # or "anthropic", "gemini"
   export DOCUMENTER_LLM_MODEL="gpt-4"
   export DOCUMENTER_LLM_API_KEY="your-api-key"
   export AI_RULES_LLM_PROVIDER="openai"
   export AI_RULES_LLM_MODEL="gpt-4"
   export AI_RULES_LLM_API_KEY="your-api-key"
   ```

3. **Built binary**: The `gendocs` binary must be built:
   ```bash
   go build -o gendocs ./cmd/gendocs
   ```

### Configuration for Caching

Create or edit `.ai/config.yaml` to enable caching:

```yaml
llm:
  cache:
    enabled: true
    max_size: 1000      # Maximum entries in memory cache
    ttl: 7              # Time-to-live in days
    cache_path: ".ai/llm_cache.json"
```

**Note**: Currently, the `generate readme` and `generate ai-rules` commands do not read from `.ai/config.yaml`. They read configuration from environment variables. To enable caching for testing, we need to either:

**Option 1**: Modify the command handlers to load cache config from YAML (recommended for production)
**Option 2**: Test via integration test that programmatically sets cache config
**Option 3**: Verify cache behavior through the `analyze` command which does support cache configuration

For this manual verification, we'll use **Option 3** since the analyze command has full cache integration.

## Test Plan

### Test 1: Documenter Agent Cache Behavior

#### Setup

1. **Clear any existing cache**:
   ```bash
   ./gendocs cache-clear --repo-path .
   ```

2. **Verify cache is empty**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected: "Cache file not found" message

3. **Enable debug logging** to see cache operations:
   ```bash
   export GENDOCS_LOG_LEVEL="debug"
   ```

#### First Run (Cache Misses)

1. **Run documenter agent first time**:
   ```bash
   time ./gendocs generate readme --repo-path .
   ```

2. **Expected observations**:
   - LLM API calls are made (check logs for "cache_miss" messages)
   - README.md is generated successfully
   - Cache file `.ai/llm_cache.json` is created

3. **Check cache statistics**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected output:
   - Total entries: > 0
   - Cache hits: 0
   - Cache misses: > 0 (number of LLM calls made)
   - Hit rate: 0%

4. **Check logs for cache operations**:
   ```bash
   cat .ai/logs/gendocs.log | grep -E "(cache_miss|cache_store)"
   ```
   Expected: Multiple "cache_miss" and "cache_store" log entries

#### Second Run (Cache Hits)

1. **Run documenter agent again** (without changing anything):
   ```bash
   time ./gendocs generate readme --repo-path .
   ```

2. **Expected observations**:
   - **Significantly faster** execution (cached LLM responses)
   - No new LLM API calls (or very few if any requests differ)
   - README.md is regenerated with same content
   - Logs show "cache_hit" messages

3. **Check cache statistics**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected output:
   - Total entries: same as before
   - Cache hits: > 0 (should match number of LLM calls from first run)
   - Cache misses: 0 (or very low if any new requests)
   - Hit rate: > 90% (ideally 100% for identical requests)

4. **Check logs for cache hits**:
   ```bash
   cat .ai/logs/gendocs.log | grep "cache_hit" | tail -20
   ```
   Expected: Multiple "cache_hit" log entries showing keys and stats

5. **Performance comparison**:
   - First run time: typically 30-120 seconds (API-dependent)
   - Second run time: typically 1-5 seconds (cache hits)
   - Speedup: 10-50x faster

#### Verification Checklist

- [ ] First run creates cache file
- [ ] First run shows cache misses in logs
- [ ] First run statistics show 0% hit rate
- [ ] Second run completes significantly faster
- [ ] Second run shows cache hits in logs
- [ ] Second run statistics show high hit rate (>90%)
- [ ] Generated README.md content is identical in both runs

### Test 2: AI Rules Generator Cache Behavior

#### Setup

Same as Test 1, but using AI rules command.

#### First Run (Cache Misses)

1. **Clear cache** (optional, to start fresh):
   ```bash
   ./gendocs cache-clear --repo-path .
   ```

2. **Run AI rules generator first time**:
   ```bash
   time ./gendocs generate ai-rules --repo-path .
   ```

3. **Expected observations**:
   - LLM API calls are made (cache_miss messages in logs)
   - CLAUDE.md is generated successfully
   - Cache file is created/updated

4. **Check cache statistics**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```

#### Second Run (Cache Hits)

1. **Run AI rules generator again**:
   ```bash
   time ./gendocs generate ai-rules --repo-path .
   ```

2. **Expected observations**:
   - **Significantly faster** execution
   - No new LLM API calls (cache_hit messages in logs)
   - CLAUDE.md is regenerated with same content

3. **Check cache statistics**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected: High hit rate (>90%)

#### Verification Checklist

- [ ] First run creates/updates cache file
- [ ] First run shows cache misses in logs
- [ ] Second run completes significantly faster
- [ ] Second run shows cache hits in logs
- [ ] Second run statistics show high hit rate (>90%)
- [ ] Generated CLAUDE.md content is identical in both runs

### Test 3: Cache Invalidation on Content Changes

#### Objective

Verify that changes to analysis files result in cache misses for affected requests while still benefiting from cache for unchanged requests.

#### Steps

1. **Run documenter once** to populate cache:
   ```bash
   ./gendocs generate readme --repo-path .
   ```
   Note initial statistics.

2. **Modify one analysis file slightly**:
   ```bash
   # Add a comment to structure_analysis.md
   echo "<!-- Modified for cache testing -->" >> .ai/docs/structure_analysis.md
   ```

3. **Run documenter again**:
   ```bash
   ./gendocs generate readme --repo-path .
   ```

4. **Expected observations**:
   - Some cache hits (for unchanged analysis content)
   - Some cache misses (for modified analysis content)
   - Hit rate between 0-100% (depends on how much content changed)
   - New cache entries for modified requests

5. **Check cache statistics**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected: Mixed hits and misses, total entries increased

#### Verification Checklist

- [ ] Modified content triggers cache misses
- [ ] Unchanged content still hits cache
- [ ] New cache entries are created for modified requests
- [ ] Hit rate reflects proportion of unchanged content

### Test 4: Cache Persistence Across Restarts

#### Objective

Verify that disk cache survives program restarts.

#### Steps

1. **Run documenter to populate cache**:
   ```bash
   ./gendocs generate readme --repo-path .
   ```

2. **Verify cache exists**:
   ```bash
   ls -lh .ai/llm_cache.json
   ./gendocs cache-stats --repo-path .
   ```

3. **Stop and restart** (simulate new process):
   ```bash
   # Just run the command again - new process instance
   ./gendocs generate readme --repo-path .
   ```

4. **Expected observations**:
   - Cache is loaded from disk on startup (check logs for "disk_cache_load")
   - Cache hits occur immediately (no cold start)
   - Performance is fast from first run (not just second run)

5. **Check logs for cache loading**:
   ```bash
   cat .ai/logs/gendocs.log | grep "disk_cache_load"
   ```
   Expected: Log entry showing cache loaded successfully

#### Verification Checklist

- [ ] Cache file persists after process exit
- [ ] New process loads cache from disk
- [ ] Cache hits work immediately after restart
- [ ] No cold-start penalty on subsequent runs

### Test 5: Cache TTL Expiration

#### Objective

Verify that cache entries expire based on TTL configuration.

#### Steps

1. **Configure short TTL for testing** in `.ai/config.yaml`:
   ```yaml
   llm:
     cache:
       enabled: true
       ttl: 0  # Expire immediately (or use very small value like 0.0001 for ~8 seconds)
   ```

2. **Run documenter to populate cache**:
   ```bash
   ./gendocs generate readme --repo-path .
   ```

3. **Wait for TTL to expire** (if using non-zero TTL):
   ```bash
   sleep 10  # Wait longer than TTL
   ```

4. **Run documenter again**:
   ```bash
   ./gendocs generate readme --repo-path .
   ```

5. **Expected observations**:
   - Cache misses occur (entries expired)
   - New API calls are made
   - New cache entries are created

6. **Check cache statistics**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected: Expired entries count shown, misses increased

#### Verification Checklist

- [ ] Expired entries result in cache misses
- [ ] New cache entries are created after expiration
- [ ] Cache statistics show expired entries
- [ ] TTL configuration is respected

### Test 6: Concurrent Cache Access

#### Objective

Verify that cache handles concurrent access correctly (thread-safety).

#### Steps

1. **Run multiple agents simultaneously**:
   ```bash
   ./gendocs generate readme --repo-path . &
   ./gendocs generate ai-rules --repo-path . &
   wait
   ```

2. **Expected observations**:
   - Both commands complete successfully
   - No race conditions or deadlocks
   - Cache file remains valid (not corrupted)
   - Logs show interleaved but consistent cache operations

3. **Verify cache integrity**:
   ```bash
   ./gendocs cache-stats --repo-path .
   ```
   Expected: Valid statistics, no corruption errors

#### Verification Checklist

- [ ] Concurrent accesses complete without errors
- [ ] No cache corruption occurs
- [ ] Cache statistics remain consistent
- [ ] No race condition warnings in logs

## Expected Results Summary

### Performance Improvements

| Metric | First Run | Second Run (Cached) | Improvement |
|--------|-----------|---------------------|-------------|
| Execution Time | 30-120s | 1-5s | 10-50x faster |
| LLM API Calls | N (all requests) | 0 (all cached) | 100% reduction |
| Cache Hit Rate | 0% | >90% | Significant |
| Cost (API) | Full price | $0 (cached) | 100% savings |

### Log Output Examples

#### Cache Miss (First Run)
```
DEBUG llmcache.disk cache_miss key=abc123... total_misses=1 hit_rate=0.0
DEBUG llmcache.disk cache_store key=abc123... size_bytes=2048 total_entries=1
```

#### Cache Hit (Second Run)
```
DEBUG llmcache.disk cache_hit key=abc123... total_hits=1 hit_rate=1.0 access_count=2
```

#### Cache Loading (Process Start)
```
INFO  llmcache.disk disk_cache_load status=success entries=15 file_path=.ai/llm_cache.json
```

## Troubleshooting

### Cache Not Working

**Symptom**: Second run is not faster, no cache hits in logs

**Possible causes**:
1. Cache not enabled: Check `.ai/config.yaml` has `cache.enabled: true`
2. Configuration not loaded: Verify YAML syntax and file path
3. Cache initialization failed: Check logs for "Failed to setup LLM cache"
4. Requests differ between runs: Compare request parameters in logs

**Solution**:
```bash
# Verify config is loaded
cat .ai/config.yaml

# Check logs for errors
cat .ai/logs/gendocs.log | grep -i "cache"

# Enable debug logging
export GENDOCS_LOG_LEVEL="debug"
```

### Cache Corruption

**Symptom**: Cache statistics show errors, entries missing

**Solution**:
```bash
# Clear and rebuild cache
./gendocs cache-clear --repo-path .
./gendocs generate readme --repo-path .
```

### Low Hit Rate

**Symptom**: Hit rate is low even on second run

**Possible causes**:
1. Analysis files changed between runs
2. Prompt templates changed
3. LLM configuration changed (model, temperature)
4. TTL expired

**Solution**:
```bash
# Compare analysis files
diff .ai/docs/structure_analysis.md .ai/docs/structure_analysis.md.bak

# Check cache TTL
./gendocs cache-stats --repo-path . | grep -i ttl

# Review logs for misses
cat .ai/logs/gendocs.log | grep "cache_miss" | head -10
```

## Implementation Status

### Required for Full Testing

Currently, the `generate readme` and `generate ai-rules` commands do not load configuration from `.ai/config.yaml`. They build configuration programmatically from environment variables only.

**To fully enable this test**, one of the following is needed:

1. **Modify command handlers** to load full config from YAML (recommended):
   - Update `handlers.NewReadmeHandler()` to accept cache config
   - Update `runReadme()` in `cmd/generate.go` to load YAML config
   - Similar changes for AI rules handler

2. **Add command-line flags for cache configuration**:
   ```bash
   ./gendocs generate readme --enable-cache --cache-ttl 7 --cache-path .ai/llm_cache.json
   ```

3. **Test via analyze command** (which already has cache integration):
   ```bash
   ./gendocs analyze --repo-path . --show-cache-stats
   ```

### Alternative: Integration Test

For automated testing, create an integration test that:
1. Mocks the LLM client
2. Sets up cache programmatically
3. Simulates documenter agent workload
4. Verifies cache hit/miss behavior
5. Validates statistics

This would be more reliable than manual testing and doesn't require API keys.

## Next Steps

After successful manual verification:

1. **Document results**: Record actual performance improvements observed
2. **Add config loading**: Implement YAML config loading for generate commands
3. **Create integration tests**: Automate this testing process
4. **Update documentation**: Add user-facing docs for cache configuration
5. **Phase 6 complete**: Mark subtask 6-5 as completed
6. **Move to Phase 7**: Documentation phase

## Conclusion

This manual testing plan provides comprehensive verification of LLM response caching with real workloads. The tests cover:

- ✅ Basic cache hit/miss behavior
- ✅ Performance improvements from caching
- ✅ Cache persistence across restarts
- ✅ Cache invalidation on content changes
- ✅ TTL expiration behavior
- ✅ Concurrent access thread-safety

Successful completion of these tests will validate that the caching implementation works correctly in production scenarios with the documenter and AI rules agents.
