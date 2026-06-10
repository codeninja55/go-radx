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

- Generated: 2026-06-10T12:01:13Z (mode: smoke)
- Host: Apple M1 / 17179869184 bytes RAM / Darwin 25.5.0 (arm64)
- Go: go version go1.26.4 darwin/arm64
- Python: 3.12.13
- Python pins: fhir.resources 8.2.0, hl7 0.4.5, numpy 2.4.6, pydicom 3.0.2, pyjpegls 1.5.1, pylibjpeg 2.1.0, pylibjpeg-libjpeg 2.4.0, pylibjpeg-openjpeg 2.5.0, pylibjpeg-rle 2.2.0, pynetdicom 3.0.4

<!-- bench-compare:end provenance -->

<!-- bench-compare:begin generated tables -->

> **PRELIMINARY** - smoke-run numbers from a loaded development host, captured only
> to prove the harness executes end-to-end. Regenerate on a quiet host via
> `mise run bench:compare` before citing any figure.

### DICOM: Part 10 decode and per-transfer-syntax pixel codecs

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| codec-decode-htj2k | HTJ2KLossless_08_RGB.dcm | 9.89M | - | 3.00 | - | - | - |
| codec-decode-htj2k | HTJ2K_08_RGB.dcm | 9.01M | - | 3.00 | - | - | - |
| codec-decode-j2k | liver_j2k.dcm | 7.98M | - | 9.00 | - | - | - |
| codec-decode-jpeg | JPGExtended.dcm | 1.04M | - | 3.00 | - | - | - |
| codec-decode-jpeg | JPGLosslessP14SV1_1s_1f_8b.dcm | 5.48M | - | 3.00 | - | - | - |
| codec-decode-jpeg | SC_jpeg_no_color_transform.dcm | 235.55k | - | 3.00 | - | - | - |
| codec-decode-jpegls | JPEGLSNearLossless_08.dcm | 6.49k | - | 3.00 | - | - | - |
| codec-decode-jpegls | JPEGLSNearLossless_16.dcm | 460.76k | - | 3.00 | - | - | - |
| codec-decode-jpegls | MR_small_jpeg_ls_lossless.dcm | 110.46k | - | 3.00 | - | - | - |
| codec-decode-jpegls | SC_rgb_jls_lossy_sample.dcm | 84.60k | - | 3.00 | - | - | - |
| codec-decode-rle | liver_rle.dcm | 746.42k | - | 6.00 | - | - | - |
| dicom-part10-read | MR-SIEMENS-DICOM-WithOverlays.dcm | 321.91k | 1.59k | 1.38k | 274.06k | 1.86k | 0.9x |
| dicom-part10-read | MR2_UNCI.dcm | 699.08k | 3.00k | 1.10k | 347.27k | 6.05k | 0.5x |
| dicom-part10-read | SC_rgb_expb.dcm | 81.69k | 382.98 | 395.00 | 167.22k | 187.10 | 2.0x |
| dicom-part10-read | basic-text-sr.dcm | 98.61k | 30.10 | 1.24k | 531.63k | 5.58 | 5.4x |
| dicom-part10-read | liver.dcm | 86.97k | 1.18k | 1.56k | 785.28k | 130.69 | 9.0x |
| pixel-decode-htj2k | HTJ2KLossless_08_RGB.dcm | 9.98M | - | 9.00 | 8.86M | - | 0.9x |
| pixel-decode-htj2k | HTJ2K_08_RGB.dcm | 9.00M | - | 9.00 | 8.71M | - | 1.0x |
| pixel-decode-j2k | liver_j2k.dcm | 7.26M | - | 17.00 | 6.61M | - | 0.9x |
| pixel-decode-jpeg | JPGExtended.dcm | 1.03M | - | 9.00 | 1.39M | - | 1.3x |
| pixel-decode-jpeg | JPGLosslessP14SV1_1s_1f_8b.dcm | 4.47M | - | 9.33 | 14.03M | - | 3.1x |
| pixel-decode-jpeg | SC_jpeg_no_color_transform.dcm | 232.29k | - | 9.00 | 796.31k | - | 3.4x |
| pixel-decode-jpegls | JPEGLSNearLossless_08.dcm | 10.49k | - | 9.00 | 75.19k | - | 7.2x |
| pixel-decode-jpegls | JPEGLSNearLossless_16.dcm | 519.83k | - | 9.00 | 328.67k | - | 0.6x |
| pixel-decode-jpegls | MR_small_jpeg_ls_lossless.dcm | 193.15k | - | 9.00 | 601.25k | - | 3.1x |
| pixel-decode-jpegls | SC_rgb_jls_lossy_sample.dcm | 89.03k | - | 9.33 | 327.93k | - | 3.7x |
| pixel-decode-rle | liver_rle.dcm | 1.04M | - | 16.67 | 438.04k | - | 0.4x |
| pixel-encode-htj2k (go-radx: unsupported) | n/a | - | - | - | - | - | - |
| pixel-encode-j2k (python: unsupported) | liver_j2k.dcm | 3.98M | - | 12.00 | - | - | - |
| pixel-encode-jpeg (go-radx: unsupported) | n/a | - | - | - | - | - | - |
| pixel-encode-jpegls | MR_small_jpeg_ls_lossless.dcm | 206.76k | - | 3.00 | 110.76k | 73.96 | 0.5x |
| pixel-encode-rle | 16-bit_mono_256x256 | 342.79k | 382.37 | 39.00 | 1.00M | 130.64 | 2.9x |
| pixel-encode-rle | 8-bit_mono_256x256 | 158.89k | 412.46 | 18.00 | 463.97k | 141.25 | 2.9x |
| pixel-encode-rle | 8-bit_rgb_256x256 | 703.35k | 279.53 | 70.00 | 1.48M | 132.66 | 2.1x |

Rows without numbers:

- `pixel-encode-htj2k` (go-radx): unsupported - OpenJPEG has no HTJ2K encoder; HTJ2K is decode-only in go-radx
- `pixel-encode-htj2k` (python): unsupported - pylibjpeg-openjpeg (OpenJPEG) decodes HTJ2K but does not encode it
- `pixel-encode-j2k` (python): unsupported - pydicom encoder rejects 1-bit pixel data (PS3.5 8.2 profile); go-radx row stands alone
- `pixel-encode-jpeg` (go-radx): unsupported - libjpeg-turbo codecs are decode-only in go-radx
- `pixel-encode-jpeg` (python): unsupported - pydicom has no JPEG Baseline/Extended/Lossless encoder (decode only)

### DIMSE: loopback C-STORE throughput (same-stack pairs)

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| dimse-cstore-loopback | 5x liver.dcm + 2x MR2_UNCI.dcm | 1.66M | 406.42 | - | 8.29M | 81.16 | 5.0x |

### HL7 v2: parse throughput

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| hl7v2-parse | adt-a01.hl7 | 10.64k | 31.87 | 448.00 | 340.88k | 0.99 | 32.1x |
| hl7v2-parse | oru-r01.hl7 | 18.67k | 31.98 | 721.12 | 269.12k | 2.22 | 14.4x |

### FHIR R5: Bundle unmarshal / marshal / validate

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| fhir-bundle-marshal | Bundle.json | 112.57k | 67.43 | 242.75 | 754.39k | 10.06 | 6.7x |
| fhir-bundle-unmarshal | Bundle.json | 518.30k | 14.64 | 1.97k | 348.23k | 21.80 | 0.7x |
| fhir-bundle-validate (python: unsupported) | Bundle.json | 51.75k | 146.68 | 789.00 | - | - | - |

Rows without numbers:

- `fhir-bundle-validate` (python): unsupported - pydantic validates during parse; covered by the unmarshal row

<!-- bench-compare:end generated tables -->

## See also

- [tools/bench-compare/README.md](../../../tools/bench-compare/README.md) - harness layout and
  developer workflow.
- The per-release Go baselines in this directory (`*-baseline.txt`), defended by the CI
  benchmark-baseline job and compared advisorily with benchstat.
