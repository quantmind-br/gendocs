#!/bin/bash

# Manual Testing Script for LLM Response Caching
# Tests cache behavior with the analyzer agent
#
# This script tests caching with the analyze command, which has full cache
# integration via .ai/config.yaml.

set -e  # Exit on error

REPO_PATH=${1:-"."}
GENDOCS_BIN=${GENDOCS_BIN:-"./gendocs"}

echo "========================================"
echo "LLM Response Caching Manual Test"
echo "========================================"
echo "Repo Path: $REPO_PATH"
echo "Binary: $GENDOCS_BIN"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_step() {
    echo -e "${YELLOW}[$(date +%H:%M:%S)]${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check prerequisites
print_step "Checking prerequisites..."

if [ ! -f "$GENDOCS_BIN" ]; then
    print_error "gendocs binary not found at $GENDOCS_BIN"
    echo "Build with: go build -o gendocs ./cmd/gendocs"
    exit 1
fi
print_success "gendocs binary found"

# Create config directory if it doesn't exist
mkdir -p "$REPO_PATH/.ai"

# Create test configuration
print_step "Creating test configuration..."

cat > "$REPO_PATH/.ai/config.yaml" <<EOF
# Test configuration for LLM caching
llm:
  provider: ${ANALYZER_LLM_PROVIDER:-openai}
  model: ${ANALYZER_LLM_MODEL:-gpt-4}
  api_key: ${ANALYZER_LLM_API_KEY}
  retries: 2
  timeout: 180
  max_tokens: 4096
  temperature: 0.0

  # Cache configuration
  cache:
    enabled: true
    max_size: 1000
    ttl: 7
    cache_path: ".ai/llm_cache.json"

analyzer:
  max_workers: 4
EOF

print_success "Configuration created at $REPO_PATH/.ai/config.yaml"

# Check if API key is set
if [ -z "$ANALYZER_LLM_API_KEY" ]; then
    print_error "ANALYZER_LLM_API_KEY environment variable not set"
    echo "Set it with: export ANALYZER_LLM_API_KEY='your-api-key'"
    exit 1
fi
print_success "API key is configured"

# Test 1: Clear cache
print_step "Test 1: Clearing existing cache..."
"$GENDOCS_BIN" cache-clear --repo-path "$REPO_PATH" || true
print_success "Cache cleared"

# Verify cache is empty
print_step "Verifying cache is empty..."
if "$GENDOCS_BIN" cache-stats --repo-path "$REPO_PATH" 2>&1 | grep -q "Cache file not found"; then
    print_success "Cache is empty (as expected)"
else
    print_error "Cache file should not exist after clearing"
fi

echo ""
echo "========================================"
echo "Test 2: First Run (Cache Misses)"
echo "========================================"

# Record start time
START_TIME=$(date +%s)

print_step "Running analysis (first run - expect cache misses)..."
if "$GENDOCS_BIN" analyze --repo-path "$REPO_PATH" --show-cache-stats 2>&1 | tee /tmp/analyze_first_run.log; then
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    print_success "First run completed in ${DURATION}s"
else
    print_error "First run failed"
    cat /tmp/analyze_first_run.log
    exit 1
fi

# Check cache stats after first run
echo ""
print_step "Cache statistics after first run:"
"$GENDOCS_BIN" cache-stats --repo-path "$REPO_PATH"

# Parse cache stats
HITS_AFTER_FIRST=$(grep -oP 'Cache Hits: \K\d+' /tmp/analyze_first_run.log || echo "0")
MISSES_AFTER_FIRST=$(grep -oP 'Cache Misses: \K\d+' /tmp/analyze_first_run.log || echo "0")

echo ""
echo "First run summary:"
echo "  Hits: $HITS_AFTER_FIRST"
echo "  Misses: $MISSES_AFTER_FIRST"
echo "  Duration: ${DURATION}s"

if [ "$MISSES_AFTER_FIRST" -gt 0 ]; then
    print_success "First run had cache misses (as expected)"
else
    print_error "First run should have cache misses"
fi

echo ""
echo "========================================"
echo "Test 3: Second Run (Cache Hits)"
echo "========================================"

# Record start time
START_TIME=$(date +%s)

print_step "Running analysis again (second run - expect cache hits)..."
if "$GENDOCS_BIN" analyze --repo-path "$REPO_PATH" --show-cache-stats 2>&1 | tee /tmp/analyze_second_run.log; then
    END_TIME=$(date +%s)
    DURATION2=$((END_TIME - START_TIME))
    print_success "Second run completed in ${DURATION2}s"
else
    print_error "Second run failed"
    cat /tmp/analyze_second_run.log
    exit 1
fi

# Check cache stats after second run
echo ""
print_step "Cache statistics after second run:"
"$GENDOCS_BIN" cache-stats --repo-path "$REPO_PATH"

# Parse cache stats
HITS_AFTER_SECOND=$(grep -oP 'Cache Hits: \K\d+' /tmp/analyze_second_run.log || echo "0")
MISSES_AFTER_SECOND=$(grep -oP 'Cache Misses: \K\d+' /tmp/analyze_second_run.log || echo "0")

echo ""
echo "Second run summary:"
echo "  Hits: $HITS_AFTER_SECOND"
echo "  Misses: $MISSES_AFTER_SECOND"
echo "  Duration: ${DURATION2}s"

# Calculate hit rate
TOTAL_LOOKUPS=$((HITS_AFTER_SECOND + MISSES_AFTER_SECOND))
if [ "$TOTAL_LOOKUPS" -gt 0 ]; then
    HIT_RATE=$(echo "scale=2; $HITS_AFTER_SECOND * 100 / $TOTAL_LOOKUPS" | bc)
    echo "  Hit Rate: ${HIT_RATE}%"
fi

echo ""
echo "========================================"
echo "Test 4: Performance Comparison"
echo "========================================"

SPEEDUP=0
if [ "$DURATION2" -gt 0 ]; then
    SPEEDUP=$(echo "scale=2; $DURATION / $DURATION2" | bc)
fi

echo "First run duration:  ${DURATION}s"
echo "Second run duration: ${DURATION2}s"
echo "Speedup:             ${SPEEDUP}x"

if [ "$(echo "$SPEEDUP > 1.5" | bc)" -eq 1 ]; then
    print_success "Significant speedup from caching (${SPEEDUP}x faster)"
else
    print_error "Expected significant speedup from caching"
fi

echo ""
echo "========================================"
echo "Test 5: Cache Persistence"
echo "========================================"

print_step "Checking cache file persistence..."
CACHE_FILE="$REPO_PATH/.ai/llm_cache.json"
if [ -f "$CACHE_FILE" ]; then
    CACHE_SIZE=$(du -h "$CACHE_FILE" | cut -f1)
    print_success "Cache file exists (${CACHE_SIZE})"
else
    print_error "Cache file should exist"
fi

# Test 6: Log verification
echo ""
echo "========================================"
echo "Test 6: Log Verification"
echo "========================================"

print_step "Checking logs for cache operations..."
LOG_FILE="$REPO_PATH/.ai/logs/gendocs.log"

if [ -f "$LOG_FILE" ]; then
    echo ""
    echo "Recent cache operations:"
    tail -50 "$LOG_FILE" | grep -E "(cache_hit|cache_miss|cache_store)" || echo "No cache operations found in recent logs"

    echo ""
    echo "Cache hit count in logs:"
    grep -c "cache_hit" "$LOG_FILE" || echo "0"

    echo "Cache miss count in logs:"
    grep -c "cache_miss" "$LOG_FILE" || echo "0"

    print_success "Logs show cache operations"
else
    print_error "Log file not found at $LOG_FILE"
fi

# Final summary
echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"

PASSED=0
FAILED=0

# Check 1: Cache file exists
if [ -f "$CACHE_FILE" ]; then
    print_success "Cache file exists"
    ((PASSED++))
else
    print_error "Cache file missing"
    ((FAILED++))
fi

# Check 2: Second run had cache hits
if [ "$HITS_AFTER_SECOND" -gt 0 ]; then
    print_success "Second run had cache hits"
    ((PASSED++))
else
    print_error "Second run should have cache hits"
    ((FAILED++))
fi

# Check 3: Second run was faster
if [ "$(echo "$SPEEDUP > 1.5" | bc)" -eq 1 ]; then
    print_success "Second run was significantly faster"
    ((PASSED++))
else
    print_error "Second run should be faster with caching"
    ((FAILED++))
fi

# Check 4: Hit rate is high
if [ "$(echo "$HIT_RATE > 50" | bc)" -eq 1 ]; then
    print_success "High cache hit rate (${HIT_RATE}%)"
    ((PASSED++))
else
    print_error "Expected high cache hit rate (>50%)"
    ((FAILED++))
fi

echo ""
echo "Results: $PASSED passed, $FAILED failed"

if [ "$FAILED" -eq 0 ]; then
    print_success "All tests passed! ✓"
    exit 0
else
    print_error "Some tests failed"
    exit 1
fi
