# File Scanning Benchmarks

This document describes the benchmarks for measuring the performance improvements from selective hashing and parallel processing.

## Running the Benchmarks

Run all benchmarks:
```bash
go test -bench=BenchmarkScanFiles -benchmem ./internal/cache/
```

Run specific benchmark:
```bash
go test -bench=BenchmarkScanFiles_WithCacheParallel -benchmem ./internal/cache/
```

Run with verbose output and multiple iterations:
```bash
go test -bench=BenchmarkScanFiles -benchmem -benchtime=5x ./internal/cache/
```

Compare benchmark results across runs:
```bash
go test -bench=BenchmarkScanFiles -benchmem ./internal/cache/ > bench1.txt
# Make changes...
go test -bench=BenchmarkScanFiles -benchmem ./internal/cache/ > bench2.txt
benchstat bench1.txt bench2.txt
```

## Benchmark Descriptions

### BenchmarkScanFiles_NoCacheSequential
**Baseline**: No cache, sequential hashing (single worker)
- Represents worst-case scenario before any optimizations
- All files are hashed one at a time
- Measures: Sequential hashing performance without caching

### BenchmarkScanFiles_WithCacheSequential
**Selective hashing only**: With cache, sequential hashing (single worker)
- All files use cached hashes (simulating unchanged repository)
- Measures: Benefit of selective hashing alone
- Expected: Much faster than baseline if all files are cached

### BenchmarkScanFiles_WithCacheParallel
**Combined optimizations**: With cache, parallel hashing (auto workers)
- All files use cached hashes
- Multiple workers process files in parallel
- Measures: Combined benefit of both optimizations
- Expected: Fastest for incremental scans

### BenchmarkScanFiles_PartialCacheParallel
**Realistic incremental scan**: Partial cache (20% changed), parallel hashing
- Simulates typical scenario where some files changed
- 20% of files need rehashing, 80% use cache
- Measures: Performance in real-world usage
- Expected: Significant improvement over full rehash

### BenchmarkScanFiles_NoCacheParallel
**Parallelism only**: No cache, parallel hashing
- All files need hashing, but done in parallel
- Measures: Benefit of parallel processing alone
- Expected: Faster than sequential, slower than cached scenarios

### BenchmarkScanFiles_LargeRepository
**Large dataset**: 500 files, with cache, parallel hashing
- Tests scalability on larger repositories
- Measures: Performance as repository size grows
- Expected: Consistent performance regardless of size (if cached)

### BenchmarkScanFiles_WithStats
**Detailed metrics**: Reports throughput statistics
- Files per second
- MB per second
- Operation latency in milliseconds
- Useful for understanding absolute performance characteristics

### BenchmarkParallelHashWorkers
**Worker scaling**: Tests different worker counts (1, 2, 4, 8)
- Helps identify optimal worker count for your system
- Measures: Scaling efficiency
- Expected: Diminishing returns after CPU count

## Expected Results

For a typical project with 1000 source files:

| Scenario | Expected Speedup |
|----------|-----------------|
| No cache, sequential | 1x (baseline) |
| No cache, parallel (4 workers) | 2-3x |
| With cache, sequential (all cached) | 10-50x |
| With cache, parallel (all cached) | 20-100x |
| Partial cache (20% changed) | 3-10x |

*Actual results depend on:
- File sizes and count
- CPU core count
- Disk I/O speed
- Cache hit rate (percentage of unchanged files)*

## Interpreting Results

### Key Metrics

1. **ns/op**: Nanoseconds per operation (lower is better)
   - Total time to scan the repository
   - Includes file walking, cache lookups, and hashing

2. **B/op**: Bytes per operation (memory allocations)
   - Lower is better
   - Should be consistent across different configurations

3. **allocs/op**: Allocations per operation
   - Lower is better
   - Indicates GC pressure

4. **files/sec**: Files processed per second
   - Higher is better
   - Throughput metric for file scanning

5. **MB/sec**: Megabytes processed per second
   - Higher is better
   - Accounts for file sizes, not just count

### Performance Goals

The implementation aims to achieve:
- **3-5x faster** for incremental scans with some changes
- **10-100x faster** for incremental scans with no changes
- **2-3x faster** for full scans on multi-core systems

## Customization

To customize benchmarks for your use case:

1. **Change file count**: Modify `numFiles` parameter in benchmark calls
2. **Change change percentage**: Modify `changePercent` in `BenchmarkScanFiles_PartialCacheParallel`
3. **Change worker counts**: Modify `workerCounts` array in `BenchmarkParallelHashWorkers`
4. **Change file content**: Modify `filePatterns` in `setupBenchmarkRepo`
