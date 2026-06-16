# FHIR feature-parity matrix

This matrix scores the go-radx FHIR subsystem — `fhir/` (R4/R5 models, primitives, validation), `fhir/rest/`
(client), and the server FHIR role (`server.NewFHIRRole`) — against two references: **fhir.resources**
(nazrulworld, Python) for model coverage, validation semantics, and serialization; and **HAPI FHIR** plus the
normative **FHIR R5 http page** for the RESTful operation surface. samply/golang-fhir-models is cross-checked
for Go-ecosystem expectations. Status values: MET, PARTIAL, NOT-MET, N-A. Size (S/M/L) estimates the build
effort for non-MET rows. Evidence is `file:symbol` in this repository; tests count toward MET.

## Summary

Across the four sections: **53 MET, 9 PARTIAL, 31 NOT-MET, 5 N-A** (98 rows).
Section split (MET/PARTIAL/NOT-MET/N-A): models 7/0/2/2, primitives-validation 11/2/6/2, REST client
20/3/13/1, server role 15/4/10/0. The model and client layers are near parity; the server role's read-
and write-sides are now both whole: versioning, vread, instance history, ETag/Last-Modified/If-Match,
server-side `$validate`, the `radx serve fhir` daemon (wave 0), and update/patch/delete with their
conditional forms (see the flipped rows below).

Top gaps, largest first:

1. **STU3 models** (L) and **R4B models** (M) — fhir.resources ships both as sub-packages; go-radx ships
   neither (deliberate v1 deferral). For STU3 the plan calls for a structural assessment of whether the
   `fhir/internal/gen` pipeline fits the STU3 definition-bundle shape before any build decision.
2. **SMART on FHIR** (L) — documented deferral on both client and server.
3. **XML serialization** (L) and YAML (M) — fhir.resources has both (experimental); go-radx is JSON-only by
   declared scope.
4. **Operations framework** (`$everything`, custom operations; client-side `$validate`) (M) — the server
   now ships `$validate` (`server/fhir_handlers.go:handleValidate`); the client has no operation
   invocation API and `$everything` is absent on both sides. HAPI treats this as core surface.
5. **Batch at the system endpoint** (M) — the server explicitly rejects a `batch` Bundle; only `transaction`
   is processed (`server/fhir_handlers.go:handleTransaction`). The client sends both.
6. **Search depth on the server** (M) — the handler forwards raw params, but the shipped `MemoryRepository`
   matches `_id` only and ignores `_count`/`_sort`/`_include`; no paging links are emitted.
7. **Nested-backbone validation** (M) — `fhir.Validate` covers top-level required/choice/binding only.
8. **Primitive lexical validation** (M) — date/dateTime/time calendar and offset rules deferred to the HL7
   validator CI gate; fhir.resources enforces them at parse time via pydantic field patterns.
9. **Client conditional update/patch/delete** (S each) and client **type/system history** (S) — the server
   role now ships the conditional writes (rows below); the client still lacks them.

Where go-radx *exceeds* the references: required-binding enums are closed and enforced at the JSON boundary
(fhir.resources exposes enum values but does not enforce them); R4 4.0.1 ships as a first-class generated
release (fhir.resources dropped R4 for R4B at v7; samply is R4-only with no R5); the lexical-preserving
`fhir.Decimal` and the byte-stable regeneration gate have no fhir.resources equivalent.

## Model coverage per FHIR release

Reference: fhir.resources README (R5 5.0.0 default; R4B 4.3.0 and STU3 3.0.2 as sub-packages; R4 dropped at
v7.0.0). Cross-check: samply/golang-fhir-models (R4 only).

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| R5 5.0.0 full resource set | fhir.resources default release | MET | `fhir/r5/` generated from pinned 5.0.0 bundle; 158 factories in `fhir/r5/registry.go:init` | — | Generated from `StructureDefinition`s, checksum-pinned (`fhir/internal/gen/testdata/definitions/r5`) |
| R4 4.0.1 full resource set | samply golang-fhir-models (R4) | MET | `fhir/r4/` generated; 146 factories in `fhir/r4/registry.go:init` | — | Exceeds fhir.resources, which dropped R4 for R4B at v7.0.0 |
| R4B 4.3.0 | fhir.resources `fhir.resources.R4B` sub-package | NOT-MET | `docs/conformance/fhir.md` "Out of scope / deferred": R4B deferred | M | Generator is release-data-driven; an R4B bundle should fit the existing pipeline with low structural risk |
| STU3 3.0.2 | fhir.resources `fhir.resources.STU3` sub-package | NOT-MET | same deferral; "FHIR STU3 — out of scope" | L | Plan requires a structural assessment of whether `fhir/internal/gen` (loader/model/plan/emit) fits the STU3 bundle shape before building — do not assume the R4/R5 pipeline transfers |
| R6 | not shipped by fhir.resources either | N-A | `docs/conformance/fhir.md`: generator designed for R6 when normative | — | Parity holds; both wait on a normative R6 |
| Release selected by import path / sub-package | fhir.resources sub-package layout | MET | `fhir/r4`, `fhir/r5` distinct type spaces; `fhir.Release` constants in `fhir/release.go` | — | Same model as fhir.resources; no runtime flag |
| Backbone elements as real nested types | fhir.resources nested models | MET | `fhir/internal/gen/model` contentReference graft; backbones deduplicated in `fhir/internal/gen/plan` | — | Fixes the prototype's empty-backbone defect |
| Required-binding value sets as enums | samply "enums for every ValueSet used in a required binding" | MET | generated `ParseXxx` + strict `UnmarshalJSON`; `fhir/binding.go:DecodeCode`; guard `TestNoEmptyRequiredBindingEnum` | — | Exceeds fhir.resources (enum values exposed, not enforced) and samply (no boundary enforcement) |
| Non-enumerable terminology boundary documented | samply open TODO "ValueSets Referring to Multiple CodeSystems" | MET | documented not-inlined boundary; intensional/external/composed sets emit `code` string with godoc reason | — | go-radx resolves what samply lists as an open TODO |
| Cross-release conversion | (neither reference ships one) | N-A | explicit no-implicit-conversion stance, `docs/conformance/fhir.md` | — | Deliberate; convert package handles workflow resources via DICOM/HL7 paths |
| Byte-stable regeneration gate | (no reference equivalent) | MET | `TestRegenerationByteForByte`; mise `gen:verify`, `gen:fhir-r4`/`gen:fhir-r5` (`mise.toml:54-74`) | — | go-radx-specific strength; generated code never hand-edited |

## Primitives, choice types, extensions, validation semantics

Reference: fhir.resources (pydantic v2 validation, `_field` siblings, summary serialization, YAML/XML
experimental, `fhir_comments`, `add_root_validator`).

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Choice (`[x]`) types, one-branch exclusivity | fhir.resources choice fields via pydantic | MET | sealed interface + `SetValueX` setters clear siblings; `fhir/validate.go:CountSet` re-checks per group | — | Stronger than fhir.resources: compile-time sealed branch set plus validator backstop |
| Primitive `_field` extension siblings | fhir.resources "FHIR Extensibility for Primitive Data Types" | MET | `fhir/primitive.go:PrimitiveElement`, `MarshalPrimitiveExtensions`, null-aligned repeating arrays | — | Round-trips scalar and repeating siblings; no spurious sibling on complex fields |
| Value-side `null` for extension-only repeating positions | FHIR JSON spec; fhir.resources handles via Optional values | PARTIAL | documented limitation in `docs/conformance/fhir.md` "Limitation (repeating extension-only positions)" | S | `_field` data survives; the value-side `null` placeholder re-marshals as the Go zero value |
| Lexical-preserving decimal | fhir.resources uses Python `Decimal` | MET | `fhir/decimal.go:Decimal` (`String`, `Float64`, `Exact`, `BigFloat`, `MarshalJSON`) | — | Shared with DICOM DS/IS; never `float64` |
| integer64 as JSON string (R5) | FHIR R5 wire form | MET | `FHIRInteger64` quoted-string codec, `fhir/r5/primitives.go` | — | |
| Cardinality / required-presence validation | pydantic field cardinality | PARTIAL | `fhir.Validate` + generated descriptors (`fhir/r5/validation_descriptors.go`); presence via non-nil pointer | M | Top-level elements only; nested-backbone required/choice/binding checks deferred. fhir.resources validates at every nesting level on construction |
| Primitive lexical validation (date/dateTime/time) | pydantic field regex/patterns | NOT-MET | deferred per `docs/conformance/fhir.md` "Scope boundary (v1)"; decimal and binding checks done | M | Authoritative cover is the HL7 validator CI gate; in-process gate does not check calendar/offset rules |
| `resourceType` integrity on decode and dispatch | fhir.resources `get_resource_type()` / construction | MET | `fhir.Unmarshal[T]` mismatch rejection, `fhir/registry.go`, `UnmarshalResourceSlice` fail-whole | — | Fuzz-guarded: 5 fuzz targets (`fhir/r5/fuzz_test.go`: 3; `fhir/r5/validate_test.go`: 2) |
| Bundle `bdl-*` invariant validation | (no fhir.resources equivalent; HAPI validates) | MET | release builders (`bundle_builders.go`) + wire-side checks in `fhir/r5/validate.go` | — | bdl-9/bdl-10 (document identifier/timestamp) and bdl-18 (searchset self link) documented as deferred |
| Intra-Bundle / contained reference integrity | HAPI validation chain | MET | `Bundle.Resolve`, `DomainResource.ResolveContained`, `Bundle.CheckReferenceIntegrity` per release | — | Aggregated `OperationOutcome`, malformed contained slots named by index |
| Validation result as OperationOutcome | fhir.resources raises pydantic ValidationError | MET | `fhir/outcome.go:OperationOutcome`, `fhir/validate.go:Validate` (all-issues-in-one-pass) | — | Severity-classifiable; never panics on malformed input |
| Custom/root validators extension point | fhir.resources `add_root_validator()` | N-A | `fhir/validate.go:RegisterValidationDescriptor` is the descriptor seam | — | Different idiom: Go consumers compose checks around `Validate`; not a like-for-like hook |
| IG / profile validation (US Core etc.) | HAPI profile validation | NOT-MET | documented deferral, `docs/conformance/fhir.md` | L | fhir.resources also does **not** do profile validation, so vs that reference this is parity; vs HAPI it is a gap |
| JSON serialization (canonical, round-trip) | fhir.resources `model_dump_json` | MET | generated `MarshalJSON`/`UnmarshalJSON`; snapshot-order canonical form; `AppendSiblings` | — | Byte-stable round-trip is beyond fhir.resources, which re-orders per pydantic |
| YAML serialization | fhir.resources `model_dump_yaml` (experimental) | NOT-MET | deferred; typed "format not supported" stance in `docs/conformance/fhir.md` | M | |
| XML serialization | fhir.resources `model_dump_xml` (experimental) | NOT-MET | same deferral | L | XML is a normative FHIR format; relevant if non-JSON peers appear |
| `_summary` serialization modes | fhir.resources `summary_only=True` | MET | `fhir/summary.go:MarshalSummary`, five modes, SUBSETTED tagging, generated descriptors | — | Exceeds fhir.resources: text/data/count modes and meta-tagging, not just isSummary filtering |
| Element-subset serialization (`_elements`-style) | pydantic `include`/`exclude` on dump | NOT-MET | no per-element filter API in `fhir/summary.go` | S | The `filterTopLevel` machinery is reusable for an `_elements` filter |
| `fhir_comments` support | fhir.resources fhir_comments field | NOT-MET | no support | S | Non-normative convention; low value — recommend explicit N-A decision rather than build |
| Resource id length/charset relaxations | fhir.resources monkey-patch knobs | N-A | — | — | go-radx validates structurally; no equivalent knob needed in v1 scope |
| Fuzzing of decode/validate surfaces | (no reference equivalent) | MET | 5 fuzz targets across `fhir/r5/fuzz_test.go` and `fhir/r5/validate_test.go`, seed corpora, CI fuzz job | — | go-radx-specific strength |

## REST client surface

Reference: FHIR R5 http.html interaction table (normative) and HAPI `IGenericClient` fluent surface. The
go-radx client is `fhir/rest.Client`, release-fixed, `application/fhir+json` only.

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| read | http.html#read | MET | `fhir/rest/crud.go:Read` | — | 404 → `ErrNotFound` sentinel |
| Conditional read (If-Modified-Since / If-None-Match) | http.html#cread | NOT-MET | no cache-validator headers in `Read` | S | |
| vread | http.html#vread | MET | `fhir/rest/crud.go:VRead` | — | |
| update (PUT, update-as-create) | http.html#update | MET | `fhir/rest/crud.go:Update`; 200 and 201 both success | — | |
| Version-aware update (If-Match / ETag) | http.html#concurrency | MET | `Update`/`Patch` ifMatch param; `Result.ETag`/`VersionID` from headers | — | 412 → `ErrConflict` |
| Conditional update (PUT `[type]?[params]`) | http.html#cond-update | NOT-MET | `Update` requires a non-empty id | S | |
| patch (JSON Patch) | http.html#patch | MET | `fhir/rest/crud.go:Patch`, `application/json-patch+json` | — | |
| FHIRPath Patch | http.html#patch (FHIRPath form) | NOT-MET | only JSON Patch content type sent | M | Needs Parameters-based patch body support |
| Conditional patch | http.html#cond-patch | NOT-MET | — | S | |
| delete | http.html#delete | MET | `fhir/rest/crud.go:Delete`; 409 → `ErrConflict` | — | |
| Conditional delete (single/multiple) | http.html#cond-delete | NOT-MET | — | S | |
| history-instance | http.html#history | MET | `fhir/rest/crud.go:History`; pages via `FollowNext` | — | |
| history-type / history-system | http.html#history | NOT-MET | `History` takes (type, id) only | S | Path-only addition |
| history `_since` / `_at` / `_count` params | http.html#history | NOT-MET | `History` accepts no params | S | |
| create | http.html#create | MET | `fhir/rest/crud.go:Create`; Prefer: return=representation | — | |
| Conditional create (If-None-Exist) | http.html#ccreate | MET | `Create` ifNoneExist → `If-None-Exist` header; 200-match handled | — | |
| search (type-level GET) | http.html#search | MET | `fhir/rest/search.go:Search` | — | |
| Search modifiers (`:exact`, `:contains`, ...) | search.html#modifiers | MET | `SearchParams.Modifier` | — | |
| Chained parameters (single-level) | search.html#chaining | MET | `SearchParams.Chain` | — | Deep multi-hop chains pass through as raw params (documented client stance) |
| `_include` / `_revinclude` (incl. `:iterate`, wildcard) | search.html#include | MET | `SearchParams.Include`/`RevInclude`; values pass through | — | |
| `_count`, `_sort` | search.html | MET | `SearchParams.Count`/`Sort` | — | |
| `_summary` / `_elements` request params | search.html#summary | PARTIAL | no dedicated helper; `SearchParams.Add("_summary", ...)` works | S | Raw `Add` covers any parameter, including `_has` reverse chaining |
| Search via POST `[type]/_search` | http.html#search | NOT-MET | GET-only in `Search` | S | Needed for long/PHI-bearing queries kept out of URLs |
| search-system (GET `[base]?`) | http.html#search | NOT-MET | `Search` requires a resourceType | S | |
| Compartment search | search.html#compartments | NOT-MET | — | M | |
| Paging (Bundle.link first/prev/next) | http.html#paging | MET | `SearchPage.HasNext/HasPrev`, `FollowNext`, `FollowPrev`, `SearchAll(maxPages)` | — | Link resolution against request URL (`search.go:resolveLink`) |
| transaction / batch submit | http.html#transaction | MET | `fhir/rest/transaction.go:Transaction` (release-checked bundle + entries) | — | One method serves both Bundle.types; per-entry batch outcomes left to caller by design |
| capabilities (GET /metadata) | http.html#capabilities | MET | `fhir/rest/capability.go:Capabilities`, `SupportsResourceInteraction`, `SupportsTransaction` | — | Both R4 and R5 CapabilityStatements parsed (`release.go:r4Capability`/`r5Capability`) |
| Extended operations (`$validate`, `$everything`, custom) | HAPI operations; operations.html | NOT-MET | no operation invocation API in `fhir/rest` | M | Needs Parameters in/out plumbing; `$everything` is common in radiology integrations |
| Prefer header control (minimal/representation/OperationOutcome) | http.html#ops | PARTIAL | always sends `return=representation` (`client.go:50`); bodyless 204 handled (`crud.go:isWriteSuccess`) | S | Not caller-selectable |
| HEAD support | http.html#head | NOT-MET | — | S | |
| `_format` / XML negotiation | http.html#mime-type | N-A | JSON-only by declared scope; `application/fhir+json` sent and accepted | — | Deliberate v1 boundary |
| OperationOutcome-typed errors | HAPI exception hierarchy | MET | `fhir/rest/errors.go:OperationOutcomeError` + sentinels (`ErrNotFound`, `ErrConflict`, `ErrUnprocessable`, `ErrUnauthorized`, `ErrUnsupported`) | — | |
| Auth: bearer token, same-origin confinement | HAPI BearerTokenAuthInterceptor | MET | `fhir/rest/auth.go:bearerAuthLayer`, `requestSameOrigin` | — | Token never sent cross-origin |
| SMART on FHIR | HAPI SMART support | NOT-MET | documented deferral (`docs/conformance/fhir.md`, `cli-server.md`) | L | A SMART-obtained token can be injected via `WithBearerToken` today |
| Response-size bounding / hostile-response safety | (go-radx-specific) | MET | `request.go:boundedBody`/`cappedReader`, `WithMaxResponseBytes`; PHI-redacting transport errors (`redactURL`) | — | Exceeds both references |
| Cross-implementation interop test vs HAPI | HAPI reference server | MET | `interop:fhir-hapi` CI leg: `fhir/rest/hapi_main_test.go` provisions the pinned HAPI container; `fhir/rest/interop_test.go:TestInteropHAPIServer` runs against it | - | Wired by issue #114; `RADX_FHIR_HAPI_BASE` substitutes an external server |

## Server FHIR role surface

Reference: FHIR R5 http.html interaction table and the HAPI plain-server operation set. The go-radx role is
`server.NewFHIRRole` (mounted via `server.WithFHIR`), one release per instance, serving the workflow resource
set over a pluggable `Repository` (`server/fhir_repository.go:Repository`).

| Feature | Reference anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| capabilities (GET /metadata) | http.html#capabilities | MET | `server/fhir_handlers.go:handleMetadata`; `server/fhir_release.go:capabilityStatement` (R4 + R5) | — | Advertises only implemented interactions — honest CapabilityStatement |
| read | http.html#read | MET | `server/fhir_handlers.go:handleRead`; 404 → OperationOutcome | — | |
| vread | http.html#vread | MET | `server/fhir_handlers.go:handleVRead` over `Repository.VRead`; 404-vs-410 split via `ErrGone`; tests `server/fhir_versioning_test.go:TestFHIRRoleVRead` / `TestFHIRRoleVReadDeletedVersionIsGone`, client interop `TestFHIRRoleClientVReadAndHistory` | — | Version store: `server/fhir_repository.go` (`byVer`, `ResourceVersion`) |
| history-instance | http.html#history | MET | `fhir_handlers.go:handleHistoryInstance` + `historyEntries`; history Bundle with per-version entry.request/response and absolute per-entry `fullUrl` (`absoluteResourceURL`, identical across versions, present on deleted entries), newest first, release-validated in `TestFHIRRoleHistoryInstance` / `TestFHIRRoleHistoryRendersUpdatesAndDeletes` | — | bdl-1/bdl-3 honoured; deleted versions render resource-less DELETE entries; `_count` honoured as a cap (newest entries, `total` stays the full count, `TestFHIRRoleHistoryInstanceHonoursCount`) — paging links deferred |
| history-type / history-system | http.html#history | NOT-MET | routes do not exist | M | |
| update (PUT) | http.html#update | MET | `fhir_write.go:handleUpdate` over `Repository.Update`; resourceType/id integrity (400 on mismatch, `checkUpdateIntegrity`), release-validated, version bumped, `If-Match` 412 via an atomic compare-and-swap inside the repository write lock (`ifMatchVersion` → `Update(..., expectedVersion)` → `checkExpectedVersion`, `ErrVersionConflict`); 200 existing / 201 update-as-create with versioned `Location`; update-as-create reserves the client-chosen numeric id so a later create never re-mints it (`reserveID`); tests `fhir_write_test.go:TestFHIRRoleUpdateExisting` / `TestFHIRRoleUpdateAsCreate` / `TestFHIRRoleUpdateAsCreateReservesID` / `TestFHIRRoleIfMatchIsAtomicCompareAndSwap` (`-race`) / `TestFHIRRoleUpdateResourceTypeIDMismatch` / `TestMemoryRepositoryUpdateAndDelete` (R4+R5) | — | Update-as-create matches the FHIR and HAPI default; a PUT after a DELETE resurrects (`TestFHIRRoleUpdateResurrectsDeleted`); two concurrent writers with the same valid If-Match cannot both commit (one 200, one 412) |
| patch | http.html#patch | MET | `fhir_write.go:handlePatch`; JSON Patch RFC 6902 (`fhir_jsonpatch.go:applyRFC6902`, full op set), applied to the current version with numbers decoded `UseNumber` so untouched FHIR `decimal`/`integer64` lexicals round-trip byte-for-byte (`decodeJSONPreservingNumbers`), re-validated through the create gate, version bumped (200), `If-Match` enforced as an atomic CAS at the write; 415 on wrong content type, 422 on a non-applying patch or a result that fails validation; audited as `fhir.patch` (`auditPatch`); tests `fhir_write_test.go:TestFHIRRolePatchJSONPatch` / `TestFHIRRolePatchValidatesResult` / `TestApplyRFC6902` / `TestApplyRFC6902PreservesNumericLexicals` | — | FHIRPath Patch (a `Parameters` body) is out of scope and documented (`cli-server.md`, `fhir.md`); matches the client row which is JSON-Patch only |
| delete | http.html#delete | MET | `fhir_write.go:handleDelete` over `Repository.Delete`; appends a deletion version (read → 410 Gone via `readLocked`/`writeRepoError`, prior versions vread-able, history shows the DELETE entry); idempotent (200 existing / 204 absent, never 404, per HAPI); `If-Match` enforced as an atomic CAS inside the repository write lock (`Delete(..., expectedVersion)`); test `fhir_write_test.go:TestFHIRRoleDeleteSequence` (delete→read-410→vread-prior-200→history-shows-delete→idempotent re-delete, R4+R5) | — | |
| Conditional update / patch / delete | http.html#cond-update etc. | PARTIAL | `fhir_write.go:handleConditionalUpdate`/`handleConditionalPatch`/`handleConditionalDelete`; resolve `[type]?[search]` through `Repository.Search` (`resolveConditional`): the 0/1/many decision uses `Bundle.total` (`searchsetBundleTotal`), not the page's entry count, so a paged searchset with `total > len(entry)` resolves as "many" (412), never a wrong single write; 0/1/many → create-or-noop/apply/412; tests `fhir_write_test.go:TestFHIRRoleConditionalUpdate` (+ multi-match) / `TestFHIRRoleConditionalDelete` / `TestFHIRRoleConditionalPatch` / `TestFHIRRoleConditionalWriteCountsBundleTotal` | S | Resolution is exactly as selective as the configured Repository's search; the dev `MemoryRepository` resolves an `_id` criterion (else all-of-type), so against it a conditional write is well-defined for `_id` — a production Repository with full search resolves any criteria. Multiple-match delete is a 412 (refine), not a bulk delete |
| create | http.html#create | MET | `fhir_handlers.go:handleCreate`; server-mints id, ignores client id (`fhir_repository.go:createLocked`); versioned `Location` `[base]/[type]/[id]/_history/[vid]` (`createdLocation`) | — | Inbound resource gated by release validator; error-severity issues → 422; client `idFromLocation` strips the `_history` suffix (`TestFHIRRoleClientCreateResolvesVersionedLocation`) |
| Conditional create (If-None-Exist) | http.html#ccreate | NOT-MET | fails closed: `handleCreate` and `validateTransactionWrites` answer `400` not-supported `OperationOutcome`, nothing persisted (`TestFHIRRoleConditionalCreateFailsClosed`; client interop `TestFHIRRoleClientConditionalCreateSurfacesTypedError`) | S | Matching semantics deferred to the search work; the client-sent header is answered honestly now — the silent-duplicate interop sharp edge is closed |
| search-type (GET `[type]?`) | http.html#search | PARTIAL | `fhir_handlers.go:handleSearch` forwards raw params; `MemoryRepository.Search` matches `_id` only, else all-of-type | M | Handler seam is correct; depth is the Repository's. Unrecognised params ignored, not rejected — documented |
| Search via POST `_search` | http.html#search | NOT-MET | GET-only on the type route | S | |
| search-system | http.html#search | NOT-MET | — | S | |
| `_include` / `_revinclude` / chaining server-side | search.html | NOT-MET | `MemoryRepository` ignores them; no SearchParameter registry | L | Conformance statement assigns these to the server/Repository; nothing implements them yet |
| `_summary` / `_elements` response shaping | search.html#summary | NOT-MET | no `_summary` handling in `server/` despite `fhir.MarshalSummary` existing | S | Cheap win: the library machinery already exists |
| Paging (Bundle.link next/prev, `_count`) | http.html#paging | NOT-MET | `MemoryRepository.Search` returns all matches, no links | M | history-instance honours `_count` as a cap (row above); link-based paging itself remains absent |
| transaction (POST [base]) | http.html#transaction | MET | `fhir_handlers.go:handleTransaction`; atomic staging-copy apply under write lock (`fhir_repository.go:Transaction`); per-entry create validation (`validateTransactionWrites`); response entries carry versioned `response.location` + `response.etag` (`createdEntryResponse`) | — | Transaction entries limited to POST/GET verbs in the dev repository |
| batch (POST [base]) | http.html#transaction (batch) | NOT-MET | explicitly rejected: "processes a transaction Bundle only" (`fhir_handlers.go` bundle-type check) | M | Batch is independent-entry semantics — simpler than the transaction already shipped |
| Versioning / ETag / Last-Modified emission | http.html#concurrency | MET | version store in `server/fhir_repository.go` (every create stamps `meta.versionId`/`meta.lastUpdated`); `fhir_handlers.go:setVersionHeaders` (ETag `W/"versionId"` + Last-Modified on read/vread/create) and an atomic If-Match compare-and-swap inside the repository write lock (`ifMatchVersion` → `checkExpectedVersion`, 412 on a stale or unsatisfiable version); tests `TestFHIRRoleReadAndCreateEmitVersionHeaders` / `TestFHIRRoleIfMatchPrecondition` / `TestFHIRRoleIfMatchIsAtomicCompareAndSwap` (`-race`) | — | The precondition is atomic with the write, so concurrent writers with the same valid If-Match cannot both commit |
| Prefer header handling (return=minimal etc.) | http.html#ops | NOT-MET | not read; create always returns the resource | S | |
| OperationOutcome on every error path | HAPI server error model | MET | `fhir_handlers.go:writeError`/`writeOutcome`; auth 401 also a release OperationOutcome (`role_fhir.go:155`) | — | PHI-safe messages via `sanitizeRepoMessage` |
| Resource coverage breadth | HAPI: all resource types | PARTIAL | workflow set only (`fhir_handlers.go:isWorkflowResourceType`, 8 types); others → 404/whitelist reject | M | Deliberate conformance subset; widening is config, not architecture |
| Extended operations (`$validate`, `$everything`) | HAPI plain server operations | PARTIAL | `$validate`: `fhir_handlers.go:handleValidate` (POST `[type]/$validate`, same release validator as create, OperationOutcome result, nothing persisted; advertised in the CapabilityStatement); test `TestFHIRRoleValidateOperation` | M | `$everything` and custom operations remain absent |
| Two releases per process | HAPI multi-tenant/versioned servers | MET | one release per role; mount two roles on `/fhir/r4` + `/fhir/r5` (`cli-server.md`) | — | Documented pattern rather than a flag |
| Pluggable storage backend | HAPI JPA/plain provider seam | MET | `server/fhir_repository.go:Repository` interface; `MemoryRepository` dev default | — | Concurrency-safe contract; transaction atomicity proven against concurrent writes |
| Auth on the role | HAPI server interceptors | MET | `server/auth_middleware.go` via `role_fhir.go:authMiddleware`; non-loopback bind guarded | — | |
| CLI `radx serve fhir` | (operational expectation) | MET | `cmd/radx/internal/command/serve.go:ServeFHIRCmd` (loopback default, `--release r4\|r5`, in-memory repository, ErrInsecureBind usage-error mapping); round-trip test `serve_test.go:TestServeFHIRLoopbackRoundTrip` (create + vread + SIGINT drain) | — | Memory-only repository: a production deployment embeds the role with its own `Repository` |
| HAPI interop CI leg (server side) | (project plan) | NOT-MET | issue #114 shipped the client-side leg (`interop:fhir-hapi`: go-radx client vs pinned HAPI container); no foreign client drives the go-radx server role in CI | M | Server-side counterpart of the now-wired client interop leg |

## Methodology

- **Date**: 2026-06-10; server-role and CLI rows updated 2026-06-11 after the wave-0 FHIR-server slice
  (version store, vread, history-instance, ETag/If-Match, server `$validate`, `radx serve fhir`) landed
  on `feat/fhir-server-versioning`; server-role update/patch/delete and their conditional forms flipped
  2026-06-16 on `feat/fhir-server-write-side` (the write side: `server/fhir_write.go`,
  `server/fhir_jsonpatch.go`, `Repository.Update`/`Delete`); the flipped rows cite their shipped evidence
  and tests directly.
- **Repository state**: main at `fdf7b54` for the original audit; `10c1174` plus the wave-0 FHIR-server
  slice for the updated rows.
- **References consulted**:
  - fhir.resources README (github.com/nazrulworld/fhir.resources), fetched 2026-06-10: release coverage
    (R5 default, R4B/STU3 sub-packages, R4 dropped at v7.0.0), pydantic v2 validation, JSON/YAML/XML dump,
    `summary_only`, `fhir_comments`, `add_root_validator`.
  - FHIR R5 http page (hl7.org/fhir/R5/http.html), fetched 2026-06-10: the normative interaction table,
    conditional interactions, concurrency, Prefer, paging, and search result parameters used as row anchors.
  - HAPI FHIR docs via context7 (/hapifhir/hapi-fhir): IGenericClient fluent surface (read, create, update,
    delete, search, history, transaction, patch, extended operations), BundleBuilder conditional entries,
    server operation set.
  - samply/golang-fhir-models README (github.com/samply/golang-fhir-models), fetched 2026-06-10: R4-only,
    generated marshal/unmarshal, required-binding enums, multi-CodeSystem ValueSet TODO.
- **go-radx evidence**: read directly from `fhir/`, `fhir/rest/`, `server/`, `cmd/radx/`, `mise.toml`, and
  `docs/conformance/fhir.md` / `cli-server.md`. Generator scope confirmed via `fhir/internal/gen` pipeline
  docs and the `gen:fhir-r4`/`gen:fhir-r5`/`gen:verify` mise tasks; vendored definition bundles under
  `fhir/internal/gen/testdata/definitions/` (R4 4.0.1 and R5 5.0.0, SHA-256 pinned).
- **Counting basis**: resource coverage counted from `RegisterFactory` calls in `fhir/r4/registry.go` (146)
  and `fhir/r5/registry.go` (158); fuzz targets counted from `func Fuzz` declarations (5 total).
- **Not verified**: the HL7-validator CI gate and the regeneration gate were taken from the conformance
  statement and mise task definitions, not re-run (no tests were executed for this audit). fhir.resources
  exact resource counts per release were not enumerated; both libraries claim full-spec coverage and the
  go-radx registry counts are consistent with the official release resource lists. HAPI server behaviour was
  taken from its documentation, not from a live server. The client-side HAPI interop rows were updated
  2026-06-13 when issue #114 wired the `interop:fhir-hapi` CI leg.
