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
- **Pixel decode per transfer syntax** - RLE Lossless, JPEG-LS (lossless + near-lossless), JPEG
  Baseline/Extended/Lossless, JPEG 2000 Lossless, and HTJ2K, each over the same vendored fixture
  on both sides. Go reuses the committed per-codec benchmarks; Python times
  `pydicom.pixels.pixel_array` with `decoding_plugin="pylibjpeg"`.
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
  rows use `-count=N` repetitions as samples. Both sides are normalized to ns/op, ops/sec, and
  MB/s by the same renderer arithmetic.
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

- Generated: 2026-06-10T09:29:22Z (mode: smoke)
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
| dicom-part10-read | MR-SIEMENS-DICOM-WithOverlays.dcm | 203.69k | 2.40k | 1.39k | 277.01k | 1.84k | 1.4x |
| dicom-part10-read | MR2_UNCI.dcm | 351.65k | 6.06k | 1.11k | 407.35k | 5.15k | 1.2x |
| dicom-part10-read | SC_rgb_expb.dcm | 35.28k | 830.35 | 395.00 | 165.69k | 188.84 | 4.7x |
| dicom-part10-read | basic-text-sr.dcm | 53.91k | 53.96 | 1.24k | 554.23k | 5.36 | 10.3x |
| dicom-part10-read | liver.dcm | 86.90k | 1.07k | 1.56k | 762.58k | 134.58 | 8.8x |
| pixel-decode-htj2k | HTJ2KLossless_08_RGB.dcm | 10.09M | - | 5.00 | 8.93M | - | 0.9x |
| pixel-decode-htj2k | HTJ2K_08_RGB.dcm | 8.94M | - | 3.00 | 8.74M | - | 1.0x |
| pixel-decode-j2k | liver_j2k.dcm | 7.92M | - | 9.00 | 6.62M | - | 0.8x |
| pixel-decode-jpeg | JPGExtended.dcm | 991.31k | - | 3.00 | 1.38M | - | 1.4x |
| pixel-decode-jpeg | JPGLosslessP14SV1_1s_1f_8b.dcm | 6.30M | - | 3.00 | 14.08M | - | 2.2x |
| pixel-decode-jpeg | SC_jpeg_no_color_transform.dcm | 327.94k | - | 3.00 | 792.85k | - | 2.4x |
| pixel-decode-jpegls | JPEGLSNearLossless_08.dcm | 5.50k | - | 3.00 | 70.58k | - | 12.8x |
| pixel-decode-jpegls | JPEGLSNearLossless_16.dcm | 428.05k | - | 3.00 | 300.60k | - | 0.7x |
| pixel-decode-jpegls | MR_small_jpeg_ls_lossless.dcm | 103.07k | - | 3.00 | 612.42k | - | 5.9x |
| pixel-decode-jpegls | SC_rgb_jls_lossy_sample.dcm | 40.88k | - | 3.00 | 324.60k | - | 7.9x |
| pixel-decode-rle | liver_rle.dcm | 462.26k | - | 6.00 | 425.78k | - | 0.9x |
| pixel-encode-htj2k (go-radx: unsupported) | n/a | - | - | - | - | - | - |
| pixel-encode-j2k (python: unsupported) | liver_j2k.dcm | 4.02M | - | 12.00 | - | - | - |
| pixel-encode-jpeg (go-radx: unsupported) | n/a | - | - | - | - | - | - |
| pixel-encode-jpegls | MR_small_jpeg_ls_lossless.dcm | 221.63k | - | 3.00 | 105.17k | 77.89 | 0.5x |
| pixel-encode-rle | 16-bit_mono_256x256 | 249.35k | 535.59 | 39.00 | 967.02k | 135.54 | 3.9x |
| pixel-encode-rle | 8-bit_mono_256x256 | 110.37k | 594.36 | 18.00 | 469.01k | 139.73 | 4.2x |
| pixel-encode-rle | 8-bit_rgb_256x256 | 667.01k | 287.82 | 70.00 | 1.46M | 134.55 | 2.2x |

Rows without numbers:

- `pixel-encode-htj2k` (go-radx): unsupported - OpenJPEG has no HTJ2K encoder; HTJ2K is decode-only in go-radx
- `pixel-encode-htj2k` (python): unsupported - pylibjpeg-openjpeg (OpenJPEG) decodes HTJ2K but does not encode it
- `pixel-encode-j2k` (python): unsupported - pydicom encoder rejects 1-bit pixel data (PS3.5 8.2 profile); go-radx row stands alone
- `pixel-encode-jpeg` (go-radx): unsupported - libjpeg-turbo codecs are decode-only in go-radx
- `pixel-encode-jpeg` (python): unsupported - pydicom has no JPEG Baseline/Extended/Lossless encoder (decode only)

### DIMSE: loopback C-STORE throughput (same-stack pairs)

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| dimse-cstore-loopback | 5x liver.dcm + 2x MR2_UNCI.dcm | 1.09M | 614.73 | - | 9.51M | 70.78 | 8.7x |

### HL7 v2: parse throughput

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| hl7v2-parse | adt-a01.hl7 | 10.08k | 33.64 | 448.00 | 303.78k | 1.12 | 30.1x |
| hl7v2-parse | oru-r01.hl7 | 17.79k | 33.55 | 721.12 | 287.11k | 2.08 | 16.1x |

### FHIR R5: Bundle unmarshal / marshal / validate

| Benchmark | Fixture | go-radx ns/op | go-radx MB/s | go allocs/op | Python ns/op | Python MB/s | Speedup (py/go) |
|---|---|---:|---:|---:|---:|---:|---:|
| fhir-bundle-marshal | Bundle.json | 109.04k | 69.61 | 241.95 | 755.90k | 10.04 | 6.9x |
| fhir-bundle-unmarshal | Bundle.json | 503.49k | 15.07 | 1.98k | 353.79k | 21.45 | 0.7x |
| fhir-bundle-validate (python: unsupported) | Bundle.json | 59.76k | 127.01 | 789.00 | - | - | - |

Rows without numbers:

- `fhir-bundle-validate` (python): unsupported - pydantic validates during parse; covered by the unmarshal row

<!-- bench-compare:end generated tables -->

## See also

- [tools/bench-compare/README.md](../../../tools/bench-compare/README.md) - harness layout and
  developer workflow.
- The per-release Go baselines in this directory (`*-baseline.txt`), defended by the CI
  benchmark-baseline job and compared advisorily with benchstat.
