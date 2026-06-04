# Security policy

go-radx handles data models for medical imaging and healthcare interoperability. Vulnerabilities in
this library can affect systems that process Protected Health Information (PHI), so we take security
reports seriously.

## Supported versions

go-radx is pre-1.0 (see the stability posture in [README.md](README.md)). Only the re-foundation
`main` branch receives security fixes. The published `v0.10.x` and earlier tags belong to the legacy
codebase on `legacy-main` and are not maintained. There is no long-term support branch while the API
is still settling; once the re-foundation is tagged, the most recent `v0.x` release will also be
covered.

## Reporting a vulnerability

Report suspected vulnerabilities privately rather than opening a public issue:

- Use GitHub's [private vulnerability reporting](https://github.com/codeninja55/go-radx/security/advisories/new)
  for this repository.

When you report, include enough detail to reproduce the issue — affected package, version or commit,
and a minimal proof of concept. Never include real PHI in a report; use clearly synthetic sentinel
values (for example `ZZZTEST^PHI^SENTINEL`).

We aim to acknowledge a report within a few working days and to keep you informed as we triage and
fix it. Coordinated disclosure is preferred: please give us a reasonable window to ship a fix before
any public write-up.

## Roadmap

This document is an interim pointer. A formal, signed coordinated-disclosure policy — covering
disclosure timelines, PGP keys, and CVE handling — lands in a later hardening phase. Until then, the
private reporting channel above is the supported path.
