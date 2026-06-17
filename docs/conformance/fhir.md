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

> **Implementation status: R4 and R5 generated.** Both release trees ship today, each generated end to end from its
> official `StructureDefinition` bundle. `fhir/r5` includes the radiology + clinical workflow resources (`ImagingStudy`,
> `DiagnosticReport`, `ServiceRequest`) and the complex datatypes (`Identifier`, `Reference`, `Coding`,
> `CodeableConcept`, `CodeableReference`) — these were briefly hand-written for the M2 walking skeleton and were retired
> in favour of the faithful generated supersets when the generator took over the full set. `fhir/r4` is generated from
> the vendored R4 4.0.1 bundle (see "Vendored definition bundles" below); the generator is data-driven from the
> `StructureDefinition`s, so the release differences fall out of the R4 definitions themselves — R4 has no
> `CodeableReference` (an R4 `ServiceRequest.code` is a `CodeableConcept`), R4 `ImagingStudy.modality` is a `Coding`
> (a `CodeableConcept` in R5), R4 `Encounter.class` is a single `Coding` (a list in R5), and R4 carries
> `Encounter.period` where R5 renamed it `actualPeriod`. The "every resource and backbone element … all of which
> compile-test" guarantee below holds for both releases.

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

### FHIR REST client and server

> **Implementation status: SHIPPED.** The FHIR REST **client** (`fhir/rest`) and the FHIR REST **server role**
> (`server.NewFHIRRole`, mounted with `server.WithFHIR`) are implemented over the generated R4 and R5 models, each fixed
> to one release per instance. The detailed conformance scope of both lives in the CLI/server conformance statement
> ([`./cli-server.md`](./cli-server.md) "FHIR REST client and server role"); this note records that they ship and that
> SMART on FHIR remains the documented deferral.

Cross-implementation conformance of the client is wired into CI: the `interop:fhir-hapi` leg (build tag `interop`)
provisions a HAPI FHIR server container pinned by digest (recorded in `tools/versions`) and drives capability
negotiation plus a create/read/search round-trip against it (`TestInteropHAPIServer` in `fhir/rest`), so the client's
request shape and parsing are proven against a reference implementation go-radx did not write. Setting
`RADX_FHIR_HAPI_BASE` to a reachable FHIR service base substitutes an external server for the container. The
client-to-`httptest` and client-to-go-radx-role round-trips remain standing correctness gates that need no container.

The client implements the FHIR HTTP interactions — `read`, `vread`, `create`, `update`, `patch`, `delete`, `history`,
type-level `search` (typed parameters, modifiers, single-level chaining, `_include`/`_revinclude`, and `Bundle.link`
paging), `transaction`/`batch`, conditional create/update with ETag concurrency, and `CapabilityStatement`
negotiation — sending and accepting `application/fhir+json` only. A non-2xx response whose body is an
`OperationOutcome` is mapped to a typed error the caller classifies by issue severity, consistent with the package's
`OperationOutcome` error model. The server role serves the conformance subset (`read`, `vread`, `history-instance`,
`create`, `update`, `patch`, `delete`, `search-type`, `transaction`, and the `$validate` operation over the workflow
resource set) over a pluggable versioned repository — every create stores version 1 with
`meta.versionId`/`meta.lastUpdated`; read, vread, and create emit `ETag`/`Last-Modified`; a create's `Location` (and a
transaction response entry's `response.location`/`response.etag`) names the created version
(`[type]/[id]/_history/[vid]`); a write's `If-Match` answers `412` on a stale version — validating inbound
resources with the release validator and answering every error with a release `OperationOutcome`. `update`
(`PUT [type]/[id]`) replaces the resource and bumps the version (`200`, or `201` for update-as-create, the FHIR and
HAPI default); `patch` (`PATCH [type]/[id]`) applies JSON Patch (RFC 6902), re-validates, and bumps the version
(`200`); `delete` (`DELETE [type]/[id]`) appends a deletion version (a later read is `410 Gone`, prior versions stay
vread-able, the history shows the deletion) and is idempotent. The conditional forms (`PUT`/`PATCH`/`DELETE
[type]?[search]`) resolve the criteria through the repository's search (zero/one/many → create-or-noop/apply/`412`),
exactly as selective as that repository's search. FHIRPath Patch is out of scope (JSON Patch only). A conditional
create (`If-None-Exist`, on the direct POST or a transaction entry) is rejected `400` with an `OperationOutcome`
rather than silently ignored — its matching semantics are deferred to the search work. The version store is
interaction-shaped, so update/patch/delete extend it by appending versions. Deep multi-hop search chaining beyond a
server's `SearchParameter` definitions and every `_include`/`_revinclude` form are the server's concern; the client
transmits whatever chained or include parameter the caller supplies rather than validating the chain itself.

The server role's `search-type` now carries the result surface beyond a bare match list. It pages with `Bundle.link`
(`self`/`next`/`prev`) over a `_count` page size and an `_offset` cursor the `next` link round-trips (`_count` defaults
to 50 and is clamped to 200); `total` reports the full match count across pages. `_include` and `_revinclude` resolve
one level deep into `entry.search.mode=include` entries (the matches themselves excluded, duplicates dropped), and a
one-hop chained parameter (`Observation?subject:Patient.name=...` or the typeless `subject.name=...`) resolves against
the configured `Repository` through a JSON-path `SearchParameter` registry covering the workflow references. The base
matching is the `Repository`'s — the in-memory default matches `_id` and reference parameters, a production
`Repository` matches any parameter. Out of scope and ignored (not errors): `:iterate` recursive include, `_has` reverse
chaining, and multi-hop chains.

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

> **Implementation status: R4 and R5 resources shipped; converters shipped for both releases.** The whole workflow set
> below ships in both releases, each generated from its `StructureDefinition` bundle: `ServiceRequest`, `ImagingStudy`,
> `DiagnosticReport`, `Observation`, `Patient`, `Encounter`, `Bundle`, and `OperationOutcome`, conformance-validated by
> the HL7 validator against 4.0.1 and 5.0.0. The workflow converters now exist for both releases: each forward converter
> (`convert.DICOMToImagingStudy…`, `convert.ORMToServiceRequest…`, `convert.SRToDiagnosticReport…`,
> `convert.ORUToDiagnosticReport…`, `convert.ADTToPatient…`, `convert.ADTToEncounter…`) and the SR/OBX reverse
> converters (`convert.DiagnosticReportToSR` / `…R4`, `convert.ObservationToContentItem` / `…R4`,
> `convert.ObservationToOBX` / `…R4`) have an `R4` and an `R5` form, and each twin's output is validated through its
> release validator (`r4.Validate` / `r5.Validate`). See [`./convert.md`](./convert.md) for the R4/R5 datatype
> differences each twin reconciles.

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
`convert.DICOMToImagingStudyR5`, `convert.SRToDiagnosticReportR5`, `convert.ORUToDiagnosticReportR5`,
`convert.ORMToServiceRequestR5`, `convert.ADTToPatientR5`, and `convert.ADTToEncounterR5` (glossary naming rule 3), each
with an `…R4` twin. The SR/OBX reverse converters keep the R5 form unsuffixed (`convert.DiagnosticReportToSR`,
`convert.ObservationToContentItem`, `convert.ObservationToOBX`) and add an `…R4`-suffixed R4 form
(`convert.DiagnosticReportToSRR4`, `convert.ObservationToContentItemR4`, `convert.ObservationToOBXR4`). Each conversion
produces resources from this set, and each is validated against the FHIR validator.

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

The generator stores a choice group as one suffixed pointer field per branch — `ValueQuantity
*Quantity`, `ValueString *FHIRString`, `ValueBoolean *FHIRBoolean`, ... — each tagged
`,omitempty`. There is no single bare untyped choice field, so the API offers no way to put a value
under one logical choice slot that could then hold two types at once; the construction path is the
typed setters, and each `SetXxx` first nils every sibling field before storing the new branch, so
through the setter API at most one branch is ever populated. The storage fields are exported
because faithful FHIR JSON requires the standard-library codec to see each suffixed key as a struct
field; the mutual-exclusion invariant is therefore enforced at the setter boundary rather than by
the type system. Bypassing the setters by writing two suffixed fields directly is a deliberate
misuse the codec cannot reject, and the at-most-one cardinality of a choice group is checked by
`Validate` once per group (it counts the non-nil suffixed storage fields). When the setters are used, at
most one storage field is non-nil and every field is `omitempty`, so marshalling authors exactly
one suffixed key. The `Value()` getter switches over the non-nil storage field and returns the
dereferenced branch value through the interface; an empty group returns `(nil, false)`. When two
FHIR element names collide and the choice stem is disambiguated (for example to `Value2`), the
getter, setters, and storage fields all follow that stem (`Value2()`, `SetValue2String`,
`Value2String`) so the group stays internally consistent and byte-stable.

The sealed interface is closed by an unexported marker method (`isObservationValue()`), emitted
on each branch type in the owning resource's file. A method on a same-package type — the generated
datatype struct (`Quantity`) or the release primitive wrapper (`FHIRString`) — declared in another
file is legal Go, so the markers live beside the choice they seal without a central registry. A
type outside the package, and any built-in scalar, can never gain the unexported method, so the
branch set is closed at compile time.

The release primitive wrappers are generated once per release into `fhir/r5/primitives.go`, one
distinct named type per FHIR primitive code: `FHIRBoolean`, `FHIRInteger`, `FHIRPositiveInt`,
`FHIRUnsignedInt`, `FHIRInteger64`, `FHIRDecimal`, `FHIRString`, `FHIRCode`, `FHIRID`,
`FHIRMarkdown`, `FHIRURI`, `FHIRURL`, `FHIRCanonical`, `FHIROID`, `FHIRUUID`, `FHIRBase64Binary`,
`FHIRInstant`, `FHIRDateTime`, `FHIRDate`, `FHIRTime`, and `FHIRXHTML`. Two codes that share a Go
scalar (`string`, `code`, `uri`) still get distinct wrappers, because the choice suffix and storage
field differ and a `Value()` type switch must recover which branch was set. The string, boolean,
and 32-bit integer wrappers marshal natively as the bare FHIR value (a JSON string, number, or
boolean — never a wrapping object). Two wrappers carry a generated `MarshalJSON`/`UnmarshalJSON`
pair where the FHIR R5 wire form is not the Go-native one: `FHIRDecimal` is a defined type over
`fhir.Decimal` whose methods delegate to `fhir.Decimal` so the lexical form (trailing zeros,
precision) survives the round trip (Codex FHIR-009); `FHIRInteger64` renders as a quoted JSON
string (`"9007199254740993"`), the FHIR R5 representation that keeps a 64-bit value exact past JSON
parsers that decode numbers as `float64`.

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

The unknown-code policy is explicit and uniform. On **unmarshal**, the generated enum's `UnmarshalJSON` applies the
strict rule by default: an out-of-set code is rejected with `ErrUnknownCode`, wrapped with the binding name and the
offending code token (never a patient value), so a non-conformant payload fails closed rather than silently populating a
required field. `ParseXxx` always applies the same strict rule regardless of any decode mode. Lenient retention is the
opt-in alternative, threaded explicitly as `fhir.DecodeLenient` through the boundary helper `fhir.DecodeCode`: under
lenient decode an out-of-set code is retained verbatim for `Validate` to surface as an `OperationOutcome` issue, so a
consumer ingesting partially-conformant data from the wild can inspect rather than reject. The mode is a value, not a
process-wide toggle, so two concurrent decodes never race on a shared policy. This is binding safety at the JSON
boundary, which is where Go can enforce it — not at every literal assignment, which Go's type system cannot enforce
(PRD §6.3).

### Terminology loading and scope boundary

The generator runs **no terminology server**. It enumerates a required binding's closed code set from the vendored,
checksum-pinned definition bundle alone (`valuesets.json`, which carries both the `ValueSet` and the `CodeSystem`
resources). A value set is **enumerable** when every `compose.include` either inlines its concepts
(`compose.include.concept`, an *extensional* definition) or names a `CodeSystem` that is vendored in the bundle with
complete content, in which case the codes come from that system's concept tree (walked depth-first so a hierarchical
code system enumerates every code). A `compose.exclude` rule removes its inlined concepts from the result, so a value
set that includes a whole system and excludes a few codes still enumerates correctly.

A value set is **not enumerable** — and the generator emits a **documented not-inlined boundary** rather than a
silently-empty const set — in three cases:

- **Intensional (filter-defined).** A `compose.include.filter` (a property/op/value rule such as `concept is-a <root>`)
  defines membership by a query a terminology server resolves, not by a literal code list. The loader captures the
  filter so the enum stage can name exactly which intensional rule made the set non-enumerable.
- **External terminology.** A `compose.include.system` that names a code system not vendored in the bundle — LOINC
  (`http://loinc.org`), SNOMED CT, UCUM (`http://unitsofmeasure.org`), an IETF BCP registry (`urn:ietf:bcp:13`,
  `urn:ietf:bcp:47`), or an ISO registry (`urn:iso:std:iso:4217`) — cannot be enumerated offline.
- **Value-set composition.** A `compose.include.valueSet` that composes another value set is followed by a terminology
  server, not the generator.

A not-inlined binding is emitted as a plain `code` string type (a `string` alias) carrying a godoc note that names the
reason (for example *"draws from code system `http://unitsofmeasure.org`, not vendored in the bundle"*), so the
terminology-scope boundary is explicit in the generated source. A field bound to such a value set keeps its plain
`code` string type and accepts any code on decode; the authoritative membership check for these codes is the official
HL7 FHIR validator in CI (PRD §11.1), not the Go type system.

This boundary is enforced as an invariant, not left to chance: an enumerable enum is **never** emitted with an empty
const set. A binding the resolver reports inlineable but that yields no codes is downgraded to a documented not-inlined
boundary, and a generator guard test (`TestNoEmptyRequiredBindingEnum`) fails the build if any required-binding enum
would ship enumerable-but-empty. For the vendored R5 5.0.0 bundle this yields closed enums for the large majority of
required bindings and a small set of documented not-inlined boundaries for the external-terminology and
value-set-composition cases above.

The M2 walking-skeleton types (`ServiceRequest`, `DiagnosticReport`, `ImagingStudy`, and the `Identifier`, `Reference`,
`Coding`, `CodeableConcept`, `CodeableReference` datatypes) were briefly hand-written under `fhir/r5` as minimal slices
while the bulk generator excluded their names to avoid same-package duplicate-type collisions. The generator now owns
them: it emits the faithful generated supersets (every field of the R5 `StructureDefinition`, choice fields through the
sealed setters, required-binding `code` fields as closed enums, and the base members embedded), the hand-written files
and the generator exclusion are gone, and the converter and the walking-skeleton end-to-end test read the generated
shapes. The required-binding code fields (`status`, `intent`) are now typed `*RequestStatus` / `*RequestIntent` /
`*DiagnosticReportStatus` / `*ImagingStudyStatus`, so an out-of-set code is rejected at the JSON boundary, completing
FHIR-013 enforcement on these types. The hand-written `Bundle` builders and the reference-resolution / integrity helpers
remain the single documented hand-written exception (the `bdl-*` prose invariants the `StructureDefinition` does not
express); they extend the generated types and live in clearly-named files (`bundle_builders.go`,
`reference_integrity.go`) outside the generated, byte-for-byte-reproducible file set.

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

The shared sibling type lives in the root package and is referenced by every generated release primitive:

```go
package fhir

// PrimitiveElement carries the id and extensions a FHIR primitive may hold
// alongside its value, serialised under the "_field" key.
type PrimitiveElement struct {
    ID        *string         `json:"id,omitempty"`
    Extension json.RawMessage `json:"extension,omitempty"`
}
```

For each true primitive element the generator emits a paired companion field next to the value field. A **scalar**
primitive `Status *string` gains `StatusElement *fhir.PrimitiveElement` tagged `json:"_status,omitempty"`; the value and
its companion round-trip through ordinary struct tags. A **repeating** primitive `Given []string` gains
`GivenElement []*fhir.PrimitiveElement` and a generated `MarshalJSON`/`UnmarshalJSON` pair that null-aligns the two
arrays. On marshal, `["Jane","Q"]` with an id on the second element only serialises as `"given":["Jane","Q"]` and
`"_given":[null,{"id":"x"}]`, the `null` keeping the companion array index-aligned with the value array; the `_given`
key is omitted entirely when no position carries an id or extension. On unmarshal, the `null` placeholder is restored as
a `nil` companion entry so the alignment survives both directions. The Go field identifier never contains the underscore
— the underscore appears only on the JSON wire key.

The `_field` companion is emitted only for an element whose single type is a FHIR primitive. A complex field
(`HumanName.period`, a `Period`), a `[x]` choice (whose branches box their own primitives under the suffixed wire key),
a backbone, and a `contentReference` recursion boundary all get no companion, so `contained`, `resource`, and
`OperationOutcome.issue` are never given a spurious `_field` sibling (Codex FHIR-005).

The `_field` companion fold preserves canonical element ordering: the value fields are encoded in their
struct-declared order and the `_field` siblings are appended after them, rather than routed through a map (which would
re-sort every key alphabetically).

> **Limitation (repeating extension-only positions).** A repeating primitive's value slice is a plain Go slice
> (`[]string`, `[]fhir.Decimal`), so a value-side JSON `null` — the FHIR encoding of a position that carries only an
> extension and no value, `"given":[null,"Q"]` — decodes to the Go zero value and re-marshals as that zero value rather
> than `null`. The `_field` extension data at that position still round-trips and stays index-aligned; only the
> value-side `null` placeholder is lost. Nullable value elements (and the presence semantics that go with them) are an
> Increment 6 concern, since they change every repeating primitive's value type.

### Bundle and summary

`Bundle` is a resource, so like every resource it is generated per release: the type, its
`BundleType` binding, and its type-specific builders live in `fhir/r4` and `fhir/r5`. There is no
root `fhir.Bundle` and no single `NewBundle`. Each builder enforces the `bdl-*` invariants up front
rather than emitting an invalid bundle (Codex FHIR-010). The builders are shown for R5; R4 has the
same signatures in its own package.

```go
package r5

// BundleType is the required binding for Bundle.type; the generated constants are the
// full code set (BundleTypeDocument, BundleTypeMessage, BundleTypeTransaction,
// BundleTypeBatch, BundleTypeSearchset, BundleTypeCollection, BundleTypeHistory,
// BundleTypeTransactionResponse, BundleTypeBatchResponse, and the R5-only
// BundleTypeSubscriptionNotification).
type BundleType string

// ErrInvalidBundle is the sentinel every builder rejection wraps; a caller matches it
// with errors.Is. The wrapped message names the offending entry index and the rule,
// never patient data.

// NewSearchSet builds a searchset Bundle. total is set because searchset is one of only
// two types (with history) for which total is meaningful; a negative total is rejected.
// Each SearchEntry may carry search metadata, which is valid only in a searchset.
func NewSearchSet(total int32, entries ...SearchEntry) (*Bundle, error)

// NewTransaction builds a transaction Bundle; every entry must carry an HTTP verb and a
// non-empty URL. NewBatch enforces the same per-entry request invariant.
func NewTransaction(entries ...TransactionEntry) (*Bundle, error)
func NewBatch(entries ...TransactionEntry) (*Bundle, error)

// NewDocument builds a document Bundle whose first entry must be a Composition.
func NewDocument(composition fhir.Resource, entries ...DocumentEntry) (*Bundle, error)

// NewMessage builds a message Bundle whose first entry must be a MessageHeader.
func NewMessage(header fhir.Resource, entries ...MessageEntry) (*Bundle, error)

// NewCollection builds a collection Bundle: an unconstrained resource set with no total,
// request, response, or search metadata.
func NewCollection(entries ...CollectionEntry) (*Bundle, error)
```

Every builder enforces its per-type `bdl-*` invariants up front and returns an error rather than
emitting an invalid bundle (Codex FHIR-010): `total` is set only by `NewSearchSet`; `entry.search`
metadata lives on `SearchEntry` so it cannot reach another type; `entry.request` is mandatory in a
transaction or batch entry and `entry.response` is unrepresentable in a transaction built through the
builder; a document's first entry must be a `Composition` and a message's a `MessageHeader`; and
`fullUrl` values are unique across the bundle. A nil or typed-nil first resource is rejected, never
dereferenced.

A bundle is **build-once-then-immutable** (Codex FHIR-015): a builder constructs, validates, and
returns a fresh bundle with no shared mutable builder state and no mutex. The returned bundle is plain
data safe to read concurrently; a single owner holds it until it is published. Mutating a bundle after
a builder returns it bypasses the invariant checks, so a caller that must change one builds a new
bundle rather than editing fields in place. The prototype's mutex-guarded mutable helper papered over a
concurrency bug instead of fixing it; the single-builder rule replaces it.

Reference resolution and integrity are hand-written per-release helpers on the generated types,
returning a release `*OperationOutcome` for an integrity walk and a Go error for a malformed contained
resource:

```go
package r5

// Resolve looks up the resource a reference points at within the Bundle: an entry whose
// fullUrl equals ref, or a "#id" fragment against the entries' contained resources. An
// external absolute URL that matches no entry fullUrl returns ok=false (Resolve never
// dereferences the network).
func (b *Bundle) Resolve(ref string) (fhir.Resource, bool)

// ResolveContained returns the contained resource whose id matches id. It returns an
// aggregate error naming the offending index when a contained slot is malformed (a nil
// or id-less contained resource), never a silent miss (Codex FHIR-011). It is named
// ResolveContained because DomainResource already carries the Contained field.
func (d *DomainResource) ResolveContained(id string) (fhir.Resource, error)

// CheckReferenceIntegrity walks every Reference in the Bundle (and contained resources)
// and reports each dangling local reference and each malformed contained resource as an
// issue, aggregating them into one OperationOutcome (Codex FHIR-011). An external
// absolute URL is not flagged.
func (b *Bundle) CheckReferenceIntegrity() *OperationOutcome
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
    SummaryCount SummaryMode = "count" // intended for Bundle: emit total and mandatory elements, no entries
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
| `transaction` | yes | yes | per-entry `request` required; verb/resource equivalence; method/url checks |
| `transaction-response` | yes | yes | per-entry `response` required |
| `batch` | yes | yes | per-entry `request` required; verb/resource equivalence |
| `batch-response` | yes | yes | per-entry `response` required |
| `searchset` | yes | yes | `total` and `entry.search` permitted; content entry requires a resource |
| `collection` | yes | yes | no request/response/search; fullUrl uniqueness; content entry requires a resource |
| `history` | yes | yes | `total` permitted |
| `subscription-notification` | no | yes (generated) | not enforced (Subscriptions deferred) |

The verb/resource equivalence (bdl-3c/bdl-3d) the builders enforce is: a `POST`, `PUT`, or `PATCH`
transaction/batch entry must carry a resource, and a `GET`, `HEAD`, or `DELETE` entry must not. R4 4.0.1
does not define `subscription-notification`; R5 5.0.0 adds it, the one `Bundle.type` difference between
the two release packages.

Some `bdl-*` rules are checked by `Validate` over a decoded bundle rather than by the builder. `Validate`
honours the `meta.versionId` exception to fullUrl uniqueness (bdl-7's clause that two entries may share a
fullUrl when they carry different `meta.versionId`, and the history-bundle exemption, bdl-8): its
uniqueness key is `(fullUrl, versionId)` and a history bundle is exempt entirely, so a valid versioned or
history bundle is not falsely rejected. The builders enforce the conservative, workflow-relevant subset —
flat fullUrl uniqueness, first-entry type, request/response presence, and verb/resource equivalence — over a
bundle they construct fresh (where no version distinguishes two entries). A document Bundle's required
`identifier` and `timestamp` (bdl-9, bdl-10) and the `searchset` `self` link (bdl-18) remain deferred.

## Serialization

JSON is the normative wire format and is the only format guaranteed conformant in v1 (PRD §6.2). It targets the FHIR
JSON representation: a `resourceType` discriminator on every resource, choice fields authored with their type suffix,
primitive `_field` siblings, and canonical element ordering on output. Output is validated by the HL7 FHIR validator in
CI.

### Canonical element ordering

A resource marshals its value fields in **`StructureDefinition` snapshot order** — the order the elements appear
in the definition the generator read — because the generated struct declares them in that order and `encoding/json`
emits struct fields in declaration order. The discriminator `resourceType` is written first. The primitive `_field`
extension
siblings are then folded in after the value object's own keys rather than interleaved next to each value: the marshaller
appends each non-empty sibling to the already-encoded object (`AppendSiblings`), which preserves the value-field order a
map round-trip would destroy. A nested object folds its own siblings at the end of that object. go-radx's canonical form
is therefore "value fields in snapshot order, then the scalar `_field` siblings"; this is the form `MarshalJSON`
produces and the form against which byte-stable round-trip is defined.

### Round-trip

Decoding canonical FHIR JSON and re-encoding it reproduces the input byte-for-byte. The generated `UnmarshalJSON` lifts
each `_field` sibling out of the object into its companion field (`SplitRawObject`/`TakeRawField`), decodes the residual
keys into the value struct, and restores a repeating primitive's null-aligned sibling array index-for-index; the
matching `MarshalJSON` re-emits the same canonical order. A repeating primitive whose extensions are sparse keeps its
positional
`null` placeholders so the value and `_field` arrays stay index-aligned. "Canonical input" means input already in
go-radx's canonical form (siblings trailing the value keys); a document that interleaves a `_field` sibling immediately
after its value is still decoded correctly but re-encodes in the canonical trailing form.

A field typed as the abstract `Resource` interface — `Bundle.entry.resource`, `Bundle.issues`,
`Bundle.entry.response.outcome`, `Parameters.parameter.resource`, and `DomainResource.contained` — cannot be decoded
by the standard codec, which has no concrete type to unmarshal a resource object into. The generated `UnmarshalJSON`
lifts each such key out of the raw object (the same `SplitRawObject`/`TakeRawField` mechanism) and routes its bytes
through
`fhir.UnmarshalResource` (peek `resourceType`, dispatch via the factory registry), so the value behind the interface is
the correct concrete type — recoverable with `fhir.As[T]` or a type switch — and a multi-resource-type `searchset`
Bundle round-trips. A repeating resource field (`contained`) routes through `fhir.UnmarshalResourceSlice`, which fails
the whole decode (it never returns a partial slice) if any element's `resourceType` is absent, empty, or unregistered;
an absent or unknown discriminator on any such field is an `ErrUnknownResourceType`, never a panic.

### Summary modes

`_summary` filtering is driven by the `isSummary` flag carried on each element in the `StructureDefinition` (the same
flag `fhir.resources` surfaces as its `summary_element_property` marker). The generator records that flag — with
each top-level element's mandatory (`min >= 1`) and modifier status, the narrative element, and the Bundle count element
— into a per-resource **summary descriptor** the release package registers at init time, so `MarshalSummary` filters
from generated metadata without a runtime `StructureDefinition` lookup or reflection over the resource. A choice (`[x]`)
group contributes one descriptor entry per suffixed branch key, all sharing the group's flags.

`MarshalSummary(r, mode)` marshals the resource in full and then drops the top-level elements the mode excludes,
preserving the canonical element order the resource's `MarshalJSON` produced (the filter walks the encoded object
key-by-key and re-emits the kept keys in place; a `_field` sibling is kept exactly when its value key is kept). The five
modes match the FHIR `_summary` parameter:

| Mode | Constant | Elements emitted |
|------|----------|------------------|
| `false` | `SummaryFull` | the full resource (identity; no filtering, no tag) |
| `true` | `SummaryTrue` | the `isSummary`, mandatory, and modifier elements, plus `id`/`meta` |
| `text` | `SummaryText` | the narrative (`text`), mandatory elements, plus `id`/`meta` |
| `data` | `SummaryData` | everything except the narrative (`text`) |
| `count` | `SummaryCount` | the Bundle `total` and the mandatory elements, plus `id`/`meta`; entries are dropped |

Every mode except `SummaryFull` always retains the infrastructure keys (`resourceType`, `id`, `meta`) so a summarised
resource stays a valid, identifiable resource. When a mode drops any element it sets the FHIR `SUBSETTED` tag on
`meta.tag` so a consumer can tell the payload is a partial view; the tag is spliced in without re-sorting the other keys
(`meta` is inserted in its canonical slot after `id` when the resource carried none). A nil resource (a nil interface or
a typed-nil pointer) returns `ErrNilResource` rather than panicking (Codex FHIR-012). A resource whose type has no
registered summary descriptor is returned in full rather than guessing which elements to drop.

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

### The validation engine and its descriptor

`fhir.Validate(r Resource) *fhir.OperationOutcome` is release-agnostic: it validates any release's resource through the
root `Resource` interface and returns a root-package `OperationOutcome` (a lightweight in-process result, distinct from
the on-the-wire `r5.OperationOutcome` resource). It reports **every** issue it finds in one pass rather than stopping at
the first, so a caller sees the full set, and it folds issues from all phases into one outcome.

The engine is **data-driven by a generated per-resource validation descriptor**, not by call-time reflection over the
resource. The generator emits one descriptor per resource into `fhir/r5/validation_descriptors.go`; each release package
registers its descriptors with the root engine at package-init time, keyed by `resourceType`, exactly as the
`resourceType`→factory registry is populated. A descriptor carries the resource's required elements, its choice (`[x]`)
groups, and its required-binding code fields as **typed closures over the concrete resource** — for example a Patient
descriptor's required check is `func(r) []string` that asserts `*r5.Patient` once and tests `v.Active != nil`. There is
no metadata reflection on the validation path: which elements are required, which form a choice group, and which carry a
required binding are all resolved at generate time. The descriptor file regenerates byte-for-byte like the rest of the
tree (`TestRegenerationByteForByte`, `gen:verify`). A resource whose type has no registered descriptor is reported as a
single `warning`-severity issue rather than silently passing, so a coverage gap is visible, not a false "valid".

Presence is tracked by the concrete field being non-nil (a single-valued pointer) or a non-empty slice (a repeating
element), never by the value being non-zero. This is the behavioural half of the required-presence fix (Codex FHIR-007):
a required boolean that is validly `false`, or a required number that is validly `0`, is a non-nil pointer and is
**present**, so it is never reported missing. Choice mutual exclusion counts the non-nil suffixed storage fields of each
`[x]` group and flags a group with more than one set, which catches a direct two-field write that bypassed the
mutually-exclusive setters (Codex FHIR-001) — the hard guarantee the setters enforce constructively but a raw struct
literal can violate.

The Bundle `bdl-*` invariants and intra-Bundle/contained reference integrity are not expressible from the
`StructureDefinition`, so — like the Bundle builders and the reference helpers — they are hand-written per release
(`fhir/r5/validate.go`) and composed into the Bundle descriptor's extra-check hook. The builders enforce the
constructive subset up front when a Bundle is built in-process; `Validate` is the gate for a Bundle that arrived over
the wire, checking `total` only on searchset/history (bdl-1), `entry.search` only on a searchset (bdl-2), the
document/message first-entry type (bdl-3), transaction/batch request and response-bundle response presence (bdl-3a/3b),
and `fullUrl`
uniqueness (bdl-7), then walking references through `CheckReferenceIntegrity`.

**Scope boundary (v1).** The structural descriptor covers a resource's **top-level** elements: top-level required
presence, top-level choice groups, and top-level required-binding codes. Nested-backbone cardinality, choice, and
binding checks are deferred (a backbone's own required elements are not yet walked); full primitive lexical validation
(`date`/`dateTime`/`time` calendar and offset rules, Codex FHIR-008) is also deferred — `decimal` already preserves
lexical precision through `fhir.Decimal` (FHIR-009), and required-binding codes are validated against their closed enum.
The HL7 FHIR validator (the conformance gate) covers the deferred depth; `Validate` is the fast in-process structural
gate for the common, top-level errors. The workflow resources (`ServiceRequest`, `DiagnosticReport`, `ImagingStudy`)
carry generated descriptors like every other generated resource, so their required elements and required-binding codes
(`status`, `intent`) are validated rather than silently skipped.

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

### Fuzzing posture

The untrusted-JSON surfaces — registry-dispatched decode and the in-process validation gate — are exercised by Go
native fuzz targets that guard the hostile-input guarantees above: on arbitrary, truncated, wrong-typed, or deeply
nested input the decoder and validator must return rather than panic (PRD §9.3), and no validation issue may echo a
patient value (PRD §9.1). The targets live in `fhir/r5/fuzz_test.go`, in the release package because that is where the
factory and validation-descriptor registries are populated, so the fuzzer exercises the production decode of registered
resources:

- `FuzzUnmarshalResource` drives `fhir.UnmarshalResource` over fuzzed bytes. It asserts the survival property (no
  panic), that a successful decode yields a non-empty `resourceType`, and that re-feeding a once-decoded resource's
  bytes never panics the polymorphic Bundle/`contained` decode path.
- `FuzzValidate` decodes then runs `fhir.Validate` over the result, asserting the validator never panics and that no
  issue diagnostic or expression contains one of the synthetic patient-data sentinels the seed corpus carries.
- `FuzzValidateTypedResource` validates a fuzzer-shaped typed `Patient` (including arbitrary `gender` codes), exercising
  the required/choice/binding closures over field combinations a decode might not reach. The never-panic property is the
  contract; the PHI-no-leak property is proven by `FuzzValidate` and the validation unit tests, not re-checked here,
  because a fuzzer-chosen field value can be any substring of a path or code.

Each target ships a version-controlled seed corpus under `fhir/r5/testdata/fuzz/<FuzzName>/`: clean workflow instances
alongside hostile regression seeds (truncated, two-choice-branch, unknown-`resourceType`, empty), so a normal `go test`
replays both as regression cases without a fuzzing build. The targets additionally seed at runtime from the synthetic
clean corpus (`testdata/fhir/r5`) and the malformed corpus (`testdata/fhir/malformed`), both PHI-free. The targets run
in the CI fuzz job (`mise run fuzz`), each `timeout`-wrapped at a budget above its `-fuzztime` so a hang fails the
build.

The **truncation contract** is asserted explicitly, not only fuzzed: `TestUnmarshalTruncatedYieldsUnexpectedEOF` (root
package) and `TestCorpusTruncationMapsToUnexpectedEOF` (over the real corpus instances) verify that a valid payload cut
short at any byte surfaces `io.ErrUnexpectedEOF` (matchable with `errors.Is`), distinct from a mid-buffer structural
syntax fault. The decoder folds the standard library's "unexpected end of JSON input" syntax error and the decoder
path's `io.EOF`/`io.ErrUnexpectedEOF` to the `io.ErrUnexpectedEOF` sentinel at every decode boundary while preserving
the original diagnostic in the error chain.

### Performance baseline

The FHIR-JSON decode, validate, and summary hot paths carry a committed benchmark baseline so a regression is a visible,
reviewable change rather than silent drift (PRD §9.3, minimise allocations in hot paths). The default-build baseline,
[`benchmarks/fhir-baseline.txt`](benchmarks/fhir-baseline.txt), is the pure-Go build (`CGO_ENABLED=0`). It covers
`BenchmarkMarshalSearchSetBundle` and `BenchmarkUnmarshalSearchSetBundle` (a 200-entry searchset Bundle — a realistic
worklist page — exercising the per-entry polymorphic marshal and the `resourceType`-peek-then-registry-dispatch decode
at scale), `BenchmarkValidateWorkflowSet` (the in-process gate over the full workflow set), and
`BenchmarkMarshalSummary` across the five `_summary` modes. Every subject is a go-radx synthetic, PHI-free corpus
instance.

The benchmark code is run once per CI build (`mise run bench`, `-benchtime=1x`) so a benchmark that no longer compiles
or panics fails the build. Regenerate the baseline with the command recorded in the file's header, and compare a
candidate run with `benchstat benchmarks/fhir-baseline.txt <candidate>.txt`.

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
obs.SetValueQuantity(r5.Quantity{Value: &value, Unit: &unit})

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

## Vendored definition bundles

The generator reads pinned, checksum-verified copies of the official HL7 FHIR definition bundles, committed under
`fhir/internal/gen/testdata/definitions/`. Both supported releases are vendored: R5 `5.0.0` (`buildId` 2aecd53, 162
resources) under `r5/`, and R4 `4.0.1` (`buildId` 9346c8cc45, 148 resources) under `r4/`. Each release directory holds
the three bundle files the generator reads (`profiles-types.json`, `profiles-resources.json`, `valuesets.json`), a
`SHA256SUMS` manifest recording a SHA-256 over each file's on-disk bytes, and a `SOURCE.md` recording the download
URL, version, build, and license. Both releases are published by HL7 under the Creative Commons CC0 1.0 public-domain
dedication; external terminologies referenced by their value sets (SNOMED CT, LOINC, DICOM, UCUM) carry their own
third-party terms and are referenced by URL rather than redistributed as code lists.

The bundles are committed verbatim as binary reference data (`.gitattributes` sets `-text` so git performs no EOL
normalisation that would break the pin); they are not stored in git-lfs. A standalone CI step runs
`shasum -a 256 -c SHA256SUMS` over each release directory, so a drifted or corrupted bundle is a hard CI error
independent of the Go suite, and the loader re-verifies the same manifest fail-closed before parsing. Refreshing a
bundle is a deliberate, reviewed change run through the refresh-only mise tasks (`mise run fhir:refresh-r5` /
`fhir:refresh-r4`), which re-download from the canonical HL7 archive, verify the release version, and re-record the
manifest; refreshing never happens at generate time. Each bundle drives its generated release package: the R5 bundle
produces `fhir/r5` and the R4 bundle produces `fhir/r4`, both reproduced byte-for-byte by the regeneration gate
(`TestRegenerationByteForByte` runs over both releases) so neither tree can be hand-edited or drift from its bundle.

## Generator pipeline

The generated FHIR packages are produced by a build-time generator under `fhir/internal/gen`, invoked by `go generate`
and never part of the runtime dependency graph. The generator is staged as a pipeline with single-responsibility
boundaries, each stage gated by its own tests:

1. **Loader** (`fhir/internal/gen/loader`) reads the vendored, checksum-pinned HL7 FHIR definition bundle
   (`profiles-types.json`, `profiles-resources.json`, `valuesets.json`), verifies every file against the committed
   `SHA256SUMS` manifest before parsing, decodes the `StructureDefinition` / `ValueSet` / `CodeSystem` entries into
   raw records, and indexes them by canonical URL and by name. It fails closed: a checksum mismatch, a missing
   required file, or a malformed entry is a typed error, never a "regenerate anyway." The loader never reaches the
   network, so generation is reproducible from the pinned input alone. **Terminology loading:** the loader decodes the
   `ValueSet` and `CodeSystem` resources the required-binding enums need, capturing each `compose.include`/`exclude`
   rule's inlined concepts, code-system reference, value-set composition, and `compose.include.filter`
   (the property/op/value triple of an *intensional* value set). Capturing the filter is what lets the enum stage tell
   an enumerable extensional value set from a non-enumerable intensional one rather than silently producing an empty
   code set.

2. **Model / IR** (`fhir/internal/gen/model`) turns the loader's flat list of dotted-path `ElementDefinition`s into
   the explicit element-path tree the later stages recurse over. From a `StructureDefinition` snapshot
   (`Observation`, `Observation.component`, `Observation.component.referenceRange`, `Observation.referenceRange.low`)
   it nests the paths into a tree so a backbone element carries its real child elements rather than an empty stub. It
   resolves the `contentReference` indirection FHIR uses for recursive or shared backbone shapes — for example
   `Observation.component.referenceRange` reuses `#Observation.referenceRange` and does not restate those children
   inline, so the model deep-copies the referenced element's children onto the referencing node. This recursion, with
   the graft, is the structural fix for empty backbone structs. Each element carries the metadata the later stages
   need: cardinality (`min`/`max`), the type set, the binding strength and value-set reference, the `isSummary` and
   `isModifier` flags, the choice (`[x]`) grouping with its branch types, and the resolved `contentReference`. The
   model classifies every `StructureDefinition` by `kind` (primitive type, complex type, resource). It is
   **release-agnostic**: it records only what the loaded bundle describes and makes no R4-versus-R5 assumption, maps
   no FHIR type to a Go type, and decides no Go name — those are the planner's job. The model emits no Go source; its
   only output is the in-memory tree, pinned by a golden snapshot test (run `go test ./fhir/internal/gen/model
   -update` to regenerate the snapshots when the IR shape changes on purpose). The model fails closed on a child
   whose parent path is absent from the snapshot and on a `contentReference` whose anchor is missing, since a silent
   drop is exactly how an empty backbone is produced.

3. **Planner** (`fhir/internal/gen/plan`) turns the release-agnostic IR into an emitter-ready plan: it decides every
   Go-shape question so the emitter only renders. Its rules are:

   - **Names.** A FHIR element name becomes an exported Go field identifier by stripping any `[x]` choice suffix,
     title-casing the words, and upper-casing known initialisms, so `start` becomes `Start`, `implicitRules` becomes
     `ImplicitRules`, `value[x]` becomes `Value`, and `id`/`url` become `ID`/`URL`. A FHIR type name maps to the same
     exported identifier (already PascalCase for complex types and resources). A nested backbone's Go type name is the
     owning type name concatenated with each occurrence-path segment (`Observation.component.referenceRange` becomes
     `ObservationComponentReferenceRange`).
   - **Collisions resolve deterministically.** Two FHIR names that map to the same Go identifier within one struct are
     disambiguated with an ascending numeric suffix (`Value`, `Value2`, `Value3`), recorded as each is assigned, so the
     result is identical on every run regardless of map iteration order. Each struct scope has its own name set, so a
     field named `Type` in one struct never perturbs a `Type` in another. Deterministic naming is what keeps the
     generated output byte-stable.
   - **Pointer versus value versus slice (cardinality).** A repeating element (`max` of `*` or greater than one) becomes
     a Go slice. A single-valued scalar becomes a pointer — including a **required** scalar, which is a pointer so a
     present `false` or `0` is distinguishable from an absent field (the structural half of the required-presence fix);
     a bare value would conflate the two. A single backbone is a pointer to its backbone type; a repeating one is a
     slice of it.
   - **Type mapping.** A FHIR primitive maps to its Go scalar (`boolean` to `bool`, the string and date/time families to
     `string`, the integer family to a sized integer, and `decimal` to `fhir.Decimal` so lexical precision survives a
     round trip rather than collapsing to `float64`). A complex or resource type maps to its Go type name.
   - **Backbones deduplicate by shape, not by path.** The IR is an occurrence-path tree, so the same anonymous backbone
     structure appears at several paths; the planner collapses structurally identical backbones (same fields, same
     decorated types, same order) to a single Go type, fingerprinting the field set rather than the path. The collected
     backbones are emitted sorted by Go name so the output order is stable.

   The planner makes no I/O and reads no template; its decisions are pinned by a golden planned-model snapshot
   (`fhir/internal/gen/plan/testdata/golden`, regenerated with `go test ./fhir/internal/gen/plan -update`).

4. **Emitter** (`fhir/internal/gen/emit`) renders the planned types through `text/template` and then a `go/format` pass,
   so the committed files are `gofmt`-clean and byte-stable; the emitter makes no Go-shape decision. Each generated file
   carries the standard `// Code generated by fhir-gen; DO NOT EDIT.` banner. **Reproducible generation:** `go generate
   ./fhir/...` regenerates the committed output byte-for-byte from the pinned bundle alone, and a committed generated
   file is never hand-edited. The property is executable, not aspirational: `TestRegenerationByteForByte`
   (`fhir/internal/gen`) regenerates into a temporary directory and diffs each file against the committed copy under
   `fhir/r5`, so a hand edit or an un-regenerated generator change fails the test with a precise message; the
   `gen:verify` mise task runs that test and then a `git diff --exit-code` over the committed tree. A generator change
   and its regenerated output are committed together (generator first); a definition-bundle change is committed
   separately.

> **Implementation status: PARTIAL.** The loader, the model / IR stage, the planner and emitter skeleton, the resource
> identity API (`Unmarshal[T]`, `As[T]`, `UnmarshalResource` with the FHIR-003 `resourceType` check), the
> init-populated `resourceType`→factory registry, the primitive → Go scalar mapping (with `decimal`→`fhir.Decimal`,
> FHIR-009), and the primitive-extension `_field` sibling machinery (shared `fhir.PrimitiveElement`, scalar and
> null-aligned repeating siblings, FHIR-005 no-sibling-on-complex) are implemented and tested. Representative datatypes
> (`r5.Period`, and `r5.HumanName` which exercises the scalar siblings, the null-aligned repeating siblings, and the
> FHIR-005 rule) and one representative resource (`r5.Flag`, with its `ResourceType` method, always-emit-`resourceType`
> `MarshalJSON`, a primitive `_status` sibling, and a generated registry entry) are generated end to end to prove the
> pipeline, the identity API, and the byte-for-byte regeneration gate. The full R5 datatype, resource, and backbone
> generation, the choice-type accessors, and the required-binding enums (closed enums with a strict-by-default
> validating decode and documented not-inlined boundaries for non-enumerable terminology, FHIR-013) are generated into
> `fhir/r5`. The hand-written Bundle builders and reference helpers are shipped, and the release-agnostic structural
> `Validate` engine (driven by a generated per-resource descriptor in `fhir/r5/validation_descriptors.go`) is shipped.
> The `_summary` serialiser (`MarshalSummary`) and the generated `fhir/r4` output are shipped: `fhir/r4` is generated
> end to end from the 4.0.1 definition bundle and conformance-tested by the merge-blocking HL7 validator gate alongside
> R5.

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
