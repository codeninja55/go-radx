# DIMSE feature-parity matrix

Parity of `dimse/` against the pynetdicom 3.0.4 documented public surface (the project's declared parity
floor, PRD §6.2): association/ACSE, the negotiation items, the documented Service Classes list, the
DIMSE-C and DIMSE-N primitives, the event/handler model (judged for Go-idiomatic equivalence), and
transport/TLS/timeouts. Evidence is file:symbol against main as of 2026-06-10. Tests count toward MET.

## Summary

**Counts: 93 features — 66 MET, 8 PARTIAL, 18 NOT-MET, 1 N-A.** (The NOT-MET count includes the
`qrscp` CLI-layer row cross-referenced to A6.)

The association plane is at full parity: requestor and acceptor, the complete PS3.8 FSM (Sta1-13
including release collision), ARTIM, all eight user-information negotiation items, A-ABORT/A-P-ABORT
in both directions, all four timeout knobs, TLS (1.2 floor + mTLS) on both sides, and nine PDU fuzz
targets. The DIMSE-C plane is at full parity, including C-CANCEL as SCU and as SCP for C-FIND,
C-GET, and C-MOVE.

The gap is now service-class breadth rather than the DIMSE-N plane foundation. pynetdicom documents
23 service classes; go-radx ships 7 of them (Verification, Storage, Q/R, Worklist, and now
both-role MPPS and Storage Commitment). The DIMSE-N foundation is complete for these: the N-GET and
N-DELETE primitives exist as both SCU and SCP, the `Server` routes all six DIMSE-N command fields to
interface-segregated N-handler hooks (the DIMSE-N SCP dispatch substrate), and the MPPS SCP
(N-CREATE/N-SET) and the Storage Commitment SCP (N-ACTION plus same-association N-EVENT-REPORT) now
plug into those hooks. What remains on the DIMSE-N plane is UPS, pynetdicom's flagship N-service, plus
the MPPS Retrieve/Notification SOP classes and the separate-association Storage Commitment report
leg. This matches the conformance statement's declared v1 deferral list, so most NOT-MET rows are
deliberate scope, not accidental gaps.

Top gaps by size:

1. Unified Procedure Step (push/pull/watch/event/query) — absent at every layer (size L). The DIMSE-N
   SCP dispatch substrate and the N-GET/N-DELETE primitives it needs now exist; the UPS service logic
   does not.
2. The MPPS Retrieve (N-GET) and Notification (N-EVENT-REPORT) SOP classes — the core MPPS SCP
   (N-CREATE/N-SET) now ships both roles; these two adjacent MPPS SOP classes are not yet admitted
   (size M).
3. Print Management — absent (size L).
4. Notification-event surface (pynetdicom's ~17 `evt.EVT_*` monitoring events; no logging/hook
   system is wired) (size M).
5. The remaining Q/R-family service classes (Hanging Protocol, Color Palette, Implant Template,
   Defined Procedure, Protocol Approval, Inventory) — each size M, all sharing the existing C-FIND/
   C-GET/C-MOVE machinery once their models are admitted. Each still needs its own non-image Storage
   SOP class for the retrieve payload, which is why they remain NOT-MET despite the generic machinery.

The Composite Instance Root / instance- and frame-level retrieve models, previously a top gap, are
now MET: the Patient/Study Only, Composite Instance Root Retrieve (IMAGE + FRAME), and Composite
Instance Retrieve Without Bulk Data models are admitted by the SCU preflight and SCP validation,
with per-model level handling (PS3.4 C.6.3, C.6.5, C.6.6). UPS remains NOT-MET (deferred to a later
increment).

## Association and ACSE

| Feature | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Request association (SCU) | `AE.associate()` | MET | `AE.Associate` `dimse/association.go:231`; `dimse/association_test.go` | - | Context-driven; per-call `AssociateOption`s |
| Accept association (SCP) | `AE.start_server()` | MET | `NewServer`/`ListenAndServe` `dimse/server.go:206,230` | - | Blocking under ctx; `Shutdown` `server.go:416` |
| Release | `Association.release()` | MET | `Association.Release` `dimse/association.go:423` | - | |
| Abort (send A-ABORT) | `Association.abort()` | MET | `Association.Abort` `dimse/association.go:443` | - | |
| A-ABORT / A-P-ABORT reception | `evt.EVT_ABORTED` | MET | `AbortedError` mapping `dimse/association.go:534-540`; AA1-AA8 `dimse/dul/action.go:45-49` | - | Surfaced as typed `AssociationError` (user vs provider source) |
| A-ASSOCIATE-RJ handling | association docs | MET | `AssociationRejected` w/ source+reason `dimse/association.go:529-532` | - | |
| Full PS3.8 state machine (Sta1-13) | FSM docs, `EVT_FSM_TRANSITION` | MET | `dimse/dul/state.go:11-23`; `dimse/dul/statemachine_test.go` | - | Includes release-collision Sta9-12 |
| ARTIM timer | ACSE timeout docs | MET | `dimse/dul/artim.go:12-65`; `dimse/dul/artim_test.go` | - | |
| Max PDU length negotiation | `maximum_pdu_size` | MET | `WithMaxPDULength` `dimse/ae.go:63`; `MaxPDULength.SendCap` `dimse/presentation.go:19` | - | Unlimited (0) handled, never a literal allocation size |
| Presentation-context negotiation | `add_requested_context` | MET | `NewPresentationContext` `dimse/presentation.go:84`; `acse.NegotiateAcceptor` `dimse/acse/negotiate.go:32` | - | Per-context result reasons `ContextResult` `presentation.go:34-40` |
| Custom/arbitrary abstract syntaxes | private SOP class support | MET | `NewPresentationContext` takes any `dicom.SOPClassUID` `dimse/presentation.go:84` | - | SCP supported-context list is caller-supplied |
| Preset context lists | `StoragePresentationContexts` etc. | MET | `VerificationContexts`/`StorageContexts`/`QueryRetrieveContexts`/`BasicWorklistContexts`/`ModalityPerformedContexts` `dimse/presets.go:99-145` | - | Curated radiology-first 37-class storage set vs pynetdicom's all-storage list; extensible per row above |
| Implementation class UID / version name | implementation identity | MET | `WithImplementationClassUID`/`WithImplementationVersionName` `dimse/ae.go:69,75`; read-back `association.go:403,413` | - | |
| Require called AE title | `require_called_aet` | MET | `WithRequireCalledAETitle` `dimse/server.go:73` | - | |
| Require calling AE titles | `require_calling_aet` | MET | `WithRequireCallingAETitles` `dimse/server.go:80` | - | Plus `WithAssociationAuthorizer` `server.go:134` (beyond pynetdicom) |
| Max concurrent associations | `maximum_associations` | MET | `WithMaxAssociations` `dimse/server.go:61` | - | |
| AE title validation | `AE(ae_title=...)` validation | MET | `AETitle` type `dimse/aetitle.go`; `dimse/aetitle_test.go` | - | |

## Extended negotiation

| Feature | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| SCP/SCU role selection | `build_role`, `SCP_SCU_RoleSelectionNegotiation` | MET | `WithRoleSelection` `dimse/association.go:216`; `acse.NegotiateRoles` `dimse/acse/negotiate.go:109`; `dimse/role_selection_test.go` | - | Acceptor never grants a role it lacks; observable via `NegotiatedRoles` `association.go:393` |
| Async operations window | `AsynchronousOperationsWindowNegotiation` | MET | `WithAsyncOps` `dimse/negotiation.go:65`; `NegotiatedAsyncOps` `:121` | - | Acceptor truthfully echoes (1,1); concurrency comes from goroutines, not the async-ops window |
| SOP class extended negotiation | `SOPClassExtendedNegotiation` | MET | `WithExtendedNegotiation` `dimse/negotiation.go:89`; read-back `:147` | - | Application-information blobs carried verbatim |
| SOP class common extended | `SOPClassCommonExtendedNegotiation` | MET | `WithCommonExtendedNegotiation` `dimse/negotiation.go:102`; read-back `:157` | - | |
| User identity, types 1-5 | `UserIdentityNegotiation` | MET | `WithUserIdentity` `dimse/negotiation.go:76`; `UserIdentity` `:27` | - | Username, username+passcode, Kerberos, SAML, JWT |
| User identity server-side verification | `evt.EVT_USER_ID` handler | MET | `WithAuthenticator` `dimse/server.go:149,592` | - | Callback gets identity + peer addr, returns response or rejection |
| User identity positive response read-back | `UserIdentityNegotiation` response | MET | `Association.UserIdentityResponse` `dimse/negotiation.go:135` | - | |
| Acceptor hooks for async-ops / SOP-extended responses | `evt.EVT_ASYNC_OPS`, `evt.EVT_SOP_EXTENDED`, `evt.EVT_SOP_COMMON` | PARTIAL | Fixed acceptor policy: (1,1) echo and verbatim blob echo (`docs/conformance/dimse.md:136-142`) | S | No per-association user hook to compute the acceptor's negotiation response |

## Service classes

pynetdicom's Service Classes index (3.0.4) documents 23 service classes. Status below is for the
service as a whole; the role split is in the notes.

| Service class | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Verification | `service_classes/verification...` | MET | SCU `Association.Echo` `dimse/echo.go:29`; SCP `EchoHandler` `dimse/handler.go:33` | - | Both roles; interop-tested (`dimse/integration/interop_test.go`) |
| Storage | `service_classes/storage_service_class` | MET | SCU `Association.Store` `dimse/store.go:77`; SCP `StoreHandler` `dimse/handler.go:42` | - | Curated 37-class radiology preset `dimse/presets.go:44`; arbitrary classes via custom contexts |
| Query/Retrieve (Patient/Study Root FIND/MOVE/GET) | `service_classes/query_retrieve...` | MET | SCU `Find`/`Move`/`Get` `dimse/find.go:97`, `dimse/move.go:33`, `dimse/get.go:38`; SCP `FindHandler`/`MoveHandler`/`GetHandler` `dimse/handler.go:59,75,95` | - | C-GET uses same-association role selection (`get.go:373`); sub-op counts `association.go:145`. The additional information models (Patient/Study Only, Composite Instance Root Retrieve, Composite Instance Retrieve Without Bulk Data) are admitted alongside the core Patient/Study Root models with per-model level validation — see the Composite Instance Root row below |
| Q/R Composite Instance Root retrieve (instance/frame level) | Q/R service class, retrieve models | MET | Models admitted by `moveModels`/`getModels` and the SCP `isMoveModel`/`isGetModel` guards (`dimse/qr_models.go`, `dimse/move.go`, `dimse/get.go`, `dimse/handler.go:411-445`); IMAGE + FRAME levels (`QueryLevelFrame` `dimse/find.go`) with per-model level validation (`validateModelLevel` `dimse/qr_models.go`); loopback `dimse/qr_models_test.go` | - | Composite Instance Root Retrieve MOVE (`1.2.840.10008.5.1.4.1.2.4.2`) and GET (`.4.3`) at IMAGE and FRAME levels (PS3.4 C.6.5); also admits Patient/Study Only (PS3.4 C.6.3) and Composite Instance Retrieve Without Bulk Data GET (PS3.4 C.6.6). SCU preflight + SCP validation reuse the existing C-FIND/C-GET/C-MOVE machinery; frame-selection attributes (Simple/Calculated Frame List, Time Range) are caller-supplied identifier elements |
| Basic Worklist Management (MWL) | `service_classes/basic_worklist_service_class` | MET | SCU `Association.FindWorklist` `dimse/find_worklist.go:37`; SCP via `findModels` incl. MWL `dimse/find.go:245-249` | - | Flat model, Q/R level suppressed (PS3.4 K.6.1.2.1); dcm4chee live leg skips pending archive MWL config |
| Modality Performed Procedure Step | `service_classes/modality_performed_procedure_step` | MET | SCU `MPPS.Create`/`MPPS.Set` `dimse/mpps.go:81,169`; SCP `MPPSProvider` (`NCreateHandler`/`NSetHandler`) `dimse/mpps_scp.go`; `dimse/mpps_test.go`, `dimse/mpps_scp_test.go`; interop `dimse/integration/mpps_interop_test.go` | - | Core N-CREATE/N-SET both roles; SCP enforces mandatory attrs + IN PROGRESS→final transition (pynetdicom MPPS SCP parity). MPPS Retrieve (N-GET) and Notification (N-EVENT-REPORT) SOP classes still absent (NOT-MET top-gap A2) |
| Storage Commitment (Push Model) | `service_classes/storage_commitment` | MET | SCU `StorageCommitment.Request` `dimse/stgcommit.go:170`; separate-association `CommitmentReceiver.ServeConn` `:365`; SCP `StorageCommitmentProvider` (`NActionHandler`+`NActionReporter`) `dimse/stgcommit_scp.go`; `dimse/stgcommit_test.go`, `dimse/stgcommit_scp_test.go` | - | Both roles. SCP commits per a `CommitmentDecider` hook, then reports success/partial-failure via same-association N-EVENT-REPORT; failed instances surface as typed failure `stgcommit.go:118`. Separate-association report leg (SCP opening a new association) deferred |
| Unified Procedure Step (push/pull/watch/event/query) | `service_classes/unified_procedure_step...` | NOT-MET | No UPS symbol anywhere in `dimse/` (grep: UPS/UnifiedProcedureStep zero hits) | L | Needs N-CREATE/N-SET/N-GET/N-ACTION/N-EVENT-REPORT both roles + UPS C-FIND |
| Print Management | `service_classes/print_management...` | NOT-MET | No print symbols in `dimse/` | L | Declared out of scope (`docs/conformance/dimse.md:144-148`) |
| Display System Management | `service_classes/display_system_service_class` | NOT-MET | N-GET primitive now exists (`dimse/nget.go`, `NGetHandler` `dimse/nhandler.go:62`); the Display System SOP class registration and device-config model are not shipped as a service | M | No longer blocked on N-GET — works mechanically over a custom N-GET context; not a preset/declared service |
| RT Machine Verification | `service_classes/rt_machine...` | NOT-MET | No RT symbols in `dimse/` | L | Declared out of scope |
| Relevant Patient Information Query | `service_classes/relevant_patient...` | NOT-MET | Model not in `findModels` `dimse/find.go:245` | S | C-FIND machinery exists; admit model + flat-query handling |
| Substance Administration Query | `service_classes/substance_admin...` | NOT-MET | Model not in `findModels` | S | Same shape as above |
| Hanging Protocol Q/R | `service_classes/hanging_protocol...` | NOT-MET | Models not in `findModels`/`moveModels`/`getModels` | M | Q/R machinery reusable; also needs the storage SOP class for retrieve payloads |
| Color Palette Q/R | `service_classes/color_palette...` | NOT-MET | As above | M | |
| Implant Template Q/R | `service_classes/implant_template...` | NOT-MET | As above | M | |
| Defined Procedure Protocol Q/R | `service_classes/defined_procedure...` | NOT-MET | As above | M | |
| Protocol Approval Q/R | `service_classes/protocol_approval...` | NOT-MET | As above | M | |
| Inventory Q/R | `service_classes/inventory...` | NOT-MET | As above | M | New in pynetdicom 3.x |
| Non-Patient Object Storage | `service_classes/non_patient_service_class` | PARTIAL | Generic C-STORE SCP validates affected-vs-negotiated only (`validateStoreContext` `dimse/handler.go:329`); classes addable via `NewPresentationContext` | S | Works mechanically with custom contexts; not preset or declared in conformance |
| Instance Availability Notification | `service_classes/instance_availability...` | NOT-MET | No generic N-CREATE entry point (`MPPS.Create` is service-scoped, `dimse/mpps.go:81`) | M | |
| Media Creation Management | `service_classes/media_creation` | NOT-MET | No media-creation symbols | L | Needs N-CREATE/N-ACTION/N-GET both roles |
| Application Event Logging | `service_classes/application_event` | NOT-MET | No symbols | M | N-ACTION-based |
| Storage Management | `service_classes/storage_management...` | NOT-MET | No symbols | L | New in pynetdicom 3.x (inventory-driven) |

## DIMSE-C primitives

| Feature | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| C-ECHO (SCU + SCP) | `send_c_echo`, `evt.EVT_C_ECHO` | MET | `dimse/echo.go:29`; `serveEcho` `dimse/handler.go:130` | - | |
| C-STORE (SCU + SCP) | `send_c_store`, `evt.EVT_C_STORE` | MET | `dimse/store.go:77`; `serveStoreMessage` `dimse/handler.go:212` | - | |
| C-STORE priority | `send_c_store(priority=...)` | MET | `WithStorePriority` `dimse/store.go:35`; `Priority` `dimse/command.go:109-117` | - | |
| C-STORE move-originator fields | `originator_aet`, `originator_id` | MET | `WithMoveOriginator` `dimse/store.go:55`; propagated by move SCP `dimse/move_scp.go:148-150` | - | |
| C-FIND (SCU + SCP), streaming pending responses | `send_c_find` iterator, `evt.EVT_C_FIND` yield | MET | `Find` streaming `dimse/find.go:97`; SCP iterator drain `dimse/find_scp.go` | - | Go iterator/callback model judged equivalent to pynetdicom's generator yield |
| C-GET (SCU + SCP) with same-association sub-ops | `send_c_get` + storage handlers | MET | `Get` `dimse/get.go:38` (requires granted Storage SCP role `:373`); SCP `dimse/get_scp.go` | - | Partial failure surfaces as Warning/Failure, never Success |
| C-MOVE (SCU + SCP) with destination resolution | `send_c_move`, `evt.EVT_C_MOVE` (yield destination) | MET | `Move` `dimse/move.go:33`; `WithMoveDestinations` `dimse/server.go:92`; sub-op store loop `dimse/move_scp.go:148` | - | Destination map is the Go equivalent of the handler-yielded (addr, port) |
| C-CANCEL send (SCU) | `Association.send_c_cancel` | MET | `Association.Cancel` `dimse/find.go:351`; auto-cancel on ctx `find.go:291` | - | |
| C-CANCEL honored by C-FIND SCP | `evt.is_cancelled` | MET | Cancel watcher `dimse/find_scp.go:59-93,126-150`; 0xFE00 terminal RSP | - | |
| C-CANCEL honored by C-GET SCP | `evt.is_cancelled` | MET | Interleaved C-CANCEL check `dimse/get_scp.go:97-99,196` | - | Terminal Cancel status carries accumulated sub-op counts |
| C-CANCEL honored by C-MOVE SCP | `evt.is_cancelled` | MET | C-FIND cancel watcher reused in the sub-operation loop `dimse/move_scp.go:120,136-144`; terminal `StatusMoveCancel` w/ accumulated counts; `TestServeMoveTerminalCancelOnInboundCancel` `dimse/move_scp_test.go:686` | - | PS3.4 C.4.2.3 |
| C-CANCEL outside an in-flight operation ignored | PS3.7 §9.3.2.3 behavior | MET | Silent drop at top-level dispatch `dimse/dispatch.go:229-237` | - | |
| Message ID management | `msg_id` parameters | MET | `nextMessageID` `dimse/association.go:95`; `WithStoreMessageID` `dimse/store.go:44` | - | |

## DIMSE-N primitives

| Feature | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| N-CREATE SCU | `send_n_create` | PARTIAL | `MPPS.Create` `dimse/mpps.go:81` (MPPS-scoped); command field `dimse/command.go:63` | S | No generic any-SOP-class primitive; SCP echo of assigned SOP Instance UID handled |
| N-SET SCU | `send_n_set` | PARTIAL | `MPPS.Set` `dimse/mpps.go:169`; command field `command.go:74` | S | Service-scoped; Requested (not Affected) UID pair used correctly |
| N-ACTION SCU | `send_n_action` | PARTIAL | `StorageCommitment.Request` `dimse/stgcommit.go:170`; command field `command.go:86` | S | Storage-Commitment-scoped (Action Type ID 1) |
| N-EVENT-REPORT receive (role-inverted acceptor) | storage commitment examples | PARTIAL | `CommitmentReceiver.ServeConn`/`serveReport` `dimse/stgcommit.go:365,402` | M | Purpose-built for Storage Commitment only; not a general N-EVENT-REPORT SCP |
| N-EVENT-REPORT send | `send_n_event_report` | PARTIAL | SCP-side same-association send via `NReportSender`/`NActionReporter` `dimse/nhandler.go`, `dimse/ndispatch.go` (`newNReportSender`), driven by the Storage Commitment SCP `dimse/stgcommit_scp.go` | M | Storage-Commitment-scoped (SCP reports a commitment result on the N-ACTION association); not yet a general any-SOP-class `send_n_event_report` primitive. Still needed for MPPS Notification and UPS watch |
| N-GET (SCU + SCP) | `send_n_get`, `evt.EVT_N_GET` | MET | SCU `Association.NGet` `dimse/nget.go:43`; command field `CommandNGetRQ`/`CommandNGetRSP` `dimse/command.go:112,118` w/ Attribute Identifier List (0000,1005) `command.go`; SCP `NGetHandler` `dimse/nhandler.go:62` via `serveNGetMessage` `dimse/ndispatch.go:43`; loopback `dimse/ndispatch_test.go` | - | SCU + SCP both roles; status against ServiceClassGeneral (PS3.7 Annex C) |
| N-DELETE (SCU + SCP) | `send_n_delete`, `evt.EVT_N_DELETE` | MET | SCU `Association.NDelete` `dimse/ndelete.go:24`; command field `CommandNDeleteRQ`/`CommandNDeleteRSP` `command.go`; SCP `NDeleteHandler` `dimse/nhandler.go:80` via `serveNDeleteMessage` `dimse/ndispatch.go:74`; loopback `dimse/ndispatch_test.go` | - | SCU + SCP both roles |
| DIMSE-N SCP dispatch in `Server` | `evt.EVT_N_*` handlers | MET | `dispatchMessage` routes all six N-service command fields `dimse/dispatch.go`; N-handler interfaces `NGetHandler`/`NDeleteHandler`/`NCreateHandler`/`NSetHandler`/`NActionHandler`/`NEventReportHandler` `dimse/nhandler.go:62-118`; serve fns `dimse/ndispatch.go`; capability-checked refusal `refuseUnsupportedN` `ndispatch.go` | - | Substrate complete: all six N-services routed with interface-segregated handler hooks and StatusSOPClassNotSupported refusal. N-GET/N-DELETE fully served; the MPPS SCP (N-CREATE/N-SET) and Storage Commitment SCP (N-ACTION plus same-association N-EVENT-REPORT via the `NActionReporter`/`NReportSender` extension) now plug in; the UPS application logic is deferred to later waves |

## Event and handler model

pynetdicom uses a dynamic `evt`/handler-binding model; go-radx uses static Go handler interfaces and
functional options. Equivalence is judged on capability, not shape.

| Feature | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| Service intervention handlers (C-services) | `evt.EVT_C_ECHO/STORE/FIND/GET/MOVE` | MET | `EchoHandler`/`StoreHandler`/`FindHandler`/`MoveHandler`/`GetHandler` `dimse/handler.go:33-113`; capability-checked dispatch `dimse/dispatch.go` | - | Missing capability refused with a proper status, not a crash |
| Per-event context info (calling/called AE, msg ID, SOP class) | `event.assoc`, `event.request` attrs | MET | `OpInfo` `dimse/handler.go:17` threaded into every handler call | - | |
| User identity intervention | `evt.EVT_USER_ID` | MET | `WithAuthenticator` `dimse/server.go:149` | - | Counted once here; also rowed under negotiation |
| Negotiation intervention (async ops / SOP extended / common) | `evt.EVT_ASYNC_OPS`, `EVT_SOP_EXTENDED`, `EVT_SOP_COMMON` | PARTIAL | Fixed acceptor policy (see negotiation table) | S | |
| Notification events (~17: requested/accepted/established/ rejected/released/aborted, conn open/close, PDU/DIMSE/ACSE sent/recv, FSM transition) | `evt` notification events | NOT-MET | No event/hook system; no logger wired (`dimse/server.go:383` comment) | M | SCU side surfaces outcomes as typed errors; SCP side has no observability hooks at all |
| Runtime handler bind/unbind | `assoc.bind()`/`unbind()` | N-A | - | - | Python-idiom dynamic dispatch; Go's static interfaces are the deliberate equivalent |
| Handler exception safety (bad handler does not kill server) | event docs | MET | Refusal paths `dimse/dispatch.go:246+`; per-conn serve isolation `dimse/server.go:353` | - | |

## Transport, TLS, and timeouts

| Feature | pynetdicom anchor | Status | go-radx evidence | Size | Notes |
|---|---|---|---|---|---|
| TLS for requestor (client) | `tls_args` / `ssl_context` | MET | `WithTLS` `dimse/ae.go:79-101`; dial path `association.go:308`; `dimse/tls_test.go` | - | TLS 1.2 floor enforced; verification on by default |
| TLS for acceptor (server) | `ServerSocket` ssl_context | MET | Same `WithTLS` applied to listener (`dimse/ae.go:90` doc + `server.go`) | - | mTLS via `ClientAuth`+`ClientCAs` on the caller's config |
| ACSE timeout | `AE.acse_timeout` | MET | `WithACSETimeout` `dimse/ae.go:42`; `acseContext` `association.go:363` | - | |
| DIMSE timeout | `AE.dimse_timeout` | MET | `WithDIMSETimeout` `dimse/ae.go:47`; `dimseContext` `association.go:292` | - | |
| Network timeout | `AE.network_timeout` | MET | `WithNetworkTimeout` `dimse/ae.go:52` | - | |
| TCP connection timeout | `AE.connection_timeout` | MET | `WithConnectionTimeout` `dimse/ae.go:57` | - | |
| Bind address / port selection | `start_server(("addr", port))` | MET | `ListenAndServe(ctx, addr)` `dimse/server.go:230`; `Addr()` `:473` | - | |
| Graceful shutdown | `ServerSocket.shutdown()` | MET | `Server.Shutdown` `dimse/server.go:416` | - | Context cancellation also tears down |
| Hostile-input PDU hardening | (pynetdicom has no fuzz suite documented) | MET | 9 fuzz targets across `dimse/pdu/associate_fuzz_test.go`, `negotiation_fuzz_test.go`, `role_selection_fuzz_test.go`, `fuzz_test.go`; bounded reader `dimse/pdu/bounded_reader.go` | - | go-radx exceeds the floor here |

## Bundled applications (cross-reference to the CLI audit, A6)

Protocol capability only is judged here; CLI flag/behavior parity is A6's scope.

| pynetdicom app | go-radx equivalent | Status | Notes |
|---|---|---|---|
| `echoscu` | `radx echo` (`cmd/radx/internal/command/echo.go`) | MET | Library capability complete |
| `findscu` | `radx find` (`command/find.go`) | MET | |
| `getscu` | `radx get` (`command/get.go`) | MET | |
| `movescu` | `radx move` (`command/move.go`) | MET | |
| `storescu` | `radx store` (`command/store.go`) | MET | |
| `storescp` | `radx scp` (`command/scp.go:47` — `EchoHandler` + `StoreHandler`) | MET | |
| `qrscp` | none — `radx scp` does not wire `FindHandler`/`MoveHandler`/`GetHandler` | NOT-MET (CLI); library SCP capability exists | Size M at the CLI layer; defer to A6 |

## Methodology

Sources fetched (2026-06-10):

- pynetdicom docs via ctx7 (`/pydicom/pynetdicom`, High reputation, benchmark 86): service-class SOP
  lists (MPPS, Media Creation, Display System, Basic Worklist, Storage Commitment), extended
  negotiation and user-identity examples, A-ASSOCIATE user-information primitives, intervention
  events, ACSE method surface.
- pynetdicom 3.0.4 stable site (direct fetch): `service_classes/index.html` (the authoritative
  23-entry Service Classes list) and `user/events_types.html` (the full `evt.EVT_*` inventory:
  17 notification + 4 association-intervention + 11 service-intervention events).
- go-radx evidence: direct reads/greps of `dimse/`, `dimse/acse/`, `dimse/dul/`, `dimse/pdu/`,
  `dimse/integration/`, `cmd/radx/internal/command/`, and `docs/conformance/dimse.md` (claims
  cross-checked against code; the conformance scaffold's tables matched the code in every case
  spot-checked).

Unverified areas:

- pynetdicom's per-service status-code catalogues and per-app CLI flags were not compared
  item-by-item; status-code parity is judged at the typed-`Status` level
  (`dimse/status.go`), not per-code.
- The dcm4chee/Orthanc live interop legs were not executed for this audit (no tests run, per
  audit constraints); their skip conditions are taken from the conformance statement and test
  source.
- pynetdicom's Inventory Q/R and Storage Management pages were not fetched individually; their
  presence and N-service requirements are taken from the index and release notes.
