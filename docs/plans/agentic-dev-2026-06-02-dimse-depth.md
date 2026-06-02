# DIMSE depth (M3) implementation plan

> **For agentic workers:** REQUIRED: Use agentic-dev:subagent-driven-development (if subagents available) or
> agentic-dev:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the query/retrieve and procedure-step depth on top of the COMPLETE M2 DIMSE foundation: **C-FIND, C-GET,
C-MOVE** (the multi-response query/retrieve services, SCU + SCP), **Modality Worklist** (MWL — a C-FIND
specialisation, SCU + SCP), **MPPS** (Modality Performed Procedure Step, N-CREATE / N-SET, SCU only per v1), and **Storage
Commitment** (Push Model, N-ACTION / N-EVENT-REPORT, SCU only). Each service is built as both SCU and SCP for the
DIMSE-C services (C-FIND/C-GET/C-MOVE/MWL), and as SCU only for the DIMSE-N services (MPPS, Storage Commitment), exactly
the scope `docs/reference/dimse.md` commits ("DIMSE-C as both SCU and SCP … DIMSE-N **SCU only** in v1"). The whole leg
conforms exactly to the committed public API in `docs/reference/dimse.md` — the iterator surface
(`iter.Seq2[Status, *dicom.DataSet]`), the typed `Status`/`ServiceClass`/`QueryLevel` model, the `Handler`
intervention-method model, and the `MPPS`/`StorageCommitment` typed operation facades — and closes the two
query/retrieve defects the M2 plan deferred (DIMSE-015: the query level silently dropped; DIMSE-016: sub-operation
C-STOREs sent with `MessageID: 0` and a single-PDU read, which miscounted failures and hung against compliant peers).
The new services are interop-gated: C-FIND and C-MOVE against both Orthanc and dcm4chee-arc; MWL, MPPS, and Storage
Commitment against dcm4chee-arc (the archive whose worklist/procedure-step/commitment services those gates exercise).

**Architecture:** M3 adds command fields and service operations to the EXISTING `dimse` machinery built in M2; it
rebuilds nothing. The three stacked layers stay intact (PRD §8.2, dimse.md "Overview of the layers"): `dimse/pdu`
(PDU/PDV codec), `dimse/dul` (the PS3.8 Table 9-10 state machine, `Conn`, and the shared `DriveInbound` inbound path),
and `dimse/acse` (association/presentation-context negotiation). Every M3 service rides the **same** message layer
(`message.go`'s `fragmentMessage`/`messageReassembler`/`sendMessage`/`receiveMessage`), the **same** `CommandSet`
Implicit-VR codec (`command.go`), the **same** `Association` SCU facade and `Server`/`dispatch.go` SCP dispatch, and
the **same** `DriveInbound` hardened inbound read. The work is therefore additive and tightly scoped:

1. **New command primitives.** C-FIND-RQ/RSP, C-GET-RQ/RSP, C-MOVE-RQ/RSP, C-CANCEL-RQ (composite), and the normalized
   N-CREATE/N-SET/N-GET (MPPS) and N-ACTION/N-EVENT-REPORT (Storage Commitment) command fields, with the group-0000
   elements the normalized services add (`AffectedSOPInstanceUID` is present; `RequestedSOPClassUID (0000,0003)`,
   `RequestedSOPInstanceUID (0000,1001)`, `ActionTypeID (0000,1008)`, `EventTypeID (0000,1002)`,
   `NumberOfRemainingSubOperations (0000,1020)` and the three sibling sub-operation counts, and `ErrorComment` as a
   skip-tolerated optional). These extend the `command.go` group-0000 model (the M2 `commandVR` dictionary already
   reserves `tagMoveDestination` VR `AE` for the DIMSE-007 regression; M3 adds the rest).
2. **The multi-response iterator (SCU) and generator (SCP).** C-FIND/C-GET/C-MOVE yield zero or more `Pending`
   (`0xFF00`/`0xFF01`) responses then one terminal status; the SCU surface is `iter.Seq2[Status, *dicom.DataSet]` with
   `Association.LastError()` for the trailing transport fault, and the SCP `Handler.Find`/`Get`/`Move` methods are
   generators of the same shape that the dispatch loop drains into one Pending RSP per yield plus a terminal RSP.
3. **Sub-operation association handling.** C-GET drives C-STORE sub-operations back to the requestor **on the same
   association** (which requires SCP-role negotiation for the Storage SOP Classes); C-MOVE drives them from the SCP to a
   **third destination AE** on a separate outbound association. Each sub-operation gets a **real, distinct Message ID**
   and each C-STORE-RSP is read through the same reassembly loop (DIMSE-016), and the SCU tracks the
   Remaining/Completed/Failed/Warning sub-operation counts the RSPs carry.

`MWL` is a C-FIND on the Modality Worklist Information Model SOP Class with no query-level element (the worklist model
has no Q/R level), so it reuses the C-FIND machinery with a different default model and the level write suppressed. The
new presentation-context presets (`QueryRetrieveContexts`, `BasicWorklistContexts`, `ModalityPerformedContexts`,
`StorageCommitmentContexts`) join `presets.go`. Every leg ends green: unit tests with `-race`, a clean `golangci-lint`,
and — where the reference PACS supports the service — a passing interop gate behind `//go:build interop`.

**Tech Stack:** Go 1.26.x, module `github.com/codeninja55/go-radx`; standard library for the wire and iteration
(`encoding/binary`, `iter`, `context`, `sync`); `go.uber.org/zap` for structured, no-PHI diagnostics; `testcontainers-go`
for the Orthanc / dcm4chee-arc interop gates (already a dependency). No CGo in any M3 leg — the four
uncompressed/deflated transfer syntaxes are the only ones the queries and retrievals negotiate and exercise end-to-end
(dimse.md "Default transfer syntaxes"); a C-GET/C-MOVE of a compressed instance is transport-only (the bytes are not
decoded by go-radx during transfer), so no codec is required.

---

## How to use this plan

Read this section once before starting; it states the conventions every task follows. They are the M2 conventions
(the walking-skeleton plan), carried forward unchanged so M3 does not drift from the foundation it extends.

**Test-first, always.** Each task is a strict TDD cycle: write the failing test, run it and confirm it fails for the
right reason, write the minimal implementation, run it and confirm it passes, then commit. Do not write implementation
before its test. See `agentic-dev:test-driven-development`.

**Canonical names are mandatory.** Use the exact Go identifiers fixed in `UBIQUITOUS_LANGUAGE.md` and `dimse.md`:
`Find`, `Get`, `Move`, `Cancel`, `QueryLevel` (`QueryLevelPatient`/`Study`/`Series`/`Image`), `QueryOption`
(`WithQueryPriority`, `WithQueryModel`), `LastError`, `MPPS` (`Create`/`Set`), `StorageCommitment` (`Request`),
`CommitmentHandler`, `StorageCommitmentResult`, `FailedSOPInstance`, `RoleSelection`, `WithStoreHandler`,
`WithCommitmentHandler`, `Handler.Find`/`Get`/`Move`, `FindHandler`. Reuse the `dicom`-owned types
(`dicom.SOPClassUID`, `dicom.SOPInstanceUID`, `dicom.UID`, `dicom.ReferencedSOPInstance`, `dicom.TransferSyntax`,
`*dicom.DataSet`) — never redeclare them, and never name a DICOM referenced instance a `Reference` (that noun is
FHIR's, per the cross-standard collision table). A presentation context is `dimse.PresentationContext`, never a bare
`Context` colliding with `context.Context`.

**Honour the committed API.** The signatures in `docs/reference/dimse.md` are the contract. The iterator return type is
exactly `iter.Seq2[Status, *dicom.DataSet]`; the parameter names in the doc are illustrative but the receiver, value
parameters, and return type are pinned. Do not invent new public API. Where a genuine gap appears, stop and surface it
(see "Open questions").

**Status is data, errors are faults.** A terminal `Failure`/`Cancel` `Status` is **data** the caller inspects, yielded
as the final element of the iterator with a `nil` dataset; it is **not** a Go `error`. A transport/protocol/association
fault that ends iteration before a clean terminal status is an `error`, retrievable via `Association.LastError()`
immediately after the `range` loop. Never launder a partial-success retrieve into success; the `0xB000` "sub-operations
complete, one or more failures" warning is reported faithfully (PRD §9.2 fail-closed; dimse.md "the streaming query
contract" guarantee 2).

**The query level is always written.** Every C-FIND/C-GET/C-MOVE writes the requested `QueryLevel` into Query/Retrieve
Level `(0008,0052)` of the identifier before sending — the DIMSE-015 fix the prototype omitted. MWL is the one exception
(no level element); the C-FIND path suppresses the level write when the model is Modality Worklist.

**Sub-operations get real Message IDs.** A C-GET/C-MOVE SCP that drives sub-operation C-STOREs allocates a **distinct,
non-zero Message ID** per sub-operation and reads each C-STORE-RSP through the same `receiveMessage` reassembly loop —
the DIMSE-016 fix (the prototype used `MessageID: 0` and read exactly one P-DATA-TF, miscounting failures and hanging).
A per-association monotonic Message ID allocator is added (the M2 `Echo`/`Store` paths used the fixed
`echoMessageID`/`storeMessageID` constant because a single operation needs no allocator; concurrent and chained
operations do).

**Bounds-check every length; diagnostics carry no PHI.** Carried forward from M2: every length read from the wire is
validated before allocation (the reassembler's `checkCap`, the PDV bounded reader); errors and logs name the concept —
a tag by keyword plus `(gggg,eeee)`, a SOP/transfer-syntax UID by registered name, a `Status` by name and class, an AE
title — never a patient value (PRD §9.1, §9.3). A query identifier is a `*dicom.DataSet` carrying matching keys (which
may be PHI); it is never logged. `OpInfo` is the structured no-PHI SCP diagnostic context, extended only with protocol
identifiers.

**No global mutable state; goroutine discipline.** Every knob is per-`AE`/per-`Association`/per-`Server` (PRD §9.4).
C-MOVE's outbound sub-operation associations and C-GET's same-association sub-operation stores are driven from tracked
goroutines or the dispatch loop's own goroutine — no fire-and-forget (PRD §9.4). An `Association` is **not** safe for
concurrent queries (`LastError()` is per-association); concurrency is one association per goroutine.

**Commit conventionally and often.** Each commit message follows `<type>(<pkg>): <description>` (for example
`feat(dimse): add C-FIND-RQ/RSP command primitives`). Source and its test commit together; fixtures and tooling commit
separately (the project Atomic Commit Strategy). This whole deliverable — the plan document — commits as a single
`docs:` commit on the `m3-dimse-depth` worktree.

**Defect traceability.** M3 closes the two query/retrieve defects M2 explicitly deferred:

| Defect | What the prototype did wrong | Where M3 fixes it |
|--------|------------------------------|-------------------|
| DIMSE-015 | C-FIND/C-GET/C-MOVE accepted a query level argument and silently dropped it (never wrote `(0008,0052)`) | Increment 2 (C-FIND SCU identifier build), regression test |
| DIMSE-016 | sub-operation C-STOREs used `MessageID: 0` and read exactly one P-DATA-TF, miscounting failures and hanging against compliant peers | Increment 5 (C-GET) and Increment 4 (C-MOVE) sub-operation handling, regression test |

A defect is not closed until its named regression test passes.

**The reference doc IS the spec.** Read the cited `docs/reference/dimse.md` section before implementing each increment.
The conformance gate lives in dimse.md "Conformance scope and limits". Note: `docs/conformance/dimse.md` does **not**
exist (only `dicom.md`, `fhir.md`, `hl7v2.md` are in `docs/conformance/`); the formal DIMSE conformance statement is
authored in M8 per PRD §13. `pynetdicom` (`dimse_messages.py`, `service_class.py`, `status.py`,
`presentation.py`/`_presentation_data.py`) and `dcm4che` are the parity references — **verify any claim about them
against the actual source; do not trust from memory.**

**Existing symbols M3 extends (do not rebuild).** The plan annotates each increment with the M2 symbols it touches.
The load-bearing ones:

- `command.go`: `CommandField` (add the C-FIND/C-GET/C-MOVE/C-CANCEL/N-service fields), `CommandSet` (add the
  normalized-service fields), `commandVR`/the tag vars (add the new group-0000 tags), `elements()`/`applyElement()`
  (emit/parse the new elements in tag order with group length last).
- `message.go`: `fragmentMessage`, `sendMessage`, `receiveMessage`, `messageReassembler`, `newMessageReassembler`/
  `newMessageReassemblerFunc` — reused verbatim; M3 sends/receives more command-and-dataset messages through them.
- `status.go`: `statusTable` (wire up the FIND/MOVE/GET/Worklist/ProcedureStep/StorageCommitment tables — the enums
  `ServiceClassFind`/`Move`/`Get`/`Worklist`/`ProcedureStep`/`StorageCommitment` already exist), and the named
  Pending/Warning constants (`StatusFindPending`, the `0xB000` sub-op-failure warning, `0xA801` move-destination-unknown).
- `association.go`: `Association` (add the Message ID allocator and the `LastError` field), the `requestor.Conn()`/
  `Machine()` accessors, `sendCap()`, `dimseContext()`.
- `dispatch.go`: `dispatchMessage` (route the new command fields), `readInbound` (already drains the reassembler);
  `handler.go`: the `Handler` union (add `Find`/`Get`/`Move`), `FindHandler`, `serveEchoCommand`/`serveStoreMessage`
  pattern (mirror it for the new services).
- `presets.go`: `contextsFor` and the SOP-class lists (add Q/R, Worklist, MPPS, Storage Commitment presets); the
  `verificationSOPClass` constant pattern.
- `ae.go`: `AEOption`s (`WithStoreHandler`, `WithCommitmentHandler`, `WithRoleSelection` plumbing through
  `AssociateOption`).
- `dul.DriveInbound`: the shared inbound path — every new inbound read routes through it (the architect's
  DUL-ownership decision); never reimplement the abort-send/clean-close distinction.

---

## Increment overview (dependency-ordered)

C-FIND leads because it is the foundation the other query/retrieve services and MWL build on (the multi-response
iterator, the level-write fix, the per-class status tables). C-MOVE precedes C-GET because C-MOVE's sub-operations go to
a third AE over a fresh association (simpler to reason about and the easier interop gate via dcm4chee `ExportStudy` /
Orthanc `StoreToModality`), whereas C-GET's sub-operations come back on the same association and require SCP-role
negotiation. The DIMSE-N services follow, smallest first.

- **Increment 0 — Status tables, presets, and the Message ID allocator (foundation).** Wire up the FIND/MOVE/GET/
  Worklist/ProcedureStep/StorageCommitment status tables and the Pending/sub-op-failure-warning/move-destination-unknown
  constants in `status.go`; add the `QueryRetrieveContexts`/`BasicWorklistContexts`/`ModalityPerformedContexts`/
  `StorageCommitmentContexts` presets and their SOP-class constants in `presets.go`; add the per-`Association` monotonic
  Message ID allocator. No service operations yet — just the shared scaffolding every later increment consumes. *Fully
  expanded into bite-sized TDD tasks below.*
- **Increment 1 — C-FIND-RQ/RSP command primitives + codec.** Add the `CommandCFindRQ`/`RSP` and `CommandCCancelRQ`
  command fields and the C-FIND group-0000 element handling (the identifier-bearing RQ, the multi-response RSP carrying
  only a status), with encode/decode round-trip tests. Extends `command.go`. *Outlined.*
- **Increment 2 — C-FIND SCU iterator + the level-write fix.** `Association.Find(ctx, query, level, opts...)
  iter.Seq2[Status, *dicom.DataSet]`: write `(0008,0052)` (DIMSE-015), select the Q/R model context, send the
  C-FIND-RQ, then loop reading RSPs — yield each `Pending` with its identifier, stop on the terminal status, send a
  C-CANCEL on early `break`/ctx-cancel, and set `LastError` on a transport fault. Extends `association.go`,
  `message.go`. *Outlined.*
- **Increment 3 — C-FIND SCP handler + dispatch + interop.** Add `Handler.Find` / the `FindHandler` interface and the
  dispatch generator that drains the handler's `iter.Seq2` into one Pending RSP per yield plus a terminal RSP; the
  in-process SCU↔SCP round-trip; the Orthanc + dcm4chee C-FIND interop gate. Extends `handler.go`, `dispatch.go`.
  *Outlined.*
- **Increment 4 — C-MOVE (SCU iterator + SCP with third-AE sub-operations) + interop.** `Association.Move(ctx, query,
  level, dest, opts...)` SCU iterator tracking the four sub-operation counts; the SCP `Handler.Move` generator that
  opens an outbound association to the destination AE and drives sub-operation C-STOREs with **distinct Message IDs**
  (DIMSE-016); the C-MOVE interop gate (dcm4chee `ExportStudy`, Orthanc `StoreToModality` to a go-radx receiving SCP).
  Extends `association.go`, `handler.go`, `dispatch.go`, the M2 `Store` sub-operation path. *Outlined.*
- **Increment 5 — C-GET (SCU iterator + same-association sub-operation stores + SCP role) + interop.**
  `Association.Get(ctx, query, level, opts...)` SCU iterator that receives sub-operation C-STOREs on the same
  association into the AE's `WithStoreHandler` sink (requiring SCP-role negotiation via `RoleSelection`); the SCP
  `Handler.Get` generator that drives sub-operation C-STOREs back over the inbound association. Extends `ae.go`
  (`WithStoreHandler`, `WithRoleSelection`), `association.go`, `handler.go`, `dispatch.go`. *Outlined.*
- **Increment 6 — Modality Worklist (MWL) C-FIND SCU + SCP + interop.** The Worklist Information Model as a C-FIND
  specialisation: suppress the `(0008,0052)` level write, default the model SOP Class to Modality Worklist, the
  `BasicWorklistContexts` preset, and the worklist status table; SCU via `Find` with `WithQueryModel`, SCP via a
  worklist `FindHandler`; the dcm4chee MWL interop gate. Extends the Increment 2/3 C-FIND paths, `presets.go`,
  `status.go`. *Outlined.*
- **Increment 7 — MPPS N-services (N-CREATE / N-SET, SCU) + interop.** The N-CREATE/N-SET command primitives (the
  normalized command fields and their group-0000 elements: `RequestedSOPClassUID`, `RequestedSOPInstanceUID`,
  `AffectedSOPInstanceUID`), the `MPPS` typed facade (`Create` returning the created SOP Instance UID; `Set` updating
  to COMPLETED/DISCONTINUED), the `ModalityPerformedContexts` preset, the ProcedureStep status table; the dcm4chee MPPS
  interop gate. Extends `command.go`, `presets.go`, `status.go`, a new `mpps.go`. *Outlined.*
- **Increment 8 — Storage Commitment (N-ACTION / N-EVENT-REPORT, SCU) + interop.** The N-ACTION/N-EVENT-REPORT command
  primitives (`ActionTypeID`, `EventTypeID`), the `StorageCommitment` typed facade (`Request` sending the N-ACTION,
  the same-association receipt of the N-EVENT-REPORT delivered to the AE's `WithCommitmentHandler`), the
  `StorageCommitmentResult`/`FailedSOPInstance` model (reusing `dicom.ReferencedSOPInstance`), the
  `StorageCommitmentContexts` preset, the StorageCommitment status table; the dcm4chee Storage Commitment interop gate.
  Extends `command.go`, `ae.go` (`WithCommitmentHandler`), `presets.go`, `status.go`, a new `stgcommit.go`. *Outlined.*

Increment 0 is fully expanded into bite-sized TDD tasks below; Increments 1–8 are outlined (Goal, Files, Key tests,
Reference-doc section, the existing symbols extended, ports-vs-extends note, and the verification gate) and are expanded
into bite-sized TDD tasks when reached — exactly as the M2 walking-skeleton plan fully expanded Increment 1 and outlined
the rest.

---

## Increment 0 — Status tables, presets, and the Message ID allocator (fully expanded)

**Scope:** The shared, dependency-free scaffolding every M3 service consumes, built before any service operation so the
later increments add behaviour against a stable foundation. Three pieces: (a) the per-service-class status
categorisation tables (FIND/MOVE/GET/Worklist/ProcedureStep/StorageCommitment) wired into the existing `statusTable`
switch, with the named Pending/Warning/Failure constants the services return; (b) the four new presentation-context
presets and their SOP-class constants; (c) the per-`Association` monotonic Message ID allocator that the
sub-operation-bearing services (C-GET/C-MOVE) and the chained N-services require, replacing the M2 fixed-ID constants
on the multi-operation paths. This increment delivers no service operation and changes no wire behaviour of C-ECHO /
C-STORE; it only adds tables, constants, presets, and one allocator, each behind its own test.

**Reference-doc section:** dimse.md "Typed status" (the per-class tables and the selected named codes table),
"Presets" (the preset counts: `QueryRetrieveContexts` 6, `BasicWorklistContexts` 1, `ModalityPerformedContexts` 1,
`StorageCommitmentContexts` 1), "Negotiation primitives". Parity reference: `pynetdicom/status.py`
(`QR_FIND_SERVICE_CLASS_STATUS`, `QR_MOVE_SERVICE_CLASS_STATUS`, `QR_GET_SERVICE_CLASS_STATUS`,
`MODALITY_WORKLIST_SERVICE_CLASS_STATUS`, `PROCEDURE_STEP_STATUS`, `STORAGE_COMMITMENT_SERVICE_CLASS_STATUS`) and
`pynetdicom/presentation.py` (`QueryRetrievePresentationContexts`, `BasicWorklistManagementPresentationContexts`,
`ModalityPerformedProcedureStepPresentationContexts`, `StorageCommitmentPresentationContexts`) — **verify the exact SOP
Class UIDs and status code categorisations against the pynetdicom source before encoding them; do not trust from
memory.**

**Existing symbols this increment extends:** `status.go` — `statusTable(sc ServiceClass)` (the switch currently maps
only Storage and Verification; add the six query/retrieve and N-service tables), the named-constant block, `statusEntry`,
`codeToCategory` (already handles `0xFF00`/`0xFF01` Pending, `0xA000–0xAFFF`/`0xC000–0xCFFF` Failure, `0xB000–0xBFFF`
Warning — verify the ranges cover the new codes before adding table entries). `presets.go` — `contextsFor`,
`VerificationContexts`/`StorageContexts` pattern, the `verificationSOPClass`/`validatedStorageSOPClasses` constant
pattern. `association.go` — the `Association` struct (add the allocator field and accessor); the M2
`echoMessageID`/`storeMessageID` constants stay for the single-operation paths.

---

### Task 0.1: Query/Retrieve FIND/MOVE/GET status tables and Pending constants

**Files:**
- Modify: `dimse/status.go`
- Test: `dimse/status_test.go` (append)

The C-FIND/C-GET/C-MOVE services categorise the same numeric code differently from Storage: `0xFF00`/`0xFF01` are
Pending (a continuing match or sub-operation), `0xB000` is the C-GET/C-MOVE "sub-operations complete, one or more
failures" **Warning** (not a failure), `0xA801` is the C-MOVE "Move Destination Unknown" Failure, and the `0xC000–0xCFFF`
band is Failure. This task adds the three Q/R tables and the named constants, wiring them into `statusTable`.

- [ ] **Step 1: Write the failing test (append to `status_test.go`)**

```go
func TestQueryRetrieveStatusCategories(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want StatusCategory
	}{
		{"find pending", StatusFindPending, StatusCategoryPending},
		{"find pending optional-keys", NewStatus(0xFF01, ServiceClassFind), StatusCategoryPending},
		{"find success", StatusFindSuccess, StatusCategorySuccess},
		{"move sub-ops failure warning", NewStatus(0xB000, ServiceClassMove), StatusCategoryWarning},
		{"move destination unknown", StatusMoveDestinationUnknown, StatusCategoryFailure},
		{"get sub-ops failure warning", NewStatus(0xB000, ServiceClassGet), StatusCategoryWarning},
		{"cancel", NewStatus(0xFE00, ServiceClassMove), StatusCategoryCancel},
	}
	for _, c := range cases {
		if got := c.s.Category(); got != c.want {
			t.Errorf("%s: Category() = %s, want %s", c.name, got, c.want)
		}
	}
	// A pending status must NEVER read as success, the laundering bug the typed model prevents.
	if StatusFindPending.IsSuccess() {
		t.Error("StatusFindPending.IsSuccess() must be false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/ -run TestQueryRetrieveStatusCategories -v`
Expected: FAIL — `undefined: StatusFindPending`, `StatusFindSuccess`, `StatusMoveDestinationUnknown`.

- [ ] **Step 3: Write minimal implementation**

Add the named constants to the `var (...)` block in `status.go`, the three tables, and the `statusTable` switch arms
(`ServiceClassFind`/`Move`/`Get` → their tables). Each table merges the general table over its specific codes (the
`storageStatusTable` pattern). Confirm `codeToCategory` already returns Pending for `0xFF00`/`0xFF01` (it does) so an
unlisted Pending code still categorises. **Before writing the code categorisations, verify each against
`pynetdicom/status.py`.**

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/ -run TestQueryRetrieveStatusCategories -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/status.go dimse/status_test.go
git commit -m "feat(dimse): add C-FIND/C-MOVE/C-GET status tables and pending constants"
```

---

### Task 0.2: Worklist, ProcedureStep, and StorageCommitment status tables

**Files:**
- Modify: `dimse/status.go`
- Test: `dimse/status_test.go` (append)

The N-services and MWL each have a status table: Modality Worklist (FIND-shaped, `0xFF00`/`0xFF01` Pending), Procedure
Step (MPPS — the general table plus the procedure-step-specific failures), and Storage Commitment. This task adds the
three tables and the named constants (`StatusWorklistPending`, and the MPPS/StgCommit success/failure names the typed
facades return), wiring them into `statusTable`.

- [ ] **Step 1: Write the failing test (append to `status_test.go`)**

```go
func TestNServiceStatusCategories(t *testing.T) {
	cases := []struct {
		s    Status
		want StatusCategory
	}{
		{StatusWorklistPending, StatusCategoryPending},
		{NewStatus(0x0000, ServiceClassWorklist), StatusCategorySuccess},
		{NewStatus(0x0000, ServiceClassProcedureStep), StatusCategorySuccess},
		{NewStatus(0x0110, ServiceClassProcedureStep), StatusCategoryFailure}, // Processing Failure (general)
		{NewStatus(0x0000, ServiceClassStorageCommitment), StatusCategorySuccess},
	}
	for _, c := range cases {
		if got := c.s.Category(); got != c.want {
			t.Errorf("%s.Category() = %s, want %s", c.s, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/ -run TestNServiceStatusCategories -v`
Expected: FAIL — `undefined: StatusWorklistPending`.

- [ ] **Step 3: Write minimal implementation**

Add the three tables and the `statusTable` switch arms. Reuse the general table as the base for ProcedureStep and
StorageCommitment (verify against `pynetdicom/status.py` that they extend `GENERAL_STATUS`). Add `StatusWorklistPending`
and any MPPS/StgCommit names the facades will return.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/ -run TestNServiceStatusCategories -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/status.go dimse/status_test.go
git commit -m "feat(dimse): add worklist, procedure-step, and storage-commitment status tables"
```

---

### Task 0.3: Query/Retrieve, Worklist, MPPS, and Storage Commitment presets

**Files:**
- Modify: `dimse/presets.go`
- Test: `dimse/presets_test.go` (append)

The four new presets bundle the presentation contexts each service needs. `QueryRetrieveContexts` returns 6 contexts
(the Patient Root and Study Root C-FIND/C-MOVE/C-GET Information Model SOP Classes); `BasicWorklistContexts` returns 1
(Modality Worklist Information Model — FIND); `ModalityPerformedContexts` returns 1 (MPPS SOP Class);
`StorageCommitmentContexts` returns 1 (Storage Commitment Push Model SOP Class). Each returns a fresh slice via the
existing `contextsFor` helper.

- [ ] **Step 1: Write the failing test (append to `presets_test.go`)**

```go
func TestNewPresetContextCounts(t *testing.T) {
	cases := []struct {
		name string
		got  []PresentationContext
		want int
	}{
		{"QueryRetrieveContexts", QueryRetrieveContexts(), 6},
		{"BasicWorklistContexts", BasicWorklistContexts(), 1},
		{"ModalityPerformedContexts", ModalityPerformedContexts(), 1},
		{"StorageCommitmentContexts", StorageCommitmentContexts(), 1},
	}
	for _, c := range cases {
		if len(c.got) != c.want {
			t.Errorf("%s returned %d contexts, want %d", c.name, len(c.got), c.want)
		}
		// Fresh slice each call (no shared backing array), matching the existing presets.
		if &c.got[0] == &c.name { /* compile guard; real check below */ }
	}
	// IDs are odd and ascending (PS3.8 9.3.2.2), and the proposed transfer syntaxes default.
	for _, pc := range QueryRetrieveContexts() {
		if pc.ID%2 == 0 {
			t.Errorf("QueryRetrieveContexts context ID %d is even, must be odd", pc.ID)
		}
	}
}
```

(Drop the `&c.got[0]` compile guard line when writing; it is a placeholder reminder to assert freshness as the existing
preset tests do.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/ -run TestNewPresetContextCounts -v`
Expected: FAIL — `undefined: QueryRetrieveContexts`, etc.

- [ ] **Step 3: Write minimal implementation**

Add the SOP-class constants (Patient Root / Study Root FIND, MOVE, GET model UIDs; Modality Worklist Information Model;
MPPS; Storage Commitment Push Model) and the four preset functions, each calling `contextsFor`. **Verify every SOP Class
UID against `pynetdicom/presentation.py` and the DICOM registry before encoding it.**

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/ -run TestNewPresetContextCounts -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/presets.go dimse/presets_test.go
git commit -m "feat(dimse): add query/retrieve, worklist, MPPS, and storage-commitment presets"
```

---

### Task 0.4: Per-association monotonic Message ID allocator

**Files:**
- Modify: `dimse/association.go`
- Test: `dimse/association_test.go` (append)

C-GET/C-MOVE drive multiple sub-operation C-STOREs and the chained N-services issue several requests, so a fixed
Message ID no longer suffices: each in-flight operation needs a **distinct, non-zero** ID (the DIMSE-016 fix that
miscounted failures with `MessageID: 0`). This task adds a per-`Association` monotonic allocator returning `1, 2, 3, …`
(wrapping at the 16-bit boundary, skipping 0). It is called by the new services in later increments; the M2
`Echo`/`Store` single-operation paths keep their fixed constants.

- [ ] **Step 1: Write the failing test (append to `association_test.go`)**

```go
func TestAssociationMessageIDAllocator(t *testing.T) {
	a := &Association{}
	first := a.nextMessageID()
	second := a.nextMessageID()
	if first == 0 || second == 0 {
		t.Fatalf("message IDs must be non-zero, got %d and %d", first, second)
	}
	if second != first+1 {
		t.Errorf("expected monotonic IDs, got %d then %d", first, second)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./dimse/ -run TestAssociationMessageIDAllocator -v`
Expected: FAIL — `undefined: (*Association).nextMessageID`.

- [ ] **Step 3: Write minimal implementation**

Add a `nextMsgID uint16` field to `Association` (guarded by the existing `a.mu` or `atomic` — a single goroutine drives
the association, but the field is touched from sub-operation send paths, so document the guard) and an unexported
`nextMessageID()` that increments and returns, skipping 0 on wrap. Do not change the `Associate` constructor's other
fields.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./dimse/ -run TestAssociationMessageIDAllocator -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dimse/association.go dimse/association_test.go
git commit -m "feat(dimse): add per-association monotonic message-id allocator"
```

---

**Increment 0 verification gate:** `go test -race ./dimse/...` is green (the new tables, presets, and allocator pass,
and the existing C-ECHO/C-STORE tests are unchanged); `golangci-lint run ./dimse/...` is clean; the new SOP Class UIDs
and status categorisations have each been verified against the pynetdicom source (not asserted from memory). No interop
gate — this increment adds no wire behaviour.

---

## Increment 1 — C-FIND-RQ/RSP command primitives + codec

**Goal:** Add the C-FIND command fields and group-0000 element handling so the message layer can build and parse a
C-FIND-RQ (an identifier-bearing request) and a C-FIND-RSP (a multi-response reply carrying a status and, on a Pending,
a matching-identifier dataset). C-FIND-RQ carries Affected SOP Class UID, Message ID, Priority, and CommandDataSetType
present (the identifier follows as the dataset); C-FIND-RSP carries the command field, Message ID Being Responded To,
the status, and CommandDataSetType present on Pending / not-present on the terminal. Also add `CommandCCancelRQ` (the
C-CANCEL-RQ command field) for the cancel path Increment 2 uses. This increment is pure command-set codec — no service
operation, no wire I/O — proven by encode/decode round-trip and tag-order tests.

**Files:** modify `dimse/command.go` (add `CommandCFindRQ`/`CommandCFindRSP`/`CommandCCancelRQ` to the `CommandField`
const block; the C-FIND elements reuse the existing tags — Affected SOP Class UID, Message ID, Priority,
CommandDataSetType, Status, Message ID Being Responded To — already in `commandVR` and `elements()`/`applyElement()`),
`dimse/command_test.go` (append round-trip tests). **Extends** `command.go`'s `CommandField`, `CommandSet`,
`elements()`, `applyElement()`, `commandVR`. No new tags are needed for C-FIND itself (it reuses the existing
group-0000 set); the test asserts a C-FIND-RQ and a C-FIND-RSP round-trip with the correct elements in increasing tag
order and Command Group Length last.

**Key tests:** `TestCFindRQRoundTrip` — a C-FIND-RQ command set encodes Affected SOP Class UID, Message ID, Priority,
and CommandDataSetType present, decodes back equal, with `(0000,0000)` group length first and elements in increasing
tag order; `TestCFindRSPPendingHasDataset` — a Pending C-FIND-RSP (`0xFF00`) declares CommandDataSetType present (a
matching identifier follows) while a terminal Success RSP declares it not-present; `TestCCancelRQEncodes` — a
C-CANCEL-RQ encodes the cancel command field and the Message ID Being Responded To.

**Reference-doc section:** dimse.md "PDU and PDV" (command-set build discipline: increasing tag order, group length
last), "C-FIND, C-GET, C-MOVE — the streaming query contract". Parity reference: `pynetdicom/dimse_messages.py`
`C_FIND_RQ` / `C_FIND_RSP` (verify the command field values `0x0020`/`0x8020` and the element set against the source).

**Verification gate:** `go test -race ./dimse/ -run 'CFind|CCancel'` green; lint clean; the round-trip and tag-order
tests pass. No interop (no wire flow yet).

---

## Increment 2 — C-FIND SCU iterator + the level-write fix (DIMSE-015)

**Goal:** `Association.Find(ctx, query *dicom.DataSet, level QueryLevel, opts ...QueryOption) iter.Seq2[Status,
*dicom.DataSet]`. The iterator: validates the association is established (typed error via the iterator's first yield, or
a documented surface for the pre-flight fault — see Open questions); selects the Q/R Information Model presentation
context (default by `level`, overridable with `WithQueryModel`); **writes the requested `QueryLevel` into `(0008,0052)`
of a copy of the query identifier before sending (DIMSE-015 — the prototype dropped it)**; sends the C-FIND-RQ command
+ identifier dataset via `sendMessage`; then loops `receiveMessage`, yielding `(Pending status, matching identifier)`
for each Pending RSP and stopping when the terminal status (Success/Failure/Cancel) arrives, yielding it last with a
`nil` dataset. Breaking the range loop early, or cancelling `ctx`, sends a **C-CANCEL-RQ** for the operation's Message
ID on the same context. A transport/protocol fault that ends iteration sets `Association.LastError()` (read immediately
after the loop) and stops the iterator.

**Files:** new `dimse/find.go` (`Find`, `QueryOption`, `WithQueryPriority`, `WithQueryModel`, `queryConfig`, the level
write, the model selection, the iterator body, the C-CANCEL on early stop), `dimse/find_test.go`; modify
`dimse/association.go` (add the `lastError error` field and `LastError()` method; the iterator sets it). **Extends**
`association.go` (`requestor.Conn()`/`Machine()`, `sendCap()`, `dimseContext()`, `nextMessageID()`),
`message.go` (`sendMessage`/`receiveMessage` reused verbatim), `command.go` (the Increment 1 C-FIND fields). Reuse the
M2 `contextForStorage` pattern for `contextForQuery(model)`.

**Key tests:** `TestFindWritesQueryLevel` (DIMSE-015 regression) — `Find` with `QueryLevelStudy` writes `"STUDY"` into
`(0008,0052)` of the sent identifier even when the caller's query omits it, asserted against a loopback SCP that
captures the received identifier; `TestFindYieldsPendingThenTerminal` — an in-process SCP that returns two Pending
matches then Success drives the iterator to yield two `(Pending, ds)` pairs and one terminal `(Success, nil)`;
`TestFindBreakSendsCancel` — `break`ing after the first match sends a C-CANCEL-RQ (asserted by the loopback SCP seeing
the cancel command field); `TestFindTransportFaultSetsLastError` — a peer that aborts mid-query ends the iterator and
`LastError()` returns the `*AbortError`/`*ProtocolError`; `TestFindOnReleasedAssociation` — `Find` on a released
association yields a single terminal failure status / sets a typed error, never panics (DIMSE-017 carried forward).

**Reference-doc section:** dimse.md "C-FIND, C-GET, C-MOVE — the streaming query contract" (the two guarantees: level
always written, terminal status faithful), "C-CANCEL", "C-FIND (study-level query) with the streaming iterator" worked
example, the `LastError` semantics paragraph. Parity reference: `pynetdicom` `AE.send_c_find` response handling.

**Verification gate:** `go test -race ./dimse/ -run Find` green; lint clean; the DIMSE-015 (`TestFindWritesQueryLevel`)
and the early-cancel and last-error regressions pass. In-process SCU↔SCP only (the real-PACS C-FIND is the Increment 3
interop gate).

---

## Increment 3 — C-FIND SCP handler + dispatch + interop

**Goal:** The SCP side of C-FIND: add `Find(ctx, query, level, info) iter.Seq2[Status, *dicom.DataSet]` to the `Handler`
union and a standalone `FindHandler` interface (interface segregation — a worklist-only SCP implements `FindHandler`
alone); add the dispatch generator that, on a received C-FIND-RQ, reads the query identifier, calls the handler's
iterator, and drains it into **one Pending C-FIND-RSP per yielded match (command + matching identifier dataset)**
followed by a **terminal C-FIND-RSP** carrying the iterator's final status (no dataset). The dispatcher honours a
C-CANCEL-RQ arriving mid-drain by stopping the handler iterator (cancelling its context) and sending a `0xFE00` Cancel
terminal RSP. Then the interop gate: the go-radx SCU runs a study-level C-FIND against Orthanc and dcm4chee, asserting
the matches; a reference SCU (or a go-radx SCU) runs C-FIND against a go-radx `FindHandler` SCP.

**Files:** modify `dimse/handler.go` (add `FindHandler`, add `Find` to `Handler`), `dimse/dispatch.go` (route
`CommandCFindRQ` to a new `serveFindMessage` generator-drain), new `dimse/find_scp.go` (`serveFindMessage`,
`refuseUnsupportedFind`), `dimse/handler_test.go` / a new `dimse/find_scp_test.go`; modify
`dimse/integration/interop_test.go` and `dimse/integration/dcm4chee_interop_test.go` (add the C-FIND gates,
`//go:build interop`). **Extends** `handler.go` (the `Handler`/`FindHandler`/`serveEchoCommand`/`serveStoreMessage`
pattern — mirror it for Find), `dispatch.go` (`dispatchMessage` switch, `acceptedTransferSyntaxResolver`,
`acceptedAbstractSyntaxResolver`, the context-validation helpers), the M2 `recordingStore` integration pattern.

**Key tests:** `TestServeFindDrainsHandlerIterator` — an in-process `FindHandler` yielding two matches then Success
produces two Pending RSPs and one terminal RSP on the wire; `TestServeFindCancelStopsHandler` — a C-CANCEL-RQ
mid-drain cancels the handler's context and sends a `0xFE00` terminal; `TestServeFindUnsupported` — a C-FIND-RQ to a
store-only handler is refused with a terminal `StatusSOPClassNotSupported` RSP, never a panic. Interop:
`TestInteropOrthancCFind` / `TestInteropDcm4cheeCFind` — the go-radx SCU C-FINDs a study previously stored (reuse the
M2 `StoreInstance`/`UploadInstance` + `HasInstanceWithSOPUID` fixtures) and asserts the study-level match;
`TestServerAnswersCFind` — a go-radx SCU C-FINDs a go-radx `FindHandler` Server.

**Reference-doc section:** dimse.md "SCP handlers and the event model" (the intervention `Find` method and the runtime
draining one Pending per yield), "C-FIND, C-GET, C-MOVE — the streaming query contract", "Conformance scope and limits"
(the C-FIND interop bullet). Parity reference: `pynetdicom`'s `evt.EVT_C_FIND` handler contract (yield `(status,
identifier)` pairs).

**Verification gate:** `go test -race ./dimse/...` green; lint clean; the drain, cancel, and unsupported regressions
pass; `mise run interop:dimse` C-FIND gates green against **both** Orthanc and dcm4chee (the C-FIND acceptance gate).

---

## Increment 4 — C-MOVE (SCU iterator + SCP third-AE sub-operations) + interop

**Goal:** C-MOVE both directions. SCU: `Association.Move(ctx, query, level, dest AETitle, opts ...QueryOption)
iter.Seq2[Status, *dicom.DataSet]` — write `(0008,0052)`, select the Q/R MOVE model context, send a C-MOVE-RQ carrying
the Move Destination `(0000,0600)` (VR `AE`, the element the M2 `commandVR` already reserves), then loop the RSPs,
yielding each Pending whose RSP carries the four **sub-operation counts** (Remaining `0000,1020` / Completed `0000,1021`
/ Failed `0000,1022` / Warning `0000,1023`) and the terminal status (faithfully reporting the `0xB000` "one or more
sub-operations failed" Warning and the `0xA801` "Move Destination Unknown" Failure). SCP: `Handler.Move` resolves the
matches, opens a **separate outbound association to the destination AE**, and drives sub-operation C-STOREs to it, each
with a **distinct non-zero Message ID** read through the full `receiveMessage` reassembly loop (DIMSE-016), accumulating
the counts and reporting them in each Pending C-MOVE-RSP. The destination-AE association is opened from a tracked
goroutine, released cleanly, and its failure surfaces as a terminal `0xA801`/`0xA702` status, not a panic (PRD §9.4).

**Files:** new `dimse/move.go` (`Move` SCU iterator, the sub-operation count parsing, the C-MOVE-RQ build), new
`dimse/move_scp.go` (`serveMoveMessage`: match resolution, destination association, sub-operation store loop, count
accumulation), `dimse/move_test.go` / `dimse/move_scp_test.go`; modify `dimse/command.go` (add `CommandCMoveRSP` — the
RQ `CommandCMoveRQ` already exists — and the four sub-operation-count tags + their `commandVR` US entries +
`elements()`/`applyElement()` handling, gated by a `HasSubOpCounts` flag like `HasPriority`), `dimse/handler.go` (add
`Move` to `Handler`), `dimse/dispatch.go` (route `CommandCMoveRQ`); modify the interop tests (dcm4chee `ExportStudy` and
Orthanc `ConfigureModality`/`StoreToModality` drive a C-MOVE into a go-radx receiving SCP). **Extends** `command.go`
(`CommandField`, `CommandSet`, `commandVR`, `elements()`/`applyElement()`), the M2 `Association.Store`
(`WithMoveOriginator` already exists for sub-operation provenance), `message.go`, the `AE.Associate` SCU path (the SCP
reuses it to reach the destination AE).

**Key tests:** `TestMoveSubOperationsUseDistinctMessageIDs` (DIMSE-016 regression) — a `Handler.Move` driving three
sub-operation C-STOREs uses three distinct non-zero Message IDs and reads each RSP through the reassembly loop;
`TestMoveSCUReportsCounts` — the SCU iterator surfaces the Remaining/Completed/Failed/Warning counts from each Pending
RSP; `TestMoveTerminalWarningOnSubOpFailure` — when one sub-operation fails the terminal status is the `0xB000` Warning,
not laundered to Success (PRD §9.2); `TestMoveDestinationUnknown` — an unresolvable destination AE yields `0xA801`, not
a panic; `TestMoveWritesQueryLevel` — DIMSE-015 carried forward. Interop: `TestInteropDcm4cheeCMove` (drive dcm4chee
`ExportStudy` to a go-radx receiving SCP via `ConfigureDestinationAE`, assert the instances arrive) and the
Orthanc equivalent via `ConfigureModality`/`StoreToModality`; `TestServerAnswersCMove` (a go-radx SCU C-MOVEs from a
go-radx SCP to a third go-radx receiving SCP, all in-process).

**Reference-doc section:** dimse.md "C-FIND, C-GET, C-MOVE — the streaming query contract", "SCP handlers and the event
model" (the sub-operation Message-ID and reassembly-loop rule — the DIMSE-016 paragraph quoted verbatim), the `0xB000`
/ `0xA801` rows of the status table. Parity reference: `pynetdicom`'s `AE.send_c_move` and the `evt.EVT_C_MOVE` handler
(it yields the destination then `(status, identifier)`), and `service_class.py` `QueryRetrieveMoveServiceClass`.

**Verification gate:** `go test -race ./dimse/...` green; lint clean; the DIMSE-016, sub-op-count, terminal-warning, and
destination-unknown regressions pass; `mise run interop:dimse` C-MOVE gates green against **both** Orthanc and dcm4chee.

---

## Increment 5 — C-GET (SCU iterator + same-association sub-operation stores + SCP role) + interop

**Goal:** C-GET both directions. SCU: `Association.Get(ctx, query, level, opts ...QueryOption) iter.Seq2[Status,
*dicom.DataSet]` — the retrieve runs over **the same association**, so the requestor must have negotiated the **SCP
role** for the relevant Storage SOP Classes (via `WithRoleSelection` on `Associate`, plumbed through the
A-ASSOCIATE-RQ); received sub-operation C-STORE-RQs are delivered to the AE's `WithStoreHandler` sink, the SCU answers
each with a C-STORE-RSP, and the iterator yields the C-GET-RSP Pending statuses with their sub-operation counts and the
terminal status. SCP: `Handler.Get` resolves matches and drives sub-operation C-STOREs **back over the inbound
association** to the requestor (each a distinct Message ID, DIMSE-016), interleaved with the C-GET-RSP Pendings. This is
the most intricate increment because one association multiplexes the C-GET responses and the inbound/outbound
sub-operation C-STOREs; the reassembler and `DriveInbound` are reused, and the SCU's receive loop must distinguish a
C-GET-RSP (answer the iterator) from a C-STORE-RQ sub-operation (route to the store sink and reply).

**Files:** new `dimse/get.go` (`Get` SCU iterator with the interleaved C-STORE-RQ handling), new `dimse/get_scp.go`
(`serveGetMessage`: match resolution and same-association sub-operation stores), `dimse/get_test.go` /
`dimse/get_scp_test.go`; modify `dimse/command.go` (add `CommandCGetRQ`/`CommandCGetRSP` and reuse the sub-operation
count tags from Increment 4), `dimse/ae.go` (add `WithStoreHandler` storing the sink on `aeConfig`, and `WithRoleSelection`
as an `AssociateOption` plumbed into the A-ASSOCIATE-RQ role-selection sub-items), `dimse/association.go` (carry the
store sink and the negotiated roles), `dimse/handler.go` (add `Get` to `Handler`), `dimse/dispatch.go` (route
`CommandCGetRQ`). **Extends** the M2 `serveStoreMessage`/`serveStore` (reused to answer inbound sub-operation
C-STORE-RQs on the SCU side via the `StoreHandler` sink), `message.go`, `acse`/`pdu` role-selection negotiation (the
M2 `RoleSelection` type exists in the doc; the A-ASSOCIATE-RQ encoder gains the SCP/SCU-role sub-item — confirm whether
the M2 `pdu` associate codec already encodes it, see Open questions), `command.go`.

**Key tests:** `TestGetReceivesSubOperationStores` — an in-process C-GET against an SCP that pushes two instances back
on the same association delivers both to the registered `StoreHandler` and yields the matching sub-operation counts;
`TestGetRequiresStoreHandler` — `Get` without `WithStoreHandler` returns a typed configuration error before sending
(fail-closed, PRD §9.2); `TestGetSubOperationStoresUseDistinctMessageIDs` (DIMSE-016) — the SCP's pushed C-STOREs use
distinct Message IDs; `TestGetRoleSelectionNegotiated` — `WithRoleSelection` causes the A-ASSOCIATE-RQ to carry the
SCP-role sub-item for the Storage SOP Classes; `TestGetWritesQueryLevel` — DIMSE-015 carried forward. Interop:
`TestInteropDcm4cheeCGet` / `TestInteropOrthancCGet` — the go-radx SCU C-GETs a previously stored study and the
registered `StoreHandler` receives the instances (both archives support C-GET — confirm Orthanc C-GET is enabled in the
container config, see Open questions).

**Reference-doc section:** dimse.md "C-FIND, C-GET, C-MOVE — the streaming query contract" (the `Get` paragraph and the
`WithStoreHandler`/SCP-role requirement), "Negotiation primitives" (`RoleSelection`), "SCP handlers and the event
model" (the same-association sub-operation rule, DIMSE-016), `AE` options `WithStoreHandler`. Parity reference:
`pynetdicom`'s `AE.send_c_get` (the `on_c_store` callback and the role-selection requirement) and
`service_class.py` `QueryRetrieveGetServiceClass`.

**Verification gate:** `go test -race ./dimse/...` green; lint clean; the sub-operation-store, store-handler-required,
distinct-Message-ID, and role-selection regressions pass; `mise run interop:dimse` C-GET gates green (against the
archives whose containers enable C-GET — at minimum dcm4chee).

---

## Increment 6 — Modality Worklist (MWL) C-FIND SCU + SCP + interop

**Goal:** Modality Worklist as a C-FIND specialisation. MWL is a C-FIND on the **Modality Worklist Information Model –
FIND** SOP Class with **no Query/Retrieve Level element** (the worklist model is not levelled), so it reuses the
Increment 2/3 C-FIND machinery with two differences: the default model SOP Class is Modality Worklist (selected when the
caller passes `WithQueryModel(modalityWorklistSOPClass)` or a dedicated `Find` entry), and the `(0008,0052)` level write
is **suppressed** for the worklist model (writing a level into a worklist query is a protocol error). SCU: `Find` with
the worklist model and the `BasicWorklistContexts` preset; the worklist query identifier carries the Scheduled
Procedure Step Sequence keys. SCP: a worklist `FindHandler` (the same interface from Increment 3) that answers worklist
queries; the runtime drains its iterator into Pending RSPs exactly as C-FIND does. Interop: a go-radx SCU runs an MWL
C-FIND against dcm4chee-arc (the archive that hosts a Modality Worklist SCP) and asserts a scheduled procedure step
matches.

**Files:** modify `dimse/find.go` (suppress the level write when the model is Modality Worklist; document the
worklist-model selection), new `dimse/worklist.go` if a dedicated convenience entry is warranted (else `WithQueryModel`
suffices — decide during expansion), `dimse/worklist_test.go`; modify `dimse/dcm4chee_interop_test.go` (add the MWL
gate). **Extends** the Increment 2 `Find` SCU path, the Increment 3 `serveFindMessage` SCP path, `presets.go`
(`BasicWorklistContexts`, the Modality Worklist SOP Class constant — both added in Increment 0), `status.go` (the
Worklist table from Increment 0). Reuses the C-FIND command primitives (Increment 1) unchanged — MWL is a C-FIND-RQ/RSP
on a different abstract syntax.

**Key tests:** `TestMWLSuppressesQueryLevel` — an MWL `Find` does **not** write `(0008,0052)` into the identifier (the
inverse of the DIMSE-015 fix: a worklist query must carry no level), asserted against a loopback SCP that captures the
identifier; `TestMWLYieldsScheduledSteps` — an in-process worklist `FindHandler` yields two scheduled procedure steps
then Success; `TestMWLModelSelectsWorklistContext` — the worklist query selects the `BasicWorklistContexts` presentation
context. Interop: `TestInteropDcm4cheeMWL` — the go-radx SCU runs an MWL C-FIND against dcm4chee (after the test seeds a
scheduled procedure step via the archive's REST/HL7 ingestion, or asserts the empty-worklist Success terminal when
seeding is out of reach — decide during expansion, see Open questions).

**Reference-doc section:** dimse.md "Presets" (`BasicWorklistContexts`), "C-FIND, C-GET, C-MOVE — the streaming query
contract" (MWL is a C-FIND on a non-levelled model), "Conformance scope and limits". Parity reference: `pynetdicom`'s
`BasicWorklistManagementServiceClass` and the `ModalityWorklistInformationFind` SOP Class UID — **verify the UID and
that MWL writes no `(0008,0052)`.**

**Verification gate:** `go test -race ./dimse/...` green; lint clean; the level-suppression and yields regressions pass;
`mise run interop:dimse` MWL gate green against dcm4chee-arc (Orthanc does not host a Modality Worklist SCP by default —
the MWL gate is dcm4chee-only; confirm during expansion).

---

## Increment 7 — MPPS N-services (N-CREATE / N-SET, SCU) + interop

**Goal:** The Modality Performed Procedure Step SCU. Add the **normalized** command primitives the N-services need:
the `CommandNCreateRQ`/`RSP` and `CommandNSetRQ`/`RSP` command fields and their group-0000 elements — Affected SOP
Class UID and (for N-CREATE-RSP) Affected SOP Instance UID, Requested SOP Class UID `(0000,0003)` and Requested SOP
Instance UID `(0000,1001)` (for N-SET, which addresses an existing instance), Message ID / Message ID Being Responded
To, CommandDataSetType, and Status. Build the `MPPS` typed facade per dimse.md: `Create(ctx, attrs) (dicom.SOPInstanceUID,
Status, error)` issues N-CREATE for a new MPPS instance (status IN PROGRESS) and returns the created SOP Instance UID;
`Set(ctx, instance, attrs) (Status, error)` issues N-SET to update it (typically to COMPLETED or DISCONTINUED). Use the
`ModalityPerformedContexts` preset and the ProcedureStep status table. SCU only — there is **no** MPPS SCP (the v1
scope; do not build a server side). Interop: a go-radx MPPS SCU runs N-CREATE then N-SET against dcm4chee-arc and
asserts the procedure step is recorded.

**Files:** modify `dimse/command.go` (add the N-CREATE/N-SET command fields, the Requested SOP Class/Instance UID tags
and their `commandVR` UI entries, `elements()`/`applyElement()` handling gated by presence flags), new `dimse/mpps.go`
(`MPPS`, `Association.MPPS()`, `Create`, `Set`, the IN PROGRESS / COMPLETED attribute helpers), `dimse/mpps_test.go`;
modify `dimse/dcm4chee_interop_test.go` (add the MPPS gate). **Extends** `command.go` (`CommandField`, `CommandSet`,
`commandVR`, `elements()`/`applyElement()`), `message.go` (`sendMessage`/`receiveMessage` carry the N-CREATE/N-SET
command + attribute dataset unchanged), `association.go` (`nextMessageID()` for the chained Create-then-Set),
`presets.go`/`status.go` (Increment 0 additions). The N-services send a dataset (the MPPS attributes) exactly as
C-STORE does, so the message layer needs no change.

**Key tests:** `TestNCreateRQRoundTrip` / `TestNSetRQRoundTrip` — the normalized command sets round-trip with the
Requested SOP Class/Instance UID elements in tag order; `TestMPPSCreateReturnsInstanceUID` — an in-process N-CREATE SCP
double returns an Affected SOP Instance UID that `Create` surfaces; `TestMPPSSetUpdatesInstance` — `Set` addresses the
created instance via Requested SOP Instance UID; `TestMPPSCreateThenSetUseDistinctMessageIDs` — the chained operations
allocate distinct Message IDs. Interop: `TestInteropDcm4cheeMPPS` — N-CREATE (IN PROGRESS) then N-SET (COMPLETED)
against dcm4chee, asserting via the archive REST API that the procedure step exists (confirm dcm4chee accepts MPPS
N-CREATE on the default config — it does; verify the SOP Class UID and the container exposes the MPPS SCP).

**Reference-doc section:** dimse.md "DIMSE-N SCU: MPPS and Storage Commitment" (the `MPPS` facade signatures),
"Presets" (`ModalityPerformedContexts`), "Conformance scope and limits" (MPPS SCU only, no SCP). Parity reference:
`pynetdicom`'s `AE.send_n_create` / `send_n_set` and the MPPS SOP Class UID and `ModalityPerformedProcedureStepSOPClass`
— **verify the SOP Class UID and the N-CREATE/N-SET command field values.**

**Verification gate:** `go test -race ./dimse/...` green; lint clean; the N-CREATE/N-SET round-trip and the
create-then-set Message-ID regression pass; `mise run interop:dimse` MPPS gate green against dcm4chee-arc.

---

## Increment 8 — Storage Commitment (N-ACTION / N-EVENT-REPORT, SCU) + interop

**Goal:** The Storage Commitment Push Model SCU. Add the `CommandNActionRQ`/`RSP` and `CommandNEventReportRQ`/`RSP`
command fields and their group-0000 elements — Action Type ID `(0000,1008)` and Event Type ID `(0000,1002)`, the
Requested SOP Class/Instance UID, Affected SOP Class/Instance UID, and Status. Build the `StorageCommitment` typed
facade per dimse.md: `Request(ctx, transactionUID dicom.UID, instances []dicom.ReferencedSOPInstance) (Status, error)`
sends the N-ACTION (action type 1) listing the referenced SOP instances to commit, then — **on the same association**
(the v1 same-association receipt path) — the SCU keeps the association open and the peer's N-EVENT-REPORT result is
delivered to the AE's registered `CommitmentHandler` (via `WithCommitmentHandler`). Build the `StorageCommitmentResult`
(TransactionUID, Successful, Failed) and `FailedSOPInstance` (the `dicom.ReferencedSOPInstance` embedding plus a
`FailureReason uint16`) model, the `StorageCommitmentContexts` preset, and the StorageCommitment status table. SCU only —
**no** receiving the report on a later peer-initiated association (the deferred SCP-side path), and **no** Storage
Commitment SCP. Interop: a go-radx SCU requests commitment of a previously stored instance against dcm4chee-arc and the
`CommitmentHandler` receives the success report.

**Files:** modify `dimse/command.go` (add the N-ACTION/N-EVENT-REPORT command fields, the Action/Event Type ID tags and
their `commandVR` US entries, `elements()`/`applyElement()` handling), new `dimse/stgcommit.go` (`StorageCommitment`,
`Association.StorageCommitment()`, `Request`, the N-ACTION action-information dataset build — the Transaction UID and
the Referenced SOP Sequence — and the same-association N-EVENT-REPORT receipt that decodes the result dataset into
`StorageCommitmentResult` and answers the N-EVENT-REPORT-RQ with an N-EVENT-REPORT-RSP), new `dimse/commitment.go`
(`StorageCommitmentResult`, `FailedSOPInstance`, `CommitmentHandler`), `dimse/stgcommit_test.go`; modify `dimse/ae.go`
(`WithCommitmentHandler` storing the sink on `aeConfig`), `dimse/dcm4chee_interop_test.go` (add the Storage Commitment
gate). **Extends** `command.go`, `ae.go` (`aeConfig`, `AEOption`), `association.go` (carry the commitment sink,
`nextMessageID()`), `message.go` (the N-ACTION/N-EVENT-REPORT carry datasets unchanged), `presets.go`/`status.go`
(Increment 0). `FailedSOPInstance` reuses `dicom.ReferencedSOPInstance` — never a bare `Reference`.

**Key tests:** `TestNActionRQRoundTrip` / `TestNEventReportRoundTrip` — the command sets round-trip with the
Action/Event Type ID elements in tag order; `TestStorageCommitmentRequestSendsReferencedInstances` — `Request` builds
the N-ACTION action-information dataset carrying the Transaction UID and the Referenced SOP Sequence of the supplied
instances; `TestStorageCommitmentDeliversResultToHandler` — an in-process peer that sends an N-EVENT-REPORT-RQ on the
same association causes the registered `CommitmentHandler.Commitment` to fire with the correct
`StorageCommitmentResult` (Successful/Failed split, the Transaction UID correlated);
`TestStorageCommitmentRequiresHandler` — `Request` without `WithCommitmentHandler` returns a typed configuration error
(fail-closed). Interop: `TestInteropDcm4cheeStorageCommitment` — store an instance, request commitment, assert the
`CommitmentHandler` receives a success report (confirm dcm4chee enables the Storage Commitment SCP and sends the
N-EVENT-REPORT on the same association vs a fresh one — see Open questions, this is the most fragile interop gate).

**Reference-doc section:** dimse.md "DIMSE-N SCU: MPPS and Storage Commitment" (the `StorageCommitment` facade, the
same-association receipt caveat, the `StorageCommitmentResult`/`FailedSOPInstance` shapes), "SCP handlers and the event
model" (`CommitmentHandler`), "Presets" (`StorageCommitmentContexts`), "Conformance scope and limits" (SCU only,
same-association receipt only). Parity reference: `pynetdicom`'s `AE.send_n_action` and the Storage Commitment Push
Model SOP Class UID and transaction model — **verify the SOP Class UID, action type 1, event type IDs, and the
same-association vs separate-association receipt behaviour against the dcm4che/pynetdicom source.**

**Verification gate:** `go test -race ./dimse/...` green; lint clean; the N-ACTION/N-EVENT-REPORT round-trip, the
result-delivery, and the handler-required regressions pass; `mise run interop:dimse` Storage Commitment gate green
against dcm4chee-arc (or, if the same-association receipt proves unsupported by the dcm4chee container, the gate is
documented as a known limitation and the in-process delivery test stands as the acceptance proof — escalate per Open
question 5).

---

## Open questions and resolutions

Resolve against the committed `docs/reference/dimse.md` (and the pynetdicom/dcm4che source) before or during execution.

1. **Pre-flight fault on `Find`/`Get`/`Move` (an iterator return type cannot return an error).** The committed
   signature returns only `iter.Seq2[Status, *dicom.DataSet]`, with transport faults surfaced via
   `Association.LastError()`. A pre-flight fault (released association, no matching context, no `WithStoreHandler` for
   `Get`) has no natural channel before the loop runs. **Proposed resolution (confirm during Increment 2):** the
   iterator yields a single terminal `Failure`-category `Status` and sets `Association.LastError()` to the typed error,
   so a caller that checks `LastError()` after the loop (as the worked example does) sees the fault; the iterator never
   panics (DIMSE-017). This matches the doc's "an operation that produced no usable results" wording. Surface to the
   architect if a different surface (e.g. an `Association.Prepare` pre-check) is preferred.

2. **Role-selection encoding in the M2 `pdu` associate codec.** C-GET (Increment 5) requires the A-ASSOCIATE-RQ to
   carry SCP/SCU-role-selection sub-items (PS3.7 D.3.3.4). **Open:** confirm whether the M2 `dimse/pdu/associate.go`
   already encodes/decodes the role-selection user-information sub-item. If not, Increment 5 must add it to the `pdu`
   associate codec (a port-with-verification against the prototype, since M2's scope was Verification/Storage which need
   no role selection). Check before starting Increment 5; it may pull a small `pdu` sub-task forward.

3. **Orthanc C-GET and MWL support.** C-GET (Increment 5) and MWL (Increment 6) interop gates assume the reference PACS
   hosts the service. **Confirmed from the helpers:** Orthanc's container helper supports C-STORE/C-FIND/C-MOVE
   (`ConfigureModality`/`StoreToModality`) and QIDO; it does **not** expose a Modality Worklist SCP by default, so the
   **MWL gate is dcm4chee-only**. Orthanc C-GET requires the GET plugin / config flag — **confirm during Increment 5
   whether the existing Orthanc container enables C-GET; if not, the C-GET interop gate is dcm4chee-only** and the
   Orthanc C-GET test is omitted (not skipped silently — documented).

4. **dcm4chee MWL seeding.** The MWL interop gate (Increment 6) needs a scheduled procedure step in dcm4chee to match.
   **Open:** whether the dcm4chee container helper can seed an MWL entry (via REST, HL7 ingestion, or a UPS/MWL import
   endpoint). If seeding is impractical, the gate asserts the empty-worklist Success terminal (proving the SCU↔archive
   MWL C-FIND negotiates and completes) rather than a specific match. Decide during Increment 6 expansion; prefer
   seeding if a REST path exists.

5. **Storage Commitment same-association N-EVENT-REPORT.** dimse.md commits the v1 path as *same-association* receipt
   (the SCU holds the association open and the peer sends the N-EVENT-REPORT back on it). **Open:** real archives
   (including dcm4chee) commonly send the N-EVENT-REPORT on a **separate, later, peer-initiated** association — which is
   the explicitly-deferred SCP-side path. If the dcm4chee container does not deliver the report on the same
   association, the **Increment 8 interop gate cannot pass as specified**; the in-process delivery test then stands as
   the acceptance proof and the same-association-only limitation is documented (it already is, in dimse.md). **Escalate
   to the architect before Increment 8** — this may reduce the Storage Commitment acceptance gate to in-process plus a
   documented interop limitation.

6. **`docs/conformance/dimse.md` does not exist — confirmed.** Only `dicom.md`, `fhir.md`, `hl7v2.md` are in
   `docs/conformance/`. M3 uses the "Conformance scope and limits" section of `docs/reference/dimse.md` as the gate;
   the formal DIMSE conformance statement is M8 per PRD §13. (The task brief referenced `docs/conformance/dimse.md`;
   that file is not present — the reference doc is the authoritative spec, as the M2 plan also resolved.)

### Reviewer corrections to apply when expanding the named increment

Apply these when expanding the outlined increments, so they do not drift from the committed API and the M2 foundation:

- **Increment 1 (command primitives, verify-before-encode):** the C-FIND command field values (`0x0020`/`0x8020`) and
  the N-service command field values are NOT yet in `command.go` — verify each against `pynetdicom/dimse_messages.py`
  before adding to the `CommandField` const block. The existing block has `CommandCMoveRQ = 0x0021` (added in M2 for
  the DIMSE-007 regression); add `CommandCMoveRSP`, the C-FIND/C-GET fields, and the N-service fields alongside it.

- **Increment 4 (sub-operation counts, datatype pin):** the four C-MOVE/C-GET sub-operation count elements
  (`0000,1020`–`0000,1023`) are VR `US`; add them to `commandVR` as `dicom.VRUS` and gate their `elements()`/
  `applyElement()` emission behind a `HasSubOpCounts` flag (the `HasPriority`/`HasStatus` pattern), so an RQ never emits
  them and only a Pending/terminal RSP that carries them does. They are present on C-MOVE-RSP and C-GET-RSP, absent on
  the RQ.

- **Increment 5 (C-GET multiplexing, the load-bearing complexity):** the SCU `Get` receive loop multiplexes two PDV
  streams on one association — C-GET-RSP responses (drive the iterator) and inbound C-STORE-RQ sub-operations (route to
  the `WithStoreHandler` sink and reply with a C-STORE-RSP). Route by command field after each `receiveMessage`: a
  `CommandCGetRSP` advances the iterator, a `CommandCStoreRQ` is answered via the M2 `serveStoreMessage` against the
  AE's store sink. Do NOT reimplement the store reply — reuse `serveStoreMessage`. This is the one increment where the
  reuse of M2's `serveStoreMessage` on the *SCU* side is essential; note it so the executor does not write a parallel
  store path.

- **Increment 7/8 (N-services send a dataset, not a bare command):** N-CREATE/N-SET (MPPS attributes) and N-ACTION
  (the action-information dataset) carry a dataset after the command set, exactly like C-STORE. They flow through the
  unchanged `sendMessage`/`fragmentMessage` (command-last bit set independently of the dataset — the M2 DIMSE-001 fix
  applies). Do NOT add a new fragmentation path; the N-services are command+dataset messages the M2 message layer
  already handles.

- **All increments (DriveInbound, non-negotiable):** every new inbound read — the SCU's RSP reads, the SCP's
  sub-operation C-STORE-RSP reads, the N-EVENT-REPORT receipt — routes through `dul.DriveInbound` against the
  operation's own `StateMachine` (the architect's DUL-ownership decision). Never reimplement the abort-send /
  clean-close / malformed-PDU distinction; it lives in `DriveInbound` alone.
