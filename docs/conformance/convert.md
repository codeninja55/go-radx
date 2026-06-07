# Cross-standard conversion conformance statement

> **Implementation status: NOT YET SHIPPED.** This is a scaffold. The conversion conformance statement — the versioned
> contract for what the `convert` package maps between DICOM, HL7 v2, and FHIR, and what fidelity each conversion
> guarantees — is not yet authored. Until this banner is removed, **no conversion is conformance-guaranteed**. The
> `convert` package now implements the workflow conversions for both FHIR releases: each forward converter and the
> SR/OBX reverse converters have an `R4` (4.0.1) and an `R5` (5.0.0) twin, and every twin's output is validated through
> its release validator. The remaining workflow conversions outside the §5.1 loop are not yet present. Do not cite this
> document as a conformance basis.

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

Each ships with an `R4` and an `R5` twin. The forward set (`DICOMToImagingStudyR4`/`R5`,
`ORMToServiceRequestR4`/`R5`, `SRToDiagnosticReportR4`/`R5`, `ORUToDiagnosticReportR4`/`R5`, `ADTToPatientR4`/`R5`,
`ADTToEncounterR4`/`R5`) and the SR/OBX reverse set (`DiagnosticReportToSR` / `DiagnosticReportToSRR4`,
`ObservationToContentItem` / `ObservationToContentItemR4`, `ObservationToOBX` / `ObservationToOBXR4`) are present for
both releases. The authored statement will declare exactly which `(conversion, release)` pairs are conformance-tested;
the implementation today validates every twin's output through its release validator (`r4.Validate` / `r5.Validate`).

The R5 mapping is documented per converter below. The R4 twin of each converter reads the same DICOM/HL7 source, applies
the same loss/substitution policy, and produces the equivalent FHIR resource in the `fhir/r4` type space; the
[R4 twins](#r4-twins) section records only the load-bearing R4/R5 datatype differences each twin reconciles, rather than
restating every field row.

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

### DiagnosticReport to SR (`DiagnosticReportToSR`)

`DiagnosticReportToSR` is the reverse of `SRToDiagnosticReportR5`: it reads a FHIR R5 `DiagnosticReport` together with
its `Observation`s and produces a DICOM Structured Report document dataset (Comprehensive SR IOD). The report's code
becomes the root `CONTAINER` Concept Name Code Sequence; the conclusion becomes a `TEXT` child; each `Observation`
becomes a measurement leaf through the shared `ObservationToContentItem` helper.

| DICOM SR target | FHIR `DiagnosticReport` source | Notes |
|-----------------|--------------------------------|-------|
| root `CONTAINER` Concept Name Code Sequence `(0040,A043)` | `code` | required; the conversion fails closed when the code maps to no concept |
| `TEXT` child | `conclusion` | the narrative re-encodes as a single `TEXT` content item |
| measurement leaves | each `Observation` | one content item per Observation via `ObservationToContentItem` |
| CompletionFlag `(0040,A491)` / VerificationFlag `(0040,A493)` | `status` | `final` → `COMPLETE`+`VERIFIED`; any other status → `COMPLETE`+`UNVERIFIED` |
| ContentDate `(0008,0023)` + ContentTime `(0008,0033)` + TimezoneOffsetFromUTC `(0008,0201)` | `effectiveDateTime` | split back into date, time, and the document-level offset |
| PatientID `(0010,0020)` | `subject` logical `Reference.identifier` | a subject carried as a resolvable `Reference.reference` URL has no DICOM PatientID home and is recorded |
| Study / Series / SOP Instance UID `(0020,000D)` / `(0020,000E)` / `(0008,0018)` | minted | deterministic under `WithUIDRoot`; see below |

The Study, Series, and SOP Instance UIDs are minted deterministically under the organisation root supplied by
`WithUIDRoot`, derived from the report's identity (its DICOM UID identifier when present, otherwise its code and
conclusion) and a per-UID role label, so the same report mints byte-identical UIDs across runs and a full round trip
from one SR source re-derives the same triple. Absent a configured root, the document carries no UIDs and the absence
is recorded in `Report.Defaulted` — go-radx ships no default registered root and never fabricates an unregistered UID.

Each `Observation` re-encodes through `ObservationToContentItem`, the inverse of `ContentItemToObservationR5`. The
`Observation.code` becomes the leaf's Concept Name Code Sequence and the `value[x]` branch chooses the SR `ValueType`:

| FHIR `Observation.value[x]` | SR `ValueType` | Notes |
|-----------------------------|----------------|-------|
| `valueQuantity` | `NUM` | `Quantity.value` → MeasuredValue `(0040,A30A)`; `Quantity.code`/`unit`/`system` → MeasurementUnits `(0040,08EA)` (the UCUM system URI maps back to the `UCUM` designator) |
| `valueCodeableConcept` | `CODE` | first `Coding` → Concept Code Sequence `(0040,A168)` triplet |
| `valueString` | `TEXT` | → Text Value `(0040,A160)` |
| `valueDateTime` | `DATE` / `DATETIME` | a date-only value yields `DATE`; a value with a time yields `DATETIME`, the offset re-encoded as the DICOM `&ZZXX` form |
| `valueTime` | — | recorded as loss; see the loss policy |

The FHIR `system` URI maps back to its DICOM coding-scheme designator (`DCM`, `SCT`, `LN`, `UCUM`); a system under the
synthetic `urn:dicom:scheme:<designator>` prefix is unwrapped to the verbatim designator it preserved on the forward
path, so a code round-trips its scheme. An `Observation` with no code, or whose `value[x]` is unset or has no SR form,
is not emitted and the loss is recorded by FHIR element path — never the clinical value.

### ORU to DiagnosticReport (`ORUToDiagnosticReportR5`)

`ORUToDiagnosticReportR5` converts an HL7 v2 observation result message (`ORU^R01`) to a FHIR R5 `DiagnosticReport`
together with the set of `Observation`s carrying its results. The first result group's OBR becomes the report; each OBX
that follows becomes one `Observation`, linked from `DiagnosticReport.result` by an intra-call `urn:uuid` logical
reference. An ORU with no OBR is rejected fail-closed (`ErrMalformedSource`): `DiagnosticReport.code` is required and
OBR-4 is its only source. A panel split across additional OBRs is recorded by locus in `Report.Dropped` (v1 maps the
first result group).

| FHIR `DiagnosticReport` element | HL7 v2 source | Notes |
|---------------------------------|---------------|-------|
| `code` | OBR-4 Universal Service Identifier (`CWE`) | required; the conversion fails closed when absent |
| `status` | OBR-25 Result Status (Table 0123) | `F`→`final`, `P`→`preliminary`, `C`→`corrected`, `X`→`cancelled`; an absent or out-of-table code defaults to `final` and is recorded (the binding is required) |
| `effectiveDateTime` | OBR-7 Observation Date/Time (`DTM`) | precision preserved; a timezone-less time is dropped to date-only and recorded |
| `result[]` | each OBX result | one `Observation` per OBX, linked by an intra-call `urn:uuid` reference |
| `subject` | PID-3 (`CX`), or `WithSubjectR5` | logical `Reference.identifier`, never a fabricated URL |

The `result[]` links are derived deterministically from the MSH-10 message control ID and each OBX's position (an RFC
4122 version-5 name-based UUID), so the same ORU produces byte-identical output. The control ID is locally unique, not a
patient value.

Each OBX maps to an `Observation` through the shared `OBXToObservationR5` helper. OBX-3 becomes the required
`Observation.code`, the status is `final`, and the value maps by OBX-2 `ValueType`:

| OBX-2 `ValueType` | FHIR `Observation.value[x]` | Notes |
|-------------------|-----------------------------|-------|
| `NM` | `valueQuantity` | OBX-5 → `Quantity.value` (a `Decimal`, lexical precision preserved); OBX-6 units (`CWE`) → `Quantity.code`/`unit`/`system` (`UCUM` → `http://unitsofmeasure.org`, otherwise the system carried verbatim) |
| `CE` / `CWE` | `valueCodeableConcept` | OBX-5 `code^text^system` → `Coding` |
| `TX` / `ST` / `FT` | `valueString` | OBX-5 carried verbatim |
| `DT` / `TS` | `valueDateTime` | day precision yields `YYYY-MM-DD`; a time is rendered only with a UTC offset, else dropped to date-only |
| `TM` | `valueTime` | rendered `hh:mm:ss`; FHIR `time` carries no timezone |

OBX-8 abnormal flags (Table 0078) map to `Observation.interpretation`: the HL7 and FHIR codes share the same symbols, so
each flag is carried verbatim under the `v3-ObservationInterpretation` system. OBX-7 maps to a single
`Observation.referenceRange`: a `low-high` form becomes `low`/`high` bare-value `Quantity`s, and any other form is
carried as `referenceRange.text` so the range is preserved. An OBX whose OBX-2 value type is outside the supported set
leaves `value[x]` unset and records the loss by locus only — the raw clinical value is never named.

### Observation to OBX (`ObservationToOBX`)

`ObservationToOBX` is the reverse of `OBXToObservationR5`: it maps one FHIR R5 `Observation` back to an HL7 v2 OBX
segment. The `Observation.code` becomes OBX-3 (the observation identifier), the `value[x]` branch chooses OBX-2 (the
value type) and renders OBX-5, the interpretation maps back to OBX-8, and a numeric reference range to OBX-7. OBX-11 is
`F` (final), matching the status the forward path assigns a reported result.

| HL7 v2 OBX target | FHIR `Observation` source | Notes |
|-------------------|---------------------------|-------|
| OBX-3 (`CWE`) | `code` | required; an Observation with no code is not emitted and the loss is recorded |
| OBX-2 / OBX-5 | `value[x]` | `valueQuantity` → `NM` + OBX-6 units; `valueCodeableConcept` → `CWE` (`code^text^system`); `valueString` → `ST`; `valueDateTime` → `TS`; `valueTime` → `TM` |
| OBX-6 (`CWE`) | `valueQuantity` units | `Quantity.code`/`unit`/`system` → CWE-1/2/3 (the UCUM system URI maps back to the `UCUM` identifier) |
| OBX-8 | `interpretation` | each interpretation `Coding.code` carried verbatim (the FHIR and HL7 symbols match) |
| OBX-7 | `referenceRange` | a `low`/`high` `Quantity` pair renders `low-high`; a text range renders its text verbatim |

A `value[x]` branch with no OBX form leaves OBX-2/5 unset and records the loss by FHIR element path only — the raw
clinical value is never named in the report.

### Round-trip fidelity

The reverse converters are verified by round-trip fixtures that assert no silent loss of a measurement, unit, code, or
identity:

- **DICOM SR → `DiagnosticReport` + `Observation`s → DICOM SR.** The rebuilt SR re-parses as a conformant content
  tree and re-converts to the same report code, conclusion, NUM value/units, and CODE value. The one branch the reverse
  content-item builder cannot carry — a `valueTime`, which maps to a DICOM SR `TIME` content item of VR TM that the
  `dicom.DT` value cannot hold from the `convert` package — is recorded in `Report.Dropped`, never dropped silently.
- **HL7 v2 OBX → `Observation` → HL7 v2 OBX.** The rebuilt OBX carries the same value type, OBX-5 value, OBX-3 code,
  OBX-6 units, OBX-7 reference range, and OBX-8 flags, rendered through the `hl7v2` message serialiser.
- **Determinism.** Converting the same input twice produces byte-identical output: the reverse SR path mints
  byte-identical Study/Series/SOP Instance UIDs under a fixed `WithUIDRoot`, and the reverse OBX path renders an
  identical segment.

### ADT to Patient (`ADTToPatientR5`)

`ADTToPatientR5` converts an HL7 v2 admission/discharge/transfer message (`ADT^Axx`) to a FHIR R5 `Patient`. PD1
(additional demographics) carries no element the v1 mapping reads. `Patient` has no FHIR-required field, so the
conversion never fails closed on a sparse PID.

| FHIR `Patient` element | HL7 v2 source | Notes |
|------------------------|---------------|-------|
| `identifier[]` | PID-3 patient identifier list (`CX`) | each → logical `Identifier` (value + assigning-authority system), never a Reference URL |
| `name` | PID-5 Patient Name (`XPN`) | family → `family`; given and middle → `given[]`; prefix/suffix → the matching lists |
| `gender` | PID-8 Administrative Sex (Table 0001) | value-set-safe `ParseAdministrativeGender`; an out-of-table code maps to `unknown` and records a `Substitution` |
| `birthDate` | PID-7 Date/Time of Birth (`DTM`) | precision preserved; FHIR `birthDate` is a date, so a time-of-birth precision is dropped and recorded |
| `address` | PID-11 Patient Address (`XAD`) | street/other → `line[]`; city/state/postalCode/country → the matching fields |

`ParseAdministrativeGender` maps `M`→`male`, `F`→`female`, `O`/`A`/`N`→`other`, `U` and the empty value→`unknown`, and
any other code→`unknown` (recording a `Substitution`). The result is always a member of the required
`AdministrativeGender` value set, so the produced `Patient` validates by construction.

### ADT to Encounter (`ADTToEncounterR5`)

`ADTToEncounterR5` converts an `ADT^Axx` message to a FHIR R5 `Encounter`, reading the trigger event, PV1, and PID-3.

| FHIR `Encounter` element | HL7 v2 source | Notes |
|--------------------------|---------------|-------|
| `status` | MSH-9.2 / EVN-1 trigger event | `A01`→`in-progress`, `A03`→`completed`, `A11`→`cancelled`; an unmapped trigger → `unknown`; every mapping records a `Substitution` (the trigger event and the encounter status are not the same concept) |
| `identifier[]` | PV1-19 Visit Number (`CX`) | logical `Identifier`, never a Reference URL |
| `class` | PV1-2 Patient Class | `I`→`IMP`, `O`/`R`/`B`→`AMB`, `E`→`EMER`, `P`→`PRENC` under the `v3-ActCode` system; an unmapped class is carried verbatim and records a `Substitution` |
| `subject` | PID-3 (`CX`), or `WithSubjectR5` | logical `Reference.identifier`, never a fabricated URL |

`Encounter.status` is always a member of the required `EncounterStatus` value set, so the produced `Encounter` validates
by construction.

### R4 twins

Every converter has an `R4` twin that produces a `fhir/r4` resource instead of a `fhir/r5` one. The twins are not
type-aliases: the FHIR R4 (4.0.1) and R5 (5.0.0) datatype models differ in ways that change the shape of the output, so
each twin reads the same source and applies the same loss policy but writes the R4 element structure. The differences
the twins reconcile are:

- **No `CodeableReference` in R4.** R5 introduced the `CodeableReference` datatype (a concept or a reference in one
  element). R4 has no such type, so where an R5 converter writes a single `CodeableReference` element, the R4 twin
  writes the classic split pair:
  - `ImagingStudyR5.reason` (`CodeableReference[]`) → `ImagingStudyR4.reasonCode` (`CodeableConcept[]`); the coded and
    free-text DICOM reasons land on `reasonCode`, and `reasonReference` is reserved for a resolvable reason the source
    does not carry.
  - `ImagingStudyR5.procedure` (`CodeableReference[]`) → `ImagingStudyR4.procedureCode` (`CodeableConcept[]`).
  - `ServiceRequestR5.code` (`CodeableReference`) → `ServiceRequestR4.code` (`CodeableConcept`).
  - `ServiceRequestR5.reason` (`CodeableReference[]`) → `ServiceRequestR4.reasonCode` (`CodeableConcept[]`); the
    OBR-31 reason for study lands on `reasonCode`.
- **`ImagingStudy.modality` is a `Coding` in R4.** R5 widened study-level and series-level `modality` to
  `CodeableConcept`. The R4 twin writes a single `Coding` under the DICOM `DCM` system for both
  `ImagingStudyR4.modality` and `series.modality`. Likewise `series.bodySite` and `series.laterality` are `Coding` in
  R4 (R5 carries `series.bodySite` as a `CodeableReference` and `series.laterality` as a `CodeableConcept`).
- **`Encounter.class` is a single `Coding` in R4.** R5 changed `Encounter.class` to a `CodeableConcept[]`. The R4 twin
  writes one `Coding` under the `v3-ActCode` system. (R4 `Encounter.period` corresponds to the R5 `actualPeriod` rename;
  the v1 ADT mapping populates neither.)
- **`EncounterStatus` value-set rename.** The R4 completed-encounter state is named `finished`; R5 renamed it
  `completed`. The R4 trigger-event mapping is therefore `A01`→`in-progress`, `A03`→`finished`, `A11`→`cancelled`,
  with the same `Substitution` recording, so the value is always a member of the R4 `EncounterStatus` binding.

The release-agnostic machinery is shared, not duplicated: the DICOM and HL7 source reading, the date/time precision
handling, the deterministic `urn:uuid` result-link and minted-UID derivation, the `Report` (`Dropped`, `Defaulted`,
`Substituted`) recording, and the loss/strict-loss policy are the same code for both releases. Only the FHIR-typed
builders are twinned, because the FHIR `Identifier`, `Coding`, `CodeableConcept`, `Reference`, `Quantity`, `HumanName`,
and `Address` types live in the distinct `fhir/r4` and `fhir/r5` sub-packages even when their field shape is identical.
The shared identity rule holds in both releases: a DICOM UID becomes an `Identifier` (`urn:dicom:uid` / `urn:oid:`),
never a `Reference.reference` URL, and a person name becomes `Reference.display` only.

The `R4` reverse twins (`DiagnosticReportToSRR4`, `ObservationToContentItemR4`, `ObservationToOBXR4`) are the inverses
of `SRToDiagnosticReportR4`, `ContentItemToObservationR4`, and `OBXToObservationR4` respectively, with the same loss
model, the same minted-UID derivation, and the same one recorded reverse-direction loss (`Observation.valueTime`, no
DICOM SR `TIME` content-item form the `convert` package can carry).

### The Substituted channel

Alongside `Report.Dropped` (source data with no target home) and `Report.Defaulted` (a target the source did not
supply), the report carries a third channel, `Report.Substituted`, for an *approximate* mapping: a source value the
converter populated into the target, but only after coercing it into something the FHIR value set or datatype permits.
A `Substitution` names the FHIR `Concept` (the element path) and the `Approximation` (the value-set-safe value chosen),
never the raw patient value (PRD §9.1). The approximate mappings recorded are:

- an unknown PID-8 administrative-sex code rendered as `Patient.gender` = `unknown`;
- an ADT trigger event mapped to a `Encounter.status` (always approximate, since the trigger event and the encounter
  status are different concepts);
- an unrecognised PV1-2 patient class carried verbatim under the `v3-ActCode` system as `Encounter.class`.

A consumer uses the channel to distinguish a lossy-but-present mapping (recorded as a `Substitution`) from a value with
no target home (recorded as a `Dropped`) and from a target the source never supplied (recorded as a `Defaulted`).

## Loss policy

Not yet authored as a complete strict-versus-lossy contract across every conversion. The reverse converters
(`DiagnosticReportToSR`, `ObservationToContentItem`, `ObservationToOBX`) follow the same loss model as the forward
ones: a clean conversion records optional, unmappable data on the `*Report` (`Dropped`, `Defaulted`, `Substituted`) and
returns a `nil` error, while a genuinely unconstructible target fails closed. `DiagnosticReportToSR` fails closed
(`ErrMalformedSource`) when the report's code maps to no Concept Name Code Sequence, because the SR document root
requires one; the leaf converters return `(_, false)` and record the FHIR element by path when an Observation has no
code or no re-encodable `value[x]`. The reverse-direction losses recorded today are an `Observation.valueTime` (no DICOM
SR `TIME` content-item form the `convert` package can carry) and a `DiagnosticReport.subject` carried only as a
resolvable `Reference.reference` URL (no DICOM PatientID home). `WithStrictLoss` escalates any recorded `Dropped` entry
to a returned `*LossError` for consumers that cannot accept loss.

## Verification

Not yet authored as a complete conformance contract. The implementation today verifies each twin's output through its
release validator: the `convert` test suite validates every produced R4 resource through `r4.Validate` and every R5
resource through `r5.Validate`, the same release-scoped descriptor registries the merge-blocking FHIR validator gate
uses (the R4 path validates against 4.0.1). The forward twins have golden/round-trip tests that assert the load-bearing
R4/R5 differences and validate the output; the SR reverse twins have round-trip tests that re-parse the rebuilt SR and
re-validate the re-forwarded resources. The authored statement will additionally state the end-to-end walking-skeleton
interop coverage and the CI job that invokes the full set. Until then, no conversion conformance claim is made.

## References

- go-radx PRD §6.1 (conformance definition), §5.1 (workflow scope), §5.3 (FHIR releases).
- go-radx reference docs: `docs/reference/convert.md`.
- Per-standard conformance statements: [`./dicom.md`](./dicom.md), [`./hl7v2.md`](./hl7v2.md), [`./fhir.md`](./fhir.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
