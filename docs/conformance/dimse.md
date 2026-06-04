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
- **Query/Retrieve**: C-FIND and C-MOVE under the Patient Root and Study Root information models, as SCU and SCP, are
  implemented. **C-GET is NOT YET SHIPPED** — it is in the v1 target but not yet exposed as an operation.
- **Modality Worklist** (C-FIND) as SCU, with a reference worklist SCP — implemented.
- **MPPS** (N-CREATE, N-SET) and **Storage Commitment** (N-ACTION, N-EVENT-REPORT), SCU only, are in the v1 target but
  the N-service operations are **NOT YET SHIPPED**; today only their presentation-context presets and status codes
  exist.

## Association negotiation

Not yet authored. This section will declare the negotiated features. Implemented today: presentation-context
negotiation, maximum PDU length, and implementation identity. The remaining v1-target features — SCP/SCU role
selection, asynchronous-operations window, user identity, SOP Class extended and common extended negotiation, and
DIMSE-TLS — are declared in [`./dicom.md`](./dicom.md) but **NOT YET SHIPPED** as negotiation options.

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
