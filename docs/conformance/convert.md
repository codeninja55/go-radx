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
