# DICOMweb client and server

The `dicomweb` package implements the RESTful DICOM services defined in DICOM PS3.18: WADO-RS (web access to
DICOM objects), STOW-RS (store over the web), and QIDO-RS (query based on ID for DICOM objects). It ships both a
**client** for talking to a remote DICOMweb origin server (Orthanc, dcm4chee-arc, a cloud archive) and an embeddable
**server** that exposes the same services over your own storage and query backends.

DICOMweb is the HTTP-based counterpart to the DIMSE services in the `dimse` package. WADO-RS corresponds to
C-GET/C-MOVE, STOW-RS to C-STORE, and QIDO-RS to C-FIND. Where DIMSE negotiates presentation contexts over a stateful
TCP association, DICOMweb is stateless HTTP with content negotiation, which makes it firewall-friendly and easy to
secure with TLS and bearer tokens.

This package shares the DICOM data model with the `dicom` package: a retrieved or stored object is a
`*dicom.DataSet`, identifiers are `dicom.UID`, and encodings are `dicom.TransferSyntax`. It does not introduce a
parallel object model. The web resource hierarchy (study, series, instance) maps directly onto `dimse.QueryLevel`.

## Scope

There is no Python parity floor for DICOMweb. `pydicom`/`pynetdicom` provide only the DICOM-JSON codec and a
bulk-data-URI retrieval hook, not a DICOMweb client or server. The v1 surface is therefore designed against the
standard (PS3.18) and the dc4che project's conventions directly, and verified by live interop against Orthanc and
dcm4chee-arc (PRD §11.1).

In scope for v1:

- **WADO-RS** retrieval: instances and metadata at study, series, and instance level; rendered resources are out of
  v1 scope (see Conformance scope and limits); frames; and bulk data.
- **STOW-RS** storage: `POST` of one or more instances as `multipart/related` with `application/dicom` parts, returning
  an `application/dicom+json` store-response document.
- **QIDO-RS** search: query at study, series, and instance level with `application/dicom+json` results, including
  `includefield`, `limit`, `offset`, and `fuzzymatching` query parameters.
- The DICOM-JSON model (PS3.18 Annex F) and `multipart/related` framing, with explicit content negotiation.
- Embeddable server handlers backed by pluggable storage and query interfaces, plus a thin reference daemon
  (filesystem object store) that runs out of the box and binds to loopback by default (PRD §9.1).

Explicitly **deferred** (architected-for, not implemented in v1): the legacy WADO-URI interface, UPS-RS, rendered
retrieval (`/rendered`, `/thumbnail`), capabilities (`/`), and bulk-data `STOW`. These return a typed
"unsupported" error rather than a silent no-op, per the fail-closed rule (PRD §9.2).

## Resource model

DICOMweb addresses objects through a study/series/instance URL hierarchy. go-radx models the path with a single
value type so a caller never assembles URL fragments by hand and so the mapping to `dimse.QueryLevel` is explicit.

```go
// ResourcePath identifies a DICOMweb resource. The deepest non-empty UID sets the level.
// Constructed through NewStudy/NewSeries/NewInstance so an invalid path is unrepresentable.
type ResourcePath struct {
    Study    dicom.UID // StudyInstanceUID;  required for SERIES and INSTANCE paths
    Series   dicom.UID // SeriesInstanceUID; required for INSTANCE paths
    Instance dicom.UID // SOPInstanceUID
}

func NewStudy(study dicom.UID) ResourcePath
func NewSeries(study, series dicom.UID) ResourcePath
func NewInstance(study, series, instance dicom.UID) ResourcePath

// Level reports the query level implied by the deepest UID set on the path.
func (p ResourcePath) Level() dimse.QueryLevel

// Path renders the URL path segment, e.g. "/studies/{study}/series/{series}".
// Each UID is validated as a DICOM UID before it is interpolated; an invalid UID is rejected, never escaped blindly.
func (p ResourcePath) Path() (string, error)
```

The mapping between the URL hierarchy and `dimse.QueryLevel` is fixed:

| Web resource | URL template | `dimse.QueryLevel` |
|--------------|--------------|--------------------|
| Study | `/studies/{StudyInstanceUID}` | `QueryLevelStudy` |
| Series | `/studies/{StudyInstanceUID}/series/{SeriesInstanceUID}` | `QueryLevelSeries` |
| Instance | `/studies/{study}/series/{series}/instances/{SOPInstanceUID}` | `QueryLevelImage` |

`QueryLevelPatient` has no DICOMweb URL form: PS3.18 search begins at the study level. A patient-level query is
expressed as a study-level QIDO-RS search filtered by `PatientID`, so a caller migrating from DIMSE C-FIND
patient-level queries maps `PatientID` into a `SearchQuery` match key rather than a path segment.

Frames and bulk data hang off an instance path:

```go
// Frames addresses one or more pixel frames of an instance (1-based, per PS3.18 §10.4.3).
// A frame is a pixel-data frame, not a DUL/PDV transport fragment.
func (p ResourcePath) Frames(frames ...int) (string, error) // "/instances/{uid}/frames/1,4,5"

// BulkDataURI is an opaque, server-issued URL for a large binary value, returned in metadata
// as the "BulkDataURI" JSON key (PS3.18 Annex F). It is fetched with Client.RetrieveBulkData.
type BulkDataURI string
```

## Client API

The client is constructed against a base URL with functional options; there is no global configuration (PRD §8.1,
§8.2). The zero-value options give a TLS-verifying client bound to standard timeouts and the hostile-input caps below.

```go
// NewClient returns a DICOMweb client for the origin server at baseURL (the path that precedes /studies).
func NewClient(baseURL string, opts ...ClientOption) (*Client, error)

type ClientOption func(*clientConfig)

func WithHTTPClient(h *http.Client) ClientOption       // bring your own transport; default verifies TLS 1.2+ peers
func WithBearerToken(token string) ClientOption        // Authorization: Bearer <token>; never logged (PRD §9.8)
func WithTransferSyntaxes(ts ...dicom.TransferSyntax) ClientOption // Accept transfer-syntax preference for WADO-RS
func WithMaxResponseBytes(n int64) ClientOption        // cap on any single response body (PRD §9.3)
```

### WADO-RS retrieval

Retrieval returns datasets as values. Multi-object retrievals (study, series) return a Go 1.23+ iterator so the caller
streams instances without buffering the whole study, matching the `dimse` iterator convention (PRD §8.1).

```go
// RetrieveInstance fetches a single instance as a parsed DataSet (multipart/related; application/dicom part).
func (c *Client) RetrieveInstance(ctx context.Context, p ResourcePath) (*dicom.DataSet, error)

// RetrieveStudy / RetrieveSeries stream the constituent instances. The iterator yields a typed error per item
// and stops on the first transport error; ctx cancellation ends iteration promptly.
func (c *Client) RetrieveStudy(ctx context.Context, p ResourcePath) iter.Seq2[*dicom.DataSet, error]
func (c *Client) RetrieveSeries(ctx context.Context, p ResourcePath) iter.Seq2[*dicom.DataSet, error]

// RetrieveMetadata fetches the application/dicom+json metadata for a study, series, or instance path
// (no bulk data inlined; large values are referenced by BulkDataURI).
func (c *Client) RetrieveMetadata(ctx context.Context, p ResourcePath) ([]*dicom.DataSet, error)

// RetrieveFrames fetches specific pixel frames as raw octet-stream parts, one []byte per requested frame.
func (c *Client) RetrieveFrames(ctx context.Context, p ResourcePath, frames ...int) ([][]byte, error)

// RetrieveBulkData fetches a single bulk-data value previously referenced by a BulkDataURI in metadata.
func (c *Client) RetrieveBulkData(ctx context.Context, uri BulkDataURI) ([]byte, error)
```

### STOW-RS storage

Storage takes datasets and returns the parsed store response so the caller can distinguish per-instance success from
failure. The response is honest about partial failure (PRD §9.2): a transfer in which some instances were accepted and
some failed returns a non-nil `*StoreResponse` together with a `*StoreError` so neither success nor failure is silently
dropped.

```go
// Store POSTs one or more instances as multipart/related (application/dicom parts) to /studies
// or /studies/{StudyInstanceUID}. Returns the parsed STOW-RS response document.
func (c *Client) Store(ctx context.Context, instances ...*dicom.DataSet) (*StoreResponse, error)

// StoreResponse models the application/dicom+json store response (PS3.18 §10.5).
type StoreResponse struct {
    RetrieveURL string                    // (0008,1190) of the response dataset, if present
    Referenced  []StoredInstance          // (0008,1199) — instances the server accepted
    Failed      []dicom.FailedSOPInstance // (0008,1198) — instances the server rejected, with a reason code
}

func (r *StoreResponse) IsComplete() bool // true only when Failed is empty

// StoredInstance is an accepted STOW-RS instance: the canonical dicom.ReferencedSOPInstance pair plus the
// HTTP-specific per-instance RetrieveURL (0008,1190) the origin assigns. Embedding the shared type keeps the SOP
// UID vocabulary identical to dimse and dicom without re-declaring it.
type StoredInstance struct {
    dicom.ReferencedSOPInstance
    RetrieveURL string
}

// ReferencedSOPInstance and FailedSOPInstance are the canonical referenced-instance shapes owned by the dicom
// package (never bare "Reference"; that noun is FHIR's — see glossary). dicomweb reuses them so a Storage-Commitment
// result (dimse) and a STOW-RS response (dicomweb) speak the same vocabulary; dimse.md uses the identical pair:
//
//	type ReferencedSOPInstance struct {
//	    SOPClassUID    dicom.SOPClassUID
//	    SOPInstanceUID dicom.SOPInstanceUID
//	}
//	type FailedSOPInstance struct {
//	    ReferencedSOPInstance
//	    FailureReason uint16 // the (0008,1197) failure-reason code, rendered by name in diagnostics
//	}
```

### QIDO-RS search

Search builds a typed query and returns matched datasets. Match keys are DICOM tags, so the caller works in the same
vocabulary as DIMSE C-FIND rather than guessing query-parameter spelling.

```go
// SearchQuery is a QIDO-RS query. Level selects /studies, /series, or /instances.
type SearchQuery struct {
    Level        dimse.QueryLevel       // QueryLevelStudy, QueryLevelSeries, or QueryLevelImage
    Scope        ResourcePath           // optional parent scope (e.g. series search within one study)
    Match        map[dicom.Tag]string   // tag -> match value; rendered as {keyword}={value} query params
    IncludeField []dicom.Tag            // includefield= attributes to return beyond the defaults
    FuzzyMatching bool                  // fuzzymatching=true
    Limit        int                    // limit=; 0 means server default
    Offset       int                    // offset=
}

// Search executes the query and returns the matched datasets (one DataSet per QIDO-RS result object).
func (c *Client) Search(ctx context.Context, q SearchQuery) ([]*dicom.DataSet, error)
```

## DICOM JSON and multipart/related

go-radx implements the DICOM-JSON model from PS3.18 Annex F as a codec over `dicom.DataSet`, distinct from FHIR JSON
(the glossary rule: never cross-feed serializers). A dataset serializes to a JSON object keyed by eight-hex-digit
tag strings, each value an object carrying `vr` and exactly one of `Value`, `BulkDataURI`, or `InlineBinary`:

```go
// MarshalJSON encodes a DataSet as DICOM JSON (PS3.18 Annex F). A PN element renders as
// {"00100010":{"vr":"PN","Value":[{"Alphabetic":"Doe^Jane"}]}} using the
// Alphabetic/Ideographic/Phonetic component-group keys; SQ nests DataSet objects;
// values too large to inline are emitted as BulkDataURI when a base URL is configured, else InlineBinary (base64).
func MarshalJSON(ds *dicom.DataSet, opts ...JSONOption) ([]byte, error)

// UnmarshalJSON decodes DICOM JSON into a DataSet. BulkDataURI values are left as references unless a
// BulkDataResolver is supplied; InlineBinary is base64-decoded with the declared VR.
func UnmarshalJSON(data []byte, opts ...JSONOption) (*dicom.DataSet, error)

type JSONOption func(*jsonConfig)

// BulkDataResolver is invoked for each BulkDataURI encountered during decode; nil leaves the value empty.
func WithBulkDataResolver(fn func(ctx context.Context, uri BulkDataURI) ([]byte, error)) JSONOption
func WithBulkDataThreshold(bytes int) JSONOption // values at/above this size emit BulkDataURI on encode
```

`multipart/related` framing (PS3.18 §6.x) carries binary payloads: STOW-RS request bodies, WADO-RS instance and frame
responses. The package provides a reader and writer that enforce the part-count and part-size caps so a hostile peer
cannot exhaust memory by declaring thousands of parts.

```go
// MultipartWriter assembles a multipart/related body with the given root media type, e.g. "application/dicom".
func NewMultipartWriter(w io.Writer, rootType string) *MultipartWriter
func (mw *MultipartWriter) AddPart(contentType string, body io.Reader) error
func (mw *MultipartWriter) Close() (boundary string, err error)

// MultipartReader iterates parts of a multipart/related body from a bounded reader.
// MaxParts and MaxPartBytes default to the caps below; exceeding either returns ErrLimitExceeded.
type MultipartReader struct{ MaxParts int; MaxPartBytes int64 }
func NewMultipartReader(r io.Reader, mediaType string) (*MultipartReader, error)
func (mr *MultipartReader) NextPart() (contentType string, body io.Reader, err error)
```

### Content negotiation

The client and server negotiate representation through standard HTTP headers, with these defaults:

- **WADO-RS instances/frames** request `multipart/related; type="application/dicom"` (and
  `type="application/octet-stream"` for frames), with a `transfer-syntax` parameter built from
  `WithTransferSyntaxes`. The server honours the first acceptable transfer syntax it can produce and otherwise returns
  `406 Not Acceptable`.
- **WADO-RS metadata** and all **QIDO-RS** responses use `application/dicom+json` (`Accept: application/dicom+json`).
  XML (`application/dicom+xml`) is deferred; an `Accept` for it yields `406`.
- **STOW-RS** sends `Content-Type: multipart/related; type="application/dicom"; boundary=...` and accepts
  `application/dicom+json` for the store response.

## Server API

The server side is a set of `http.Handler`-producing constructors over three small backend interfaces, so the
storage, query, and retrieval policy is the consumer's. The interfaces are segregated by service (ISP, PRD §8.2):
a query-only deployment implements only `QueryBackend`.

```go
// Store persists posted instances and reports per-instance outcome (used by STOW-RS).
type StoreBackend interface {
    Store(ctx context.Context, ds *dicom.DataSet) error
}

// Retrieve resolves a ResourcePath to instances, metadata, frames, and bulk data (used by WADO-RS). The method set
// mirrors the client retrieval surface so the embeddable server can answer every WADO-RS resource the client requests
// and the conformance statement claims: /instances, /metadata, /frames, and /bulkdata.
type RetrieveBackend interface {
    RetrieveInstances(ctx context.Context, p ResourcePath) iter.Seq2[*dicom.DataSet, error]
    RetrieveMetadata(ctx context.Context, p ResourcePath) ([]*dicom.DataSet, error)
    RetrieveFrames(ctx context.Context, p ResourcePath, frames ...int) ([][]byte, error)
    RetrieveBulkData(ctx context.Context, uri BulkDataURI) ([]byte, error)
}

// Query answers QIDO-RS searches.
type QueryBackend interface {
    Query(ctx context.Context, q SearchQuery) iter.Seq2[*dicom.DataSet, error]
}

// NewServer wires the implemented backends into an http.Handler mounted at the DICOMweb root.
// Unimplemented services return 501 with a typed problem document, never a 200 no-op (PRD §9.2).
func NewServer(opts ...ServerOption) (*Server, error)

func WithStoreBackend(b StoreBackend) ServerOption
func WithRetrieveBackend(b RetrieveBackend) ServerOption
func WithQueryBackend(b QueryBackend) ServerOption
func WithMaxRequestBytes(n int64) ServerOption   // body cap (PRD §9.3)
func WithMaxMultipartParts(n int) ServerOption   // multipart part-count cap

func (s *Server) Handler() http.Handler          // mount under your own mux/middleware
```

The reference daemon (`server` package wiring, exercised by `radx dicomweb serve`) binds the handler to a
filesystem-backed `StoreBackend`/`RetrieveBackend` and an index-backed `QueryBackend`, listening on loopback by
default; a non-loopback bind is explicit (PRD §9.1).

## Behaviour and error model

Errors are returned as values, typed, and rendered in human terms (PRD §8.2). DICOM concepts appear by name —
tags as keyword plus `(gggg,eeee)`, transfer syntaxes and SOP classes by registered name, the STOW failure-reason
code (0008,1197) rendered by its registered name — and never carry PHI by default (PRD §9.1).

```go
// HTTPError carries the transport-level failure with enough context to act, without PHI.
type HTTPError struct {
    StatusCode int    // the HTTP status returned by the origin server
    Method     string // GET, POST
    URL        string // request URL with any query values redacted of PHI-bearing match keys
    Detail     string // server-provided problem detail, if any
}

func (e *HTTPError) Error() string

// Sentinel errors for the §9.3 caps and unsupported features. Check with errors.Is.
var (
    ErrLimitExceeded   = errors.New("dicomweb: input limit exceeded") // body/part/frame cap hit
    ErrNotAcceptable   = errors.New("dicomweb: no acceptable representation") // content negotiation failed (406)
    ErrUnsupported     = errors.New("dicomweb: service or media type not supported in v1") // 501/deferred surface
    ErrInvalidResource = errors.New("dicomweb: invalid resource path or UID")
)
```

Behavioural rules, in priority order:

1. **Fail-closed on partial work.** `Store` reports the parsed `StoreResponse` and a non-nil error when any instance
   failed; the caller decides whether partial acceptance is acceptable. The `radx dicomweb stow` command exits non-zero
   on any failed instance, matching the CLI exit-code taxonomy (PRD §8).
2. **Truncation is failure.** A short multipart part, a body that ends mid-frame, or a metadata document that ends
   mid-object yields `io.ErrUnexpectedEOF` wrapped in a typed error. A truncated transfer is never accepted as
   complete (PRD §9.2).
3. **Hostile input is bounded before allocation.** Response and request bodies are read through a bounded reader;
   multipart part counts and per-part sizes are capped; declared frame counts and lengths are validated against bytes
   actually remaining before any `make([]byte, n)`. Malformed input returns a typed error, never a panic (PRD §9.3).
4. **Context cancellation is honoured.** Every client and server method takes a `context.Context`; cancellation or
   timeout aborts the in-flight HTTP request and ends any iterator promptly (PRD §9.4).
5. **TLS by default.** The default `http.Client` verifies peer certificates (TLS 1.2+, prefer 1.3); `InsecureSkipVerify`
   is reachable only through an explicitly flagged test mode, never the default path (PRD §9.7).

### Hostile-input caps

These defaults apply to the client (`WithMaxResponseBytes`) and the server (`WithMaxRequestBytes`,
`WithMaxMultipartParts`) and are tunable per instance. They are the package's contribution to the §9.3 hardening floor;
exceeding any returns `ErrLimitExceeded` with the offending limit named in the message.

| Cap | Default | Option |
|-----|---------|--------|
| Max request/response body | 512 MiB | `WithMaxRequestBytes` / `WithMaxResponseBytes` |
| Max multipart parts | 10,000 | `WithMaxMultipartParts` |
| Max per-part size | 256 MiB | `MultipartReader.MaxPartBytes` |
| Max DICOM-JSON nesting depth | 64 | `JSONOption` (`WithMaxJSONDepth`) |

## Usage examples

### Retrieve a study and store it elsewhere

```go
package main

import (
    "context"
    "log"

    "github.com/codeninja55/go-radx/dicom"
    "github.com/codeninja55/go-radx/dicomweb"
)

func main() {
    ctx := context.Background()

    src, err := dicomweb.NewClient("https://pacs.example.org/dicom-web",
        dicomweb.WithBearerToken("..."))
    if err != nil {
        log.Fatalf("source client: %v", err)
    }
    dst, err := dicomweb.NewClient("https://archive.example.org/dicom-web",
        dicomweb.WithBearerToken("..."))
    if err != nil {
        log.Fatalf("dest client: %v", err)
    }

    studyUID := dicom.UID("1.2.840.113619.2.55.3.604688119.971.1577232000.123")
    study := dicomweb.NewStudy(studyUID)

    for ds, err := range src.RetrieveStudy(ctx, study) {
        if err != nil {
            log.Fatalf("retrieve: %v", err) // typed, names the failing instance, no PHI
        }
        resp, err := dst.Store(ctx, ds)
        if err != nil {
            log.Fatalf("store: %v (accepted %d, failed %d)", err, len(resp.Referenced), len(resp.Failed))
        }
    }
}
```

### QIDO-RS search at study level

```go
q := dicomweb.SearchQuery{
    Level: dimse.QueryLevelStudy,
    Match: map[dicom.Tag]string{
        dicom.TagPatientID:        "12345",
        dicom.TagStudyDate:        "20260101-20260531",
        dicom.TagModalitiesInStudy: "CT",
    },
    IncludeField: []dicom.Tag{dicom.TagStudyDescription},
    Limit:        50,
}

studies, err := client.Search(ctx, q)
if err != nil {
    log.Fatalf("qido: %v", err)
}
for _, ds := range studies {
    uid, _ := ds.GetUID(dicom.TagStudyInstanceUID)
    log.Printf("study %s", uid)
}
```

### Embed a STOW-RS + QIDO-RS server

```go
srv, err := dicomweb.NewServer(
    dicomweb.WithStoreBackend(myStore),     // implements StoreBackend
    dicomweb.WithQueryBackend(myIndex),      // implements QueryBackend
    dicomweb.WithMaxRequestBytes(256<<20),   // 256 MiB cap on POSTed bodies
)
if err != nil {
    log.Fatal(err)
}

mux := http.NewServeMux()
mux.Handle("/dicom-web/", http.StripPrefix("/dicom-web", srv.Handler()))

// Loopback bind by default; a non-loopback address is an explicit operator choice (PRD §9.1).
log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
```

## Conformance scope and limits

The published Conformance Statement (`docs/conformance/`) is the single source of truth for the supported subset; this
section summarises the v1 boundary.

**Supported in v1:**

- WADO-RS retrieval of instances, metadata, frames, and bulk data at study, series, and instance level.
- STOW-RS storage of `application/dicom` instances via `multipart/related`, with a parsed store response that
  distinguishes accepted from failed instances.
- QIDO-RS search at study, series, and instance level with `includefield`, `limit`, `offset`, `fuzzymatching`, and
  tag-keyed match parameters.
- DICOM-JSON (PS3.18 Annex F) encode/decode over `dicom.DataSet`, including PersonName component groups, nested `SQ`,
  `InlineBinary` (base64), and `BulkDataURI` references.
- `application/dicom+json` and `multipart/related` content negotiation with explicit `406`/`501` responses.
- Verified by live interop against Orthanc and dcm4chee-arc in CI (PRD §11.1).

**Not in v1 (deferred, returns a typed `ErrUnsupported` / `501`, never a silent no-op):**

- Legacy WADO-URI (the single-object query-parameter interface superseded by WADO-RS).
- Rendered and thumbnail retrieval (`/rendered`, `/thumbnail`); the package retrieves and decodes pixels but does not
  render display images (PRD §3.2 non-goals: no rendering/display pipeline beyond decode/encode).
- UPS-RS (Unified Procedure Step over the web).
- `application/dicom+xml` and YAML representations (DICOM-JSON only in v1).
- The capabilities resource (`OPTIONS /`) and bulk-data `STOW`.

**Out of scope (not a goal, per PRD §3.2):** a full PACS/archive product. The reference daemon is a thin, runnable
default for development and the CLI, not an archive; production storage, indexing, retention, and access control are
the integrating consumer's responsibility.

## See also

- [DIMSE networking](dimse.md) — the stateful TCP counterpart; `QueryLevel`, `Status`, and `AETitle` are shared types.
- [DICOM data model](dicom.md) — `DataSet`, `Tag`, `UID`, `TransferSyntax`, and the Part 10 codec.
- [DICOM PS3.18 (Web Services)](https://dicom.nema.org/medical/dicom/current/output/html/part18.html) — the standard
  this package conforms to.
- [DICOMweb overview](https://www.dicomstandard.org/using/dicomweb) — services and RESTful structure.
