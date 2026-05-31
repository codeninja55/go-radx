# FHIR conformance statement

This document is the single source of truth for the FHIR scope of go-radx (PRD §6.1). It is a versioned conformance
statement in the spirit of an HL7 FHIR `CapabilityStatement`, written as prose: it declares which FHIR releases the
`fhir` package targets, which resource types are generated versus conformance-tested, which `Bundle` types and
serialization modes are supported, what validation the library performs, and what is explicitly deferred.

"Conformant" here means what the PRD defines it to mean: **100% conformant to an explicitly declared, versioned subset,
verified against reference validators** — not "implements all of FHIR." Growth of this subset is a deliberate change,
reviewed against this file, not an implementation accident.

The companion documents are the DICOM conformance statement (`./dicom.md`), the HL7 v2 conformance statement
(`./hl7v2.md`), and the FHIR package reference (`../reference/fhir.md`), which carries the full public API. Where this
statement and the reference docs disagree on *scope*, this statement wins; where they disagree on *API shape*, the PRD
§8.1 commitments win.

## Conformance metadata

| Field | Value |
|-------|-------|
| Standard | HL7 FHIR |
| Releases (v1) | R4 `4.0.1` and R5 `5.0.0` |
| go-radx package | `github.com/codeninja55/go-radx/fhir` |
| Statement version | 1.0 (tracks go-radx v1) |
| Conformance basis | Generated from the official HL7 `StructureDefinition` bundles for 4.0.1 and 5.0.0 |
| Reference validator | The official HL7 FHIR validator (merge-blocking in CI — PRD §11.1) |
| Serialization | JSON (normative); `_summary` modes; XML and YAML optional/deferred |

The version strings `4.0.1` and `5.0.0` are the exact FHIR release versions, not the go-radx library version. Each
generated resource and the package root expose them so a consumer can assert the release at runtime:

```go
package fhir

// Release identifies a generated FHIR release. The two v1 releases live in sibling packages.
type Release string

const (
    R4 Release = "4.0.1" // HL7 FHIR R4
    R5 Release = "5.0.0" // HL7 FHIR R5
)
```

## Overview and scope

The `fhir` package provides type-safe, idiomatic Go models for HL7 FHIR R4 (4.0.1) and R5 (5.0.0), generated from the
official HL7 `StructureDefinition` definitions. It is a ground-up re-foundation: the prior prototype's generator is
rewritten because its model was structurally wrong (see "What this statement fixes" below), while the regeneration
approach is kept (PRD §12). Generated code is never hand-edited.

Two release packages are produced and ship side by side, so a consumer choosing a release does so by import path, not by
a runtime flag:

```
fhir/         # release-neutral core: Resource interface, primitives, Decimal, validation engine, Bundle/summary logic
fhir/r4/      # generated R4 4.0.1 resources, backbone elements, datatypes, and required-binding enums
fhir/r5/      # generated R5 5.0.0 resources, backbone elements, datatypes, and required-binding enums
```

R4 and R5 are deliberately distinct type spaces. There is no implicit cross-release conversion in v1; a consumer who
needs to move data between releases does so explicitly. This mirrors the PRD §5.3 decision to generate R4 4.0.1
directly (it remains the most deployed release and US Core runs on it) rather than fold it into R4B the way
`fhir.resources` does from its v7.0.0 onward.

### In scope (v1)

- Typed Go models for **every resource and backbone element** of R4 4.0.1 and R5 5.0.0, all of which compile-test.
- The **radiology + clinical workflow resource set** (enumerated below), which is conformance-tested against the HL7
  FHIR validator and exercised by the cross-standard conversions and the walking skeleton.
- Choice types (`[x]`) as a sealed value interface with a `Value()` getter and `SetValueX()` setters that enforce
  mutual exclusion (PRD §8.1, §6.3).
- Required-strength value-set bindings as closed Go enums with a validating parse and an explicit unknown-code policy.
- Primitive types with the `_field` primitive-extension sibling round-tripped in JSON, including array alignment for
  repeating primitives.
- A lexical-preserving `Decimal` for the FHIR `decimal` primitive, shared with DICOM `DS`/`IS`.
- Cardinality and required-presence validation, `resourceType` integrity on polymorphic dispatch, canonical element
  ordering on serialize, and `contained` / `Bundle.entry.resource` polymorphic dispatch to concrete types.
- `OperationOutcome` modelling and `transaction` / `searchset` / `document` / `message` / `batch` / `collection` Bundle
  semantics, with intra-Bundle and contained reference-integrity resolution.
- `_summary` modes (`full` / `true` / `text` / `data` / `count`) for bandwidth-constrained serialization.
- JSON serialization as the normative wire format.

### Out of scope / deferred (v1)

These are architected-for but not implemented in v1 (PRD §3.2, §5.3). They are listed here so the boundary is explicit
and a consumer is never surprised:

- **SMART on FHIR** authorization (full implementation) — deferred.
- **US Core** (and any other Implementation Guide) profile validation — deferred.
- **FHIR Subscriptions** (the subscription framework and `subscription-notification` Bundle handling) — deferred.
- **FHIR R6** — the generator is designed so R6 can be added when normative, but R6 is not generated in v1.
- **FHIR R4B (4.3.0)** — deferred, open only as a future addition.
- **FHIR STU3** — out of scope.
- **XML and YAML serialization** — optional and deferred; JSON is the only normative wire format in v1.

## Generated versus conformance-tested resources

Two tiers of guarantee apply, and the distinction is load-bearing for what a consumer can rely on:

1. **Generated and compile-tested (all resources).** Every resource and backbone element of R4 4.0.1 and R5 5.0.0 is
   generated from the official `StructureDefinition` set and compiles. A consumer can construct, populate, marshal, and
   unmarshal any of them. Generated-only resources still benefit from the structural guarantees (choice types, primitive
   extensions, `Decimal`, required-binding enums) because those are produced by the generator uniformly.

2. **Conformance-tested (the workflow set).** A defined subset is additionally validated end-to-end against the HL7
   FHIR validator in CI and exercised by golden round-trip tests and the cross-standard conversions. For these resources
   go-radx asserts conformant JSON output, not merely "it compiles."

The conformance-tested set is the radiology and clinical workflow resources required to close the PRD §5.1 loop:

| Resource | Role in the workflow | Releases |
|----------|----------------------|----------|
| `ServiceRequest` | Imaging order (from HL7 v2 `ORM`/`OMG`) | R4, R5 |
| `ImagingStudy` | Study/series/instance grouping (from DICOM) | R4, R5 |
| `DiagnosticReport` | Radiology report (from DICOM SR / HL7 v2 `ORU`) | R4, R5 |
| `Observation` | Measurement/finding (from DICOM SR content / HL7 v2 `OBX`) | R4, R5 |
| `Patient` | Demographics (from HL7 v2 `PID` / `ADT`) | R4, R5 |
| `Encounter` | Visit context (from HL7 v2 `PV1` / `ADT`) | R4, R5 |
| `Bundle` | Container for the above | R4, R5 |
| `OperationOutcome` | Structured operation result | R4, R5 |

These are the resources the cross-standard conversions target. Every resource-producing converter is release-explicit,
so each name carries an `R4` or `R5` suffix and returns the matching release sub-package type (the `R5` forms shown):
`convert.DICOMToImagingStudyR5`, `convert.SRToDiagnosticReportR5` / `convert.DiagnosticReportToSRR5`,
`convert.ORUToDiagnosticReportR5`, `convert.ORMToServiceRequestR5`, `convert.ADTToPatientR5`, and
`convert.ADTToEncounterR5` (glossary naming rule 3), each with an `…R4` twin. Each conversion produces resources from
this set, and each is validated against the FHIR validator.

A resource moving from "generated" to "conformance-tested" is a reviewed change to this statement, not a silent code
change.

## Public API contract

The full API lives in `../reference/fhir.md`. This statement reproduces the load-bearing signatures so the conformance
behaviour is reviewable here. All signatures honour PRD §8.1 (notably: **no generic methods** — Go 1.26 forbids type
parameters on interface and struct methods, so all type-parameterised dispatch is via package-level generic functions)
and the glossary's canonical names. Signatures shown are release-neutral; release-specific resource types live in
`fhir/r4` and `fhir/r5` and implement the same `Resource` interface.

### Resource identity and type-safe access

Every resource implements a minimal interface carrying its discriminator. An internal `resourceType`→factory registry
returns the interface; package-level generic functions give the static type and — critically — verify the embedded
`resourceType` matches the requested type. This closes the prototype's unchecked `UnmarshalResource[T]` (FHIR-003).

```go
// Resource is the base unit of exchange; ResourceType returns the JSON discriminator value.
type Resource interface {
    ResourceType() string
}

// Unmarshal decodes FHIR JSON into T, verifying the embedded "resourceType" matches T.
// A Patient payload decoded as Observation returns an error (it does not silently succeed).
func Unmarshal[T Resource](data []byte) (T, error)

// As is a checked downcast from the Resource interface to a concrete type.
// Call site: patient, ok := fhir.As[*r5.Patient](entry.Resource)
func As[T Resource](r Resource) (T, bool)

// UnmarshalResource decodes FHIR JSON into the concrete type named by its "resourceType",
// using the internal registry. The returned Resource is the dynamic type, e.g. *r5.Patient.
func UnmarshalResource(data []byte) (Resource, error)
```

`Unmarshal[T]` and `UnmarshalResource` both reject a payload whose `resourceType` is absent, empty, or unknown to the
target release's registry (Codex FHIR-004). Marshalling always emits the constant `resourceType` for the concrete type
and never the zero-value empty string.

### Choice types

Each `[x]` choice group is one logical field exposed through a sealed value interface, a `Value()` getter, and per-type
`SetValueX()` setters. Setting one branch clears the siblings, so mutual exclusion holds at the API boundary and the
generated JSON authors exactly one suffixed property (`valueQuantity`, `deceasedBoolean`, etc.). This fixes the
prototype's emit-every-branch-as-required model (Codex FHIR-001) and the R4 `*any` choice fields (Codex FHIR-002).

```go
// ObservationValue is the sealed interface for Observation.value[x]. It is implemented only by
// NAMED FHIR datatype types — never the built-in string, bool, or int32, which cannot carry the
// unexported marker method. The complex branches are the generated datatype structs (Quantity,
// CodeableConcept, Range, Ratio, Period, ...); the primitive branches are the release primitive
// wrappers (FHIRString, FHIRBoolean, FHIRInteger, FHIRDateTime, FHIRTime, ...).
type ObservationValue interface{ isObservationValue() }

// Value returns the currently-set choice value, or (nil, false) if value[x] is absent.
func (o *Observation) Value() (ObservationValue, bool)

// SetValueQuantity sets value[x] to a Quantity and clears any other value[x] sibling.
func (o *Observation) SetValueQuantity(q Quantity)
```

A `string`-valued choice branch boxes the value in the release primitive wrapper type
(`r5.FHIRString`), which carries the marker method; the built-in `string` cannot. The
primitive-mapping (FHIR `string` → Go `string`) applies only to plain fields, not to choice
branches. Recovering the plain value from a wrapper is one explicit conversion, for example
`string(val)` after a type switch lands on `r5.FHIRString`.

### Required-binding enums

Only **required**-strength value-set bindings become closed Go enums (PRD §6.3; glossary "Value Set Binding"). Each one
is a defined string type, a const set, and a validating `ParseXxx`. Extensible, preferred, and example bindings stay a
`code` string, because closing them would reject conformant data. This closes the prototype's empty-`EnumValues` TODO
(Codex FHIR-013).

Required-binding enums are generated per release, so they live in the `fhir/r4` and `fhir/r5`
sub-packages alongside the resources that reference them (shown here under `r5`):

```go
package r5

// AdministrativeGender is the required binding on Patient.gender (and elsewhere).
type AdministrativeGender string

const (
    GenderMale    AdministrativeGender = "male"
    GenderFemale  AdministrativeGender = "female"
    GenderOther   AdministrativeGender = "other"
    GenderUnknown AdministrativeGender = "unknown"
)

// ParseAdministrativeGender validates s against the closed code set.
func ParseAdministrativeGender(s string) (AdministrativeGender, error)
```

The unknown-code policy is explicit and uniform: on **unmarshal**, an unknown code for a required binding is reported as
a validation issue (an `OperationOutcome` issue when validating; a returned error from `ParseXxx`), never silently
coerced. The raw value is preserved on the field so a strict consumer can inspect it; it is the validator that flags it.
This is binding safety at the JSON boundary, which is where Go can enforce it — not at every literal assignment, which
Go's type system cannot enforce (PRD §6.3).

### Lexical-preserving decimal

The FHIR `decimal` primitive maps to a `Decimal` that preserves the source lexical form, including trailing zeros and
exponent notation. It does not map to `float64` (Codex FHIR-009). It is the same `Decimal` type shared with DICOM
`DS`/`IS` (glossary "Decimal String / Integer String").

```go
// Decimal preserves the source lexical form of a FHIR decimal / DICOM DS/IS value.
// Conversion to a numeric type is explicit; there is no in-place arithmetic.
type Decimal struct{ /* preserves source lexical form */ }

// String returns the preserved lexical form exactly as parsed.
func (d Decimal) String() string

// Float64 returns the value as a float64. ok is false only when the lexical form has no finite
// float64 representation; a value that is representable but not exactly (such as 0.1) still
// returns ok == true. Callers that need full precision use String or BigFloat. This signature
// and semantics are identical across dicom.md, fhir.md, and the conformance statements.
func (d Decimal) Float64() (f float64, ok bool)

// Exact reports whether the lexical form converts to a float64 with no loss of precision. It is
// the exactness signal kept separate from Float64's ok return, so the bool on Float64 is never
// overloaded to mean two different things.
func (d Decimal) Exact() bool

// BigFloat returns a *big.Float with precision sufficient for the lexical form.
func (d Decimal) BigFloat() *big.Float

// MarshalJSON emits the preserved lexical form, not a re-rendered float.
func (d Decimal) MarshalJSON() ([]byte, error)
```

### Primitive extensions

Every FHIR primitive carries an optional `_field` sibling holding its `id` and `extension`s. The pair round-trips:
unmarshalling a value with a `_`-sibling reconstructs both; marshalling re-emits both. Repeating primitives use aligned
arrays with `null` placeholders so positions line up between the value array and the `_`-sibling array. The `_`-sibling
is generated **only** for true primitive elements, never for complex fields such as `contained`, `resource`, or `issue`
(Codex FHIR-005).

### Bundle and summary

`Bundle` is a resource, so like every resource it is generated per release: the type, its
`BundleType` binding, and its type-specific builders live in `fhir/r4` and `fhir/r5`. There is no
root `fhir.Bundle` and no single `NewBundle`. Each builder enforces the `bdl-*` invariants up front
rather than emitting an invalid bundle (Codex FHIR-010). The builders are shown for R5; R4 has the
same signatures in its own package.

```go
package r5

// BundleType is the required binding for Bundle.type; the permitted set differs by release.
type BundleType string

const (
    BundleDocument    BundleType = "document"
    BundleMessage     BundleType = "message"
    BundleTransaction BundleType = "transaction"
    BundleBatch       BundleType = "batch"
    BundleSearchSet   BundleType = "searchset"
    BundleCollection  BundleType = "collection"
    // history, transaction-response, batch-response also defined per release
)

// NewSearchSet builds a searchset Bundle; total is the only bundle type for which total is set.
func NewSearchSet(total int, entries ...SearchEntry) *Bundle

// NewTransaction builds a transaction Bundle; each entry carries a request (method + url).
func NewTransaction(entries ...TransactionEntry) (*Bundle, error)

// NewDocument builds a document Bundle whose first entry must be a Composition.
func NewDocument(composition fhir.Resource, entries ...DocumentEntry) (*Bundle, error)

// NewMessage builds a message Bundle whose first entry must be a MessageHeader.
func NewMessage(header fhir.Resource, entries ...MessageEntry) (*Bundle, error)
```

`_summary` serialization is release-agnostic machinery that operates over the `Resource` interface,
so `SummaryMode` and `MarshalSummary` live in the root `fhir` package:

```go
package fhir

// SummaryMode selects the _summary serialization view; the five modes follow the FHIR SummaryEnum.
type SummaryMode string

const (
    SummaryFull  SummaryMode = "false" // full resource (no filtering)
    SummaryTrue  SummaryMode = "true"  // elements flagged isSummary in the StructureDefinition
    SummaryText  SummaryMode = "text"  // narrative (text), id, meta, and mandatory elements only
    SummaryData  SummaryMode = "data"  // everything except the narrative (text)
    SummaryCount SummaryMode = "count" // intended for Bundle: emit total only, no entries
)

// MarshalSummary serializes a resource under the given summary mode. It returns an error for a nil
// resource rather than panicking (Codex FHIR-012).
func MarshalSummary(r Resource, mode SummaryMode) ([]byte, error)
```

## Bundle type support

The supported `Bundle.type` codes follow the release. The full code set is generated for both releases; the codes
go-radx implements *semantics* for in v1 (invariant enforcement and reference resolution) are the workflow-relevant
ones. `subscription-notification` exists only in R5 and is generated but not semantically supported, because
Subscriptions are deferred (PRD §3.2).

| `Bundle.type` | R4 4.0.1 | R5 5.0.0 | v1 semantics enforced |
|---------------|----------|----------|------------------------|
| `document` | yes | yes | first-entry `Composition`, fullUrl uniqueness |
| `message` | yes | yes | first-entry `MessageHeader`, fullUrl uniqueness |
| `transaction` | yes | yes | per-entry `request` required, method/url checks |
| `transaction-response` | yes | yes | per-entry `response` required |
| `batch` | yes | yes | per-entry `request` required |
| `batch-response` | yes | yes | per-entry `response` required |
| `searchset` | yes | yes | `total` and `entry.search` permitted |
| `collection` | yes | yes | no request/response/search; fullUrl uniqueness |
| `history` | yes | yes | `total` permitted |
| `subscription-notification` | no | yes (generated) | not enforced (Subscriptions deferred) |

R4 4.0.1 does not define `subscription-notification`; R5 5.0.0 adds it. This is the one `Bundle.type` difference between
the two release packages.

## Serialization

JSON is the normative wire format and is the only format guaranteed conformant in v1 (PRD §6.2). It targets the FHIR
JSON representation: a `resourceType` discriminator on every resource, choice fields authored with their type suffix,
primitive `_field` siblings, and canonical element ordering on output. Output is validated by the HL7 FHIR validator in
CI.

`_summary` filtering is driven by the `isSummary` flag carried on each element in the `StructureDefinition` (the same
flag `fhir.resources` surfaces as its `summary_element_property` marker). The generator records that flag per element so
`SummaryTrue` can include exactly the summary elements without a runtime spec lookup. The five modes match the FHIR
`_summary` parameter: `full`, `true`, `text`, `data`, and `count`.

XML and YAML are optional and deferred in v1. The API is shaped so they can be added as alternative codecs over the same
generated model without changing resource types; until then, calling an XML/YAML path returns a typed "format not
supported" error rather than producing non-conformant output (PRD §9.2 fail-closed rule).

## Validation performed

go-radx validation is **structural and binding-level**, not full Implementation-Guide profile validation. It checks what
can be checked from the base `StructureDefinition` for a release; it does not assert US Core or any other IG. Phases:

1. **`resourceType` integrity.** On unmarshal and on polymorphic dispatch (`contained`, `Bundle.entry.resource`), the
   discriminator is verified against the target type; mismatch and unknown types are errors, not silent successes
   (Codex FHIR-003, FHIR-004, FHIR-011).
2. **Cardinality and required presence.** Required (`min >= 1`) elements must be *present*, where presence is tracked
   separately from value. A required boolean that is validly `false`, or a required number that is validly `0`, is
   present and passes; this fixes the prototype's `reflect.IsZero` bug that read `false`/`0` as missing
   (Codex FHIR-007). Max cardinality is enforced by the generated type (a `0..1` element is a pointer/optional; a `*`
   element is a slice).
3. **Choice-type mutual exclusion.** At most one branch of a `[x]` group may be set; the typed setters enforce this at
   the boundary and the validator confirms it (Codex FHIR-001).
4. **Required value-set bindings.** Codes bound with `required` strength are validated against the closed enum; an
   unknown code is a validation issue (Codex FHIR-013). Extensible/preferred/example bindings are not closed and are not
   rejected.
5. **Primitive datatype validity.** Primitives are validated per the FHIR rules, not by loose regex: `date` rejects
   invalid calendar dates; `dateTime` requires a timezone offset when hours and minutes are present; `time` allows leap
   seconds; `decimal` preserves lexical precision (Codex FHIR-008, FHIR-009).
6. **Bundle invariants.** Per-type `bdl-*` constraints are checked before marshal/send: `total` only on
   searchset/history, `entry.search` only on searchset, document/message first-resource rules, transaction/batch
   request and response presence, and `fullUrl` uniqueness (Codex FHIR-010).
7. **Reference integrity (intra-Bundle and contained).** Internal references (`#id` to `contained`, `fullUrl`/relative
   references within a Bundle) are resolved; an unresolved internal reference is reported as a validation issue, not
   silently dropped.

Validation results are returned as an `*OperationOutcome` (or a Go error for parse-level failures), so a caller can map
issues to severities. Validation never panics on malformed input (PRD §9.3); a nil or structurally broken input yields
an error or an issue, not a crash.

The authoritative external check is the **HL7 FHIR validator**, merge-blocking in CI (PRD §11.1). go-radx's own
validation is the fast in-process gate; the official validator is the conformance gate.

## Behaviour and error model

- **Errors are values.** All fallible operations return an `error`; nothing panics on malformed external input
  (PRD §9.3). Parse and `resourceType`-mismatch failures return Go errors; semantic and binding failures collect into
  an `*OperationOutcome`.
- **Fail-closed.** A path that cannot perform the requested operation (an unsupported serialization format, a deferred
  feature) returns a typed error and writes nothing. It never no-ops and reports success (PRD §9.2(a)).
- **Truncation is failure.** A short read mid-value propagates `io.ErrUnexpectedEOF`; a clean record-boundary EOF is the
  only successful end of input (PRD §9.2(b)).
- **No silent skips.** A malformed `contained` resource or an unresolved internal reference is surfaced as an issue or
  error with the offending index, not silently treated as "not found" (Codex FHIR-011).
- **No PHI by default.** Validation issues and error messages name elements, paths, resource types, and codes — never
  patient data values (PRD §9.1; glossary naming rule 4). Verbosity that would surface a value is opt-in.
- **Concurrency.** Generated resource structs are plain data and are safe to read concurrently; they are not safe to
  mutate concurrently, and any mutating builder documents that it is not concurrent-safe or guards itself (Codex
  FHIR-015). The package holds no global mutable state (PRD §9.4).

## Worked examples

### Type-safe decode with discriminator check

```go
import (
    "github.com/codeninja55/go-radx/fhir"
    "github.com/codeninja55/go-radx/fhir/r5"
)

// Decoding a Patient payload as Patient succeeds.
patient, err := fhir.Unmarshal[*r5.Patient](payload)
if err != nil {
    return fmt.Errorf("decode patient: %w", err)
}

// Decoding the same payload as Observation fails — the resourceType does not match.
_, err = fhir.Unmarshal[*r5.Observation](payload)
// err is non-nil: `resourceType "Patient" does not match requested type Observation`
```

### Required-binding enum with explicit unknown-code policy

```go
g, err := r5.ParseAdministrativeGender("male")
if err != nil {
    return err // never reached for a valid code
}
patient.Gender = &g

// An unknown code is rejected at the boundary, not coerced to a default.
if _, err := r5.ParseAdministrativeGender("M"); err != nil {
    // err: `"M" is not a valid AdministrativeGender (required binding: male|female|other|unknown)`
}
```

### Choice type with mutual exclusion

```go
value, err := fhir.ParseDecimal("7.40")
if err != nil {
    return err
}
unit := "mmol/L"
obs := &r5.Observation{ /* status, code, subject ... */ }
obs.SetValueQuantity(r5.Quantity{Value: value, Unit: &unit})

// Setting another branch replaces the first; only one value[x] is ever authored. A string-valued
// branch is boxed in the release primitive wrapper, never the built-in string.
obs.SetValueString(r5.FHIRString("see attached"))

if v, ok := obs.Value(); ok {
    if s, ok := v.(r5.FHIRString); ok {
        _ = string(s) // the Quantity was cleared by the later setter
    }
}
```

### Summary-mode serialization

```go
// A full record over a constrained link is wasteful; request the summary view instead.
summary, err := fhir.MarshalSummary(study, fhir.SummaryTrue)
if err != nil {
    return err
}
// summary contains only the elements flagged isSummary in the ImagingStudy StructureDefinition.
```

### Building and validating a transaction Bundle

The transaction builder lives in the release package and enforces the `bdl-*` invariants as it
constructs, so an invalid bundle never reaches the marshaller. Marshalling uses the standard
library — each generated type implements `MarshalJSON`, so `json.Marshal` emits canonical FHIR JSON.

```go
// r5.NewTransaction enforces per-entry request presence and fullUrl uniqueness up front; an
// invariant violation is returned as an error rather than producing an invalid bundle. Each
// r5.TransactionEntry pairs a resource with its request (method + url).
b, err := r5.NewTransaction(
    r5.TransactionEntry{Resource: patient, Method: r5.HTTPPost, URL: "Patient"},
    r5.TransactionEntry{Resource: study, Method: r5.HTTPPost, URL: "ImagingStudy"},
)
if err != nil {
    return fmt.Errorf("build bundle: %w", err)
}

// fhir.Validate runs the same structural checks over the Resource interface for any release.
if oo := fhir.Validate(b); oo.HasErrors() {
    // oo is a *fhir.OperationOutcome; each issue names the path and severity, no PHI.
    return fmt.Errorf("invalid bundle: %w", oo.Error())
}
data, err := json.Marshal(b) // canonical element ordering; fullUrl uniqueness already checked
```

## What this statement fixes (re-foundation note)

The prototype FHIR subsystem was audited as not production-grade: the generator emitted every choice branch as required,
left nested backbone structs empty, produced structurally wrong primitive extensions, generated no required-binding
enums, mapped `decimal` to `float64`, validated required presence with `reflect.IsZero` (so a valid `false` read as
missing), did not enforce Bundle invariants, and shipped an `UnmarshalResource[T]` that never checked `resourceType`.
This statement commits the conformance behaviour that closes each of those defects (the parenthetical `Codex FHIR-NNN`
references above map to the audit findings). The generator is rewritten; the regeneration approach is retained;
generated code is never hand-edited (PRD §12).

## Conformance scope and limits (summary)

- **Releases:** R4 4.0.1 and R5 5.0.0 only. No STU3; no R4B; no R6 in v1.
- **Resources:** all generated and compile-tested; the radiology + clinical workflow set is conformance-tested against
  the HL7 FHIR validator.
- **Bundle types:** all generated; workflow-relevant types have enforced semantics; `subscription-notification`
  (R5-only) is generated but not semantically supported.
- **Serialization:** JSON normative with `_summary` modes; XML and YAML optional and deferred.
- **Validation:** structural, cardinality, required-presence, required-binding, primitive, Bundle-invariant, and
  intra-Bundle/contained reference integrity — not IG profile validation.
- **Deferred:** SMART on FHIR, US Core (and all IG profiles), Subscriptions, R6, R4B, STU3, XML/YAML serialization.
- **Authoritative gate:** the official HL7 FHIR validator in CI; go-radx's own validation is the in-process fast gate.

## See also

- FHIR package reference: `../reference/fhir.md`
- DICOM conformance statement: `./dicom.md`
- HL7 v2 conformance statement: `./hl7v2.md`
- Product requirements document: `../prd/go-radx-prd.md` (§5.3 FHIR release scope, §6 conformance model, §8.1 API)
- Ubiquitous language glossary: `../../UBIQUITOUS_LANGUAGE.md` (FHIR section, cross-standard collisions)
