# Cross-standard conversion

The `convert` package bridges the three standards go-radx implements — DICOM (NEMA PS3), HL7 v2.x, and HL7 FHIR
(R4 4.0.1 and R5 5.0.0). It exists because the same clinical concept is modelled differently in each standard, and
silently aliasing those models is a primary source of conversion bugs (see `UBIQUITOUS_LANGUAGE.md`, "Cross-standard
collisions"). Every converter keeps each standard's own nouns and bridges between them explicitly, so direction and both
type spaces are unambiguous from the signature alone.

## Overview and scope

The package surface follows one convention without exception: `convert.<Source>To<Target><Release>`, where `<Source>`
and `<Target>` are each standard's canonical Go noun and `<Release>` is the explicit FHIR release suffix (`R4` or `R5`)
on every converter that produces or consumes a FHIR resource. The release is part of the function name, not a runtime
flag, because the FHIR resource models live in separate release sub-packages (`fhir/r4` and `fhir/r5`) that are
deliberately distinct type spaces — there is no release-agnostic `ImagingStudy` to return. Bidirectional mappings are
two separately named functions, never a single function with a direction flag. There is no shared cross-standard
`Study`, `Instance`, `Reference`, `Observation`, `Report`, or `Order` type — the DICOM side stays UID-keyed dataset
access (`*dicom.DataSet`), the FHIR side is a generated release-typed resource (`*r5.ImagingStudy`,
`*r4.DiagnosticReport`, …), and the HL7 side is a typed segment or message (`*hl7v2.Message`, `hl7v2.OBX`).

The FHIR package layout this package targets is the one documented in [`fhir.md`](fhir.md): all resources and datatypes
(`r5.Patient`, `r5.ImagingStudy`, `r5.Reference`, `r5.Identifier`, …, and the same set under `r4`) live in the release
sub-packages; the root `fhir` package holds only release-agnostic machinery (the `Resource` interface, `Unmarshal[T]`
/`As[T]`, `Decimal`, the sentinel errors, and the `resourceType`→factory registry). Every signature below names the
release sub-package type it returns, so a reader knows from the signature alone whether an `*r4` or `*r5` value comes
back. The DICOM Structured Report content-item model the SR converters pivot on (`dicom.ContentItem`,
`dicom.ConceptNameCode`, the `ValueType` vocabulary, and SR document read/build) is defined in
[`dicom.md`](dicom.md); the SR converters reference it rather than redefining it.

The v1 converters, taken from PRD §8 and the glossary cross-standard table, come in release-explicit pairs. Each FHIR
resource producer/consumer below has both an `…R4` and an `…R5` form, each returning the matching release
sub-package type; the table lists the `…R5` form, with the `…R4` form differing only in the FHIR types named:

| Function (R5 form; an R4 twin exists) | Source type | Target type |
|----------------------------------------|-------------|-------------|
| `DICOMToImagingStudyR5` | `[]*dicom.DataSet` (one or many instances) | `*r5.ImagingStudy` |
| `SRToDiagnosticReportR5` | `*dicom.DataSet` (SR document) | `*r5.DiagnosticReport` + `[]*r5.Observation` |
| `DiagnosticReportToSRR5` | `*r5.DiagnosticReport` + observations | `*dicom.DataSet` (SR document) |
| `ORUToDiagnosticReportR5` | `*hl7v2.Message` (`ORU^R01`) | `*r5.DiagnosticReport` + `[]*r5.Observation` |
| `ORMToServiceRequestR5` | `*hl7v2.Message` (`ORM^O01` / `OMG^O19`) | `*r5.ServiceRequest` |
| `ADTToPatientR5` | `*hl7v2.Message` (`ADT^Axx`) | `*r5.Patient` |
| `ADTToEncounterR5` | `*hl7v2.Message` (`ADT^Axx`) | `*r5.Encounter` |
| `OBXToObservationR5` / `ObservationToOBXR5` | `hl7v2.OBX` ↔ `*r5.Observation` | element-level helpers |
| `ContentItemToObservationR5` / `ObservationToContentItemR5` | `dicom.ContentItem` ↔ `*r5.Observation` | element-level helpers |

Out of scope for v1 (deferred, see PRD §3.2): FHIR-to-DICOM `ImagingStudy` reconstruction (the worklist/storage path
produces DICOM directly, not via conversion), FHIR `MessageHeader`/Bundle-message round-trips of HL7, US Core profile
conformance of the produced resources, and terminology translation between code systems (a `Coding` is carried across
verbatim; go-radx does not map SNOMED to LOINC or vice versa).

### FHIR release targeting

Every resource-producing converter targets a specific FHIR release because the resource models differ between R4 and
R5 in load-bearing ways (notably `ImagingStudy.modality` is `Coding` in R4 but `CodeableConcept` in R5, and
`ImagingStudy` procedure/reason fields were restructured for R5). The release is fixed by the function name (`…R4` vs
`…R5`), which is what makes the return type concrete: `DICOMToImagingStudyR5` returns `*r5.ImagingStudy` and
`DICOMToImagingStudyR4` returns `*r4.ImagingStudy`. A single release-agnostic return type is impossible given the
separate-package FHIR layout, so the release is lifted into the name rather than a runtime flag.

The shared options carry per-call configuration that is independent of release. The `WithSubject` option is itself
release-typed because a FHIR `Reference` is a release sub-package datatype, so it comes in `R4` and `R5` forms that
pair with the matching converter:

```go
// Option configures a conversion. Zero options means: lossless-or-error off, generate UIDs as needed.
type Option func(*config)

func WithUIDRoot(root dicom.UID) Option         // org root for any minted UIDs (DiagnosticReportToSR…)
func WithStrictLoss() Option                    // turn lossy drops into a returned *LossError instead of a Report entry
func WithSubjectR5(ref r5.Reference) Option     // inject the Patient reference the source cannot supply (R5 converters)
func WithSubjectR4(ref r4.Reference) Option     // the R4 twin, for R4 converters (see identity)
```

`convert` depends only on `dicom`, `hl7v2`, and `fhir` (including its `fhir/r4` and `fhir/r5` sub-packages). It pulls in
no CLI, server, or network dependency, honouring the PRD §7.4 rule that library consumers never inherit the CLI
dependency graph.

## The conversion report and the error model

Conversion is inherently lossy in places: DICOM carries attributes FHIR has no home for, FHIR markdown narrative has no
DICOM equivalent, and HL7 v2 free-text fields do not always map to a coded FHIR element. go-radx never silently discards
data and never reports success on a lossy or failed conversion in a way that hides the loss (PRD §9.2, honest failure
reporting). Two mechanisms carry this:

```go
// Report accompanies every successful conversion and records what could not be mapped cleanly.
type Report struct {
    Dropped     []DroppedField // source data with no target home
    Defaulted   []DefaultedField // target required a value the source did not supply
    Substituted []Substitution // a value mapped approximately (e.g. precision narrowing)
}

type DroppedField struct {
    Source  string // human-readable source locus: "DICOM (0008,1030) StudyDescription" or "OBX-17 ObservationMethod"
    Reason  string // why it was dropped, in plain language
}

// LossError is returned (instead of a Report entry) only when WithStrictLoss is set and a drop occurs.
type LossError struct{ Dropped []DroppedField }

func (e *LossError) Error() string
```

Converters return the produced resource(s), a non-nil `*Report`, and an `error`. The `error` is reserved for genuine
failures — malformed source, a missing required identifier the converter cannot synthesise, or (under
`WithStrictLoss`) a `*LossError`. A clean conversion that nonetheless dropped optional, unmappable attributes returns a
nil `error` and a populated `Report`. Callers inspect the `Report` to decide whether the loss is acceptable for their
use; the library provides the facts, the consumer owns the policy (PRD §9.1, §9.5).

Errors are sentinel-comparable with `errors.Is`:

```go
var (
    ErrMissingIdentifier = errors.New("convert: source lacks a required identifier")
    ErrUnsupportedSource = errors.New("convert: source message type or SOP class is out of scope")
    ErrMalformedSource   = errors.New("convert: source failed structural validation")
)
```

Diagnostics name concepts, never raw codes (PRD §8.2): a dropped DICOM attribute is rendered as its keyword plus
`(gggg,eeee)`, an HL7 locus as its `SEG-Fn` accessor, and a FHIR locus by element path. No converter logs or embeds PHI
by default; the `Source` strings carry identifiers and structure, not patient values.

## Identity handling — the single most important rule

A DICOM UID is a globally unique ISO object identifier; a FHIR `Reference` is a URL or logical pointer to a resource
that may not exist in any accessible server. These are not the same thing, and conflating them produces dangling
references.
The rule, fixed by the glossary, is absolute:

**A DICOM UID always becomes a FHIR `Identifier`, never a `Reference.reference` URL.** The `Identifier` uses the
DICOM URN namespace as its `system`. Because `Identifier` is a release sub-package datatype, the shared helper comes in
the same release-explicit pair as the converters:

```go
// Helpers used by every DICOM-sourced converter, so the SR and ImagingStudy paths share one rule.
func UIDIdentifierR5(uid dicom.UID) r5.Identifier // system="urn:dicom:uid", value="urn:oid:" + uid
func UIDIdentifierR4(uid dicom.UID) r4.Identifier // the R4 twin

// e.g. dicom.UID("1.2.840.113619.2.55.3.604688.1") becomes (R5 shown):
//   r5.Identifier{ System: "urn:dicom:uid", Value: "urn:oid:1.2.840.113619.2.55.3.604688.1" }
```

The `ImagingStudy.identifier`, `ImagingStudySeries.uid`, and `ImagingStudySeriesInstance.uid` elements receive these
`Identifier`/`id` forms. The `ImagingStudy.subject` and `DiagnosticReport.subject` are genuine `Reference`s to a
`Patient` — but the source DICOM dataset does not contain a FHIR Patient resource ID, so the converter cannot invent
one. It therefore populates `subject` only when the caller supplies it via `WithSubjectR4`/`WithSubjectR5`; otherwise it
leaves `subject` unset and records a `Defaulted` entry. The patient's DICOM identity (PatientID, IssuerOfPatientID) is
preserved as a `Reference.identifier` (a logical reference) rather than fabricated as a `Reference.reference` URL:

```go
// When no WithSubject is given, the produced Patient reference carries the DICOM patient identity logically (R5 shown):
//   r5.Reference{ Identifier: &r5.Identifier{ System: issuerOfPatientID, Value: patientID }, Type: "Patient" }
```

HL7 v2 identifiers (`CX`: ID + assigning authority) map to FHIR `Identifier` with `system` derived from the assigning
authority (HD) and `value` from the ID component — again never a `Reference.reference` URL. Cross-resource references
within a single conversion call (for example, a `DiagnosticReport.result` pointing at the `Observation`s produced in the
same call) use intra-Bundle logical references (`urn:uuid:` placeholders) so the resources can be assembled into a
transaction `Bundle` by the caller without a round trip to a server.

## DICOMToImagingStudy

```go
// DICOMToImagingStudyR5 builds a FHIR R5 ImagingStudy from one or more DICOM instances of the same study.
// Pass every available instance dataset; the converter groups by Series Instance UID and SOP Instance UID,
// recomputes numberOfSeries/numberOfInstances, and de-duplicates. A single-instance call is valid.
func DICOMToImagingStudyR5(instances []*dicom.DataSet, opts ...Option) (*r5.ImagingStudy, *Report, error)

// DICOMToImagingStudyR4 is the R4 twin, returning *r4.ImagingStudy. The R4-vs-R5 field differences
// (modality Coding vs CodeableConcept, bodySite Coding vs CodeableReference) are handled by each form.
func DICOMToImagingStudyR4(instances []*dicom.DataSet, opts ...Option) (*r4.ImagingStudy, *Report, error)
```

The converter reads the standard Patient, General Study, General Series, and SOP Common modules. Field mapping, with the
R4/R5 release differences called out, is:

| FHIR `ImagingStudy` element | DICOM source attribute | Notes |
|------------------------------|------------------------|-------|
| `identifier` | Study Instance UID `(0020,000D)` | via `UIDIdentifierR4`/`R5`; never a `Reference` |
| `status` | — | defaulted to `available`; recorded in `Report.Defaulted` |
| `subject` | PatientID `(0010,0020)` + Issuer `(0010,0021)` | logical `Reference.identifier`, or `WithSubjectR4`/`R5` |
| `started` | StudyDate `(0008,0020)` + StudyTime `(0008,0030)` | combined to a FHIR `dateTime`; precision preserved |
| `numberOfSeries` | computed | distinct Series Instance UIDs seen |
| `numberOfInstances` | computed | distinct SOP Instance UIDs seen |
| `description` | StudyDescription `(0008,1030)` | |
| `modality` (study-level) | union of series Modality `(0008,0060)` | **R5 `CodeableConcept`; R4 `Coding`** — release-gated |
| `referrer` | ReferringPhysicianName `(0008,0090)` | name only; logical reference, no URL |
| `series[]` | per Series Instance UID `(0020,000E)` | one `ImagingStudySeries` each |
| `series.uid` | Series Instance UID `(0020,000E)` | FHIR `id`-typed UID string |
| `series.number` | SeriesNumber `(0020,0011)` | `IS` → `unsignedInt`; negative rejected |
| `series.modality` | Modality `(0008,0060)` | **R5 `CodeableConcept`; R4 `Coding`** |
| `series.description` | SeriesDescription `(0008,103E)` | |
| `series.bodySite` | BodyPartExamined `(0018,0015)` | **R5 `CodeableReference`; R4 `Coding`** |
| `series.laterality` | Laterality `(0020,0060)` | **R5 `CodeableConcept`; R4 `Coding`** |
| `series.started` | SeriesDate/Time `(0008,0021)`/`(0008,0031)` | |
| `series.instance[]` | per SOP Instance UID `(0008,0018)` | one `ImagingStudySeriesInstance` each |
| `series.instance.uid` | SOP Instance UID `(0008,0018)` | |
| `series.instance.sopClass` | SOP Class UID `(0008,0016)` | `Coding{ system:"urn:ietf:rfc:3986", code:"urn:oid:"+uid }` |
| `series.instance.number` | InstanceNumber `(0020,0013)` | `IS` → `unsignedInt` |

Release gating is mechanical and is fixed by which function the caller invokes: `DICOMToImagingStudyR5` emits the
modality, body site, and laterality elements as `CodeableConcept`/`CodeableReference`, while `DICOMToImagingStudyR4`
emits them as `Coding`, and the R5-only `procedure` and `reason` (`CodeableReference`) are mapped by the R4 form to
R4's `procedureCode`/`procedureReference` and `reasonCode`/`reasonReference` if the corresponding DICOM attributes are
present.

Lossy behaviour: attributes outside the General Study/Series/SOP-Common modules (acquisition parameters, equipment,
pixel-level metadata) are not represented in `ImagingStudy` and are recorded in `Report.Dropped`. This is expected and
correct — `ImagingStudy` is an index, not a copy of the dataset. The pixel data is never read by this converter.

## SRToDiagnosticReport and DiagnosticReportToSR

A DICOM Structured Report is a tree of content items (PS3.3 C.17). The SR content-item model these converters pivot on
(`dicom.ContentItem`, its `ConceptNameCode`, the relationship types, and the SR document read/build entry points) is
defined in [`dicom.md`](dicom.md); this package consumes that model rather than redefining it. The `ValueType`
vocabulary is `CONTAINER`, `TEXT`, `CODE`, `NUM`, `DATETIME`, `DATE`, `TIME`, `UIDREF`, `PNAME`, `COMPOSITE`, `IMAGE`,
`WAVEFORM`, `SCOORD`, `SCOORD3D`, and `TCOORD`. Each item carries a `ConceptNameCode` (the "what is this") and a
relationship (`CONTAINS`, `HAS PROPERTIES`, `INFERRED FROM`, …) to its parent.

```go
// SRToDiagnosticReportR5 converts a DICOM SR document dataset to a FHIR R5 DiagnosticReport plus the Observations
// that carry its measurements. The returned Observations are referenced from report.Result via intra-call
// logical references; assemble them into a transaction Bundle to persist. An R4 twin returns *r4 types.
func SRToDiagnosticReportR5(sr *dicom.DataSet, opts ...Option) (*r5.DiagnosticReport, []*r5.Observation, *Report, error)
func SRToDiagnosticReportR4(sr *dicom.DataSet, opts ...Option) (*r4.DiagnosticReport, []*r4.Observation, *Report, error)

// DiagnosticReportToSRR5 builds a Basic Text or Comprehensive SR document dataset from a FHIR R5 DiagnosticReport
// and its Observations. WithUIDRoot supplies the org root for the SOP/Study/Series UIDs that must be minted.
func DiagnosticReportToSRR5(
    report *r5.DiagnosticReport, observations []*r5.Observation, opts ...Option,
) (*dicom.DataSet, *Report, error)
func DiagnosticReportToSRR4(
    report *r4.DiagnosticReport, observations []*r4.Observation, opts ...Option,
) (*dicom.DataSet, *Report, error)
```

`SRToDiagnosticReportR5`/`R4` mapping (FHIR element names shown for R5; the release-gated rows note the R4 difference):

| FHIR `DiagnosticReport` element | DICOM SR source | Notes |
|----------------------------------|-----------------|-------|
| `identifier` | SOP Instance UID `(0008,0018)` | via `UIDIdentifierR4`/`R5` |
| `status` | CompletionFlag `(0040,A491)` + VerificationFlag `(0040,A493)` | `COMPLETE`+`VERIFIED` → `final`; `PARTIAL` → `preliminary` |
| `code` | root `CONTAINER` ConceptNameCodeSequence `(0040,A043)` | `Coding` from the `Code` triplet |
| `category` | Modality `(0008,0060)` (`SR`) → imaging category | |
| `subject` | PatientID, or `WithSubjectR4`/`R5` | logical reference (see identity) |
| `effectiveDateTime` | ContentDate/Time `(0008,0023)`/`(0008,0033)` | |
| `issued` | VerificationDateTime `(0040,A030)` | |
| `conclusion` | concatenated `TEXT` items under the impression container | FHIR `markdown` |
| `result[]` | each `NUM`/`CODE`/`TEXT` measurement leaf | one `Observation` per leaf, referenced logically |
| `study` (R5) / `imagingStudy` (R4) | CurrentRequestedProcedureEvidenceSequence | **release-gated field name**; via `UIDIdentifier` references |
| `presentedForm` | — | not produced; rendered PDF SR is out of scope, recorded in `Report` |

Each measurement leaf becomes an `Observation` via the shared `ContentItemToObservationR4`/`R5` helper (below). The SR
tree's nesting is flattened: a `CONTAINER` with a `ConceptNameCode` of an organising heading becomes an
`Observation.category` or a grouping `hasMember` relationship, not a separate report.

`DiagnosticReportToSRR5`/`R4` is the reverse. It builds a TID-1500-shaped Comprehensive SR (or a Basic Text SR when the
report has only narrative): the report `code` becomes the root container concept name, `conclusion` becomes a `TEXT`
item under an impression container, and each `Observation` becomes a `NUM`, `CODE`, or `TEXT` content item by inspecting
its `value[x]`. The Study/Series/SOP Instance UIDs of the produced SR are minted under `WithUIDRoot`; the patient and
accession identifiers are written back from the report's `subject.identifier` and `basedOn`. Loss here is the inverse:
FHIR extensions and `Reference.reference` URLs that have no DICOM home are dropped and reported.

The `ContentItem`/`Observation` element-level helpers express the leaf mapping once and are reused by both directions:

```go
// ContentItemToObservationR5 maps one SR measurement/code/text leaf to a FHIR R5 Observation.
// CONTAINER and spatial-coordinate items return (nil, false): they are structure, not observations.
func ContentItemToObservationR5(item dicom.ContentItem, opts ...Option) (*r5.Observation, bool)
func ContentItemToObservationR4(item dicom.ContentItem, opts ...Option) (*r4.Observation, bool)

func ObservationToContentItemR5(o *r5.Observation, opts ...Option) (dicom.ContentItem, error)
func ObservationToContentItemR4(o *r4.Observation, opts ...Option) (dicom.ContentItem, error)
```

The leaf value mapping is by SR `ValueType`:

| SR `ValueType` | FHIR `Observation.value[x]` | Notes |
|----------------|------------------------------|-------|
| `NUM` | `valueQuantity` | MeasuredValue `(0040,A30A)` → `Quantity.value` (a `Decimal`); MeasurementUnits `(0040,08EA)` → `Quantity.code`/`unit` |
| `CODE` | `valueCodeableConcept` | ConceptCodeSequence `Code` triplet → `Coding` |
| `TEXT` | `valueString` | TextValue `(0040,A160)` |
| `DATE` / `TIME` / `DATETIME` | `valueDateTime` / `valueTime` | precision preserved by the shared `dicom`/`fhir` date types |
| `PNAME` | `valueString` | PersonName `(0040,A123)` formatted; no FHIR `HumanName` value[x] exists |
| `UIDREF` / `COMPOSITE` / `IMAGE` | — | become `derivedFrom`/`Identifier` references, not a `value[x]` |

In every case the item's `ConceptNameCode` triplet becomes `Observation.code` (a `CodeableConcept` with one `Coding`),
mapping the DICOM `Code(value, scheme_designator, meaning)` to FHIR `Coding(code, system, display)`. The
`scheme_designator` (`DCM`, `SCT`, `LN`, …) is mapped to its registered FHIR `system` URI; an unknown scheme is
carried as `urn:dicom:scheme:<designator>` and recorded as a `Substitution`.

## ORUToDiagnosticReport

```go
// ORUToDiagnosticReportR5 converts an HL7 v2 ORU^R01 result message to a FHIR R5 DiagnosticReport and its Observations.
// One OBR (order/observation request) becomes the DiagnosticReport; each subordinate OBX becomes an Observation.
func ORUToDiagnosticReportR5(
    msg *hl7v2.Message, opts ...Option,
) (*r5.DiagnosticReport, []*r5.Observation, *Report, error)
func ORUToDiagnosticReportR4(
    msg *hl7v2.Message, opts ...Option,
) (*r4.DiagnosticReport, []*r4.Observation, *Report, error)
```

The message is read through the typed segment API (`msg.OBR()`, the `OBX` repetitions under each `OBR`) — callers of
this converter never index by position. Mapping the `OBR` segment to `DiagnosticReport`:

| FHIR `DiagnosticReport` element | HL7 v2 source | Notes |
|----------------------------------|---------------|-------|
| `identifier` | OBR-2 (Placer) and OBR-3 (Filler Order Number, `EI`) | each → `Identifier`, system from the namespace component |
| `status` | OBR-25 Result Status | `F`→`final`, `P`→`preliminary`, `C`→`corrected`, `X`→`cancelled` |
| `code` | OBR-4 Universal Service Identifier (`CWE`) | → `CodeableConcept` |
| `category` | OBR-24 Diagnostic Service Section ID | e.g. `RAD` → imaging category |
| `subject` | `PID-3` patient identifier list (`CX`) | logical reference, or `WithSubjectR4`/`R5` |
| `effectiveDateTime` | OBR-7 Observation Date/Time (`DTM`) | precision preserved |
| `issued` | OBR-22 Results Rpt/Status Chng Date/Time | |
| `resultsInterpreter` | OBR-32 Principal Result Interpreter | logical reference |
| `result[]` | each `OBX` | via `OBXToObservationR4`/`R5`, referenced logically |
| `conclusion` | `OBX` rows of value type `TX`/`FT` flagged as impression | concatenated `markdown` |

The shared OBX-level helpers:

```go
// OBXToObservationR5 maps one HL7 v2 OBX segment to a FHIR R5 Observation. The enclosing message supplies PID/PV1
// context for subject and encounter resolution; pass it so the Observation is self-consistent.
func OBXToObservationR5(obx hl7v2.OBX, ctx *hl7v2.Message, opts ...Option) (*r5.Observation, *Report, error)
func OBXToObservationR4(obx hl7v2.OBX, ctx *hl7v2.Message, opts ...Option) (*r4.Observation, *Report, error)

func ObservationToOBXR5(o *r5.Observation, setID int, opts ...Option) (hl7v2.OBX, error)
func ObservationToOBXR4(o *r4.Observation, setID int, opts ...Option) (hl7v2.OBX, error)
```

`OBXToObservationR5`/`R4` mapping, grounded in the canonical OBX layout
(`OBX|1|SN|1554-5^GLUCOSE^...||^182|mg/dl|70_105|H|||F`):

| FHIR `Observation` element | HL7 v2 OBX field | Notes |
|-----------------------------|------------------|-------|
| `code` | OBX-3 Observation Identifier (`CWE`) | → `CodeableConcept`; e.g. `1554-5^GLUCOSE^LN` |
| `value[x]` | OBX-5 Observation Value, typed by OBX-2 | see value-type table below |
| `valueQuantity.unit`/`code` | OBX-6 Units (`CWE`) | UCUM where the sender uses it |
| `referenceRange.text` | OBX-7 References Range | parsed to low/high when numeric, else `text` |
| `interpretation` | OBX-8 Abnormal Flags | HL7 Table 0078 → FHIR interpretation `Coding` |
| `status` | OBX-11 Observation Result Status | `F`→`final`, `P`→`preliminary`, `C`→`corrected`, `X`→`cancelled` |
| `effectiveDateTime` | OBX-14 Date/Time of the Observation | `DTM`, precision preserved |
| `performer` | OBX-16 Responsible Observer | logical reference |
| `subject` | enclosing `PID-3` | logical reference, or `WithSubjectR4`/`R5` |
| `encounter` | enclosing `PV1-19` Visit Number | logical reference |

OBX value-type (OBX-2) to FHIR `value[x]`:

| OBX-2 | FHIR `value[x]` | Notes |
|-------|------------------|-------|
| `NM` | `valueQuantity` | numeric value parsed into a `Decimal`, lexical form preserved |
| `SN` | `valueQuantity` or `valueRange` | structured numeric: comparator + number; ranges → `valueRange` |
| `CE` / `CWE` / `CNE` | `valueCodeableConcept` | coded value |
| `ST` / `TX` / `FT` | `valueString` | free or formatted text (formatting escapes stripped) |
| `DT` / `TM` / `TS` | `valueDateTime` / `valueTime` | precision preserved |
| `CX` / `XCN` | `valueString` | identifier/name rendered to string; no FHIR identifier-valued `value[x]` |

Lossy behaviour: HL7 fields with no FHIR Observation home (OBX-17 method when uncoded, sub-ID grouping in OBX-4 beyond
`hasMember` linking) are recorded in `Report.Dropped`. Numeric reference ranges that cannot be parsed are kept verbatim
as `referenceRange.text` and noted as a `Substitution`.

## ORMToServiceRequest

```go
// ORMToServiceRequestR5 converts an HL7 v2 order message (ORM^O01 or OMG^O19) to a FHIR R5 ServiceRequest.
// One hl7v2.OrderGroup{Common ORC; Requests []OBR} becomes one ServiceRequest; a message with multiple order
// groups is rejected with ErrUnsupportedSource in v1 (single-order-per-call is the documented v1 limit) — split
// the message into its OrderGroups upstream and call once per group.
func ORMToServiceRequestR5(msg *hl7v2.Message, opts ...Option) (*r5.ServiceRequest, *Report, error)
func ORMToServiceRequestR4(msg *hl7v2.Message, opts ...Option) (*r4.ServiceRequest, *Report, error)
```

The single-order-per-call limit is deliberate for v1: it keeps each `ServiceRequest` mapping unambiguous and pushes
the fan-out decision to the caller, which already has the `hl7v2.OrderGroup` repetitions from the message. A
multi-order convenience wrapper returning `[]*r5.ServiceRequest` is deferred to a later release (PRD §3.2). There is
no FHIR `Order` resource; the order maps to `ServiceRequest` (glossary). Mapping reads ORC (common order) and OBR
(observation request):

| FHIR `ServiceRequest` element | HL7 v2 source | Notes |
|-------------------------------|---------------|-------|
| `identifier` | ORC-2 Placer + ORC-3 Filler Order Number (`EI`) | each → `Identifier` |
| `status` | ORC-5 Order Status / ORC-1 Order Control | `NW`→`active`, `CA`→`revoked`, `CM`→`completed` |
| `intent` | — | defaulted to `order`; recorded in `Report.Defaulted` |
| `priority` | OBR-27.6 / ORC-7 Quantity-Timing priority | `S`→`stat`, `R`→`routine`, `A`→`asap` |
| `code` | OBR-4 Universal Service Identifier (`CWE`) | **R5 `CodeableReference`; R4 `CodeableConcept`** — release-gated |
| `subject` | `PID-3` (`CX`) | logical reference, or `WithSubjectR4`/`R5` |
| `encounter` | `PV1-19` Visit Number | logical reference |
| `authoredOn` | ORC-9 Date/Time of Transaction (`DTM`) | |
| `requester` | ORC-12 Ordering Provider (`XCN`) | logical reference, name + identifier |
| `reason` (R5) / `reasonCode` (R4) | OBR-31 Reason for Study (`CWE`) | **release-gated field name and type** |
| `bodySite` | OBR-15.4 / study body part when present | `CodeableConcept` |
| `occurrenceDateTime` | OBR-6 / OBR-27 requested date/time | |

Loss: ORC scheduling and timing detail beyond the single occurrence, and any local Z-segments, are dropped and reported.

## ADTToPatient and ADTToEncounter

The HL7 v2 `ADT` (admit/discharge/transfer) message carries both demographic and visit data; go-radx splits this into
two converters because `Patient` and `Encounter` are distinct FHIR resources with distinct lifecycles (glossary:
`Encounter` is a FHIR-only noun). A typical caller runs both on the same message and assembles a transaction `Bundle`.

```go
// ADTToPatientR5 converts the PID (and PD1) segments of an ADT message to a FHIR R5 Patient.
func ADTToPatientR5(msg *hl7v2.Message, opts ...Option) (*r5.Patient, *Report, error)
func ADTToPatientR4(msg *hl7v2.Message, opts ...Option) (*r4.Patient, *Report, error)

// ADTToEncounterR5 converts the PV1 segment and the ADT trigger event (EVN) to a FHIR R5 Encounter.
// The produced Encounter references the Patient logically via PID-3; pass WithSubjectR5 for a concrete reference.
func ADTToEncounterR5(msg *hl7v2.Message, opts ...Option) (*r5.Encounter, *Report, error)
func ADTToEncounterR4(msg *hl7v2.Message, opts ...Option) (*r4.Encounter, *Report, error)
```

`ADTToPatientR5`/`R4` mapping from the `PID` segment:

| FHIR `Patient` element | HL7 v2 PID field | Notes |
|-------------------------|------------------|-------|
| `identifier` | PID-3 Patient Identifier List (`CX`, repeating) | each `CX` → `Identifier`, system from assigning authority (`HD`) |
| `name` | PID-5 Patient Name (`XPN`, repeating) | `XPN` → `HumanName`; use code 'official' for the legal name |
| `birthDate` | PID-7 Date/Time of Birth (`DTM`) | date precision preserved |
| `gender` | PID-8 Administrative Sex | mapped to the release `AdministrativeGender` enum via `r5.ParseAdministrativeGender` (R4: `r4.`) |
| `address` | PID-11 Patient Address (`XAD`, repeating) | `XAD` → `Address` |
| `telecom` | PID-13 Home / PID-14 Business Phone | `XTN` → `ContactPoint` |
| `maritalStatus` | PID-16 Marital Status (`CWE`) | `CodeableConcept` |
| `deceasedDateTime` | PID-29 Patient Death Date/Time | when PID-30 indicates deceased |
| `communication.language` | PID-15 Primary Language (`CWE`) | |

The PID-8 sex code is the canonical place the value-set-binding safety shows up: an unknown PID-8 value does not
silently become an empty string. The release `ParseAdministrativeGender` (`r5.ParseAdministrativeGender`, or `r4.`)
returns an error for an out-of-binding code, and the converter records a `Substitution` mapping the unknown code to
`unknown` (the binding's documented unknown-code policy) rather than dropping the patient's stated sex without trace.

`ADTToEncounterR5`/`R4` mapping from `PV1` and `EVN`:

| FHIR `Encounter` element | HL7 v2 source | Notes |
|---------------------------|---------------|-------|
| `identifier` | PV1-19 Visit Number (`CX`) | → `Identifier` |
| `status` | ADT trigger event (MSH-9.2) | `A01`→`in-progress`, `A03`→`completed`, `A11`→`cancelled` (cancel admit) |
| `class` | PV1-2 Patient Class | **R5 `class` is `CodeableConcept` list; R4 is single `Coding`** — release-gated |
| `subject` | `PID-3` | logical reference, or `WithSubjectR4`/`R5` |
| `actualPeriod` (R5) / `period` (R4) | PV1-44 Admit / PV1-45 Discharge Date/Time | **release-gated field name** |
| `location` | PV1-3 Assigned Patient Location (`PL`) | `Encounter.location.location` logical reference |
| `serviceType` | PV1-10 Hospital Service | |
| `participant` | PV1-7 Attending / PV1-8 Referring / PV1-9 Consulting Doctor | each `XCN` → `participant` with a role `Coding` |

The trigger-event-to-status mapping is the load-bearing decision: the same `PV1` produces a different `Encounter.status`
depending on whether the message is an admit (`A01`), discharge (`A03`), or update (`A08`). The converter reads the
trigger event from MSH-9.2 (the only authoritative place per the glossary) and records the chosen status as a
`Substitution` so the mapping decision is auditable.

## Worked usage examples

### DICOM study to a FHIR R5 ImagingStudy

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/convert"
    "github.com/codeninja55/go-radx/dicom"
)

func main() {
    var instances []*dicom.DataSet
    for _, path := range []string{"img001.dcm", "img002.dcm"} {
        ds, err := dicom.ReadFile(path)
        if err != nil {
            log.Fatalf("read %s: %v", path, err)
        }
        instances = append(instances, ds)
    }

    // The R5 form returns *r5.ImagingStudy; call DICOMToImagingStudyR4 for an *r4.ImagingStudy.
    study, report, err := convert.DICOMToImagingStudyR5(instances)
    if err != nil {
        log.Fatalf("convert: %v", err) // malformed source or missing Study Instance UID
    }

    // Honest reporting: surface what did not map cleanly rather than assuming a clean conversion.
    for _, d := range report.Dropped {
        fmt.Printf("dropped: %s — %s\n", d.Source, d.Reason)
    }

    // FHIR resources marshal with the standard library; there is no fhir.Marshal (see fhir.md).
    data, err := json.Marshal(study)
    if err != nil {
        log.Fatalf("marshal: %v", err)
    }
    fmt.Println(string(data))
}
```

### HL7 ORU to a FHIR DiagnosticReport, fail-closed on loss

```go
msg, err := hl7v2.Parse(raw) // raw is one ORU^R01 message
if err != nil {
    log.Fatalf("parse ORU: %v", err)
}

report, observations, conv, err := convert.ORUToDiagnosticReportR5(
    msg,
    convert.WithStrictLoss(), // turn any silent drop into a returned error
)
var loss *convert.LossError
switch {
case errors.As(err, &loss):
    for _, d := range loss.Dropped {
        log.Printf("unmappable: %s — %s", d.Source, d.Reason)
    }
    return // policy decision: this consumer will not accept lossy result conversion
case err != nil:
    log.Fatalf("convert ORU: %v", err)
}

// report.Result references each Observation via an intra-call urn:uuid; assemble a transaction Bundle to persist.
fmt.Printf("report status=%s with %d observations\n", report.Status, len(observations))
_ = conv // a non-nil Report even on success; empty here because StrictLoss was set
```

### ADT to Patient and Encounter together

```go
// This example targets R5; the Patient/Encounter and Reference types come from the fhir/r5 sub-package.
import "github.com/codeninja55/go-radx/fhir/r5"

msg, _ := hl7v2.Parse(adtRaw)

patient, pReport, err := convert.ADTToPatientR5(msg)
if err != nil {
    log.Fatalf("ADT to Patient: %v", err)
}

// Reuse the Patient's logical identity as the Encounter subject so both resources cohere.
subject := r5.Reference{
    Type:       "Patient",
    Identifier: &patient.Identifier[0], // the PID-3 identifier carried across, not a fabricated URL
}
enc, eReport, err := convert.ADTToEncounterR5(msg, convert.WithSubjectR5(subject))
if err != nil {
    log.Fatalf("ADT to Encounter: %v", err)
}

_ = pReport
_ = eReport
fmt.Printf("patient gender=%s, encounter status=%s\n", patient.Gender, enc.Status)
```

## Conformance scope and limits

What the `convert` package guarantees, and what it deliberately does not:

- **Scope is the workflow set, not every resource.** Only the conversions in the table above are implemented in v1. A
  source it cannot handle (an unsupported message type, a non-imaging SOP class to `ImagingStudy`, a multi-order `ORM`)
  returns `ErrUnsupportedSource` — it never produces a partial or guessed result. This is the §9.2 fail-closed rule.
- **R4 4.0.1 and R5 5.0.0 only.** Release is selected by which function the caller invokes (`…R4` vs `…R5`); the
  return type is the matching `fhir/r4` or `fhir/r5` sub-package type. Field-name and datatype differences between
  releases (modality `Coding` vs `CodeableConcept`, `imagingStudy` vs `study`, `period` vs `actualPeriod`, single vs
  list `Encounter.class`) are handled mechanically by each form. STU3 and R4B are out of scope (PRD §5.3).
- **Identity is preserved, never fabricated.** DICOM UIDs become FHIR `Identifier`s under `urn:dicom:uid`/`urn:oid` and
  HL7 `CX` identifiers become `Identifier`s under their assigning authority. No converter invents a server-resolvable
  `Reference.reference` URL; absent a `WithSubjectR4`/`WithSubjectR5`, the patient/encounter link is a logical
  `Reference.identifier`. A caller that needs resolvable references resolves them against their own server after
  assembling a `Bundle`.
- **Loss is reported, never hidden.** Every successful conversion returns a `*Report` enumerating dropped, defaulted,
  and substituted fields, named by concept (DICOM keyword + tag, HL7 accessor, FHIR path). `WithStrictLoss` escalates
  any drop to a returned `*LossError`. There is no mode in which loss occurs silently.
- **No terminology translation.** A `Coding` is carried across verbatim, with the scheme designator mapped to its
  registered FHIR `system` URI. go-radx does not translate between SNOMED CT, LOINC, RadLex, or local code systems;
  unknown schemes are preserved under a `urn:dicom:scheme:<designator>` system and flagged as substitutions.
- **No PHI in diagnostics.** `Report` and error strings carry identifiers and structure, not patient values, honouring
  the §9.1 no-PHI-by-default rule. Verbosity that would surface patient data is opt-in and out of this package's scope.
- **Produced resources are valid but not profile-conformant.** Output validates against the base FHIR R4/R5
  StructureDefinitions (the §11.1 FHIR-validator gate), but go-radx does not assert US Core or any other IG profile
  conformance in v1 (deferred, PRD §3.2). A consumer that needs profile conformance validates the output against their
  profile separately.
- **SR coverage is the measurement/text/code leaf set.** `SRToDiagnosticReportR4`/`R5` maps
  `NUM`/`CODE`/`TEXT`/date-time leaves to `Observation`s and the document narrative to `conclusion`, reading the
  `dicom.ContentItem` SR model defined in [`dicom.md`](dicom.md). Spatial and temporal coordinate items
  (`SCOORD`/`SCOORD3D`/`TCOORD`), waveform and image references become `derivedFrom`/`Identifier` links rather than
  observation values, and rendered presentation forms (PDF SR) are not produced. The limits are recorded per conversion
  in the `Report`.
