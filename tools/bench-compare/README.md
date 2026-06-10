# bench-compare

Comparative benchmark harness for PRD section 11.3: go-radx versus the Python reference libraries
(pydicom + pylibjpeg plugins, pynetdicom, python-hl7, fhir.resources) over the same vendored
fixtures on both sides. Methodology, caveats, and the published tables live in
[docs/conformance/benchmarks/comparative.md](../../docs/conformance/benchmarks/comparative.md).

## Prerequisites

- `uv` on PATH (https://docs.astral.sh/uv/ - `brew install uv` or the standalone installer).
  The harness syncs its own environment from `uv.lock` on the exact CPython patch pinned in
  `.python-version` (3.12.13); nothing is installed globally. Bump the interpreter pin and the
  lock together, deliberately.
- Go toolchain via mise (repo standard).
- Optional, for the go-radx cgo codec rows: OpenJPEG, libjpeg-turbo, and CharLS resolvable via
  pkg-config (`brew install openjpeg jpeg-turbo charls` locally; CI builds them from source).
  Without them those rows are marked `unavailable`; everything else still runs.

## Run

```bash
mise run bench:compare                   # full publishing run
mise run bench:compare -- --mode smoke   # tiny-N end-to-end harness validation
```

Or directly from this directory:

```bash
uv sync --frozen
uv run python -m benchcompare --mode full
```

Outputs:

- `results/comparative.json` - machine-readable raw samples + derived-metric inputs (gitignored).
- `results/comparative.md` - standalone rendering of the tables (gitignored; CI artifact).
- `docs/conformance/benchmarks/comparative.md` - the committed doc; only its marked generated
  block is rewritten, so a publishing run is: quiet host, `mise run bench:compare`, commit.

## Layout

- `benchcompare/` - Python package: per-area runners (`bench_dicom`, `bench_dimse`, `bench_hl7`,
  `bench_fhir`), the go-side driver (`gorunner`), the renderer (`render`), and the orchestrator
  (`__main__`).
- `gobench/` - standalone Go module (built with `GOWORK=off`, `replace` to the in-tree library)
  covering the areas the committed `go test -bench` suites do not: DIMSE loopback C-STORE, HL7 v2
  parse, FHIR Bundle over the shared file fixture. The DICOM rows REUSE the committed benchmarks
  in `./dicom/` via `go test -bench`.
- `pyproject.toml` / `uv.lock` - exact Python pins; bump only via a deliberate `uv lock --upgrade`.

## What it measures

One sample times `ops` back-to-back operations; each series is warmup + N measured samples and the
median is published. Both sides are normalized (ns/op, ops/sec, MB/s) by the same renderer code.
Operations that one side cannot perform (JPEG-family encode, HTJ2K encode, a separate FHIR
validate pass in pydantic) are recorded as `unsupported` rows, never silently dropped.

Lint the Python side with `uv run ruff check benchcompare`.
