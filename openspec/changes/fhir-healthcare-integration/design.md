# Design Document: FHIR Healthcare Integration Enhancement

## Context

go-radx provides comprehensive FHIR R4/R5 resources (304 total) and complete DICOM core with DIMSE networking, but
lacks the interoperability layer needed for production healthcare systems. This design addresses four critical gaps:

1. **DICOM ↔ FHIR conversion** - Mapping radiology imaging data to clinical systems
2. **SMART authorization** - OAuth2-based security for healthcare applications
3. **US Core compliance** - Implementation Guide support for US healthcare market
4. **Event notifications** - Real-time subscriptions for clinical workflows

These features are foundational for:
- Harrison Open Platform integration (AI workflow orchestration)
- Annalise Container deployment (secure multi-tenant access)
- EHR system integration (US Core compliance)
- Clinical decision support (real-time notifications)

## Goals

### Primary Goals
1. **Production-ready interoperability** - Enable real-world healthcare system integration
2. **Standards compliance** - Follow SMART, US Core, FHIR R5 Subscriptions specifications exactly
3. **Type safety** - Maintain Go's compile-time safety advantages
4. **Zero breaking changes** - All enhancements are additive
5. **Minimal dependencies** - Standard library first, minimal external deps
6. **Performance** - <100ms for typical operations
7. **Developer experience** - Clear APIs, comprehensive examples, excellent documentation

### Non-Goals
1. **Not a full FHIR server** - Library for building servers, not a server itself
2. **Not replacing existing FHIR/DICOM** - Building on top of existing foundations
3. **Not supporting FHIR R4B** - Focus on R4/R5, R4B handled separately if needed
4. **Not implementing all IGs** - US Core first, framework for others
5. **Not a generic OAuth2 library** - SMART-specific implementation

## Architecture Overview

### Package Structure

```
fhir/
├── r4/resources/         # Existing R4 resources (146 files)
├── r5/resources/         # Existing R5 resources (158 files)
├── primitives/           # Existing FHIR primitives
├── validation/           # Existing validation framework
├── bundle.go             # Existing bundle utilities
│
├── mapping/              # NEW: DICOM ↔ FHIR conversion
│   ├── imagingstudy.go   # DICOM Study → ImagingStudy
│   ├── patient.go        # DICOM Patient → Patient
│   ├── diagnosticreport.go # DICOM SR ↔ DiagnosticReport
│   ├── options.go        # Mapping configuration
│   └── errors.go         # Mapping-specific errors
│
├── smart/                # NEW: SMART on FHIR
│   ├── client.go         # Main SMART client
│   ├── config.go         # Client configuration
│   ├── auth/
│   │   ├── oauth2.go     # OAuth2 authorization code flow
│   │   ├── backend.go    # Backend Services (JWT)
│   │   ├── token_manager.go # Token refresh, caching
│   │   └── pkce.go       # PKCE implementation
│   ├── launch/
│   │   ├── ehr.go        # EHR launch flow
│   │   ├── standalone.go # Standalone launch flow
│   │   └── context.go    # Launch context resolution
│   ├── scopes.go         # Scope parsing and validation
│   └── conformance.go    # SMART capability discovery
│
├── ig/                   # NEW: Implementation Guides
│   ├── validator.go      # Generic IG validation framework
│   └── uscore/
│       ├── profiles.go   # US Core profile definitions
│       ├── validator.go  # US Core validation logic
│       ├── extensions.go # US Core extensions (race, ethnicity, etc.)
│       └── valuesets.go  # US Core value set bindings
│
└── subscriptions/        # NEW: FHIR Subscriptions
    ├── manager.go        # Subscription lifecycle management
    ├── subscription.go   # Subscription resource handling
    ├── topic.go          # SubscriptionTopic handling
    ├── notifier.go       # Notification dispatcher
    ├── webhook.go        # Webhook delivery
    ├── filters.go        # Subscription filters
    ├── store.go          # Subscription persistence interface
    └── memory_store.go   # In-memory store implementation
```

## Decisions

### Decision 1: DICOM Mapping - Dataset-Based vs. Object-Based API

**Context**: DICOM files can be parsed into datasets. Should mapping functions accept datasets or domain objects?

**Decision**: Accept `*dicom.Dataset` directly for maximum flexibility.

**Rationale**:
- ✅ Works with both file-based and network-based DICOM (DIMSE)
- ✅ Users can pre-process/filter datasets before mapping
- ✅ Avoids creating intermediate domain models
- ✅ Consistent with existing DICOM package API

**Implementation**:
```go
// Preferred approach
func DICOMStudyToImagingStudy(dataset *dicom.Dataset, opts MappingOptions) (*resources.ImagingStudy, error) {
    studyUID, err := dataset.GetString(dicom.TagStudyInstanceUID)
    if err != nil {
        return nil, fmt.Errorf("missing study UID: %w", err)
    }

    // Extract metadata from dataset tags
    study := &resources.ImagingStudy{
        Identifier: []resources.Identifier{{
            System: stringPtr("urn:dicom:uid"),
            Value:  stringPtr("urn:oid:" + studyUID),
        }},
        // ... map other fields
    }
    return study, nil
}
```

**Alternatives Considered**:

**Alternative A: Domain model wrapper**
```go
type DICOMStudy struct {
    UID         string
    PatientName string
    Series      []DICOMSeries
}

func DICOMStudyToImagingStudy(study *DICOMStudy) (*resources.ImagingStudy, error)
```
- ❌ Requires parsing entire DICOM structure upfront
- ❌ Duplicates dataset functionality
- ❌ Less flexible for partial mappings

**Alternative B: Fluent builder API**
```go
builder := mapping.NewImagingStudyBuilder(dataset)
study := builder.
    WithEndpoint("https://pacs.example.com/wado").
    WithModalityFilter("CT", "MR").
    Build()
```
- ❌ More complex for simple cases
- ❌ Not idiomatic Go (too fluent)
- ✅ Could add later as convenience layer

---

### Decision 2: SMART Client - Configuration vs. Discovery

**Context**: SMART servers expose authorization endpoints via FHIR CapabilityStatement. Should clients require manual
configuration or auto-discover?

**Decision**: Support both explicit configuration and automatic discovery.

**Rationale**:
- ✅ Auto-discovery aligns with SMART specification
- ✅ Explicit config needed for testing/development
- ✅ Provides flexibility for different deployment scenarios

**Implementation**:
```go
// Auto-discovery (production)
client, err := smart.NewClient(smart.Config{
    FHIRBaseURL: "https://fhir.example.com",
    ClientID:    "my-app",
    ClientSecret: os.Getenv("CLIENT_SECRET"),
    RedirectURI: "http://localhost:8080/callback",
})
// Client fetches CapabilityStatement and discovers auth endpoints

// Explicit config (testing/development)
client, err := smart.NewClient(smart.Config{
    FHIRBaseURL:      "https://fhir.example.com",
    AuthorizeURL:     "https://auth.example.com/authorize",
    TokenURL:         "https://auth.example.com/token",
    ClientID:         "my-app",
    ClientSecret:     os.Getenv("CLIENT_SECRET"),
    RedirectURI:      "http://localhost:8080/callback",
    SkipDiscovery:    true,
})
```

**Alternatives Considered**:

**Alternative A: Discovery only**
- ❌ Harder to test locally
- ❌ Requires valid FHIR server for development
- ❌ Less control for advanced scenarios

**Alternative B: Manual configuration only**
- ❌ Violates SMART specification (discovery is required feature)
- ❌ More error-prone (typos in URLs)
- ❌ Doesn't handle endpoint changes gracefully

---

### Decision 3: US Core Validation - Code Generation vs. Runtime Validation

**Context**: US Core profiles define constraints on FHIR resources. Should we generate Go types for each profile or
validate at runtime?

**Decision**: Runtime validation with profile-aware validator, NOT code generation.

**Rationale**:
- ✅ Avoids code explosion (6.1.0 has 50+ profiles)
- ✅ Profiles can be loaded dynamically
- ✅ Users can create custom IGs without recompilation
- ✅ Consistent with existing validation framework
- ✅ Easier to update when US Core versions change

**Implementation**:
```go
// Load US Core profiles
validator := uscore.NewValidator()

// Validate patient against US Core Patient profile
patient := &resources.Patient{ /* ... */ }
if err := validator.ValidateProfile(patient, uscore.ProfilePatient); err != nil {
    // Handle validation errors
    for _, issue := range err.(*validation.Errors).List() {
        log.Printf("Validation issue: %s - %s", issue.Field, issue.Message)
    }
}

// Check Must Support elements
missing := validator.CheckMustSupport(patient, uscore.ProfilePatient)
if len(missing) > 0 {
    log.Printf("Missing Must Support elements: %v", missing)
}
```

**Profile Definition Format**:
```go
var ProfilePatient = &Profile{
    URL: "http://hl7.org/fhir/us/core/StructureDefinition/us-core-patient",
    BaseProfile: "Patient",
    MustSupport: []string{
        "identifier",
        "identifier.system",
        "identifier.value",
        "name",
        "name.family",
        "name.given",
        "gender",
    },
    Extensions: []ExtensionDefinition{
        {
            URL: "http://hl7.org/fhir/us/core/StructureDefinition/us-core-race",
            Cardinality: "0..1",
        },
        {
            URL: "http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity",
            Cardinality: "0..1",
        },
    },
    ValueSetBindings: map[string]ValueSetBinding{
        "gender": {
            Strength: "required",
            ValueSet: "http://hl7.org/fhir/ValueSet/administrative-gender",
        },
    },
}
```

**Alternatives Considered**:

**Alternative A: Generate typed structs per profile**
```go
type USCorePatient struct {
    resources.Patient
    Race      *RaceExtension     // Embedded
    Ethnicity *EthnicityExtension // Embedded
}
```
- ❌ 50+ new types to maintain
- ❌ Breaking changes on US Core updates
- ❌ Can't support custom IGs without recompilation
- ❌ Inheritance doesn't match FHIR profiling model

**Alternative B: StructureDefinition JSON loading**
- Load US Core StructureDefinition JSON files directly
- Parse at runtime to build validation rules
- ✅ Most flexible, no hardcoding
- ❌ Performance overhead (JSON parsing)
- ❌ Complex StructureDefinition parsing logic
- **Decision**: Consider for v2 if dynamic IG loading is needed

---

### Decision 4: Subscriptions - Push vs. Pull Model

**Context**: FHIR Subscriptions can use push (webhooks) or pull (REST endpoint) delivery.

**Decision**: Implement push (webhook) delivery first, pull as future enhancement.

**Rationale**:
- ✅ Webhooks are most common deployment model
- ✅ Lower latency for time-sensitive notifications
- ✅ Simpler client implementation (no polling loop)
- ✅ Aligns with event-driven architecture patterns
- ⚠️ Pull model still useful for firewall-restricted clients (defer to v2)

**Implementation**:
```go
type Manager struct {
    store    SubscriptionStore
    notifier Notifier
    client   *http.Client
}

// Webhook delivery with retry
func (m *Manager) DeliverWebhook(ctx context.Context, sub *Subscription, bundle *resources.Bundle) error {
    data, err := json.Marshal(bundle)
    if err != nil {
        return err
    }

    // Retry with exponential backoff
    return retry.Do(func() error {
        req, _ := http.NewRequestWithContext(ctx, "POST", sub.Endpoint, bytes.NewReader(data))
        req.Header.Set("Content-Type", "application/fhir+json")

        resp, err := m.client.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode >= 400 {
            return fmt.Errorf("webhook failed: %d", resp.StatusCode)
        }
        return nil
    },
        retry.Attempts(3),
        retry.Delay(1*time.Second),
        retry.DelayType(retry.BackOffDelay),
    )
}
```

**Alternatives Considered**:

**Alternative A: Pull-based (REST endpoint)**
```go
GET /Subscription/$events?id=sub-123
```
- ✅ Works behind firewalls
- ❌ Requires clients to poll regularly
- ❌ Higher latency
- ❌ More complex client implementation
- **Decision**: Add in future if requested

**Alternative B: WebSocket delivery**
- ✅ Real-time bidirectional
- ❌ More complex server/client
- ❌ Not in FHIR spec
- ❌ Connection management overhead
- **Decision**: Out of scope

---

### Decision 5: Error Handling - Typed Errors vs. Error Wrapping

**Context**: Healthcare systems need detailed error context for debugging and audit trails.

**Decision**: Use error wrapping with `fmt.Errorf("%w")` and sentinel errors for common cases.

**Rationale**:
- ✅ Standard Go error handling patterns
- ✅ Preserves error chains for debugging
- ✅ Compatible with `errors.Is()` and `errors.As()`
- ✅ No custom error types needed for most cases

**Implementation**:
```go
// Sentinel errors for common cases
var (
    ErrMissingStudyUID    = errors.New("DICOM study missing StudyInstanceUID")
    ErrInvalidModality    = errors.New("invalid DICOM modality")
    ErrUnauthorized       = errors.New("SMART authorization failed")
    ErrInvalidScope       = errors.New("invalid SMART scope")
    ErrSubscriptionFailed = errors.New("subscription webhook delivery failed")
)

// Wrap errors with context
func DICOMStudyToImagingStudy(dataset *dicom.Dataset, opts MappingOptions) (*resources.ImagingStudy, error) {
    studyUID, err := dataset.GetString(dicom.TagStudyInstanceUID)
    if err != nil {
        return nil, fmt.Errorf("extracting study UID: %w", err)
    }

    if studyUID == "" {
        return nil, fmt.Errorf("%w: empty StudyInstanceUID", ErrMissingStudyUID)
    }

    // ...
}

// Usage
study, err := mapping.DICOMStudyToImagingStudy(dataset, opts)
if err != nil {
    if errors.Is(err, mapping.ErrMissingStudyUID) {
        // Handle missing UID specifically
    }
    log.Printf("Mapping failed: %v", err) // Full error chain
}
```

**Alternatives Considered**:

**Alternative A: Custom error types**
```go
type MappingError struct {
    Tag     dicom.Tag
    Message string
    Cause   error
}
```
- ❌ Verbose for callers (type assertions)
- ❌ Doesn't add value over wrapping
- ✅ Could add later if structured error data needed

**Alternative B: Error codes (HL7 OperationOutcome style)**
- ❌ Overly complex for library
- ❌ Better suited for FHIR server implementations
- ❌ Doesn't leverage Go's error handling

---

### Decision 6: Dependency Management - Minimal vs. Feature-Rich

**Context**: Adding OAuth2, JWT, HTTP client features. Use standard library or external packages?

**Decision**: Minimal external dependencies - only `golang.org/x/oauth2` and `github.com/google/uuid`.

**Rationale**:
- ✅ `golang.org/x/oauth2` is semi-official, widely used, well-maintained
- ✅ `github.com/google/uuid` is standard for UUID generation
- ✅ Minimizes supply chain risk
- ✅ Faster builds, smaller binaries
- ✅ Easier security audits

**Dependencies**:
```go
// go.mod additions
require (
    golang.org/x/oauth2 v0.15.0        // OAuth2 client (Apache-2.0)
    github.com/google/uuid v1.5.0      // UUID generation (BSD-3-Clause)
)

// Optional (only if implementing JWT backend services)
require (
    github.com/golang-jwt/jwt/v5 v5.2.0 // JWT parsing (MIT)
)
```

**Standard Library Usage**:
- `net/http` - HTTP client for webhooks
- `encoding/json` - JSON marshaling
- `crypto/rand` - PKCE code verifier
- `crypto/sha256` - PKCE code challenge
- `time` - Token expiry, retry delays
- `context` - Request cancellation

**Alternatives Considered**:

**Alternative A: Zero external dependencies**
- Implement OAuth2 client from scratch
- ❌ High effort, error-prone
- ❌ Reinventing the wheel
- ❌ Security risk (OAuth2 is complex)

**Alternative B: Rich HTTP client (resty, req)**
- ❌ Unnecessary complexity
- ❌ Larger dependency tree
- ✅ `net/http` sufficient for our needs

**Alternative C: Full FHIR client library**
- Use existing Go FHIR client for SMART
- ❌ Circular dependency (we ARE the FHIR library)
- ❌ Lock-in to specific client design

## Component Interactions

### DICOM to FHIR Mapping Flow

```
┌─────────────────┐
│  DICOM File     │
│  or DIMSE       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ dicom.Dataset   │──────┐
│ (existing)      │      │
└─────────────────┘      │
                         │
         ┌───────────────┘
         │
         ▼
┌──────────────────────────────────┐
│ mapping.DICOMStudyToImagingStudy │
│                                  │
│ 1. Extract Study UID             │
│ 2. Map Patient demographics      │
│ 3. Map Series metadata           │
│ 4. Map Instance references       │
│ 5. Add WADO-RS endpoints         │
└─────────────┬────────────────────┘
              │
              ▼
┌─────────────────────────┐
│ resources.ImagingStudy  │
│ (FHIR R5)               │
└─────────────────────────┘
```

### SMART Authorization Flow

```
┌──────────────┐
│ FHIR Client  │
│ Application  │
└──────┬───────┘
       │
       │ 1. Launch
       ▼
┌──────────────────────┐
│ smart.Client         │
│  .EHRLaunch()        │
└──────┬───────────────┘
       │
       │ 2. Discover endpoints
       ▼
┌──────────────────────┐
│ FHIR Server          │
│ CapabilityStatement  │
└──────┬───────────────┘
       │
       │ 3. Authorization URL
       ▼
┌──────────────────────┐
│ Authorization Server │
│ /authorize           │
└──────┬───────────────┘
       │
       │ 4. User login & consent
       │ 5. Authorization code
       ▼
┌──────────────────────┐
│ smart.Client         │
│  .ExchangeCode()     │
└──────┬───────────────┘
       │
       │ 6. Token exchange
       ▼
┌──────────────────────┐
│ Authorization Server │
│ /token               │
└──────┬───────────────┘
       │
       │ 7. Access token + refresh token
       ▼
┌──────────────────────┐
│ Token Manager        │
│ - Cache token        │
│ - Auto refresh       │
└──────────────────────┘
```

### US Core Validation Flow

```
┌─────────────────────┐
│ resources.Patient   │
└─────────┬───────────┘
          │
          │ 1. Validate
          ▼
┌────────────────────────────┐
│ uscore.Validator           │
│  .ValidateProfile()        │
└────────┬───────────────────┘
         │
         │ 2. Load profile definition
         ▼
┌────────────────────────────┐
│ uscore.ProfilePatient      │
│ - Must Support elements    │
│ - Extensions               │
│ - Value set bindings       │
└────────┬───────────────────┘
         │
         │ 3. Run checks
         ├──────────────────────┐
         │                      │
         ▼                      ▼
┌─────────────────┐   ┌─────────────────┐
│ Base FHIR       │   │ US Core         │
│ Validation      │   │ Constraints     │
│ (existing)      │   │ - Must Support  │
│                 │   │ - Extensions    │
└────────┬────────┘   │ - Value sets    │
         │            └────────┬────────┘
         │                     │
         └─────────┬───────────┘
                   │
                   ▼
          ┌────────────────┐
          │ Validation     │
          │ Results        │
          └────────────────┘
```

### Subscription Notification Flow

```
┌─────────────────────┐
│ Resource Change     │
│ (Patient updated)   │
└─────────┬───────────┘
          │
          │ 1. Trigger event
          ▼
┌──────────────────────────┐
│ subscriptions.Manager    │
│  .NotifyChange()         │
└────────┬─────────────────┘
         │
         │ 2. Find matching subscriptions
         ▼
┌──────────────────────────┐
│ Subscription Store       │
│ - Query by topic         │
│ - Apply filters          │
└────────┬─────────────────┘
         │
         │ 3. Matching subscriptions
         ▼
┌──────────────────────────┐
│ subscriptions.Notifier   │
│ - Build notification     │
│ - Create history bundle  │
└────────┬─────────────────┘
         │
         │ 4. Notification bundle
         ▼
┌──────────────────────────┐
│ subscriptions.Webhook    │
│  .DeliverWebhook()       │
│ - Retry logic            │
│ - Exponential backoff    │
└────────┬─────────────────┘
         │
         │ 5. HTTP POST
         ▼
┌──────────────────────────┐
│ Subscriber Endpoint      │
│ https://app.example.com  │
└──────────────────────────┘
```

## Performance Considerations

### DICOM Mapping

**Target**: <50ms for typical study with 100 instances

**Optimization Strategies**:
1. **Lazy parsing** - Only extract tags needed for mapping
2. **Batch processing** - Map multiple series in parallel
3. **Caching** - Cache frequently used DICOM tags
4. **Memory efficiency** - Stream large pixel data, don't load into memory

**Benchmark Goals**:
```
BenchmarkDICOMToImagingStudy-8    10000    120000 ns/op    45000 B/op    150 allocs/op
```

### SMART Authorization

**Target**: <200ms for token exchange, <5ms for cached token access

**Optimization Strategies**:
1. **Token caching** - Cache access tokens in memory, avoid repeated exchanges
2. **Connection pooling** - Reuse HTTP connections to auth server
3. **Lazy discovery** - Cache CapabilityStatement, refresh only when needed
4. **Concurrent refresh** - Refresh tokens in background before expiry

**Token Cache Design**:
```go
type TokenCache struct {
    mu     sync.RWMutex
    tokens map[string]*CachedToken
}

type CachedToken struct {
    Token      *oauth2.Token
    ExpiresAt  time.Time
    RefreshAt  time.Time  // Proactive refresh before expiry
}
```

### US Core Validation

**Target**: <10ms for typical patient resource

**Optimization Strategies**:
1. **Profile caching** - Load profiles once at startup
2. **Reflection caching** - Cache struct field lookups
3. **Parallel validation** - Run independent checks concurrently
4. **Early exit** - Stop on first critical error (optional)

### Subscription Delivery

**Target**: <100ms for webhook delivery, 99.9% success rate

**Optimization Strategies**:
1. **Async delivery** - Don't block on webhook calls
2. **Connection pooling** - Reuse HTTP connections per endpoint
3. **Batch notifications** - Group rapid changes (optional)
4. **Circuit breaker** - Disable failing endpoints temporarily

**Retry Strategy**:
- Initial delay: 1s
- Max retries: 3
- Backoff: Exponential (1s, 2s, 4s)
- Dead letter queue: After max retries, log failure for manual intervention

## Testing Strategy

### Unit Tests

**Coverage Target**: >85% for all new packages

**Test Categories**:
1. **Happy path** - Typical use cases with valid inputs
2. **Error cases** - Invalid inputs, missing data, malformed resources
3. **Edge cases** - Empty fields, maximum cardinality, choice types
4. **Concurrency** - Race condition testing with `-race` flag

### Integration Tests

**DICOM Mapping**:
- Real-world DICOM files from public datasets
- Orthanc integration tests (existing infrastructure)
- Roundtrip validation (DICOM → FHIR → DICOM metadata preserved)

**SMART Authorization**:
- Mock auth server for deterministic testing
- [SMART Launcher](https://launch.smarthealthit.org/) conformance tests
- Token refresh simulation (expired tokens)

**US Core Validation**:
- All US Core examples from HL7 specification
- Must Support element coverage tests
- Extension validation tests

**Subscriptions**:
- Mock HTTP server for webhook delivery
- Retry logic verification
- Concurrent notification handling
- Filter matching accuracy

### Performance Tests

**Benchmarks**:
```bash
go test -bench=. -benchmem ./fhir/mapping
go test -bench=. -benchmem ./fhir/smart
go test -bench=. -benchmem ./fhir/ig/uscore
go test -bench=. -benchmem ./fhir/subscriptions
```

**Load Tests**:
- 1000 concurrent DICOM mappings
- 100 concurrent token refreshes
- 10,000 subscription notifications/second

### Security Tests

**SMART Authorization**:
- PKCE code verifier randomness
- Token storage security (no plaintext logging)
- Scope validation enforcement

**Subscription Webhooks**:
- SSRF prevention (URL validation)
- Request timeout enforcement
- Rate limiting support

## Documentation Plan

### User Guide

**fhir/mapping/**:
- Getting started with DICOM → FHIR conversion
- Mapping options and customization
- Handling missing DICOM tags gracefully
- Performance tuning for large datasets

**fhir/smart/**:
- SMART App Launch quickstart
- EHR launch vs. standalone launch
- Backend Services for system-to-system auth
- Token management best practices

**fhir/ig/uscore/**:
- US Core validation quickstart
- Must Support element checking
- US Core extensions (race, ethnicity, birthsex)
- Custom IG development guide

**fhir/subscriptions/**:
- Creating and managing subscriptions
- Webhook endpoint implementation
- Subscription filters and topics
- Monitoring and troubleshooting

### API Documentation

**godoc coverage**: 100% for exported types and functions

**Required documentation**:
- Package-level overview with examples
- Function/method documentation with parameters and return values
- Code examples for common use cases
- Links to relevant standards (SMART, US Core, FHIR specs)

### Examples

**Runnable examples**:
```
fhir/mapping/examples/
  - dicom_to_imagingstudy/
  - patient_mapping/
  - sr_to_diagnosticreport/

fhir/smart/examples/
  - ehr_launch/
  - standalone_launch/
  - backend_services/
  - token_refresh/

fhir/ig/uscore/examples/
  - validate_patient/
  - check_mustsupport/
  - custom_ig/

fhir/subscriptions/examples/
  - webhook_server/
  - subscription_manager/
  - custom_filters/
```

## Risks & Mitigation

### Risk 1: SMART Implementation Complexity

**Risk**: OAuth2 flows are complex, easy to implement incorrectly, security vulnerabilities.

**Mitigation**:
1. Use `golang.org/x/oauth2` for core OAuth2 logic (battle-tested)
2. Follow SMART App Launch specification exactly
3. Security review by external experts
4. Conformance testing with SMART Launcher
5. Comprehensive security documentation

**Impact**: High (security critical)
**Probability**: Medium
**Mitigation Effort**: High

---

### Risk 2: US Core Profile Evolution

**Risk**: US Core releases new versions, breaking changes, deprecated profiles.

**Mitigation**:
1. Version-specific profile packages (`uscore/v6`, `uscore/v7`)
2. Clear documentation of supported US Core version
3. Migration guides for version updates
4. Generic IG validation framework supports custom profiles

**Impact**: Medium (affects US market users)
**Probability**: High (US Core updates regularly)
**Mitigation Effort**: Low (design anticipates this)

---

### Risk 3: Subscription Delivery Reliability

**Risk**: Webhooks fail due to network issues, subscriber downtime, rate limiting.

**Mitigation**:
1. Retry logic with exponential backoff
2. Dead letter queue for persistent failures
3. Circuit breaker pattern for failing endpoints
4. Monitoring hooks for delivery status
5. Pluggable persistence for durable queuing

**Impact**: Medium (affects real-time notifications)
**Probability**: High (network is unreliable)
**Mitigation Effort**: Medium (built into design)

---

### Risk 4: DICOM Tag Variability

**Risk**: Real-world DICOM files have missing/invalid tags, non-standard values.

**Mitigation**:
1. Graceful degradation (optional fields remain nil)
2. Validation at mapping input (fail fast)
3. Comprehensive error messages with tag context
4. Mapping options for lenient vs. strict mode
5. Extensive testing with real-world files

**Impact**: High (affects mapping reliability)
**Probability**: High (DICOM in the wild is messy)
**Mitigation Effort**: Medium (testing and error handling)

---

### Risk 5: Dependency Security

**Risk**: External dependencies (`oauth2`, `uuid`, `jwt`) have vulnerabilities.

**Mitigation**:
1. Pin dependency versions in `go.mod`
2. Regular `govulncheck` in CI
3. Dependabot for security updates
4. Minimal dependency surface area
5. Vendor dependencies if needed

**Impact**: High (security critical)
**Probability**: Low (mature, well-audited libraries)
**Mitigation Effort**: Low (automated tooling)

## Open Questions

### Q1: Should DICOM SR mapping support all TID templates?

**Context**: DICOM Structured Reports have 50+ Template IDs (TID), each with specific structure.

**Options**:
A. Support TID 1500 (Measurement Report) and TID 1501 (CAD) only (most common)
B. Support all TIDs with best-effort mapping
C. Extensible framework, users implement custom TID mappers

**Recommendation**: Option A for Phase 1, Option C framework for extensibility.

---

### Q2: Should SMART Backend Services use asymmetric (RS384) or symmetric (HS256) JWT?

**Context**: SMART Backend Services spec allows both, but asymmetric is recommended.

**Options**:
A. RS384 only (asymmetric, more secure, key management complexity)
B. Both RS384 and HS256 (flexibility, more code)
C. HS256 only (simpler, less secure, not recommended by spec)

**Recommendation**: Option A (RS384 only) - follow spec recommendation.

---

### Q3: Should US Core validator support multiple versions simultaneously?

**Context**: Healthcare systems may need to validate against multiple US Core versions during transitions.

**Options**:
A. Single version at a time (simpler, version-specific packages)
B. Multi-version support (complex, valuable for transitions)
C. Version detection from Meta.profile (automatic, complex)

**Recommendation**: Option A initially, consider Option B if requested.

---

### Q4: Should subscription webhook delivery be synchronous or asynchronous?

**Context**: Async is more scalable but requires background worker infrastructure.

**Options**:
A. Synchronous (blocking, simpler, lower throughput)
B. Asynchronous with goroutines (non-blocking, higher throughput, no persistence)
C. Asynchronous with persistent queue (most robust, requires external queue)

**Recommendation**: Option B (async with goroutines) for library, users can add queue if needed.

---

### Q5: Should we provide CLI tools for these features?

**Context**: CLI can help with testing and debugging, but increases scope.

**Options**:
A. Library only (focus on core functionality)
B. Optional CLI tools (convenience, examples)
C. Full-featured CLI (high effort, maintenance burden)

**Recommendation**: Option A for initial release, Option B as future enhancement.

## Success Metrics

### Functional Metrics

1. ✅ **DICOM Mapping**: Convert 100+ real DICOM studies without errors
2. ✅ **SMART Conformance**: Pass all SMART App Launch conformance tests
3. ✅ **US Core Validation**: Validate 95%+ of US Core examples from HL7
4. ✅ **Subscription Delivery**: 99.9% webhook success rate in load tests

### Performance Metrics

1. ✅ **DICOM Mapping**: <50ms per study (100 instances)
2. ✅ **SMART Token**: <200ms for token exchange, <5ms cached access
3. ✅ **US Core Validation**: <10ms per patient resource
4. ✅ **Subscription Delivery**: <100ms per webhook (excluding network)

### Code Quality Metrics

1. ✅ **Test Coverage**: >85% for all new packages
2. ✅ **No Breaking Changes**: Existing tests pass unchanged
3. ✅ **Linting**: 100% golangci-lint pass
4. ✅ **Documentation**: 100% godoc coverage for exports

### Adoption Metrics (Post-Release)

1. 📊 **GitHub Stars**: +50 within 3 months
2. 📊 **Downloads**: 1000+ go get downloads/month
3. 📊 **Issues**: <5 open bugs, <2 week response time
4. 📊 **Community**: 3+ external contributors

## Future Enhancements

### After Phase 4

**Bulk FHIR** (requires SMART Backend Services from Phase 2):
- System-level export: `GET /fhir/$export`
- NDJSON streaming for large datasets
- Async operation polling

**Additional IGs**:
- IPA (International Patient Access)
- mCODE (oncology)
- C-CDA on FHIR

**Advanced DICOM Mapping**:
- DICOM SEG → FHIR ImagingSelection
- DICOM GSPS → FHIR annotations
- DICOM RT (radiation therapy)

**Subscription Enhancements**:
- Pull-based delivery (REST endpoint)
- WebSocket support
- Batch notifications

**CLI Tools**:
```bash
radx smart launch --ehr https://launch.smarthealthit.org
radx uscore validate --profile patient patient.json
radx subscribe create --topic patient-update --webhook https://app.example.com/notify
```

## References

**Standards**:
- [SMART App Launch 2.0](http://hl7.org/fhir/smart-app-launch/)
- [US Core Implementation Guide v6.1.0](http://hl7.org/fhir/us/core/)
- [FHIR R5 Subscriptions](http://hl7.org/fhir/R5/subscriptions.html)
- [DICOM PS3.16 Content Mapping](https://dicom.nema.org/medical/dicom/current/output/chtml/part16/chapter_A.html)

**Reference Implementations**:
- [HAPI FHIR](https://github.com/hapifhir/hapi-fhir) - Java FHIR server
- [pydicom](https://github.com/pydicom/pydicom) - Python DICOM library
- [SMART JavaScript Library](https://github.com/smart-on-fhir/client-js)

**Testing Resources**:
- [SMART Launcher](https://launch.smarthealthit.org/)
- [US Core Examples](https://hl7.org/fhir/us/core/examples.html)
- [DICOM Library](https://www.dicomlibrary.com/)
