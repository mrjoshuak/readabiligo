# ReadabiliGo Performance Benchmarks

This directory contains benchmark tests for measuring the performance of the ReadabiliGo library. These benchmarks are designed to evaluate different aspects of the extraction process and identify areas for optimization.

## Benchmark Categories

1. **Default Extraction (BenchmarkDefaultExtraction)**
   - Measures baseline performance for the complete extraction process
   - Tests different document sizes (small, medium, large)
   - Tests different content types (reference, article, technical)
   - Tests edge cases (nested content, table layouts, etc.)

2. **Content Type Awareness (BenchmarkContentTypeAwareExtraction)**
   - Compares auto-detection vs. explicit content type settings
   - Measures performance impact of content type detection
   - Tests all five content types (Reference, Article, Technical, Minimal, Error)

3. **Feature Overhead (BenchmarkFeatureOverhead)**
   - Measures the performance impact of optional features
   - Tests different combinations of features:
     - Content digests
     - Node indexes
     - Important link preservation
     - All features enabled

4. **DOM Operations (BenchmarkDOMOperations)**
   - Focuses on DOM traversal and manipulation performance
   - Tests different document complexities
   - Identifies bottlenecks in DOM processing

5. **Timeout Impact (BenchmarkTimeoutImpact)**
   - Measures how different timeout values affect performance
   - Tests various timeout settings
   - Evaluates the overhead of timeout checking

6. **Memory Usage (BenchmarkMemoryUsage)**
   - Measures the impact of different buffer sizes on performance
   - Tests small, default, and large buffer configurations
   - Evaluates memory-performance tradeoffs

## Running the Benchmarks

To run all benchmarks:

```bash
cd readabiligo
go test -bench=. ./test/benchmark_test.go
```

To run a specific benchmark:

```bash
go test -bench=BenchmarkDefaultExtraction ./test/benchmark_test.go
```

To generate a CPU profile:

```bash
go test -bench=. -cpuprofile=cpu.prof ./test/benchmark_test.go
```

To view the profile:

```bash
go tool pprof cpu.prof
```

## Analyzing Results

The benchmark results include:

- **Iterations**: Number of benchmark iterations (higher is better for performance)
- **Time per operation**: Average time per extraction operation
- **Memory allocations**: Number and size of memory allocations

When analyzing results, look for:

1. Significant differences between default and specialized extraction modes
2. Performance impact of different features
3. Scaling behavior with document size and complexity
4. Memory allocation patterns
5. Impact of timeout settings

## Using Benchmarks for Optimization

1. Run benchmarks to establish a baseline
2. Implement optimizations for identified bottlenecks
3. Run benchmarks again to measure improvement
4. Document the performance changes

## Test Data Requirements

These benchmarks require test data files, which can be downloaded using the script in the `data` directory:

```bash
cd test/data
chmod +x download_real_world_examples.sh
./download_real_world_examples.sh
```

Some benchmarks will be skipped if required test files are not available.