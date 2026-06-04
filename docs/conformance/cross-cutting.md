# Cross-cutting conformance statement

> **Implementation status: NOT YET SHIPPED.** This is a scaffold. The cross-cutting conformance statement — the
> versioned contract for the engineering posture that holds *across* every subsystem (supply chain, interop
> determinism, build and module layout, coverage, concurrency, conformance-drift control, and governance) — is not yet
> authored. Until this banner is removed and the sections below are filled, **no cross-cutting guarantee is
> conformance-backed**. The CI workflow at `.github/workflows/ci.yml` runs the gates described below, but several of
> them are still being assembled and, as recorded under [Gate enforcement status](#gate-enforcement-status), none of
> them is merge-blocking yet. Do not cite this document as a conformance basis.

| Field | Value |
|-------|-------|
| Scope | Engineering posture shared across every subsystem (CI, supply chain, interop, build, coverage, concurrency) |
| Library | `github.com/codeninja55/go-radx` |
| Conformance version | unassigned (statement not yet authored) |
| Status | **NOT YET SHIPPED** — scaffold only |
| Scope authority | This document will be the single source of truth for the cross-cutting contract (PRD §6.1) |

This document is the cross-cutting counterpart to the per-subsystem conformance statements. Where the DICOM
([`./dicom.md`](./dicom.md)), DIMSE ([`./dimse.md`](./dimse.md)), DICOMweb ([`./dicomweb.md`](./dicomweb.md)),
conversion ([`./convert.md`](./convert.md)), HL7 v2 ([`./hl7v2.md`](./hl7v2.md)), FHIR ([`./fhir.md`](./fhir.md)), and
CLI/server ([`./cli-server.md`](./cli-server.md)) statements each declare *what* their subsystem supports against a
healthcare standard, this statement declares *how* the whole library is built, verified, and governed: the policies
that no single subsystem owns but every subsystem inherits. It restates none of their scope; it fixes the shared
engineering contract that makes their claims trustworthy and repeatable.

## Supply chain

go-radx runs on a single pinned Go toolchain across every CI job and every module. The version is Go 1.26.4, declared in
three places that must stay in agreement: the module root `go.mod`, the `cmd/radx` module's `go.mod`, and the `[tools]`
table in `mise.toml`. `mise.toml` is the installer of record — `jdx/mise-action` provisions that toolchain for the
`lint-test`, `conformance`, `interop`, and `codecs` jobs — while the `govulncheck` job pins the same `1.26.4` through
`actions/setup-go`, since it does not run under mise.

The `govulncheck` vulnerability scan runs on every push and pull request to `main` against the module root (`go.mod`).
The scanner itself is pinned to a released version: the job runs
`go install golang.org/x/vuln/cmd/govulncheck@v1.3.0` rather than `@latest`, so the scanner's analysis behaviour is
fixed run to run rather than dependent on whichever release the proxy resolves at install time. The vulnerability
database stays live — `govulncheck` queries `vuln.go.dev` by default — so a freshly disclosed advisory still fails
the gate; the pin fixes the analysis tooling, not the advisory feed.

Two drift surfaces remain open against this posture and are owned elsewhere in the cross-cutting contract. The Go
version pin is held in agreement by review today, not by an automated check that fails when the three declarations
diverge. And `cmd/radx` is a separate module that the `govulncheck` job does not yet scan — closing that is tracked
under [Build and module layout](#build-and-module-layout-gowork-cmdradx-ci), which will bring `cmd/radx` into CI under
the same pinned toolchain.

## Interop determinism: pinned tools and images

Not yet authored. This section will declare the determinism contract for the interop and conformance gates: the pinned
versions of the reference tools (`dciodvfy` from dicom3tools, `pydicom`, the from-source C codec libraries) and the
pinned digests of the interop container images (Orthanc, dcm4chee-arc), so an interop result is reproducible rather than
dependent on whatever a runner happens to pull. Several inputs still float against that target. The C codec libraries
are version-pinned and built from source, but `dciodvfy` (from `dicom3tools`) and `pydicom` are installed unpinned via
`apt-get` and `pip`, and the Orthanc image is referenced as `orthancteam/orthanc:latest` in both the DIMSE and DICOMweb
interop harnesses; the dcm4chee-arc images are tag-pinned but not digest-pinned. Closing those is the work of pinning
the reference tools and interop image digests and adding a pin-drift check.

## Interop-matrix coverage

Not yet authored. This section will enumerate which subsystems are exercised against which reference origin servers and
validators, and the gaps that remain — so a reader can see at a glance which conformance claims are interop-backed
today and which are still asserted against unit fixtures only.

## Build and module layout (go.work, cmd/radx CI)

Not yet authored. This section will declare the multi-module build contract: the library packages at the module root,
the separate `cmd/radx` module that carries the CLI's dependency graph so library consumers do not inherit it, the Go
workspace (`go.work`) that composes them, and how `cmd/radx` is built and vetted in CI under the same pinned toolchain.
Two pieces of that target are not yet present: there is no `go.work` tying the modules, and the CI workflow does not
build or vet the `cmd/radx` module, so the module is uncompiled and unvetted in CI today.

## Coverage targets and critical-path enumeration

Not yet authored. This section will declare the coverage contract: the measurement method, the enforced floor, and the
explicit enumeration of the critical paths (the Part 10 reader, the DIMSE association and DIMSE-C services, the DICOMweb
round-trips, and the cross-standard converters) that the floor is meant to protect — so coverage is a guarantee on the
paths that matter, not just an aggregate percentage.

## Concurrency and race posture

Not yet authored. This section will declare the concurrency contract: no global mutable state, concurrent-safe public
APIs, the standing `go test -race ./...` gate, and the per-server review checklist that a new server entry point must
clear before it ships.

## Conformance-drift methodology

Not yet authored. This section will declare how the conformance statements are kept honest as the implementation moves:
the merge-blocking drift check that fails when a statement and the code it describes diverge, and the documentation-site
build that proves the statements render and cross-link cleanly.

## Governance and stability posture

The project's governance and stability documents are in place and authoritative. go-radx is pre-1.0 and every public
package is experimental: exported types, signatures, and behaviour can change between any two `v0.x` releases, as
declared in the stability banner in [`README.md`](../../README.md) and mirrored by the one-line stability marker each
standard and server package (`convert`, `dicom`, `dicomweb`, `dimse`, `fhir`, `hl7v2`, `server`) carries in its godoc.
Security reporting follows [`SECURITY.md`](../../SECURITY.md) (private vulnerability reporting, synthetic-sentinel-only
proofs of concept), contribution conventions and the local gate list follow [`CONTRIBUTING.md`](../../CONTRIBUTING.md),
and community conduct follows [`CODE_OF_CONDUCT.md`](../../CODE_OF_CONDUCT.md). This section will, when authored, fix
the versioning and deprecation policy that turns the experimental posture into a predictable contract once a stable
release is tagged.

## Gate enforcement status

The CI workflow at `.github/workflows/ci.yml` runs on every push and pull request to `main` and defines five jobs:
`lint-test` (gofmt, `go vet`, golangci-lint on the default and interop builds, `go build`, and `go test -race ./...`),
`conformance` (the `dciodvfy` and `pydicom` gates with `CI=true`), `interop` (the DIMSE testcontainers gate),
`govulncheck` (the vulnerability scan), and `codecs` (the C-backed pixel codecs built from source). These jobs report
status on every pull request.

They are **currently advisory, not merge-blocking.** The `main` branch ruleset exists but its enforcement is set to
*disabled*, and `main` has no branch-protection configured, so a red CI run does not block a merge at the GitHub level.
Enforcement was left off intentionally to keep the merge path open while the gates are still being assembled. The
consequence: until enforcement is turned on, the local gates in [`CONTRIBUTING.md`](../../CONTRIBUTING.md) and
adversarial review are the only pre-merge net, and the CI signal is advisory rather than a guarantee.

This is a **known gap against the Phase 0 definition of done.** Closing it means enabling the `main` ruleset with the CI
jobs as required status checks, so the gates this statement relies on actually bind. Until enforcement is enabled, treat
every "gated by CI" claim in the per-subsystem statements as gated-but-not-enforced.

## References

- go-radx PRD §6.1 (conformance definition), §9 (NFRs, observability, and PHI policy), §11 (verification strategy).
- CI workflow: `.github/workflows/ci.yml`. Toolchain and task pins: `mise.toml`.
- Governance: [`SECURITY.md`](../../SECURITY.md), [`CONTRIBUTING.md`](../../CONTRIBUTING.md),
  [`CODE_OF_CONDUCT.md`](../../CODE_OF_CONDUCT.md), and the stability banner in [`README.md`](../../README.md).
- Per-subsystem conformance statements: [`./dicom.md`](./dicom.md), [`./dimse.md`](./dimse.md),
  [`./dicomweb.md`](./dicomweb.md), [`./convert.md`](./convert.md), [`./hl7v2.md`](./hl7v2.md),
  [`./fhir.md`](./fhir.md), [`./cli-server.md`](./cli-server.md).
- go-radx glossary: `UBIQUITOUS_LANGUAGE.md`.
