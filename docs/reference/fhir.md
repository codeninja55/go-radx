# FHIR R4/R5

The `fhir` package is go-radx's type-safe implementation of HL7 FHIR. It models every resource and backbone element of
**R4 (4.0.1)** and **R5 (5.0.0)** as generated Go structs, gives you a checked resource-identity API
(`Unmarshal[T]` / `As[T]`), enforces choice-type mutual exclusion and `required`-binding value sets at the
serialization boundary, preserves decimal lexical fidelity, and round-trips the FHIR-JSON primitive-extension
(`_field`) mechanic that most libraries get wrong.

This document is the public-API contract for the package. The implementation conforms to it; where the PRD
(`docs/prd/go-radx-prd.md`) committed a signature in §8.1, this document honours it verbatim. Terminology follows the
project glossary (`UBIQUITOUS_LANGUAGE.md`); in particular, the FHIR `Element` base component (generated per release as
`r4.Element`/`r5.Element`) is never conflated with a DICOM data element, and the FHIR `Reference` datatype is never used
for a DICOM referenced-SOP UID pair.

## Scope and conformance

go-radx generates FHIR types from the official HL7 `StructureDefinition` packages for exactly two releases in v1:

- **R4** — FHIR 4.0.1 (the most-deployed release; US Core runs on it).
- **R5** — FHIR 5.0.0.

STU3 is out of scope. R4B (4.3.0) and R6 are deferred. The generator is architected so R6 can be added when it
becomes normative, but no v1 deliverable depends on it. Unlike `fhir.resources` (which from v7.0.0 collapses R4 into
its R4B sub-package), go-radx generates R4 4.0.1 directly, so the two releases are independent and faithful to their
own specifications.

All resources for both releases are generated and compile-tested. The radiology and clinical workflow set
(`Patient`, `Encounter`, `ServiceRequest`, `ImagingStudy`, `DiagnosticReport`, `Observation`, `Bundle`,
`OperationOutcome`) is conformance-tested against the official HL7 FHIR validator in CI. The published Conformance
Statement (`docs/conformance/`) is the single source of truth for which resources and profiles are verified.

In v1 the package serializes **JSON** only. XML and YAML are deferred. Profile validation against implementation guides
(US Core and the like), SMART on FHIR, and Subscriptions are out of scope for v1.

### Release selection in code

R4 and R5 are distinct, independently importable packages so a consumer never mixes the two type spaces by accident.
The root `fhir` package re-exports the release-agnostic machinery (the `Resource` interface, `Unmarshal`/`As`, the
`Decimal` primitive, the error types, and the `OperationOutcome` issue severities, which are stable across releases).

```go
import (
    "github.com/codeninja55/go-radx/fhir"       // release-agnostic: Resource, Unmarshal, As, Decimal, errors
    "github.com/codeninja55/go-radx/fhir/r4"     // R4 4.0.1 resources and datatypes
    "github.com/codeninja55/go-radx/fhir/r5"     // R5 5.0.0 resources and datatypes
)
```

Each release package's resource types satisfy `fhir.Resource`, so `fhir.Unmarshal[*r5.Patient]` and
`fhir.As[*r4.Patient]` both work without the caller importing release-specific helper functions.

The root `fhir` package holds **only** release-agnostic machinery: the `Resource` interface, `Unmarshal[T]`/`As[T]`,
`UnmarshalResource`, the `Decimal` primitive, the `OperationOutcome` issue-severity constants, the sentinel errors, and
the `resourceType`-to-factory registry. There is no `fhir.Patient`, `fhir.Reference`, or `fhir.Bundle` at the root —
every resource, backbone element, and complex datatype is generated per release under `fhir/r4` and `fhir/r5`.

### Complex datatypes

Complex datatypes (`Reference`, `Identifier`, `Coding`, `CodeableConcept`, `Quantity`, `HumanName`, `Period`, and the
rest) are generated per release alongside resources, so a consumer writes `r5.Reference` and `r5.Identifier`, never a
root `fhir.Reference`. The two datatypes that other go-radx packages depend on most — `convert` builds them from DICOM
and HL7 v2 data, `Bundle.Resolve` consumes them — have this committed shape (shown for R5; R4 is identical in these
fields):

```go
// Reference is a FHIR reference datatype: a pointer to another resource by
// relative/absolute URL, by logical Identifier, or both, with an optional
// human-readable Display and a Type hint naming the referenced resource.
type Reference struct {
    Reference  *string     `json:"reference,omitempty"`  // e.g. "Patient/pat-1" or "#contained-id"
    Type       *string     `json:"type,omitempty"`       // referenced resourceType, e.g. "Patient"
    Identifier *Identifier `json:"identifier,omitempty"` // logical (identifier-based) reference
    Display    *string     `json:"display,omitempty"`
}

// Identifier is a FHIR identifier datatype: a value qualified by the system that
// issued it. DICOM UIDs map to an Identifier with system "urn:dicom:uid" (the
// glossary rule), never to a Reference.
type Identifier struct {
    Use      *string          `json:"use,omitempty"`
    Type     *CodeableConcept `json:"type,omitempty"`
    System   *string          `json:"system,omitempty"`
    Value    *string          `json:"value,omitempty"`
    Assigner *Reference       `json:"assigner,omitempty"`
}
```

## The Resource interface and type-safe access

Every generated resource implements one small interface. The discriminator is the FHIR `resourceType` JSON property.

```go
// Resource is the base unit of FHIR exchange. ResourceType returns the FHIR
// discriminator (for example "Patient"), which is a compile-time constant per
// type, not a mutable field.
type Resource interface {
    ResourceType() string
}
```

The PRD-committed generic functions give static typing on top of a `resourceType`-keyed registry. Go cannot express a
"typed registry" directly, so the registry returns the interface and the generic functions add the *check* — that is
the win over the prototype's unchecked `UnmarshalResource[T]`, which never compared `resourceType` against `T`.

```go
// Unmarshal decodes FHIR JSON into a concrete resource type T and verifies that
// the embedded "resourceType" matches T. On mismatch it returns ErrResourceTypeMismatch
// rather than silently populating the wrong type.
func Unmarshal[T Resource](data []byte) (T, error)

// As is a checked downcast from the Resource interface to a concrete type T.
// It returns (zero, false) when r does not hold a T. Typical call site:
//   pat, ok := fhir.As[*r5.Patient](entry.Resource)
func As[T Resource](r Resource) (T, bool)

// UnmarshalResource decodes FHIR JSON whose resource type is not known at compile
// time. It peeks "resourceType", looks the factory up in the registry, and returns
// the concrete value behind the Resource interface. Unknown types return ErrUnknownResourceType.
func UnmarshalResource(data []byte) (Resource, error)
```

`Unmarshal[T]` peeks the `resourceType` token before fully decoding, so a `Patient` payload handed to
`fhir.Unmarshal[*r5.Observation]` fails fast with `ErrResourceTypeMismatch` and never produces a half-populated
`Observation`. This is the FHIR-003 fix from the Codex audit.

Marshalling is symmetric. Each generated type implements `MarshalJSON` so that `resourceType` is always emitted as the
type's constant value and a zero-value resource never serialises `"resourceType":""` (the FHIR-004 fix). You marshal
with the standard library:

```go
data, err := json.Marshal(patient) // emits {"resourceType":"Patient", ...}
```

### Worked example: round-trip and downcast

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/codeninja55/go-radx/fhir"
    "github.com/codeninja55/go-radx/fhir/r5"
)

func main() {
    raw := []byte(`{"resourceType":"Patient","id":"pat-1","gender":"female"}`)

    // Type-checked decode: fails if resourceType != "Patient".
    pat, err := fhir.Unmarshal[*r5.Patient](raw)
    if err != nil {
        panic(err)
    }
    fmt.Println(pat.Gender) // "female" (typed r5.AdministrativeGender)

    // Decode an unknown type, then narrow it.
    res, err := fhir.UnmarshalResource(raw)
    if err != nil {
        panic(err)
    }
    if p, ok := fhir.As[*r5.Patient](res); ok {
        out, _ := json.Marshal(p)
        fmt.Println(string(out))
    }
}
```

## Primitive types and the `_field` extension sibling

FHIR primitives map to Go scalars, with one deliberate exception: `decimal` maps to `fhir.Decimal` to preserve lexical
precision (see below). The mapping is:

| FHIR primitive | Go type |
|----------------|---------|
| `boolean` | `bool` |
| `integer`, `integer64`, `positiveInt`, `unsignedInt` | `int32` / `int64` |
| `string`, `code`, `id`, `uri`, `url`, `canonical`, `oid`, `uuid`, `markdown`, `base64Binary` | `string` (or an enum) |
| `decimal` | `fhir.Decimal` |
| `date`, `dateTime`, `instant`, `time` | `string`, validated on decode (calendar, timezone, leap-second rules) |

Required scalar primitives are generated as **pointers** (`*bool`, `*int32`, and so on) so that absence is
distinguishable from a valid zero value. This is the structural half of the FHIR-007 fix: a required `false` is present
because the pointer is non-nil, regardless of the value it points to.

The Go-scalar mapping above applies to **plain primitive fields**. When a primitive appears as a branch of a choice
group (see [Choice types](#choice-types-typed-accessors-and-mutual-exclusion)), the value is boxed in the release's
named primitive wrapper type — `r5.FHIRString`, `r5.FHIRBoolean`, `r5.FHIRInteger`, and so on — because the built-in
Go `string`/`bool`/`int32` cannot carry the marker method that seals the choice interface. The wrapper's underlying
type is the same scalar, so the conversion cost is a single `r5.FHIRString("text")` at the call site.

### The `_field` sibling

In FHIR JSON, a primitive element may carry an `id` and `extension` through a sibling property whose name is the
primitive's name prefixed with an underscore. For a primitive field `Foo`, the generated struct carries a paired
`FooElement` field of type `*PrimitiveElement`. The Go field name uses the idiomatic `XxxElement` suffix (the
convention HAPI and most FHIR codegen use); the JSON tag still maps it to the `_foo` wire sibling, so the underscore
lives only in the serialised form, never in a Go identifier:

```go
// PrimitiveElement carries the id and extensions of a primitive value through
// its "_field" JSON sibling. It corresponds to FHIRPrimitiveExtension in the
// FHIR specification.
type PrimitiveElement struct {
    ID        *string      `json:"id,omitempty"`
    Extension []*Extension `json:"extension,omitempty"`
}

// Example: Patient.gender and its "_gender" sibling.
type Patient struct {
    // ...
    Gender        AdministrativeGender `json:"gender,omitempty"`
    GenderElement *PrimitiveElement    `json:"_gender,omitempty"`
    // ...
}
```

`_field` siblings are generated **only** for true primitive elements — never for complex datatypes, backbone elements,
`contained`, or `Bundle.entry.resource`. That is the FHIR-005 fix. For a **repeating** primitive
(`[]string` and similar), the sibling is a parallel slice (`[]*PrimitiveElement`) and custom marshalling preserves
positional alignment, emitting JSON `null` placeholders so the value array and the `_field` array stay index-aligned
per the FHIR primitive-array rules.

```go
// Repeating primitive: aligned arrays, null-padded on marshal.
//   "given":  ["Jane", "Q"]
//   "_given": [null, {"id":"middle-initial"}]
type HumanName struct {
    Given        []string            `json:"given,omitempty"`
    GivenElement []*PrimitiveElement `json:"_given,omitempty"`
    // ...
}
```

A primitive value and its `_field` sibling round-trip together: decode then re-encode reproduces both, including the
`null` placeholders in repeating arrays.

## The Decimal primitive

`fhir.Decimal` preserves the exact lexical form of a FHIR `decimal` (and is the same type used for DICOM `DS`/`IS`
values), so `1.20` and `1.2` are distinguishable and trailing zeros survive a round-trip. This is the FHIR-009 fix; the
prototype mapped `decimal` to `float64` and lost precision.

```go
// Decimal preserves the source lexical form of a FHIR decimal. Conversion to a
// machine number is explicit; the type performs no in-place arithmetic.
type Decimal struct {
    // unexported: preserves the source lexical form
}

// ParseDecimal builds a Decimal from a lexical string, validating it against the
// FHIR decimal production. An empty or malformed input returns an error.
func ParseDecimal(s string) (Decimal, error)

// String returns the preserved lexical form exactly as parsed.
func (d Decimal) String() string

// Float64 returns the value as a float64. ok is false only when the lexical form
// has no finite float64 representation; a value that is representable but not
// exactly (such as 0.1) still returns ok == true. Callers that need full
// precision use String or BigFloat. This signature and semantics are identical
// across dicom.md, fhir.md, and the conformance statements.
func (d Decimal) Float64() (f float64, ok bool)

// Exact reports whether the lexical form converts to a float64 with no loss of
// precision. It is the exactness signal kept separate from Float64's ok return,
// so the bool on Float64 is never overloaded to mean two different things.
func (d Decimal) Exact() bool

// BigFloat returns a *big.Float with precision sufficient for the lexical form.
func (d Decimal) BigFloat() *big.Float

// MarshalJSON emits the preserved lexical form as a JSON number, not a string.
func (d Decimal) MarshalJSON() ([]byte, error)

// UnmarshalJSON captures the raw JSON number token verbatim before any float
// conversion, so precision is preserved on decode.
func (d *Decimal) UnmarshalJSON(data []byte) error
```

The contract is deliberately conservative: `Decimal` carries a value, it does not compute one. There is no `Add` or
`Mul`. A consumer that needs arithmetic converts explicitly via `BigFloat` and decides its own rounding, so go-radx
never silently changes a clinically meaningful number.

## Choice types: typed accessors and mutual exclusion

A FHIR choice element (the `nnn[x]` pattern, for example `Observation.value[x]`) permits exactly one value drawn from a
set of allowed types. On the wire, FHIR replaces `[x]` with the chosen type's name (`valueQuantity`,
`valueCodeableConcept`, `valueString`, and so on). The reference library `fhir.resources` expands this to N nullable
suffixed fields plus a "one of many" validator, which leaves it possible to set two branches at once in Go before
validation runs.

go-radx keeps the suffixed storage fields for faithful JSON round-tripping but makes the **typed accessor pair the
canonical API**, so mutual exclusion is enforced at the boundary rather than only at validation time. Each choice group
gets a sealed value interface, a `Value()` getter, and one `SetValueX` setter per allowed branch; every setter clears
the other siblings.

```go
// ObservationValue is the sealed value type for Observation.value[x]. It is
// implemented only by NAMED FHIR datatype types — never the built-in string,
// bool, or int32, which cannot carry the unexported marker method. The complex
// branches (Quantity, CodeableConcept, Range, Ratio, SampledData, Period, and
// the rest defined by the StructureDefinition for the release) are the generated
// datatype structs; the primitive branches are the release primitive wrappers
// (FHIRString, FHIRBoolean, FHIRInteger, FHIRDateTime, FHIRTime, ...).
type ObservationValue interface {
    isObservationValue()
}

// Value returns the currently set value[x] branch. ok is false when no branch
// is set.
func (o *Observation) Value() (ObservationValue, bool)

// SetValueQuantity sets value[x] to a Quantity and clears every other value[x]
// sibling, so at most one branch is ever populated.
func (o *Observation) SetValueQuantity(q Quantity)

// SetValueCodeableConcept sets value[x] to a CodeableConcept and clears the
// other siblings. There is one such setter per allowed branch.
func (o *Observation) SetValueCodeableConcept(c CodeableConcept)
```

Reading a choice value uses a type switch on the sealed interface. A primitive branch is the release wrapper type, so
recovering the plain Go value is one explicit conversion — the cost of making `string`-valued choices type-safe:

```go
if v, ok := obs.Value(); ok {
    switch val := v.(type) {
    case r5.Quantity:
        fmt.Printf("%s %s\n", val.Value, val.Unit) // val.Value is fhir.Decimal
    case r5.CodeableConcept:
        fmt.Println(val.Text)
    case r5.FHIRString:
        fmt.Println(string(val)) // FHIRString's underlying type is string
    }
}
```

Setting a `string`-valued branch boxes the value the same way, for example
`obs.SetValueString(r5.FHIRString("normal"))`. The wrapper appears only on choice branches; a plain `Observation.note`
text field is still a Go `string`.

Because every `SetValueX` clears the others, you cannot author a resource with both `valueQuantity` and `valueString`
populated, and the marshaller therefore cannot emit an invalid two-branch choice. This is the FHIR-001 fix; it also
removes the prototype's unsuffixed `*any` choice fields (FHIR-002), which never round-tripped conformant JSON.

When a choice group is `required` (minimum cardinality 1), validation reports a missing-choice issue if no branch is
set, and a multiple-branch issue is unreachable through the accessors. Group cardinality is validated once per group,
not once per branch.

## Generated enums for required value-set bindings

A FHIR element with a `code` (or `Coding`/`CodeableConcept`) datatype may carry a value-set binding at one of four
strengths: `required`, `extensible`, `preferred`, or `example`. Only **`required`** strength becomes a closed Go enum;
the others stay `string`, because only `required` strength forbids codes outside the value set.

For each required binding the generator emits a defined string type, a const set, and a validating parser. This is the
FHIR-013 fix and the core of the type-safety thesis — `fhir.resources` stores these bindings as `enum_values`
*metadata only* and never enforces them.

```go
// AdministrativeGender is the required binding for fields such as Patient.gender.
type AdministrativeGender string

const (
    GenderMale    AdministrativeGender = "male"
    GenderFemale  AdministrativeGender = "female"
    GenderOther   AdministrativeGender = "other"
    GenderUnknown AdministrativeGender = "unknown"
)

// ParseAdministrativeGender validates s against the required binding. An
// out-of-set code returns ErrUnknownCode wrapped with the binding name and the
// offending value, with no PHI.
func ParseAdministrativeGender(s string) (AdministrativeGender, error)
```

### Unknown-code policy

Go cannot enforce a defined string type at every literal assignment; `r5.AdministrativeGender("banana")` compiles. The
enforcement boundary is therefore JSON decode and explicit `Parse`, where the policy depends on binding strength:

- **Required binding, strict decode (the default).** An unknown code on decode is an error. `UnmarshalJSON` for the
  enum type rejects out-of-set values with `ErrUnknownCode`, so a non-conformant payload does not silently populate the
  field.
- **Required binding, lenient decode (opt-in).** With the lenient decode option set, an unknown required code is
  retained verbatim and surfaced as an `OperationOutcome` issue from `Validate`, so a consumer ingesting
  partially-conformant data from the wild can choose to inspect rather than reject. This is opt-in precisely because
  fail-closed is the safe default (PRD §9.2).
- **Extensible, preferred, example bindings.** The field is `string`; any value is accepted. The generator emits the
  const set as documentation-only constants where useful, but no `Parse` rejects out-of-set codes.

`ParseXxx` always applies the strict rule regardless of decode options, so application code that wants a guaranteed-
valid value calls the parser explicitly:

```go
g, err := r5.ParseAdministrativeGender(userInput)
if errors.Is(err, fhir.ErrUnknownCode) {
    // userInput was not one of male|female|other|unknown
}
```

## Cardinality and required validation

FHIR cardinality is `min..max`. The Go representation makes the common cases unrepresentable-when-wrong where the type
system allows, and the rest is checked by `Validate`:

- **0..1** optional scalar: a pointer (`*string`, `*bool`) or, for complex datatypes, a pointer to a struct. Absence is
  `nil`.
- **1..1** required scalar: a pointer that `Validate` requires to be non-nil. Generating it as a pointer is what makes
  a valid `false` or `0` count as present — this is the FHIR-007 fix. Required is about **presence, not truthiness**.
- **0..\*** / **1..\*** repeating: a slice. A required repeating element must be non-empty.

```go
// Validate checks structural conformance for one resource: required-element
// presence (presence, never IsZero on the value), choice-group cardinality,
// required-binding codes, and fixed/pattern constraints expressible from the
// StructureDefinition. It returns an *OperationOutcome whose issues enumerate
// every problem, or nil when the resource is conformant.
func Validate(r Resource) *OperationOutcome
```

`Validate` reports every issue it finds rather than stopping at the first, and reports presence by tracking whether the
field (or its `_field` sibling) was set, not by reflecting on the Go zero value. It does not perform terminology
expansion, profile-slicing, or FHIRPath invariant evaluation in v1; those are deferred. The authoritative cross-check
remains the official HL7 FHIR validator in CI (PRD §11.1).

## Bundles

A `Bundle` is a typed resource whose processing semantics depend on `Bundle.type`. Like every resource it is generated
per release, so the type and its builders live in `fhir/r4` and `fhir/r5` (`r5.Bundle`, `r5.NewSearchSet`, and so on);
there is no root `fhir.Bundle`. go-radx models the entry structure faithfully (`fullUrl`, `resource`, `request`,
`response`, `search`) and provides type-specific construction so you cannot accidentally produce a bundle that violates
the FHIR `bdl-*` invariants. This is the FHIR-010 fix; the prototype incremented `total` for every bundle type and
enforced no per-type constraints. The builders below are shown for R5; R4 has the same signatures in its own package.

```go
// BundleType is the required binding for Bundle.type.
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

// NewSearchSet builds a searchset Bundle. total is the server's match count and
// is the only bundle type for which total is permitted.
func NewSearchSet(total int, entries ...SearchEntry) *Bundle

// NewTransaction builds a transaction Bundle. Each entry carries a request
// (method + url); total is not set, and the builder rejects an entry with a
// response (responses belong to transaction-response bundles).
func NewTransaction(entries ...TransactionEntry) (*Bundle, error)

// NewDocument builds a document Bundle whose first entry must be a Composition,
// per the document first-resource rule.
func NewDocument(composition Resource, entries ...DocumentEntry) (*Bundle, error)

// NewMessage builds a message Bundle whose first entry must be a MessageHeader.
func NewMessage(header Resource, entries ...MessageEntry) (*Bundle, error)
```

The builders enforce the invariants the validator would otherwise catch only at marshal time:

- `total` is set only for `searchset` and `history`.
- `entry.search` appears only in a `searchset`.
- `entry.request` is required in every `transaction`/`batch` entry; `entry.response` only in the response variants.
- A `document` bundle's first entry is a `Composition`; a `message` bundle's first entry is a `MessageHeader`.
- `fullUrl` is unique across entries; the transaction builder rejects duplicates.

`Validate` applies the same checks to a bundle decoded from JSON, so an inbound non-conformant bundle is reported, not
silently accepted.

The bundle builders are **not** safe for concurrent mutation; a single bundle should be built by one goroutine. Once
built and marshalled, the JSON is immutable and freely shareable. This resolves FHIR-015 by making the constraint
explicit rather than papering over it with a mutex on a mutable helper.

## Reference integrity

go-radx resolves references within a bundle and within `contained` resources, the two contexts where FHIR defines local
resolution. These are methods on the release-specific `Bundle` and `DomainResource` types (shown for R5); they take a
release `Reference` and return the root `fhir.Resource` interface so the caller narrows with `fhir.As[T]`:

```go
// Resolve looks up the resource that ref points to within the scope of the bundle.
// It resolves entry.fullUrl matches and "#id" contained references. ok is false
// when the target is not present in the bundle (an external reference, or a
// dangling local one).
func (b *Bundle) Resolve(ref Reference) (res fhir.Resource, ok bool)

// Contained returns the contained resource with the given anchor id ("#id" without
// the leading '#'). It returns an error — not a silent miss — when a contained
// entry is structurally malformed, so a corrupt payload surfaces as a data-quality
// failure rather than a false "not found". This is the FHIR-011 fix.
func (r *DomainResource) Contained(id string) (fhir.Resource, error)

// CheckReferenceIntegrity walks every Reference in the bundle and reports, as
// OperationOutcome issues, any local reference (fullUrl or "#id") that does not
// resolve. External references (absolute URLs to other servers) are not flagged.
func (b *Bundle) CheckReferenceIntegrity() *fhir.OperationOutcome
```

`Contained` returns an aggregate error identifying the offending entry index when a contained resource is malformed,
rather than skipping it. That distinction matters in clinical payloads: a corrupt contained `Observation` must not
disappear into a "not found".

## OperationOutcome and the error model

The package distinguishes two failure channels, and uses each deliberately:

1. **Go errors** for operations that either succeed or fail as a unit — JSON decode, `Parse`, type mismatch. These are
   sentinel-comparable with `errors.Is`.
2. **`OperationOutcome`** for validation, where the natural result is a *set* of issues with severities. `Validate`,
   `CheckReferenceIntegrity`, and the bundle builders' validators return an `*OperationOutcome`.

The `OperationOutcome` resource is generated per release like any other resource (`r4.OperationOutcome`,
`r5.OperationOutcome`). The `IssueSeverity` binding and its constants are stable across R4 and R5, so they live in the
root `fhir` package as release-agnostic machinery and both release packages alias them:

```go
// OperationOutcome is a structured FHIR result carrying zero or more issues. It
// is a generated resource (r4.OperationOutcome / r5.OperationOutcome) with the
// standard DomainResource fields (id, meta, text, and so on) elided here.
type OperationOutcome struct {
    Issue []*OperationOutcomeIssue `json:"issue"`
}

// IssueSeverity is the required binding for OperationOutcome.issue.severity. It
// lives in the root fhir package because the severities are identical in R4 and
// R5.
type IssueSeverity string

const (
    SeverityFatal       IssueSeverity = "fatal"
    SeverityError       IssueSeverity = "error"
    SeverityWarning     IssueSeverity = "warning"
    SeverityInformation IssueSeverity = "information"
    SeveritySuccess     IssueSeverity = "success" // R5 only
)

// HasErrors reports whether any issue is fatal or error severity. A nil
// *OperationOutcome has no errors, so the conformant path returns nil and
// callers can write: if oo.HasErrors() { ... }
func (oo *OperationOutcome) HasErrors() bool

// Error lets an OperationOutcome be returned as a Go error when a caller prefers
// the error channel. The message names the resource and the first error issue,
// honouring the no-PHI-by-default rule (it names elements and paths, not patient
// values).
func (oo *OperationOutcome) Error() string
```

Each issue carries a `severity`, a `code` (the FHIR issue-type), `diagnostics` text, and an `expression` (a FHIRPath-
style path to the offending element). Diagnostics name the element path and the rule violated, never the patient value
— consistent with PRD §8.2 and §9.1.

### Sentinel errors

```go
var (
    // ErrResourceTypeMismatch is returned by Unmarshal[T] when the payload's
    // resourceType does not match T.
    ErrResourceTypeMismatch = errors.New("fhir: resourceType does not match target type")

    // ErrUnknownResourceType is returned by UnmarshalResource when resourceType
    // is absent or not in the registry.
    ErrUnknownResourceType = errors.New("fhir: unknown resourceType")

    // ErrUnknownCode is returned by ParseXxx and by strict decode of a required
    // binding when a code is outside the bound value set.
    ErrUnknownCode = errors.New("fhir: code not in required value set")
)
```

All three are designed for `errors.Is`. Wrapped instances add the binding name, the target type, or the offending
token, but never PHI.

## Summary mode

`_summary` controls how much of a resource is serialised — a bandwidth optimisation FHIR servers use to return lighter
payloads. go-radx carries this prototype feature forward as an explicit, panic-free serialiser. The five modes follow
the FHIR `SummaryEnum`:

```go
// SummaryMode selects how much of a resource MarshalSummary emits.
type SummaryMode string

const (
    SummaryFull  SummaryMode = "false" // full resource (no filtering)
    SummaryTrue  SummaryMode = "true"  // only elements flagged isSummary in the StructureDefinition
    SummaryText  SummaryMode = "text"  // narrative (text), id, meta, and mandatory elements only
    SummaryData  SummaryMode = "data"  // everything except the narrative (text)
    SummaryCount SummaryMode = "count" // intended for Bundle: emit total only, no entries
)

// MarshalSummary serialises r under the chosen summary mode. A nil resource
// returns an error rather than panicking (the FHIR-012 fix). When the server
// has dropped elements, it sets meta.tag SUBSETTED per the FHIR summary rules.
func MarshalSummary(r Resource, mode SummaryMode) ([]byte, error)
```

`SummaryTrue` filtering is driven by the `isSummary` flag the generator records per element from the
`StructureDefinition`, so the filter is data-driven rather than hand-maintained. `MarshalSummary(nil, ...)` returns
`ErrNilResource`; it does not reflect on an invalid value.

## The code generator

The generated Go in `fhir/r4` and `fhir/r5` is produced by the generator under `fhir/internal/gen` from the official
HL7 `StructureDefinition` packages (`hl7.fhir.r4.core#4.0.1` and `hl7.fhir.r5.core#5.0.0`). The generator is a build-
time tool, not part of the runtime API surface; consumers import the generated packages, never the generator.

The generated code is **never hand-edited**. Regeneration is reproducible from pinned, checksum-verified
`StructureDefinition` inputs, and `go generate ./fhir/...` reproduces the committed output byte-for-byte. Pixel of
trust: the prototype's generator was rewritten wholesale (PRD §12) because its model was structurally wrong; the
regeneration *approach* was kept, the model was not.

### What the generator must get right

The generator builds an explicit element tree from full `StructureDefinition` element paths
(`Observation.component.referenceRange.low` and the like) and recurses from that tree, so nested backbone elements are
fully populated. Empty backbone structs like the prototype's `ObservationComponentReferenceRange` are a defect the
generator's golden tests guard against (the FHIR-006 fix). The generator's responsibilities, each gated by a golden
test, are:

- **Resources and backbone elements.** Every resource and every nested backbone element of the release, with the
  correct fields recursed from the element-path tree.
- **Choice groups.** One sealed value interface, one `Value()` getter, and one `SetValueX` setter per branch, plus the
  suffixed storage fields for JSON. Group cardinality is validated once per group.
- **Primitive extensions.** A `*PrimitiveElement` sibling for each scalar primitive and a `[]*PrimitiveElement` for each
  repeating primitive, with null-aligned marshalling — and no `_field` sibling on any non-primitive. The sibling field
  uses the idiomatic `XxxElement` Go name with a `json:"_xxx,omitempty"` tag, so no Go identifier carries an underscore.
- **Required-binding enums.** A defined string type, const set, `ParseXxx`, and a validating `UnmarshalJSON` for every
  `required`-strength binding; `string` for weaker strengths.
- **`resourceType`.** A constant `ResourceType()` method and a `MarshalJSON` that always emits the constant.
- **Canonical element ordering.** Fields and JSON keys are emitted in `StructureDefinition` order so output is stable
  and diff-friendly.
- **Registry.** A `resourceType → factory` entry per resource, populated in each release package's `init`, backing
  `UnmarshalResource` and `Unmarshal[T]`/`As[T]`.

### Generated package layout

```
fhir/
├── resource.go          # Resource interface, Unmarshal[T], As[T], UnmarshalResource, registry, sentinel errors
├── decimal.go           # Decimal primitive (Float64, Exact, BigFloat)
├── primitive.go         # PrimitiveElement, primitive-extension marshalling helpers (shared by both releases)
├── summary.go           # SummaryMode, MarshalSummary
├── outcome.go           # IssueSeverity binding and severity constants (the resource is generated per release)
├── internal/gen/        # the code generator (build-time only)
├── r4/                  # generated R4 4.0.1: resources, datatypes, enums, OperationOutcome, registry init
└── r5/                  # generated R5 5.0.0: resources, datatypes, enums, OperationOutcome, registry init
```

The root machinery is release-agnostic; everything that differs between R4 and R5 — resources, complex datatypes such
as `Reference` and `Identifier`, the per-release `OperationOutcome` resource, and the required-binding enums — lives
in `r4` and `r5`. Generated release structs reference the shared `fhir.PrimitiveElement` for their `_field` siblings.

R4 and R5 are genuinely separate packages with their own type names, resolving the FHIR-014 defect where the prototype
declared `package resources` inside a `types` directory and produced two incompatible sets of similarly named types.

## Conformance scope and limits

What v1 guarantees:

- All R4 4.0.1 and R5 5.0.0 resources and backbone elements are generated and compile-test clean.
- JSON serialisation round-trips, including the `_field` primitive-extension mechanic (scalar and repeating, with null
  alignment), `decimal` lexical fidelity, choice-type single-branch encoding, and `resourceType` integrity.
- The workflow set (`Patient`, `Encounter`, `ServiceRequest`, `ImagingStudy`, `DiagnosticReport`, `Observation`,
  `Bundle`, `OperationOutcome`) validates against the official HL7 FHIR validator in CI.
- `Validate` enforces required-element presence (by presence, not truthiness), choice-group cardinality, required-
  binding codes, and the `bdl-*` Bundle invariants reachable through the builders.

What v1 does **not** do (deferred or out of scope):

- XML and YAML serialisation (JSON only in v1).
- Profile validation against implementation guides (US Core and others), terminology-server expansion, FHIRPath
  invariant evaluation, and slicing — `Validate` is structural, not a full profile validator.
- SMART on FHIR, Subscriptions, R4B, R6, and STU3.
- Compile-time enforcement of enum values at every literal assignment — Go cannot express it; enforcement is at the
  JSON boundary and at `ParseXxx`.

For the verified resource, profile, and message subset, the FHIR Conformance Statement in `docs/conformance/` is
authoritative.

## See also

- [go-radx PRD](../prd/go-radx-prd.md) — §6.2 parity floor, §8.1 API commitments, §8.2 design principles, §9 NFRs.
- [Ubiquitous language glossary](../../UBIQUITOUS_LANGUAGE.md) — canonical Go names and collision rules.
- HL7 FHIR R5 — <https://www.hl7.org/fhir/R5/>
- HL7 FHIR R4 — <https://hl7.org/fhir/R4/>
