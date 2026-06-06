# Cross-standard conversion conformance statement

> **Implementation status: NOT YET SHIPPED.** This is a scaffold. The conversion conformance statement — the versioned
> contract for what the `convert` package maps between DICOM, HL7 v2, and FHIR, and what fidelity each conversion
> guarantees — is not yet authored. Until this banner is removed, **no conversion is conformance-guaranteed**. The
> `convert` package implements a partial R5-only set of converters; the full release-explicit matrix (R4 twins and the
> remaining workflow conversions) is not yet present. Do not cite this document as a conformance basis.

| Field | Value |
|-------|-------|
| Standards bridged | DICOM (NEMA PS3), HL7 v2.x, HL7 FHIR (R4 4.0.1, R5 5.0.0) |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | unassigned (statement not yet authored) |
| Status | **NOT YET SHIPPED** — scaffold only |
| Scope authority | This document will become the single source of truth for conversion scope (PRD §6.1) |

This document will be the conversion conformance statement: it will declare which cross-standard mappings the `convert`
package supports, the release suffix on each FHIR-producing converter, the loss policy (strict versus lossy), and how
each mapping is validated against the FHIR validator and the workflow fixtures. The per-standard scope contracts are
the DICOM ([`./dicom.md`](./dicom.md)), HL7 v2 ([`./hl7v2.md`](./hl7v2.md)), and FHIR ([`./fhir.md`](./fhir.md))
conformance statements; this statement bridges them and does not restate their scope.

## Scope summary

Not yet authored. This section will enumerate the in-scope conversions, each release-explicit per the glossary naming
rule `convert.<Source>To<Target><Release>`. The conversions intended to close the PRD §5.1 workflow loop are:

- `DICOMToImagingStudy` — DICOM study/series/instance grouping to FHIR `ImagingStudy`.
- `ORMToServiceRequest` — HL7 v2 `ORM`/`OMG` imaging order to FHIR `ServiceRequest`.
- `SRToDiagnosticReport` / `DiagnosticReportToSR` — DICOM Structured Report to and from FHIR `DiagnosticReport`.
- `ORUToDiagnosticReport` — HL7 v2 `ORU` result to FHIR `DiagnosticReport`.
- `ADTToPatient` / `ADTToEncounter` — HL7 v2 `ADT` demographics and visit context to FHIR `Patient` / `Encounter`.

Each is intended to ship with an `R4` and `R5` twin. The currently implemented set is R5-only and partial; the
authored statement will declare exactly which `(conversion, release)` pairs are conformance-tested.

### DICOM to ImagingStudy (`DICOMToImagingStudyR5`)

`DICOMToImagingStudyR5` groups one or more DICOM instances of a study by Series Instance UID and SOP Instance UID and
produces a FHIR R5 `ImagingStudy`. It reads the Patient, General Study, General Series, and SOP Common modules;
`numberOfSeries` and `numberOfInstances` are recomputed from the distinct UIDs seen, never copied from a possibly stale
source attribute. `ImagingStudy` is an index, not a copy of the dataset, so attributes outside those modules are
recorded in `Report.Dropped`.

| FHIR `ImagingStudy` element | DICOM source | Notes |
|------------------------------|--------------|-------|
| `identifier` | Study Instance UID `(0020,000D)` | logical `urn:dicom:uid` / `urn:oid:` identifier, never a Reference URL |
| `status` | — | defaulted to `available`; recorded in `Report.Defaulted` |
| `subject` | PatientID `(0010,0020)` + Issuer `(0010,0021)`, or `WithSubjectR5` | logical `Reference.identifier`, never a fabricated URL |
| `started` | StudyDate `(0008,0020)` + StudyTime `(0008,0030)` + TimezoneOffsetFromUTC `(0008,0201)` | precision preserved; a timezone-less time is dropped to date-only and recorded |
| `numberOfSeries` / `numberOfInstances` | computed | distinct Series / SOP Instance UIDs seen |
| `description` | StudyDescription `(0008,1030)` | |
| `modality` (study-level) | union of series Modality `(0008,0060)` | `CodeableConcept` under the DICOM `DCM` system |
| `referrer` | ReferringPhysicianName `(0008,0090)` | `Reference.display` only (rendered PN), never a fabricated URL |
| `procedure[]` | ProcedureCodeSequence `(0008,1032)` | each coded item → `CodeableReference.concept` |
| `reason[]` | ReasonForRequestedProcedureCodeSequence `(0040,100A)` + ReasonForStudy `(0032,1030)` | coded items → concept `Coding`; the free-text reason → concept `text` |
| `series.uid` | Series Instance UID `(0020,000E)` | |
| `series.modality` | Modality `(0008,0060)` | required; repaired from a later instance when the first lacks it; the series is dropped (recorded) when no instance carries one |
| `series.number` | SeriesNumber `(0020,0011)` | `IS` → `unsignedInt`; a negative or out-of-range value is dropped and recorded |
| `series.description` | SeriesDescription `(0008,103E)` | |
| `series.bodySite` | BodyPartExamined `(0018,0015)` | `CodeableReference.concept` (free-text rendering) |
| `series.laterality` | Laterality `(0020,0060)` | `CodeableConcept` `Coding` under the DICOM `DCM` system |
| `series.started` | SeriesDate `(0008,0021)` + SeriesTime `(0008,0031)` + TimezoneOffsetFromUTC `(0008,0201)` | precision preserved; a timezone-less time is dropped to date-only |
| `series.instance.uid` | SOP Instance UID `(0008,0018)` | |
| `series.instance.sopClass` | SOP Class UID `(0008,0016)` | required `Coding{ system:"urn:ietf:rfc:3986", code:"urn:oid:"+uid }`; an instance with no SOP Class UID is dropped (recorded) |
| `series.instance.number` | InstanceNumber `(0020,0013)` | `IS` → `unsignedInt`; a negative or out-of-range value is dropped and recorded |

A coded entry (procedure, reason) carries its DICOM coding-scheme designator to the registered FHIR `system` URI (`DCM`,
`SCT`, `LN`); an unknown designator is carried verbatim under `urn:dicom:scheme:<designator>` so no value is lost. The
identity rule holds throughout: a UID becomes an `Identifier`, never a `Reference.reference` URL, and a person name
becomes `Reference.display` only.

### ORM to ServiceRequest (`ORMToServiceRequestR5`)

`ORMToServiceRequestR5` converts one HL7 v2 order group (an ORC with its OBR requests) from an `ORM^O01` / `OMG^O19`
message to a FHIR R5 `ServiceRequest`. A message carrying multiple order groups is rejected fail-closed
(`ErrUnsupportedSource`): v1 maps one order per call. It reads MSH, PID, PV1, ORC, and OBR.

| FHIR `ServiceRequest` element | HL7 v2 source | Notes |
|-------------------------------|---------------|-------|
| `identifier` | ORC-2 Placer + ORC-3 Filler Order Number | each → `Identifier.value` |
| `status` | ORC-1 Order Control / ORC-5 Order Status | `NW`/`XO`→`active`, `CA`→`revoked`, `CM`→`completed`; unrecognised → `active` |
| `intent` | — | defaulted to `order`; recorded in `Report.Defaulted` |
| `priority` | ORC-7 / OBR-27 Quantity-Timing priority (component 6) | `S`→`stat`, `A`→`asap`, `R`/`P`→`routine`; an out-of-table code is dropped and recorded (the binding is required) |
| `code` | OBR-4 Universal Service Identifier (`CWE`) | `CodeableReference.concept`; extra OBR requests are dropped and recorded by locus |
| `subject` | PID-3 (`CX`), or `WithSubjectR5` | logical `Reference.identifier`, never a fabricated URL |
| `encounter` | PV1-19 Visit Number (`CX`) | logical `Reference.identifier`, never a fabricated URL |
| `authoredOn` | ORC-9 Date/Time of Transaction (`DTM`) | precision preserved; a timezone-less time is dropped to date-only and recorded |
| `requester` | ORC-12 Ordering Provider (`XCN`) | logical `Reference.identifier` (ID component) plus `Reference.display` (`family^given`), never a fabricated URL |
| `reason[]` | OBR-31 Reason for Study (`CWE`) | `CodeableReference.concept` carrying code, system, and display verbatim |
| `occurrenceDateTime` | OBR-6 / OBR-27 requested date/time (`DTM`) | precision preserved; a timezone-less time is dropped to date-only and recorded |

`bodySite` has no `ORM^O01` / `OMG^O19` v1 source field, so it is left unset; an imaging order conveys the body region
through the ordered procedure code (OBR-4), not a discrete field. go-radx does not translate between code systems; a
`CWE` coding system is carried verbatim. The identity rule holds throughout: every patient, encounter, and requester
link is a logical `Reference.identifier` (with an optional non-resolving `display`), never a `Reference.reference` URL.

### SR to DiagnosticReport (`SRToDiagnosticReportR5`)

`SRToDiagnosticReportR5` reads a DICOM Structured Report document dataset and produces a FHIR R5 `DiagnosticReport`
together with the set of `Observation`s carrying its measurements. The document-level attributes map to the report; the
SR content tree is walked depth-first for measurement leaves, and each becomes one `Observation`.

| FHIR `DiagnosticReport` element | DICOM SR source | Notes |
|---------------------------------|-----------------|-------|
| `identifier` | SOP Instance UID `(0008,0018)` | logical `urn:dicom:uid` / `urn:oid:` identifier, never a Reference URL |
| `status` | CompletionFlag `(0040,A491)` + VerificationFlag `(0040,A493)` | `COMPLETE`+`VERIFIED` → `final`, otherwise `preliminary`; absent flag defaulted |
| `category` | imaging service section | fixed `IMG` coding |
| `code` | root `CONTAINER` Concept Name Code Sequence `(0040,A043)` | required; the conversion fails closed when absent |
| `effectiveDateTime` | ContentDate `(0008,0023)` + ContentTime `(0008,0033)` + TimezoneOffsetFromUTC `(0008,0201)` | a timezone-less time is dropped to date-only and recorded |
| `conclusion` | concatenated `TEXT` content items | narrative items join in document order |
| `result[]` | each `NUM`/`CODE`/date-time measurement leaf | one `Observation` per leaf, linked by an intra-call `urn:uuid` reference |
| `subject` | PatientID `(0010,0020)` or `WithSubjectR5` | logical `Reference.identifier`, never a fabricated URL |

The `result[]` links are derived deterministically from the SR SOP Instance UID and each leaf's position (an RFC 4122
version-5 name-based UUID), so the same SR produces byte-identical output. `TEXT` leaves form the narrative conclusion
rather than separate observations, so a finding is represented once.

Each measurement leaf maps to an `Observation` through the shared `ContentItemToObservationR5` helper. The leaf's
Concept Name Code Sequence becomes the required `Observation.code`, the status is `final`, and the value maps by SR
`ValueType`:

| SR `ValueType` | FHIR `Observation.value[x]` | Notes |
|----------------|-----------------------------|-------|
| `NUM` | `valueQuantity` | MeasuredValue `(0040,A30A)` → `Quantity.value`; MeasurementUnits `(0040,08EA)` → `Quantity.code`/`unit`/`system` (UCUM → `http://unitsofmeasure.org`) |
| `CODE` | `valueCodeableConcept` | Concept Code Sequence `(0040,A168)` triplet → `Coding` |
| `TEXT` | `valueString` | Text Value `(0040,A160)`; in `SRToDiagnosticReportR5` the report-level walk routes `TEXT` to `conclusion` instead |
| `DATE` / `TIME` / `DATETIME` | `valueDateTime` | day precision yields `YYYY-MM-DD`; a time is rendered only with a UTC offset, else dropped to date-only |

The DICOM coding-scheme designator maps to its registered FHIR `system` URI (`DCM`, `SCT`, `LN`, `UCUM`); an unknown
designator is carried verbatim under `urn:dicom:scheme:<designator>` so no value is lost.

## Loss policy

Not yet authored. This section will declare the strict-versus-lossy contract: which conversions fail closed on
unmappable input (the `WithStrictLoss` option) and which record loss without rejecting, so a consumer knows when a
conversion is total and when it is best-effort.

## Verification

Not yet authored. This section will state how each conversion is validated: golden round-trip fixtures, the HL7 FHIR
validator on produced resources, and the end-to-end walking-skeleton interop suite, plus the CI job that invokes them.
Until then, no conversion conformance claim is made.

## References

- go-radx PRD §6.1 (conformance definition), §5.1 (workflow scope), §5.3 (FHIR releases).
- go-radx reference docs: `docs/reference/convert.md`.
- Per-standard conformance statements: [`./dicom.md`](./dicom.md), [`./hl7v2.md`](./hl7v2.md), [`./fhir.md`](./fhir.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
