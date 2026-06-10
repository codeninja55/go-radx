# DICOMweb parity matrix

This matrix compares the `dicomweb` package (client and embeddable server) and the `radx serve` DICOMweb role
against two references: the documented public surface of the Python
[dicomweb-client](https://dicomweb-client.readthedocs.io/en/latest/package.html) (ImagingDataCommons), and the
full transaction and resource set of DICOM PS3.18 (Web Services). PS3.18 is the ceiling: every service the
standard defines is enumerated, including the ones the package deliberately defers. The companion conformance
statement is [`../dicomweb.md`](../dicomweb.md); this matrix verifies its claims against code.

Status legend: MET (implemented with evidence), PARTIAL (implemented with a material limitation),
NOT-MET (absent), N-A (not applicable). Size estimates for non-MET rows: S (days), M (1-2 weeks), L (multi-week).

## Summary

Across 75 rows: 42 MET, 4 PARTIAL, 27 NOT-MET, 2 N-A.

The shipped core — QIDO-RS search (all six resource paths, full PS3.4 matching semantics), WADO-RS retrieval
(study/series/instance/metadata/frames/bulkdata), and STOW-RS storage (both body variants server-side) — is at
or above dicomweb-client parity on the transactions it implements, with stronger PHI hygiene, origin-scoped
credentials, and an AWS SigV4 adapter dicomweb-client lacks. The gaps are whole PS3.18 services and the
rendered/consumer-format surface:

1. UPS-RS Worklist Service, all 11 transactions, client and server (PS3.18 §11) — L
2. Rendered resources (instance/series/frames `/rendered`, server-side rendering to JPEG/PNG) — L
3. Server-side transcoding: the negotiation seam exists but no pixel-data transcoders ship — L
4. Non-Patient Instance Service (PS3.18 §12) — L
5. Notifications / WebSocket event channel (PS3.18 §8.10) — L
6. Thumbnail resources (`/thumbnail` at every level) — M
7. Pixel data resources of Table 10.1-1 — M
8. `application/dicom+xml` media type, responses and STOW bodies (answered 406 today) — M
9. WADO-URI legacy URI service (PS3.18 §9) — M
10. Capabilities discovery (`OPTIONS /`, PS3.18 §8.9) — M
11. STOW-RS metadata + bulkdata variant on the client (server accepts it; client posts whole objects) — M

## dicomweb-client parity (client-side)

Reference: dicomweb-client package API (readthedocs, fetched 2026-06-10). Rows cover `DICOMwebClient`,
`DICOMfileClient`, session utilities, and the CLI.

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Client construction (base URL, custom HTTP transport, headers) | `DICOMwebClient.__init__` | MET | `dicomweb/client.go:149` `NewClient`; `WithHTTPClient` (:60), `WithRoundTripper` (`auth.go:32`) | - | Custom headers via a caller RoundTripper |
| Per-service URL prefixes (`qido/wado/stow/delete_url_prefix`) | `DICOMwebClient.__init__` | NOT-MET | single `baseURL` only (`client.go:149`) | S | Rarely needed; most origins share one base path |
| Streaming / `chunk_size`, `set_chunk_size` | `DICOMwebClient.set_chunk_size` | PARTIAL | streaming iterators (`client_retrieve.go:22,29`); `WithMaxResponseBytes` (`client.go:72`) | S | Reads stream and are bounded; no chunked-transfer control on store |
| HTTP retry (`set_http_retry_params`) | `DICOMwebClient.set_http_retry_params` | NOT-MET | no retry logic in `dicomweb/client.go` | S | Achievable today via a retrying `WithRoundTripper` |
| `search_for_studies` | QIDO-RS study query | MET | `dicomweb/qido_client.go:49` `SearchStudies` | - | |
| `search_for_series` | QIDO-RS series query | MET | `qido_client.go:56` `SearchSeries` (with or without study scope) | - | |
| `search_for_instances` | QIDO-RS instance query | MET | `qido_client.go:67` `SearchInstances` | - | |
| Search params: `fuzzymatching`, `limit`, `offset`, `fields`, `search_filters` | search method params | MET | `qido_client.go:21` `SearchQuery` (Fuzzy, Limit, Offset, IncludeFields, IncludeAll, Match) | - | Query string stripped from errors (PHI) |
| Auto-pagination (`get_remaining`) | search method params | NOT-MET | manual Limit/Offset only | S | Caller loops on offset; Warning 299 signals truncation server-side |
| `retrieve_study` / `iter_study` | WADO-RS study retrieve | MET | `client_retrieve.go:22` `RetrieveStudy` (iterator); `:64` `RetrieveStudyObjects` (byte-preserving) | - | Iterator-first design covers both Python forms |
| `retrieve_series` / `iter_series` | WADO-RS series retrieve | MET | `client_retrieve.go:29` `RetrieveSeries`; `:71` `RetrieveSeriesObjects` | - | |
| `retrieve_instance` | WADO-RS instance retrieve | MET | `dicomweb/client.go:327` `RetrieveInstance`; `:342` `RetrieveInstanceObject` | - | Object form preserves origin transfer syntax byte-for-byte |
| `retrieve_study/series/instance_metadata` | WADO-RS metadata | MET | `client_retrieve.go:156` `RetrieveMetadata(ResourcePath)` covers all three levels | - | `application/dicom+json` only |
| `retrieve_instance_frames` / `iter_instance_frames` | WADO-RS frames | MET | `client_retrieve.go:186` `RetrieveFrames` (1-based) | - | `application/octet-stream` parts only |
| Per-call `media_types` (incl. transfer-syntax UID pairs) | retrieve method params | PARTIAL | client-level `WithTransferSyntaxes` (`client.go:78`) drives the Accept header | M | No per-call media type; no consumer-format (image/jpeg) Accept |
| `retrieve_bulkdata` (by URL) | WADO-RS bulkdata | MET | `client_retrieve.go:196` `RetrieveBulkData`; `:222` `ResolveBulkDataURI`; `bulkdata.go` `BulkDataURIs` | - | Origin-scoped: foreign-host URIs blocked unless allowlisted (`client.go:98,108`) |
| `byte_range` on bulkdata retrieval | `retrieve_bulkdata(byte_range=...)` | NOT-MET | no Range header in `client_retrieve.go` | S | |
| `retrieve_instance_rendered` | WADO-RS rendered | NOT-MET | no rendered path in `dicomweb/` | L | Deferred in conformance v1 (`../dicomweb.md` out-of-scope) |
| `retrieve_series_rendered` | WADO-RS rendered | NOT-MET | none | L | |
| `retrieve_instance_frames_rendered` | WADO-RS rendered frames | NOT-MET | none | L | |
| `store_instances` (optionally to a study) | STOW-RS store | MET | `client.go:241` `Store`; `:251` `StoreToStudy`; fail-closed `*StoreError` on 202/409 | - | Stricter than dicomweb-client: partial store is an error |
| `delete_study/series/instance` | non-standard delete | NOT-MET | no DELETE in `dicomweb/` | S | Not a PS3.18 transaction; vendor extension (Orthanc, Google) |
| `DICOMfileClient` (same API over local Part 10 files) | `DICOMfileClient` | NOT-MET | none | L | Nearest substitute: embeddable `dicomweb.Server` + `server/filestore.go` |
| `lookup_keyword` / `lookup_tag` | static helpers | MET | `dicom/dictionary.go:51` `LookupKeyword`; entry `Keyword` field (`dictionary.go:9`) for the reverse | - | Lives in the `dicom` package, not `dicomweb` |
| Session from username/password | `create_session_from_user_pass` | MET | `dicomweb/auth.go:39` `WithBasicAuth` (origin-scoped) | - | |
| Session from auth object / token | `create_session_from_auth` | MET | `auth.go:32` `WithRoundTripper`; `:54` `WithTokenSource`; `client.go:67` `WithBearerToken` | - | OAuth2 refresh handled by the token source |
| Session from GCP credentials | `create_session_from_gcp_credentials` | MET | `dicomweb/auth/gcp` `TokenSource` (ADC, cloud-platform scope); tested in `gcp_test.go` | - | go-radx adds AWS SigV4 (`dicomweb/auth/aws`), absent in dicomweb-client |
| Certificates on the session (`add_certs_to_session`) | session_utils | PARTIAL | `auth.go:73` `WithClientCertificate` (mTLS) | S | Custom CA bundle only via a caller-built `WithHTTPClient` |
| CLI: search / retrieve / store | `dicomweb_client` CLI | MET | `cmd/radx/internal/command/dicomweb.go` (`qido`, `wado` incl. `--metadata`, `stow`) | - | No rendered/delete CLI verbs (features absent) |
| `configure_logging` | logging helper | N-A | library logs via injected logger; PHI-free policy | - | Different design intent, not a gap |

## PS3.18 transaction table — client support

References: PS3.18 §8 (common), §9 (URI Service), §10 (Studies Service, Table 10.1-1: 28 resources incl.
metadata, bulkdata, pixel data, rendered, thumbnail variants), §11 (Worklist Service / UPS-RS), §12
(Non-Patient Instance Service). Fetched from dicom.nema.org 2026-06-10.

| Transaction / capability | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Retrieve Capabilities (`OPTIONS /`) | §8.9 | NOT-MET | none | M | Standard says all RESTful services shall implement it |
| Search studies | §10.6, `/studies` | MET | `qido_client.go:49` | - | |
| Search series (incl. `/studies/{s}/series`) | §10.6 | MET | `qido_client.go:56` | - | |
| Search instances (all three paths) | §10.6 | MET | `qido_client.go:67` | - | |
| Retrieve study | §10.4, `/studies/{s}` | MET | `client_retrieve.go:22` | - | Streaming multipart of `application/dicom` |
| Retrieve series | §10.4 | MET | `client_retrieve.go:29` | - | |
| Retrieve instance | §10.4 | MET | `client.go:327` | - | |
| Retrieve metadata (study/series/instance) | §10.4, `.../metadata` | MET | `client_retrieve.go:156` | - | JSON only; no `application/dicom+xml` Accept |
| Retrieve frames | §10.4, `.../frames/{list}` | MET | `client_retrieve.go:186` | - | Octet-stream parts; no consumer media types |
| Retrieve bulkdata | §10.4, `.../bulkdata` | MET | `client_retrieve.go:196,222` | - | No Range requests |
| Retrieve pixel data resources | Table 10.1-1 pixel data | NOT-MET | none (`pixeldata` appears only in path redaction, `client.go:510`) | M | |
| Retrieve rendered (study/series/instance/frames, incl. volumetric) | §10.4 rendered | NOT-MET | none | L | |
| Retrieve thumbnail (every level) | §10.4 thumbnail | NOT-MET | none | M | |
| Store instances (`POST /studies[/{s}]`) | §10.5 | MET | `client.go:241,251`; status mapping per §10.5.3 (`client.go:295`) | - | |
| Store metadata + bulkdata variant (client body) | §10.5 | NOT-MET | client encodes whole objects only (`client.go:270`) | M | Server side accepts both variants |
| Transfer-syntax negotiation in Accept | §8.7.3.5.2 | MET | `client.go:78` `WithTransferSyntaxes` -> `acceptInstances` (`client_retrieve.go:104`) | - | |
| `application/dicom+xml` media type | §8.7.3 / Annex A | NOT-MET | deferred (`negotiation.go:267`) | M | Deliberate; recorded in conformance out-of-scope |
| URI Service (WADO-URI) retrieve | §9, `?requestType=WADO` | NOT-MET | none | M | Legacy; superseded by WADO-RS for in-scope workflows |
| Worklist Service (UPS-RS), all transactions | §11 | NOT-MET | none | L | DIMSE Modality Worklist is the worklist surface today |
| Notifications (WebSocket event channel) | §8.10 / §11 | NOT-MET | none | L | Depends on UPS-RS |
| Non-Patient Instance Service | §12 | NOT-MET | none | L | Hanging protocols, color palettes, implant templates |

## PS3.18 transaction table — server support

Two server surfaces exist: the embeddable `dicomweb.Server` (library) and the `radx serve` DICOMweb role
(`server/role_dicomweb.go`), which wires the library server over the shared object store and catalogue. Where
they differ, status reflects the library and the daemon gap is noted.

| Transaction / capability | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Retrieve Capabilities (`OPTIONS /`) | §8.9 | NOT-MET | router handles GET/POST only (`server.go:134`) | M | Unrouted paths answer 501, never a silent empty body |
| Search studies/series/instances (all six resource paths) | §10.6 | MET | `server.go:266,270` routing; `qido_server.go:69`; daemon: `role_dicomweb.go:103` `QueryBackend` | - | |
| Matching semantics (single, wildcard, UID list, range, universal, fuzzy PN) | PS3.4 C.2.2.2 | MET | `dicomweb/qido_match.go`; tests `qido_test.go` | - | Fuzzy is substring, not phonetic (documented) |
| `includefield` + default return attributes | §10.6, Tables 10.6.1-5/-5a/-5b | MET | `qido.go:114,196` | - | Unresolvable attribute rejected, not dropped |
| `limit`/`offset` + Warning 299 on truncation | §10.6.1.4 | MET | `qido_server.go:113,131`; cap via `WithMaxQueryResults` (`server.go:74`) | - | |
| Retrieve instance | §10.4 | MET | `server.go:513` `handleRetrieveInstance`; daemon: `role_dicomweb.go:193` | - | |
| Retrieve study | §10.4 | MET | library: `retrieve.go:98` via optional `StudyRetriever` (:56); daemon: `role_dicomweb.go:219` `RetrieveStudy`; `TestDaemonDICOMwebRetrievalRoutes` (`daemon_roles_test.go`) | - | Daemon enumerates the study through the Catalogue and fetches from the ObjectStore |
| Retrieve series | §10.4 | MET | library: `retrieve.go:123` via `SeriesRetriever` (:61); daemon: `role_dicomweb.go:226` `RetrieveSeries` | - | |
| Retrieve metadata | §10.4.1.1.5 | MET | library: `retrieve.go:208` via `MetadataRetriever` (:76); BulkDataURI emission per instance; daemon: `role_dicomweb.go:260` `RetrieveMetadata` | - | JSON only |
| Retrieve frames | §10.4 | MET | library: `retrieve.go:292` via `FrameRetriever` (:83); daemon: `role_dicomweb.go:280` `RetrieveFrames` (native frame slicing via `dicom.PixelData`) | - | |
| Retrieve bulkdata | §10.4 | PARTIAL | library: `retrieve.go:314` via `BulkDataRetriever` (:89); daemon: `role_dicomweb.go:326` `RetrieveBulkData` (top-level binary values) | M | Per-attribute selection missing (whole-instance bulkdata returned) |
| Transfer-syntax negotiation (passthrough / transcode / 406) | §8.7.3.3 | MET | `negotiation.go:168` `negotiateRetrieveTransferSyntax`; `StoredInstanceRetriever` (`retrieve.go:69`); tests `negotiation_test.go` | - | Wildcard never transcodes; unservable syntax answers 406 |
| Pixel-data transcoding (shipped transcoders) | §8.7.3.5.2 | NOT-MET | seam exists; no transcoders registered (`instance.go:55`, `../dicomweb.md`) | L | Honest 406 today; a deployment may supply transcoders |
| Store whole-object (`type="application/dicom"`) | §10.5 | MET | `server.go:356` `storeDICOMParts`; daemon: `role_dicomweb.go:101` | - | Study-constrained target enforced (0xC120 rejection) |
| Store metadata + bulkdata (`type="application/dicom+json"`) | §10.5 | MET | `server.go:422` `storeMetadataBulkData`; reference resolution fail-closed | - | |
| Store response document (Referenced/Failed SOP seq, Retrieve URLs, 200/202/409) | §10.5.3 | MET | `dicomweb/store_response.go`; `server.go:471`; `WithStoreRetrieveURLBase` (`server.go:100`) | - | Warning Reason via `WarnableStoreBackend` |
| Failure/Warning reason codes | §10.5.3.1 | MET | code table in `store_response.go` / `store.go`; `FailureReasonError` | - | Unregistered codes rendered as hex, never dropped |
| Rendered / thumbnail / pixel data resources | §10.4, Table 10.1-1 | NOT-MET | none | L | |
| `application/dicom+xml` (responses, STOW bodies) | Annex A | NOT-MET | gated 406 / unparsed (`negotiation.go:75,267`, `server.go:327`) | M | |
| URI Service (WADO-URI) | §9 | NOT-MET | none | M | |
| Worklist Service (UPS-RS) | §11 | NOT-MET | none | L | |
| Notifications (WebSocket) | §8.10 | NOT-MET | none | L | |
| Non-Patient Instance Service | §12 | NOT-MET | none | L | |
| Delete (vendor extension) | not in PS3.18 | N-A | none | - | Non-standard; listed only because dicomweb-client exposes it |

Security posture (verified, not gaps): the daemon binds loopback by default and refuses a non-loopback bind
without explicit opt-in (`server/bind.go:8`, `ErrInsecureBind`); client TLS verification is on by default with
`WithInsecureSkipVerify` as an explicit opt-in (`client.go:119-136`); credentials are origin-scoped
(`auth.go`); QIDO query strings and resource UIDs are redacted from errors and logs. Five fuzz targets cover
the trust-boundary parsers (`parser_fuzz_test.go`: 3, `qido_fuzz_test.go`: 2).

## Methodology

- Date: 2026-06-10. go-radx at `main` (fdf7b54).
- dicomweb-client surface: Context7 (`/imagingdatacommons/dicomweb-client`, source reputation High) plus the
  readthedocs package API page (en/latest). Version pin of the docs not stated on the page; the surface
  matches the 0.5x series.
- PS3.18: dicom.nema.org current edition, chunked HTML — chapter 9 (URI Service), chapter 10 (Studies
  Service, Table 10.1-1), chapter 11 (Worklist Service), section 8.9 (Capabilities). The full per-transaction
  requirement tables of §10.2-10.6 and §11.2-11.11 were not fetched in full; transaction enumeration for UPS-RS
  is from the §11 resource table plus the standard's structure, so per-transaction nuances (e.g. UPS state
  machine details) are summarized, not audited line-by-line.
- go-radx evidence: read/grepped `dicomweb/` and `server/role_dicomweb.go`; `docs/conformance/dicomweb.md`
  used as a map and spot-verified against code (routing, backends, negotiation, store response, auth options,
  CLI). No code was modified and no tests were run for this audit; "tested in" claims cite test files present
  in the tree, not executions performed here.
- Unverified areas: interop tests (`dicomweb/integration`, Orthanc-gated) were not executed; Table 10.1-1's
  exact 28-resource enumeration (pixel data and volumetric rendered variants) relies on a summarized fetch of
  chapter 10 rather than the raw table; dicomweb-client behavioural details beyond its documented signatures
  (e.g. exact retry semantics) were not exercised.
