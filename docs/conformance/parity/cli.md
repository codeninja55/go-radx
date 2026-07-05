# radx CLI parity with the dcmtk application suite

This matrix compares the `radx` CLI (`cmd/radx`, audited at commit `fdf7b54`, re-verified at `c1b839d`
2026-07-04) against the dcmtk command-line
tool suite as documented at <https://support.dcmtk.org/docs/pages.html>. Scope is limited to dcmtk apps whose
underlying capability go-radx implements or plans (DICOM data layer, DIMSE/DICOMweb networking, pixel codecs).
Apps outside that domain are listed under "Out of scope" with a one-line justification rather than silently
dropped. Parity is judged at feature level (the tool's core function and headline options), not flag-by-flag.

Statuses: **MET** (core function and headline behaviour available), **PARTIAL** (core function available with
a named feature gap, or the capability ships in the library but has no CLI command), **NOT-MET** (no usable
equivalent). Size estimates (S/M/L) apply to non-MET rows and gauge the work to close the gap given what the
library already provides.

## Summary

28 in-scope dcmtk tools/tool-pairs: **9 MET, 12 PARTIAL, 7 NOT-MET.**

Top gaps, largest first:

1. **Transcode command shipped; JPEG encode remains (M).** `radx transcode` (`command/transcode.go`)
   rewrites a file's transfer syntax through the library seam (`dicom.Transcode` + `File.SetPixelData`,
   PR #121): `--to` accepts a UID or dicom keyword, `--output-dir`/`--in-place` destinations,
   pixel-less objects pass through with a meta rewrite, and unsupported encode targets fail closed with
   the library's typed error (exit 3). This closes dcmcrle/dcmdrle outright and moves dcmconv,
   dcmdjpeg, and dcmcjpls/dcmdjpls to their remaining named gaps (charset conversion; CGo-gated JPEG
   decode; encode policy). dcmcjpeg (lossy/lossless JPEG *encode*) still needs encode-side codec work
   in the library before the CLI can honour it.
2. **Image rendering and import (L).** No `dcm2pnm`/`dcm2img`/`img2dcm` equivalent: the repo has no PNG/PPM
   export or consumer-image import path (no `image/png` use anywhere). The windowing/VOI/modality LUT and
   palette/photometric pipeline now ships in the library (`dicom/lut.go`, `dicom/colorspace.go`, PRs
   #127/#128); the remaining gap is display-value-to-image composition and the format writers.
3. **Q/R SCP archive (L).** `dcmqrscp` maps to `server.NewDIMSERole` (C-ECHO/C-STORE/C-FIND over
   `ObjectStore`+`Catalogue`, optional MWL), but C-GET/C-MOVE SCP are explicitly unmounted
   (`server/role_dimse.go:182`) and there is no `radx serve dimse` subcommand.
4. **CLI TLS (M, cross-cutting).** `dimse.WithTLS` ships in the library (`dimse/ae.go:79`), but no `radx`
   network command (`echo`, `store`, `find`, `get`, `move`, `scp`) exposes a TLS flag; dcmtk's `+tls` family
   has no CLI counterpart. This caveat applies to every dcmnet row below and is not repeated per row.
5. **DICOMDIR (M).** `dcmgpdir`/`dcmmkdir` have no CLI equivalent, but `dicom.NewFileSetBuilder` builds and
   writes a file-set and `dicom.OpenFileSet` reads and queries one (PR #119); a `radx` DICOMDIR subcommand
   over the shipped FileSet closes it, with no further library work for create/read.

## DICOM data tools (dcmdata, dcmqrdb index)

| dcmtk app | Function | Status | radx evidence | Size | Notes |
|---|---|---|---|---|---|
| dcmdump | Dump file/dataset elements | MET | `radx dump` (`command/dump.go`): `-R` recursive, `--tag`/`--group` filters, `--process-pixel-data`, human/json/csv | - | Values shown by default with `--redact` opt-in masking PS3.15 confidentiality attributes - an intentional design decision matching dcmdump's posture, not a gap |
| dcm2json | DICOM to PS3.18 JSON | PARTIAL | `radx dump --format json` emits radx's own tag-keyed shape (`dumpFile`/`dumpElement`), not the PS3.18 Annex F model | S | PS3.18 DICOM-JSON marshal/unmarshal already ships in the `dicomweb` package (QIDO/WADO metadata); wiring it into dump or a dedicated flag is small |
| dcm2xml | DICOM to XML (Native Model) | PARTIAL | `dicomweb/xml.go:MarshalXML` (PS3.19 Native DICOM Model, PR #136); tests `dicomweb/xml_test.go`, `metadata_xml_test.go` | S | Library-only: `MarshalXML` encodes any `*dicom.DataSet`, but no CLI command emits XML; wiring it into `dump` or a flag closes this |
| xml2dcm | XML to DICOM | NOT-MET | `dicomweb/xml.go:UnmarshalXML` (PS3.19, PR #136) parses Native Model XML into a `*dicom.DataSet` | M | `dicomweb` has the XML unmarshal and `dicom` has the Part 10 writer; a CLI builder command is glue plus VR/meta handling (same shape as json2dcm) |
| json2dcm | PS3.18 JSON to DICOM | NOT-MET | None | M | `dicomweb` has the JSON unmarshal and `dicom` has the Part 10 writer; a CLI builder command is glue plus VR/meta handling |
| dump2dcm | ASCII dump to DICOM | NOT-MET | None | M | No dump-text parser; lower value given json2dcm would cover the round-trip need |
| dcmodify | Insert/modify/erase tags, generate UIDs | PARTIAL | `radx modify` (`command/modify.go`): `--insert`, `--delete`, `--regenerate-{study,series,instance,all}-uids` with batch-consistent UID remapping, atomic temp-and-rename writes | M | No sequence-path syntax (`seq[n].element`), no `--modify-all`/`--erase-all` wildcards, no file-based value insert; UID regeneration preserves the study/series reference graph across a batch, which dcmodify does not do per-run |
| dcmconv | Convert file encoding (implicit/explicit, deflated, charset) | PARTIAL | `radx transcode` (`command/transcode.go`, `TranscodeCmd`): `--to` UID/keyword targets across the uncompressed and Deflated syntaxes, `--output-dir`/`--in-place`, `-R`; atomic temp-and-rename writes; pixel-less meta rewrite (`TestTranscodePixelLessObjectRewritesMetaOnly`, `TestTranscodeDecompressesRLEToExplicitVRLE`, `command/transcode_test.go`) | S | Charset conversion (dcmconv's `+U8` character-set repertoire rewrite) not covered; that is the remaining named gap |
| dcmcrle / dcmdrle | RLE encode/decode | MET | `radx transcode --to RLELossless` encodes and `--to ExplicitVRLittleEndian` decodes via the pure-Go RLE codec (`dicom/codec_rle.go`); pixel round-trip pinned by `TestTranscodeEncodesToRLEByKeyword` and `TestTranscodeDecompressesRLEToExplicitVRLE` (`command/transcode_test.go`) | - | Both directions ship pure Go in the default build |
| dcmcjpeg | Encode to JPEG syntaxes | NOT-MET | Library JPEG codecs are decode-only for Baseline/Extended/Lossless (`docs/conformance/dicom.md` Tier 3 table); `radx transcode --to` a JPEG syntax fails closed with `ErrEncodeUnsupported` (`TestTranscodeUnsupportedEncodeTargetFailsClosed`) | L | Needs encode-side libjpeg work in the library before the CLI can honour it |
| dcmdjpeg | Decode JPEG to uncompressed | PARTIAL | `radx transcode --to ExplicitVRLittleEndian` decodes JPEG Baseline/Extended/Lossless/SV1 sources when the binary is built with the `dicom_libjpeg` CGo tag (`dicom/codec_libjpeg.go`); the default pure-Go build fails closed with `ErrCodecUnavailable` (exit 3) | S | Core function ships but is CGo-build-gated; a default `radx` binary cannot decode JPEG sources |
| dcmcjpls / dcmdjpls | JPEG-LS encode/decode | PARTIAL | `radx transcode` drives the CharLS codec when built with the `dicom_charls` CGo tag: decode for lossless and near-lossless, encode for lossless only (`dicom/codec_charls.go`); default pure-Go build fails closed | S | CGo-build-gated, and near-lossless encode is absent by policy (decode-only) |
| dcmftest | Test for Part 10 format | PARTIAL | No dedicated command; `radx dump` exits 3 on a malformed/non-DICOM file and 0 on a valid one | S | Scriptable today via dump's exit code; a quiet `--check` mode would be a direct equivalent |
| dcmgpdir / dcmmkdir | Create DICOMDIR | PARTIAL | Library: `dicom.NewFileSetBuilder` builds and writes a DICOMDIR file-set (`Add`/`AddFile`/`Write`, `dicom/fileset_write.go`), `dicom.OpenFileSet` reads and queries one (`Find`/`FindValues`/`Records`/`Instances`, `dicom/fileset.go`); no CLI command | M | FileSet build/read shipped in PR #119; closed by a `radx` DICOMDIR subcommand over it (no further library work for create/read; update-in-place remains out of scope) |
| img2dcm | Consumer image (JPEG/BMP) to DICOM | NOT-MET | None | M | Needs SC Image IOD construction plus image import; no current plan in conformance docs |
| dcm2pnm / dcmj2pnm / dcml2pnm / dcm2img | Render DICOM to PGM/PNG/TIFF/BMP/JPEG | NOT-MET | No image export path in the repo; windowing/VOI/modality LUT and palette/photometric conversion ship in the library (`dicom/lut.go`, `dicom/colorspace.go`, PRs #127/#128) | L | Remaining gap is display-value-to-image composition and the format writers, not the rendering pipeline |
| dcmqridx | Register files in a query database index | MET | `radx catalogue` (`command/catalogue.go`): indexes a directory into SQLite, `--rebuild`, `--query`, read-only `--sql` | - | Exceeds dcmqridx: queryable SQL, PHI gate (`--confirm-phi`), `--redact` identifier hashing, 0600 db file |

## DICOM networking tools (dcmnet, dcmqrdb, dcmwlm)

| dcmtk app | Function | Status | radx evidence | Size | Notes |
|---|---|---|---|---|---|
| echoscu | C-ECHO verification SCU | MET | `radx echo HOST PORT` (`command/echo.go`): AE titles, timeout, max-PDU, named DIMSE status, exit 4 on peer "no" | - | TLS flag absent (cross-cutting gap above) |
| storescu | C-STORE SCU with TS proposal sets | PARTIAL | `radx store` (`command/store.go`): `-R` recursive, `--workers` pooled associations (1-128), `--continue-on-error`, `--max-pdu` (capped 131072, same ceiling as dcmtk), per-file JSON Lines + summary | M | No `--propose-*` compression proposal sets (proposes the default TS set per context); `--transcode-to` transcodes to uncompressed targets via `dicom.Transcode` (`prepareForStore`), passes pixel-less objects unchanged, and fails closed for encapsulated targets; no `--repeat`/UID-invention options. Worker-pooled multi-association batching exceeds storescu |
| dcmsend | Simplified storage SCU | MET | Same `radx store` surface | - | No `--report-file`; the JSON Lines stream plus summary line serves the same automation need |
| storescp | Storage SCP writing received files | PARTIAL | `radx scp` (`command/scp.go`): loopback-default `--bind`, `--aet`, `--output-dir`, `--no-accept-echo`, `--max-conns`, UID-validated paths, graceful drain | S | No study-folder sorting (`--sort-conc-studies`), no `--exec-on-reception` hooks, no filename templates; loopback-by-default and traversal-safe UID paths exceed storescp's defaults |
| dcmrecv | Simplified storage SCP | MET | Same `radx scp` surface | - | - |
| findscu | C-FIND SCU (Q/R models + worklist) | MET | `radx find` (`command/find.go`): `--level PATIENT/STUDY/SERIES/IMAGE`, repeatable `--match key=value` (keyword or tag forms), streamed matches as JSON Lines/CSV; proposes all six Patient Root + Study Root FIND/MOVE/GET contexts (`dimse/presets.go:115`); `-W`/`--worklist` queries the Modality Worklist model via `dimse.FindWorklist` + `BasicWorklistContexts` on the SPS-sequence skeleton, routing Table K.6-1 SPS `--match` keys into the sequence item via `dimse.SetWorklistMatch` (`TestFindWorklistFlagStreamsScheduledSteps`, `TestFindWorklistRoutesSPSMatchKeysIntoSequence`, `command/find_test.go`) | - | No `--extract` of responses to DICOM/XML files; matches stream as JSON Lines/CSV instead |
| getscu | C-GET SCU (same-association retrieve) | MET | `radx get` (`command/get.go`): levels, match keys, `--output-dir`, Storage SCP role negotiation for sub-operation C-STOREs, sub-op counts in the result | - | - |
| movescu | C-MOVE SCU to a destination AE | MET | `radx move` (`command/move.go`): `--move-destination`, levels, match keys, faithful Warning/Failure terminal reporting (exit 4) | - | No embedded receive port (`movescu --port` spawns its own SCP); run `radx scp` alongside instead |
| termscu | Association termination test SCU | NOT-MET | None | S | Niche diagnostic tool; the dimse layer has full ACSE association/release/abort, so a command is small if ever wanted |
| dcmqrscp | Image archive: storage + Q/R SCP + DB | PARTIAL | Library: `server.NewDIMSERole` serves C-ECHO/C-STORE/C-FIND over `ObjectStore` + `Catalogue` (`server/role_dimse.go`); CLI: `radx scp` (storage only) + `radx catalogue` (index/query) | L | C-GET/C-MOVE SCP are explicitly unmounted in the role ("a later increment", `server/role_dimse.go:182`) and no `radx serve dimse` subcommand exposes the role |
| wlmscpfs | Modality Worklist SCP from data files | PARTIAL | Library: `server.WithWorklistSource` mounts an MWL SCP in the DIMSE role (`server/role_dimse.go:46`); no CLI command | M | Needs a `radx serve dimse` (or worklist) subcommand plus a file-backed `WorklistSource`; the SCU side ships as `radx find -W` (see findscu row) |

Note: the audit brief listed `dcmanonymize`; no dcmtk tool of that name exists in the current docs index.
dcmtk de-identification is done through dcmodify recipes; the radx counterparts are `modify --delete` +
`--regenerate-*-uids` and the `dump --redact` / `catalogue --redact` views. A PS3.15 E.1 de-identification
profile command exists in neither suite.

## Out of scope

dcmtk apps excluded because go-radx neither implements nor plans the underlying capability:

- **dcmpstat suite** (dcmmkcrv, dcmmklut, dcmp2pgm, dcmprscp, dcmprscu, dcmpsmk, dcmpsprt, dcmpsrcv,
  dcmpssnd, dcmpschk): softcopy presentation state and DICOM print management are outside go-radx's
  data/networking/codec scope.
- **dcmsr rendering tools** (dsr2html, dsr2xml, dsrdump, xml2dsr): SR tree rendering/authoring is out of
  scope; go-radx's SR support is extraction to FHIR (`radx convert sr-to-fhir`), and `radx dump` reads an SR
  file at the element level.
- **dcmsign**: digital signatures (PS3.15 profiles) are not in scope.
- **dcmquant, dcmscale, dcmicmp** (dcmimage processing): palette quantisation, scaling, and image-difference
  metrics require the image-processing pipeline go-radx does not plan.
- **dcm2img/dcmimgle calibration tools** (dcmdspfn, dcod2lum, dconvlum): display calibration (GSDF) is out of
  scope. (`dcm2img` itself is rowed above with dcm2pnm as the rendering gap.)
- **Encapsulation tools** (pdf2dcm, dcm2pdf, cda2dcm, dcm2cda, stl2dcm, dcmencap, dcmdecap): encapsulated
  PDF/CDA/STL document objects are outside the radiology imaging scope.
- **dcmqrti**: an interactive telnet-style test client for dcmqrscp; diagnostic UI, not a capability.
- **drtdump**: RT-specific dump; `radx dump` reads RT files generically, and RT module semantics are not in
  scope.
- **oficonv tools** (mkcsmapper, mkesdb): build-time character-set database generators internal to dcmtk.

## Beyond dcmtk

radx capabilities with no dcmtk equivalent:

- **DICOMweb clients**: `radx dicomweb wado` (objects byte-preserving or `--metadata`), `radx dicomweb stow`
  (study-scoped or root, honest partial-failure exit), `radx dicomweb qido` (match/include/limit, streamed
  matches). dcmtk's core suite has no DICOMweb tools.
- **DICOMweb server**: `radx serve dicomweb` runs the reference daemon (WADO-RS/STOW-RS/QIDO-RS) over the
  filesystem object store and SQLite catalogue, loopback-default with `ErrInsecureBind` fail-closed on
  non-loopback binds without authentication.
- **FHIR server**: `radx serve fhir` runs the FHIR REST reference daemon (`server.NewFHIRRole` over the
  in-memory development repository, one release per process via `--release r4|r5`), loopback-default with
  `ErrInsecureBind` fail-closed on non-loopback binds without authentication (`command/serve.go`).
- **HL7 v2 over MLLP**: `radx hl7 send` (file or stdin, named MSA-1 ACK result, non-accept ACK exits 4) and
  `radx hl7 listen` (loopback-default MLLP responder with frame cap).
- **Cross-standard conversion**: `radx convert dicom-to-fhir | sr-to-fhir | oru-to-fhir | orm-to-fhir |
  adt-to-fhir`, each release-explicit (R4/R5).
- **Catalogue queries**: `radx catalogue --sql` read-only SQL with six render modes, plus the PHI gate and
  redaction described above.
- **Operational contract**: machine-clean stdout vs stderr diagnostics, `--format human|json|csv`, a typed
  exit-code taxonomy (`internal/exitcode`), env-var binding (`RADX_*`), and `radx organize` / `radx lookup`
  utilities (lookup has no standalone dcmtk tool; organize overlaps storescp's sorting only as a receiver
  feature there).

## Methodology

- dcmtk inventory: fetched <https://support.dcmtk.org/docs/pages.html> (current docs, dcmtk 3.6.x line) on
  2026-06-10, plus the per-tool pages for storescu, findscu, and dcmodify to verify names and headline
  options. The index fetch rendered two tool names with typos (`dcmmodify`, `dcmmmkdir`); the verified names
  are `dcmodify` and `dcmmkdir`. Headline options for tools whose pages were not individually fetched
  (storescp, movescu, dcmqrscp, dcm2pnm, and others) come from general dcmtk knowledge and were not
  re-verified against current pages - flag-level claims for those rows should be re-checked before being
  cited externally.
- radx inventory: read from source at `cmd/radx/internal/command/*.go` and `cmd/radx/internal/cli/*.go`
  (commit `fdf7b54`), cross-checked against `docs/conformance/cli-server.md` and `docs/conformance/dicom.md`.
  The binary was not executed; the Kong grammar in `command/root.go` is the authoritative command tree.
- Library-capability claims (codec tiers, TLS, worklist, DIMSE role services, DICOMDIR deferral) were
  verified by grep/read of `dicom/`, `dimse/`, `dicomweb/`, and `server/` sources cited inline.
- Unverified areas: dcmtk behaviour under its config-file system (storescp/dcmqrscp association profiles) was
  not compared; interop behaviour of radx commands against real PACS was not exercised in this audit (the
  repo's own interop suites cover that and are declared in the DICOM/DICOMweb conformance statements).
