# go-radx Benchmarking Guide

This directory contains comprehensive benchmarks for measuring the performance of go-radx across various operations.

## Overview

The benchmarking suite covers:

1. **Dataset Operations** - Tag access, modification, iteration, and management
2. **Anonymization** - DICOM PS3.15 compliant de-identification operations
3. **LUT Transformations** - Window/level, modality LUT, presentation LUT, and palette color
4. **Pixel Data** - Extraction and decompression across transfer syntaxes
5. **Memory Profiling** - Memory usage and allocation patterns

## Running Benchmarks

### Quick Start

Run all benchmarks:

```bash
cd benchmarks
go test -bench=. -benchmem
```

### Specific Benchmark Categories

Run dataset operation benchmarks:

```bash
go test -bench=BenchmarkDataSet -benchmem
```

Run anonymization benchmarks:

```bash
go test -bench=BenchmarkAnonymize -benchmem
```

Run LUT transformation benchmarks:

```bash
go test -bench=BenchmarkApply -benchmem
```

### Performance Profiling

Generate CPU profile:

```bash
go test -bench=BenchmarkDataSetWalk -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

Generate memory profile:

```bash
go test -bench=BenchmarkAnonymize -memprofile=mem.prof
go tool pprof mem.prof
```

Generate trace for detailed analysis:

```bash
go test -bench=BenchmarkApplyWindowLevel -trace=trace.out
go tool trace trace.out
```

## Benchmark Categories

### 1. Dataset Benchmarks (`dataset_bench_test.go`)

Measures performance of core dataset operations:

- **Creation**: `BenchmarkDataSetCreation`, `BenchmarkElementCreation`
- **Tag Operations**: `BenchmarkDataSetAdd`, `BenchmarkDataSetGet`, `BenchmarkDataSetContains`
- **Iteration**: `BenchmarkDataSetWalk`, `BenchmarkDataSetWalkModify`
- **Modification**: `BenchmarkDataSetModifyTags` (PatientName, PatientID, UIDs)
- **Cleanup**: `BenchmarkDataSetRemovePrivateTags`, `BenchmarkDataSetRemoveGroupTags`

**Key Metrics**:
- Operations per second
- Memory allocations per operation
- Bytes allocated per operation

**Usage**:
```bash
go test -bench=BenchmarkDataSet -benchmem -benchtime=5s
```

### 2. Anonymization Benchmarks (`anonymize_bench_test.go`)

Measures DICOM de-identification performance:

- **Profile Creation**: `BenchmarkAnonymizerCreation`
- **Anonymization**: `BenchmarkAnonymizeBasicDataSet`
- **Options**: `BenchmarkAnonymizeWithOptions` (RetainUIDs, CleanDescriptors, etc.)
- **Actions**: `BenchmarkActionApplication` (Dummy, Remove, UID generation)
- **Memory**: `BenchmarkAnonymizeMemory`

**Profiles Tested**:
- Basic Application Level Confidentiality Profile
- Clean Profile (with descriptors and pixel data cleaning)
- Retain UIDs Profile
- Retain Device Identity Profile

**Usage**:
```bash
go test -bench=BenchmarkAnonymize -benchmem
```

### 3. LUT Transformation Benchmarks (`lut_bench_test.go`)

Measures image transformation performance:

- **Window/Level**: `BenchmarkApplyWindowLevel` (512×512, 1024×1024, 2048×2048)
- **Modality LUT**: `BenchmarkApplyModalityLUT` (HU conversion)
- **Full Pipeline**: `BenchmarkApplyFullImagePipeline` (Modality + VOI LUT)
- **Presentation LUT**: `BenchmarkApplyPresentationLUT` (Shape and Table-based)
- **Palette Color**: `BenchmarkApplyPaletteColorLUT`
- **Segmented LUT**: `BenchmarkSegmentedLUTExpansion`
- **Color Space**: `BenchmarkColorSpaceConversion` (sRGB, Linear RGB)

**Image Sizes**:
- 512×512 (typical X-ray)
- 1024×1024 (typical CT slice)
- 2048×2048 (high-resolution imaging)

**Bit Depths**:
- 8-bit (display)
- 16-bit (acquisition)

**Usage**:
```bash
go test -bench=BenchmarkApply -benchmem
```

### 4. Pixel Data Benchmarks (`pixel_benchmark.go`)

Command-line tool for benchmarking pixel extraction across transfer syntaxes:

```bash
# Build the benchmark tool
go build -o pixel_benchmark pixel_benchmark.go

# Run against a DICOM file
./pixel_benchmark ../testdata/sample.dcm
```

**Output** (JSON):
```json
{
  "success": true,
  "parse_time_us": 1234.56,
  "pixel_time_us": 5678.90,
  "transfer_syntax": "1.2.840.10008.1.2.5",
  "pixel_hash": "abc123...",
  "image_shape": [512, 512]
}
```

## Interpreting Results

### Throughput Metrics

Benchmarks report:
- **ns/op**: Nanoseconds per operation (lower is better)
- **MB/s**: Megabytes per second for data processing
- **allocs/op**: Memory allocations per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)

Example output:
```
BenchmarkDataSetWalk/100_elements-8   50000   28394 ns/op   0 allocs/op   0 B/op
```

This means:
- Processed 100-element dataset in ~28μs
- Zero allocations (very efficient!)
- Zero bytes allocated

### Memory Efficiency

Lower allocation counts indicate better memory efficiency:

```
BenchmarkAnonymizeBasicDataSet/100_elements-8   5000   345678 ns/op   152 allocs/op   12584 B/op
```

This shows:
- Anonymization took ~346μs for 100 elements
- Made 152 allocations
- Allocated ~12.3 KB total

### Scaling Analysis

Run benchmarks with different sizes to analyze scaling:

```bash
go test -bench=BenchmarkDataSetWalk -benchmem | grep elements
```

Expected O(n) linear scaling for most operations.

## Comparison with pydicom

Use the existing Python benchmarking scripts to compare:

```bash
# Run Python benchmarks
python pixel_benchmark.py ../testdata/sample.dcm

# Compare results
python visualize_benchmarks.py
```

Current performance metrics (from PERFORMANCE_ANALYSIS.md):
- **go-radx**: 1.68x faster than pydicom for pixel extraction
- **Memory**: Lower allocation overhead
- **Concurrency**: Better multi-threaded performance

## Continuous Benchmarking

### Baseline Snapshots

Save benchmark results for comparison:

```bash
go test -bench=. -benchmem > baseline.txt
```

### Regression Detection

Compare against baseline:

```bash
go test -bench=. -benchmem > current.txt
benchstat baseline.txt current.txt
```

Install benchstat:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

### CI Integration

Add to CI pipeline:

```yaml
- name: Run Benchmarks
  run: |
    cd benchmarks
    go test -bench=. -benchmem -benchtime=10s > bench_results.txt
```

## Best Practices

1. **Run Multiple Times**: Use `-benchtime=10s` or `-count=10` for stable results
2. **Warm Up**: First run may include compilation; look at subsequent runs
3. **Isolate**: Close other applications to minimize system noise
4. **Consistent Hardware**: Run on same machine for comparisons
5. **Check Allocations**: Zero allocations is ideal for hot paths
6. **Profile Outliers**: Use profiling tools to investigate slow benchmarks

## Adding New Benchmarks

Template for new benchmarks:

```go
func BenchmarkMyOperation(b *testing.B) {
    // Setup
    setup := prepareTestData(b)

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        result := myOperation(setup)
        if result == nil {
            b.Fatal("operation failed")
        }
    }
}
```

Key points:
- Use `b.Helper()` in setup functions
- Call `b.ResetTimer()` after setup
- Use `b.ReportAllocs()` for memory metrics
- Call `b.Fatal()` for unexpected errors
- Set `b.SetBytes()` for throughput metrics

## Troubleshooting

### Benchmark Not Running

Check test file naming:
- Must end in `_test.go`
- Benchmark functions must start with `Benchmark`

### Inconsistent Results

Causes:
- System load from other processes
- CPU throttling/turbo boost
- Background tasks

Solutions:
- Close unnecessary applications
- Run with `nice` priority: `nice -n -20 go test -bench=.`
- Use `-benchtime=30s` for longer, more stable runs

### Out of Memory

For large benchmarks:
- Reduce test data sizes
- Run specific benchmarks instead of all: `-bench=BenchmarkSpecific`
- Increase system memory limits

## References

- [Go Benchmarking Guide](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Performance Optimization](https://github.com/golang/go/wiki/Performance)
- [benchstat Documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [pprof Tutorial](https://github.com/google/pprof/blob/main/doc/README.md)