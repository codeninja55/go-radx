# Comparative benchmarks: go-radx versus the Python reference libraries

PRD section 11.3 commits go-radx to comparative benchmarks against the reference libraries, with
compression and transfer-syntax throughput as the priority area, DIMSE C-STORE transfer throughput
against `pynetdicom`, and FHIR (de)serialization / HL7 v2 parse as secondary areas. This document
is produced by the harness in `tools/bench-compare/`; the tables between the generated-block
markers are rewritten by `mise run bench:compare`, so republishing numbers is one command.

## How to run

```bash
mise run bench:compare              # full publishing run (median of 7 samples)
mise run bench:compare -- --mode smoke   # tiny-N harness validation only
```

Prerequisites: `uv` on PATH (the harness syncs its pinned Python environment), Go via mise, and -
for the cgo codec rows - the native codec libraries (OpenJPEG, libjpeg-turbo, CharLS) resolvable
through pkg-config. When the native libraries do not link, the go-radx codec rows are marked
`unavailable` rather than silently dropped.

## What is measured

- **DICOM Part 10 decode** - full-file decode (preamble, file meta, dataset) of the five
  uncompressed fixtures `dicom/codec_bench_test.go` walks, via the committed `BenchmarkReadFile`
  on the Go side and `pydicom.dcmread` on the Python side.
- **Pixel decode per transfer syntax** (`pixel-decode-*`) - RLE Lossless, JPEG-LS (lossless +
  near-lossless), JPEG Baseline/Extended/Lossless, JPEG 2000 Lossless, and HTJ2K, each over the
  same vendored fixture on both sides, user-facing path against user-facing path. The Go side
  parses the Part 10 object once with `dicom.ReadPixelData`, then each timed operation iterates
  `PixelData.Frames()`: fragment-to-frame mapping plus codec decode of every frame. The Python
  side calls `pydicom.dcmread` once, then each timed operation runs `pydicom.pixels.pixel_array`
  with `decoding_plugin="pylibjpeg"`: option extraction, validation, decode of every frame, and
  reshape/colour processing. Each side does its own library's full decode work per operation.
- **Raw codec decode** (`codec-decode-*`) - go-internal rows reused from the committed per-codec
  `go test -bench` suites: bare `codec.Decode` over pre-extracted encoded frames, with frame
  mapping and dataset handling outside the timed loop. These have **no Python pair** (pydicom
  exposes no equivalent raw-codec entry point) and are published for codec-layer trend tracking
  only; never compare them against a Python column.
- **Pixel encode per transfer syntax** - RLE Lossless over byte-identical synthetic frames
  (replicas of `syntheticFrameRLE`), and JPEG-LS Lossless / JPEG 2000 Lossless re-encode of the
  decoded frames of the same fixtures. JPEG-family and HTJ2K encode are unsupported on **both**
  sides and recorded as such.
- **DIMSE loopback C-STORE** - same-stack pairs: go-radx SCU to go-radx SCP versus pynetdicom SCU
  to pynetdicom SCP, one association per sample carrying 200 small (~100 KiB Segmentation) plus
  20 medium (~2 MiB MR) sequential C-STOREs over 127.0.0.1. This measures stack throughput (PDU
  framing plus dataset codec), **not** interop; the interop gates live in section 11.1.
- **HL7 v2 parse** - `hl7v2.Parse` versus `hl7.parse` over the corpus ADT-A01 and ORU-R01
  fixtures (CR segment terminators, identical bytes both sides).
- **FHIR R5 Bundle** - unmarshal, marshal, and validate over `testdata/fhir/r5/Bundle.json`
  (a synthetic transaction bundle) via `fhir/r5` versus `fhir.resources`.

## Methodology

- **Same fixtures both sides.** Every row reads the identical vendored file under `testdata/`
  (or a byte-identical synthetic frame); nothing is fetched at benchmark time and the DIMSE area
  never leaves loopback.
- **Warmup then median-of-N wall clock.** Each series runs unmeasured warmup samples, then N
  measured samples (N = 7 full, N = 2 smoke; the PRD floor for published numbers is N >= 5) of
  `ops` back-to-back operations; the published figure is the median sample. Go `go test -bench`
  rows run `-count=N+warmup` repetitions, the first `warmup` repetitions are discarded, and the
  remaining N are the samples; their ns/op, MB/s, and allocs/op all derive from the measured
  repetitions (median), never from the first repetition alone. Both sides are normalized to
  ns/op, ops/sec, and MB/s by the same renderer arithmetic.
- **Allocations** are reported where the tooling provides them: `allocs/op` from `go test -bench`
  and a `runtime.MemStats` mallocs delta in the gobench areas. CPython has no equivalent metric,
  so the Python column omits it.
- **Environment is recorded** in the provenance block: CPU model (`sysctl -n
  machdep.cpu.brand_string` on macOS, `/proc/cpuinfo` on Linux), memory, OS, Go version, Python
  version, and the exact Python library pins from `tools/bench-compare/uv.lock`.

## Honest caveats

- **Cross-language comparison is indicative only.** The two sides build different object models
  (go-radx decodes into typed Go structs; pydicom builds lazy Python datasets, fhir.resources
  builds pydantic models), run different memory managers (tracing GC versus refcounting), and
  cross different FFI boundaries. Ratios say "what a user of each stack experiences on this
  operation", not "language X is Nx faster than language Y".
- **The codec rows compare C against C.** pylibjpeg's codecs are C/C++ under Python exactly as
  go-radx's cgo codecs are C/C++ under Go (both link OpenJPEG and CharLS lineages); those rows
  mostly measure binding overhead and frame-handling, not codec algorithmics.
- **pydantic validates during parse**, so fhir.resources has no separate validate step; the FHIR
  validate row is go-radx-only with the Python cost folded into unmarshal.
- **DIMSE rows exclude association setup** and neither side persists received instances; the
  measured window is first C-STORE request to last response on an established association.
- **One host, one run.** Numbers below come from the single machine named in the provenance
  block; they are tracked for trend and regression, not as portable absolutes.

## Results

<!-- bench-compare:begin provenance -->

- Generated: 2026-07-04T22:09:26Z (mode: full)
- Host: Apple M1 / 17179869184 bytes RAM / Darwin 25.5.0 (arm64)
- Go: go version go1.26.4 darwin/arm64
- Python: 3.12.13
- Python pins: fhir.resources 8.2.0, hl7 0.4.5, numpy 2.4.6, pydicom 3.0.2, pyjpegls 1.5.1, pylibjpeg 2.1.0, pylibjpeg-libjpeg 2.4.0, pylibjpeg-openjpeg 2.5.0, pylibjpeg-rle 2.2.0, pynetdicom 3.0.4

<!-- bench-compare:end provenance -->

<!-- bench-compare:begin generated tables -->

### DICOM: Part 10 decode and per-transfer-syntax pixel codecs

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| codec-decode-htj2k | HTJ2KLossless_08_RGB.dcm | 10.02M | - | 3.00 | - | - | - |
| codec-decode-htj2k | HTJ2K_08_RGB.dcm | 8.89M | - | 3.00 | - | - | - |
| codec-decode-j2k | liver_j2k.dcm | 7.25M | - | 9.00 | - | - | - |
| codec-decode-jpeg | JPGExtended.dcm | 584.21k | - | 3.00 | - | - | - |
| codec-decode-jpeg | JPGLosslessP14SV1_1s_1f_8b.dcm | 4.25M | - | 3.00 | - | - | - |
| codec-decode-jpeg | SC_jpeg_no_color_transform.dcm | 111.15k | - | 3.00 | - | - | - |
| codec-decode-jpegls | JPEGLSNearLossless_08.dcm | 3.29k | - | 3.00 | - | - | - |
| codec-decode-jpegls | JPEGLSNearLossless_16.dcm | 231.76k | - | 3.00 | - | - | - |
| codec-decode-jpegls | MR_small_jpeg_ls_lossless.dcm | 109.64k | - | 3.00 | - | - | - |
| codec-decode-jpegls | SC_rgb_jls_lossy_sample.dcm | 31.55k | - | 3.00 | - | - | - |
| codec-decode-rle | liver_rle.dcm | 444.38k | - | 6.00 | - | - | - |
| dicom-part10-read | MR-SIEMENS-DICOM-WithOverlays.dcm | 113.28k | 4.51k | 1.38k | 272.87k | 1.87k | 2.4x |
| dicom-part10-read | MR2_UNCI.dcm | 213.94k | 9.81k | 1.11k | 342.23k | 6.13k | 1.6x |
| dicom-part10-read | SC_rgb_expb.dcm | 28.43k | 1.10k | 404.00 | 162.72k | 192.28 | 5.7x |
| dicom-part10-read | basic-text-sr.dcm | 45.40k | 65.38 | 1.25k | 538.80k | 5.51 | 11.9x |
| dicom-part10-read | liver.dcm | 101.19k | 1.01k | 1.57k | 767.63k | 133.70 | 7.6x |
| pixel-decode-htj2k | HTJ2KLossless_08_RGB.dcm | 9.81M | - | 9.00 | 8.80M | - | 0.9x |
| pixel-decode-htj2k | HTJ2K_08_RGB.dcm | 8.91M | - | 9.00 | 8.66M | - | 1.0x |
| pixel-decode-j2k | liver_j2k.dcm | 7.21M | - | 17.05 | 6.54M | - | 0.9x |
| pixel-decode-jpeg | JPGExtended.dcm | 588.01k | - | 9.00 | 1.35M | - | 2.3x |
| pixel-decode-jpeg | JPGLosslessP14SV1_1s_1f_8b.dcm | 4.30M | - | 9.05 | 13.98M | - | 3.3x |
| pixel-decode-jpeg | SC_jpeg_no_color_transform.dcm | 114.60k | - | 9.00 | 774.48k | - | 6.8x |
| pixel-decode-jpegls | JPEGLSNearLossless_08.dcm | 3.17k | - | 9.00 | 64.26k | - | 20.3x |
| pixel-decode-jpegls | JPEGLSNearLossless_16.dcm | 232.13k | - | 9.00 | 297.47k | - | 1.3x |
| pixel-decode-jpegls | MR_small_jpeg_ls_lossless.dcm | 81.53k | - | 9.00 | 571.53k | - | 7.0x |
| pixel-decode-jpegls | SC_rgb_jls_lossy_sample.dcm | 32.49k | - | 9.00 | 316.31k | - | 9.7x |
| pixel-decode-rle | liver_rle.dcm | 435.89k | - | 14.00 | 422.64k | - | 1.0x |
| pixel-encode-htj2k (go-radx: unsupported) | n/a | - | - | - | - | - | - |
| pixel-encode-j2k (python: unsupported) | liver_j2k.dcm | 3.94M | - | 12.00 | - | - | - |
| pixel-encode-jpeg (go-radx: unsupported) | n/a | - | - | - | - | - | - |
| pixel-encode-jpegls | MR_small_jpeg_ls_lossless.dcm | 94.63k | - | 3.00 | 102.42k | 79.99 | 1.1x |
| pixel-encode-rle | 16-bit_mono_256x256 | 247.61k | 529.36 | 39.00 | 957.64k | 136.87 | 3.9x |
| pixel-encode-rle | 8-bit_mono_256x256 | 108.28k | 605.28 | 18.00 | 460.70k | 142.25 | 4.3x |
| pixel-encode-rle | 8-bit_rgb_256x256 | 560.23k | 350.94 | 70.00 | 1.48M | 133.03 | 2.6x |

Rows without numbers:

- `pixel-encode-htj2k` (go-radx): unsupported - OpenJPEG has no HTJ2K encoder; HTJ2K is decode-only in go-radx
- `pixel-encode-htj2k` (python): unsupported - pylibjpeg-openjpeg (OpenJPEG) decodes HTJ2K but does not encode it
- `pixel-encode-j2k` (python): unsupported - pydicom encoder rejects 1-bit pixel data (PS3.5 8.2 profile); go-radx row stands alone
- `pixel-encode-jpeg` (go-radx): unsupported - libjpeg-turbo codecs are decode-only in go-radx
- `pixel-encode-jpeg` (python): unsupported - pydicom has no JPEG Baseline/Extended/Lossless encoder (decode only)

### DIMSE: loopback C-STORE throughput (same-stack pairs)

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| dimse-cstore-loopback | 200x liver.dcm + 20x MR2_UNCI.dcm | 458.41k | 619.85 | - | 9.91M | 28.66 | 21.6x |

### HL7 v2: parse throughput

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| hl7v2-parse | adt-a01.hl7 | 10.70k | 31.68 | 449.00 | 160.11k | 2.12 | 15.0x |
| hl7v2-parse | oru-r01.hl7 | 17.64k | 33.85 | 722.00 | 255.63k | 2.34 | 14.5x |

### FHIR R5: Bundle unmarshal / marshal / validate

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| fhir-bundle-marshal | Bundle.json | 114.27k | 66.42 | 241.62 | 758.95k | 10.00 | 6.6x |
| fhir-bundle-unmarshal | Bundle.json | 509.21k | 14.91 | 1.98k | 346.00k | 21.94 | 0.7x |
| fhir-bundle-validate (python: unsupported) | Bundle.json | 51.16k | 148.36 | 789.14 | - | - | - |

Rows without numbers:

- `fhir-bundle-validate` (python): unsupported - pydantic validates during parse; covered by the unmarshal row

<!-- bench-compare:end generated tables -->

## See also

- [tools/bench-compare/README.md](../../../tools/bench-compare/README.md) - harness layout and
  developer workflow.
- The per-release Go baselines in this directory (`*-baseline.txt`), defended by the CI
  benchmark-baseline job and compared advisorily with benchstat.
