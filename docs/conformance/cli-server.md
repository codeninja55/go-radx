# CLI and server conformance statement

> **Implementation status: NOT YET SHIPPED.** This is a scaffold. The `radx` command-line interface is not yet
> implemented: `cmd/radx` is a placeholder that prints a not-implemented notice, so none of the command groups below
> exist yet. The embeddable library server types (`dimse.Server`, `dicomweb.Server`) do exist, but their CLI wiring and
> the operator-facing guarantees described here do not. Until this banner is removed, **no CLI behaviour is
> conformance-guaranteed**, and the command surface, flag contract, exit-code policy, logging behaviour, and PHI policy
> below are the planned design, not shipped behaviour. Do not cite this document as a conformance basis.

| Field | Value |
|-------|-------|
| Surface | `radx` CLI (`cmd/radx`) and the embeddable server entry points |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | unassigned (statement not yet authored) |
| Status | **NOT YET SHIPPED** — scaffold only; `cmd/radx` prints a not-implemented notice |
| Scope authority | This document will become the single source of truth for CLI/server scope (PRD §6.1) |

This document will declare the conformance scope of the operator-facing surface: the `radx` command groups, their
flags and exit codes, and the behaviour of the DICOM/DICOMweb servers when invoked through the CLI. The planned command
design lives in `docs/reference/cli.md` and the server design in `docs/reference/servers.md`; this statement will fix
the scope and the operational guarantees once the implementation exists.

## Command surface

Not yet authored. This section will enumerate the in-scope command groups (the planned set includes `echo`, `store`,
`scp`, `dump`, `modify`, `organize`, `lookup`, `catalogue`, and the `hl7`, `dicomweb`, and `convert` groups), each with
its flag contract and documented exit codes. The design is in `docs/reference/cli.md`.

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
statements. Command-level smoke tests are not yet authored — the CLI command surface is not implemented — so no CLI
*behaviour* conformance claim is made yet; the gate today proves the module builds, vets, lints, and scans clean.

## References

- go-radx PRD §6.1 (conformance definition), §9 (NFRs and PHI policy).
- go-radx reference docs: `docs/reference/cli.md`, `docs/reference/servers.md`.
- Per-standard conformance statements: [`./dicom.md`](./dicom.md), [`./dicomweb.md`](./dicomweb.md),
  [`./hl7v2.md`](./hl7v2.md), [`./fhir.md`](./fhir.md), [`./convert.md`](./convert.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
