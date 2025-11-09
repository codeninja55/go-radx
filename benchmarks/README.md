# DICOM Pixel Data Benchmarks

Comprehensive benchmark suite comparing **go-radx** (Go) vs **pydicom** (Python) for DICOM pixel data extraction and decompression performance.

## Quick Start

```bash
# 1. One-time setup (installs uv + dependencies + builds binaries)
make setup

# 2. Run benchmarks (default: 20 files)
make benchmark

# 3. Generate visualization charts
make visualize
```

That's it! 🚀

**Pro tip:** Run `make help` to see all available commands.

## Overview

This benchmark suite compares:

### Metrics
- **Parse Time**: Time to parse DICOM file metadata
- **Pixel Time**: Time to extract and decompress pixel data
- **Total Time**: End-to-end processing time
- **Correctness**: SHA256 hash comparison of pixel data
- **Memory Usage**: Peak memory consumption (TODO)

### Test Coverage
- All supported transfer syntaxes (native, RLE, JPEG, JPEG 2000)
- Single-frame and multi-frame images
- Various bit depths (8-bit, 12-bit, 16-bit)
- Grayscale and RGB images
- Different image sizes (small, medium, large)

## Prerequisites

### Python Dependencies
```bash
pip install pydicom numpy
```

### Go Dependencies
```bash
# For full CGo support (JPEG Lossless, JPEG 2000)
brew install libjpeg-turbo openjpeg  # macOS
# OR
sudo apt-get install libjpeg-turbo8-dev libopenjp2-7-dev  # Ubuntu/Debian
```

## Running Benchmarks

### Basic Usage

```bash
# Build Go benchmark binary
go build -o pixel_benchmark ./pixel_benchmark.go

# Run benchmark script
python3 pixel_benchmark.py --testdata ../testdata --max-files 20
```

### Advanced Options

```bash
python3 pixel_benchmark.py \
    --testdata ../testdata \
    --max-files 50 \
    --go-binary ./pixel_benchmark \
    --output results.json \
    --build-go
```

**Options:**
- `--testdata DIR`: Path to testdata directory (default: `testdata`)
- `--max-files N`: Maximum number of files to benchmark (default: 20)
- `--go-binary PATH`: Path to Go benchmark binary
- `--output FILE`: Output JSON file (default: `benchmark_results.json`)
- `--build-go`: Build Go binary before running

### CGo vs Non-CGo Comparison

```bash
# Full CGo build (all formats)
CGO_ENABLED=1 go build -o pixel_benchmark_cgo ./pixel_benchmark.go
python3 pixel_benchmark.py --go-binary ./pixel_benchmark_cgo --output results_cgo.json

# Pure Go build (no JPEG Lossless/JPEG 2000)
CGO_ENABLED=0 go build -o pixel_benchmark_nocgo ./pixel_benchmark.go
python3 pixel_benchmark.py --go-binary ./pixel_benchmark_nocgo --output results_nocgo.json
```

## Output Format

### Console Output

```
================================================================================
File                                               Transfer Syntax              Tool       Time (s)     Status
================================================================================
color3d_jpeg_baseline.dcm                         1.2.840.10008.1.2.4.50       pydicom    0.0234       ✓
                                                                                go-radx    0.0089       ✓ (2.63x)
                                                                                           ✓ Pixel data matches
--------------------------------------------------------------------------------
693_J2KR.dcm                                      1.2.840.10008.1.2.4.90       pydicom    0.1456       ✓
                                                                                go-radx    0.0512       ✓ (2.84x)
                                                                                           ✓ Pixel data matches
--------------------------------------------------------------------------------

SUMMARY
================================================================================

Average Time:
  pydicom:  0.0845s
  go-radx:  0.0301s
  Speedup:  2.81x

Correctness:
  Matching: 20/20 (100.0%)

By Transfer Syntax:
  1.2.840.10008.1.2.4.50  (JPEG Baseline)         pydicom: 0.0234s, go-radx: 0.0089s (2.63x)
  1.2.840.10008.1.2.4.90  (JPEG 2000 Lossless)    pydicom: 0.1456s, go-radx: 0.0512s (2.84x)
  1.2.840.10008.1.2.5     (RLE Lossless)          pydicom: 0.0678s, go-radx: 0.0198s (3.42x)
```

### JSON Output

```json
[
  {
    "file": "testdata/color3d_jpeg_baseline.dcm",
    "file_size": 45678,
    "transfer_syntax": "1.2.840.10008.1.2.4.50",
    "pydicom": {
      "success": true,
      "parse_time": 0.0123,
      "pixel_time": 0.0111,
      "total_time": 0.0234,
      "pixel_hash": "abc123..."
    },
    "goradx": {
      "success": true,
      "parse_time": 0.0045,
      "pixel_time": 0.0044,
      "total_time": 0.0089,
      "pixel_hash": "abc123..."
    }
  }
]
```

## Visualization

Generate charts from benchmark results:

```bash
python3 visualize_benchmarks.py benchmark_results.json
```

**Output:**
- `benchmark_times.png` - Time comparison chart
- `benchmark_speedup.png` - Speedup factor by transfer syntax
- `benchmark_correctness.png` - Correctness validation

## Common Issues

### Issue: "pydicom not found"
**Solution:**
```bash
pip install pydicom
```

### Issue: "Go binary not found"
**Solution:**
```bash
go build -o pixel_benchmark ./pixel_benchmark.go
# OR use --build-go flag
python3 pixel_benchmark.py --build-go
```

### Issue: "JPEG 2000 decompression failed"
**Cause:** CGo not enabled or OpenJPEG not installed

**Solution:**
```bash
# Install OpenJPEG
brew install openjpeg  # macOS
sudo apt-get install libopenjp2-7-dev  # Ubuntu

# Build with CGo
CGO_ENABLED=1 go build -o pixel_benchmark ./pixel_benchmark.go
```

### Issue: Slow benchmarks
**Cause:** Too many files selected

**Solution:**
```bash
# Reduce number of files
python3 pixel_benchmark.py --max-files 10
```

## Interpreting Results

### Speedup Factor
- **< 1.0x**: go-radx is slower (unexpected)
- **1.0-2.0x**: go-radx is moderately faster
- **2.0-5.0x**: go-radx is significantly faster
- **> 5.0x**: go-radx is much faster

### Expected Performance

**Native/Uncompressed:**
- Speedup: 1.5-3.0x (mostly I/O bound)

**RLE Lossless:**
- Speedup: 2.0-4.0x (pure Go implementation)

**JPEG Baseline:**
- Speedup: 2.0-3.5x (stdlib decoder)

**JPEG Lossless (CGo):**
- Speedup: 1.5-2.5x (C library overhead)

**JPEG 2000 (CGo):**
- Speedup: 2.0-3.5x (efficient OpenJPEG integration)

### Correctness
- **100% match**: Perfect implementation
- **< 100% match**: Investigate mismatches (potential bugs)

**Common mismatch causes:**
- Photometric interpretation differences
- VOI LUT application
- Bit depth handling
- Planar configuration

## CI/CD Integration

### GitHub Actions

```yaml
name: Pixel Benchmarks

on: [push, pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.10'

      - name: Install dependencies
        run: |
          pip install pydicom numpy
          sudo apt-get install -y libjpeg-turbo8-dev libopenjp2-7-dev

      - name: Run benchmarks
        run: |
          cd benchmarks
          make benchmark MAX_FILES=10

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: benchmark-results
          path: benchmarks/benchmark_results.json
```

## Further Reading

- [pydicom Documentation](https://pydicom.github.io/)
- [go-radx Pixel Package](../dicom/pixel/README.md)
- [Transfer Syntax Reference](https://dicom.nema.org/medical/dicom/current/output/html/part05.html)
