# radx CLI

!!! info "Command tree implemented — `serve fhir` deferred"

    This document describes the `radx` command-line interface. The foundation and the command tree are implemented:
    the Kong scaffold, the global output contract, the `RADX_*` environment configuration, the exit-code taxonomy, and
    the honest-failure rules are in place, and every command is wired end to end against the library — `echo`, `store`,
    `find`, `get`, `move`, `scp`, `dump`, `modify`, `organize`, `lookup`, `catalogue`, and the `hl7`, `dicomweb`,
    `convert`, and `serve dicomweb` commands. The one exception is `serve fhir`: the FHIR REST server role is a
    separate increment, so `serve fhir` fails closed — it returns a typed "not implemented" error and exits `1` until
    that role lands. The command surface below is the contract every command conforms to.

The `radx` command-line interface is go-radx's flagship first-party consumer. It is the tool that proves the library
API is usable, and it serves practitioners and operators who want dcmtk-class breadth from a single binary. It lives in
a separate Go module (`github.com/codeninja55/go-radx/cmd/radx`, `docs/prd/go-radx-prd.md` §7.4) so that library
consumers importing the core packages never inherit the CLI's Kong parser and terminal-UI dependency graph.

This reference defines the public command surface and the behaviour every command shares: the command tree, the
per-command flags and input/output, the global output formats, the 12-factor environment configuration, the exit-code
taxonomy, and the honest-failure rules. For each command it names the dcmtk tool a practitioner would reach for instead,
so the migration path is explicit.

The CLI is a ground-up rewrite of the prototype's command semantics. The Kong parser foundation is the salvageable part
and is ported; the command behaviour is not (PRD §12 verdict: REWRITE). An adversarial Codex audit
(`/tmp/go-radx-codex-reviews/cmd-radx.md`, findings RADX-001 through RADX-022) found the prototype's `modify` to be a
no-op that reported success, `store` exiting zero on failed transfers, banners corrupting machine-readable stdout, PHI
written to logs and an unprotected SQLite catalogue, and an all-interfaces SCP bind by default. Every behaviour here
exists to close one of those defects. Where this document must commit a detail the PRD left open, it makes a sensible
choice and records it in the assumptions section.

## Scope

In scope for v1, carrying forward the preserved command set named in the PRD (`docs/prd/go-radx-prd.md` §8) and
extending it with the new groups:

- The top-level DICOM commands: `echo`, `store`, `scp`, `dump`, `modify`, `organize`, `lookup`, `catalogue`.
- The `hl7` group: HL7 v2 messaging over MLLP (`send`, `listen`).
- The `dicomweb` group: WADO-RS, STOW-RS, and QIDO-RS clients (`wado`, `stow`, `qido`).
- The `convert` group: cross-standard conversion (`dicom-to-fhir`, `sr-to-fhir`, `oru-to-fhir`, `orm-to-fhir`,
  `adt-to-fhir`).
- The `serve` group: thin reference daemons for the DICOMweb and FHIR REST servers (`dicomweb`, `fhir`), the CLI entry
  point for the `server` package's embeddable roles (`docs/reference/servers.md`, PRD §7.2). The DIMSE Storage/QR SCP
  and the HL7 v2 MLLP listener keep their existing top-level homes (`scp` and `hl7 listen`).
- The global behaviour: output formats, environment configuration, exit codes, logging, and the honest-failure rules.

Out of scope for v1 (deferred, not designed against here): the SCP/server sides of MPPS and Storage Commitment (the v1
N-services are SCU-only, PRD §5.1); UPS-RS and the legacy WADO-URI interface; and the FHIR REST server role behind
`serve fhir`, which fails closed until that role lands. The DIMSE query/retrieve subcommands (`find`, `get`, `move`)
were the top-three dcmtk-parity gaps the Codex audit named (PRD §12); they are now wired against the `dimse`
query/retrieve iterators.

## Command tree

`radx` is a Kong-parsed tree. The DICOM verbs are top-level — the prototype nested them under a `dicom` group, which the
rewrite flattens because the workflow is DICOM-first and the extra path segment bought nothing. The cross-standard work
lives in named groups.

```text
radx <command> [flags]

  echo            Verify DICOM connectivity (C-ECHO)            ~ echoscu
  store           Send DICOM objects (C-STORE SCU)              ~ storescu
  find            Query a remote AE (C-FIND SCU)                ~ findscu
  get             Retrieve over the same association (C-GET)    ~ getscu
  move            Retrieve to a destination AE (C-MOVE)         ~ movescu
  scp             Run a Storage/Verification SCP                ~ storescp     [loopback by default]
  dump            Inspect DICOM file contents                   ~ dcmdump
  modify          Edit DICOM tags and regenerate UIDs           ~ dcmodify
  organize        Reorganise files by Study/Series/SOP UID      ~ dcmsort / a dcm2xml pipeline
  lookup          Resolve DICOM tag dictionary information       ~ dcmdump +dictionary lookups
  catalogue       Index and query a local DICOM catalogue       ~ (no direct dcmtk equivalent)

  hl7             HL7 v2 messaging over MLLP
    send          Send a message and read the ACK
    listen        Receive messages and reply with ACK/NAK
  dicomweb        DICOMweb clients
    wado          Retrieve via WADO-RS                          ~ a curl/dcm2json pipeline
    stow          Store via STOW-RS                             ~ a curl POST pipeline
    qido          Search via QIDO-RS                            ~ a curl GET pipeline
  convert         Cross-standard conversion
    dicom-to-fhir Build a FHIR ImagingStudy from DICOM
    sr-to-fhir    Map a DICOM SR to a DiagnosticReport
    oru-to-fhir   Map an HL7 v2 ORU to a DiagnosticReport
    orm-to-fhir   Map an HL7 v2 ORM to a ServiceRequest
    adt-to-fhir   Map an HL7 v2 ADT to a Patient / Encounter
  serve           Run a reference daemon (server package)
    dicomweb      Serve WADO-RS / STOW-RS / QIDO-RS              ~ (no direct dcmtk equivalent)
    fhir          Serve the FHIR REST API                        ~ (no direct dcmtk equivalent)
```

The `[loopback]` note is a stable part of the contract, not transient build state: `scp` binds loopback unless
explicitly told otherwise.

## Global flags and output contract

Every command shares one output contract. This is the direct fix for the prototype defects where a global `--format`
flag was advertised but ignored by most commands (RADX-021) and banners polluted stdout (RADX-004).

```text
Global flags (apply to every command):
  -f, --format=human   Output format: human | json | csv          [env: RADX_FORMAT]
  -o, --output=-       Write machine output to a file ("-" = stdout)
  -q, --quiet          Suppress the banner and progress on stderr
      --no-color       Disable ANSI colour in human output
  -l, --log-level=info Log verbosity: trace|debug|info|warn|error   [env: RADX_LOG_LEVEL]
      --log-format=text Log encoding: text | json                   [env: RADX_LOG_FORMAT]
  -V, --version        Print build information and exit
  -h, --help           Show contextual help and exit
```

The contract has three rules, binding on every command:

- **Machine output is clean.** Under `--format json` or `--format csv`, stdout contains only the requested payload —
  no banner, no build string, no progress bar, no log line. A JSON document or JSON Lines stream, or RFC 4180 CSV, and
  nothing else. A consumer can pipe stdout into `jq` or a CSV reader without filtering. This closes RADX-004 and
  RADX-021.
- **Diagnostics go to stderr.** The banner, progress indicators, structured logs, and warnings are written to stderr.
  In `human` format the banner is shown on an interactive terminal and suppressed when stdout is not a TTY or `--quiet`
  is set. In `json`/`csv` format the banner is always suppressed.
- **One canonical machine shape per command.** Each command defines a stable JSON object (or JSON Lines for streaming
  results such as query matches and per-file store results), with a stable `status` field, so automation can branch on
  outcome without parsing human text. CSV is offered for the tabular commands (`dump`, `lookup`, `catalogue`,
  `find`/`qido`) and is a typed "format not supported for this command" usage error elsewhere.

`human` is the default because the primary interactive user is a practitioner at a terminal. Scripts set
`--format json` or `RADX_FORMAT=json` once and get stable output everywhere.

### Logging

Logging uses `go.uber.org/zap`, configured once at the root and injected through the command context — not a package
global mutated with `log.SetDefault` (the prototype used `charmbracelet/log` with a global default, RADX-006; the
project standard is zap, PRD §9.10). Logs are structured and human-readable at once, render DICOM/HL7/FHIR concepts by
name rather than raw hex or numeric codes (PRD §8.2), and never carry PHI at the default verbosity (PRD §9.1). The
`--log-format json` mode emits one JSON object per line on stderr for ingestion.

## Environment configuration (12-factor)

Configuration follows the twelve-factor model: every operational setting is bindable from an environment variable so a
deployment configures `radx` without baking flags into scripts (PRD §9; the prototype was flags-only with no env
binding, RADX-005). Precedence is **flags > environment > defaults**. A v1 config file is not part of the contract; the
environment is the configuration surface.

| Variable | Binds | Example |
|----------|-------|---------|
| `RADX_FORMAT` | global `--format` | `json` |
| `RADX_LOG_LEVEL` | global `--log-level` | `debug` |
| `RADX_LOG_FORMAT` | global `--log-format` | `json` |
| `RADX_HOST` | default remote DICOM host for `echo`/`store`/`find`/`get`/`move` | `pacs.example.org` |
| `RADX_PORT` | default remote DICOM port | `11112` |
| `RADX_CALLED_AE` | default Called AE Title (the remote AE) | `ORTHANC` |
| `RADX_CALLING_AE` | default Calling AE Title (this client) | `RADX` |
| `RADX_TIMEOUT` | default operation timeout (Go duration) | `5m` |
| `RADX_MAX_PDU` | default maximum PDU length in bytes | `16382` |
| `RADX_BIND` | default `scp`/`listen` bind address | `127.0.0.1` |
| `RADX_DICOMWEB_URL` | default DICOMweb base URL | `https://dicom.example.org/dicom-web` |
| `RADX_MLLP_HOST` / `RADX_MLLP_PORT` | default HL7 v2 MLLP endpoint | `ris.example.org` / `2575` |
| `RADX_BEARER_TOKEN` | bearer token for DICOMweb / FHIR HTTP auth | (read from env, never logged) |

The maximum PDU length has one canonical default across the CLI: `16382` bytes, matching `dimse.WithMaxPDULength`'s
`dimse.MaxPDULength` default (the value the `dimse` layer negotiates). Every command that opens an association —
`echo`, `store`, `scp` — defaults `--max-pdu` to `16382`, and `RADX_MAX_PDU` binds the same value everywhere. The
`store` command alone accepts an explicit raise up to a `131072`-byte cap, because a high-throughput batch transfer can
benefit from larger P-DATA-TF PDUs; that is an operator override, not a different default. A value of `0` requests
"no maximum specified" (unlimited), as in the `dimse` layer.

Secrets — bearer tokens, TLS key passphrases — are read from the environment or a referenced file per 12-factor, are
never logged, and are never written to the catalogue (PRD §9.8). `AETitle`, `TransferSyntax`, and the host/port values
are validated and type-checked before any network call; an `AETitle` longer than 16 characters is a usage error, not a
silently truncated string (glossary naming rule 1).

## Exit-code taxonomy

A single, documented exit-code map lets operators distinguish failure classes without scraping text (the prototype
collapsed everything to `1`, RADX-022). Every command maps its typed errors onto this taxonomy (PRD §8):

| Code | Meaning | Example |
|------|---------|---------|
| `0` | Success | All work completed; no failures. |
| `1` | General failure | An otherwise-unclassified runtime error, or an unimplemented capability (fail-closed). |
| `2` | Usage error | Unknown flag, missing required argument, mutually exclusive flags, bad enum value. |
| `3` | DICOM / parse error | Malformed `.dcm`, HL7 v2, or FHIR input; truncated object; failed validation. |
| `4` | Network error | Connection refused, association rejected/aborted, DIMSE failure status, HTTP transport error. |
| `5` | File I/O error | Cannot read input, cannot create output, permission denied, disk full. |

Partial failure is non-zero by default (PRD §9.2). A batch `store`, `dump`, or `catalogue` index that processes some
inputs and fails on others exits non-zero; the per-item outcome is in the machine output, and an explicit
`--ignore-errors` opts in to a zero exit for exploratory use. There is no flag that hides a final failure: the
prototype's `store` exited zero unless `--fail-fast` was set (RADX-003), and the rewrite inverts that — any failed
transfer yields a non-zero exit, and the continue-on-error flag changes only whether processing stops, never the final
status.

## Honest-failure rules

Two rules the prototype broke (PRD §9.2) are load-bearing across every command:

- **Fail-closed on unimplemented or partial capability.** A path that cannot perform the requested mutation returns a
  typed error and writes nothing. A stub errors; it never no-ops and reports success. This is the direct fix for the
  prototype's `modify`, whose `applyModifications` logged "Would delete" / "Would insert" and returned nil while writing
  unchanged files (RADX-001, RADX-002). In v1, if `modify` cannot apply an edit it returns an error with exit `1` and
  leaves no output file; `serve fhir` returns "not implemented" and exits `1` until the FHIR server role lands.
- **Truncation and incompleteness are failures.** Parsers distinguish a clean record-boundary EOF from a short read
  mid-value and propagate `io.ErrUnexpectedEOF`; accepting a truncated object as complete is a defect. `dump` and
  `catalogue` that encounter an unparseable input record the failure and exit non-zero rather than logging and returning
  nil (RADX-012, RADX-013).

Both rules carry regression tests in the implementation (PRD §9.2). The result a script can rely on: `radx` never
reports success on failed work.

## DICOM commands

### echo — verify connectivity (C-ECHO)

Opens an association to a remote AE, sends a C-ECHO, and reports the verification status. The dcmtk equivalent is
`echoscu`.

```text
radx echo <host> <port> [flags]
      --called-ae=ANY-SCP    Called AE Title (the remote AE)   [env: RADX_CALLED_AE]
      --calling-ae=RADX      Calling AE Title (this client)    [env: RADX_CALLING_AE]
      --timeout=30s          Connection and operation timeout  [env: RADX_TIMEOUT]
      --max-pdu=16382        Maximum PDU length in bytes        [env: RADX_MAX_PDU]
      --tls                  Negotiate DIMSE-TLS (verifies the peer certificate)
```

Input: a host and port (positional). Output: in `human`, a one-line success or the named DIMSE status on failure; in
`json`, `{"status":"success","elapsed_ms":12,"called_ae":"ORTHANC"}`. A refused connection or rejected association exits
`4`; a non-success C-ECHO status exits `4` with the status rendered by name (PRD §8.2), never as raw hex.

### store — send objects (C-STORE SCU)

Stores one or more DICOM objects to a remote Storage SCP over a negotiated association. The dcmtk equivalent is
`storescu`.

```text
radx store <path>... [flags]
      --host=...             Remote host (required)             [env: RADX_HOST]
      --port=11112           Remote port                        [env: RADX_PORT]
      --called-ae=ANY-SCP    Called AE Title                    [env: RADX_CALLED_AE]
      --calling-ae=RADX      Calling AE Title                   [env: RADX_CALLING_AE]
  -R, --recursive            Descend into directories
      --timeout=5m           Operation timeout                  [env: RADX_TIMEOUT]
      --max-pdu=16382        Maximum PDU length in bytes (cap 131072) [env: RADX_MAX_PDU]
      --workers=4            Concurrent worker associations (1-128)
      --transcode-to=""      Transcode to this transfer syntax before sending (default: send as stored)
      --continue-on-error    Keep processing after a failed object (final exit still non-zero)
```

Input: file or directory paths (with `-R` to recurse). Output: a per-object result. In `json` the command emits JSON
Lines, one object per file (`{"file":"a.dcm","sop_instance_uid":"…","status":"success"}` …) followed by a summary line;
in `human`, a progress bar on stderr and a final tally.

Three behaviours diverge deliberately from the prototype. First, transcoding is **off by default and is an explicit
opt-in**: objects are sent as stored, matching `storescu`, and `--transcode-to` names the target transfer syntax rather
than the prototype's silent default-on transcode to JPEG 2000 (RADX-011) — medical-image fidelity is not altered
without an explicit instruction. Transcoding (the encode side) is available only in a build that includes the optional
CGo codec tag and only for the syntaxes a codec exists for (RLE and JPEG 2000 lossless first, per `dicom.md` and
`conformance/dicom.md`); a `--transcode-to` request against a syntax this build cannot encode is a usage error, not a
silent passthrough. Decoding compressed input for storage is always available; only encode/transcode is gated.
Second, any failed transfer makes the command exit non-zero; `--continue-on-error` changes only whether the batch stops,
not the final status (RADX-003). Third, each worker owns its association lifecycle, so a reconnect replaces the worker's
client cleanly rather than leaking the original (RADX-009). A study with files larger than the PDU is streamed in PDV
fragments; there is no rate-limit window that turns a valid large object into a failure (RADX-010).

### find / get / move — query and retrieve (C-FIND / C-GET / C-MOVE)

The query/retrieve trio, equivalent to dcmtk's `findscu`, `getscu`, and `movescu`. These are the top-three parity gaps
the Codex audit named (PRD §12), wired against the `dimse` query/retrieve iterators. `find` streams one match per
result; `get` retrieves over the same association, acting as the Storage SCP for the sub-operation C-STOREs and writing
each received instance under `--output-dir`; `move` retrieves to a named `--move-destination` AE. A non-success terminal
status (a partial-failure retrieve, a refused query, a "Move Destination Unknown") is reported faithfully and exits `4`,
never laundered into success.

```text
radx find [flags]    --level=STUDY  --match key=value...    ~ findscu
radx get  [flags]    --level=SERIES --match key=value...    ~ getscu
radx move [flags]    --move-destination=AE ...              ~ movescu
```

`--level` takes a typed `QueryLevel` (`PATIENT|STUDY|SERIES|IMAGE`, glossary), `--match` builds the identifier dataset,
and `find` returns one match per JSON Line, mirroring the streaming multi-response contract of the underlying
`Association.Find` iterator (PRD §8.1). `move` requires `--move-destination`, a named `AETitle`.

### scp — receive objects (Storage / Verification SCP)

Runs a Storage SCP that accepts C-STORE (and, by default, C-ECHO), writing received objects to a directory. The dcmtk
equivalent is `storescp`.

```text
radx scp [flags]
      --bind=127.0.0.1       Listen address (loopback by default)  [env: RADX_BIND]
      --port=11112           Listen port
      --aet=RADX-SCP         This SCP's AE Title
      --output-dir=./dicom-received   Where to write received objects
      --organize             Lay out by Study/Series/SOP UID (default on)
      --accept-echo          Accept C-ECHO (default on)
      --max-pdu=16382        Maximum PDU length in bytes        [env: RADX_MAX_PDU]
      --max-conns=10         Maximum concurrent associations
      --tls-cert=/--tls-key= Serve over DIMSE-TLS (mutual TLS optional)
```

Input: incoming associations on the bound address. Output: received objects on disk plus a per-object log line on
stderr.

Two safe defaults are non-negotiable. The SCP **binds to loopback** (`127.0.0.1`) by default; a non-loopback bind
(`--bind 0.0.0.0`) is an explicit, logged opt-in (the prototype always bound all interfaces, RADX-017; PRD §9.1).
Output paths are derived from received UIDs only after validating each UID against DICOM UID syntax and rejecting path
separators, and objects are written with exclusive-create semantics under a documented duplicate-instance policy, so a
malformed sender-controlled identifier cannot escape the output directory or silently overwrite a stored instance
(RADX-016). The SCP shuts down gracefully on `SIGINT`/`SIGTERM`, draining in-flight associations; an interrupted
shutdown is reported, not masked.

### dump — inspect file contents

Parses DICOM files and prints their elements with tags rendered by keyword and `(gggg,eeee)`, VRs by name, and UIDs by
their registered names. The dcmtk equivalent is `dcmdump`.

```text
radx dump <path>... [flags]
  -R, --recursive           Descend into directories
  -t, --tag=<tag>...        Show only these tags ((GGGG,EEEE), GGGGEEEE, or keyword)
  -g, --group=<group>...    Show only these groups (GGGG or a group name, e.g. "patient")
      --process-pixel-data  Parse pixel-data elements (off by default)
      --redact              Mask PHI-sensitive element values as [redacted]
      --ignore-errors       Exit 0 even if some inputs failed (exploratory)
```

Input: file or directory paths (with `-R` to recurse). Output: in `human`, an indented element listing per file; in
`json`, a tag-keyed object per file (a single file is one object, multiple files are a JSON Lines stream so the output
stays parseable); in `csv`, one row per element. Element values are shown by default — a dump is an explicit,
authorized inspection of a file you already hold (the dcmtk `dcmdump` posture); the no-PHI rule targets ambient logging,
not a command you deliberately ran on a local file. `--redact` masks the values of PHI-sensitive elements (the PS3.15
confidentiality attributes) to `[redacted]` so you can share a listing; the structure is always shown. Pixel-data values
stay out of the listing unless `--process-pixel-data` is set. If any input fails to parse, `dump` exits `3` and the
per-file machine output flags which file failed; the prototype logged and returned nil (RADX-012). `--ignore-errors`
opts into a zero exit.

### modify — edit tags and regenerate UIDs

Edits elements in DICOM files: insert or update a tag, delete a tag, or regenerate Study/Series/SOP Instance UIDs. The
dcmtk equivalent is `dcmodify`.

```text
radx modify <path>... [flags]
      --output-dir=...       Write modified files here (required unless --in-place)
  -i, --in-place             Overwrite the originals in place
  -I, --insert=<tag=value>... Insert or update a tag ((GGGG,EEEE)=value)
  -D, --delete=<tag>...      Delete a tag ((GGGG,EEEE))
  -R, --recursive            Descend into directories
      --regenerate-study-uid    New Study Instance UID  (0020,000D)
      --regenerate-series-uid   New Series Instance UID (0020,000E)
      --regenerate-instance-uid New SOP Instance UID    (0008,0018)
      --regenerate-all-uids     Regenerate all three, preserving the reference graph
```

Input: files to edit. Output: modified files at `--output-dir` (or in place), and a per-file summary of the edits
applied.

This command really mutates the dataset — the single most important correction in the CLI. The prototype's `modify` was
a no-op that logged "Would insert" and wrote unchanged files while reporting success (RADX-001), and its UID
regeneration generated and logged new UIDs without ever writing them (RADX-002). In v1, each `--insert`/`--delete` and
each `--regenerate-*` is applied to the in-memory `DataSet`, the file is re-encoded and the round-trip is verified, and
the written file is what the flags asked for. UID regeneration writes the new `UID` to the correct element, validates
its syntax, and remaps consistently so cross-references stay intact. If any requested edit cannot be applied, `modify`
returns an error, exits `1`, and writes no output for that file (fail-closed). Inserted tag values are PHI and are never
logged at default verbosity (RADX-007).

### organize — reorganise by UID structure

Lays out a flat or mixed directory of DICOM files into a `Study/Series/SOP` hierarchy by reading each file's UIDs. The
closest dcmtk path is a `dcmsort` or `dcm2xml`-driven script; there is no single equivalent tool.

```text
radx organize <dir> [flags]
      --output-dir=...   Destination root (required)
  -R, --recursive        Descend into the source (default on)
      --move             Move instead of copy
      --dry-run          Report the planned layout without touching files
      --overwrite        Allow overwriting an existing destination file
```

Input: a source directory. Output: an organised tree at `--output-dir`, or, with `--dry-run`, the planned moves. UIDs
are validated and sanitised before they become path segments (the prototype's `sanitizeUID` was a no-op, RADX-018), and
files are written with exclusive-create semantics unless `--overwrite` is set, so a duplicate or malformed UID cannot
silently truncate an existing file. `--dry-run` performs no I/O.

### lookup — resolve dictionary information

Resolves a DICOM tag, keyword, or free-text fragment against the standard data dictionary and prints authoritative
information. There is no standalone dcmtk tool; `dcmdump` ships the dictionaries this command queries directly.

```text
radx lookup <query>... [flags]
```

Input: a tag (`(GGGG,EEEE)` or `GGGGEEEE`), a keyword (`PatientName`), or a search fragment. Output: the canonical name,
keyword, VR, value multiplicity, retired status, and tag type. Lookup is wired to the generated dictionary (~5,189
standard entries) with repeating-group resolution, not the prototype's hand-curated partial list with heuristic VR
inference (RADX-019). In `csv`/`json` it emits one record per match.

### catalogue — index and query a local catalogue

Indexes a directory of DICOM files into a local SQLite catalogue and queries it by tag filters or read-only SQL. There
is no dcmtk equivalent; this is a go-radx convenience for triage and inventory.

```text
radx catalogue [<dir>] [flags]
  -d, --database=dicom-catalogue.db   Catalogue file path
      --rebuild              Drop and rebuild from scratch
  -R, --recursive            Descend into the source (default on)
  -q, --query=<key=value>... Tag filters (e.g. --query modality=CR)
      --sql=<select>         Read-only SQL (SELECT only)
  -m, --mode=table           SQL result rendering: table|csv|json|jsonl|list|markdown
      --schema               Print the catalogue schema and exit
      --confirm-phi          Acknowledge that the catalogue stores PHI
      --redact               Index structural fields only; omit PHI columns
      --ignore-errors        Exit 0 even if some files failed to index
```

Input: a directory to index, or an existing catalogue to query. Output: the indexed catalogue, query results, or the
schema.

The catalogue stores patient identifiers, so it is a **PHI-bearing convenience store and is opt-in** (PRD §9.1). The
prototype created a PHI database by default with no warning, no file-permission hardening, and logged raw SQL and query
filters (RADX-007, RADX-008). In v1: creating a catalogue with PHI columns requires `--confirm-phi`; `--redact` indexes
only structural fields (UIDs, modality, transfer syntax, counts) and omits names, IDs, birth dates, and accession
numbers; the database file is created with restrictive permissions (`0600`); and neither SQL text nor filter values
are logged at default verbosity. The `--sql` input is validated to be a non-empty `SELECT` before execution — empty or
whitespace SQL is a clean usage error, not a panic (RADX-014) — and every row-iteration loop checks its error before
reporting success (RADX-015). Indexing that fails on some files exits non-zero unless `--ignore-errors` is set
(RADX-013).

## hl7 — HL7 v2 over MLLP

The `hl7` group sends and receives HL7 v2.x messages over MLLP (the `0x0B … 0x1C 0x0D` framing, glossary). There is no
dcmtk equivalent; this is the HL7 surface the Codex audit found entirely absent (RADX-020).

```text
radx hl7 send --host=... --port=2575 [<file>|-] [flags]
      --timeout=30s          Read/write timeout
      --max-frame=1MiB       Maximum MLLP frame length (hostile-input cap)

radx hl7 listen --bind=127.0.0.1 --port=2575 [flags]
      --ack=AA               Default acknowledgment code (AA|AE|AR)
      --max-frame=1MiB       Maximum MLLP frame length
```

`send` reads a message from a file or stdin, frames it over MLLP, and prints the parsed ACK — including the `MSA-1`
acknowledgment code rendered by name (there is no "NAK" message; a negative ack is an ACK with `MSA-1` = `AE`/`AR`,
glossary). `listen` binds loopback by default (PRD §9.1), receives messages, and replies with an ACK built from the
inbound `MSH`. Both enforce a maximum frame length so a hostile peer cannot exhaust memory (PRD §9.3), and both honour
`context.Context` cancellation on `SIGINT`. Message content is PHI and is not logged at default verbosity.

## dicomweb — WADO-RS, STOW-RS, QIDO-RS clients

The `dicomweb` group is the RESTful counterpart to DIMSE query/retrieve and store. Each subcommand is the firewall-
friendly equivalent of a hand-rolled `curl` pipeline against a DICOMweb server.

```text
radx dicomweb wado --url=... --study=<uid> [--series=<uid>] [--instance=<uid>] [flags]
      --metadata             Retrieve metadata (application/dicom+json) instead of objects
      --output-dir=.         Where to write retrieved objects

radx dicomweb stow --url=... <path>... [flags]
      --study=<uid>          Target Study Instance UID (optional)

radx dicomweb qido --url=... --level=studies [--match key=value...] [flags]
      --limit=0              Maximum matches (0 = server default)
      --include=<tag>...     Additional return tags
```

`--url` is the DICOMweb base (`RADX_DICOMWEB_URL`); a bearer token comes from `RADX_BEARER_TOKEN` and is never logged.
`wado` retrieves objects (or, with `--metadata`, `application/dicom+json`) addressed by the
`/studies/{uid}/series/{uid}/instances/{uid}` resource path. `stow` POSTs instances as `multipart/related`. `qido`
searches and emits one match per JSON Line in `json` mode. HTTP servers are reached over TLS 1.2+ with peer
verification by default; `InsecureSkipVerify` is never set outside an explicitly flagged test mode (PRD §9.7). The DICOM
JSON these commands consume is the PS3.18 Annex F tag-keyed schema, never cross-fed to the FHIR JSON serializer
(glossary).

## convert — cross-standard conversion

The `convert` group drives the `convert/` package's named conversions, each using both standards' canonical nouns so the
direction is unambiguous (glossary naming rule 3). There is no dcmtk equivalent; cross-standard mapping is unique to
go-radx.

```text
radx convert dicom-to-fhir <path>... [--release=R5] [--output=-]    -> ImagingStudy
radx convert sr-to-fhir    <path>...                                -> DiagnosticReport + Observation
radx convert oru-to-fhir   [<file>|-]                               -> DiagnosticReport + Observation
radx convert orm-to-fhir   [<file>|-]                               -> ServiceRequest
radx convert adt-to-fhir   [<file>|-] [--as=patient|encounter]      -> Patient / Encounter
```

`--release` selects FHIR R4 (4.0.1) or R5 (5.0.0); the default is `R5` (PRD §5.3). The release is not cosmetic: it picks
the release-explicit converter the `convert` package exposes (the `…R5` form, e.g. `convert.DICOMToImagingStudyR5`,
returning a `*r5.ImagingStudy`, or its `…R4` twin returning a `*r4.ImagingStudy`; see `docs/reference/convert.md`). The
result is serialised with `encoding/json` and written as a FHIR resource to stdout (or `--output`). Each conversion
returns the `convert` package's conversion report; a conversion that cannot faithfully map a required element returns
an error and exits `3` rather than emitting a lossy resource (fail-closed, PRD §9.2). DICOM UIDs map to FHIR
`Identifier` values (`urn:dicom:uid` / `urn:oid`), never to a `Reference.reference` (glossary).

## serve — reference daemons (DICOMweb, FHIR REST)

The `serve` group runs the thin reference daemons that wrap the `server` package's embeddable roles, giving the
DICOMweb and FHIR REST servers a CLI entry point alongside the DIMSE SCP (`scp`) and the HL7 v2 MLLP listener
(`hl7 listen`). Each subcommand wires the shared backends — a filesystem object store and a SQLite catalogue — binds to
loopback by default, and uses no-authentication (`AllowAll`) on that loopback bind, exactly as
`docs/reference/servers.md` describes. There is no dcmtk equivalent; these are go-radx reference servers.

```text
radx serve dicomweb [flags]
      --bind=127.0.0.1       Listen address (loopback by default)  [env: RADX_BIND]
      --port=8042            Listen port
      --base-path=/dicom-web DICOMweb base path
      --object-store=...     Filesystem object-store root (required)
      --catalogue=...        SQLite catalogue path (required; PHI store, never a default path)
      --max-request-bytes=... Request body cap (hostile-input limit)
      --tls-cert=/--tls-key= Serve over TLS (peer verification on by default)

radx serve fhir [flags]
      --bind=127.0.0.1       Listen address (loopback by default)  [env: RADX_BIND]
      --port=8080            Listen port
      --base-path=/fhir      FHIR REST base path
      --repository=...       Repository backend selector / DSN (required)
      --tls-cert=/--tls-key= Serve over TLS (peer verification on by default)
```

Both daemons follow the same safe defaults as the other servers. They **bind to loopback** (`127.0.0.1`); a
non-loopback bind (`--bind 0.0.0.0`) is an explicit, logged opt-in that requires authentication to be configured
(servers.md "Bind policy"). The SQLite catalogue holds PHI, so the `dicomweb` daemon requires an explicit `--catalogue`
path and never creates a PHI-bearing database at a default path. Each daemon shuts down gracefully on
`SIGINT`/`SIGTERM`, draining in-flight requests within the configured timeout. For embedding these roles in your own
process, or running several together behind one composition root, use the `server` package directly
(`docs/reference/servers.md`).

## Worked examples

### Verify a PACS, scripted

```bash
export RADX_CALLED_AE=ORTHANC RADX_CALLING_AE=RADX RADX_FORMAT=json
radx echo orthanc.local 4242 | jq -e '.status == "success"'
# exit 0 on a successful C-ECHO; exit 4 on a refused or rejected association
```

### Send a study and fail the build if any object failed

```bash
radx store ./study --host pacs.local --port 11112 --called-ae PACS \
  --format json --continue-on-error > results.jsonl
# results.jsonl is one JSON object per object plus a summary line; clean stdout.
# A single failed C-STORE makes radx exit non-zero, so CI catches partial failure.
if [ $? -ne 0 ]; then echo "some objects failed; see results.jsonl" >&2; fi
```

### Receive on loopback, organised by UID

```bash
radx scp --port 11112 --aet RADX-SCP --output-dir ./received --organize
# Binds 127.0.0.1 by default. To accept from the network you must opt in:
radx scp --bind 0.0.0.0 --port 11112 --aet RADX-SCP --output-dir ./received
```

### Serve a DICOMweb reference daemon on loopback

```bash
radx serve dicomweb --port 8042 --object-store ./objects --catalogue ./catalogue.db
# Binds 127.0.0.1 by default; --catalogue is explicit because it stores PHI.
# To expose it on the network you must opt in and configure authentication:
radx serve dicomweb --bind 0.0.0.0 --port 8042 \
  --object-store ./objects --catalogue ./catalogue.db --tls-cert server.crt --tls-key server.key
```

### De-identify-grade UID re-keying that actually writes

```bash
radx modify ./study/*.dcm --output-dir ./rekeyed --regenerate-all-uids
# New Study/Series/SOP UIDs are written and the reference graph is preserved.
# If any file cannot be re-keyed, that file produces no output and radx exits 1.
```

### Inspect a file as machine data

```bash
radx dump scan.dcm --format json | jq '.["0010,0010"]'   # tag-keyed, banner-free stdout
radx lookup PatientName --format csv                      # authoritative dictionary record
```

## Behaviour and error model

The CLI maps the library's typed errors onto the exit-code taxonomy and renders them through zap with concepts named,
not numbered (PRD §8.2):

- A Kong parse failure — unknown flag, missing required argument, bad enum, mutually exclusive flags — exits `2` and
  prints usage to stderr.
- A malformed `.dcm`, HL7 v2 message, or FHIR document, including a truncated object, exits `3`; the library
  distinguishes a clean EOF from `io.ErrUnexpectedEOF` and the CLI surfaces the latter as a parse failure (PRD §9.2).
- A refused connection, rejected or aborted association, non-success DIMSE status, or HTTP transport error exits `4`,
  with the DIMSE `Status` rendered by name and class.
- An input that cannot be read or an output that cannot be created exits `5`.
- An unimplemented capability (`serve fhir`, any not-yet-built path) exits `1` with a "not implemented" error and
  writes nothing (fail-closed).
- Partial failure across a batch exits non-zero unless `--ignore-errors` is explicitly set; no flag converts a final
  failure into success.

No command panics on hostile input: malformed files, oversized MLLP frames, and unbounded HTTP bodies hit configured
caps and return typed "limit exceeded" errors (PRD §9.3). No command logs PHI at default verbosity, binds non-loopback
without explicit opt-in, or writes secrets to logs or the catalogue (PRD §9.1, §9.8).

## Conformance scope and limits

The `radx` CLI is conformant to the command surface and behaviour declared here; it does not wrap every library
capability. For v1:

- **Commands.** `echo`, `store`, `scp`, `dump`, `modify`, `organize`, `lookup`, `catalogue`, plus the `hl7`,
  `dicomweb`, `convert`, and `serve` groups, with the shared output contract, env binding, exit-code taxonomy, and
  honest-failure rules on every command.
- **Reference servers.** The DIMSE Storage/QR SCP (`scp`) and HL7 v2 MLLP listener (`hl7 listen`) run as top-level
  commands; the DICOMweb and FHIR REST reference daemons run under `serve` (`serve dicomweb`, `serve fhir`). All four
  bind loopback by default and wrap the `server` package's embeddable roles (`docs/reference/servers.md`).
- **DIMSE.** C-ECHO and C-STORE (SCU), C-FIND/C-GET/C-MOVE (`find`/`get`/`move`), and a loopback
  Storage/Verification SCP, all wired against the `dimse` layer.
- **Transport security.** DIMSE-TLS and DICOMweb/FHIR HTTP default to TLS 1.2+ with peer verification; mutual TLS is a
  documented option; `InsecureSkipVerify` exists only behind an explicit test flag (PRD §9.7).
- **PHI defaults.** No PHI in stdout, stderr, logs, or telemetry at default verbosity; servers bind loopback; the
  catalogue's PHI columns are opt-in with a redacted mode. These are verified by the CI basic-safety checks — the
  PHI-default sanity test and the bind-default test (PRD §11.2).

The CLI is a general-purpose tool built to good engineering standards; it makes no medical-device certification claim
and imposes no compliance machinery (PRD §9.5). PHI governance — encryption at rest, retention, access control, audit —
belongs to the operator who deploys it.

## Assumptions

Details this document committed that the PRD left open, for review:

- **Top-level DICOM verbs.** The prototype nested the DICOM commands under a `dicom` group (`radx dicom store`); this
  reference flattens them to top level (`radx store`) to match the PRD's flat command list (§8) and the DICOM-first
  workflow. If a future HL7-or-FHIR-heavy surface needs the namespace back, this is the reversible choice.
- **`RADX_*` variable names.** The PRD mandates 12-factor env config and gives example names (`RADX_DICOM_HOST`,
  `RADX_CALLED_AE`, `RADX_LOG_LEVEL`); the full table here is a concrete, consistent choice. `RADX_HOST` is used rather
  than the example `RADX_DICOM_HOST` for brevity, since the CLI is DICOM-first; the HL7 endpoint is namespaced
  (`RADX_MLLP_*`) to avoid colliding with the DICOM host.
- **Streaming machine output.** Commands with per-item results (`store`, `find`, `qido`, `hl7 send` batches) emit JSON
  Lines rather than a single JSON array, so a consumer can process results as they arrive and a long run is observable.
  Single-result commands (`echo`, `convert`) emit one JSON object.
- **Default output format.** `human` is the default; scripts opt into `json`/`csv`. The alternative — defaulting to a
  machine format — was rejected because the primary interactive user is a practitioner.
- **`store --transcode-to` semantics.** Transcoding is off by default (matching `storescu`); `--transcode-to` takes an
  explicit `TransferSyntax`. The prototype's boolean default-on `--transcode` is not carried forward.
- **Catalogue PHI gate.** Indexing PHI columns requires `--confirm-phi` and a `--redact` structural-only mode is
  offered; the database is `0600`. The PRD requires the store be opt-in with a redacted mode but names no flags.
- **`convert` default release.** `--release` defaults to `R5`, matching the PRD's R5-first sequencing (§5.3, M6a); R4 is
  available via `--release R4`.

## See also

- `docs/prd/go-radx-prd.md` — the umbrella PRD (command floor and exit codes §8, design principles §8.2, NFRs §9,
  milestones §13).
- `UBIQUITOUS_LANGUAGE.md` — canonical Go names (`AETitle`, `TransferSyntax`, `QueryLevel`, `Status`) and the
  cross-standard collision rules.
- The `dicom` reference — the data model and Part 10 I/O behind `dump`, `modify`, `organize`, `store`, `scp`.
- The `dimse` reference — associations, presentation contexts, and DIMSE services behind `echo`, `store`, `scp`, and
  the `find`/`get`/`move` verbs.
- The `dicomweb` reference — the WADO-RS / STOW-RS / QIDO-RS clients behind `radx dicomweb`.
- The `hl7v2` reference — the MLLP client/server behind `radx hl7`.
- The `convert` reference — the cross-standard conversions behind `radx convert`.
- The `servers` reference — the embeddable server roles and reference daemons behind `radx scp`, `radx hl7 listen`,
  and `radx serve`.
