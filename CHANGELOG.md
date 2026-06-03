# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This changelog begins with the re-foundation of `go-radx`. Earlier `v0.x` tags belong to the
legacy codebase (`legacy-main`) and are not continued here.

## [Unreleased]

### Added

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

[Unreleased]: https://github.com/codeninja55/go-radx/commits/main
[#34]: https://github.com/codeninja55/go-radx/pull/34
[#35]: https://github.com/codeninja55/go-radx/pull/35
[#36]: https://github.com/codeninja55/go-radx/pull/36
[#38]: https://github.com/codeninja55/go-radx/pull/38
