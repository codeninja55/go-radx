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
  **Implemented; client interop-gated against Orthanc** — see [STOW-RS](#stow-rs-implemented) below.
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

## STOW-RS (implemented)

The `dicomweb` package implements STOW-RS storage (PS3.18 §10.5) on both the embeddable server and the client. The
overall statement above remains a scaffold until the full surface is authored; this section declares the STOW surface
that is shipped today.

### Store targets and variants

The server accepts a POST to `/studies` (unconstrained) or `/studies/{study}` (constrained to one study). An instance
whose `StudyInstanceUID` does not match a `/studies/{study}` target is rejected into the Failed SOP Sequence rather
than stored under an unrelated hierarchy. The `multipart/related` body's `type` parameter selects the variant:

| Variant | Body `type` | Parts |
|---------|-------------|-------|
| Whole object | `application/dicom` | one Part 10 object per `application/dicom` part |
| Metadata + bulk data | `application/dicom+json` | one DICOM-JSON metadata part plus separate bulk-data parts |

In the metadata-plus-bulkdata variant the metadata part is a DICOM-JSON array of instances; each binary value is a
`BulkDataURI` referencing a separate bulk-data part by its `Content-Location` or `Content-ID`. The server collects the
bulk-data parts, reassembles each instance by resolving its references, and stores the result. A reference that names
no part fails that instance closed (the instance is not stored), never stored with a partial value.

### Store response

The store response is an `application/dicom+json` document (PS3.18 §10.5.3) carrying:

- a study-level **Retrieve URL** (0008,1190) once any instance was accepted under a known study;
- a **Referenced SOP Sequence** (0008,1199) of accepted instances, each with its SOP identity, a per-instance Retrieve
  URL (0008,1190), and a **Warning Reason** (0008,1196) when the store warned (for example a coerced data element);
- a **Failed SOP Sequence** (0008,1198) of rejected instances, each with its **Failure Reason** (0008,1197);
- a top-level **Failure Reason** (0008,1197) for a fault that belongs to no single instance (the "Other failures"
  path), for example a metadata instance that carries no SOP identity.

The Retrieve URLs are rooted at the configured `WithStoreRetrieveURLBase`, or derived from the request's scheme and
host when none is set (set the option behind a reverse proxy that rewrites the public origin). A per-instance Retrieve
URL the response carries resolves to the stored instance through the same origin. The HTTP status follows PS3.18
§10.5.3: `200 OK` when every instance was accepted, `202 Accepted` for a partial store, and `409 Conflict` when none
was accepted. The client is fail-closed (PRD §9.2): a `202`/`409`, or a parsed response with any failure, returns a
`*StoreError` alongside the parsed response so neither the partial success nor the failure is silently dropped.

A backend reports a per-instance Warning Reason through the optional `WarnableStoreBackend`; a backend that implements
only the base `StoreBackend` reports a clean accept.

### Failure and warning reasons

The Failure Reason (0008,1197) and Warning Reason (0008,1196) codes are rendered by their registered name for
diagnostics, never by any patient value. The reasons the package names:

| Code | Kind | Meaning |
|------|------|---------|
| `0x0110` | Failure | Processing failure (the default for an untyped backend error or an unreferenceable instance) |
| `0x0122` | Failure | Referenced SOP Class not supported |
| `0x0119` | Failure | Class-instance conflict |
| `0x0242` | Failure | SOP Instance access denied |
| `0xA700` | Failure | Out of resources |
| `0xA730` | Failure | Intended recipient not supported |
| `0xA800` / `0xA900` | Failure | Data set does not match SOP Class |
| `0xC000` | Failure | Cannot understand |
| `0xC120` | Failure | Referenced SOP Instance is not in the requested Study (the study-mismatch rejection) |
| `0xC122` | Failure | Transfer syntax not supported |

A code outside this set is rendered with its hex value so an unregistered reason is never silently dropped. A backend
returns a specific Failure Reason for a rejected instance through `FailureReasonError`; any other error defaults to
processing failure (`0x0110`).

### Errors and PHI

A store fault that aborts the whole request (a malformed body, a part that is not a valid instance, a part with no SOP
identity, a malformed target study) is a typed problem document carrying the mapped HTTP status and a PHI-free
structural detail. A remote error body is never copied into a client error, and a resource UID is never echoed: the
request path is redacted. The store-response document carries only SOP identity, registered reason codes, and
origin-rooted Retrieve URLs, never a patient value (PRD §9.1).

### Roles

| Service | Server | Client |
|---------|--------|--------|
| STOW-RS store (whole object) | Implemented (`StoreBackend`; `WarnableStoreBackend` for a per-instance warning) | Implemented (`Store`) |
| STOW-RS store (metadata + bulk data) | Implemented | Deferred — the client posts whole objects today |

## Client authentication

Client authentication is a pluggable transport concern: each scheme is an `http.RoundTripper` layered over the
client's base transport, so a new scheme adds no branch to the request path and the cloud adapters compose through the
same seam without modifying the core client. Every credential scheme is **origin-scoped** — it injects its credential
only when the request targets the client's configured origin (matching scheme, host, and port), so a server-supplied
absolute `BulkDataURI` on a foreign host can never harvest the credential, whichever scheme is configured (PRD §9.8). A
credential is never logged or placed in an error message.

| Mode | Option | Header / mechanism | Refreshes | Origin-scoped |
|------|--------|--------------------|-----------|---------------|
| Static bearer | `WithBearerToken` | `Authorization: Bearer` | No | Yes |
| HTTP Basic | `WithBasicAuth` | `Authorization: Basic` | No | Yes |
| OAuth2 token source | `WithTokenSource` | `Authorization: Bearer` from an `oauth2.TokenSource` | Yes (on expiry) | Yes |
| Mutual TLS | `WithClientCertificate` | client certificate in the TLS handshake | n/a | n/a (TLS to the configured origin) |
| Custom | `WithRoundTripper` | caller-supplied `http.RoundTripper` | per the transport | per the transport |

`WithTokenSource` covers static, refresh-token, and client-credentials flows in one abstraction; the token source
caches a token until it expires and fetches a fresh one on demand, so a long-lived client re-authenticates mid-session
without caller involvement. `WithRoundTripper` is the escape hatch the cloud auth adapters (Google ADC, AWS SigV4)
build on; a custom transport enforces its own origin scoping and may inject a header or sign the request per-request.

### Cloud-provider adapters

Two cloud archives are reachable over DICOMweb — Google Cloud Healthcare and AWS HealthImaging — and each composes
through the core seam above without modifying the core client. The provider SDKs live only in the
`dicomweb/auth/gcp` and `dicomweb/auth/aws` subpackages, so a caller who never touches a cloud adapter never pulls the
AWS SDK or the Google SDK into their import graph; this isolation is enforced by a guard test
(`TestCoreImportGraphExcludesCloudSDKs`).

| Provider mode | Adapter | Wires via | Mechanism | Refreshes |
|---------------|---------|-----------|-----------|-----------|
| Google Cloud Healthcare (ADC) | `dicomweb/auth/gcp.TokenSource` | `WithTokenSource` | `Authorization: Bearer` from Application Default Credentials, scoped to `cloud-platform` | Yes (on expiry) |
| AWS HealthImaging (SigV4) | `dicomweb/auth/aws.SigV4RoundTripper` | `WithRoundTripper` | per-request `Authorization: AWS4-HMAC-SHA256` signature for the `medical-imaging` service | Per request |
| AWS HealthImaging (OIDC) | none (core path) | `WithTokenSource` | `Authorization: Bearer` from any `oauth2.TokenSource` | Yes (on expiry) |

The Google adapter resolves a token from the standard ADC chain (the `GOOGLE_APPLICATION_CREDENTIALS` service account,
gcloud user credentials, or the GCE/GKE metadata server) scoped to `https://www.googleapis.com/auth/cloud-platform`,
the scope the Cloud Healthcare DICOMweb endpoint requires. The AWS SigV4 adapter signs each request independently with
a current timestamp and a payload hash over the request body, drawing credentials from the supplied `aws.Config` on
every request so a rotating or assumed-role credential is always current; it is not a static header. AWS HealthImaging's
OIDC access mode needs no adapter because it authenticates with a standard OAuth2 bearer, which the core
`WithTokenSource` path already covers. See `../user-guide/dicomweb/cloud-auth.md` for worked wiring examples.

## Out of scope

The full deferral list is still being authored. The deferrals recorded today:

- **`application/dicom+xml` metadata** — the XML representation of DICOM-JSON (PS3.18 Annex A). Metadata and store
  responses are `application/dicom+json` only; an `Accept` naming only XML is answered `406 Not Acceptable`, and a
  STOW-RS `type="application/dicom+xml"` body is not parsed. Deferred because DICOM-JSON is the modern, sufficient
  representation and supporting both doubles the encode/decode surface; XML support would be added behind the same
  negotiation seam without changing the JSON path.
- **WADO-URI** (the legacy single-object query-parameter retrieval, PS3.18 §9) — superseded by WADO-RS for the
  workflows in scope. Retrieval is WADO-RS (`multipart/related`) only; the `?requestType=WADO` query form is not
  served. Deferred as a legacy surface with no new capability over WADO-RS.

Further deferrals — for example UPS-RS (worklist over the web), capabilities (`/`) discovery, rendered retrieval
(`/rendered`), and thumbnail endpoints — will be recorded here so the boundary is explicit and a consumer is never
surprised.

## Verification

The shipped WADO-RS, QIDO-RS, and STOW-RS surfaces are interop-gated against a real Orthanc origin in
`dicomweb/integration` (behind the `interop` build tag), each STOWing a vendored instance before retrieving or
searching it back:

- `TestInteropStowThenWadoOrthanc` — STOW then WADO-RS instance retrieve round-trip.
- `TestInteropWADOStudySeriesOrthanc` — WADO-RS study and series retrieval through the streaming iterators.
- `TestInteropWADOMetadataBulkDataOrthanc` — WADO-RS metadata retrieve, then resolving an emitted `BulkDataURI` back
  to its octets through the bulkdata sub-resource.
- `TestInteropWADOFramesOrthanc` — WADO-RS frame retrieval as `application/octet-stream` parts.
- `TestInteropQIDOOrthanc` — QIDO-RS search at the study, series, and instance levels.

The STOW-RS store-response completeness, the metadata-plus-bulkdata variant, the per-instance Retrieve URL and Warning
Reason, the Other-failures path, the q=0 negotiation refusal, and every client authentication mode (bearer, basic,
OAuth2 token source, mutual TLS, custom RoundTripper) are unit-tested in `dicomweb` (`store_test.go`, `auth_test.go`).
QIDO parameter parsing is fuzzed (`FuzzParseQueryRequest`). A dcm4chee-arc DICOMweb leg is not yet wired (the
WADO/STOW interop is Orthanc-only today) and is recorded as a follow-up.

The cloud-provider adapters are unit-tested without any live cloud call. The Google ADC token source is exercised
against a mock OAuth2 token endpoint that returns a bearer (`dicomweb/auth/gcp`, `gcp_test.go`). The AWS SigV4
RoundTripper is verified by re-deriving the signature from the documented canonical-request algorithm and matching it
against the SDK-produced `AWS4-HMAC-SHA256` header, for both empty-body and body-bearing requests
(`dicomweb/auth/aws`, `aws_test.go`). The dependency-isolation guarantee is checked by
`TestCoreImportGraphExcludesCloudSDKs`, which shells out to `go list -deps` and fails if either cloud SDK enters the
core `dicomweb` import graph.

## References

- DICOM PS3.18 (Web Services).
- go-radx PRD §6.1 (conformance definition), §5.1 (workflow scope).
- go-radx reference docs: `docs/reference/dicomweb.md`.
- DICOM conformance statement: [`./dicom.md`](./dicom.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
