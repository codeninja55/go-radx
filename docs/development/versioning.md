# Versioning and API stability

This document is the versioning contract for go-radx: how releases are numbered, how stable each part
of the API is today, how breaking changes and deprecations are handled, and what the Go module path
means for a major-version bump. It is the developer-facing companion to the stability banner in
[the README](../../README.md) and the governance section of the
[cross-cutting conformance statement](../conformance/cross-cutting.md).

go-radx is pre-1.0. The honest summary is that the whole public API is experimental and may change
between any two `v0.x` releases. The rest of this document makes that precise and says what will
change at `v1.0.0`.

## Semantic versioning

go-radx follows [Semantic Versioning 2.0.0](https://semver.org/). A release is `MAJOR.MINOR.PATCH`:

- `MAJOR` increments for an incompatible public-API change.
- `MINOR` increments for backward-compatible additions.
- `PATCH` increments for backward-compatible fixes.

The public API is the exported surface of the library packages (`dicom`, `dimse`, `dicomweb`,
`hl7v2`, `fhir`, `convert`, `server`) reachable through `go get github.com/codeninja55/go-radx`. The
`internal/` tree and the `cmd/radx` CLI module are not part of the library's API-compatibility
contract; the CLI has its own command and output contract documented under
[`docs/reference/cli.md`](../reference/cli.md).

Versions are git tags on the `main` branch. There are no binary releases of the library — it is
consumed as a Go module through the module proxy. The one binary artifact is the `radx` CLI, released
separately as described under [released-artifact integrity](#released-artifact-integrity).

## The pre-1.0 reality: 0.x allows breaking changes between minors

Under SemVer, a `0.y.z` release makes no compatibility promise across `MINOR` versions: anything may
change while `MAJOR` is `0`. go-radx uses that latitude deliberately. While the project is pre-1.0:

- Exported types, function signatures, and behaviour can change between any two `v0.x` releases,
  including between `v0.3.0` and `v0.4.0`, without a deprecation cycle.
- A `0.x` minor bump (`v0.3.x` to `v0.4.0`) may carry breaking changes. A `0.x` patch bump
  (`v0.4.0` to `v0.4.1`) is intended to stay backward-compatible, but this is best-effort, not a
  guarantee, until `v1.0.0`.
- Pin an exact version in your `go.mod` and expect to adjust call sites when you upgrade.

Do not depend on go-radx in production-critical paths until it reaches `v1.0.0`. The legacy
`v0.10.x` and earlier tags belong to a different codebase on the `legacy-main` branch and are not
continued here; the re-foundation history starts fresh.

## Per-package stability tiers

Each top-level library package carries a one-line `Stability:` marker in its package godoc, and the
[conformance-drift check](../conformance/cross-cutting.md#the-drift-check) fails CI if a package drops
its marker. There are two tiers:

- **experimental** — the API is still moving; expect breaking changes between `v0.x` releases.
- **stabilising** — the API is settling toward `v1`; changes are made more conservatively, though
  they are still possible before `v1.0.0`.

Today **every library package is marked `experimental`.** No package has earned the `stabilising`
marker yet. The table below records each package's current marker alongside how far its conformance
statement has progressed, so you can judge which surfaces are closest to settling — without implying a
stability the godoc does not yet claim.

| Package    | godoc marker | Conformance-statement status                                          |
| ---------- | ------------ | --------------------------------------------------------------------- |
| `dicom`    | experimental | Statement authored; the implementation conforms to it.                |
| `dimse`    | experimental | Standalone statement is a scaffold; the DICOM statement is authoritative for DIMSE. |
| `hl7v2`    | experimental | Statement is normative for v1.                                        |
| `fhir`     | experimental | R4 (4.0.1) and R5 (5.0.0) resources generated; validated by the HL7 validator. |
| `dicomweb` | experimental | Statement published; the `Server` concurrency posture is still being proven. |
| `convert`  | experimental | Statement is a scaffold; the §5.1 workflow loop ships with R4 and R5 twins. |
| `server`   | experimental | Statement is a scaffold; `dimse.Server` is the reference server surface. |

A package moves from `experimental` to `stabilising` only as a reviewed change: its conformance
statement is authored (or its `NOT YET SHIPPED` banner removed), its API has held steady across at
least one release, and the godoc marker and this table are updated together. Promotion is never
silent.

## Deprecation policy

Before `v1.0.0`, an experimental package may remove or change exported API without a deprecation
cycle, because the `0.x` contract above already warns callers to expect breakage. Where it is
practical, a removal will still be softened:

- Mark the old symbol with a `// Deprecated:` godoc comment that names the replacement, so `go vet`
  and editors flag uses of it.
- Note the change in [`CHANGELOG.md`](../../CHANGELOG.md) under the release that makes it.

From `v1.0.0` onward, the deprecation policy tightens: a stable exported symbol that is to be removed
is first marked `// Deprecated:` in a release, kept working for at least one subsequent `MINOR`
release, and only then removed in a `MAJOR` release. The full `v1` deprecation guarantee is fixed
when `v1.0.0` is tagged.

## Major versions and the module path

Go's module system encodes a major version of `2` or higher in the import path. When go-radx reaches
`v2.0.0`, the module path gains a `/v2` suffix:

```text
v0.x and v1.x:  github.com/codeninja55/go-radx
v2.x:           github.com/codeninja55/go-radx/v2
```

This is the standard Go "semantic import versioning" rule: `v0` and `v1` share the unsuffixed path,
and each major version from `v2` onward lives at its own `/vN` path so two majors can coexist in one
build. A `v2+` bump therefore changes every import line in your code, which is the intended, explicit
signal that you are adopting an incompatible release. The `cmd/radx` module path will gain the same
suffix at its own `v2`.

## Released-artifact integrity

The library is verified through the Go module proxy (`proxy.golang.org` and `sum.golang.org`), so a
pinned module version is already checksum-verified by the Go toolchain.

The `radx` CLI is released as a signed binary. Each tagged `v*` release is built by
[`.github/workflows/release.yml`](../../.github/workflows/release.yml) with
[GoReleaser](https://goreleaser.com/) and carries:

- Cross-platform archives for linux, macOS, and Windows on amd64 and arm64, each stamping the build
  version, commit, and date into `radx --version` via linker flags.
- A `checksums.txt` listing the SHA-256 of every archive.
- A `cosign` keyless signature over the checksums file. Signing uses Sigstore with the GitHub Actions
  OIDC token, so there is no long-lived signing key to manage; verify it with
  `cosign verify-blob`.
- A Software Bill of Materials per archive, in both CycloneDX and SPDX JSON, generated by
  [syft](https://github.com/anchore/syft).
- A SLSA build-provenance attestation for each artifact, written with
  [`actions/attest-build-provenance`](https://github.com/actions/attest-build-provenance) and
  verifiable with `gh attestation verify`.

These artifacts are produced only by the tag-triggered release pipeline. A `goreleaser check` and a
no-publish snapshot build run on every pull request, so the release configuration is exercised before
a tag is ever pushed.

## See also

- [README stability banner](../../README.md) — the one-paragraph statement of the pre-1.0 posture.
- [Cross-cutting conformance statement](../conformance/cross-cutting.md) — the governance and
  stability posture across every subsystem.
- [Security policy](../../SECURITY.md) — vulnerability disclosure and supported versions.
