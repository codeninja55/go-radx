# FHIR generator (M6a / M6b) implementation plan

> **For agentic workers:** REQUIRED: Use agentic-dev:subagent-driven-development (if subagents available) or
> agentic-dev:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the FHIR resource **code generator** and the generated packages it produces (PRD §13 milestones
**M6a** — generator + R5 core, and **M6b** — full R4). The generator ingests the official HL7 FHIR
`StructureDefinition` bundles for R5 5.0.0 and R4 4.0.1 and emits type-safe, never-hand-edited Go for the full resource
set (every resource and backbone element — about 158 R5 resources and the R4 set), the complex datatypes, the
choice-type accessor machinery, the `required`-binding enums, the primitive-extension (`_field`) siblings, the
`resourceType` registry, cardinality/required `Validate`, the per-type `OperationOutcome`, the `Bundle` builders and
reference-integrity, and `_summary` filtering. It **subsumes** the M2 Increment 11 hand-written
`ServiceRequest`/`DiagnosticReport`/`ImagingStudy`: the generator must produce those (and the rest), and this plan
migrates `convert` and the skeleton off the hand-written minimal resources onto the generated ones. The output conforms
exactly to the committed public API in `docs/reference/fhir.md` and the scope in `docs/conformance/fhir.md`; the
workflow set is gated against the official HL7 FHIR validator in CI.

**Architecture:** The generator is a **build-time tool** under `fhir/internal/gen`, invoked by `go generate`, that
writes committed `.go` files into `fhir/r4` and `fhir/r5`. Consumers import plain generated Go — there is no runtime
reflection, no codegen at `go build`, and no generator in the runtime dependency graph (fhir.md "The code generator").
The package layout is the one the reference doc commits (fhir.md "Generated package layout"): the root `fhir` package
holds release-agnostic machinery (`Resource`, `Unmarshal[T]`/`As[T]`/`UnmarshalResource`, the registry, `Decimal`,
`PrimitiveElement`, `SummaryMode`/`MarshalSummary`, `IssueSeverity`, the sentinel errors, and the `Release` constants);
`fhir/r4` and `fhir/r5` are genuinely distinct type spaces (the FHIR-014 fix), each holding generated resources,
backbone elements, complex datatypes, required-binding enums, the per-release `OperationOutcome` resource, and the
`init`-populated `resourceType→factory` registry entries. The generator is staged as a clean pipeline with
single-responsibility boundaries (PRD §8.2): a **loader** (read + checksum-verify the pinned definition bundles) → a
**model/IR** (the explicit element-path tree the spec demands) → a **planner** (decide Go names, types, pointer-vs-slice,
choice groups, enums, `_field` siblings per element) → an **emitter** (templates → `gofmt`/`goimports`-stable Go). Each
stage is gated by golden tests; the generated output is gated by "it compiles", "it round-trips known HL7 example JSON",
and finally "the workflow set passes the official HL7 FHIR validator".

**Tech Stack:** Go 1.26.x, module `github.com/codeninja55/go-radx`; standard library for the generator
(`encoding/json`, `text/template`, `go/format`, `archive/zip`, `crypto/sha256`, `embed` or a vendored definitions
directory, `io/fs`, `sort`); `go.uber.org/zap` is **not** used in the generator (a build tool logs to stderr plainly).
The generated runtime code depends only on the standard library plus the existing `github.com/codeninja55/go-radx/dicom`
(for the shared `Decimal`). The official HL7 FHIR validator runs in CI as the conformance gate (a Java jar or the
existing M0 validator container — confirm the M0 wiring; see Open questions). No CGo anywhere in this milestone.

---

## How to use this plan

Read this section once before starting; it states the conventions every task follows.

**Test-first, always.** Each task is a strict TDD cycle: write the failing test, run it and confirm it fails for the
right reason, write the minimal implementation, run it and confirm it passes, then commit. Do not write implementation
before its test. See `agentic-dev:test-driven-development`. For a code **generator**, "test-first" has a specific shape:
the generator's stages (loader, model, planner, emitter) are tested with ordinary unit tests on small hand-built inputs;
the *generated output* is tested with **golden-file tests** (a committed expected `.go` fragment the emitter must
reproduce byte-for-byte) and with **behavioural tests** that compile and exercise the generated code (round-trip,
`Validate`, choice accessors). A generator increment is not done until both its golden test and the relevant behavioural
test are green.

**Canonical names are mandatory.** Use the exact Go identifiers fixed in `docs/reference/fhir.md`,
`docs/conformance/fhir.md`, and `UBIQUITOUS_LANGUAGE.md`: `Resource`, `ResourceType`, `Unmarshal`, `As`,
`UnmarshalResource`, `Decimal`, `ParseDecimal`, `PrimitiveElement`, `Element`, `Reference`, `Identifier`, `Coding`,
`CodeableConcept`, `CodeableReference`, `Quantity`, `HumanName`, `Period`, `Bundle`, `BundleType`, `NewSearchSet`,
`NewTransaction`, `NewDocument`, `NewMessage`, `OperationOutcome`, `OperationOutcomeIssue`, `IssueSeverity`,
`SummaryMode`, `MarshalSummary`, `Validate`, `Release` (`R4`/`R5`), `ErrResourceTypeMismatch`, `ErrUnknownResourceType`,
`ErrUnknownCode`, `ErrNilResource`. Required-binding enums follow the reference-doc examples: the defined string type is
the value-set's Go name (`AdministrativeGender`, `BundleType`, `IssueSeverity`), constants are `<Prefix><Code>`
(`GenderMale`, `BundleSearchSet`, `SeverityError`), and the parser is `Parse<TypeName>`. The primitive-extension sibling
field is `XxxElement` with a `json:"_xxx,omitempty"` tag (the underscore lives only on the wire, never in a Go
identifier — FHIR-005). Never reintroduce the prototype's broken model: no `package resources` inside a `types`
directory (FHIR-014), no unsuffixed `*any` choice fields (FHIR-002), no `_field` siblings on non-primitives (FHIR-005),
no `decimal`→`float64` mapping (FHIR-009), no empty backbone structs (FHIR-006).

**The generated code is never hand-edited.** Every byte under `fhir/r4` and `fhir/r5` is produced by the generator.
`go generate ./fhir/...` must reproduce the committed output **byte-for-byte** (fhir.md "The code generator"); a
regeneration that changes a committed file is either a generator change (commit the generator and the regenerated output
together, generator first) or a definition-bundle change (commit the bundle bump separately). A reproducibility test in
CI regenerates into a temp dir and diffs against the committed tree.

**The definition bundles are pinned and checksum-verified.** Generation is reproducible only from a fixed input. The
StructureDefinition bundles are vendored and committed with a recorded SHA-256; the loader verifies the checksum before
parsing and refuses to run against an unpinned or mismatched bundle (Increment 1). A download task exists for *refreshing*
the pinned bundle, never for generate-time fetching (see Open questions, resolved).

**Build the model from full element paths, then recurse.** The generator builds an explicit element tree from the
`StructureDefinition` snapshot's full element paths (`Observation.component.referenceRange.low`) and recurses from that
tree, so nested backbone elements are fully populated. Empty backbone structs are the FHIR-006 defect; a golden test
guards a known-deep backbone (`Observation.component.referenceRange`) against it.

**Diagnostics carry no PHI.** This rule applies to the *generated runtime* code, not the build-time generator (which
sees only the spec, never patient data). `Validate`, `CheckReferenceIntegrity`, `OperationOutcome.Error`, and
`ErrUnknownCode` name the element path, the binding name, and the offending *code token* (a code is value-set membership,
not PHI), never a patient value (fhir.md "OperationOutcome and the error model"; PRD §8.2, §9.1).

**Fail closed on unknown codes.** Strict decode of a `required`-binding enum rejects an out-of-set code with
`ErrUnknownCode`; lenient decode (opt-in) retains it and surfaces it via `Validate` (fhir.md "Unknown-code policy"; PRD
§9.2). `ParseXxx` always applies the strict rule. The safe default is fail-closed.

**Commit conventionally and often.** Each commit message follows `<type>(<scope>): <description>`. The generator source
and its unit tests commit together; the *generated output* commits separately (it is generated data, per the project
Atomic Commit Strategy: "Source code — yes; Generated data, fixtures — separate"). Vendored definition bundles commit
separately again. Typical scopes: `fhir/gen` (the generator), `fhir/r5` / `fhir/r4` (regenerated output — committed as
`chore(fhir/r5): regenerate ...`), `fhir` (the root machinery).

**Codex defect traceability.** The FHIR model defects this milestone closes are listed below; each is gated by a named
regression test in the increment that closes it. A defect is not closed until its test passes.

| Defect | What the prototype did wrong | Where M6 fixes it |
|--------|------------------------------|-------------------|
| FHIR-001 | choice types could have two branches set at once | Increment 7 (sealed value interface + `SetValueX` clears siblings) |
| FHIR-002 | unsuffixed `*any` choice fields that never round-tripped conformant JSON | Increment 7 (suffixed storage fields + accessors) |
| FHIR-003 | `UnmarshalResource[T]` never compared `resourceType` against `T` | Increment 4 (`Unmarshal[T]` peeks + checks `resourceType`) |
| FHIR-004 | zero-value resource serialised `"resourceType":""` | Increment 6 (`MarshalJSON` always emits the type constant) |
| FHIR-005 | `_field` siblings emitted on complex/backbone elements | Increment 5 (siblings only on true primitives) |
| FHIR-006 | empty backbone structs (no recursion down element paths) | Increment 2 (element-path tree) + Increment 6 golden test |
| FHIR-007 | required `false`/`0` indistinguishable from absent | Increment 5/8 (required scalars as pointers; presence-not-truthiness) |
| FHIR-009 | `decimal` mapped to `float64`, losing precision | Increment 5 (`decimal`→`fhir.Decimal`) |
| FHIR-010 | `Bundle` incremented `total` for every type; no per-type invariants | Increment 9 (typed builders enforce `bdl-*`) |
| FHIR-011 | `Contained` silently skipped a malformed contained resource | Increment 10 (returns an aggregate error) |
| FHIR-012 | `MarshalSummary(nil, ...)` panicked | Increment 11 (`ErrNilResource`) |
| FHIR-013 | `required` bindings stored as metadata only, never enforced | Increment 8 (closed Go enums + validating decode) |
| FHIR-014 | `package resources` in a `types` dir → two incompatible type sets | Increment 0/3 (distinct `r4`/`r5` packages) |
| FHIR-015 | mutable bundle helper with a mutex papering over a concurrency bug | Increment 9 (immutable-after-build; documented single-builder rule) |

(There is no FHIR-008 in the audit; the numbering skips it.)

**Salvage map (port-with-fixes vs rewrite).** The read-only prototype lives at `/Users/codeninja/vcs/go-radx/fhir`. Per
PRD §12 the FHIR generator is a **REWRITE** (its model is structurally wrong) but the **regeneration approach is kept**.
Concretely:

| Verdict | Prototype source | M6 increment |
|---------|------------------|--------------|
| **REWRITE (model)** | `fhir/scripts/gen/model/definition.go` (the IR) | Increment 2 |
| **REWRITE (planner+emitter)** | `fhir/scripts/gen/codegen/{builder,generator}.go` (emits the broken shapes) | Increments 3–8 |
| **PORT-WITH-FIXES (loader)** | `fhir/scripts/gen/parser/structdef.go` (the StructureDefinition JSON reader) | Increment 1 |
| **PORT-WITH-FIXES (definition handling)** | `fhir/scripts/download_r5_schemas.sh` + the `fhir_schemas/` layout | Increment 1 (vendor + checksum, not download-at-gen) |
| **KEEP (hand-written core)** | the M2 `fhir/resource.go`, `fhir/decimal.go`, `fhir/r5/datatypes.go` shapes | matched/superseded by the generator |

Port the *StructureDefinition JSON reading and the regeneration workflow*; rewrite the *model and the emitted Go* so the
output matches `docs/reference/fhir.md`, not the prototype's `package resources` / `*any`-choice / `float64`-decimal
shapes.

**The reference doc IS the spec.** Read the cited `docs/reference/fhir.md` section before implementing each increment.
The scope gate is `docs/conformance/fhir.md` ("In scope (v1)" and "Deferred"). Where this plan shows a signature, it is
copied from the reference doc; if you find a genuine gap, stop and surface it (see "Open questions").

---

## Milestone naming (M6 vs M6a/M6b)

The orchestrating task calls this "M6 — the FHIR code generator". The PRD §13 splits the same work into **M6a** (the
rewritten generator plus the R5 core, gated by the FHIR validator) and **M6b** (full R4 4.0.1 generation and
R4-conformance for the workflow set). This plan covers both: Increments 0–14 are M6a (generator + R5, R5-validator
gated) and Increment 15 is M6b (run the same generator over the R4 bundle, R4-validator gate). Splitting the validator
gate per release is the PRD's risk mitigation ("FHIR generator correctness is the highest-leverage single component …
split into M6a/M6b"). Throughout, "M6" means the whole milestone; "M6a"/"M6b" name the release halves.

---

## Increment overview (dependency-ordered)

The generator is built bottom-up: you cannot emit code until you have a model, and you cannot build a model until you
can load and verify the definitions. The runtime *machinery* the generated code leans on (the registry, primitives,
`Decimal`) is mostly already present from M2 and is extended rather than rebuilt. Within R5 the order is loader → model
→ scaffolding/emitter → datatypes → primitives/`_field` → resources/backbones → choice groups → enums → validation →
bundles → reference-integrity → summary → migration → R5 validator gate; then R4 reruns the whole pipeline.

- **Increment 0 — Generator package scaffold + R5 definition bundle vendoring.** Create `fhir/internal/gen` skeleton,
  the `go generate` directive, the `fhir/r4`/`fhir/r5` package shells, extend the root `fhir` package with the `Release`
  constants, vendor the pinned R5 `definitions.json.zip` (the `profiles-types.json` / `profiles-resources.json` /
  `valuesets.json` bundle) with a recorded SHA-256, and wire the `gen:*` / `test:fhir-gen` mise tasks. No emitting logic
  yet. *Outlined.*
- **Increment 1 — Definition loader (checksum-verified).** Read the vendored bundle, verify its SHA-256, unzip and
  parse the StructureDefinition + ValueSet + CodeSystem entries into raw decoded records, reject an unpinned/mismatched
  bundle. **Fully expanded into bite-sized TDD tasks below.** Port-with-fixes from the prototype's `parser`. Gate: unit
  + lint; loads the real R5 bundle and reports the expected resource/datatype/valueset counts. *Expanded.*
- **Increment 2 — The model / IR (element-path tree).** Build the explicit `StructureDefinition` model and the
  element-path tree (`Observation` → `component` → `referenceRange` → `low`), classify each `StructureDefinition` by
  `kind` (primitive-type, complex-type, resource), and attach element metadata (min/max, types, binding+strength,
  `isSummary`, `isModifier`, fixed/pattern). Closes the structural half of FHIR-006. *Outlined.*
- **Increment 3 — Planner + emitter skeleton (one trivial datatype end-to-end).** The name mapper (FHIR path → Go
  identifier, collision rules from `UBIQUITOUS_LANGUAGE.md`), the type planner (FHIR type → Go type, pointer-vs-slice
  from cardinality), the `text/template` + `go/format` emitter, and the golden-test harness — proven end-to-end by
  emitting one simple complex datatype (`Period`) into `fhir/r5` and asserting it matches a committed golden file and
  compiles. *Outlined.*
- **Increment 4 — Resource identity machinery + registry generation.** Extend the root `fhir` package's
  `Unmarshal[T]`/`As[T]`/`UnmarshalResource` (the FHIR-003 `resourceType` check) and generate each release package's
  `init`-populated `resourceType→factory` registry. *Outlined.*
- **Increment 5 — Primitive types + the `_field` extension sibling.** The primitive→Go-scalar mapping (with `decimal`→
  `fhir.Decimal`, FHIR-009), required scalars as pointers (FHIR-007), the shared `fhir.PrimitiveElement`, the scalar and
  repeating `_field` sibling generation with null-aligned marshalling, and the no-sibling-on-non-primitive rule
  (FHIR-005). *Outlined.*
- **Increment 6 — Complex datatypes + full resource & backbone generation.** Generate every complex datatype, every
  resource, and every nested backbone element for R5 from the element-path tree, with `ResourceType()` constants,
  always-emit-`resourceType` `MarshalJSON` (FHIR-004), and canonical element ordering. This is the bulk-generation
  increment (about 158 resources). Golden test guards the FHIR-006 deep-backbone case. *Outlined.*
- **Increment 7 — Choice-type accessor generation.** For each `value[x]`-style group: the sealed value interface, the
  suffixed storage fields, the `Value()` getter, and one `SetValueX` setter per branch that clears the siblings
  (FHIR-001/002), plus custom `MarshalJSON`/`UnmarshalJSON` that emit/parse exactly one suffixed key, and the primitive
  wrapper types (`FHIRString`, `FHIRBoolean`, …) for primitive branches. *Outlined.*
- **Increment 8 — Required-binding enum generation.** For each `required`-strength binding: the defined string type, the
  const set, `ParseXxx`, and a validating `UnmarshalJSON` honouring the strict/lenient decode policy (FHIR-013); weaker
  strengths stay `string`. *Outlined.*
- **Increment 9 — Bundle generation + typed builders.** The generated `Bundle` resource plus the hand-written
  release builders (`NewSearchSet`/`NewTransaction`/`NewDocument`/`NewMessage`) that enforce the `bdl-*` invariants
  (FHIR-010), and the documented build-once-then-immutable rule (FHIR-015). *Outlined.*
- **Increment 10 — Reference integrity + `OperationOutcome`.** Generate the per-release `OperationOutcome` resource;
  add `IssueSeverity` to the root; implement `Bundle.Resolve`, `DomainResource.Contained` (aggregate error, FHIR-011),
  and `Bundle.CheckReferenceIntegrity`. *Outlined.*
- **Increment 11 — Cardinality / required `Validate` + summary mode.** The `Validate(r Resource) *OperationOutcome`
  engine (required presence by presence-not-truthiness, choice-group cardinality, required-binding codes, `bdl-*`), and
  `SummaryMode`/`MarshalSummary` driven by the generator-recorded `isSummary` flag (FHIR-012 nil-guard). *Outlined.*
- **Increment 12 — Migrate `convert` and the skeleton off the hand-written resources.** Replace the M2 Increment 11
  hand-written `ServiceRequest`/`DiagnosticReport`/`ImagingStudy` (and the hand-written `r5/datatypes.go`) with the
  generated equivalents, fix `convert`'s field accesses to the generated shapes, and keep the M2 end-to-end skeleton
  test green. *Outlined.*
- **Increment 13 — Reproducibility + full-set compile gate.** A CI test that regenerates R5 into a temp dir and diffs
  byte-for-byte against the committed tree, plus a compile-test of the entire generated `fhir/r5` package and a
  round-trip test over the vendored HL7 R5 example JSON corpus. *Outlined.*
- **Increment 14 — R5 conformance gate (HL7 FHIR validator).** Run the workflow set (`Patient`, `Encounter`,
  `ServiceRequest`, `ImagingStudy`, `DiagnosticReport`, `Observation`, `Bundle`, `OperationOutcome`) through the
  official HL7 FHIR validator in CI; this is the **M6a acceptance gate**. *Outlined.*
- **Increment 15 — M6b: R4 4.0.1 generation + R4 conformance gate.** Vendor the pinned R4 bundle, run the same
  generator over it into `fhir/r4`, reconcile R4↔R5 datatype differences (for example `ServiceRequest.code` is a
  `CodeableConcept` in R4, a `CodeableReference` in R5), and pass the workflow set through the validator at R4. This is
  the **M6b acceptance gate**. *Outlined.*

Increments 2 through 15 are outlined here (goal, files, key tests, reference-doc section, ports-vs-rewrite note, and
verification gate) and are expanded into bite-sized TDD tasks when reached, exactly as the M2 plan fully expanded its
Increment 1 and outlined the rest.

---

## Increment 0 — Generator package scaffold + R5 definition bundle vendoring

**Scope:** Stand up the generator package, the generated-package shells, the root `Release` constants, and the pinned R5
definition bundle, so every later increment has a place to write and a verified input to read. No emitting logic.

**Reference-doc section:** fhir.md "Generated package layout", "The code generator"; conformance/fhir.md "Conformance
metadata" (the `Release` type, the `4.0.1`/`5.0.0` versions, "Generated from the official HL7 `StructureDefinition`
bundles"); PRD §12 (regeneration approach kept).

**Ports vs rewrite:** Port the *definition-bundle layout* idea from the prototype's `fhir_schemas/` directory and
`download_r5_schemas.sh`, but **vendor and checksum** the bundle rather than fetch at generate-time, and rewrite the
download script into a *refresh-only* mise task (Open question 1, resolved: vendor + commit + pin).

**Files:**
- Create: `fhir/internal/gen/doc.go` (one-paragraph package comment: build-time generator, never in the runtime graph),
  `fhir/internal/gen/gen.go` (a `main`-less library entry `Generate(cfg Config) error` stub returning
  `errors.New("not implemented")` so the package compiles), `fhir/internal/gen/cmd/fhir-gen/main.go` (the thin CLI that
  calls `gen.Generate`).
- Create: `fhir/gen.go` (the `//go:generate go run ./internal/gen/cmd/fhir-gen -release r5` directive and nothing else).
- Modify: `fhir/r5/doc.go` (note it is generated; the M2 hand-written resources still live here until Increment 12),
  create `fhir/r4/doc.go`.
- Modify: `fhir/release.go` (new) — the `Release` type and `R4`/`R5` constants from conformance/fhir.md.
- Create: `fhir/internal/gen/testdata/definitions/r5/` holding the vendored bundle (`profiles-types.json`,
  `profiles-resources.json`, `valuesets.json`) **plus** `SHA256SUMS` recording each file's checksum, and a `SOURCE.md`
  recording the exact download URL (`https://hl7.org/fhir/R5/definitions.json.zip`) and date.
- Modify: `mise.toml` — add `[tasks."gen:fhir-r5"]` (`go generate ./fhir/...` filtered to R5), `[tasks."gen:fhir"]`
  (all releases), `[tasks."gen:verify"]` (regenerate to temp + diff — wired in Increment 13), `[tasks."fhir:refresh-r5"]`
  (download + re-checksum the bundle, refresh-only), and `[tasks."test:fhir-gen"]` (`go test -race ./fhir/internal/...`).
- Modify: `Makefile` — mirror the gen/test targets.

**Key tests:** `go build ./fhir/...` is green; `go test ./fhir/...` reports the new `internal/gen` package compiles;
`fhir.R5 == "5.0.0"` and `fhir.R4 == "4.0.1"`; a tiny loader-independent test asserts the vendored bundle files exist and
their recorded SHA-256 in `SHA256SUMS` matches the on-disk bytes (so the pin is real before Increment 1 consumes it).

**Verification gate:** `go build ./...` green; `mise run test:fhir-gen` runs; the checksum-pin test passes. Commit the
generator scaffold, the vendored bundle (separately, `chore(fhir/gen): vendor pinned R5 StructureDefinition bundle`), and
the mise/make config as separate atomic commits.

---

## Increment 1 — Definition loader (checksum-verified, fully expanded)

**Scope:** The bottom of the generator pipeline: read the vendored R5 bundle, verify its SHA-256 against the committed
`SHA256SUMS`, unzip the entries, and decode each `Bundle.entry.resource` into a raw `StructureDefinition` / `ValueSet` /
`CodeSystem` record keyed by canonical URL and by name. This increment loads and indexes; it does **not** build the
element tree (Increment 2) or emit code. It closes the prototype's most basic gap: the prototype trusted whatever JSON
was on disk and fetched at generate time, so a corrupted or drifted bundle silently produced wrong code. Here the loader
**fails closed** on a checksum mismatch, a missing required file, or a malformed entry.

**Reference-doc section:** fhir.md "The code generator" (reproducible from pinned, checksum-verified inputs);
conformance/fhir.md "Conformance basis"; PRD §9.2 (fail-closed), §9.3 (validate external input — the bundle is external
data the build tool ingests). The prototype source to port-with-fixes is
`/Users/codeninja/vcs/go-radx/fhir/scripts/gen/parser/structdef.go` (the StructureDefinition JSON reader) and
`model/definition.go` (the raw field shapes).

**Porting note — representation changes from the prototype.** The prototype read a single `profiles-resources.json`
passed by `-input` flag with no integrity check and parsed it loosely. This loader: (a) reads from the **vendored**
directory (Increment 0), never the network; (b) verifies SHA-256 before parsing (any mismatch is a hard error naming the
file, never a "regenerate anyway"); (c) loads all three bundle files (`profiles-types.json`, `profiles-resources.json`,
`valuesets.json`) because complex datatypes, resources, and value sets live in separate bundles; (d) indexes by both
canonical `url` and `name` so the model layer can resolve type references and `valueSet` bindings; (e) keeps the raw
decoded records as plain structs mirroring the FHIR JSON, with the element-tree construction deferred to Increment 2
(single-responsibility: load vs model). The loader returns a typed `*LoadError` naming the file and the failure.

**File structure:**
- `fhir/internal/gen/loader/loader.go` + `loader_test.go` — `Load(dir string) (*Bundle, error)`, the checksum verifier,
  the unzip+decode.
- `fhir/internal/gen/loader/records.go` + `records_test.go` — the raw decoded record structs (`StructureDefinition`,
  `ElementDefinition`, `ValueSet`, `CodeSystem`) mirroring the FHIR JSON, with the fields the model layer needs.
- `fhir/internal/gen/loader/errors.go` — the `*LoadError` typed error.

---

### Task 1.1: Checksum verification of the vendored bundle

**Files:**
- Create: `fhir/internal/gen/loader/loader.go`, `fhir/internal/gen/loader/errors.go`
- Test: `fhir/internal/gen/loader/loader_test.go`

- [ ] **Step 1: Write the failing test.** A test writes two tiny files plus a `SHA256SUMS` into a temp dir; calling
  `verifyChecksums(dir)` succeeds. Then it corrupts one file and asserts `verifyChecksums` returns a `*LoadError` whose
  message names the offending file and does **not** contain the file bytes (no leaking arbitrary content). A missing file
  listed in `SHA256SUMS` is also an error.

```go
func TestVerifyChecksumsRejectsMismatch(t *testing.T) {
    dir := t.TempDir()
    writeFile(t, dir, "a.json", []byte(`{"x":1}`))
    writeSums(t, dir, map[string]string{"a.json": sha256Hex([]byte(`{"x":1}`))})
    if err := verifyChecksums(dir); err != nil {
        t.Fatalf("verifyChecksums on matching bundle: %v", err)
    }
    writeFile(t, dir, "a.json", []byte(`{"x":2}`)) // drift
    err := verifyChecksums(dir)
    if err == nil {
        t.Fatal("verifyChecksums should reject a drifted file")
    }
    var le *LoadError
    if !errors.As(err, &le) {
        t.Fatalf("error = %T, want *LoadError", err)
    }
    if !strings.Contains(le.Error(), "a.json") {
        t.Errorf("error %q should name the offending file", le.Error())
    }
}
```

- [ ] **Step 2: Run test to verify it fails.** `go test ./fhir/internal/gen/loader/ -run TestVerifyChecksums -v` —
  FAIL: `undefined: verifyChecksums`, `LoadError`.
- [ ] **Step 3: Write minimal implementation.** `verifyChecksums` reads `SHA256SUMS`, computes `sha256.Sum256` per
  listed file, compares hex, returns `*LoadError{File, Detail}` on the first mismatch or missing file. `LoadError.Error()`
  is `"fhir/gen: " + File + ": " + Detail`.
- [ ] **Step 4: Run test to verify it passes.**
- [ ] **Step 5: Commit.** `feat(fhir/gen): verify vendored definition bundle checksums (fail-closed)`

---

### Task 1.2: Raw record structs decoded from FHIR JSON

**Files:**
- Create: `fhir/internal/gen/loader/records.go`
- Test: `fhir/internal/gen/loader/records_test.go`

- [ ] **Step 1: Write the failing test.** Decode a hand-written minimal `StructureDefinition` JSON (a `Patient` stub with
  `kind:"resource"`, a `snapshot` of two `ElementDefinition`s — `Patient.id` min 0 max 1 type `id`, and
  `Patient.gender` min 0 max 1 type `code` with a `required` binding to `administrative-gender`) into the `StructureDefinition`
  record and assert the parsed fields: `Kind == "resource"`, the two element `Path`s, the second element's
  `Binding.Strength == "required"` and its `ValueSet` URL. Decode a minimal `ValueSet` and assert its `compose.include`
  concepts are reachable.

- [ ] **Step 2: Run test to verify it fails.** `undefined: StructureDefinition` / `ElementDefinition` / `ValueSet`.
- [ ] **Step 3: Write minimal implementation.** Plain structs with `json` tags mirroring the FHIR JSON: `StructureDefinition`
  (`ResourceType`, `URL`, `Name`, `Kind`, `Abstract`, `Type`, `BaseDefinition`, `Snapshot`), `Snapshot.Element
  []ElementDefinition`, `ElementDefinition` (`Path`, `Min`, `Max`, `Type []ElementType`, `Binding *ElementBinding`,
  `IsModifier`, `IsSummary` via the `"isSummary"` JSON key, `Fixed*`/`Pattern*` captured as `json.RawMessage` keyed by
  the choice suffix — deferred refinement noted), `ElementType` (`Code`, `TargetProfile`), `ElementBinding` (`Strength`,
  `ValueSet`), `ValueSet`/`CodeSystem` (enough to enumerate the codes of a required binding). Mirror the prototype's
  `model/definition.go` field set, adding `IsSummary` and the fixed/pattern raw capture the prototype lacked.
- [ ] **Step 4: Run test to verify it passes.**
- [ ] **Step 5: Commit.** `feat(fhir/gen): add raw StructureDefinition/ValueSet decode records`

---

### Task 1.3: Load + index the full bundle

**Files:**
- Modify: `fhir/internal/gen/loader/loader.go`
- Test: `fhir/internal/gen/loader/loader_test.go` (append)

- [ ] **Step 1: Write the failing test.** Point `Load` at the vendored R5 directory (Increment 0) and assert: it returns
  no error; the index contains the well-known resources (`Patient`, `Observation`, `Bundle`, `OperationOutcome`,
  `ServiceRequest`, `ImagingStudy`, `DiagnosticReport`) by name; it contains the well-known complex datatypes
  (`Reference`, `Identifier`, `CodeableConcept`, `Quantity`, `HumanName`, `Period`); it resolves the
  `administrative-gender` value set; and the resource count is in the expected band (assert `>= 140 && <= 170` rather
  than an exact 158, so a definition-bundle patch release does not break the test — the exact count is asserted in the
  conformance statement, not here). Then a second test points `Load` at a temp dir missing `valuesets.json` and asserts a
  `*LoadError` naming the missing file.

- [ ] **Step 2: Run test to verify it fails.** `undefined: Load`, `Bundle.StructureDefinition(name)`.
- [ ] **Step 3: Write minimal implementation.** `Load(dir)` → `verifyChecksums(dir)`; read each `profiles-*.json` /
  `valuesets.json`; each file is a FHIR `Bundle` whose `entry[].resource` is a `StructureDefinition`/`ValueSet`/
  `CodeSystem`; decode each entry by peeking its `resourceType`, append to typed indexes keyed by `url` and `name`;
  return `*Bundle{ byName, byURL, valueSets, codeSystems }` with accessor methods.
- [ ] **Step 4: Run test to verify it passes.** Run against the real vendored bundle.
- [ ] **Step 5: Commit.** `feat(fhir/gen): load and index the verified R5 definition bundle`

---

**Increment 1 verification gate:** `go test -race ./fhir/internal/gen/loader/...` green; `golangci-lint run
./fhir/internal/...` clean; the checksum-mismatch and missing-file regressions pass; `Load` against the real vendored R5
bundle indexes the workflow resources and the core datatypes. The loader imports only the standard library; it never
reaches the network (verified by inspection — no `net/http` import).

---

## Increment 2 — The model / IR (element-path tree)

**Goal:** Turn the loader's flat list of `ElementDefinition`s (each keyed by a dotted path) into the explicit **element
tree** the generator recurses over, and classify every `StructureDefinition` by `kind`. From `Observation`'s flat
snapshot — `Observation`, `Observation.component`, `Observation.component.referenceRange`,
`Observation.component.referenceRange.low` — build a tree where `component` is a backbone element with a `referenceRange`
child that itself has a `low` child, so backbone structs are fully populated (closing the structural half of FHIR-006).
Attach per-element metadata the later increments consume: cardinality (`min`/`max`), the type set (for choice detection),
the binding name+strength, `isSummary`, `isModifier`, and any fixed/pattern value. Detect choice elements (`path` ending
in `[x]`) and group their branch types. This increment produces no Go; it produces the in-memory model and is gated by
tests on that model.

**Files:** `fhir/internal/gen/model/tree.go` (`Element` node: `Name`, `Path`, `Min`, `Max`, `Children []*Element`,
`Types`, `Binding`, `IsSummary`, `IsModifier`, `IsChoice`, `ChoiceBranches`, `Fixed`), `model/build.go`
(`BuildTree(sd *loader.StructureDefinition) (*Element, error)` — splits paths, nests children, errors on a child whose
parent path is absent — the missing-parent guard that would otherwise produce an empty backbone), `model/classify.go`
(`Classify(sd) Kind` → primitive-type / complex-type / resource), `model/choice.go` (`[x]` detection + branch typing),
tests for each.

**Key tests:** `BuildTree(Observation)` yields a `component` child that has a `referenceRange` child that has a `low`
child (the FHIR-006 deep-backbone regression — `TestBackboneTreeFullyRecursed`); a `[x]` element (`Observation.value[x]`)
is flagged `IsChoice` with its branch type set (`Quantity`, `CodeableConcept`, `string`, …); cardinality parses
(`min:1`/`max:"1"` → required scalar, `max:"*"` → slice); `Classify` puts `Period` in complex-type, `boolean` in
primitive-type, `Patient` in resource; an element whose parent path is missing from the snapshot is a hard error, not a
silent drop.

**Reference-doc section:** fhir.md "What the generator must get right" (the element-path-tree bullet and the FHIR-006
guard), "Choice types", "Cardinality and required validation". **Rewrite** of the prototype `model/definition.go` (its
IR had `Field`s but no tree, so backbones were empty).

**Verification gate:** `go test -race ./fhir/internal/gen/model/...` green; lint clean; the FHIR-006 deep-backbone and
choice-detection regressions pass.

---

## Increment 3 — Planner + emitter skeleton (one trivial datatype end-to-end)

**Goal:** Prove the whole back half of the pipeline — name mapping, type planning, templating, formatting, and the
golden-test harness — by generating exactly one simple complex datatype (`Period`, two optional `dateTime` fields) into
`fhir/r5`. Build the **name mapper** (FHIR element name → exported Go field name, FHIR type name → Go type name,
honouring the `UBIQUITOUS_LANGUAGE.md` collision rules so a FHIR `Reference` never collides with anything and Go
keywords/initialisms are handled), the **type planner** (FHIR primitive/complex/resource → Go type; cardinality →
pointer for optional scalar, slice for repeating, value for required-after-validate), the **emitter** (`text/template`
producing Go source, run through `go/format` so output is `gofmt`-stable and diff-friendly), and the **golden harness**
(`generateToString(name)` compared against `testdata/golden/r5/period.go.golden`). Emitting one datatype end-to-end
de-risks every later increment, which then only adds template branches.

**Files:** `fhir/internal/gen/plan/names.go` (`GoFieldName`, `GoTypeName`, the collision table), `plan/types.go`
(`PlanField(*model.Element) Field` → Go type + pointer/slice decision), `gen/emit/emit.go` (the template registry +
`go/format` pass), `gen/emit/templates/datatype.go.tmpl`, `gen/emit/golden_test.go` (the golden harness),
`fhir/internal/gen/testdata/golden/r5/period.go.golden`, and the first generated output `fhir/r5/period.go` (committed
separately as generated data).

**Key tests:** `GoFieldName("value[x]")`/`GoTypeName("CodeableConcept")` map per the collision table; `PlanField` makes
an optional `0..1 dateTime` a `*string` and a `1..* Identifier` a `[]Identifier`; the emitter's output for `Period`
matches `period.go.golden` byte-for-byte after `go/format`; the generated `fhir/r5/period.go` compiles and a round-trip
test (`json.Marshal`/`Unmarshal` of a `Period`) passes. The golden harness fails loudly (showing a diff) when the
template drifts.

**Reference-doc section:** fhir.md "Complex datatypes", "Canonical element ordering", "The code generator" (gofmt-stable,
byte-for-byte reproducible); UBIQUITOUS_LANGUAGE.md (Go naming + cross-standard collisions). **Rewrite** of the prototype
`codegen/{builder,generator}.go`.

**Verification gate:** `go test -race ./fhir/internal/gen/...` green; lint clean; `fhir/r5/period.go` compiles and
round-trips; `gen:fhir-r5` regenerates `period.go` identically (byte-for-byte).

---

## Increment 4 — Resource identity machinery + registry generation

**Goal:** Make the generated resources usable through the committed identity API. Extend the root `fhir` package's
`Unmarshal[T]`/`As[T]`/`UnmarshalResource` so `Unmarshal[T]` **peeks `resourceType` before fully decoding** and returns
`ErrResourceTypeMismatch` on a mismatch (FHIR-003), `As[T]` is a checked downcast, and `UnmarshalResource` consults the
`resourceType→factory` registry (returning `ErrUnknownResourceType` for an unknown type). Generate each release package's
registry as an `init()` that registers every resource's factory. This increment wires identity for the one resource so
far (`Period` is a datatype, so add a tiny generated stub resource or defer the behavioural test to Increment 6 — see the
note) and proves the registry mechanism; the bulk resources arrive in Increment 6 and auto-register.

**Files:** `fhir/resource.go` (extend with `Unmarshal[T]`, `As[T]`, `UnmarshalResource`, the registry map +
`registerFactory`), `fhir/resource_test.go`, the emitter's `registry.go.tmpl` producing `fhir/r5/registry.go` (the
`init()` registering factories), `gen/emit/golden_test.go` (registry golden).

**Key tests:** `Unmarshal[*r5.Patient]` of a `Patient` payload succeeds; of an `Observation` payload returns
`ErrResourceTypeMismatch` and a zero `*Patient` (the FHIR-003 regression — `TestUnmarshalChecksResourceType`);
`UnmarshalResource` of an unknown `resourceType` returns `ErrUnknownResourceType`; `As[*r5.Patient]` on a `Resource`
holding an `Observation` returns `(nil, false)`. (Use a generated test resource or run these against the real `Patient`
once Increment 6 lands; pin the behavioural test there and keep the unit tests on the generic functions here.)

**Reference-doc section:** fhir.md "The Resource interface and type-safe access" (the three function signatures, the
FHIR-003 fix), "Registry" bullet under "What the generator must get right". **Rewrite** (the prototype's
`UnmarshalResource[T]` never checked `resourceType`).

**Verification gate:** `go test -race ./fhir/...` green; lint clean; the FHIR-003 `resourceType`-mismatch regression
passes; the generated registry compiles.

---

## Increment 5 — Primitive types + the `_field` extension sibling

**Goal:** Generate the primitive layer correctly. Establish the primitive→Go-scalar mapping from fhir.md's table
(`boolean`→`bool`, the integer family→`int32`/`int64`, the string family→`string`, `decimal`→`fhir.Decimal` per
FHIR-009, the date/time family→`string` validated on decode), generate **required scalar primitives as pointers** so a
required `false`/`0` is present-because-non-nil (FHIR-007, the structural half), and generate the `_field`
primitive-extension sibling: the shared `fhir.PrimitiveElement` type in the root, a `XxxElement *PrimitiveElement` field
for each scalar primitive and a `XxxElement []*PrimitiveElement` for each repeating primitive, with custom marshalling
that null-aligns the repeating arrays (FHIR-005 — and crucially, **no `_field` sibling on any complex/backbone element**).
The primitive wrapper types for choice branches (`FHIRString`, `FHIRBoolean`, `FHIRInteger`, `FHIRDateTime`, …) are
generated here too, since Increment 7 needs them.

**Files:** `fhir/primitive.go` (the shared `PrimitiveElement` + the scalar/repeating null-alignment marshalling
helpers), `fhir/primitive_test.go`, `plan/primitives.go` (the primitive→Go map + the wrapper-type set), the emitter
templates `primitive_wrappers.go.tmpl` and the `_field` sibling logic woven into the datatype/resource templates,
`fhir/r5/primitives.go` (generated wrappers), golden + behavioural tests.

**Key tests:** the primitive map matches fhir.md's table exactly; a generated struct with a scalar primitive
(`Patient.gender`) gets a `GenderElement *PrimitiveElement` with a `json:"_gender,omitempty"` tag and no underscore in
the Go identifier; a repeating primitive (`HumanName.given`) gets `GivenElement []*PrimitiveElement` that round-trips
`"given":["Jane","Q"]` / `"_given":[null,{"id":"x"}]` with the `null` placeholder preserved
(`TestRepeatingPrimitiveNullAlignment`); a **complex** field (`Patient.name`) gets **no** `_field` sibling (the FHIR-005
regression — `TestNoUnderscoreSiblingOnComplex`); `decimal` maps to `fhir.Decimal` and a `1.20` round-trips lexically.

**Reference-doc section:** fhir.md "Primitive types and the `_field` extension sibling", "The Decimal primitive",
"Cardinality and required validation" (pointers for required scalars). **New** (the prototype emitted `_field` siblings
on non-primitives and mapped `decimal` to `float64`).

**Verification gate:** `go test -race ./fhir/...` green; lint clean; the FHIR-005 (no sibling on complex), FHIR-007
(required-as-pointer), FHIR-009 (decimal lexical), and repeating-null-alignment regressions pass.

---

## Increment 6 — Complex datatypes + full resource & backbone generation

**Goal:** The bulk-generation increment. Run the full pipeline over every R5 `StructureDefinition` and emit: every
complex datatype, every resource, and every nested backbone element, recursed from the element-path tree (Increment 2)
so no backbone is empty (FHIR-006). Each resource gets a `ResourceType()` returning a per-type constant and a
`MarshalJSON` that **always** emits `"resourceType"` as that constant — even on a zero value (FHIR-004) — and as the
first key. Fields and JSON keys are emitted in `StructureDefinition` snapshot order (canonical element ordering, so
output is stable and diff-friendly). This increment also makes the generated datatypes **supersede** the M2 hand-written
`fhir/r5/datatypes.go` shapes (`Reference`, `Identifier`, `Coding`, `CodeableConcept`, `CodeableReference`) — the
generated versions must match those committed shapes (the migration that removes the hand-written file is Increment 12).

**Files:** the emitter templates `resource.go.tmpl`, `backbone.go.tmpl`, `datatype.go.tmpl` (extended), and the
**generated output** — `fhir/r5/<resource>.go` for each resource (committed as generated data in a small number of
`chore(fhir/r5): regenerate ...` commits, not 158 commits), `fhir/r5/datatypes.go` (regenerated, superseding the M2
hand-written one in Increment 12), `gen/emit/golden_test.go` (goldens for `Reference`, a representative resource, and the
deep backbone), `fhir/r5/roundtrip_test.go`.

**Key tests:** the generated `r5.Reference`/`r5.Identifier`/`r5.CodeableConcept`/`r5.CodeableReference` match the
committed fhir.md shapes byte-for-byte against goldens; `r5.Observation` has a populated nested
`ObservationComponentReferenceRange` backbone with a `Low`/`High` field (the FHIR-006 regression —
`TestObservationBackboneNotEmpty`); `json.Marshal(&r5.Patient{})` emits `{"resourceType":"Patient"}` not
`{"resourceType":""}` (the FHIR-004 regression); JSON keys appear in snapshot order; the entire generated `fhir/r5`
package **compiles** (the headline gate); a Patient/Observation round-trips known JSON.

**Reference-doc section:** fhir.md "Complex datatypes", "What the generator must get right" (resources/backbones,
`resourceType`, canonical ordering), conformance/fhir.md "In scope (v1)" (every resource and backbone element).
**Rewrite** (the prototype produced empty backbones and `package resources` in a `types` dir).

**Verification gate:** `go test -race ./fhir/r5/...` green; lint clean; the FHIR-004 and FHIR-006 regressions pass; the
full generated `fhir/r5` package compiles; `gen:fhir-r5` reproduces every generated file byte-for-byte.

---

## Increment 7 — Choice-type accessor generation

**Goal:** Generate the choice-type machinery for every `[x]` group (for example `Observation.value[x]`,
`Patient.deceased[x]`). Per group emit: a **sealed value interface** (`ObservationValue interface{ isObservationValue() }`)
implemented only by named datatype types and the primitive wrappers from Increment 5 (never the built-in `string`/`bool`/
`int32`, which cannot carry the marker method); the **suffixed storage fields** (`ValueQuantity *Quantity`,
`ValueString *FHIRString`, …) for faithful JSON; a `Value() (ObservationValue, bool)` getter; and one `SetValueX` setter
per branch that **clears every other sibling** so at most one is ever populated (FHIR-001). Generate custom
`MarshalJSON`/`UnmarshalJSON` that emit/parse exactly the one set suffixed key (so the prototype's unsuffixed `*any`
field, which never round-tripped conformant JSON, is gone — FHIR-002). A `required` choice group is validated once per
group in Increment 11.

**Files:** the emitter template `choice.go.tmpl` (the interface, storage fields, getter, setters, and the
marshal/unmarshal that dispatches on the suffix), `plan/choice.go` (branch → suffix + Go type + wrapper-or-struct
decision), regenerated `fhir/r5/observation.go` etc. carrying the accessors, `gen/emit/golden_test.go` (choice golden),
`fhir/r5/choice_test.go`.

**Key tests:** `obs.SetValueQuantity(q)` then `obs.SetValueString(r5.FHIRString("x"))` leaves only `valueString` set —
setting a branch clears the others (the FHIR-001 regression — `TestChoiceSetterClearsSiblings`); `obs.Value()` returns
the set branch via a type switch; marshalling an `Observation` with a `Quantity` value emits `"valueQuantity":{...}` and
no other `value*` key, and round-trips (the FHIR-002 regression); a primitive-valued branch boxes through the wrapper
(`SetValueString(r5.FHIRString("normal"))` → `"valueString":"normal"`); two branches are unreachable through the API.

**Reference-doc section:** fhir.md "Choice types: typed accessors and mutual exclusion", "What the generator must get
right" (choice groups bullet). **Rewrite** (FHIR-001/002).

**Verification gate:** `go test -race ./fhir/r5/...` green; lint clean; the FHIR-001 (setter clears siblings) and
FHIR-002 (single suffixed key round-trip) regressions pass; the generated choice code compiles and reproduces
byte-for-byte.

---

## Increment 8 — Required-binding enum generation

**Goal:** Generate closed Go enums for every `required`-strength value-set binding, the core of the type-safety thesis
(FHIR-013 — the prototype stored these as metadata only and never enforced them). Per required binding emit: a defined
string type (`AdministrativeGender string`), a const set (`GenderMale`, `GenderFemale`, …) enumerated from the bound
ValueSet/CodeSystem (resolved by the loader), a validating `ParseXxx(s) (T, error)` that returns `ErrUnknownCode`
(wrapped with the binding name and offending token, no PHI) for an out-of-set code, and a `UnmarshalJSON` honouring the
decode policy: strict (default) rejects an unknown code with `ErrUnknownCode`; lenient (opt-in) retains it for `Validate`
to surface. Weaker-strength bindings (`extensible`/`preferred`/`example`) stay `string` (the const set may be emitted as
documentation-only constants but no `Parse` rejects). The required scalar enum field is still generated to honour the
present-not-truthy rule.

**Files:** the emitter template `enum.go.tmpl` (type + consts + `Parse` + `UnmarshalJSON`), `plan/binding.go` (resolve
the binding's codes from the loader's ValueSet/CodeSystem index, decide strict-vs-string from strength), the decode-policy
plumbing (a generator-independent `DecodeOptions` in the root `fhir` package read by the generated `UnmarshalJSON`),
regenerated enum files, `gen/emit/golden_test.go` (enum golden), `fhir/r5/enum_test.go`.

**Key tests:** `r5.ParseAdministrativeGender("female")` succeeds and `"banana"` returns `ErrUnknownCode` (the FHIR-013
regression — `TestParseRequiredBindingRejectsUnknown`); strict decode of `{"gender":"banana"}` into `Patient` errors;
lenient decode retains `"banana"` and `Validate` reports it; an `extensible` binding field stays `string` and accepts any
value; the const set matches the bound ValueSet's codes; `ErrUnknownCode`'s message names the binding and token, never a
patient value.

**Reference-doc section:** fhir.md "Generated enums for required value-set bindings", "Unknown-code policy". **Rewrite**
(FHIR-013).

**Verification gate:** `go test -race ./fhir/r5/...` green; lint clean; the FHIR-013 unknown-code regression and the
strict/lenient decode policy tests pass; enum files reproduce byte-for-byte.

---

## Increment 9 — Bundle generation + typed builders

**Goal:** Generate the `Bundle` resource and hand-write the release builders that make the `bdl-*` invariants
unrepresentable-when-wrong (FHIR-010 — the prototype incremented `total` for every type and enforced nothing). The
generated `r5.Bundle` models the entry structure faithfully (`fullUrl`, `resource`, `request`, `response`, `search`); the
hand-written builders `NewSearchSet(total, ...SearchEntry)`, `NewTransaction(...TransactionEntry) (*Bundle, error)`,
`NewDocument(composition, ...) (*Bundle, error)`, `NewMessage(header, ...) (*Bundle, error)` enforce: `total` only for
searchset/history; `entry.search` only in searchset; `entry.request` required in every transaction/batch entry and
`entry.response` only in response variants; a document's first entry is a `Composition` and a message's first a
`MessageHeader`; `fullUrl` unique across entries. The `BundleType` required-binding enum comes from Increment 8. The
build-once-then-immutable rule is documented, not mutex-guarded (FHIR-015).

**Files:** the generated `fhir/r5/bundle.go` (the resource), the hand-written `fhir/r5/bundle_builders.go`
(`NewSearchSet`/`NewTransaction`/`NewDocument`/`NewMessage`, the `SearchEntry`/`TransactionEntry`/… entry types), and
`fhir/r5/bundle_test.go`. (The builders are hand-written per-release because they encode FHIR's per-type prose rules,
which are not in the StructureDefinition; this is the one deliberate exception to "all generated", and it is noted in the
package doc.)

**Key tests:** `NewSearchSet(3, ...)` sets `total:3` and `type:"searchset"`; `NewTransaction` with an entry carrying a
`response` returns an error (the FHIR-010 regression — `TestTransactionRejectsResponseEntry`); a transaction with
duplicate `fullUrl` is rejected; `NewDocument` with a non-`Composition` first entry errors; `total` is unset for a
collection bundle; `Validate` (Increment 11) applies the same checks to a decoded bundle.

**Reference-doc section:** fhir.md "Bundles" (all five builders and the invariant list), "What the generator must get
right". **Rewrite** of the prototype `fhir/bundle.go` (FHIR-010/015).

**Verification gate:** `go test -race ./fhir/r5/...` green; lint clean; the FHIR-010 per-type-invariant regressions
pass; the generated `Bundle` reproduces byte-for-byte and the hand-written builders are clearly separated from generated
code.

---

## Increment 10 — Reference integrity + `OperationOutcome`

**Goal:** Generate the per-release `OperationOutcome` resource, add the release-agnostic `IssueSeverity` binding to the
root, and implement local reference resolution. `IssueSeverity` and its constants (`SeverityFatal`/`Error`/`Warning`/
`Information`/`Success`) live in the root `fhir` package because they are identical across R4/R5 (the R5-only `success` is
noted); both release `OperationOutcome`s alias them. Implement `Bundle.Resolve(ref) (fhir.Resource, ok)` (resolves
`entry.fullUrl` and `#id` contained references), `DomainResource.Contained(id) (fhir.Resource, error)` returning an
**aggregate error naming the offending entry index** when a contained resource is malformed rather than a silent miss
(FHIR-011), and `Bundle.CheckReferenceIntegrity() *fhir.OperationOutcome` walking every `Reference` and reporting
unresolved local references (external absolute URLs are not flagged). `OperationOutcome.HasErrors()` and `.Error()` (the
no-PHI Go-error adapter) live with the root machinery.

**Files:** `fhir/outcome.go` (the root `IssueSeverity` + constants + `HasErrors`/`Error` on the generated outcome — or a
small root interface the generated outcome satisfies; pin the split in expansion against fhir.md "OperationOutcome and
the error model"), generated `fhir/r5/operation_outcome.go`, `fhir/r5/reference.go` (the `Resolve`/`Contained`/
`CheckReferenceIntegrity` methods — these are hand-written per-release helpers on the generated types, like the bundle
builders), `fhir/r5/reference_test.go`.

**Key tests:** `Bundle.Resolve` finds an entry by `fullUrl` and a `#id` contained resource; an external absolute URL
resolves to `ok=false`; `Contained` of a malformed contained entry returns an error naming the index, not a silent miss
(the FHIR-011 regression — `TestContainedMalformedReturnsError`); `CheckReferenceIntegrity` reports a dangling `#id` as an
issue and ignores an external URL; `HasErrors` is false on a nil `*OperationOutcome` and on an all-information outcome.

**Reference-doc section:** fhir.md "Reference integrity", "OperationOutcome and the error model". **New / rewrite**
(FHIR-011).

**Verification gate:** `go test -race ./fhir/...` green; lint clean; the FHIR-011 malformed-contained regression passes;
the generated `OperationOutcome` reproduces byte-for-byte.

---

## Increment 11 — Cardinality / required `Validate` + summary mode

**Goal:** Implement the structural validator and the summary serialiser. `Validate(r Resource) *OperationOutcome` checks,
reporting **every** issue (not stopping at the first): required-element presence by **presence, not truthiness** (track
whether the field or its `_field` sibling was set, never `IsZero` on the value — FHIR-007's behavioural half),
choice-group cardinality (a required group with no branch set is one issue per group), required-binding codes (an
out-of-set code retained under lenient decode), and the `bdl-*` Bundle invariants reachable through the builders. It does
**not** do terminology expansion, profile slicing, or FHIRPath in v1 (deferred; conformance/fhir.md). `MarshalSummary(r,
mode)` implements the five `SummaryMode`s driven by the generator-recorded `isSummary` flag per element (so `SummaryTrue`
is data-driven), sets `meta.tag SUBSETTED` when elements are dropped, and returns `ErrNilResource` for a nil resource
rather than panicking (FHIR-012). To make `Validate` data-driven, the generator emits a per-resource **validation
descriptor** (required paths, choice groups, required bindings, summary flags) the engine consumes — decide table-driven
descriptor vs reflection in expansion (recommend a generated descriptor so there is no reflection on the hot path).

**Files:** `fhir/validate.go` (the engine + the descriptor interface), `fhir/summary.go` (`SummaryMode`,
`MarshalSummary`, `ErrNilResource`), the emitter template `descriptor.go.tmpl` producing
`fhir/r5/validation_descriptors.go` (the per-resource required/choice/binding/summary metadata), `fhir/validate_test.go`,
`fhir/summary_test.go`.

**Key tests:** `Validate` of a resource missing a required element reports it; a present required `false` (non-nil
pointer) is **not** reported missing (the FHIR-007 behavioural regression — `TestRequiredFalseIsPresent`); a required
choice group with no branch set is reported once; `Validate` reports every issue, not just the first;
`MarshalSummary(nil, SummaryTrue)` returns `ErrNilResource` not a panic (the FHIR-012 regression);
`MarshalSummary(patient, SummaryTrue)` emits only `isSummary` elements and sets the `SUBSETTED` tag; `SummaryCount` on a
bundle emits `total` and no entries.

**Reference-doc section:** fhir.md "Cardinality and required validation", "Summary mode". **New / rewrite**
(FHIR-007 behavioural, FHIR-012).

**Verification gate:** `go test -race ./fhir/...` green; lint clean; the FHIR-007 (present-false) and FHIR-012
(nil-summary) regressions pass; the validation descriptors reproduce byte-for-byte.

---

## Increment 12 — Migrate `convert` and the skeleton off the hand-written resources

**Goal:** Retire the M2 Increment 11 hand-written minimal resources now that the generator produces the full set. Delete
`fhir/r5/service_request.go`, `diagnostic_report.go`, `imaging_study.go`, and the hand-written `fhir/r5/datatypes.go`
(now generated by Increment 6), and re-point the `convert` package and the M2 end-to-end skeleton test at the generated
types. The generated `ServiceRequest`/`DiagnosticReport`/`ImagingStudy` carry **all** their fields, so `convert` sets the
same subset it set before; the field names match (the generator's name mapper produces the same idiomatic names the
hand-written resources used — verify and reconcile any drift, for example a generated `Series []ImagingStudySeries` vs a
hand-written one). Keep `TestSkeletonEndToEnd` green: this is the regression that proves the generator's output is a
drop-in superset of the hand-written slice. The shared `fhir.Decimal` alias to `dicom.Decimal` stays (it already matches
the generator's need — confirm the M6a unification noted in the M2 plan's Open question 1: the leading option is to keep
the `type Decimal = dicom.Decimal` alias and split the length rule so a long-but-valid FHIR decimal is not rejected by the
DICOM DS 16-byte cap).

**Files:** delete the four hand-written `fhir/r5/*.go` resource/datatype files; modify `convert/orm_servicerequest.go`,
`convert/sr_diagnosticreport.go`, `convert/dicom_imagingstudy.go`, `convert/options.go` (the `WithSubjectR5(r5.Reference)`
option), and any `convert/*_test.go` golden JSON that pinned the hand-written shape; modify `fhir/decimal.go` if the
DS-cap split is taken (resolve in expansion). Touch nothing in the other parallel worktrees.

**Key tests:** `go test -race ./convert/...` green against the generated resources; `TestSkeletonEndToEnd` (the M2
acceptance test) green; `ORMToServiceRequestR5`/`DICOMToImagingStudyR5` produce JSON matching the previously-committed
golden shapes (the generated types are a superset, so the marshalled subset is identical given `omitempty`); the
UID→Identifier identity rule still holds (never a `Reference.reference` URL for a DICOM UID).

**Reference-doc section:** fhir.md "Release selection in code", "Complex datatypes"; convert.md (the converter field
tables); the M2 walking-skeleton plan Increment 11/12 (what the hand-written slice provided). **Migration** (no new
public API; swap the implementation under the same `r5.*` names).

**Verification gate:** `go test -race ./convert/... ./fhir/...` green; lint clean; `mise run test:skeleton` green; the
hand-written files are gone and nothing imports them.

---

## Increment 13 — Reproducibility + full-set compile gate

**Goal:** Lock in the "never hand-edited, byte-for-byte reproducible" guarantee and the "every resource compiles"
guarantee. Add a CI-grade test that regenerates the entire R5 tree into a temp directory and **diffs it byte-for-byte**
against the committed `fhir/r5` (failing with the offending file path on any drift), a compile-test that builds the whole
generated package, and a round-trip test over the **vendored HL7 R5 example JSON corpus** (a sample of the official
`examples-json.zip`, vendored and checksum-pinned like the definitions) asserting decode-then-encode reproduces the
input for the workflow resources. Wire `gen:verify` into the mise/CI gate.

**Files:** `fhir/internal/gen/reproduce_test.go` (regenerate-to-temp + diff), `fhir/r5/examples_test.go` (the example-JSON
round-trip), `fhir/internal/gen/testdata/examples/r5/` (vendored example JSON + `SHA256SUMS`), `mise.toml` (`gen:verify`
wired), `.github/workflows/ci.yml` (add the regenerate-and-diff + example-round-trip jobs — coordinate with the existing
CI structure noted in the build-state memory; confirm at expansion).

**Key tests:** regenerating R5 into a temp dir matches the committed tree exactly (the reproducibility regression —
`TestRegenerationByteForByte`); the entire `fhir/r5` package compiles (`go build ./fhir/r5/`); a vendored `Patient`,
`Observation`, `Bundle`, and `ServiceRequest` example round-trips byte-stably (modulo FHIR's permitted whitespace/key
ordering — pin the comparison semantics in expansion: structural-equal after canonicalisation, or byte-stable if the
generator preserves order).

**Reference-doc section:** fhir.md "The code generator" (byte-for-byte reproducible, never hand-edited), "Conformance
scope and limits" (round-trip guarantees); PRD §11.1. **New.**

**Verification gate:** `mise run gen:verify` green (regeneration is a no-op diff); the full-set compile and example
round-trip pass; CI runs both.

---

## Increment 14 — R5 conformance gate (HL7 FHIR validator) — M6a acceptance

**Goal:** The M6a acceptance gate. Run the workflow set — `Patient`, `Encounter`, `ServiceRequest`, `ImagingStudy`,
`DiagnosticReport`, `Observation`, `Bundle`, `OperationOutcome` — produced by go-radx through the **official HL7 FHIR
validator** in CI and require zero errors. Marshal a representative, fully-populated instance of each workflow resource
(including choice types, primitive extensions, and a bundle with references) and validate it against the R5 profile; a
validator error is merge-blocking (PRD §11.1). This proves the generated R5 code is not merely compilable but
spec-conformant on the resources go-radx commits to.

**Files:** `fhir/conformance/r5_validator_test.go` (`//go:build conformance` or a CI-only tag, mirroring the M2 interop
gating so it skips locally and runs under CI), `tools/fhir-conformance/` (the validator runner — a wrapper around the
HL7 validator jar/container; reuse the M0 FHIR-validator wiring per PRD §13 M0 "CI with … the FHIR validator"), the
workflow-resource fixtures, `mise.toml` (`[tasks."conformance:fhir"]`), `.github/workflows/ci.yml` (the merge-blocking
FHIR-validator job).

**Key tests:** each workflow resource marshalled by go-radx validates with **zero errors** against the HL7 FHIR R5
validator (the M6a acceptance regression); a deliberately-invalid fixture (wrong cardinality) is caught by the validator
(proving the gate actually validates); a bundle with intact references validates.

**Reference-doc section:** fhir.md "Scope and conformance" (the workflow set conformance-tested against the validator),
"Conformance scope and limits"; conformance/fhir.md (the conformance-tested resource list); PRD §11.1, §13 M6a, §14
(generator correctness mitigation). **New.**

**Verification gate:** `mise run conformance:fhir` green against the HL7 FHIR R5 validator for the full workflow set in
CI; the gate is not skipped in CI (merge-blocking). **This is the M6a acceptance gate.**

---

## Increment 15 — M6b: R4 4.0.1 generation + R4 conformance gate

**Goal:** Run the same generator over R4. Vendor the pinned R4 4.0.1 `definitions.json.zip` (checksum-pinned, like R5),
add `-release r4` to the `go generate` directive, generate the full R4 resource/datatype/enum/backbone set into
`fhir/r4`, and reconcile the R4↔R5 datatype differences the generator must respect from each release's own
StructureDefinitions (for example R4 has no `CodeableReference`, so `ServiceRequest.code` is a `CodeableConcept` and
`ImagingStudy.series.bodySite` is a `Coding`; `OperationOutcome.issue.severity` lacks the R5-only `success`). R4 and R5
remain genuinely distinct type spaces (FHIR-014). Then pass the workflow set through the validator at R4 (US Core runs on
R4, so R4 conformance matters most for the deployed base). The generator should require **no R5-specific branches** — if a
release difference forces a code path, that is a signal the model is leaking release assumptions; surface it.

**Files:** `fhir/internal/gen/testdata/definitions/r4/` (vendored R4 bundle + `SHA256SUMS` + `SOURCE.md`), the generated
`fhir/r4/*.go` (committed as generated data), `fhir/gen.go` (add the R4 directive), `fhir/conformance/r4_validator_test.go`,
the R4 example corpus for the round-trip, `mise.toml` (`gen:fhir-r4`, extend `conformance:fhir`).

**Key tests:** the full generated `fhir/r4` package compiles; R4 regenerates byte-for-byte; `r4.ServiceRequest.Code` is a
`*CodeableConcept` (not `CodeableReference`) — the release-difference regression; `r4.OperationOutcome` has no `success`
severity; the workflow set validates with zero errors against the HL7 FHIR R4 4.0.1 validator; an R4 example corpus
round-trips. The R4 and R5 `Reference`/`Identifier` are distinct types in distinct packages (FHIR-014 — a value of one
does not assign to the other).

**Reference-doc section:** fhir.md "Scope and conformance" (R4 generated directly, not folded into R4B), "Release
selection in code"; conformance/fhir.md "Conformance metadata" (R4 4.0.1); PRD §5.3, §13 M6b. **New** (the same generator,
a new release).

**Verification gate:** `go test -race ./fhir/r4/...` green; lint clean; R4 regenerates byte-for-byte; the FHIR-014
distinct-type-space and the R4-vs-R5 datatype-difference regressions pass; `mise run conformance:fhir` green for the
workflow set at R4. **This is the M6b acceptance gate.**

---

## Open questions and resolutions

Resolved against the committed specs and the prototype's approach where the evidence is clear; flagged for the architect
where a real choice remains.

1. **Spec source — RECOMMENDED (vendor + checksum + commit; refresh-only download).** The StructureDefinitions come from
   the official HL7 FHIR **definitions** package per release — `https://hl7.org/fhir/R5/definitions.json.zip` and
   `https://hl7.org/fhir/R4/definitions.json.zip`, which unzip to `profiles-types.json`, `profiles-resources.json`, and
   `valuesets.json` (the StructureDefinition bundles plus the value sets the required-binding enums need). The prototype
   already used exactly these files (it downloaded `fhir.schema.json.zip` but the generator read `profiles-*.json`); the
   recommendation is to **vendor and commit** that bundle under `fhir/internal/gen/testdata/definitions/<release>/`,
   record a **SHA-256** per file in a committed `SHA256SUMS`, and have the loader verify the checksum before parsing.
   Generation never touches the network. A **refresh-only** mise task (`fhir:refresh-r5`) downloads a new bundle and
   re-checksums it, run deliberately when bumping the FHIR release — not at generate time. Rationale: a library generator
   must be reproducible offline and in CI without trusting a live HL7 endpoint (twelve-factor "build, release, run"
   separation; PRD §9.2 fail-closed; fhir.md "reproducible from pinned, checksum-verified … inputs"). The committed
   bundle is sizeable (the R5 `definitions.json.zip` is a few MB unzipped); if repo weight is a concern, the alternative
   is committing only the SHA-256 + `SOURCE.md` and fetching-then-verifying in a CI cache, but that reintroduces a network
   dependency for generation — **recommend full vendoring**; flag the repo-weight tradeoff for the architect.

2. **Generate-time vs build-time — RECOMMENDED (`go generate` producing committed `.go` files; no runtime reflection).**
   Generation is a **build-time** `go generate` step that writes committed Go into `fhir/r4`/`fhir/r5`; consumers import
   plain generated Go. This is what fhir.md commits ("The generator is a build-time tool, not part of the runtime API
   surface; consumers import the generated packages, never the generator" and "`go generate ./fhir/...` reproduces the
   committed output byte-for-byte"). No runtime reflection and no codegen at `go build` — a library consumer must get
   ordinary, inspectable, IDE-navigable Go with no build-time toolchain requirement. This is not really open; it is
   pinned by the reference doc and restated here so an executor does not drift into a reflection-based design.

3. **HL7 FHIR validator wiring — FLAG for the architect.** The conformance gate (Increments 14/15) needs the official HL7
   FHIR validator in CI. PRD §13 M0 says CI was wired with "the FHIR validator", but the build-state memory's CI
   description (the five jobs added in the M2 detour) does not list a FHIR-validator job — so the M0 wiring may be
   nominal, not yet active. **Resolve before Increment 14:** confirm whether the validator runs as the Java
   `validator_cli.jar`, a `hl7fhir/validator` container, or the Inferno/Touchstone path, and whether the M0 setup left a
   runner stub. The plan assumes a jar-or-container wrapper under `tools/fhir-conformance/`; the exact mechanism is the
   architect's call.

4. **`fhir.Decimal` / `dicom.Decimal` unification — RECOMMENDED (keep the alias; split the length rule).** The M2 plan's
   Open question 1 deferred this to M6a. The generator needs `decimal` pervasively. The current `fhir/decimal.go` already
   aliases `type Decimal = dicom.Decimal` and carries a `TODO(M6a)` noting the DICOM DS 16-byte cap is stricter than the
   FHIR decimal production. **Recommend keeping the alias** (the glossary's single `Decimal` noun; intra-library coupling
   is acceptable per PRD §7.4, which constrains only the CLI module graph) and **splitting the length validation** so the
   shared lexical type accepts a long-but-valid FHIR decimal while `dicom.ParseDecimal` keeps the DS cap. Resolve the
   mechanism (a length-limit parameter on the shared parser, or a FHIR-side `ParseDecimal` that relaxes the cap) in
   Increment 5 expansion. Flag if the architect prefers a separate `fhir.Decimal` type instead.

5. **Bundle builders and reference helpers as hand-written per-release code — FLAG (intentional exception).** Increments
   9 and 10 hand-write the `Bundle` builders and the `Resolve`/`Contained`/`CheckReferenceIntegrity` helpers per release,
   because FHIR's `bdl-*` rules and the document/message first-resource rules are prose invariants, not expressible in the
   StructureDefinition the generator reads. Everything else under `fhir/r4`/`fhir/r5` is generated. This is the single
   deliberate exception to "all generated"; it should be in a clearly-named hand-written file (`bundle_builders.go`,
   `reference.go`) excluded from the byte-for-byte reproducibility diff. Confirm the architect is comfortable with this
   split rather than encoding the invariants as generator templates.

6. **Validation: generated descriptor vs reflection — RECOMMENDED (generated descriptor).** `Validate` and
   `MarshalSummary` need per-element metadata (required paths, choice groups, required bindings, `isSummary`). Recommend
   the generator emit a per-resource **validation descriptor** the engine consumes, so there is no reflection on the
   validation/serialisation hot path (PRD §9 NFRs: minimise allocations, no global mutable state). Confirm in Increment 11
   expansion; the alternative (struct-tag reflection) is simpler to write but slower and is the prototype's approach.

7. **Resource count band, not an exact 158 — RESOLVED.** The "~158 R5 resources" figure is approximate and varies by how
   backbone/abstract types are counted and by FHIR patch releases. The loader test asserts a **band** (`>= 140 && <=
   170`) and the **conformance statement** (`docs/conformance/fhir.md`) holds the exact enumerated list as the source of
   truth (PRD §6.1). Increment 14's validator gate, not a magic number, is the real conformance proof.

8. **Example-JSON round-trip comparison semantics — FLAG for Increment 13 expansion.** FHIR permits key reordering and
   whitespace, so "round-trip" for the HL7 example corpus is **structural** equality (decode → re-encode → decode →
   `reflect.DeepEqual`, or canonical-JSON comparison), not necessarily byte-stable, because the examples were not authored
   in the generator's canonical element order. The generator's *own* output is byte-stable (Increment 13's
   regenerate-and-diff); the *example* round-trip is structural. Pin the exact comparison in expansion.

### Notes for the executor (so the outlines do not drift)

- **Two reproducibility guarantees, do not conflate them.** (a) `go generate` reproduces the committed `fhir/r4`/`fhir/r5`
  tree byte-for-byte from the pinned bundle (Increment 13's `TestRegenerationByteForByte`). (b) The HL7 *example* corpus
  round-trips *structurally* (Increment 13's example test, Open question 8). Different tests, different comparison.
- **Generated vs hand-written commits.** Commit the generator source with its unit tests; commit the generated output
  separately as `chore(fhir/r5): regenerate ...`; commit vendored bundles separately again. The reproducibility diff
  excludes the hand-written `bundle_builders.go`/`reference.go` (Open question 5).
- **Match the committed shapes.** The generated `Reference`/`Identifier`/`CodeableConcept`/`CodeableReference` must match
  the M2 hand-written `fhir/r5/datatypes.go` field-for-field (Increment 6 goldens enforce this), so Increment 12's
  migration is a clean swap and `convert` does not change behaviour.
- **R5 first, R4 by rerun.** All generator logic is release-agnostic; R4 (Increment 15) should be a bundle swap plus
  release-difference reconciliation, not new generator branches. A release-specific code path is a smell — surface it.
