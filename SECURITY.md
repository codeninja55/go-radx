# Security policy

go-radx handles data models for medical imaging and healthcare interoperability. Vulnerabilities in
this library can affect systems that process Protected Health Information (PHI), so we take security
reports seriously and treat the disclosure process described here as the supported path.

go-radx is pre-1.0 and developed by a small team. This policy sets honest expectations for that
reality: the disclosure channel is real and monitored, but the response times below are best-effort
targets, not a contractual service-level agreement.

## Reporting a vulnerability

Report suspected vulnerabilities **privately**. Do not open a public issue, pull request, or
discussion for a security problem, because a public report discloses the issue before a fix exists.

Use GitHub's private vulnerability reporting for this repository:

- Open a private advisory through the repository's
  [security advisories page](https://github.com/codeninja55/go-radx/security/advisories/new).

This routes the report to the maintainers through a GitHub Security Advisory (GHSA) that stays
private until we coordinate disclosure. If you cannot use GitHub advisories, you may instead contact
the maintainer through the address on the
[GitHub profile](https://github.com/codeninja55) and we will move the conversation into a private
advisory.

### What to include

Give us enough to reproduce and assess the issue:

- The affected package or subsystem (for example `dicom`, `dimse`, `hl7v2`, `fhir`, `dicomweb`,
  `convert`, `server`, or the `radx` CLI).
- The version, tag, or commit SHA you tested against.
- A minimal proof of concept: the input bytes or call sequence that triggers the issue, and what you
  expected versus what happened.
- Your assessment of impact (for example denial of service via unbounded allocation, a parser that
  panics on hostile input, or a path that could leak data across a trust boundary).

### Never include real PHI

go-radx parses medical data, so a natural proof of concept is a real DICOM object, HL7 v2 message, or
FHIR resource. **Do not send real Protected Health Information.** Reproduce the issue with clearly
synthetic sentinel values (for example a patient name of `ZZZTEST^PHI^SENTINEL`) or with structurally
equivalent fabricated data. If you can only reproduce with a real artifact, tell us that in the
report and we will arrange a safe way to share a minimised, de-identified sample rather than have you
attach PHI to an advisory.

## What to expect

These are best-effort targets for a small pre-1.0 project, not guarantees:

- **Acknowledgement** within roughly five working days that we have received the report.
- **Initial triage** — a severity assessment and whether we can reproduce it — within roughly ten
  working days.
- **Ongoing updates** as we investigate and work on a fix, on a cadence we agree with you.

If you do not hear back within those windows, please send a follow-up on the same advisory; a missed
response is far more likely to be limited maintainer time than a decision to ignore the report.

## Coordinated disclosure

We prefer coordinated disclosure. Please give us a reasonable window to ship a fix before any public
write-up. We aim to agree a disclosure date with you during triage; absent a specific agreement, a
90-day window from the acknowledgement is a reasonable default, shortened if a fix lands sooner or if
the issue is already being exploited in the open.

When a fix is ready we will publish a GitHub Security Advisory, request a CVE through GitHub where the
issue warrants one, and credit you in the advisory unless you ask to remain anonymous. Because the
project is pre-1.0, a fix normally ships on `main` and in the next tagged release rather than as a
backport to an older line (see supported versions below).

## Supported versions

go-radx is pre-1.0; see the stability posture in [README.md](README.md) and the
[versioning and API-stability policy](docs/development/versioning.md). Security support reflects that
status and is best-effort:

| Version line                        | Branch        | Security support                                       |
| ----------------------------------- | ------------- | ------------------------------------------------------ |
| Re-foundation `main`                | `main`        | Supported — security fixes land here first.            |
| Most recent tagged `v0.x` release   | tag on `main` | Supported once tagged — fix ships in the next release. |
| Older `v0.x` re-foundation releases | tag on `main` | Not maintained — upgrade to the latest release.        |
| Legacy `v0.10.x` and earlier        | `legacy-main` | **Not maintained.** Belongs to the legacy codebase.    |

There is no long-term-support branch while the API is still settling. We do not backport security
fixes to older `v0.x` releases; the supported response is to upgrade to the latest release on `main`.
This will be revisited at `v1.0.0`, when a stable line gains a defined support window.

## Supply-chain integrity of released artifacts

The only binary artifact go-radx publishes is the `radx` command-line interface; the library itself
is consumed as a Go module through the module proxy. Released `radx` archives carry supply-chain
metadata so you can verify what you run — a `cosign` keyless signature over the checksums file, a
CycloneDX and SPDX SBOM per archive, and a SLSA build-provenance attestation. The mechanics are
documented in the
[versioning and API-stability policy](docs/development/versioning.md#released-artifact-integrity).
