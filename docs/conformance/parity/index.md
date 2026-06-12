# Reference-library parity: index and build backlog

This index aggregates the six subsystem parity matrices in this directory. Each matrix compares a go-radx
subsystem against the documented public feature surface of its reference implementation, with every status
verified against go-radx source and tests (file:symbol evidence). The matrices are the authoritative backlog
for the parity build-out: every PARTIAL and NOT-MET row is a tracked task, and a row flips to MET only in the
same PR that ships the feature (lockstep with the conformance statements).

Audited 2026-06-10 against main @ fdf7b54. Statuses: MET (implemented and tested), PARTIAL (usable subset),
NOT-MET (absent), N-A (no sensible Go equivalent). Sizes: S (under 1 day), M (1-3 days), L (over 3 days).

## Aggregate results

| Subsystem | Reference(s) | Matrix | Rows | MET | PARTIAL | NOT-MET | N-A |
|---|---|---|---|---|---|---|---|
| DICOM data layer | pydicom + pylibjpeg | [dicom.md](dicom.md) | 88 | 54 | 8 | 20 | 6 |
| DIMSE networking | pynetdicom 3.0.4 | [dimse.md](dimse.md) | 93 | 60 | 9 | 23 | 1 |
| HL7 v2 (floor) | python-hl7 | [hl7v2.md](hl7v2.md) | 29 | 24 | 4 | 0 | 1 |
| FHIR | fhir.resources + HAPI REST | [fhir.md](fhir.md) | 98 | 50 | 8 | 35 | 5 |
| DICOMweb | dicomweb-client + PS3.18 | [dicomweb.md](dicomweb.md) | 75 | 42 | 4 | 27 | 2 |
| radx CLI | dcmtk application suite | [cli.md](cli.md) | 28 | 8 | 10 | 10 | 0 |
| Total | | | 411 | 238 | 43 | 115 | 15 |

The HL7 matrix additionally carries a clearly-labelled stretch section against the HAPI v2 message catalogue
(~195 typed structures per version vs go-radx's 5 radiology-scoped families); those rows are sized in
[hl7v2.md](hl7v2.md) and scheduled in wave 6 below rather than counted in the floor.

## Reading the results

The python-hl7 floor is effectively met (zero NOT-MET). The DIMSE association plane and DIMSE-C services are
at full pynetdicom parity; gaps concentrate in DIMSE-N SCP sides and UPS. FHIR models and the REST client are
near parity; roughly two thirds of FHIR gaps sit in the server role's write side. The DICOM data layer's
former headline finding - `dicom.Read` rejecting every encapsulated transfer syntax - is resolved: compressed
Part 10 files read with the dataset retained, write back byte-identically, and transcode at the dataset level,
which also unblocked `radx dump`, `radx modify` and `radx store --transcode-to` on compressed files.

## Wave plan

Wave 0 is pre-authorised. Wave order from 1 onward is the proposed sequence; Andru re-prioritises between
waves. Each wave ends with full local gates, real-CI green, matrix rows flipped, and conformance statements
updated in the same PR.

### Wave 0 - floor violations, tracked issues, and audit quick wins

| Item | Matrix anchor | Size |
|---|---|---|
| Compressed Part 10 read with dataset retained (`dicom.Read` encapsulated TS) | dicom.md #1 | M |
| Compressed Part 10 write (encapsulated pixel element, BOT) | dicom.md #2 | M |
| Dataset-level transcode round-trip (unblocks `radx store --transcode-to`) | dicom.md #3 | M |
| DICOMDIR FileSet read + write | dicom.md #4-5 | L |
| Deferred/lazy large-element reads (PRD 6.2 `defer_size` analogue) | dicom.md #6 | M |
| C-CANCEL during C-MOVE (move_scp.go:103 TODO) | dimse.md #9 | S |
| Issue #113 audit hook | tracked issue | S |
| Issue #114 MLLP + FHIR-HAPI interop CI legs | tracked issue | M |
| `radx serve fhir` stub to real subcommand | fhir.md / cli.md | M |
| FHIR server vread/history 501s implemented (needs version store) | fhir.md #1-2 | L |
| DICOMweb daemon retrieval wiring (role mounts study/series/metadata/frames/bulkdata) | dicomweb.md #10 | S |
| HL7 `AsOMG` typed accessor | hl7v2.md | S |
| `radx find -W` worklist flag (FindWorklist exists, unexposed) | cli.md #6 | S |
| FHIR server `$validate` (release validator already gates create) | fhir.md #7 | S |

### Wave 1 - DICOM data layer (unlocks waves 4 and 5 codec work)

VOI/modality LUT and windowing utilities (M each), palette colour expansion and colour-space conversion
(M each), private block API and private-creator dictionaries (M each), charset table completion: ISO 2022
IR 149 Korean, IR 58 Chinese, ISO_IR 166 Thai, bare ISO_IR 13 (S each), overlays (M), waveforms (M),
python-hl7-style small utilities flagged PARTIAL in dicom.md.

### Wave 2 - DIMSE-N and remaining service classes

N-GET and N-DELETE primitives (M each), DIMSE-N SCP dispatch and handler interfaces (L), MPPS SCP plus
Retrieve/Notification SOP classes (L), Storage Commitment SCP side (M), UPS push/pull/watch/event/query (L),
Composite Instance Root and instance/frame-level retrieve (M), notification/monitoring event hooks (M),
remaining Q/R-family service classes reusing existing machinery (M each). Print Management (L) is a
residual-candidate for Andru's call.

### Wave 3 - FHIR server depth and release breadth

Server write side beyond create: update/patch/delete with conditional forms (L, builds on wave 0 version
store), search depth: parameter registry, `_include`/`_revinclude`, chaining (L), paging with `Bundle.link`
(M), batch at the base endpoint (M), operations framework with `$everything` (M), R4B models via generator
re-run (M), STU3 structural assessment first - report before building (assessment S, build TBD), nested
backbone validation and primitive lexical validation (M each). SMART on FHIR (L) and XML serialisation (L)
are residual-candidates.

### Wave 4 - DICOMweb completion

`application/dicom+xml` media type (M), WADO-URI service (M), capabilities discovery (M), thumbnail
resources (M), pixel-data resources of Table 10.1-1 (M), rendered resources (L), server-side transcoding
(L, builds on wave 1 codecs), client small misses: byte ranges, retries, auto-pagination (S each). UPS-RS
worklist service (L, builds on wave 2 UPS) and notifications/WebSocket (L) are residual-candidates.

### Wave 5 - radx CLI parity

`radx transcode` command closing the dcmconv/dcmcrle/dcmdrle/dcmdjpeg/dcmcjpls rows (M, builds on waves 0-1),
JPEG baseline encode for dcmcjpeg parity (L), TLS flags across the six network commands (M), `radx serve
dimse` / dcmqrscp-equivalent wiring with C-GET/C-MOVE SCP mounts (M), worklist SCP CLI (S), DICOMDIR
commands over the wave 0 FileSet (M). Image rendering/import (dcm2pnm/img2dcm family, L) is a
residual-candidate.

### Wave 6 - HL7 breadth (HAPI stretch)

Four S-sized python-hl7 partials (predicates, `split_file`, charset parameter, custom escape map). Typed
families by clinical value: QBP/RSP query (L), MFN (L), SIU/MDM/DFT/VXU (M each), pharmacy families (L).
v2 XML and conformance-profile validation remain deliberately out of scope (recorded in hl7v2.md).

## Residual-candidate summary

Items flagged above as residual-candidates (Print Management, SMART on FHIR, FHIR XML, DICOMweb
notifications and rendered-at-L scope, dcm2pnm/img2dcm, HL7 v2 XML) need an explicit keep-or-defer decision
from Andru; deferred items get a tracked issue and an N-A-or-deferred row note rather than silent omission.

## Maintenance

Each feature PR flips its matrix row(s) in the same change, with the evidence column updated to the shipped
symbols. The conformance-drift gate guards the statements in `docs/conformance/*.md`; these matrices are
audit artifacts and are refreshed by re-running the audit, not by the drift tool.
