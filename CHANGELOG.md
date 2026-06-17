# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This changelog begins with the re-foundation of `go-radx`. Earlier `v0.x` tags belong to the
legacy codebase (`legacy-main`) and are not continued here.

## [Unreleased]

### Added

- DIMSE-N N-GET and N-DELETE primitives and a DIMSE-N SCP dispatch substrate (pynetdicom `send_n_get`/
  `send_n_delete` and `evt.EVT_N_*` parity, PS3.7 §10.1.2 / §10.1.6). `(*dimse.Association).NGet` reads
  attributes of a managed SOP Instance, naming the wanted attributes through the Attribute Identifier List
  (0000,1005, VR AT) command-set element and returning the SCP's typed status with the returned attribute
  data set; `(*dimse.Association).NDelete` removes a managed SOP Instance. Both reference the target by the
  Requested SOP Class/Instance UID pair the DIMSE-N services use, fail closed before any wire I/O on missing
  references or an unestablished association, and surface a Failure-category status as in-band data rather
  than a Go error. The `dimse.Server` now routes all six DIMSE-N command fields to interface-segregated
  handler hooks (`NGetHandler`, `NDeleteHandler`, `NCreateHandler`, `NSetHandler`, `NActionHandler`,
  `NEventReportHandler`); a DIMSE-N request reaching a server with no handler for that operation is refused
  with a `StatusSOPClassNotSupported` response rather than aborting, the same interface-segregation contract
  as the DIMSE-C dispatch. N-GET and N-DELETE are served end to end (SCU primitive plus SCP serve-and-respond,
  exercised by an in-process SCU-to-SCP loopback test); the N-CREATE/N-SET/N-ACTION/N-EVENT-REPORT SCP hooks
  are the foundation substrate that the deferred MPPS SCP, Storage Commitment SCP, and UPS plug into.
- FHIR server-role write side: `update` (PUT), `patch` (PATCH), and `delete` (DELETE) with their
  conditional forms, replacing the prior `501` stubs (`server/fhir_write.go`, `server/fhir_jsonpatch.go`).
  `update` does a full-resource replace with resourceType/id integrity (`400` on a mismatch), release
  validation, version bump, and `If-Match` optimistic locking (`412` on a stale version), answering `200`
  for an existing resource or `201` for update-as-create (the FHIR and HAPI default; a PUT after a DELETE
  resurrects the resource). `patch` supports JSON Patch (RFC 6902, `application/json-patch+json`): the patch
  applies to the current version, the result is re-validated through the create gate and stored as the next
  version (`415` on the wrong content type, `422` on a non-applying patch); FHIRPath Patch is out of scope.
  `delete` appends a deletion version to the store so a later read is `410 Gone`, prior versions stay
  vread-able, and the history shows the deletion; it is idempotent (`200`/`204`, never `404`, per HAPI). The
  conditional forms (`PUT`/`PATCH`/`DELETE [type]?[search]`) resolve the criteria through the configured
  `Repository`'s search (zero/one/many → create-or-no-op/apply/`412`), as selective as that repository's
  search. The `Repository` interface gains `Update` and `Delete`; the audit hook emits `fhir.update`,
  `fhir.patch`, and `fhir.delete` events. Errors are PHI-free release `OperationOutcome`s throughout.
- DICOM private-block API and a private-creator dictionary (pydicom `private_block`/`private_creators`/
  `get_private_item` and `private_dictionaries` parity). `(*dicom.DataSet).PrivateBlock(group, creator,
  create)` resolves the private-creator element (gggg,0010-00FF) whose value matches `creator` and returns a
  typed `PrivateBlock` view of its reserved data elements (gggg, bb00-bbFF, where bb is the block number =
  the low byte of the creator element, per PS3.5 section 7.8.1); with `create=true` it reserves the lowest
  free block and writes the creator element. `PrivateCreators(group)` lists the creators in a group and
  `GetPrivateItem(group, offset, creator)` fetches a single private element. The block resolver handles
  creators stored as text or as raw bytes (UN/Implicit-VR reads). `dicom.LookupPrivate(creator, group,
  offset)` and `PrivateBlock.Lookup(offset)` resolve a private tag's VR/keyword/description through the
  private-creator dictionary; the lookup mechanism is complete and the seed is minimal and attributed (the
  pydicom illustrative "ACME 3.1" creator). Vendor catalogues (Siemens/GE/Philips) are deferred to a future
  `gdcmPrivateDict.xml` generator (no such source is vendored, so vendor tag meanings are not invented).
  Private blocks round-trip through encode/decode.
- DICOM Korean, Simplified Chinese, Thai, and bare half-width katakana Specific Character Sets (0008,0005),
  completing the pydicom `pydicom.charset` mapping parity for the previously unmapped repertoires. `ISO 2022
  IR 149` (Korean, KS X 1001) and `ISO 2022 IR 58` (Simplified Chinese, GB2312) are decoded as two-byte ISO
  2022 G1 code extensions (escapes `ESC $ ) C` and `ESC $ ) A`) backed by `golang.org/x/text/encoding/korean`
  (EUC-KR) and `.../simplifiedchinese` (GBK, the 8-bit superset of GB2312); contiguous high-byte runs are
  decoded as character pairs so a delimiter byte value inside a double-byte character is never mistaken for a
  PersonName separator. `ISO_IR 166` / `ISO 2022 IR 166` (Thai, TIS 620) is decoded via `.../charmap`
  (Windows-874, a TIS-620 superset) in both the bare GR form and the `ESC - T` G1 designation. Bare `ISO_IR
  13` (JIS X 0201 half-width katakana, bytes 0xA1-0xDF mapping to U+FF61-U+FF9F) is decoded standalone via
  `.../japanese` (Shift-JIS), complementing the existing `ISO 2022 IR 13` escape form. Verified against the
  PS3.5 Annex I.2 Korean and Annex K.2 Simplified Chinese worked PersonName examples and pydicom
  `test_charset.py` Thai and katakana vectors, decoding alphabetic/ideographic/phonetic component groups.
- DICOM overlay-plane and waveform extraction (pydicom `overlay_array`/`waveform_array` parity):
  `(*dicom.DataSet).OverlayArray(group)` unpacks the 1-bit-per-pixel overlay bitmap from a 60xx repeating
  group (6000, 6002, ... 60FE) into a dense row-major boolean plane, reading bits least-significant-first
  per PS3.5 section 8.1.2; `OverlayGroups()` lists the present groups. Retired embedded-in-pixel-data
  overlays (non-zero Overlay Bit Position) are rejected with a typed error rather than guessed at.
  `(*dicom.DataSet).WaveformArray(index, byteOrder)` decodes a Waveform Sequence multiplex group into a
  `[channel][sample]` real-unit matrix, applying the PS3.3 C.10.9 per-channel scaling
  `raw*ChannelSensitivity*ChannelSensitivityCorrectionFactor + ChannelBaseline`; `WaveformGroups()` counts
  the multiplex groups. Supports 8/16-bit samples, signed/unsigned (SS/US/SB/UB/MB) interpretation, and
  big- or little-endian sample words. Verified pixel-exact against the 484x484 MR-SIEMENS overlay fixture
  (323 set bits) and with synthetic datasets carrying known bitmaps and scaled samples.
- DICOM palette-colour LUT expansion and photometric colour-space conversion (the pydicom `apply_color_lut`
  and `convert_color_space` analogues). `dicom.ApplyColorLUT` expands a `PALETTE COLOR` frame to interleaved
  RGB through the Red/Green/Blue Palette Color Lookup Tables, honouring the descriptor's first-mapped-value
  offset, 8- and 16-bit entries, the two-entries-per-word packing, and the descriptor[0]=0 means 65536 rule
  (PS3.3 C.7.9, C.7.6.3.1.5; non-segmented path). `dicom.ConvertColorSpace` converts `YBR_FULL` <-> `RGB`
  and `YBR_FULL_422` -> `RGB`/`YBR_FULL` using the PS3.3 C.7.6.3.1.2 full-range equations, handling
  `PlanarConfiguration` 0 and 1; out-of-scope pairs (`YBR_PARTIAL_*`, ICT/RCT) fail closed with a typed error.
- DICOM pixel presentation pipeline (`dicom/lut.go`): `ApplyModalityLUT`, `ApplyVOILUT`, and `ApplyWindowing`
  turn stored pixel values into presentation values per PS3.3 §C.11, the pydicom `apply_modality_lut` /
  `apply_voi_lut` / `apply_windowing` analogues. The Modality LUT applies a `ModalityLUTSequence` table when
  present (it takes precedence) and otherwise a linear `RescaleSlope`/`RescaleIntercept` rescale to measured
  units such as Hounsfield (§C.11.1.1.2). The VOI stage applies a `VOILUTSequence` table when present and
  otherwise windowing, with the `VOILUTFunction` LINEAR (the default, one-LSB-shifted form), LINEAR_EXACT,
  and SIGMOID variants (§C.11.2.1.2 / §C.11.2.1.3) and indexed multi-pair `WindowCenter`/`WindowWidth`
  selection. Table paths honour the LUT Descriptor first-mapped value and clamp out-of-range inputs to the
  end entries (§C.11.1.1.1); LUT Data is read from US or OW. A non-positive window width is rejected fail-closed
  rather than emitting NaN/Inf into a displayed image.
- Deferred (lazy) reads of large element values: `ReadFile` with `dicom.WithDeferredValues(threshold)`
  records values above the threshold as re-openable placeholders and skips the bytes, keeping memory bounded
  on large objects (the pydicom `defer_size` analogue). The value loads on first access, with the byte window
  re-validated against the source - a shrunk, swapped, or unparseable file is a typed `DeferredLoadError`,
  never a wrong value. Deferral is rejected fail-closed for a generic `io.Reader` and for Deflated Explicit
  VR LE; accessors and the write path materialise transparently, and `WriteFile` materialises before
  truncating so an in-place round-trip cannot destroy its own source (#125).
- Wave 0 quick wins: C-CANCEL honoured during C-MOVE sub-operations (prompt terminal Cancel, in-flight
  store aborted, races classified honestly); the DICOMweb daemon role mounts the full WADO-RS retrieval
  surface (study/series/metadata/frames/bulkdata, backend faults 500 never 404, BulkDataURI attribute
  locators resolve exactly); HL7 `AsOMG` typed accessor; `radx find -W` Modality Worklist queries with
  PS3.4 K.6-1 SPS match-key routing (#122).
- FHIR server versioning: a version store behind the repository seam, vread and history-instance with
  correct entry request/response, absolute fullUrl, _count, ETag W/"versionId" + Last-Modified, If-Match
  preconditions (412/404), versioned create Location on the direct and transaction paths, server-side
  $validate, and a real `radx serve fhir` subcommand replacing the fail-closed stub. Conditional create
  (If-None-Exist) fails closed with a 400 not-supported OperationOutcome on both paths, closing the
  audit-flagged client-vs-own-server duplicate-create asymmetry (#120).
- Compressed Part 10 IO: `dicom.Read`/`ReadFile`/`DecodeDataSet` accept the recognised encapsulated
  transfer syntaxes, retaining the pixel stream verbatim (aggregate-capped, structurally validated) for
  byte-identical re-write; `File.SetPixelData` closes the dataset-level transcode loop, reconciling
  PlanarConfiguration, PhotometricInterpretation, NumberOfFrames, and lossy-compression bookkeeping with
  the decoded bytes; `radx store --transcode-to` decompress-on-send works, and `radx dump`/`radx modify`
  read compressed objects (#121).
- Optional data-modification audit hook (PRD §9.5, closes the issue #113 gap): `dicom.WithAudit` on the
  PS3.15 de-identification profile emits one `dicom.AuditEvent` per successful `Deidentify` listing the applied
  (tag, action) changes — tag coordinates and action names only, no values and no UIDs — and `server.WithAudit`
  on the daemon emits one `server.AuditEvent` per durably committed server-side write (DIMSE C-STORE, STOW-RS
  stored instance, FHIR create), with an `Outcome` field separating the clean stored-and-indexed write from the
  durably-stored-but-un-indexed warning state so a durable write is never unaudited. Events never carry
  attribute values (sentinel-tested) but the server-side events do carry object-identity UIDs, which are
  PHI-adjacent under PS3.15 — the audit sink warrants archive-grade access control. The default is no hook with
  a nil-comparison disabled cost (#123).
- Comparative benchmark harness `tools/bench-compare` (PRD §11.3): a uv-pinned Python environment (pydicom
  3.0.2 + pylibjpeg plugins, pynetdicom 3.0.4, python-hl7 0.4.5, fhir.resources 8.2.0) benchmarked against
  go-radx over the same vendored fixtures - DICOM decode and per-TS pixel codecs (user-facing path on both
  sides), DIMSE loopback C-STORE, HL7 v2 parse, FHIR Bundle round-trip. Results publish to
  `docs/conformance/benchmarks/comparative.md` via `mise run bench:compare`, with a manual-dispatch CI
  workflow and an advisory (non-gating) benchstat step on the existing benchmark job (#118).

- DICOMDIR file-set support in the `dicom` package, closing the documented v1 deferral: `OpenFileSet` parses a
  DICOMDIR and resolves the Directory Record Sequence into a typed Patient/Study/Series/Instance hierarchy with
  query (`Find`/`FindValues`) and member loading; `FileSetBuilder` builds and writes a new file-set (DICOMDIR plus
  members under conformant generated File IDs) from Part 10 files or in-memory datasets. Hostile offset links
  (cycles, out-of-range, misaligned) and root-escaping Referenced File IDs fail closed with typed errors;
  cross-validated both directions against dcmtk's `dcmmkdir` (#119).
- Reference-library parity matrices under `docs/conformance/parity/` — six audited matrices comparing each
  subsystem against its reference's documented public surface (DICOM vs pydicom + pylibjpeg, DIMSE vs
  pynetdicom, HL7 v2 vs python-hl7 with a HAPI catalogue stretch, FHIR vs fhir.resources + HAPI REST,
  DICOMweb vs dicomweb-client + PS3.18, CLI vs the dcmtk suite), with file:symbol evidence, S/M/L sizing on
  every non-MET row, and an aggregate `index.md` that schedules the gaps into build waves (#116).
- Credential-leak sweep `internal/credsweep` (`mise run cred-sweep`): synthetic credential sentinels driven
  through DIMSE user-identity, DICOMweb/FHIR REST Authorization, and the daemon auth middleware, with all
  four phisweep sinks scanned and a planted-leak canary - the first automated test of PRD §9.8 (#117).
- TLS-downgrade regression tests for MLLP, the DICOMweb and FHIR REST clients, and the daemon HTTP roles,
  each with a control proving the legacy TLS 1.1 fixture genuinely bites (#117).

### Changed

- gosec joined the lint gate across all four configs; every production finding resolved with a real guard
  or a per-site `#nosec` naming the verified invariant. DICOM/DIMSE/DICOMweb binary writers now refuse
  values that would silently truncate a 16/32-bit wire length field, hostile wider-VR Image Pixel
  attributes are rejected rather than truncated into wrong frame geometry, and received DICOM files are
  written 0600 (#117).

### Documentation

- Refreshed the reference-library parity backlog (`docs/conformance/parity/`) to current `main`. Re-tallied
  every subsystem matrix from its actual status rows and reconciled each matrix summary and the `index.md`
  aggregate to that tally; the new aggregate is 414 rows / 261 MET / 44 PARTIAL / 94 NOT-MET / 15 N-A.
  Corrected the `index.md` DICOM (now 66 MET / 7 NOT-MET), DIMSE (63 / 20), HL7 floor (27 MET / 4 PARTIAL),
  and FHIR (54 MET / 8 PARTIAL) rows that had drifted from their matrices, and fixed the stale `hl7v2.md`
  (26/5 → 27/4) and `fhir.md` (53/9 → 54/8) summary counts. Updated the "reading the results" prose and the
  wave plan to mark waves 0-3 done or partly done (DICOM data-layer LUT/colour-space/overlays/charsets/
  private-blocks, DIMSE-N N-GET/N-DELETE + dispatch, FHIR update/patch/delete) while keeping the remaining
  work listed.

### Fixed

- DICOM Implicit VR LE decode of the ambiguous dictionary placeholder VRs no longer corrupts binary value
  fields. PS3.6 marks tags such as Smallest Image Pixel Value, the LUT Descriptor, LUT Data, Overlay Data,
  and Waveform Data with ambiguous VRs (`US or SS`, `US or OW`, `US or SS or OW`, `OB or OW`); under Implicit
  VR LE the reader took the dictionary VR, and these placeholders previously fell through to the text decode
  arm, materialising as `*Strings` and splitting any value byte equal to `0x5C` (the backslash value
  delimiter). The decoder now materialises `US or SS` as 16-bit integers (`*Ints`, preserved unsigned;
  signedness is reinterpreted from context such as Pixel Representation) and the word/byte placeholders
  (`US or OW`, `US or SS or OW`, `OB or OW`) as lossless raw bytes (`*Bytes`), exactly as OW/OB. A 16-bit
  numeric tag read under Implicit VR LE is now readable through `GetInt`/`GetInts`, and palette-colour LUTs
  (whose descriptor is `US or SS`, data `US or OW`) decode under Implicit VR LE rather than failing. On
  Explicit VR write the placeholders resolve to a concrete, spec-valid VR so an implicit-to-explicit
  transcode emits a valid VR, not `UN`. `US or SS` resolves by the dataset's Pixel Representation
  (0028,0103): `SS` when it is `1` (two's-complement signed), else `US` (unsigned, also the default when the
  element is absent), so a signed value such as a Modality LUT Descriptor first-mapped `-1000` or a signed
  Pixel Padding Value keeps its signed semantics rather than being re-read as `64536`; the word placeholders
  resolve to `OW` (PS3.5 §6.2, §8.1.1).

- hl7v2 MLLP and the server HTTP roles enforced no TLS floor: a caller-pinned MinVersion of 1.0/1.1 took
  effect despite the documented TLS 1.2 contract. Both now clone-and-clamp like the DIMSE AE; HTTP roles
  also gain a ReadHeaderTimeout against slowloris peers (#117).

- Fuzz targets and benchmark baselines for the FHIR R5 decode / validate / summary hot paths (F1-P).
  `fhir/r5/fuzz_test.go` adds three Go native fuzz targets — `FuzzUnmarshalResource` (registry-dispatched decode),
  `FuzzValidate` (decode-then-validate, asserting no validation issue echoes a synthetic patient-data sentinel), and
  `FuzzValidateTypedResource` (a fuzzer-shaped typed `Patient`) — each guarding the never-panic and no-PHI-leak
  contracts (PRD §9.1, §9.3) over arbitrary, truncated, wrong-typed, and deeply nested input. Each target ships a
  version-controlled seed corpus under `fhir/r5/testdata/fuzz/<FuzzName>/` and is wired into the CI fuzz job
  (`mise run fuzz`), `timeout`-wrapped so a hang fails the build. A vendored, attributed, synthetic
  (no real PHI) malformed-FHIR corpus seeds the hostile-input space (`testdata/fhir/malformed/`, with a `SOURCE.md`
  documenting each fault class) — the first contribution to the Phase 4 hostile-input gate. Decode now maps a
  truncated payload to `io.ErrUnexpectedEOF` at every decode boundary (folding the standard library's "unexpected end
  of JSON input" syntax error and the decoder path's `io.EOF`/`io.ErrUnexpectedEOF`), distinct from a mid-buffer
  syntax fault; `TestUnmarshalTruncatedYieldsUnexpectedEOF` and `TestCorpusTruncationMapsToUnexpectedEOF` assert the
  contract explicitly. Benchmarks (`fhir/r5/bench_test.go`) cover marshal/unmarshal of a 200-entry searchset Bundle,
  `Validate` over the workflow set, and the five `_summary` modes, with a benchstat-comparable baseline recorded in
  `docs/conformance/benchmarks/fhir-baseline.txt` and run once per CI build (`mise run bench`). The fuzzing-posture
  and performance-baseline sections of `docs/conformance/fhir.md` document both.
- Vendored the official HL7 FHIR R4 `4.0.1` `StructureDefinition` / `ValueSet` definition bundle
  (`fhir/internal/gen/testdata/definitions/r4`, F1-N) mirroring the R5 vendoring discipline: a checksum-pinned
  `SHA256SUMS`, a `SOURCE.md` recording the download URL, version, build (`buildId` 9346c8cc45, 2019-11-01), and CC0
  license, and the shared `.gitattributes` binary pin (no git-lfs). A standalone CI step runs `shasum -c` over the R4
  and R5 manifests so a drifted bundle is a hard error, and the generator's loader load-verifies the R4 bundle
  (`TestLoadVendoredR4Bundle` loads 148 resources, 63 datatypes, 672 value sets, 495 code systems). The
  `fhir:refresh-r4` mise task re-downloads and re-checksums it; this is the load-only precursor to R4 code generation
  (M6b), which is not yet wired.
- A merge-blocking FHIR R5 conformance gate (M6 Increment 14, the M6a acceptance gate) that runs the official
  HL7 FHIR validator (`validator_cli.jar`, hapifhir/org.hl7.fhir.core, pinned to version `6.9.9` with a recorded
  SHA-256 in `tools/versions`) over the go-radx-generated workflow set — `Patient`, `Encounter`, `ServiceRequest`,
  `ImagingStudy`, `Observation`, `DiagnosticReport`, `OperationOutcome`, `CapabilityStatement`, and a `collection`
  `Bundle` that references them. `tools/fhir-conformance/fixtures` marshals a fully-populated instance of each
  resource through the generated `MarshalJSON`, and `tools/fhir-conformance/validate.sh` validates that JSON against
  FHIR R5 `5.0.0` (zero errors required), mirroring the DICOM conformance gate's skip-locally / fail-when-`CI=true`
  structure. The gate also validates a deliberately-invalid negative fixture
  (`tools/fhir-conformance/negative/invalid-observation.json`, an `Observation` missing the required `status`) and
  fails unless the validator rejects it, proving the gate bites. A new `fhir-conformance` CI job installs a JDK,
  downloads and SHA-256-verifies the pinned jar, and runs the gate; the `conformance:fhir` mise task runs it locally.
  A curated R5 example corpus (synthetic, no PHI) is vendored under `testdata/fhir/` with provenance and CC0
  attribution (`SOURCE.md`, `LICENSE-hl7-fhir.txt`), and `fhir/r5/corpus_test.go` keeps it load-bearing in the unit
  suite (decode, structural round-trip, and go-radx's own `Validate` with no errors).
- Wired every Phase 0 gate into the CI workflow so it runs on every push and pull request: `phi-sanity`
  (the PHI-default log sweep, `internal/phisweep`), `fuzz` (a bounded smoke run over all five fuzz targets —
  `FuzzRead`/`FuzzReadPixelDataFrom` in `dicom` and `FuzzReadPDU`/`FuzzDecodeAssociateAC`/`FuzzDecodePDV` in
  `dimse/pdu` — each wrapped in `timeout` so a hang fails the build rather than being skipped),
  `benchmark-baseline` (a run-once pass over the `dicom` benchmarks so the benchmark code and its committed
  baselines cannot rot), `conformance-drift` (`tools/conformance-drift`), `docs` (`mkdocs build --strict` on a
  pinned `mkdocs`/`mkdocs-material` toolchain recorded in `tools/versions`), and `tracked-binary-hygiene`
  (fails if a compiled binary is tracked under `cmd/`). Added matching `mise` tasks (`phi-sweep`, `fuzz`,
  `bench`) and updated the cross-cutting statement's "Gate enforcement status" to list the now-wired jobs,
  which run on every push and pull request but remain advisory (the `main` ruleset is still disabled).
- A conformance-drift check (`tools/conformance-drift`, run as `go test ./tools/conformance-drift/...`
  or `mise run conformance-drift`) and an mkdocs documentation site (`mkdocs.yml`, built with
  `mise run docs:build` / `mkdocs build --strict`) so the conformance statements cannot silently
  diverge from the code. The check verifies three drift classes against the code: the countable
  presentation-context preset claims in `docs/conformance/dicom.md` match the live `dimse` preset
  counts (preset existence is read from the `dimse` source via AST, with NOT-YET-SHIPPED presets
  asserted absent), every scaffold statement (`dicomweb`, `dimse`, `convert`, `cli-server`,
  `cross-cutting`) carries its NOT-YET-SHIPPED header banner, and every top-level public package in
  the documented stability set carries its `Stability:` godoc marker; table-driven self-tests prove
  each class is detected against temp fixtures without mutating the real docs. The strict mkdocs
  build fails on navigation drift. The cross-cutting statement's "Conformance-drift methodology" section
  documents the approach. Reconciled a documentation-only drift the check surfaced: the
  `AllStorageContexts()` Storage forwarding preset was named as shipped API in `docs/conformance/dicom.md`
  and `docs/reference/dimse.md` but is not implemented, and is now marked NOT YET SHIPPED.
- Formalized the pure-Go `go test -race ./...` unit-test run as a standing required gate and authored
  the cross-cutting conformance statement's "Concurrency and race posture" section. The section
  documents why `-race` is the pure-Go unit-test gate and intentionally does not extend to the cgo
  `codecs` job (synchronous per-call transcoders with no Go goroutines) or the container-bound
  `interop` matrix legs (where wire-protocol round-trip correctness, not in-process Go races, is the
  failure mode), and adds a per-server `-race` checklist enumerating the concurrent surfaces each
  server increment (`dimse.Server`, `dicomweb.Server`, the planned MLLP and FHIR servers, and the
  `cmd/radx` daemon composition root) must exercise. It records honestly a single intermittent
  `go test -race ./...` failure observed once on CI (commit `e2c12ab`) that did not reproduce in three
  local runs and is not yet root-caused, as a known risk flagged for M8 hardening. The `lint-test`
  race-test step's comment now points to this contract; no test logic changed.
- Aggregate coverage measurement with an enforced 80% floor (PRD §11.4) in the CI `lint-test` job.
  The race-test step now runs `mise run cover:check`, which writes a single merged profile with
  `go test -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...` (scoped to the
  root module via `GOWORK=off`, so `cmd/radx` is gated by its own job) and fails the build via
  `tools/cover-check.sh` if the total drops below the floor. `-race` is preserved. The cross-cutting
  conformance statement's coverage section now documents the method and enumerates the critical paths
  (Part 10 reader, DIMSE association and DIMSE-C services, DICOMweb round-trips, cross-standard
  converters) that carry a higher 90% target.
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
- The generator model / IR stage (`fhir/internal/gen/model`): a release-agnostic element-path
  tree built from a StructureDefinition snapshot, with backbone elements fully recursed and
  `contentReference` shapes resolved by grafting the referenced element's children onto the
  referencing node (the empty-backbone fix). Each element carries its cardinality, type set,
  binding strength and value-set reference, summary/modifier flags, and choice (`[x]`) grouping.
  Pinned by a golden snapshot test over the real R5 `Observation`/`Period`/`boolean` definitions
  (`go test ./fhir/internal/gen/model -update` to regenerate). Fails closed on a missing parent
  path or a dangling `contentReference`.
- The generator planner and emitter skeleton (`fhir/internal/gen/plan`, `fhir/internal/gen/emit`):
  a planner that maps FHIR names to idiomatic Go identifiers with deterministic collision resolution,
  decides pointer-versus-value-versus-slice per element (required scalars become pointers so a present
  `false`/`0` is distinguishable from absent, repeats become slices, `decimal` maps to `fhir.Decimal`),
  and deduplicates nested backbone types by shape rather than path; and a `text/template` plus
  `go/format` emitter producing `gofmt`-clean, byte-stable Go. The planner's decisions are pinned by a
  golden planned-model snapshot. The representative datatype `r5.Period` is generated end to end to
  prove the pipeline, with `go generate ./fhir/...` reproducing it byte-for-byte and a
  `TestRegenerationByteForByte` gate (wired into the `gen:verify` task) that fails on a hand edit, so
  "generated, never hand-edited" is a verifiable property.
- The resource identity API in the root `fhir` package: `Unmarshal[T]` peeks a payload's
  `resourceType` and verifies it matches `T` before decoding, returning `ErrResourceTypeMismatch` on a
  mismatch and the zero value (the FHIR-003 checked-decode fix); `As[T]` is a checked downcast that fails
  closed on a nil interface and a typed-nil pointer; and `UnmarshalResource` dispatches by `resourceType`
  through an `init`-populated factory registry, returning `ErrUnknownResourceType` for an absent, empty,
  or unregistered type. The registry is guarded by a read-write mutex and populated only from generated
  per-release `init()` via the exported `RegisterFactory` hook, which panics on a duplicate registration.
- Resource generation in the emitter: a generated resource emits a `ResourceType()` method and a
  `MarshalJSON` that always writes the `resourceType` constant — even for a zero value, never the empty
  string (the FHIR-004 fix) — plus a per-release `registry.go` whose `init()` registers each resource's
  factory. The representative resource `r5.Flag` is generated end to end to exercise the identity API and
  the registry, pinned by emitter golden tests and the byte-for-byte regeneration gate.
- Primitive types and the `_field` extension sibling: a shared `fhir.PrimitiveElement` (an `id` plus a
  raw-JSON `extension`) referenced by every generated release primitive, and a planner/emitter that pairs
  each true primitive element with its `_field` companion. A scalar primitive gains a
  `XxxElement *fhir.PrimitiveElement` field that round-trips through ordinary struct tags; a repeating
  primitive gains a `XxxElement []*fhir.PrimitiveElement` field and a generated `MarshalJSON`/`UnmarshalJSON`
  pair that null-aligns the value array and the `_field` array, so a partially-extended repeating primitive
  (`"given":["Jane","Q"]` / `"_given":[null,{"id":"x"}]`) lines up position-for-position on both marshal and
  unmarshal. The companion is emitted only for a true primitive — never for a complex field, a `[x]` choice,
  a backbone, or a `contentReference` boundary — so `contained`, `resource`, and `OperationOutcome.issue`
  carry no spurious sibling (the FHIR-005 fix), and `decimal` maps to `fhir.Decimal`, never `float64`
  (FHIR-009). The representative datatype `r5.HumanName` is generated end to end to exercise the scalar
  siblings, the null-aligned repeating siblings, and the no-sibling-on-complex rule, pinned by planner and
  emitter golden tests, the byte-for-byte regeneration gate, and root-package round-trip and null-alignment
  unit tests.
- The FHIR choice-type (`[x]`) machinery, replacing the planner stub that emitted each choice group as a single
  first-branch-typed field. Each `value[x]`-style group is now stored as one suffixed pointer field per branch
  (`ValueQuantity *Quantity`, `ValueString *FHIRString`, ...), and the generator emits a sealed value interface
  closed by an unexported marker method (`isObservationValue`), a `Value()` getter returning the set branch, and
  one `SetValueX` setter per branch that nils every sibling before storing the new branch, so the setter API never
  populates two branches at once (the FHIR-001 fix). Because each storage field is `omitempty`, a value built
  through the setters marshals exactly one suffixed key — the prototype's unsuffixed `*any` choice field that never
  round-tripped conformant JSON is gone (the FHIR-002 fix). The storage fields are exported because faithful JSON
  requires the codec to see each suffixed key; the mutual-exclusion invariant is enforced at the setter boundary,
  and the at-most-one cardinality of a choice group is checked by `Validate` in the choice-group validation
  increment. Primitive branches box through release primitive
  wrapper types generated into `fhir/r5/primitives.go` (`FHIRString`, `FHIRBoolean`, `FHIRDecimal`, ... one per
  primitive code), since a built-in scalar cannot carry the unexported marker; `FHIRDecimal` delegates its JSON to
  `fhir.Decimal` so the lexical form survives (FHIR-009). The full R5 tree is regenerated so every choice field
  across all resources uses the real machinery, pinned by planner and emitter golden tests (`Annotation`), the
  byte-for-byte regeneration gate (now covering `primitives.go`), and choice round-trip / mutual-exclusion unit
  tests (`fhir/r5/choice_test.go`).
- Required-binding enum generation (FHIR-013): every `required`-strength value-set binding on a `code` field now
  becomes a closed Go enum — a defined string type, a const set enumerated from the bound value set, a set-membership
  validator, a strict-by-default `ParseXxx`, and a strict `UnmarshalJSON` that rejects an out-of-set code at the JSON
  boundary with `fhir.ErrUnknownCode` (wrapped with the binding name and offending token, no PHI). `ParseXxx` is always
  strict; lenient retention is the opt-in alternative threaded as `fhir.DecodeLenient` through the boundary helper
  `fhir.DecodeCode` (a value, not a global toggle, so concurrent decodes never race). The generator enumerates a value
  set from the vendored bundle when it is extensional (inlined concepts) or names a vendored `CodeSystem` with complete
  content, applying `compose.exclude` rules; a value set defined intensionally by a `compose.include.filter`, drawing
  from an un-vendored external terminology (LOINC, UCUM, IETF/ISO registries), or composed from another value set is
  emitted as a documented not-inlined plain `code` string with a godoc reason — never a silently-empty const set. To
  feed that distinction the loader now captures `ValueSetInclude.Filter` (the property/op/value triple of an
  intensional include), pinned by a golden loader test. The empty-const-set invariant is enforced by a guard test
  (`TestNoEmptyRequiredBindingEnum`) that fails the build if any required-binding enum would ship enumerable-but-empty,
  and a binding the resolver claims inlineable but that yields no codes is downgraded to the not-inlined boundary. The
  full R5 tree is regenerated so every required-binding code field across all resources uses the real enum (a repeating
  binding becomes a slice of the enum, a single one a pointer, keeping the presence and `_field`-sibling rules intact),
  pinned by an emitter golden (`bindings.go.golden`), planner binding unit tests, the byte-for-byte regeneration gate
  (now covering `bindings.go`), and root-package and `fhir/r5/enum_test.go` decode/parse regressions.
- Bundle typed builders and reference integrity (FHIR-010, FHIR-011, FHIR-015). Hand-written per-release builders
  (`r5.NewSearchSet`, `r5.NewTransaction`, `r5.NewBatch`, `r5.NewDocument`, `r5.NewMessage`, `r5.NewCollection`) make the
  FHIR `bdl-*` invariants unrepresentable-when-wrong: each produces a Bundle of exactly one type and validates its
  invariants up front, rejecting a violating bundle with `r5.ErrInvalidBundle` (matchable with `errors.Is`) naming the
  offending entry index. `total` is set only by `NewSearchSet`; search metadata lives on `SearchEntry` so it cannot reach
  another type; `entry.request` is mandatory in every transaction/batch entry and `entry.response` is unrepresentable in a
  builder-produced transaction; a document's first entry must be a `Composition` and a message's a `MessageHeader`; and
  `fullUrl` values are unique across the bundle (the FHIR-010 fix for a prototype that set `total` for every type and
  enforced nothing). A nil or typed-nil first resource is rejected, never dereferenced. Bundles are
  build-once-then-immutable with no shared mutable builder state and no mutex, replacing the prototype's mutex-papered
  mutable helper (FHIR-015). The root `fhir` package gains the release-agnostic `IssueSeverity` type and its
  `Severity*` constants plus an `IsError` predicate. Reference resolution and integrity are hand-written per-release
  helpers on the generated types: `Bundle.Resolve` resolves an intra-bundle `fullUrl` or a `#id` contained fragment;
  `DomainResource.ResolveContained` returns an aggregate error naming the offending index when a contained slot is
  malformed rather than a silent miss (the FHIR-011 fix); and `Bundle.CheckReferenceIntegrity` walks every `Reference`
  in the bundle and its contained resources, aggregating each dangling local reference and each malformed contained
  resource into one `*r5.OperationOutcome` while leaving external absolute URLs alone. `OperationOutcome.HasErrors` and
  `OperationOutcome.Error` are nil-safe and report nothing for an all-information outcome. These builders and helpers are
  the one deliberate hand-written-per-release exception to "all generated"; they live outside the generated file set, so
  the byte-for-byte regeneration gate still reproduces the generated tree. Covered by `fhir/r5/bundle_builders_test.go`
  and `fhir/r5/reference_test.go` (one test per `bdl-*` rule, the FHIR-010/011 regressions, and the dangling/malformed
  aggregation), with the conformance statement's Bundle-semantics section filled in `docs/conformance/fhir.md`.
- Structural FHIR `Validate` engine (FHIR-007, FHIR-001, FHIR-013). The release-agnostic `fhir.Validate(r Resource)
  *fhir.OperationOutcome` checks any release's resource and reports every issue it finds in one pass: `resourceType`
  integrity, required-element presence by presence rather than truthiness (a present required `false` or `0` is a
  non-nil pointer and is never reported missing — the FHIR-007 behavioural fix for the prototype's `reflect.IsZero`
  bug), choice-group mutual exclusion (it counts the non-nil suffixed storage fields and flags a `[x]` group with more
  than one set, catching a direct two-field write the mutually-exclusive setters prevent — FHIR-001), and required
  value-set binding codes (an out-of-set code retained under lenient decode is surfaced against its closed enum —
  FHIR-013). The engine is data-driven by a generated per-resource validation descriptor
  (`fhir/r5/validation_descriptors.go`) that each release registers with the root engine at init time, keyed by
  `resourceType`, so the validation path takes no call-time metadata reflection: each descriptor carries the resource's
  required elements, choice groups, and required-binding codes as typed closures over the concrete resource. The Bundle
  `bdl-*` invariants and intra-Bundle reference integrity, which the `StructureDefinition` does not express, are
  hand-written per release (`fhir/r5/validate.go`) and composed into the Bundle descriptor's extra-check hook (`total`
  only on searchset/history, `entry.search` only on a searchset, the document/message first-entry type,
  transaction/batch request and response-bundle response presence, `fullUrl` uniqueness, then
  `CheckReferenceIntegrity`). The root `fhir`
  package gains the release-agnostic `OperationOutcome`/`OutcomeIssue`/`IssueType`/`BindingIssue` types and
  `RegisterValidationDescriptor`. `Validate` never panics on malformed or partial input and never leaks PHI: every issue
  names an element, a path, a resource type, or a code, never a patient value — proven by the `fhir/r5` PHI-sentinel
  test and by wiring a FHIR exercise into the library-wide PHI sweep (`internal/phisweep`). Two fuzz targets
  (`FuzzValidateNeverPanics`/`FuzzValidateTypedNeverPanics`) drive `Validate` over arbitrary decoded and typed
  resources.
  Covered by `fhir/validate_test.go`, `fhir/r5/validate_test.go`, `fhir/internal/gen/plan/descriptor_test.go`, and
  `fhir/internal/gen/emit/descriptor_golden_test.go`; the validation-contract section is filled in
  `docs/conformance/fhir.md`. The descriptor file regenerates byte-for-byte like the rest of the tree.
- FHIR `_summary` serialization and byte-stable JSON round-trip (FHIR-012). The release-agnostic
  `fhir.MarshalSummary(r Resource, mode SummaryMode)` serializes a resource under the five FHIR `_summary` modes —
  `SummaryFull` (identity), `SummaryTrue` (the `isSummary`, mandatory, and modifier elements), `SummaryText` (the
  narrative plus mandatory elements), `SummaryData` (everything but the narrative), and `SummaryCount` (a Bundle's
  `total` plus the mandatory elements so the count view stays a valid Bundle) — by marshalling in full and then dropping
  the top-level elements the mode excludes, preserving the
  canonical element order the resource's `MarshalJSON` produced (the filter walks the encoded object key-by-key and
  re-emits the kept keys in place, never re-sorting through a map; a primitive `_field` sibling is kept exactly when its
  value key is). When a mode drops any element it sets the FHIR `SUBSETTED` tag on `meta.tag`. A nil resource returns
  the new `fhir.ErrNilResource` sentinel rather than panicking (the FHIR-012 regression fix), and a resource with no
  registered summary descriptor is returned in full rather than guessing which elements to drop. Filtering is
  data-driven by a generated per-resource summary descriptor emitted into `fhir/r5/validation_descriptors.go` and
  registered with the root serialiser at init time (`fhir.RegisterSummaryDescriptor`, `fhir.SummaryDescriptor`/
  `fhir.SummaryElement`), so the serialization path takes no call-time metadata reflection: each descriptor carries
  every top-level wire key with its `isSummary`, mandatory, modifier, narrative, and Bundle-count flags, with a choice
  (`[x]`)
  group contributing one entry per suffixed branch key. Canonical FHIR JSON now round-trips byte-for-byte: the generated
  `UnmarshalJSON` lifts each `_field` sibling into its companion field and decodes the residual keys into the value
  struct, and the matching `MarshalJSON` re-emits the same canonical order (value fields in `StructureDefinition`
  snapshot order, then the scalar `_field` siblings). Covered by `fhir/summary_test.go`, `fhir/r5/summary_test.go`,
  `fhir/internal/gen/plan/descriptor_test.go`, and `fhir/internal/gen/emit/descriptor_golden_test.go`; the serialization
  section is filled in `docs/conformance/fhir.md`. The descriptor file regenerates byte-for-byte like the rest of the
  tree.
- Migrated the `convert` package and the M2 walking-skeleton end-to-end test off the hand-written FHIR R5 resources
  onto the generated supersets (M6 Increment 12). Dropped the generator exclusion that kept `ServiceRequest`,
  `DiagnosticReport`, `ImagingStudy`, and the `Identifier`/`Reference`/`Coding`/`CodeableConcept`/`CodeableReference`
  datatypes hand-written, deleted those hand-written files and their tests, and regenerated the faithful R5 shapes
  (every `StructureDefinition` field, choice fields via the sealed setters, required-binding `code` fields as closed
  enums, base members embedded). Re-pointed the converters (`ORMToServiceRequestR5`, `SRToDiagnosticReportR5`,
  `DICOMToImagingStudyR5`) at the generated types — `status`/`intent` now carry `RequestStatus`/`RequestIntent`/
  `DiagnosticReportStatus`/`ImagingStudyStatus` enums, `DiagnosticReport.effective[x]` is set through
  `SetEffectiveDateTime`, and the `unsignedInt` counts follow the generated `*int32` type — while preserving the
  UID-to-Identifier identity rule (a DICOM UID is never a `Reference.reference` URL). Moved the hand-written
  reference-resolution / integrity helpers into `fhir/r5/reference_integrity.go` so the generated `Reference` datatype
  can own `fhir/r5/reference.go`, and removed the now-redundant hand-written validation descriptors for the three
  workflow resources (the generator emits them). `TestSkeletonEndToEnd` stays green against the generated types.

- Conformance-statement scaffolds for DICOMweb, DIMSE, cross-standard conversion, and the CLI/server surface, each
  flagged as not yet shipped; resolved the dangling DICOMweb cross-reference in the DICOM statement; annotated the
  HL7 v2 and FHIR statements where they declared scope ahead of the implementation.
- Cross-cutting conformance statement (`docs/conformance/cross-cutting.md`) covering the engineering posture shared
  across every subsystem: supply chain, interop determinism, the interop matrix, build and module layout, coverage
  targets, concurrency posture, conformance-drift methodology, and governance. It links every per-subsystem statement
  and records an honest gate-enforcement note — CI runs on every push and pull request to `main` but is currently
  advisory, since the `main` branch ruleset is disabled — flagged as a known gap against the Phase 0 definition.

### Fixed

- Polymorphic decode of interface-typed resource fields. A field typed as the abstract FHIR `Resource`
  (`Bundle.entry.resource`, `Bundle.issues`, `Bundle.entry.response.outcome`, `Parameters.parameter.resource`, and the
  `DomainResource.contained` slice) previously failed to unmarshal — the generated `UnmarshalJSON` fell back to the
  standard codec, which cannot decode a resource object into the `fhir.Resource` interface, so decoding a searchset
  `Bundle` or any resource carrying `contained` resources errored. The generator now emits decode handling that lifts
  each such field out of the raw object and routes it through `fhir.UnmarshalResource` (resourceType peek then registry
  dispatch), so the value behind the interface is the correct concrete type, recoverable with `fhir.As[T]` or a type
  switch, and a multi-resource-type Bundle round-trips. A repeating field decodes through the new
  `fhir.UnmarshalResourceSlice`, which fails the whole decode (never a partial slice) when any element's `resourceType`
  is absent, empty, or unregistered; such a discriminator surfaces a clear `ErrUnknownResourceType` rather than a panic.
  The `docs/conformance/fhir.md` round-trip section and the `fhir/r5` corpus test, which had excluded the workflow
  Bundle from full decode round-trip as a known gap, are corrected now that the gap is closed.

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
