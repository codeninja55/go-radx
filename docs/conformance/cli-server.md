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

Not yet authored. This section will declare the operational logging contract: structured logging via `go.uber.org/zap`,
the context-injected logger convention, and the PHI policy — **no Protected Health Information is logged unless the
operator explicitly opts in** (CLAUDE.md, PRD §9). It will state which fields are redacted by default, how the opt-in
is expressed, and how the CLI and servers honour the same policy. Until authored, treat the default as PHI-suppressing
and verify against the implementation when it lands.

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
