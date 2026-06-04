# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This changelog begins with the re-foundation of `go-radx`. Earlier `v0.x` tags belong to the
legacy codebase (`legacy-main`) and are not continued here.

## [Unreleased]

### Added

- A root `go.work` workspace tying the library root module and the separate `cmd/radx` CLI module,
  and a `cmd-radx` CI job that builds, vets, lints, and pin-scans (`govulncheck@v1.3.0`) the CLI
  module on every push and pull request to `main`. The job runs with `GOWORK=off` so it gates the
  module against its own `go.mod`/`go.sum` as a downstream consumer would. The existing library jobs
  are unaffected because the `./...` pattern stops at the nested module boundary. `.claude/worktrees/`
  is now git-ignored. The build-and-module-layout sections of the cross-cutting and CLI/server
  conformance statements document the layout.
- Extended the CI `interop` job into a matrix over three legs — `dimse`, `dicomweb`, and `convert` —
  so the previously-uninvoked DICOMweb STOW/WADO and convert end-to-end interop tests now run against
  the pinned Orthanc image, closing the regression window where they compiled but no job ran them.
  Each leg invokes a `mise run interop:<leg>` task (a new `interop:convert` task joins the existing
  `interop:dimse`/`interop:dicomweb`), and a normally-skipped negative-control guard
  (`TestInteropGuardBrokenDICOMWebPathFails`, gated by `RADX_INTEROP_REGRESSION_GUARD`) proves the
  DICOMweb gate bites. The interop-matrix coverage section of the cross-cutting conformance statement
  now documents the matrix.
- Structured logging via `go.uber.org/zap` in a new root `logging` package: a `NewLogger`
  constructor, context injection (`WithContext`/`FromContext`, with a no-op fallback and no
  package global), and PHI-aware field helpers (`DICOMTag`, `HL7Field`, `FHIRPath`) that render
  DICOM/HL7/FHIR concepts by name and refuse raw patient values by construction. The PHI policy
  is documented in the CLI/server conformance statement.
- Root governance docs (`SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`), a `README.md`
  declaring the pre-1.0 "everything experimental" stability posture, and a one-line stability
  marker in each top-level public package's godoc.
- DICOM data-layer fuzz targets for `dicom.Read` (Part 10 / dataset reader) and
  `dicom.ReadPixelDataFrom` (pixel reader, including the encapsulated path), plus
  version-controlled seed corpora for those and the existing DIMSE PDU targets.
- Library-wide PHI-default sanity sweep in a new `internal/phisweep` package
  (`go test ./internal/phisweep/`): it exercises representative DICOM and HL7 v2
  entry points at default verbosity over fixtures carrying synthetic PHI sentinel
  tokens and fails if any token surfaces in stdout, stderr, a returned error
  string, or the structured log. A deliberately-leaking negative case proves the
  sweep detects a planted leak in each sink. The swept sinks are documented in the
  CLI/server conformance statement.

#### DICOM benchmark baseline (Phase 0)

- Part 10 dataset decode (`BenchmarkReadFile`) and pure-Go RLE Lossless codec encode/decode
  benchmarks in the default build, plus per-transfer-syntax JPEG-family codec benchmarks
  behind the `dicom_openjpeg`, `dicom_libjpeg`, and `dicom_charls` build tags.
- Committed benchstat-comparable baselines under `docs/conformance/benchmarks/`, referenced
  from a performance section in the DICOM conformance statement.

#### DIMSE query/retrieve depth (M3) — [#36]

- C-FIND SCU iterator and SCP handler with dispatch routing and presentation-context guards.
- C-MOVE SCU iterator with sub-operation count tracking, and C-MOVE SCP that performs the
  third-AE sub-operation stores.
- C-FIND-RQ/RSP, C-MOVE-RSP, C-CANCEL-RQ command primitives and sub-operation count fields.
- C-FIND/C-MOVE/C-GET, worklist, procedure-step, and storage-commitment status tables and
  pending constants, plus query/retrieve/worklist/MPPS/storage-commitment presets.
- Per-association monotonic message-id allocator.
- C-FIND and C-MOVE interop gates against Orthanc and dcm4chee (under `//go:build interop`).

#### HL7 v2 typed-segment depth (M5) — [#34]

- Typed EVN, PV1, OBX, MSA, and ERR segments.
- `AckCode` acknowledgement-code enum.
- CWE extended to its six modelled components; XPN Degree component; XAD extended-address
  composite; typed-composite to-repetition renderers.
- Vendored HL7 v2 corpus and test harness, with a composite round-trip property test.

#### FHIR R5 generator (M6) — [#35]

- FHIR generator scaffold and `Release` constants.
- Vendored pinned HL7 FHIR R5 StructureDefinition bundle with fail-closed checksum
  verification.
- Raw StructureDefinition/ValueSet decode records, and a loader that indexes the verified
  R5 definition bundle (fail-closed on resource-less entries, unpinned required files,
  truncated bundles, and trailing garbage).

### Documentation

- Conformance-statement scaffolds for DICOMweb, DIMSE, cross-standard conversion, and the CLI/server surface, each
  flagged as not yet shipped; resolved the dangling DICOMweb cross-reference in the DICOM statement; annotated the
  HL7 v2 and FHIR statements where they declared scope ahead of the implementation.
- Cross-cutting conformance statement (`docs/conformance/cross-cutting.md`) covering the engineering posture shared
  across every subsystem: supply chain, interop determinism, the interop matrix, build and module layout, coverage
  targets, concurrency posture, conformance-drift methodology, and governance. It links every per-subsystem statement
  and records an honest gate-enforcement note — CI runs on every push and pull request to `main` but is currently
  advisory, since the `main` branch ruleset is disabled — flagged as a known gap against the Phase 0 definition.

### Removed

- Stopped tracking the compiled `cmd/radx/radx` binary and added it to `.gitignore`; the CLI
  is rebuilt on demand into an ignored path.

### Foundation (already on `main`)

- DICOM data layer: data elements, value representations, transfer syntaxes, and Part 10
  file-format read/write (M1).
- DIMSE base: A-ASSOCIATE negotiation, C-ECHO, and C-STORE over the upper-layer protocol.
- DICOMweb: WADO-RS, STOW-RS, and QIDO-RS client and server, with an Orthanc
  STOW-then-WADO interop gate.
- Thin HL7 v2 ORM parser for the walking skeleton.
- Minimal hand-written FHIR R5 resources (ImagingStudy, ServiceRequest, DiagnosticReport,
  shared datatypes).
- Cross-standard converter slice: DICOM to ImagingStudy, HL7 ORM to ServiceRequest, and
  DICOM SR to DiagnosticReport, with an end-to-end walking-skeleton proof.

### Security

- Pinned the Go toolchain to 1.26.4 to clear two standard-library advisories,
  `GO-2026-5039` (`net/textproto`) and `GO-2026-5037` (`crypto/x509`). ([#38])
- Pinned the `govulncheck` scanner to `v1.3.0` in the CI vulnerability job (was floating on
  `@latest`) so the scanner's analysis behaviour no longer drifts between runs; the vulnerability
  database stays live so a newly disclosed advisory still fails the gate. Reconciled the `cmd/radx`
  module's `go.mod` to Go 1.26.4 so every module and CI job runs the single pinned toolchain. The
  supply-chain section of the cross-cutting conformance statement now documents both pins.
- Pinned every reference tool and interop image the conformance and interop gates depend on, so a
  gate result is reproducible rather than dependent on whatever a runner pulls. The Orthanc and
  dcm4chee-arc testcontainers images are now pinned by immutable `@sha256:` digest (Orthanc was
  `:latest`; the dcm4chee-arc stack was tag-only). The `conformance` job pins `dicom3tools` to the
  exact Ubuntu archive version and `pydicom` to `3.0.2`. A new `tools/versions` manifest records
  every pin (including the FHIR `validator_cli.jar` mechanism, pinned to release `6.9.9` ahead of
  its Phase 1 gate), and a new `tools/pin-drift.sh` check — run as the `pin-drift` step of the
  `lint-test` job and via `mise run pin-drift` — fails CI if any reference floats back to an
  unpinned tag. The interop-determinism section of the cross-cutting conformance statement now
  documents the contract.

[Unreleased]: https://github.com/codeninja55/go-radx/commits/main
[#34]: https://github.com/codeninja55/go-radx/pull/34
[#35]: https://github.com/codeninja55/go-radx/pull/35
[#36]: https://github.com/codeninja55/go-radx/pull/36
[#38]: https://github.com/codeninja55/go-radx/pull/38
