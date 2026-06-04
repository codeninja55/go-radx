# Contributing to go-radx

Thanks for your interest in go-radx, a Go library for medical imaging and healthcare
interoperability standards (FHIR, DICOM, DICOMweb, HL7 v2.x, and DIMSE). This guide explains how to
set up a development environment, the gates your change must pass, and the conventions we follow.

By participating, you agree to abide by the [Code of Conduct](CODE_OF_CONDUCT.md).

## Stability posture

go-radx is pre-1.0 and every public package is experimental — the API can change between any two
`v0.x` releases. See the stability section in [README.md](README.md) before you build on a package,
and check the one-line stability marker in each top-level public package's godoc. Contributions that
stabilise an experimental package are welcome; raise an issue first so we can agree on the API
surface.

## Prerequisites

- Go 1.26.x. The toolchain is pinned to 1.26.4 and managed with [mise](https://mise.jdx.dev/); see
  `mise.toml`.
- `golangci-lint` 2.x. mise installs the pinned version for you.
- Optional, for conformance and interop work: `dciodvfy` (dicom3tools), `pydicom`, and Docker (for
  the testcontainers-based interop gates).

Install the pinned toolchain:

```bash
mise install
```

## Local gates

Run these before opening a pull request. They mirror what CI enforces.

```bash
go build ./...
gofmt -l .                       # must print nothing
mise exec -- golangci-lint run ./...
go vet ./...
go test -race ./...
```

Per-package test suites are available as mise tasks (and mirrored in the `Makefile`), for example:

```bash
mise run test:dicom
mise run test:fhir
mise run test:skeleton           # the cross-standard walking-skeleton suites
```

Run only the suites for the packages you touched, plus `go build ./...` and `gofmt -l .` across the
whole tree.

## Coding conventions

- Target modern Go 1.26: use `any` rather than `interface{}`, `errors.Is`/`errors.As` for error
  checks, and generics where they add type safety.
- No global mutable state; keep APIs concurrent-safe.
- Prefer table-driven tests.
- Match the style, naming, and comment density of the surrounding code. Default to no comments; add
  one only when the *why* is non-obvious. Keep comments evergreen — do not reference the current
  task, issue, or recent changes.
- Each public package (`convert`, `dicom`, `dicomweb`, `dimse`, `fhir`, `hl7v2`, and `server`)
  carries a one-line stability marker in its godoc; keep it accurate when a package's maturity
  changes.

## Handling health data

This library processes Protected Health Information (PHI). Never commit real PHI to code, tests,
fixtures, or logs. Use clearly synthetic sentinel values such as `ZZZTEST^PHI^SENTINEL`. The library
never logs PHI by default.

## Commits and pull requests

- Use [Conventional Commits](https://www.conventionalcommits.org/): `<type>(scope): <subject>`, with
  types such as `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`, `build`, and
  `security`. Split source, documentation, and configuration into separate atomic commits.
- The commit body explains *why*, not *what*.
- After a pull request is created, update `CHANGELOG.md`: move entries from `[Unreleased]` into a
  versioned section when appropriate and add the PR number, following the
  [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.

## Reporting bugs and requesting features

Open a GitHub issue with enough detail to reproduce: the affected package, the version or commit, and
a minimal example. For security issues, follow [SECURITY.md](SECURITY.md) instead of opening a public
issue.
