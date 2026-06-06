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
  **Implemented; client interop-gated against Orthanc** — see [QIDO-RS](#qido-rs-implemented) below.

The supported query parameters, `Accept`/`Content-Type` negotiation, transfer-syntax selection, bulk-data
referencing, and pagination semantics will be declared here with their client and server roles.

## QIDO-RS (implemented)

The `dicomweb` package implements QIDO-RS search (PS3.18 §10.6) on both the embeddable server and the client. The
overall statement above remains a scaffold until WADO-RS and STOW-RS are likewise authored; this section declares the
QIDO surface that is shipped today.

### Search resources (server)

The server answers a search at three levels, scoped by the request path:

- Studies: `/studies`
- Series: `/series` and `/studies/{study}/series`
- Instances: `/instances`, `/studies/{study}/instances`, and `/studies/{study}/series/{series}/instances`

A path that is not one of these search resources is answered with `501 Not Implemented`, never a silent empty result. A
malformed parent UID in the path is rejected with `400 Bad Request`.

### Attribute matching

Matching is dispatched by the attribute VR (PS3.4 C.2.2.2):

| Match type | VRs | Semantics |
|------------|-----|-----------|
| Single value | string VRs | Exact, case-sensitive |
| Wildcard | string VRs | `*` (zero or more characters) and `?` (exactly one) |
| UID list | UI | Backslash-separated, exact, case-sensitive |
| Range | DA, TM, DT | `lo-hi`, `lo-`, `-hi`; a malformed range fails closed (matches nothing) |
| Universal | any | An empty value or a bare `*` matches every candidate |
| Fuzzy (person name) | PN | `fuzzymatching=true`: case-insensitive, component-group-insensitive substring (not phonetic) |

A present, non-universal matching key against an absent attribute never matches. A multi-valued attribute matches when
any one of its values matches.

### includefield and return attributes

`includefield` accepts attribute keywords or `GGGGEEEE` tag strings (comma-separated or repeated), plus
`includefield=all`. When `includefield` names no extra fields, the server projects the per-level default return
attributes (PS3.18 Tables 10.6.1-5/-5a/-5b): study, series, and instance identity, related counts, and the common
patient and study attributes. An unresolvable attribute reference is rejected, never silently dropped.

### Paging and truncation

`limit` and `offset` page the result set. The server caps results at 5,000 by default (configurable with
`WithMaxQueryResults`); a response truncated by the cap carries the `Warning: 299` header (PS3.18 §10.6.1.4) so a
truncated page is never read as the complete result.

### Media type and errors

Results are returned as `application/dicom+json`. A request fault is a typed `QueryError` carrying the mapped HTTP
status and a PHI-free structural detail: a rejected attribute is named by keyword only, never by its (potentially
patient-identifying) value, and a query value is never echoed in an error or log line. A backend error fails the query
closed (`500`) rather than reporting an empty result a caller would read as "no matches".

### Client

The client exposes `SearchStudies`, `SearchSeries`, and `SearchInstances`. The query string is stripped from any URL
recorded in an error, since a QIDO query string can carry patient identifiers.

### Roles

| Service | Server | Client |
|---------|--------|--------|
| QIDO-RS search (study / series / instance) | Implemented (pluggable `QueryBackend`) | Implemented |

## Out of scope

Not yet authored. The explicit deferral list — for example UPS-RS (worklist over the web), capabilities (`/`)
discovery, rendered retrieval (`/rendered`), and thumbnail endpoints — will be recorded here so the boundary is
explicit and a consumer is never surprised.

## Verification

The full statement's gating story (WADO-RS, STOW-RS) is still being authored. For the shipped QIDO-RS surface, the
client is gated by `TestInteropQIDOOrthanc` in `dicomweb/integration` (behind the `interop` build tag): it STOWs a
vendored instance to a real Orthanc origin and searches it back at the study, series, and instance levels, proving the
client's query encoding and `application/dicom+json` parsing against a compliant origin. QIDO parameter parsing is
fuzzed (`FuzzParseQueryRequest`). A dcm4chee-arc DICOMweb leg is not yet wired (the STOW/WADO interop is likewise
Orthanc-only today) and is recorded as a follow-up. Until the remaining services are authored, no WADO-RS/STOW-RS
verification claim is made beyond the existing STOW-then-WADO Orthanc round-trip.

## References

- DICOM PS3.18 (Web Services).
- go-radx PRD §6.1 (conformance definition), §5.1 (workflow scope).
- go-radx reference docs: `docs/reference/dicomweb.md`.
- DICOM conformance statement: [`./dicom.md`](./dicom.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
