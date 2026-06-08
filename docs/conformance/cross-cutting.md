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
`lint-test`, `conformance`, `interop`, `cmd-radx`, and `codecs` jobs — while the `govulncheck` job pins the same
`1.26.4` through `actions/setup-go`, since it does not run under mise.

The `govulncheck` vulnerability scan runs on every push and pull request to `main` against the module root (`go.mod`).
The scanner itself is pinned to a released version: the job runs
`go install golang.org/x/vuln/cmd/govulncheck@v1.3.0` rather than `@latest`, so the scanner's analysis behaviour is
fixed run to run rather than dependent on whichever release the proxy resolves at install time. The vulnerability
database stays live — `govulncheck` queries `vuln.go.dev` by default — so a freshly disclosed advisory still fails
the gate; the pin fixes the analysis tooling, not the advisory feed.

The `cmd/radx` module is scanned too: the `cmd-radx` job runs the same `@v1.3.0`-pinned `govulncheck` against the CLI
module's own dependency graph, so a vulnerability in a CLI-only dependency fails CI rather than passing unscanned. See
[Build and module layout](#build-and-module-layout-gowork-cmdradx-ci) for how that module is built and gated.

One drift surface remains open against this posture and is owned elsewhere in the cross-cutting contract. The Go
version pin is held in agreement by review today, not by an automated check that fails when the three declarations
diverge.

## Interop determinism: pinned tools and images

Every external input the interop and conformance gates run today is pinned, so a gate result is reproducible rather than
dependent on whatever a runner happens to pull. The pins are held next to where each tool or image is invoked, indexed
in one human-readable manifest at [`tools/versions`](../../tools/versions), and enforced by a
[pin-drift check](#pin-drift-enforcement) that fails CI if any reference floats back to an unpinned tag. The one input
that is *not* yet a fixed-byte pin is the FHIR validator, which no gate invokes yet; it is pinned by mechanism and
release version, with its asset digest deferred until the gate is wired, as called out below.

The reference tools the gates run are version-pinned. The `conformance` job installs `dicom3tools` (which provides the
`dciodvfy` IOD validator) at the exact Ubuntu noble archive version `1.00~20240118131615-1` via `apt-get`, and `pydicom`
at the exact PyPI release `3.0.2` via `pip` — the 3.x line is required because the round-trip gate uses
`save_as(enforce_file_format=...)`, a pydicom 3.0 API. `apt` has no point-in-time archive snapshot, so the
`dicom3tools` pin is the exact version string and the install fails loudly if the runner image rolls the package
forward; that failure is the intended signal to re-pin deliberately, not a silent float. The C codec libraries
(`OpenJPEG`, `libjpeg-turbo`, `CharLS`) remain version-pinned and built from source in the `codecs` job, as the
[Supply chain](#supply-chain) posture already requires.

The interop container images are pinned by immutable digest, which is stronger than a tag: a tag can be re-pushed, a
digest cannot. The Orthanc image used by both the DIMSE and DICOMweb harnesses is
`orthancteam/orthanc:26.6.0@sha256:510ef4ce24699104244b00d2b93350a801fc2f1c6b0bfc6a1f15e546bff2d1f4`. The dcm4chee-arc
stack is three digest-pinned images — `dcm4che/slapd-dcm4chee:2.6.10-34.2`, `dcm4che/postgres-dcm4chee:17.4-34`, and
`dcm4che/dcm4chee-arc-psql:5.34.2`, each carrying its `@sha256:` digest in the testcontainers helper. The human-readable
version tag is retained alongside each digest for legibility, but the digest is what binds. Re-resolving a digest after
a deliberate version bump is `docker buildx imagetools inspect <image>:<version>`.

The FHIR validator is pinned by mechanism and release version ahead of use, not yet by asset digest. The mechanism
decision is the official HL7 `validator_cli.jar` from the `hapifhir/org.hl7.fhir.core` releases (not a container): the
jar is the canonical reference, runs on the JDK every runner can provision, resolves to a single GitHub release asset,
and avoids a second container-registry dependency. The pinned release is recorded as `6.9.9` in
[`tools/versions`](../../tools/versions). CI does not invoke the validator yet — that is the M6a (Phase 1) FHIR
conformance gate — so this is a recorded decision rather than a live, byte-pinned input. When the gate is wired, its
runner must download exactly that release asset, record the asset SHA-256 in the manifest, and verify it before use,
closing the validator to a fixed-byte pin like the tools and images above.

### Pin-drift enforcement

The pin-drift check ([`tools/pin-drift.sh`](../../tools/pin-drift.sh), run in CI as the `pin-drift` step of the
`lint-test` job and locally via `mise run pin-drift`) scans the files that pull external tools and images — the CI
workflow, `mise.toml`, and the testcontainers helpers — and fails if any of them reintroduces an unpinned reference: a
`:latest` (or other floating) image tag, an image tag without an `@sha256:` digest, an `@latest` install, a `mise`
`[tools]` entry pinned to `"latest"`, or a `pip` / `apt-get` install of a known reference tool without an exact version
pin (a non-exact specifier such as `~=` / `>=`, or the apt target-release form `tool/suite`, is treated as drift). It
enforces the *shape* of a pin (that an exact one is present), not the exact value — bumping a pin stays a deliberate,
reviewed change recorded in [`tools/versions`](../../tools/versions).

## Interop-matrix coverage

The `interop` job in `.github/workflows/ci.yml` runs as a matrix over three legs — `dimse`, `dicomweb`, and
`convert` — each on its own runner. The legs all drive containers through testcontainers, so isolating them keeps the
memory-heavy DIMSE `dcm4chee-arc` stack (LDAP + PostgreSQL + WildFly) off the same runner as the Orthanc-only DICOMweb
and convert legs; `fail-fast: false` reports every leg even when one fails. Each leg invokes the matching
`mise run interop:<leg>` task, so the CI command and the local command stay in agreement. Every container the legs
start resolves to the digest-pinned images recorded under
[Interop determinism](#interop-determinism-pinned-tools-and-images).

Each subsystem is exercised against a real reference origin server, not only unit fixtures:

| Leg | Subsystem | Reference origin(s) | What it proves |
|-----|-----------|---------------------|----------------|
| `dimse` | DIMSE | Orthanc + dcm4chee-arc | C-ECHO / C-STORE / C-FIND / C-GET / C-MOVE against two independent SCPs |
| `dicomweb` | DICOMweb | Orthanc (DICOMweb plugin) | STOW-RS store then WADO-RS retrieve round-trip |
| `convert` | convert (+ DIMSE, DICOMweb, HL7 v2, FHIR) | Orthanc | the M2 walking-skeleton six-leg end-to-end proof |

The `convert` leg is the cross-standard end-to-end proof: it C-STOREs to an Orthanc DIMSE SCP, STOW/WADO round-trips
against a separate Orthanc DICOMweb endpoint, then runs the three pure-conversion legs (HL7 ORM to ServiceRequest, DICOM
SR to DiagnosticReport, DICOM instance to ImagingStudy) in-process. Before this matrix, the `dicomweb/integration` and
`convert` interop tests compiled under `go build -tags interop ./...` but no CI job invoked them; only the `dimse` leg
ran. The matrix closes that regression window so a break in the DICOMweb or convert round-trip fails CI rather than
passing unnoticed.

The DICOMweb interop net carries a negative control that proves the gate bites without leaving a failing test in CI.
`TestInteropGuardBrokenDICOMWebPathFails` (in `dicomweb/integration`, behind the `interop` tag) starts the same real
Orthanc origin the positive test uses but points the client at a DICOMweb root that does not exist on the server,
then asserts the STOW-RS store fails — a passing store there would mean the gate could go green against an origin
that never accepted the instance. The guard is skipped unless `RADX_INTEROP_REGRESSION_GUARD=1` is set, so the matrix
stays green; run it on demand to confirm the gate catches a DICOMweb regression.

Two gaps remain against full interop-backed coverage. The HL7 v2 and FHIR subsystems are exercised only in-process
today (the convert leg's conversion legs run against vendored fixtures, not a live HL7 listener or FHIR server), so
their conformance claims rest on unit fixtures rather than a reference origin. And the FHIR validator gate is not yet
wired — its pin is recorded but no leg invokes it, as noted under
[Interop determinism](#interop-determinism-pinned-tools-and-images).

## Build and module layout (go.work, cmd/radx CI)

go-radx is two Go modules. The library packages live at the repository root (`github.com/codeninja55/go-radx`); the
`radx` command-line interface lives in its own module under `cmd/radx` (`github.com/codeninja55/go-radx/cmd/radx`) so
that consumers importing the library packages do not inherit the CLI's dependency graph. Both modules declare the same
pinned Go toolchain, `go 1.26.4`, in agreement with the `[tools]` table in `mise.toml`, as the
[Supply chain](#supply-chain) posture requires.

A Go workspace ties the two modules. The committed `go.work` at the repository root declares `go 1.26.4` and
`use (. ./cmd/radx)`, so editor tooling and cross-module commands resolve both modules as one tree — the CLI builds
against the in-tree library rather than a published version — without per-module directory switches. The `go.work.sum`
lock file is generated and stays git-ignored; the `go.work` file itself is committed because it is the workspace
contract, not a local convenience. The workspace does **not** widen the `./...` package pattern: that pattern stops at
the nested `cmd/radx` module boundary, so `go build ./...`, `go vet ./...`, `go test -race ./...`,
`golangci-lint run ./...`, and `govulncheck ./...` run from the repository root still resolve to the root module only;
they do not descend into `cmd/radx`. The `lint-test`, `codecs`, and `govulncheck` jobs are therefore unaffected by the
presence of `go.work`.

The `cmd-radx` job builds, vets, lints, and vulnerability-scans the CLI module so it is no longer uncompiled and
unvetted in CI. It runs from `cmd/radx` with `GOWORK=off`, which makes every step resolve against the module's own
`go.mod`/`go.sum` rather than the workspace — the module is gated exactly as a downstream consumer building the CLI
would see it, not as a workspace shortcut. The four steps are `go build ./...`, `go vet ./...`,
`golangci-lint run ./...` (the same mise-pinned `2.12.2` the library jobs use), and `govulncheck ./...` (the same
`@v1.3.0`-pinned scanner the root [Supply chain](#supply-chain) gate uses, resolved via `go run …@v1.3.0` under the
mise-provided Go). The job runs on every push and pull request to `main` alongside the other gates.

## Coverage targets and critical-path enumeration

go-radx enforces an aggregate statement-coverage floor on the library module: the build fails if coverage drops below
it. The floor is **80%** (PRD §11.4). It is measured and enforced in the `lint-test` job's race-test step
([`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)), which runs the `cover:check` mise task. That task first
runs `cover` — a single `go test -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...` over the
root module — then runs [`tools/cover-check.sh`](../../tools/cover-check.sh), which reads the merged profile's
`total:` line and fails the step if the aggregate is below the floor. The same gate runs locally via
`mise run cover:check`.

The measurement method matters for what the number means. `-coverpkg=./...` puts every statement in the root module in
the denominator, so a package with no tests (today `server` and `fhir/r4`) lowers the aggregate rather than being
silently excluded; the percentage is union coverage across all test binaries, not a per-binary figure. The run is
scoped to the root module with `GOWORK=off`, so the `go.work` workspace does not pull `cmd/radx` into the figure — the
CLI module is gated by its own `cmd-radx` job (see
[Build and module layout](#build-and-module-layout-gowork-cmdradx-ci)), not by this floor. `-race` is preserved
alongside `-covermode=atomic` (atomic is the coverage mode the race detector
requires), so the concurrency posture in [Concurrency and race posture](#concurrency-and-race-posture) and the coverage
floor are enforced by one run rather than two.

The 80% aggregate is a floor on the whole module, not a guarantee on the paths that carry patient data. The critical
paths below carry a higher **90%** target so coverage is a guarantee where a defect is most consequential. This target
is **enforced as a separate gate** by [`tools/cover-critical.sh`](../../tools/cover-critical.sh), run in the
`lint-test` job's "critical-path coverage gate" step directly after the 80%-floor step (and locally via
`mise run cover:critical`). It reuses the *same* merged race profile the 80% floor reads — the single
`go test -race -covermode=atomic -coverpkg=./... ./...` the `cover` task already wrote — so the two gates share one
race run rather than two, and it computes per-package *union* coverage (every test binary's contribution to a package's
own statements, generated files excluded) the same way the floor does. The critical paths, with the primary files each
comprises, are:

- **Part 10 reader / writer** (`dicom`): `file.go`, `file_meta.go`, `file_meta_write.go`, `dataset.go`,
  `dataset_codec.go`, `dataset_stream.go`, `dataset_writefile.go`, `element_header.go`, `reader_writer.go`,
  `bounded_reader.go`, `pixel_reader.go`.
- **DIMSE association** — the DUL/ACSE state machine and presentation-context negotiation (`dimse/dul`, `dimse/acse`):
  `dul/statemachine.go`, `dul/state.go`, `dul/event.go`, `dul/action.go`, `dul/connection.go`, `acse/negotiate.go`,
  `acse/associate.go`.
- **DIMSE-C services** — SCU and SCP sides (`dimse`, `dimse/pdu`): `association.go`, `command.go`, `message.go`,
  `dispatch.go`, `handler.go`, `server.go`, `echo.go`, `find.go`, `find_scp.go`, `move.go`, `move_scp.go`, `store.go`,
  and the PDU/PDV encode-decode in `dimse/pdu`.
- **DICOMweb round-trips** (`dicomweb`): `client.go`, `server.go`, `instance.go`, `resource.go`, `multipart.go`,
  `json.go`, `negotiation.go`, `store_response.go`, `bulkdata.go`.
- **Cross-standard converters** (`convert`): `orm_servicerequest.go`, `sr_diagnosticreport.go`,
  `dicom_imagingstudy.go`, `report.go`, `identity.go`, `subject.go`, `dicom_helpers.go`, `hl7_helpers.go`.

The file lists name the units the 90% target protects; they track the source tree and are re-checked when the tree
moves. A new critical path (a new standard subsystem, or a new server entry point on an existing one) is added to this
enumeration as it ships, so it stays the authoritative map of what the coverage contract is meant to guarantee. The set
deliberately **excludes** the generated FHIR R4/R5 trees (`fhir/r4`, `fhir/r5`), which are gated by the byte-for-byte
regeneration test rather than unit coverage, and the pure-glue packages (`dicomweb/auth/*`, `server`, `logging`); the
`fhir` entry above is the hand-written validate/decode core (`validate.go`, `resource.go`, `primitive.go`,
`binding.go`), not the generated tree.

Honesty over a green dashboard: raising every critical-path package to 90% is a larger test effort than one increment,
so rather than lower the 90% bar or silently exclude the packages still short of it, the gate splits the set in two and
the short packages are a documented TODO carried in [`tools/cover-critical.sh`](../../tools/cover-critical.sh):

- **Enforced at 90% today.** `dimse/dul` (94.5%). The gate FAILS if it drops below 90%. This is the live, biting 90%
  contract.
- **Ratchet (TODO toward 90%).** The remaining critical-path packages are below 90% on union coverage today and are
  being brought up: `hl7v2` (89.9%), `dimse/pdu` (89.7%), `fhir` (88.7%), `dicom` (85.8%), `dimse` (83.8%),
  `dimse/acse` (83.4%), `convert` (81.8%), and `dicomweb` (80.1%). Each carries a recorded baseline in the gate; the
  gate FAILS if any regresses more than **0.5 percentage points** below its baseline, so a TODO package can only move
  toward 90%, never meaningfully backslide. The 0.5pp tolerance is deliberate, not a hidden weakening: union coverage
  from the `-race -coverpkg=./...` run is not bit-stable run-to-run (which tests exercise cross-package code shifts
  slightly under `-race` timing), so a baseline pinned to the exact measured value false-fails on sub-1% noise. The
  tolerance absorbs that jitter while still catching a real regression (a drop of more than 0.5pp). It applies only to
  the per-package no-regression baseline; the 90% target itself carries **no** tolerance — it is a fixed bar, and an
  enforced package must be at or above 90.0%. When a ratchet package reaches 90% the gate prints a `PROMOTE` notice
  (and fails until acted on) so it is moved into the enforced set and this list is updated. The percentages above are
  the union-coverage numbers measured at the time of writing; re-run `tools/cover-critical.sh <profile>` to print the
  current figures.

## Concurrency and race posture

go-radx is a concurrent library: the servers accept connections and dispatch handlers on their own goroutines. The
target contract is that each public type states its concurrency posture in its godoc — some safe for concurrent use
(`dimse.Server` declares this), others deliberately single-flight (`dimse.Association` documents that it is not safe
for concurrent queries, because `LastError()` is per-association; run one `Find`/`Get`/`Move` iterator per association
at a time). The library does **not** claim every public API is concurrent-safe, and the contract is **not yet complete
on every surface**: `dicomweb.Server` and its `Handler` do not yet state a concurrency posture in their godoc, and
documenting it (and proving it under the race gate) is an open item on the per-server checklist below, not a satisfied
one. Where a posture is stated, the race gate exists to catch a violation of either kind — a type that should be safe
but races, or a single-flight type misused under test. That stated-and-proven contract is held by a
**standing required gate**, not by review alone. The `cover` mise task — the
one the `lint-test` job invokes through `cover:check` — is a single
`go test -race -covermode=atomic -coverpkg=./... ./...`
over the root module, so `-race` and the [coverage floor](#coverage-targets-and-critical-path-enumeration) are enforced
by the same pass rather than two. `-covermode=atomic` is the coverage mode the race detector requires, so adding
coverage did not weaken the race gate; the two are coupled by construction. The same race run is available locally via
`mise run cover:check` (or per-package via `mise run test:<subsystem>`, each of which is a `go test -race` task), and
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) lists `go test -race ./...` as a local pre-merge gate. The gate is standing:
it is not opt-in per package and a new package inherits it the moment it has a test binary, because `./...` enumerates
the whole module.

The standing race gate is the **pure-Go default unit-test build** gate, but it is no longer the *only* race surface. The
`codecs` job builds the C-backed pixel codecs (`OpenJPEG`, `libjpeg-turbo`, `CharLS`) under cgo with the
`dicom_openjpeg dicom_libjpeg dicom_charls` build tags, and after its Release `go test -tags "…" ./dicom/...` step it
runs two sanitiser passes over the codec packages:

- **A `-race` pass** (`go test -race -tags "…" ./dicom/...`) against the clean Release-built libraries. The codec
  transcoders are synchronous per-call functions today with no goroutines of their own, so this is a guard that a
  future concurrent codec path (a worker pool, shared mutable state) cannot land a Go data race unnoticed; it also runs
  the hostile-input subprocess harness under `-race`.
- **An ASAN + UBSan pass** (`go test -asan -tags "…" ./dicom/...`) against the codec libraries *rebuilt from the
  cached, SHA-256-verified source with `-fsanitize=address,undefined -shared-libasan`*. Go's `-asan` instruments the Go
  heap and the cgo boundary; the ASAN-rebuilt C libraries add the codec internals, so a heap-buffer-overflow,
  use-after-free, or undefined-behaviour fault on a malformed JPEG/JPEG2000/JPEG-LS/HTJ2K codestream
  (`codec_*_hostile_test.go`) aborts the job (`ASAN_OPTIONS`/`UBSAN_OPTIONS` set `halt_on_error=1`) rather than silently
  corrupting memory. `-race` and `-asan` are never combined in one binary (both runtimes intercept allocation), so they
  run as two passes; `-asan` is supported on the linux/amd64 runner but **not** darwin/arm64, so the ASAN pass is
  CI-only and does not run on a macOS dev box. If a sanitiser flags a real fault it is fixed, not suppressed.

  **Scope of the sanitiser gate.** AddressSanitizer (memory safety) runs in full across everything — go-radx's cgo
  glue, the cgo boundary, the Go heap, and the vendored codec internals — and is never narrowed. UBSan, by contrast,
  does **not** gate on style or function-pointer undefined behaviour *inside* the vendored upstream codecs
  (OpenJPEG 2.5.4, CharLS 2.4.3, libjpeg-turbo 3.1.4). Strict UBSan's `-fsanitize=function` fires on those libraries'
  own function-pointer casts — for example OpenJPEG calling `opj_j2k_setup_decoder` through a generic function-pointer
  type at `openjpeg.c:434` — which are legitimate by C convention but are third-party code go-radx neither owns nor
  can fix upstream. The codec source trees are therefore excluded from UBSan instrumentation via a committed
  ignorelist ([`tools/ubsan-ignorelist.txt`](../../tools/ubsan-ignorelist.txt), wired into `-fsanitize-ignorelist` for
  both the codec rebuild and the test step). UBSan still fully checks go-radx's own code and the cgo glue; the
  ignorelist only suppresses UBSan, not ASAN, so memory-safety coverage of the codec internals is untouched. In short:
  the gate covers go-radx memory safety, the cgo boundary, and our codec usage; it does not police upstream OpenJPEG /
  CharLS / libjpeg-turbo for style-level UB.

One test surface still sits outside the race detector by deliberate scope decision: the `interop` matrix legs
(`mise run interop:<leg>`) run without `-race`. They are `go test -tags interop -count=1` runs that drive real
containerised origin servers (Orthanc, dcm4chee-arc), where the failure modes that matter are wire-protocol and
round-trip correctness against an external server, not in-process Go data races, and a `-race` build would add
instrumentation overhead to an already container-bound, resource-heavy leg. The concurrency that matters in-process
lives in the pure-Go servers, which the `lint-test` race gate covers in full. If an interop test grows an in-process
concurrent client harness worth racing, that decision is revisited and a `-race` run is added at that point; until then
the interop gate stays a correctness gate.

### Per-server race checklist

A server entry point is the highest-value race surface in the library: it multiplexes inbound work across goroutines
under a shared lifecycle (accept, dispatch, shutdown). Each server increment — a new server, or a new concurrent
capability on an existing one — MUST exercise the following concurrent surfaces under `-race` before it ships, so the
standing gate has assertions to bite on rather than passing vacuously. The list is the union of what a correct server
lifecycle must survive; not every item applies to every server, but an increment must justify any it skips.

- **Concurrent clients against one server.** Drive the server from multiple goroutines at once (multiple associations,
  HTTP requests, or MLLP messages in flight) and assert no data race on shared server state.
- **Bounded concurrency under load.** If the server caps concurrency (the `dimse.Server` semaphore is the model:
  capacity is enforced *before* a handler goroutine is spawned, never after N+1 already exist), flood it past the cap
  and assert excess work is refused without spawning unbounded goroutines or racing the counter.
- **Graceful shutdown while work is in flight.** Park a handler mid-operation, call `Shutdown`, and assert
  connections close and goroutines drain without a race between the accept loop, the in-flight handlers, and the
  shutdown path.
- **Idempotent and deadline-bounded shutdown.** Call `Shutdown` more than once, and call it with a deadline that
  expires while a handler is still parked, asserting repeated calls share one waiter rather than leaking a goroutine
  per call and that a second call with a fresh deadline still completes.
- **Context cancellation stops the server.** Cancel the context passed to `ListenAndServe` (or the serve equivalent)
  and assert the accept loop and its goroutines unwind cleanly.
- **Per-connection timeouts.** Where the server enforces idle, negotiation, or completion timeouts, assert a timeout
  fires on its own goroutine without racing a concurrent shutdown or a slow but legitimate large transfer.

The surfaces each current and planned server must clear under `-race`:

- **`dimse.Server`** (shipped). The accept loop, the semaphore-bounded spawn, `Shutdown` (idempotent,
  deadline-bounded, and retried after a deadline), context-cancel stop, and the idle, negotiation, and completion
  timeouts. This server is the reference implementation of the checklist below.
- **`dicomweb.Server`** (handler shipped; race coverage outstanding). An `http.Handler` mounted under a caller's mux.
  Its functional behaviour is tested (store, retrieve round-trip, fail-closed store, content negotiation), but those
  tests are sequential: the concurrent-request surface is **not yet** exercised under `-race`, and the public `Server`
  and `Handler` godoc does not yet state a concurrency posture. The outstanding work is to document that posture, what
  a `StoreBackend`/retrieve backend must guarantee under concurrency, and add a concurrent-request race test; the full
  server lifecycle (shutdown, drain) is added once it owns an `http.Server` rather than only exposing a handler.
- **MLLP server (HL7 v2)** (planned). The accept loop over MLLP framing, concurrent message handling, ACK/NACK
  ordering, and graceful shutdown — the same checklist as `dimse.Server`, since both are connection-accepting servers.
- **FHIR server role** (planned). An `http.Handler` over FHIR resource routes: concurrent reads and writes,
  reference integrity held under concurrency, and shutdown.
- **daemon composition root** (`cmd/radx`, planned). The process that composes the servers above behind one
  lifecycle: concurrent server startup, a shared signal-driven shutdown that drains every server, and no shared
  mutable state across them.

The `dimse.Server` suite ([`dimse/server_test.go`](../../dimse/server_test.go)) is the reference implementation of this
checklist today: `TestServerMaxAssociationsRefusesBeforeSpawn`,
`TestServerShutdownClosesConnectionsWhileHandlerBlocked`, `TestServerShutdownIsIdempotent`,
`TestServerShutdownRetryAfterDeadline`, `TestServerListenAndServeStopsOnContextCancel`, and the idle, negotiation, and
completion-timeout tests each pin one item of the list above and all run under the standing `-race` gate. A new
server's increment is not done until its tests cover the applicable items the same way.

### Known intermittent race (not yet root-caused)

Honesty about the standing-gate claim requires recording one open risk. A single `go test -race ./...` failure was
observed **once** on GitHub CI, on commit `e2c12ab`, and did **not** reproduce in three local race runs of the full
suite afterward. It has **not** been root-caused, so it is not yet possible to say which package or which concurrent
surface produced it, nor to assert it is a test-harness artefact rather than a real data race. It is recorded here as a
**known, not-yet-reproduced intermittent** flagged for M8 hardening, where the work is to reproduce it under a stress
loop (`go test -race -count=N -run …`), localise it, and either fix the underlying race or prove the flake is in test
setup. Until then the standing-race-gate claim is accurate but not absolute: the gate runs on every change and the
current concurrent code passes it, yet one historical intermittent remains open and unexplained.

## Hostile-input robustness

go-radx parses untrusted bytes on every trust boundary — a DICOM Part 10 stream, a DIMSE PDU, an HL7 v2 frame, a
DICOMweb JSON or multipart body, a FHIR resource, and a compressed pixel codestream. The robustness contract (PRD §9.3)
is that a malformed, truncated, oversized, or adversarial input must surface a *typed error* rather than panic, hang, or
exhaust memory. Two CI gates hold that contract.

### Bounded fuzz smoke

The `fuzz` job runs a bounded smoke pass over every committed fuzz target. The target list is *discovered* from
`go test -list '^Fuzz'` (the `fuzz` mise task), not hand-maintained, so it cannot drift out of step with the tree; each
target runs for a fixed `-fuzztime` against its seed corpus, wrapped in coreutils `timeout` so a wedged target is killed
and the gate bites — a hang is a FAILURE, never a skip. This is a regression gate that mutates outward from the seeds,
not a fuzzing campaign.

### Hostile-input memory-capped corpus

The `hostile-corpus` job (`mise run hostile:corpus`) replays the *fixed, committed* malformed-input corpus through its
parsers under an **enforced memory ceiling** and a wall-clock timeout, asserting no parser OOMs, panics, or hangs on
hostile bytes. Where the `fuzz` job explores outward from the seeds, this gate proves the committed corpus survives a
tight memory cap on every run. It is two passes, both run as `GOMEMLIMIT=<cap> timeout <wall> go test …`:

- **The raw malformed corpus.** The harness at [`internal/hostilecorpus`](../../internal/hostilecorpus) walks the
  on-disk corpus under [`dicomweb/testdata/malformed`](../../dicomweb/testdata/malformed) — the DICOM-JSON and
  multipart/related files that are byte-for-byte the parser inputs — and feeds each to the exported `dicomweb`
  parsers (`UnmarshalJSON`; `NewMultipartReader` + a `NextPart` drain), mirroring the consumption convention the
  package fuzz targets use. It recovers any panic into a named failure and logs the peak heap each file drove so a
  creep toward the cap is visible before it becomes an OOM.
- **The Go-fuzz seed corpora.** A plain `go test -run '^Fuzz'` of the `dicom`, `dimse/pdu`, `hl7v2`, `fhir/r5`, and
  `dicomweb` packages replays every committed seed once as a subtest (no `-fuzz`), so each parser's seeds cross it under
  the same cap.

The enforcement is the process, not the test code: `GOMEMLIMIT` is a soft limit the GC works to honour, set well above
the corpus's tiny logged peak heap, so a parser that over-allocates past it is taken into a hard out-of-memory abort (a
non-zero exit, a FAILURE), and a parser that wedges is killed by `timeout` (exit 124, also a FAILURE). The cap and wall
budget are tunable via `HOSTILE_MEMLIMIT` and `HOSTILE_TIMEOUT`; the defaults are a 512 MiB cap and a 300 s wall.

## Conformance-drift methodology

A conformance statement is only worth citing if it cannot quietly fall out of step with the code it describes. Two
mechanical gates keep these statements honest as the implementation moves: a drift check that fails when a countable or
structural claim diverges from the code, and a documentation-site build that fails when a statement falls out of the
site navigation.

### The drift check

The drift check lives in `tools/conformance-drift` and runs as `go test ./tools/conformance-drift/...` (also exposed as
`mise run conformance-drift`). It compares three classes of claim against the code and fails the test on any mismatch:

- **Countable preset claims.** The "Presentation-context preset summary" table in [`./dicom.md`](./dicom.md) names each
  `dimse` presentation-context preset and the number of contexts it returns. The check parses that table and, for every
  preset, asserts the function exists and that its live context count (for example `len(dimse.StorageContexts())`)
  equals the documented number. A preset the table marks **NOT YET SHIPPED** is asserted to be *absent* from the public
  API, so a deferred surface cannot be quietly shipped without updating the statement, and a preset present in code but
  missing from the table is also surfaced. This catches both directions of drift: a count that changes in code without a
  doc update, and a doc that names a preset the code does not (or no longer) provides.
- **Not-yet-shipped banners.** Every scaffold statement for an unimplemented or not-yet-authored surface —
  [`./dicomweb.md`](./dicomweb.md), [`./dimse.md`](./dimse.md), [`./convert.md`](./convert.md),
  [`./cli-server.md`](./cli-server.md), and this cross-cutting statement — must carry the `NOT YET SHIPPED` banner.
  The check fails if any of these drops its banner, so an unfinished surface can never be silently presented as
  conformance-guaranteed by deleting the warning.
- **Stability markers.** Each top-level public package (`convert`, `dicom`, `dicomweb`, `dimse`, `fhir`, `hl7v2`,
  `server`) must carry its one-line `Stability:` godoc marker described under
  [Governance and stability posture](#governance-and-stability-posture). The check fails if any package drops it, so
  the stability posture stated here stays reflected in every package's godoc.

The check is proven to bite, not merely assumed to: alongside the live gate, the test suite mutates temporary copies of
the real tree to introduce each drift class in turn — a wrong preset count, a preset removed from the code, a
code-only preset absent from the table, a deferred preset that is suddenly defined, a removed banner, and a stripped
stability marker — and asserts the matching failure is reported, with a companion case asserting an unmutated copy
stays clean.
Preset existence is read from the `dimse` package source, so a preset added to the code is surfaced even if nobody
updates the count registry. The real statements and sources are never mutated. As new countable claims are added to a
statement, they are wired into this check so the statement and the code stay locked together.

### The documentation-site build

The statements are published as an [mkdocs](https://www.mkdocs.org/) site configured by `mkdocs.yml` at the repository
root, built with `mkdocs build --strict` (exposed as `mise run docs:build`; `mise run docs:serve` previews it with live
reload). Strict mode turns navigation drift into a build failure: a statement added under `docs/` but missing from the
site navigation, or a navigation entry pointing at a missing or excluded file, aborts the build. This keeps the set of
published statements in step with the `docs/` tree, so a new statement cannot be authored and then silently left out of
the site, and a removed one cannot linger as a dead navigation entry.

Both gates run locally (`mise run conformance-drift`, `mise run docs:build`) and as CI jobs: the `conformance-drift`
job runs the drift check, and the `docs` job provisions the pinned `mkdocs` toolchain and runs the strict site build.
Like every CI job they report status on each push and pull request but remain advisory rather than merge-blocking —
see [Gate enforcement status](#gate-enforcement-status).

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

The CI workflow at `.github/workflows/ci.yml` runs on every push and pull request to `main` and defines fourteen jobs.
The core build and test jobs are:
`lint-test` (gofmt, `go vet`, golangci-lint on the default and interop builds, `go build`, the `pin-drift` check, the
standing [`-race` gate](#concurrency-and-race-posture) step that also enforces the
[80% coverage floor](#coverage-targets-and-critical-path-enumeration), and the
[critical-path coverage gate](#coverage-targets-and-critical-path-enumeration) step that enforces the per-package 90%
target on the untrusted-input and conversion core),
`conformance` (the `dciodvfy` and `pydicom` gates with `CI=true`),
`fhir-conformance` (the FHIR R4 + R5 conformance gate that marshals the go-radx workflow set and validates it with the
pinned HL7 validator), `interop` (the testcontainers matrix over the DIMSE,
DICOMweb, and convert legs), `govulncheck` (the vulnerability scan of the root module), `cmd-radx` (build, vet, lint,
and vulnerability scan of the `cmd/radx` CLI module), and `codecs` (the C-backed pixel codecs built from source, then a
[`-race` pass and an ASAN + UBSan pass](#concurrency-and-race-posture) over the codec tests including the hostile
pixel-data corpora).

The Phase 0 Lane-A artifacts and the hostile-input gates are wired as their own jobs so each runs on every push and pull
request: `phi-sanity` (the PHI-default log sweep, `internal/phisweep`), `fuzz` (a bounded smoke run over every committed
fuzz target — the list is discovered from `go test -list '^Fuzz'` rather than hand-maintained, each target wrapped in
`timeout` so a hang is a failure, never a skip; see [Bounded fuzz smoke](#bounded-fuzz-smoke)),
`hostile-corpus` (the [hostile-input memory-capped corpus gate](#hostile-input-memory-capped-corpus) that replays the
malformed corpus and the fuzz seed corpora under `GOMEMLIMIT` + `timeout`, so an OOM, panic, or hang is a failure),
`benchmark-baseline` (a run-once pass
over the `dicom` benchmarks so the benchmark code and the committed baselines under `docs/conformance/benchmarks/`
cannot rot), `conformance-drift` (the drift check at `tools/conformance-drift`), `docs` (the strict
`mkdocs build --strict` site build on a pinned `mkdocs` toolchain), and `tracked-binary-hygiene` (fails if a compiled
binary is tracked under `cmd/`, which is committed as source only). All fourteen jobs report status on every pull
request.

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
