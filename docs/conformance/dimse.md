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
- **MPPS** (N-CREATE, N-SET) and **Storage Commitment** (N-ACTION, N-EVENT-REPORT), SCU only, are in the v1 target but
  the N-service operations are **NOT YET SHIPPED**; today only their presentation-context presets and status codes
  exist.

The DIMSE-C service operations and the roles each ships as today:

| Service | Information model | SCU | SCP | Reference |
|---------|-------------------|-----|-----|-----------|
| C-ECHO | Verification | Shipped | Shipped | PS3.7 §9.1.5 |
| C-STORE | Storage (radiology-first set) | Shipped | Shipped | PS3.4 B |
| C-FIND | Patient Root / Study Root Q/R | Shipped | Shipped | PS3.4 C.4.1 |
| C-FIND | Modality Worklist (flat, no level) | Shipped | Shipped | PS3.4 K.4.1.2, K.6.1.2.1 |
| C-MOVE | Patient Root / Study Root Q/R | Shipped | Shipped | PS3.4 C.4.2 |
| C-GET | Patient Root / Study Root Q/R | Shipped | Shipped | PS3.4 C.4.3 |

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
