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
  bulk data. **Implemented; client interop-gated against Orthanc** — see [WADO-RS](#wado-rs-implemented) below.
- **STOW-RS** (store over the web) — RESTful storage via HTTP POST of `multipart/related` payloads.
- **QIDO-RS** (query based on ID for DICOM objects) — RESTful search over studies, series, and instances.
  **Implemented; client interop-gated against Orthanc** — see [QIDO-RS](#qido-rs-implemented) below.

The supported query parameters, `Accept`/`Content-Type` negotiation, transfer-syntax selection, bulk-data
referencing, and pagination semantics will be declared here with their client and server roles.

## WADO-RS (implemented)

The `dicomweb` package implements WADO-RS retrieval (PS3.18 §10.4) on both the embeddable server and the client. The
overall statement above remains a scaffold until STOW-RS is likewise authored; this section declares the WADO surface
that is shipped today.

### Retrieve resources

The server answers retrieval at the study, series, and instance levels and at the metadata, frames, and bulkdata
sub-resources. The client exposes one method per resource.

| Resource | Path | Response framing | Client method |
|----------|------|------------------|---------------|
| Study | `/studies/{study}` | `multipart/related` of `application/dicom` parts | `RetrieveStudy` (streaming iterator) |
| Series | `/studies/{study}/series/{series}` | `multipart/related` of `application/dicom` parts | `RetrieveSeries` (streaming iterator) |
| Instance | `/studies/{study}/series/{series}/instances/{sop}` | `multipart/related` of one `application/dicom` part | `RetrieveInstance` |
| Metadata | `.../metadata` (study, series, or instance level) | `application/dicom+json` array | `RetrieveMetadata` |
| Frames | `.../instances/{sop}/frames/{frameList}` | `multipart/related` of `application/octet-stream` parts | `RetrieveFrames` (1-based) |
| Bulk data | `.../instances/{sop}/bulkdata` (with optional reference suffix) | `multipart/related` of `application/octet-stream` parts | `RetrieveBulkData`, `ResolveBulkDataURI` |

A path that is not one of these resources is answered with `501 Not Implemented`, never a silent empty body. A
malformed UID in the path is rejected with `400 Bad Request`. A study or series with no matching instances, and an
instance with no requested frames or bulk data, answer `404 Not Found` rather than an empty `200` a caller would read
as a complete-but-empty result.

### Media types

| Resource | Accept | Emitted parts |
|----------|--------|---------------|
| Study / series / instance | `multipart/related; type="application/dicom"` (or a wildcard) | `application/dicom` (Part 10) |
| Metadata | `application/dicom+json` (or a wildcard) | one DICOM-JSON object per instance |
| Frames / bulk data | `multipart/related; type="application/octet-stream"` (or a wildcard) | `application/octet-stream` |

`application/dicom+xml` metadata is deferred; an `Accept` naming only XML is answered `406 Not Acceptable`. The media
type is gated before the backend is consulted, so a wholly unservable `Accept` fails fast without a lookup.

### Transfer-syntax policy

Instance, study, and series retrieval apply a documented transfer-syntax policy per instance (PS3.18 §8.7.3.3):

- **No `transfer-syntax` parameter, or `transfer-syntax=*`** — the instance is served in its stored syntax
  (passthrough). The wildcard explicitly means "any syntax you hold", so the origin never transcodes for it.
- **`transfer-syntax` names the stored syntax** — passthrough.
- **`transfer-syntax` names a syntax the server can transcode to** — transcode to that syntax. The shipped server
  registers no pixel-data transcoders, so this branch is reachable only by a deployment that supplies them.
- **`transfer-syntax` names no servable syntax** — `406 Not Acceptable`.

This replaces the prior unconditional re-encode to Explicit VR Little Endian: an instance stored in a compressed syntax
that the client does not accept is now answered with an honest `406`, never silently re-encoded. A backend reports its
stored syntax by implementing `StoredInstanceRetriever`; a backend that implements only the base `RetrieveBackend` is
treated as storing the default uncompressed syntax (Explicit VR Little Endian).

### Bulk-data referencing

A metadata response emits each binary value as a `BulkDataURI` rooted at that instance's own bulkdata sub-resource, so
a client resolves it through the same origin. The client leaves a `BulkDataURI` unresolved on decode; `BulkDataURIs`
enumerates the pending references in a returned dataset and `ResolveBulkDataURI` fetches a reference's octets. A
relative reference is joined to the client's `WithClientBulkDataBaseURL` (or the origin base URL when none is set); an
absolute reference is fetched as given. The current bulkdata server returns every bulk-data value of the instance for
any attribute reference under `.../bulkdata`; per-attribute selection is a recorded follow-up.

### Errors

A retrieval fault is a typed problem document carrying the mapped HTTP status and a PHI-free structural detail. A
resource UID is never echoed in an error or log line: the request path is redacted (every UID replaced by `{uid}`) and a
remote error body is never copied into the returned error. A frame list or UID that does not parse is rejected without
echoing the offending text.

### Roles

| Service | Server | Client |
|---------|--------|--------|
| WADO-RS study / series retrieve | Implemented (optional `StudyRetriever` / `SeriesRetriever`) | Implemented (streaming) |
| WADO-RS instance retrieve | Implemented (`RetrieveBackend`; `StoredInstanceRetriever` for the TS policy) | Implemented |
| WADO-RS metadata retrieve | Implemented (optional `MetadataRetriever`) | Implemented |
| WADO-RS frames retrieve | Implemented (optional `FrameRetriever`) | Implemented |
| WADO-RS bulkdata retrieve | Implemented (optional `BulkDataRetriever`) | Implemented |
| Pixel-data transcoding | Deferred (policy answers `406` for an unservable syntax) | Deferred |

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

The full statement's gating story (STOW-RS) is still being authored. The shipped WADO-RS and QIDO-RS surfaces are
interop-gated against a real Orthanc origin in `dicomweb/integration` (behind the `interop` build tag), each STOWing a
vendored instance before retrieving or searching it back:

- `TestInteropStowThenWadoOrthanc` — STOW then WADO-RS instance retrieve round-trip.
- `TestInteropWADOStudySeriesOrthanc` — WADO-RS study and series retrieval through the streaming iterators.
- `TestInteropWADOMetadataBulkDataOrthanc` — WADO-RS metadata retrieve, then resolving an emitted `BulkDataURI` back
  to its octets through the bulkdata sub-resource.
- `TestInteropWADOFramesOrthanc` — WADO-RS frame retrieval as `application/octet-stream` parts.
- `TestInteropQIDOOrthanc` — QIDO-RS search at the study, series, and instance levels.

QIDO parameter parsing is fuzzed (`FuzzParseQueryRequest`). A dcm4chee-arc DICOMweb leg is not yet wired (the WADO/STOW
interop is Orthanc-only today) and is recorded as a follow-up. Until STOW-RS is authored, no STOW-RS verification claim
is made beyond the existing STOW-then-WADO Orthanc round-trip.

## References

- DICOM PS3.18 (Web Services).
- go-radx PRD §6.1 (conformance definition), §5.1 (workflow scope).
- go-radx reference docs: `docs/reference/dicomweb.md`.
- DICOM conformance statement: [`./dicom.md`](./dicom.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
