# CLI and server conformance statement

| Field | Value |
|-------|-------|
| Surface | `radx` CLI (`cmd/radx`) and the embeddable server entry points |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | 1 |
| Status | Published |
| Scope authority | This document is the single source of truth for CLI/server scope (PRD §6.1) |

This document declares the conformance scope of the operator-facing surface: the `radx` command groups, their flags
and exit codes, and the behaviour of the DICOM, DICOMweb, MLLP, and FHIR servers when invoked through the CLI. The
command reference is `docs/reference/cli.md` and the server design is `docs/reference/servers.md`; where this statement
and the reference docs disagree on scope, this statement wins. The behaviour below is shipped, not planned.

## Command surface

The `radx` command tree ships today, parsed by Kong, with every command group registered so `radx --help` lists the
full surface. The DICOM command groups are `echo` (C-ECHO), `store` (C-STORE SCU), `find` (C-FIND SCU), `get` (C-GET),
`move` (C-MOVE), `scp` (Storage/Verification SCP), `dump` (inspect a Part 10 file), `modify` (edit tags, regenerate
UIDs), `organize` (reorganise files by Study/Series/SOP UID), `lookup` (resolve tag dictionary information), and
`catalogue` (index and query a local DICOM catalogue). The cross-standard groups are `hl7` (HL7 v2 over MLLP),
`dicomweb` (DICOMweb clients), `convert` (cross-standard conversion), and `serve` (run a reference daemon over the
`server` package). The flag contract and per-command exit codes are documented in `docs/reference/cli.md`.

Every command fails closed and never reports a false success: a command that cannot perform its requested operation
returns a typed error that classifies to a non-zero exit code and writes nothing to stdout, rather than no-opping and
exiting 0 (the prototype defect the honest-failure rules — RADX-001/002 — exist to prevent). No registered stub
remains: `radx serve fhir` is wired end to end over the FHIR server role (`server.NewFHIRRole` over the in-memory
development repository), the DICOMweb serve daemon is wired (`radx serve dicomweb`), and the DIMSE SCP serves through
`radx scp`. The honest-failure rule still binds any future committed-but-unbuilt subcommand.

## Embeddable server composition layer

The `server` package is the embeddable composition layer the CLI will eventually wrap. It ships ahead of the CLI as a
library surface; its full public-API contract is `docs/reference/servers.md`. What ships today:

- **`server.Daemon` composition root.** Constructed with `server.New(opts ...Option)`, it owns the shared logger
  (`WithLogger`, default no-op), the OpenTelemetry tracer and meter providers (`WithTracerProvider`,
  `WithMeterProvider`, both defaulting to the no-op providers that export nowhere until an operator wires an exporter),
  the listener policy, and the graceful-shutdown lifecycle. `Run(ctx)` starts every mounted role and blocks until the
  context is cancelled (the `SIGINT`/`SIGTERM` path the CLI will wire) or `Shutdown(ctx)` is called, then drains every
  role within `WithShutdownTimeout` (default 30 seconds). A role that does not drain in time yields a role-naming
  `ErrShutdownTimeout` from `Run`, an honest report that the stop was not clean — never a silent success.
- **Four pluggable backends.** `ObjectStore` (the binary object plane behind C-STORE/C-GET/C-MOVE and STOW-RS/WADO-RS),
  `Catalogue` (the queryable metadata plane behind C-FIND and QIDO-RS), `WorklistSource` (the Modality Worklist plane),
  and `Authenticator` (the identity plane), each segregated so a deployment implements only what it serves.
- **Four server roles.** A DIMSE SCP role (`NewDIMSERole`, wrapping `dimse.Server`, storing via `ObjectStore` and
  indexing via `Catalogue`, with an optional Modality Worklist SCP fed by a `WorklistSource`), a DICOMweb role
  (`NewDICOMwebRole`, wrapping `dicomweb.Server` over the same backends and mounting the full WADO-RS retrieval
  surface — instance, study, series, metadata, frames, and bulkdata — alongside QIDO-RS search and STOW-RS storage),
  an HL7 v2 MLLP role (`NewMLLPRole`, wrapping
  `hl7v2.Server`), and a FHIR REST role (`NewFHIRRole`, over a `Repository` bound to one FHIR release; see
  [FHIR REST client and server role](#fhir-rest-client-and-server-role) below). Each applies the daemon's shared bind,
  TLS, and observability policy uniformly.
- **Default backends.** `server.FileStore(root)` persists each object as a Part 10 file in a study/series/instance
  layout, treating UIDs as untrusted input (every path component is validated as a conformant DICOM UID, so a
  traversal-style identifier is rejected before any path is built). `server.SQLiteCatalogue(ctx, dbPath, ...)` indexes
  the queryable attributes in a SQLite database via the pure-Go `modernc.org/sqlite` driver, so the default build stays
  cgo-free. The catalogue path is **required and never defaulted**, because the catalogue holds PHI; a redacted mode
  (`WithRedaction`) hashes direct identifiers for non-clinical use.

### Bind policy and the PHI default

Every role binds to **loopback only** (`127.0.0.1`) unless the operator passes `WithBind` (the CLI's future `--bind`).
A non-loopback bind without an explicit `Authenticator` fails closed: `server.New` returns `ErrInsecureBind` rather than
silently exposing an unauthenticated server. Binding wide open with no authentication is reachable only by passing
`WithAuthenticator(server.AllowAll())` explicitly — a visible, reviewable choice. The no-PHI-by-default behaviour is
test-enforced by a server-package PHI sweep that mirrors the `internal/phisweep` harness: it drives a runnable daemon
over a sentinel-bearing object at default verbosity through a real C-STORE and asserts no sentinel surfaces in stdout,
stderr, returned errors, or the structured log.

### FHIR REST client and server role

The FHIR REST client and the FHIR REST server role now ship as library surfaces. Both are fixed to one FHIR release per
instance, because an R4 resource and the corresponding R5 resource are distinct Go types in distinct packages
(`r4.Patient` vs `r5.Patient`); a client targets one release at construction and a role serves one release at mount.

The **client** (`fhir/rest`) is a type-safe FHIR RESTful API client over the generated models, constructed with
`rest.NewClient(release, baseURL, ...)`. It implements `read`, `vread`, `create`, `update`, `patch`, `delete`,
`history`, type-level `search` (with typed parameters, modifiers, single-level chaining, `_include`/`_revinclude`, and
`Bundle.link` `next`/`previous` paging), `transaction`/`batch` submission, conditional create/update
(`If-None-Exist`/`If-Match`) with ETag concurrency, and `CapabilityStatement` negotiation. It sends and accepts
`application/fhir+json` only. A non-2xx FHIR response whose body is an `OperationOutcome` is surfaced as a typed
`*rest.OperationOutcomeError` the caller classifies by issue severity and by an `errors.Is`-comparable sentinel
(`ErrNotFound`, `ErrConflict`, `ErrUnprocessable`, `ErrUnauthorized`, `ErrUnsupported`), aligning with the `fhir`
package's `OperationOutcome` error model and `exitcode.FromOperationOutcome`. Authentication is a pluggable transport
concern (a bearer token via `WithBearerToken`, or any scheme via `WithRoundTripper`), origin-scoped so a credential is
never sent cross-origin, mirroring the DICOMweb client's auth seam. **SMART on FHIR is deferred**: a SMART access token
is supplied through the bearer/round-tripper seam, but the SMART authorization flow itself is not implemented.

The **server role** (`server.NewFHIRRole`, mounted with `server.WithFHIR`) serves the conformance subset over a
pluggable `server.Repository`: `read`, `vread`, `history-instance`, `create`, `search-type`, `transaction`, and the
`$validate` operation over the workflow resource set (`Patient`, `Encounter`, `ServiceRequest`, `ImagingStudy`,
`DiagnosticReport`, `Observation`), as `application/fhir+json`. The repository versions every create
(`meta.versionId`/`meta.lastUpdated`); read, vread, and create responses carry `ETag: W/"versionId"` and
`Last-Modified`, a create's `Location` is the versioned `[base]/[type]/[id]/_history/[vid]` (FHIR R5
`http.html#create`) and a transaction response entry carries the same versioned `response.location` plus
`response.etag` (`http.html#transaction-response`), a vread of an unknown version is a `404` and of a deleted
version a `410`, the history Bundle
carries per-version `entry.request`/`entry.response` per FHIR R5 `http.html#history` and the resource's absolute
`fullUrl` (`[base]/[type]/[id]`, identical for every version and present on a deleted version's resource-less entry,
R5 `bundle.html`), history honours `_count` as a cap (at most the newest `_count` entries; `total` still reports the
full version count; paging links over history are deferred), and a write's `If-Match`
precondition is evaluated against the current version (`412` on a stale version, `http.html#concurrency`). It
validates inbound resources with the release validator (a resource with error-severity issues is rejected `422`;
`POST [type]/$validate` runs the same validator and returns the findings as an `OperationOutcome` without
persisting), returns a release `OperationOutcome` for every error (a `404` read miss, a `400` malformed body, a
`405`/`501` deferred interaction — `update`/`delete`/`patch` are answered with a `501` `OperationOutcome`, never a
silent no-op), and serves a `CapabilityStatement` at `[base]/metadata` advertising exactly the supported
interactions. A conditional create (FHIR R5 `http.html` conditional create) fails closed: a create carrying
`If-None-Exist` — on the direct POST or as a transaction entry's `request.ifNoneExist` — is rejected `400` with a
`not-supported` `OperationOutcome` and persists nothing, never silently ignored into a duplicate; the matching
semantics are deferred to the search work. The version store is interaction-shaped (one record per version, newest first), so the deferred
update/patch/delete and conditional writes extend it by appending versions rather than reshaping it. The release is
fixed with `WithFHIRRelease` (default R5); to serve both releases from one process, mount two roles on different base
paths (for example `/fhir/r4` and `/fhir/r5`). A default in-memory `server.MemoryRepository` makes the role runnable
out of the box; a production deployment supplies its own `Repository`. The role plugs into the `Daemon` exactly like
the others: it honours the loopback-default bind, and a non-loopback FHIR bind without an `Authenticator` fails
closed with `ErrInsecureBind`.

## Structured logging and PHI policy

Structured logging uses [`go.uber.org/zap`](https://pkg.go.dev/go.uber.org/zap) and lives in the `logging` package at
the module root. The CLI and the embeddable servers, once wired, will obtain their logger from this package and honour
the policy below; the package itself is implemented today even though the operator-facing CLI is not.

The contract has two parts, both following the PRD's observability and PHI rules (PRD §9.10, §9.1):

- **Context-injected logger, no global.** The logger is constructed once at the composition root via
  `logging.NewLogger`. It flows through call chains on `context.Context`: `logging.WithContext(ctx, logger)` attaches it
  and `logging.FromContext(ctx)` retrieves it. There is no package-global logger. `FromContext` returns a safe no-op
  logger when none is attached, so library code logs unconditionally without a nil check and a bare context never
  panics.
- **No PHI through the provided helpers.** The policy is that no Protected Health Information — patient names,
  identifiers, dates, element values, query text, or file paths that embed them — is logged at default verbosity. The
  package enforces this for the field helpers it provides, which render DICOM, HL7 v2, and FHIR concepts **by name,
  never by value**: `DICOMTag(group, element, keyword)` logs the tag coordinate and dictionary keyword (for example
  `(0010,0010)` / `PatientName`), `HL7Field(segment, field)` logs the segment-and-field locator (for example `PID-5`),
  and `FHIRPath(path)` logs the element path (for example `Patient.name.family`). Each helper takes a structural
  locator, not patient data, so the API refuses raw patient values by construction — there is deliberately no helper
  that logs an element's value. As defense-in-depth, each string locator is shape-validated against its concept's
  grammar, so a value misrouted into a locator slot (`DOE^JANE`, a date, an MRN, a raw `PID|...` segment) is redacted
  rather than logged. Shape validation cannot tell a bare identifier-shaped token (a surname like `Smith`) from a real
  keyword; binding a locator to the canonical DICOM/HL7 vocabulary is the caller's responsibility at the
  domain-package boundary, which owns those dictionaries. `FromContext` hands back a raw `*zap.Logger`, so the no-PHI
  rule still binds the caller: log structure through the helpers and never pass a raw patient value to `zap.String` or
  a sibling field.

PHI governance beyond these safe defaults — encryption at rest, retention and erasure, access control, and audit — is
the integrating consumer's responsibility (PRD §9.1). The library ships the safe defaults and the structural field
vocabulary, not a compliance regime.

### PHI-default sanity sweep

The no-PHI-by-default behaviour is test-enforced, not convention-enforced, by a library-wide sanity sweep (PRD §11.2).
The sweep lives in `internal/phisweep` and runs as `go test ./internal/phisweep/` (Unix only: it redirects the
process file descriptors 1 and 2, which the matrix builds on Linux and macOS). It builds DICOM datasets and HL7 v2
messages carrying known, distinctive PHI sentinel tokens (synthetic values such as `SENTINEL^PHI^DONOTLOG`, never real
patient data), exercises representative entry points and parsers at default verbosity — `dicom.ReadFile` and dataset
access, `hl7v2.Parse` with segment/field accessors and round-trip marshalling, seeded with the `testdata/dicom` and
`testdata/hl7v2` fixtures — and scans every observable sink for any sentinel. A single appearance fails the sweep.

The sweep scans four sinks, the channels through which a careless path could surface a value at default verbosity:

| Sink | What it captures |
|------|------------------|
| `stdout` | The process standard output stream, redirected through an OS pipe for the run. |
| `stderr` | The process standard error stream, redirected through an OS pipe for the run. |
| `error` | The strings of any errors returned by an exercised entry point. |
| `log` | The structured-log output captured from the `logging` package at default (info) verbosity. |

A deliberately-leaking negative case plants a sentinel into each sink in turn and asserts the sweep detects it, so the
gate is proven to bite rather than merely assumed to. As new server and CLI paths land, they extend the sweep's
exercised entry points; the four swept sinks are the stable contract.

## Build and module layout

`cmd/radx` lives in its own Go module (`github.com/codeninja55/go-radx/cmd/radx`) so that consumers importing the
library packages do not inherit the CLI's dependency graph. Both the CLI module and the library root module declare the
same pinned Go toolchain, `go 1.26.4`, in agreement with `mise.toml`. A committed `go.work` at the repository root
(`use (. ./cmd/radx)`) composes the two modules into one Go workspace for local builds and editor tooling; the
`go.work.sum` lock file stays git-ignored. The workspace is a development convenience and does not change how either
module is built in isolation.

The `cmd-radx` CI job builds, vets, lints, and vulnerability-scans this module on every push and pull request to
`main`. It runs with `GOWORK=off` so each step resolves against `cmd/radx`'s own `go.mod`/`go.sum` — the module is
gated as a downstream consumer building the CLI would see it. The full build-and-module contract, including how the
workspace leaves the library jobs' root-only scope intact, is fixed in the cross-cutting statement's
[Build and module layout](./cross-cutting.md#build-and-module-layout-gowork-cmdradx-ci) section.

## Verification

The `cmd/radx` module is gated in CI by the `cmd-radx` job, which runs `go build ./...`, `go vet ./...`,
`golangci-lint run ./...`, and a pinned `govulncheck ./...` against the module. That closes the window where the
separate CLI module was uncompiled and unvetted in CI. The server interop suites are declared in the DICOM and DICOMweb
statements. Command-level tests ship alongside the commands: the `command` package's contract tests assert the
operator-facing invariants the CLI conformance rests on — machine stdout stays clean of diagnostics
(`TestMachineStdoutIsClean`), debug logging goes to stderr not stdout (`TestDebugLogsGoToStderrNotStdout`), the version
flag is coherent (`TestVersionFlagCoherent`), and a not-yet-implemented command fails closed rather than no-opping to
success (`TestStubFailsClosed`) — and the `serve` tests drive a loopback DICOMweb daemon round-trip, the fail-closed
non-loopback bind, and graceful shutdown on signal. These run under the standing `-race` gate the
[cross-cutting statement](./cross-cutting.md#concurrency-and-race-posture) describes. The gate proves the module builds,
vets, lints, scans clean, and that its contract and serve behaviour hold.

## References

- go-radx PRD §6.1 (conformance definition), §9 (NFRs and PHI policy).
- go-radx reference docs: `docs/reference/cli.md`, `docs/reference/servers.md`.
- Per-standard conformance statements: [`./dicom.md`](./dicom.md), [`./dicomweb.md`](./dicomweb.md),
  [`./hl7v2.md`](./hl7v2.md), [`./fhir.md`](./fhir.md), [`./convert.md`](./convert.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
