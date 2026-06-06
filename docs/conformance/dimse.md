# DIMSE conformance statement

> **Implementation status: NOT YET SHIPPED.** This is a scaffold. A standalone, versioned DIMSE conformance
> statement — the network-plane scope contract for the `dimse` package presented on its own — is not yet authored.
> Until this banner is removed, the authoritative DICOM-network scope contract is the DICOM conformance statement at
> [`./dicom.md`](./dicom.md), which already declares the supported SOP Classes, presentation-context presets, transfer
> syntaxes, association-negotiation features, and SCU/SCP roles for the DIMSE services. Do not cite this scaffold as a
> conformance basis; cite `./dicom.md`.

| Field | Value |
|-------|-------|
| Standard | DICOM DIMSE networking (DICOM PS3.7, PS3.8) |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | unassigned (standalone statement not yet authored) |
| Status | **NOT YET SHIPPED** — scaffold only; see [`./dicom.md`](./dicom.md) |
| Scope authority | The DICOM statement ([`./dicom.md`](./dicom.md)) is currently authoritative for DIMSE scope |

This document will, when authored, present the DIMSE network plane as a focused conformance statement separate from the
DICOM data layer: the DICOM Upper Layer protocol (DUL, PS3.8), the ACSE association negotiation, and the DIMSE-C /
DIMSE-N service operations (PS3.7) as both Service Class User and Service Class Provider. Today that scope is declared
in [`./dicom.md`](./dicom.md), which is the single source of truth until this statement supersedes the DIMSE portion of
it. Splitting it out is a deliberate, reviewed change, not a silent capability change.

## Scope summary

Not yet authored. This section will enumerate the DIMSE services and their roles. The v1 target is declared in
[`./dicom.md`](./dicom.md); the `dimse` package implements that target incrementally, so what ships today is a subset
of it:

- **Verification** (C-ECHO) as SCU and SCP — implemented.
- **Storage** (C-STORE) as Storage SCU and Storage SCP over the radiology-first SOP Class set — implemented.
- **Query/Retrieve**: C-FIND, C-MOVE, and C-GET under the Patient Root and Study Root information models, as SCU and
  SCP, are implemented. C-GET uses same-association SCP/SCU role selection: the SCP C-STOREs each matched instance back
  to the requestor over the same association, so the requestor takes the Storage SCP role to receive the sub-operations.
  A partial failure (one or more sub-operations the destination rejects) surfaces as a Warning or Failure terminal
  status, never laundered into Success.
- **Modality Worklist** (C-FIND) as SCU — implemented. The `Association.FindWorklist` entry point queries the Modality
  Worklist Information Model — FIND SOP Class. The worklist is a flat information model with no level hierarchy, so the
  SCU suppresses Query/Retrieve Level (0008,0052): unlike Patient Root or Study Root C-FIND it is never written into
  the sent identifier (PS3.4 K.6.1.2.1). Match and return keys live under the Scheduled Procedure Step Sequence
  (0040,0100) and its nested attributes. This is step 2 of the radiology workflow — the modality queries its worklist.
  The SCU is verified by unit tests against a mock worklist SCP. The dcm4chee live interop leg is present but **skips**
  until the archive is configured as a Modality Worklist FIND SCP: dcm4chee-arc's default archive AE rejects the MWL FIND
  presentation context as abstract-syntax-not-supported, so the dcm4chee MWL interop gate is a documented follow-up.
- **MPPS** (Modality Performed Procedure Step) as SCU — implemented. The `Association.MPPS()` entry point returns an
  `*MPPS` whose `Create` issues the N-CREATE that opens a procedure step in the **IN PROGRESS** state and whose `Set`
  issues the N-SET that advances it to **COMPLETED** or **DISCONTINUED** (PS3.4 F.7.1, the Performed Procedure Step
  Status keywords of PS3.3 C.4.14). `Create` defaults the IN PROGRESS state into a copy of the caller's attribute set
  when absent and returns the procedure-step SOP Instance UID — the SCU's own when supplied, otherwise the one the SCP
  assigns and echoes in the N-CREATE-RSP — which the caller passes to `Set`. The N-SET references the step by Requested
  SOP Class UID (0000,0003) and Requested SOP Instance UID (0000,1001), the normalised reference pair distinct from the
  C-service Affected SOP Class/Instance UID. Each operation surfaces a typed Status categorised against the Procedure
  Step service class; a Failure (for example, the step may no longer be updated) is in-band data, never laundered to
  success. This is step 3.5 of the radiology workflow — the modality reporting its procedure step. The SCU is verified
  by unit tests against an in-process mock N-service SCP. The dcm4chee live interop leg drives N-CREATE then N-SET to
  IN PROGRESS then COMPLETED; it confirms the exchange round-trips with a typed status against the real archive, and
  **skips** the success assertion when dcm4chee rejects the data set on its MPPS IOD attribute enforcement rather than
  fail — completing the data set to dcm4chee's exact Type-1/2 MPPS requirements is a documented follow-up. The SCP side
  is deferred.
- **Storage Commitment** (Push Model) as SCU — implemented. The `Association.StorageCommitment()` entry point returns a
  `*StorageCommitment` whose `Request` issues the N-ACTION (Action Type ID 1, "Request Storage Commitment") against the
  well-known Storage Commitment Push Model SOP Instance `1.2.840.10008.1.20.1.1` (PS3.4 J.3.2). The request carries a
  caller-allocated Transaction UID (0008,1195) and the Referenced SOP Sequence (0008,1199) of the SOP Instances whose
  commitment is sought; it references the target by Requested SOP Class UID (0000,0003) and Requested SOP Instance UID
  (0000,1001), the normalised reference pair distinct from the C-service Affected pair, and carries the Action Type ID
  (0000,1008). `Request` surfaces the N-ACTION-RSP as a typed Status categorised against the Storage Commitment service
  class; a Success there means the SCP accepted the *request*, not that commitment succeeded. The authoritative
  commitment outcome arrives later via N-EVENT-REPORT.

  **Separate-association reporting model (supported).** A real Storage Commitment SCP (for example dcm4chee-arc) reports
  the result on a LATER association it opens back to the SCU, on which the roles invert: the original SCU becomes the
  acceptor and the N-EVENT-REPORT receiver. go-radx implements this supported model with `CommitmentReceiver`: the SCU
  accepts an inbound connection on its listening port and drives `ServeConn`, which negotiates the Storage Commitment
  context, reads the N-EVENT-REPORT-RQ, parses the result, invokes the configured `WithCommitmentHandler` callback, and
  acknowledges the report with an N-EVENT-REPORT-RSP. Receiving the result SYNCHRONOUSLY on the original N-ACTION request
  association is a **stated limitation, not supported** — in the Push Model the request and the report are distinct
  associations.

  **Partial-failure semantics.** The parsed `StorageCommitmentResult` lists the committed instances (Referenced SOP
  Sequence) and the failed instances (Failed SOP Sequence, each with its Failure Reason 0008,1197). A non-empty failed
  list is a FAILURE: `HasFailures()` reports it and `Err()` returns a typed `*CommitmentFailureError`, so a failed
  instance is never laundered into success — the same typed-status honesty as C-GET/STOW partial failures (PRD §9.2
  fail-closed). The SCU is verified by unit tests against an in-process mock N-service SCP (the N-ACTION request, the
  separate-association N-EVENT-REPORT receipt, and a failed-instance result that must surface as failure). The
  dcm4chee live interop leg drives the N-ACTION request and asserts it round-trips with a typed status; it **skips**
  when dcm4chee does not advertise the Storage Commitment context as SCP, and the live N-EVENT-REPORT receipt is a
  documented follow-up because dcm4chee opens the reporting association only to a remote AE it has been configured with
  (a known AE Title bound to the SCU's host and port), which the ephemeral interop harness does not register. The SCP
  side (committing instances) is deferred.

The DIMSE-C service operations and the roles each ships as today:

| Service | Information model | SCU | SCP | Reference |
|---------|-------------------|-----|-----|-----------|
| C-ECHO | Verification | Shipped | Shipped | PS3.7 §9.1.5 |
| C-STORE | Storage (radiology-first set) | Shipped | Shipped | PS3.4 B |
| C-FIND | Patient Root / Study Root Q/R | Shipped | Shipped | PS3.4 C.4.1 |
| C-FIND | Modality Worklist (flat, no level) | Shipped | Shipped | PS3.4 K.4.1.2, K.6.1.2.1 |
| C-MOVE | Patient Root / Study Root Q/R | Shipped | Shipped | PS3.4 C.4.2 |
| C-GET | Patient Root / Study Root Q/R | Shipped | Shipped | PS3.4 C.4.3 |

The DIMSE-N service operations and the roles each ships as today:

| Service | SOP Class | SCU | SCP | Reference |
|---------|-----------|-----|-----|-----------|
| N-CREATE, N-SET | Modality Performed Procedure Step | Shipped | Deferred | PS3.4 F.7.1, PS3.7 §10.1 |
| N-ACTION, N-EVENT-REPORT | Storage Commitment Push Model | Shipped (SCU; separate-assoc report) | Deferred | PS3.4 J.3 |

## Association negotiation

Not yet authored. This section will declare the negotiated features. The table below tracks the negotiation options
against the v1 target declared in [`./dicom.md`](./dicom.md); a feature is shipped only when both peer roles
(requestor and acceptor) round-trip it through the codec and negotiation layers.

| Negotiation feature | Reference | Status |
|---------------------|-----------|--------|
| Presentation-context negotiation | PS3.8 §9.3.3.2 | Shipped |
| Maximum PDU length | PS3.7 Annex D.1 | Shipped |
| Implementation identity (class UID, version name) | PS3.7 D.3.3.2 | Shipped |
| SCP/SCU role selection (sub-item 0x54) | PS3.7 D.3.3.4 | Shipped |
| Asynchronous-operations window | PS3.7 D.3.3.3 | Not yet shipped |
| User identity | PS3.7 D.3.3.7 | Not yet shipped |
| SOP Class extended and common extended negotiation | PS3.7 D.3.3.5, D.3.3.6 | Not yet shipped |
| DIMSE-TLS | PS3.15 §B.1 | Not yet shipped |

SCP/SCU role selection lets the requestor propose, per SOP Class, which of the SCU and SCP roles each peer plays; the
acceptor responds with the roles it grants, never granting a role it does not itself support. The negotiated SCP role
is observable on the established association, which is the prerequisite for same-association C-GET. The remaining
not-yet-shipped features are declared in [`./dicom.md`](./dicom.md).

## Out of scope

Not yet authored. The explicit deferral list (the SCP side of MPPS and Storage Commitment, the other DIMSE-N services,
Print Management, UPS, RT Machine Verification, and the non-radiology Query/Retrieve models) is declared in
[`./dicom.md`](./dicom.md) until this statement supersedes it.

## Verification

Not yet authored. This section will state how conformance to the declared DIMSE subset is gated: DIMSE interop against
Orthanc and dcm4chee-arc under a build-tagged interop suite, and the CI job that invokes it. The current declaration is
in [`./dicom.md`](./dicom.md).

## References

- DICOM PS3.7 (Message Exchange), PS3.8 (Network Communication Support).
- go-radx PRD §6.1 (conformance definition), §6.2 (parity floor), §5.1 (workflow scope).
- go-radx reference docs: `docs/reference/dimse.md`.
- DICOM conformance statement: [`./dicom.md`](./dicom.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
