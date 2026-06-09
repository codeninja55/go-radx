# Embeddable servers and reference daemons

!!! warning "Planned design — not yet implemented"

    This document describes the planned `server` package. The package is not yet implemented: `server/` currently
    contains only a package doc. The composition layer, pluggable backends, and reference daemons below are the
    contract the implementation will conform to as the package is built.

The `server` package is go-radx's composition layer for the receiving side of the radiology workflow. The four
server roles — the DIMSE SCP (including a Modality Worklist SCP), the DICOMweb server, the FHIR REST server, and the
HL7 v2 MLLP server — already expose their own embeddable handlers in their respective packages (`dimse`, `dicomweb`,
`fhir`, `hl7v2`). What `server` adds is the cross-cutting glue every deployment needs and no single protocol package
should own: a set of small pluggable backends (object store, catalogue, authenticator), shared observability wiring
(structured logging with `zap` and OpenTelemetry tracing and metrics, never carrying PHI), shared listener policy
(loopback bind by default, with an explicit `--bind`/`WithBind` opt-in), and shared graceful-shutdown lifecycle. On top
of those primitives it ships **thin reference daemons** that wire sane defaults — a filesystem object store and a
SQLite catalogue — so the `radx` CLI and a first-time user get something runnable immediately.

This document is the public-API contract for the `server` package. The implementation will conform to it. Where the product
requirements document (`docs/prd/go-radx-prd.md`) committed a behaviour in §7.2 (servers), §9.1 (PHI and bind
defaults), §9.2 (honest failure), §9.4 (concurrency), §9.7 (transport security), and §9.10 (observability), this
document honours it. Terminology follows the project glossary (`UBIQUITOUS_LANGUAGE.md`); in particular the package
reuses `dicom.DataSet`, `dicom.UID`, `dimse.AETitle`, `dimse.Status`, and `dicomweb.ResourcePath` rather than
re-declaring them, and never introduces a bare ambiguous type.

The line this package draws is deliberate and stated up front. "Production-ready" here means the building blocks are
concurrency-safe, observable, shut down gracefully, default to safe binds, and the reference daemons are deployable for
development and the CLI. It does **not** mean a full PACS or archive product; that is an explicit non-goal (PRD §3.2).
The reference daemons are a runnable default, not an archive — production storage, indexing, retention, encryption at
rest, and access control remain the integrating consumer's responsibility (PRD §9.1, §9.5).

## Scope

In scope for v1:

- Pluggable backend interfaces shared across the server roles: `ObjectStore`, `Catalogue`, `WorklistSource`, and
  `Authenticator`, each segregated by responsibility (ISP, PRD §8.2) so a deployment implements only what it serves.
- A composition root (`Daemon`) that owns shared listener policy, the logger and tracer, and the lifecycle of the
  mounted server roles.
- **Loopback-bind defaults** for every role, with one explicit opt-in (`WithBind`, surfaced to the CLI as `--bind`).
- **Graceful shutdown**: stop accepting, drain in-flight work bounded by a context deadline, release listeners, in a
  fixed order across all mounted roles.
- **Observability hooks**: a `zap.Logger` and an OpenTelemetry `trace.TracerProvider` / `metric.MeterProvider`, both
  honouring the no-PHI rule (PRD §9.1, §9.10). OTel exports nowhere by default; it is operator-opt-in.
- The four reference daemons (DIMSE SCP with optional Modality Worklist SCP, DICOMweb, FHIR REST, HL7 v2 MLLP) wired to
  the default filesystem object store and SQLite catalogue. The DIMSE SCP is launched by `radx scp` and the MLLP server
  by `radx hl7 listen` (see [radx CLI](cli.md)); the DICOMweb and FHIR REST daemons are embeddable library features in
  v1 with no dedicated CLI subcommand (the `dicomweb`/`convert` CLI groups are clients, not servers — cli.md Scope).
- A minimal **FHIR REST** server surface (the conformance subset of `read`, `create`, `search-type`, and `transaction`
  for the workflow resource set) over a pluggable `server.Repository`, serving a single FHIR release fixed at role
  construction (see "FHIR REST server").

Out of scope for v1 (architected-for, deferred — PRD §3.2, §5.1):

- The SCP / server sides of MPPS and Storage Commitment (the v1 N-services are SCU-only; the SCP cannot answer them).
- A full PACS/archive: multi-tenant access control, retention/erasure, replication, HA, and audit storage are the
  consumer's product, built on top of these primitives.
- SMART on FHIR, FHIR Subscriptions, and the full FHIR REST capability surface (only the conformance subset is served).
- DICOMweb rendered/thumbnail retrieval, WADO-URI, and UPS-RS (see [DICOMweb](dicomweb.md) for that boundary).

## How `server` relates to the protocol packages

Each protocol package already defines the handler or backend interface a caller implements to answer inbound requests.
The `server` package does not replace those; it provides default implementations of them backed by the shared backends,
plus the lifecycle and observability that sit above all four. The relationship is:

| Role | Handler/backend interface (defined in) | Wire format |
|------|----------------------------------------|-------------|
| DIMSE SCP | `dimse.Handler` (`dimse`) | DICOM Upper Layer over TCP/TLS |
| Modality Worklist SCP | `dimse.FindHandler` over a `WorklistSource` (`server`) | DIMSE C-FIND, MWL information model |
| DICOMweb | `dicomweb.StoreBackend` / `RetrieveBackend` / `QueryBackend` (`dicomweb`) | HTTP/TLS, `dicom+json` |
| FHIR REST | `server.Repository` (`server`, over `fhir`) | HTTP/TLS, `application/fhir+json` |
| HL7 v2 MLLP | `hl7v2.Handler` (`hl7v2`) | MLLP over TCP/TLS |

The DIMSE, DICOMweb, and MLLP server *types* (`dimse.Server`, `dicomweb.Server`, `hl7v2.Server`) live in their own
packages and are documented there. `server` constructs those types from the shared backends, registers them with the
`Daemon`, and applies the shared listener and observability policy. A consumer who wants only one role can use the
protocol package directly; `server` is for the common case of wiring one or more roles with shared defaults.

## Pluggable backends

The backends are the seam between go-radx and the consumer's storage and identity systems. Each is a small interface so
a deployment implements only what it needs (interface segregation, PRD §8.2): a write-only ingest node implements
`ObjectStore`; a query node implements `Catalogue`; a worklist provider implements `WorklistSource`. Every method takes
a `context.Context` for cancellation and deadline propagation (PRD §9.4).

### ObjectStore — the binary object plane

`ObjectStore` persists and retrieves whole DICOM objects keyed by their SOP Instance UID. It is the storage primitive
behind the DIMSE C-STORE / C-GET / C-MOVE handlers and the DICOMweb STOW-RS / WADO-RS backends, so one implementation
serves both protocol planes.

```go
// ObjectStore persists DICOM objects as opaque, addressable blobs keyed by SOP Instance UID. Implementations must be
// safe for concurrent use (PRD §9.4). All methods honour ctx cancellation and never log PHI (PRD §9.1).
type ObjectStore interface {
    // Put persists ds. It is idempotent on SOP Instance UID: storing the same instance twice is not an error, and the
    // implementation decides whether to overwrite or de-duplicate. Returning success without durably persisting is a
    // defect (PRD §9.2 honest-failure rule, which the prototype's store path violated).
    Put(ctx context.Context, ds *dicom.DataSet) error

    // Get retrieves one stored object by SOP Instance UID. It returns ErrNotFound (errors.Is-comparable) when absent.
    Get(ctx context.Context, instance dicom.SOPInstanceUID) (*dicom.DataSet, error)

    // Exists reports presence without materialising the object, so a STOW-RS de-duplication check is cheap.
    Exists(ctx context.Context, instance dicom.SOPInstanceUID) (bool, error)

    // Delete removes one object. It returns ErrNotFound when the instance is absent so a caller can distinguish a
    // no-op delete from a successful one (never a silent success on missing data).
    Delete(ctx context.Context, instance dicom.SOPInstanceUID) error
}
```

### Catalogue — the queryable metadata plane

`Catalogue` is the index the C-FIND and QIDO-RS handlers query. It is deliberately separate from `ObjectStore`: the
binary plane stores bytes, the catalogue answers questions about them. The catalogue holds **PHI** (patient identifiers,
names, study descriptions), so it is the component the PHI-safety defaults most directly govern (see "PHI posture").

```go
// Catalogue indexes stored objects for query. The query vocabulary is DICOM tags, matching DIMSE C-FIND and QIDO-RS, so
// a handler translates neither — it forwards the tag-keyed query. Implementations must be safe for concurrent use.
type Catalogue interface {
    // Index records or updates the queryable attributes of ds. It extracts only the attributes the supported query
    // models need; it does not store pixel data.
    Index(ctx context.Context, ds *dicom.DataSet) error

    // Query answers a hierarchical query at the given level (PATIENT/STUDY/SERIES/IMAGE). It streams results as a Go
    // 1.23+ iterator so a large match set is not buffered (the iterator convention used across go-radx, PRD §8.1).
    // Each yielded DataSet carries the requested return keys; the terminal iteration carries a nil error when clean.
    Query(ctx context.Context, q CatalogueQuery) iter.Seq2[*dicom.DataSet, error]

    // Remove drops the index entry for one instance (paired with ObjectStore.Delete).
    Remove(ctx context.Context, instance dicom.SOPInstanceUID) error
}

// CatalogueQuery is a normalised query shared by the C-FIND and QIDO-RS handlers, so both planes hit one index API.
type CatalogueQuery struct {
    Level   dimse.QueryLevel     // PATIENT, STUDY, SERIES, or IMAGE
    Match   map[dicom.Tag]string // tag -> match value (single-value, range, wildcard, or UID-list matching)
    Return  []dicom.Tag          // attributes to return beyond the level's defaults
    Limit   int                  // 0 means no limit (QIDO-RS limit=); DIMSE C-FIND ignores this
    Offset  int                  // QIDO-RS offset=; DIMSE C-FIND ignores this
    Fuzzy   bool                 // QIDO-RS fuzzymatching=true
}
```

The matching rules (single-value, list-of-UID, wildcard, range, universal, and sequence matching per DICOM PS3.4 Annex
C) are applied by the handler before it reaches the `Catalogue`, or by the `Catalogue` itself for the SQL backend; the
contract is that the iterator yields exactly the matching objects and never silently over- or under-matches.

### WorklistSource — the Modality Worklist plane

The Modality Worklist SCP answers a DIMSE C-FIND against the Modality Worklist Information Model. Because a worklist is
not the same data as the stored-object catalogue (it is scheduled procedure steps, typically fed from an HL7 `ORM`/`OMG`
order or a FHIR `ServiceRequest`), it has its own backend. This is the SCP side of workflow step 2 (PRD §5.1), built so
the worklist leg is testable end-to-end against the `dimse` C-FIND SCU.

```go
// WorklistSource answers Modality Worklist queries. The query is a DIMSE C-FIND identifier (tag-keyed match keys
// such as ScheduledProcedureStepStartDate, Modality, and the ScheduledProcedureStepSequence), and each yielded
// DataSet is one matching Scheduled Procedure Step worklist item.
type WorklistSource interface {
    // Find yields one worklist item per match. Matching follows the MWL information model (PS3.4 Annex K). The iterator
    // terminates with a nil error on clean completion; a backend failure terminates it with a typed error that the
    // handler maps to a DIMSE failure status (never a laundered success, PRD §9.2).
    Find(ctx context.Context, query *dicom.DataSet) iter.Seq2[*dicom.DataSet, error]
}
```

The handler that adapts a `WorklistSource` to a DIMSE C-FIND emits a `dimse.Status` of `0xFF00` (Pending, current match
supplied) for each yielded item and a terminal `0x0000` (Success) when the iterator ends cleanly, mirroring the
`pynetdicom` Modality Worklist service class. A backend error becomes the appropriate failure status (for example
`0xA700` out of resources, or `0xC000` unable to process) with the cause logged structurally and without PHI.

### Authenticator — the identity plane

Authentication is pluggable and uniform across the HTTP roles (DICOMweb, FHIR REST) and, where applicable, the DIMSE
role (mapping to DICOM user-identity negotiation, PS3.7 D.3.3.7). The default daemons authenticate nothing beyond the
listener policy; a production deployment supplies an `Authenticator`.

```go
// Authenticator verifies a request's credentials and returns the authenticated Principal, or an error. It is consulted
// by the HTTP middleware (bearer token, mutual-TLS client certificate) and by the DIMSE association handler (DICOM
// user-identity). Credentials come from the request only; secrets are never logged (PRD §9.8).
type Authenticator interface {
    // AuthenticateHTTP inspects an HTTP request's Authorization header and/or verified client certificate.
    AuthenticateHTTP(ctx context.Context, r *http.Request) (Principal, error)

    // AuthenticateDIMSE inspects a DICOM user-identity negotiation item presented during A-ASSOCIATE.
    AuthenticateDIMSE(ctx context.Context, id dimse.UserIdentity, calling dimse.AETitle) (Principal, error)
}

// Principal is the authenticated identity. It carries no secret material — only the identity and any coarse scopes a
// deployment chooses to attach — so it is safe to put in a request context and (its ID, not its tokens) in a span.
type Principal struct {
    ID     string   // stable subject identifier (username, certificate subject, token subject)
    Scopes []string // optional, deployment-defined authorisation scopes
}

// AllowAll is the explicit no-authentication Authenticator used by the loopback reference daemons. It is named so a
// deployment cannot enable "no auth" by accident: choosing AllowAll is a visible, reviewable decision.
func AllowAll() Authenticator
```

`AllowAll` exists so that "no authentication" is always an explicit, named choice rather than a silent default the
reader of a deployment must infer. The reference daemons use it precisely because they bind to loopback; a non-loopback
bind that still uses `AllowAll` is a configuration the operator owns and the bind-default test (PRD §11.2) is designed
to make conspicuous.

## The Daemon composition root

A `Daemon` is the composition root: it owns the shared logger and tracer, the listener policy, and the set of mounted
server roles, and it drives their start and graceful shutdown as a unit. It carries no global mutable state — every
knob is per-`Daemon` and is set with functional options (PRD §9.4).

```go
// Daemon composes one or more server roles behind shared observability and lifecycle policy. Construct one per process.
// Safe for concurrent use after construction.
type Daemon struct { /* unexported */ }

// New builds a Daemon with the given options. Zero options yield a daemon with loopback-only binds, a no-op logger, a
// no-op tracer, and no roles mounted (a daemon with no roles serves nothing and Run returns immediately).
func New(opts ...Option) (*Daemon, error)

type Option func(*config)

// Observability. Both default to no-ops; neither exports anything until the operator wires it (PRD §9.10).
func WithLogger(l *zap.Logger) Option
func WithTracerProvider(tp trace.TracerProvider) Option
func WithMeterProvider(mp metric.MeterProvider) Option

// Listener policy. WithBind is the single explicit opt-out of loopback-only binding (see "Bind policy").
func WithBind(host string) Option        // e.g. "0.0.0.0" or a specific interface IP; surfaced to the CLI as --bind
func WithTLS(cfg *tls.Config) Option     // applies to every HTTP and DIMSE-TLS role; TLS 1.2+ enforced (PRD §9.7)
func WithShutdownTimeout(d time.Duration) Option // default 30s; bounds graceful drain

// Authentication, applied to every role that supports it.
func WithAuthenticator(a Authenticator) Option // default AllowAll() ONLY when bound to loopback (see "Bind policy")

// Role mounting. Each returns an Option so roles compose in one New call.
func WithDIMSE(role *DIMSERole) Option
func WithDICOMweb(role *DICOMwebRole) Option
func WithFHIR(role *FHIRRole) Option
func WithMLLP(role *MLLPRole) Option
```

```go
// Run starts every mounted role and blocks until ctx is cancelled or any role fails to start. On ctx cancellation it
// performs a graceful shutdown of all roles (see "Graceful shutdown") bounded by the configured shutdown timeout, and
// returns the first non-nil shutdown error, or nil on a clean stop. A role that fails to bind aborts startup and Run
// returns that error after stopping any roles already started (no half-started daemon is left running).
func (d *Daemon) Run(ctx context.Context) error

// Addrs reports the resolved listen address per role after a successful start, so a test that bound to ":0" can read
// the OS-assigned port. It returns nil before Run has bound the listeners.
func (d *Daemon) Addrs() map[string]net.Addr
```

Each role is built from the shared backends and its own options. The role constructors are where a consumer plugs in an
`ObjectStore`, a `Catalogue`, a `WorklistSource`, or a `server.Repository`.

```go
// DIMSERole configures the DIMSE SCP. The title is a validated dimse.AETitle (produce it with dimse.ParseAETitle, never
// a bare string); supported contexts are required; the worklist source is optional and, when supplied, mounts the
// Modality Worklist SCP alongside the C-STORE/C-FIND/C-GET/C-MOVE/C-ECHO services.
func NewDIMSERole(title dimse.AETitle, store ObjectStore, cat Catalogue, opts ...DIMSERoleOption) (*DIMSERole, error)

type DIMSERoleOption func(*dimseRoleConfig)

func WithDIMSEPort(port int) DIMSERoleOption                 // default 11112
func WithDIMSEContexts(c []dimse.PresentationContext) DIMSERoleOption // default Storage + QueryRetrieve + Verification
func WithWorklistSource(w WorklistSource) DIMSERoleOption    // mounts the Modality Worklist SCP
func WithMaxAssociations(n int) DIMSERoleOption              // capacity acquired before spawning a handler (DIMSE-013)

// DICOMwebRole configures the DICOMweb HTTP server over the shared backends.
func NewDICOMwebRole(store ObjectStore, cat Catalogue, opts ...DICOMwebRoleOption) (*DICOMwebRole, error)

type DICOMwebRoleOption func(*dicomwebRoleConfig)

func WithDICOMwebPort(port int) DICOMwebRoleOption           // default 8042
func WithDICOMwebBasePath(p string) DICOMwebRoleOption       // default "/dicom-web"
func WithMaxRequestBytes(n int64) DICOMwebRoleOption         // body cap (PRD §9.3)

// FHIRRole configures the FHIR REST server over a pluggable server.Repository bound to one FHIR release (see "FHIR
// REST server"). The release defaults to R5; set it with WithFHIRRelease and pass a repo that stores that release.
func NewFHIRRole(repo Repository, opts ...FHIRRoleOption) (*FHIRRole, error)

type FHIRRoleOption func(*fhirRoleConfig)

func WithFHIRPort(port int) FHIRRoleOption                   // default 8080
func WithFHIRBasePath(p string) FHIRRoleOption               // default "/fhir"
func WithFHIRRelease(r fhir.Release) FHIRRoleOption          // default fhir.R5; one role serves one release

// MLLPRole configures the HL7 v2 MLLP server over an hl7v2.Handler.
func NewMLLPRole(h hl7v2.Handler, opts ...MLLPRoleOption) (*MLLPRole, error)

type MLLPRoleOption func(*mllpRoleConfig)

func WithMLLPPort(port int) MLLPRoleOption                   // default 2575
func WithMaxFrameSize(n int) MLLPRoleOption                  // default 16 MiB (hl7v2.DefaultMaxFrameSize)
```

## Bind policy: loopback by default, `--bind` to opt out

Every role binds to **loopback only** (`127.0.0.1` and `::1`) unless the operator explicitly passes `WithBind` (surfaced
to the `radx` CLI as `--bind`). This is the §9.1 safe default, and it directly fixes the prototype's defect: the
prototype's SCP defaulted its listen address to `0.0.0.0:11119` / `0.0.0.0:11120` (all interfaces) while creating a
PHI-bearing SQLite catalogue, exposing patient data on every network interface out of the box (PRD §2.2; Codex
`dimse.md` listen-addr findings).

The rule, applied uniformly to all four roles:

- With **no** `WithBind`, each role's listener binds to `127.0.0.1` (and `::1` where the platform dual-stacks) on its
  configured port. The default `Authenticator` is `AllowAll()`, because the surface is reachable only from localhost.
- With `WithBind(host)`, the listeners bind to the given host. A non-loopback host is the operator's explicit decision.
  When the bind is non-loopback **and** no `Authenticator` was supplied, `New` returns an error rather than silently
  exposing an unauthenticated server to the network. To bind wide open with no auth (a development convenience), the
  operator must pass `WithAuthenticator(server.AllowAll())` explicitly — making "exposed and unauthenticated" a
  visible, reviewable choice, never an accident.

```go
// ErrInsecureBind is returned by New when a non-loopback WithBind is combined with no explicit Authenticator, so the
// fail-closed default cannot be bypassed by omission. Pass WithAuthenticator(AllowAll()) to override deliberately.
var ErrInsecureBind = errors.New("server: non-loopback bind requires an explicit Authenticator")
```

Having an `Authenticator` is not enough on its own; each role must actually run it against its callers. The DIMSE role
wires the `Authenticator` into the association-accept layer: an unauthorized Calling AE Title is refused with an
A-ASSOCIATE-RJ before any C-ECHO/C-STORE/C-FIND runs, so a non-loopback DIMSE bind authenticates every association
rather than serving it. The DICOMweb role runs the `Authenticator` as HTTP middleware on every request.

MLLP is the exception. The protocol carries no application-level identity, so a generic `Authenticator` cannot gate it.
For MLLP, network exposure requires **mutual TLS**: a non-loopback MLLP bind must terminate a TLS config whose
`ClientAuth` is `tls.RequireAndVerifyClientCert` so the transport, not the message, authenticates the peer. A
non-loopback MLLP bind without client-certificate-verifying TLS is refused with `ErrInsecureBind` at start, the same
fail-closed posture as an unauthenticated DIMSE or HTTP bind. A loopback MLLP bind is unconstrained.

The §11.2 bind-default CI check asserts that, with default options, every role listens on loopback; it is a
merge-blocking sanity test, not a compliance regime.

## Graceful shutdown

When `Run`'s context is cancelled (typically by `SIGINT`/`SIGTERM` wired by the CLI), the daemon shuts every role down
gracefully, in a fixed order, bounded by the configured shutdown timeout (default 30 seconds):

1. **Stop accepting.** Each listener stops accepting new connections or associations immediately. New DIMSE
   A-ASSOCIATE-RQ, new HTTP connections, and new MLLP frames are refused.
2. **Drain in-flight work.** Active DIMSE associations finish or are released, in-flight HTTP requests complete, and
   open MLLP frames are read to their end sequence and answered. Each role delegates to its own `Shutdown(ctx)` — for
   DIMSE this closes association connections *before* waiting on handlers, the precise fix for the prototype's
   `Shutdown` that left handlers blocked in `ReadPDU` (Codex DIMSE-014).
3. **Release listeners and flush observability.** Sockets close, the trace exporter and metric reader flush, and the
   logger syncs.

If the shutdown timeout elapses before all roles have drained, the daemon forces the remaining connections closed and
`Run` returns a wrapped deadline error naming the role that did not drain — an honest report that shutdown was not
clean, never a silent success (PRD §9.2). The shutdown is idempotent: a second cancellation or a `Shutdown` call after
`Run` has returned is a no-op.

```go
// Shutdown triggers a graceful shutdown independently of Run's context, bounded by ctx. It is safe to call concurrently
// with Run; the first to fire wins and the other observes a clean stop. Used by tests and by callers that drive the
// lifecycle without a cancellable Run context.
func (d *Daemon) Shutdown(ctx context.Context) error
```

## Observability hooks

Observability is wired through two operator-controlled providers, both defaulting to no-ops, both bound by the no-PHI
rule (PRD §9.1, §9.10). Neither emits anything to a vendor until the operator wires an exporter; this reconciles the
"no telemetry/data collection" stance (which meant outbound vendor collection) with local operator observability.

### Structured logging (zap)

The daemon and every role log through a `*zap.Logger` supplied by `WithLogger` (default: `zap.NewNop()`). Logs are
structured and human-readable at once and render DICOM, HL7, and FHIR concepts by name per the §8.2 legibility rule —
a tag as its keyword plus `(gggg,eeee)`, transfer syntaxes and SOP classes by registered name, DIMSE statuses by name
and class, FHIR resource types and element paths. At default verbosity, log fields carry **structure and identifiers
only**: AE Titles, SOP Class UIDs, study/series/instance UIDs, accessor paths, HTTP method and redacted URL, byte
offsets and limits. They do not carry patient values (`PatientName`, `PatientID` content, observation values, free
text). Surfacing PHI requires opt-in verbosity, which a deployment enables knowingly.

```go
// Log fields are emitted through helpers that guarantee the no-PHI contract. Example shape, not an exhaustive list:
//   logger.Info("c-store received",
//       zap.Stringer("sop_class", info.SOPClassUID),     // registered name, not raw UID where known
//       zap.Stringer("calling_ae", info.CallingAETitle), // identifier, not PHI
//       zap.Uint16("message_id", info.MessageID))
// A field carrying a patient value would be a defect the §11.2 PHI-default test is designed to catch.
```

### Tracing and metrics (OpenTelemetry)

Tracing and metrics use the OpenTelemetry API via `WithTracerProvider` and `WithMeterProvider` (defaults: the no-op
providers, exporting nowhere). Each role wraps its request handling in a span named for the operation (for example
`dimse.c-store`, `dicomweb.stow-rs`, `fhir.transaction`, `hl7v2.mllp-receive`). Span attributes and metric labels follow
the same no-PHI rule as logs: identifiers and structure, never patient values. A study UID may appear as a span
attribute (it is an identifier, not PHI in the §9.1 sense for diagnostics); a patient name never does.

The metric set is intentionally small and low-cardinality: request/operation counts by role and outcome, in-flight
gauges, and operation-latency histograms. High-cardinality labels (per-patient, per-study) are excluded by construction
so a metrics backend cannot become an inadvertent PHI store.

## FHIR REST server

The FHIR REST server exposes the conformance subset of the FHIR HTTP API over a pluggable repository. It serves
`application/fhir+json` only (matching the `fhir` package's JSON-only v1 scope), validates inbound resources with
`fhir.Validate`, and returns a `fhir.OperationOutcome` for every error — the FHIR-native error channel — rather than
a bare HTTP body.

### Release selection: one role serves one release

A FHIR R4 resource and the corresponding R5 resource are **distinct Go types** in distinct packages (`r4.Patient`
vs `r5.Patient`, `r4.Bundle` vs `r5.Bundle`; see [FHIR R4/R5](fhir.md), which states there is no root `fhir.Patient`
or `fhir.Bundle`). A single Go `Repository` value therefore cannot transparently store both releases through one
release-neutral type. go-radx resolves this by **fixing the release at role construction**: each `FHIRRole` serves
exactly one FHIR release, chosen with `WithFHIRRelease`, and the `Repository` it wraps stores resources of that
release. To serve both releases from one process, mount two `FHIRRole`s on different base paths (for example `/fhir/r4`
and `/fhir/r5`). A request never selects the release; the base path the request hit determines it. This mirrors how the
DIMSE role composes concrete `dimse` types rather than a release-neutral abstraction.

The wire types crossing the `Repository` seam are release-neutral only at the **interface boundary**: methods exchange
the release-agnostic `fhir.Resource` interface (every `r4`/`r5` resource satisfies it) and the role's configured
release tells the implementation which concrete type to materialise. `Bundle`, however, is release-specific, so the
search and transaction methods are typed against the configured release's `Bundle`. The interface below is shown for an
R5 role; an R4 role substitutes `r4.Bundle` for `r5.Bundle` identically.

```go
// Repository is the storage seam for the FHIR REST server, defined in the server package (use server.Repository). It is
// the FHIR counterpart of ObjectStore + Catalogue: resources are stored and searched by type and id. A Repository is
// bound to one FHIR release (chosen via WithFHIRRelease); the fhir.Resource values it exchanges are that release's
// concrete types. Implementations are safe for concurrent use.
type Repository interface {
    // Read returns the current version of one resource by type and id, or ErrNotFound. The returned fhir.Resource is a
    // concrete resource of the role's release (e.g. *r5.Patient).
    Read(ctx context.Context, resourceType, id string) (fhir.Resource, error)

    // Create stores a new resource, assigning a server id when the resource has none, and returns the stored resource.
    // It validates with fhir.Validate first; a resource with error-severity issues is rejected with that outcome.
    Create(ctx context.Context, r fhir.Resource) (fhir.Resource, error)

    // Search executes a type-level search and returns a searchset Bundle of the role's release (built with the
    // release's NewSearchSet, e.g. r5.NewSearchSet, so total and the bdl-* invariants hold). params are the raw FHIR
    // search parameters. The *r5.Bundle return is *r4.Bundle for an R4 role.
    Search(ctx context.Context, resourceType string, params url.Values) (*r5.Bundle, error)

    // Transaction processes a transaction Bundle atomically and returns the transaction-response Bundle. A failed entry
    // rolls back the whole transaction (FHIR transaction semantics), and the response reports each entry's outcome. The
    // *r5.Bundle is *r4.Bundle for an R4 role.
    Transaction(ctx context.Context, b *r5.Bundle) (*r5.Bundle, error)
}
```

`fhir.Resource`, `fhir.Validate`, and `fhir.OperationOutcome` are the release-agnostic machinery the root `fhir` package
publishes; `r5.Bundle` / `r4.Bundle` and the resource types (`r5.Patient`, `r4.Patient`, and so on) live in the
`fhir/r4` and `fhir/r5` packages, all documented in [FHIR R4/R5](fhir.md). The server introduces no parallel resource
model. The served interactions in v1 are `read`, `create`, `search-type`, and `transaction` for the workflow resource
set (`Patient`, `Encounter`, `ServiceRequest`, `ImagingStudy`, `DiagnosticReport`, `Observation`, in the role's
release); `update`, `delete`, `vread`, `history`, and `patch` are deferred and return a `405`/`501` with an
`OperationOutcome`, never a silent no-op (PRD §9.2).

The HTTP status mapping is explicit: `200` on a successful read or search, `201` on create (with a `Location` header),
`400` with an `error`-severity `OperationOutcome` when `fhir.Validate` rejects the body, `404` with an
`OperationOutcome` on a missing resource, `422` for a well-formed but unprocessable resource, and `401`/`403` from the
`Authenticator`. The base path defaults to `/fhir` (`WithFHIRBasePath`).

## Thin reference daemons

The reference daemons are the runnable defaults: each wires the shared backends to a **filesystem object store** and a
**SQLite catalogue**, binds to loopback, and uses `AllowAll()` authentication, so a daemon starts a working server with
no configuration. The DIMSE SCP daemon is launched from the `radx scp` command and the MLLP daemon from `radx hl7
listen` (see [radx CLI](cli.md)); the DICOMweb and FHIR REST daemons are embedded via the `server` package in v1, with
no dedicated `radx` subcommand. They are the development default the PRD calls for (§7.2), and they deliberately
mirror the `pynetdicom` `qrscp` reference app's shape (a SQLite database plus a filesystem instance store), which
go-radx's interop tests already exercise.

```go
// FileStore is the default ObjectStore: it persists each object as a Part 10 file under root, in a
// study/series/instance directory layout. Construct it with the dir; it creates the tree lazily.
func FileStore(root string) (ObjectStore, error)

// SQLiteCatalogue is the default Catalogue: a SQLite database indexing the queryable attributes of stored objects. The
// dbPath defaults are NOT created implicitly by any command — the daemon requires an explicit path, because the
// catalogue holds PHI (the prototype created a PHI-bearing "dicom-catalogue.db" by default; this contract forbids that,
// Codex RADX PHI finding). It offers a redacted mode that hashes direct identifiers for non-clinical use.
func SQLiteCatalogue(ctx context.Context, dbPath string, opts ...CatalogueOption) (Catalogue, error)

type CatalogueOption func(*catalogueConfig)

// WithRedaction stores one-way hashes of direct identifiers (PatientName, PatientID) instead of cleartext, for a
// non-clinical or shared-development catalogue. Off by default; redaction is a deliberate choice, not a hidden default.
func WithRedaction(enabled bool) CatalogueOption
```

Two PHI-safety rules are baked into the defaults and were defects in the prototype:

1. **No implicit PHI catalogue.** No command or daemon silently creates a PHI-bearing catalogue at a default path. The
   SQLite catalogue requires an explicit `dbPath`, and the CLI requires the operator to name the database. The prototype
   defaulted the catalogue to `dicom-catalogue.db` in the working directory and populated it with `PatientName` /
   `PatientID` / `PatientBirthDate` — a PHI store created by accident (PRD §2.2; Codex RADX catalogue findings).
2. **Loopback bind.** The reference daemons bind to loopback; exposing them is the `--bind` opt-in, which then requires
   explicit authentication for a non-loopback bind (see "Bind policy").

### Wiring the default DIMSE SCP daemon (with Modality Worklist)

```go
store, err := server.FileStore("/var/lib/radx/objects")
if err != nil {
    log.Fatalf("object store: %v", err)
}
cat, err := server.SQLiteCatalogue(ctx, "/var/lib/radx/catalogue.db") // explicit path; never an implicit default
if err != nil {
    log.Fatalf("catalogue: %v", err)
}

aet, err := dimse.ParseAETitle("RADX-SCP") // AETitle is a validated value type, never a bare string
if err != nil {
    log.Fatalf("ae title: %v", err)
}
dimseRole, err := server.NewDIMSERole(aet, store, cat,
    server.WithDIMSEPort(11112),
    server.WithWorklistSource(myWorklist), // mounts the Modality Worklist SCP; omit to serve storage/QR only
)
if err != nil {
    log.Fatal(err)
}

logger, _ := zap.NewProduction()
d, err := server.New(
    server.WithLogger(logger),
    server.WithDIMSE(dimseRole),
    // No WithBind => loopback only; AllowAll() authentication is the safe localhost default.
)
if err != nil {
    log.Fatal(err)
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
if err := d.Run(ctx); err != nil { // Run blocks until SIGINT/SIGTERM, then drains gracefully
    log.Fatalf("daemon: %v", err)
}
```

## Composing multiple roles in one daemon

The end-to-end workflow runs several roles in one process. They share the object store, the catalogue, and the
observability wiring, so a study stored over DIMSE C-STORE is immediately queryable over QIDO-RS and retrievable over
WADO-RS without a second copy.

```go
store, _ := server.FileStore("/var/lib/radx/objects")
cat, _ := server.SQLiteCatalogue(ctx, "/var/lib/radx/catalogue.db")

aet, err := dimse.ParseAETitle("RADX-SCP") // validated AETitle, not a bare string literal
if err != nil {
    log.Fatal(err)
}
dimseRole, _ := server.NewDIMSERole(aet, store, cat, server.WithWorklistSource(worklist))
webRole, _ := server.NewDICOMwebRole(store, cat) // same store + catalogue as DIMSE
mllpRole, _ := server.NewMLLPRole(orderHandler)  // HL7 v2 results/orders over MLLP
fhirRole, _ := server.NewFHIRRole(fhirRepo,      // FHIR REST over a server.Repository
    server.WithFHIRRelease(fhir.R5))             // one role serves one release; fhirRepo stores R5 resources

d, err := server.New(
    server.WithLogger(logger),
    server.WithTracerProvider(tp), // OTel exports only where the operator wired the exporter
    server.WithDIMSE(dimseRole),
    server.WithDICOMweb(webRole),
    server.WithMLLP(mllpRole),
    server.WithFHIR(fhirRole),
    server.WithBind("0.0.0.0"),                  // explicit non-loopback bind...
    server.WithAuthenticator(myAuthenticator),   // ...therefore an explicit Authenticator is required
    server.WithTLS(tlsConfig),                   // TLS 1.2+ enforced across every HTTP and DIMSE-TLS role
)
if err != nil {
    log.Fatal(err) // e.g. ErrInsecureBind if WithBind is non-loopback and no Authenticator was supplied
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
log.Fatal(d.Run(ctx))
```

## Behaviour and error model

Errors are values, typed, and rendered in human terms without PHI (PRD §8.2). The package's own behaviour follows the
same rules the protocol packages do; the cross-cutting guarantees are:

1. **Fail-closed composition.** A role that cannot start (port in use, TLS misconfigured, non-loopback bind without
   auth) aborts `New`/`Run` with a typed error; the daemon never starts a subset of roles and reports success. A backend
   that cannot perform a store or index returns an error the handler maps to a protocol failure status, never a
   laundered success (PRD §9.2). `ErrInsecureBind` and `ErrNotFound` are `errors.Is`-comparable.
2. **Truncation is failure.** Inbound payloads — DICOM datasets, multipart bodies, MLLP frames, FHIR JSON — that end
   mid-value surface `io.ErrUnexpectedEOF` through the owning protocol package; the daemon does not accept a truncated
   object as complete (PRD §9.2). Object-store `Put` is all-or-nothing per object.
3. **Hostile input is bounded.** The HTTP roles enforce body and multipart caps (`WithMaxRequestBytes`), the MLLP role
   enforces a frame cap (`WithMaxFrameSize`), and the DIMSE role enforces the PDU and association caps from the `dimse`
   package. Exceeding any returns a typed limit error and the role answers with the appropriate protocol-level
   rejection, never an OOM or panic (PRD §9.3).
4. **Context cancellation is honoured.** Every backend and handler method takes a `context.Context`; cancelling `Run`'s
   context drains the daemon, and a per-request deadline aborts that request promptly (PRD §9.4).
5. **TLS by default for exposed binds.** `WithTLS` enforces TLS 1.2+ (preferring 1.3) and peer-certificate verification;
   `InsecureSkipVerify` is reachable only through an explicitly flagged test mode, never the default path (PRD §9.7).
   Secrets (bearer tokens, client keys, DICOM passcodes) come from environment or files, are never logged, and are
   never written to the catalogue (PRD §9.8).

```go
// Sentinel errors. Compare with errors.Is.
var (
    ErrNotFound      = errors.New("server: object or resource not found")
    ErrInsecureBind  = errors.New("server: non-loopback bind requires an explicit Authenticator")
    ErrRoleNotMounted = errors.New("server: requested role is not mounted on this daemon")
    ErrShutdownTimeout = errors.New("server: graceful shutdown exceeded its deadline")
)
```

## PHI posture

The `server` package is where PHI most concentrates, so the §9.1 defaults are stated explicitly:

- **No PHI in logs, spans, or metrics by default.** Diagnostics carry identifiers and structure (AE Titles, UIDs,
  element keywords, accessor paths, HTTP method and redacted URL), not patient values. The §11.2 PHI-default CI test
  runs the daemons at default verbosity against fixtures with known PHI tokens and fails if any token surfaces in
  stdout, stderr, logs, or telemetry.
- **No implicit PHI store.** The PHI-bearing SQLite catalogue is never created at a default path; the operator names it.
  A redacted mode (`WithRedaction`) hashes direct identifiers for non-clinical use.
- **Loopback by default.** A non-loopback `--bind` is explicit and, without an `Authenticator`, fails closed
  (`ErrInsecureBind`).
- **Governance is the consumer's.** Encryption at rest, retention/erasure, access control, and audit storage are the
  integrating consumer's responsibility (PRD §9.1, §9.5). The data-modification audit hook envisaged in PRD §9.5 is
  **not yet implemented**: the package exposes no audit or modification-callback seam, so a consumer cannot currently
  wire one. Its shape is deferred and tracked in issue
  [#113](https://github.com/codeninja55/go-radx/issues/113).

## Conformance scope and limits

The authoritative, versioned scope lives in the per-standard Conformance Statements (`docs/conformance/`); this section
states the v1 server boundary so the API contract is self-contained.

What v1 provides:

- Pluggable `ObjectStore`, `Catalogue`, `WorklistSource`, and `Authenticator` backends shared across the server roles.
- A `Daemon` composition root with loopback-bind defaults, the `WithBind` opt-in, graceful shutdown, and zap + OTel
  observability honouring the no-PHI rule.
- A DIMSE SCP role (C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE) with an optional Modality Worklist SCP, a DICOMweb role
  (STOW-RS, QIDO-RS, WADO-RS), an HL7 v2 MLLP role, and a FHIR REST role serving the conformance-subset interactions.
- Reference daemons wired to a filesystem object store and a SQLite catalogue, deployable for development and the CLI,
  verified against Orthanc and dcm4chee-arc (DIMSE/DICOMweb) and the HL7 FHIR validator (FHIR) in CI (PRD §11.1).

Explicit limits (deferred, architected-for — PRD §3.2, §5.1):

- **No SCP / server side of MPPS or Storage Commitment** — the v1 DIMSE-N services are SCU-only, so the daemon cannot
  answer N-CREATE/N-SET/N-ACTION for those flows.
- **Not a PACS/archive.** No multi-tenant access control, retention/erasure, replication, HA, or audit storage; those
  are the consumer's product built on these primitives.
- **FHIR REST is the conformance subset.** Only `read`, `create`, `search-type`, and `transaction` for the workflow
  resource set; `update`/`delete`/`vread`/`history`/`patch`, SMART on FHIR, and Subscriptions are deferred and answered
  with a typed `OperationOutcome`, never a silent no-op.
- **DICOMweb deferrals** (rendered/thumbnail, WADO-URI, UPS-RS) and **HL7 v2 deferrals** (inline charset escapes,
  message-profile validation) are inherited from the [DICOMweb](dicomweb.md) and [HL7 v2](hl7v2.md) packages.

## See also

- [DIMSE networking](dimse.md) — the `dimse.AE`, `dimse.Server`, `dimse.Handler`, `dimse.OpInfo`, and `dimse.Status`
  types the DIMSE role composes.
- [DICOMweb client and server](dicomweb.md) — the `dicomweb.StoreBackend`/`RetrieveBackend`/`QueryBackend` that the
  DICOMweb role composes.
- [FHIR R4/R5](fhir.md) — the release-agnostic `fhir.Resource`, `fhir.OperationOutcome`, and `fhir.Validate`, and the
  release-specific `r4.Bundle`/`r5.Bundle` resources the FHIR role composes.
- [HL7 v2 messaging and MLLP](hl7v2.md) — the `hl7v2.Handler` and MLLP server the MLLP role composes.
- `docs/prd/go-radx-prd.md` §7.2 (servers), §9.1 (PHI and bind defaults), §9.4 (concurrency), §9.7 (transport),
  §9.10 (observability).
- `UBIQUITOUS_LANGUAGE.md` — canonical Go names and the cross-standard collision table.
