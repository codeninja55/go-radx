# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This changelog begins with the re-foundation of `go-radx`. Earlier `v0.x` tags belong to the
legacy codebase (`legacy-main`) and are not continued here.

## [Unreleased]

### Added

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
