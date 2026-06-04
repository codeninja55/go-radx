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
- **No PHI is logged at default verbosity.** No Protected Health Information — patient names, identifiers, dates,
  element values, query text, or file paths that embed them — is logged unless the operator explicitly raises
  verbosity. The package's field helpers render DICOM, HL7 v2, and FHIR concepts **by name, never by value**:
  `DICOMTag(group, element, keyword)` logs the tag coordinate and dictionary keyword (for example `(0010,0010)` /
  `PatientName`), `HL7Field(segment, field)` logs the segment-and-field locator (for example `PID-5`), and
  `FHIRPath(path)` logs the element path (for example `Patient.name.family`). Each helper takes a structural locator,
  not patient data, so the API refuses raw patient values by construction — there is deliberately no helper that logs
  an element's value. As defense-in-depth, each string locator is shape-validated against its concept's grammar, so a
  value misrouted into a locator slot (`DOE^JANE`, a date, an MRN, a raw `PID|...` segment) is redacted rather than
  logged. Shape validation cannot tell a bare identifier-shaped token (a surname like `Smith`) from a real keyword;
  binding a locator to the canonical DICOM/HL7 vocabulary is the caller's responsibility at the domain-package
  boundary, which owns those dictionaries.

PHI governance beyond these safe defaults — encryption at rest, retention and erasure, access control, and audit — is
the integrating consumer's responsibility (PRD §9.1). The library ships the safe default and the structural field
vocabulary, not a compliance regime.

## Build and module layout

Not yet authored. This section will declare the build and module contract: `cmd/radx` lives in its own Go module so
that consumers importing the library packages do not inherit the CLI's dependency graph, how the CLI module composes
the library modules (for example via a Go workspace), the supported Go toolchain version, and the reproducible-build
posture. Until authored, the layout is whatever `cmd/radx/go.mod` and the repository root declare.

## Verification

Not yet authored. This section will state how the CLI and server surface is gated: a build-and-vet gate for the
`cmd/radx` module, command-level smoke tests, and the server interop suites already declared in the DICOM and DICOMweb
statements. Until then, no CLI or server conformance claim is made.

## References

- go-radx PRD §6.1 (conformance definition), §9 (NFRs and PHI policy).
- go-radx reference docs: `docs/reference/cli.md`, `docs/reference/servers.md`.
- Per-standard conformance statements: [`./dicom.md`](./dicom.md), [`./dicomweb.md`](./dicomweb.md),
  [`./hl7v2.md`](./hl7v2.md), [`./fhir.md`](./fhir.md), [`./convert.md`](./convert.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
