# DICOMweb conformance statement

> **Implementation status: NOT YET SHIPPED.** This is a scaffold. The DICOMweb conformance statement — the versioned,
> validator-backed scope contract for the `dicomweb` package — is not yet authored. Until this banner is removed and
> the sections below are filled, **no DICOMweb behaviour is conformance-guaranteed**. The `dicomweb` package contains a
> client and an embeddable server, but their declared scope, role matrix, and validator gates are deferred to a later
> milestone. Do not cite this document as a conformance basis.

| Field | Value |
|-------|-------|
| Standard | DICOMweb (DICOM PS3.18, RESTful services) |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | unassigned (statement not yet authored) |
| Status | **NOT YET SHIPPED** — scaffold only |
| Scope authority | This document will become the single source of truth for DICOMweb scope (PRD §6.1) |

This document will be the DICOMweb Conformance Statement in the sense of PRD §6.1: it will declare exactly which
DICOMweb services, query parameters, content types, and authentication modes the `dicomweb` package supports, and the
client/server role for each — verified against reference origin servers. It is the HTTP-based counterpart to the DIMSE
conformance statement in [`./dicom.md`](./dicom.md); the two share the `dicom.TransferSyntax` and SOP Class UID
vocabulary but negotiate transport differently. Until it is authored, the DIMSE statement remains the only versioned
DICOM-network scope contract.

## Scope summary

Not yet authored. This section will enumerate the in-scope services and their roles:

- **WADO-RS** (web access to DICOM objects) — RESTful retrieval of studies, series, instances, frames, metadata, and
  bulk data.
- **STOW-RS** (store over the web) — RESTful storage via HTTP POST of `multipart/related` payloads.
- **QIDO-RS** (query based on ID for DICOM objects) — RESTful search over studies, series, and instances.

The supported query parameters, `Accept`/`Content-Type` negotiation, transfer-syntax selection, bulk-data
referencing, and pagination semantics will be declared here with their client and server roles.

## Out of scope

Not yet authored. The explicit deferral list — for example UPS-RS (worklist over the web), capabilities (`/`)
discovery, rendered retrieval (`/rendered`), and thumbnail endpoints — will be recorded here so the boundary is
explicit and a consumer is never surprised.

## Verification

Not yet authored. This section will state how conformance to the declared subset is gated: round-trips against
reference DICOMweb origin servers (Orthanc, dcm4chee-arc) under a build-tagged interop suite, and the CI job that
invokes it. Until then, no DICOMweb verification claim is made.

## References

- DICOM PS3.18 (Web Services).
- go-radx PRD §6.1 (conformance definition), §5.1 (workflow scope).
- go-radx reference docs: `docs/reference/dicomweb.md`.
- DICOM conformance statement: [`./dicom.md`](./dicom.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
