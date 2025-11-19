# FHIR Healthcare Integration Enhancement

## Why

The current go-radx implementation provides complete FHIR R4/R5 resources (304 total) and comprehensive DICOM core
with DIMSE networking, but lacks **critical healthcare interoperability features** needed for production healthcare
systems and clinical workflows:

1. **No DICOM ↔ FHIR conversion** - Despite having both complete DICOM and FHIR implementations, there's no mapping
   layer to convert between them. This prevents radiology workflow integration where DICOM studies need to be
   represented as FHIR ImagingStudy resources for clinical systems.

2. **No authorization framework** - Missing SMART on FHIR means no OAuth2-based security, no third-party app
   integration, and no fine-grained access control. This blocks enterprise adoption and regulatory compliance
   requirements.

3. **No Implementation Guide support** - Without US Core IG, the library can't be used in US healthcare systems that
   require compliance with US Core profiles. This limits market applicability and FDA/regulatory pathways.

4. **No real-time notifications** - Absence of FHIR Subscriptions means systems must poll for changes, creating
   inefficient architectures and preventing event-driven clinical workflows (e.g., "notify when urgent imaging study
   arrives").

These gaps prevent go-radx from being production-ready for:
- **Harrison Open Platform integration** - Needs DICOM→FHIR conversion for orchestrating AI workflows
- **Annalise Container deployment** - Requires SMART authorization for secure multi-tenant access
- **US healthcare market** - Mandates US Core compliance for EHR integration
- **Clinical decision support** - Needs real-time subscriptions for time-sensitive notifications

## What Changes

This proposal adds four foundational healthcare integration capabilities:

### 1. **DICOM to FHIR ImagingStudy Mapping** (`dicom-fhir-mapping` spec)

Bidirectional conversion layer between DICOM studies and FHIR resources:

```go
// New package: fhir/mapping
func DICOMStudyToImagingStudy(dataset *dicom.Dataset, opts MappingOptions) (*resources.ImagingStudy, error)
func DICOMPatientToFHIRPatient(dataset *dicom.Dataset) (*resources.Patient, error)
func DiagnosticReportToDICOMSR(report *resources.DiagnosticReport, template SRTemplateID) (*dicom.Dataset, error)
```

**Key Features:**
- DICOM Study/Series/Instance → FHIR ImagingStudy with full metadata preservation
- DICOM Patient demographics → FHIR Patient resource
- DICOM SR (Structured Reports) ↔ FHIR DiagnosticReport + Observation
- Endpoint configuration for WADO-RS references
- Provenance tracking and audit trails

**Value:** Enables radiology PACS integration, imaging AI workflows, and clinical system interoperability.

---

### 2. **SMART on FHIR Authorization** (`smart-on-fhir` spec)

OAuth2-based authorization framework for FHIR applications:

```go
// New package: fhir/smart
type Client struct {
    Config       *ClientConfig
    TokenManager *TokenManager
}

func (c *Client) EHRLaunch(ctx context.Context, launchToken string) (*LaunchContext, error)
func (c *Client) StandaloneLaunch(ctx context.Context) (*LaunchContext, error)
func (c *Client) BackendServices(ctx context.Context, jwt string) (*Token, error)
```

**Key Features:**
- **EHR Launch** - Launch apps from within EHR systems with context (patient, encounter)
- **Standalone Launch** - Independent app authorization with user login
- **Backend Services** - System-to-system auth using JWT assertions (asymmetric keys)
- **Token Management** - Automatic refresh, caching, secure storage, PKCE support
- **Scope Validation** - Fine-grained access control (patient/*, user/*, system/*)
- **Conformance Discovery** - Automatic SMART capability detection

**Value:** Enables secure third-party apps, enterprise SSO integration, and regulatory compliance.

---

### 3. **US Core Implementation Guide** (`us-core-ig` spec)

US healthcare-specific FHIR profiles and validation:

```go
// New package: fhir/ig/uscore
func ValidateUSCorePatient(patient *resources.Patient) error
func ValidateUSCoreObservation(obs *resources.Observation) error
func EnforceUSCoreProfile(resource any, profileURL string) error
```

**Key Features:**
- **US Core Profiles** - Patient, Observation, DiagnosticReport, ImagingStudy, etc.
- **Must Support Validation** - Enforce required elements per US Core profiles
- **Value Set Bindings** - US-specific terminologies (administrative-gender, race, ethnicity)
- **Extensions** - US Core Race, Ethnicity, BirthSex extensions
- **Profile Validation Layer** - Extensible framework for other IGs (IPA, mCODE)

**Value:** Unlocks US healthcare market, enables EHR integration, supports FDA regulatory pathways.

---

### 4. **FHIR Subscriptions** (`fhir-subscriptions` spec)

Event-driven notifications for resource changes:

```go
// New package: fhir/subscriptions
type Manager struct {
    Store    SubscriptionStore
    Notifier Notifier
}

func (m *Manager) Subscribe(ctx context.Context, sub *Subscription) error
func (m *Manager) NotifyChange(ctx context.Context, event ResourceEvent) error
func (m *Manager) DeliverWebhook(ctx context.Context, url string, bundle *Bundle) error
```

**Key Features:**
- **R5 Topic-Based Subscriptions** - SubscriptionTopic resource support
- **R4 Backport Support** - Using FHIR Subscriptions Backport IG
- **Webhook Delivery** - HTTP POST notifications with retry logic
- **Filters** - FHIRPath expressions and query parameter filters
- **Notification Bundles** - `history` bundle type with before/after state
- **Delivery Tracking** - Status monitoring, failure handling, exponential backoff

**Value:** Enables real-time clinical decision support, reduces polling overhead, reactive architectures.

---

## Impact

### Affected Specs

Four new capability specs:

1. **ADDED**: `dicom-fhir-mapping` - DICOM to FHIR conversion layer
2. **ADDED**: `smart-on-fhir` - OAuth2 authorization framework
3. **ADDED**: `us-core-ig` - US Core Implementation Guide support
4. **ADDED**: `fhir-subscriptions` - Real-time event notifications

### Affected Code

**New Packages:**
```
fhir/
├── mapping/              # DICOM ↔ FHIR conversion (NEW)
│   ├── imagingstudy.go
│   ├── patient.go
│   ├── diagnosticreport.go
│   └── options.go
├── smart/                # SMART on FHIR (NEW)
│   ├── client.go
│   ├── auth/
│   │   ├── oauth2.go
│   │   ├── backend.go
│   │   ├── token_manager.go
│   │   └── pkce.go
│   ├── launch/
│   │   ├── ehr.go
│   │   ├── standalone.go
│   │   └── context.go
│   └── conformance.go
├── ig/                   # Implementation Guides (NEW)
│   ├── uscore/
│   │   ├── profiles.go
│   │   ├── validator.go
│   │   ├── extensions.go
│   │   └── valuesets.go
│   └── validator.go
└── subscriptions/        # FHIR Subscriptions (NEW)
    ├── manager.go
    ├── subscription.go
    ├── notifier.go
    ├── webhook.go
    ├── filters.go
    └── store.go
```

**Modified Files:**
- `fhir/validation/validator.go` - Add US Core profile validation hooks
- `fhir/bundle.go` - Add subscription notification bundle helpers
- `dicom/dataset.go` - Add FHIR mapping helper methods (optional)

**Test Files:**
```
fhir/mapping/mapping_test.go              (~500 lines)
fhir/mapping/imagingstudy_test.go         (~300 lines)
fhir/smart/auth/oauth2_test.go            (~400 lines)
fhir/smart/launch/ehr_test.go             (~250 lines)
fhir/ig/uscore/validator_test.go          (~350 lines)
fhir/subscriptions/manager_test.go        (~300 lines)
fhir/subscriptions/webhook_test.go        (~200 lines)
```

### New Features

1. **DICOM Conversion CLI** (optional):
   ```bash
   radx dicom to-fhir study.dcm --output imagingstudy.json
   radx fhir to-dicom diagnosticreport.json --template TID1500 --output sr.dcm
   ```

2. **SMART Authorization Helpers**:
   - Token refresh middleware
   - Scope validation decorators
   - Launch context helpers

3. **US Core Validation CLI** (optional):
   ```bash
   radx fhir validate --profile us-core-patient patient.json
   radx fhir check-mustsupport --profile us-core-observation obs.json
   ```

4. **Subscription Management**:
   - In-memory subscription store (default)
   - Pluggable persistence backends
   - Webhook delivery monitoring

### Breaking Changes

**NONE** - All changes are additive:
- New packages in `fhir/mapping`, `fhir/smart`, `fhir/ig`, `fhir/subscriptions`
- No modifications to existing FHIR resource types or validation behavior
- Opt-in features requiring explicit import and usage

### Dependencies

**New External Dependencies:**

1. **golang.org/x/oauth2** (Apache-2.0) - OAuth2 client for SMART authorization
   - Well-maintained, official Go library
   - Already widely used in Go ecosystem
   - Minimal transitive dependencies

2. **github.com/google/uuid** (BSD-3-Clause) - UUID generation for identifiers
   - Standard library quality, widely adopted
   - Zero external dependencies
   - Already likely in dependency tree

**Optional Dependencies (for enhanced features):**

3. **github.com/golang-jwt/jwt/v5** (MIT) - JWT parsing for Backend Services
   - Only if implementing SMART Backend Services
   - Can use standard library crypto/jwt as alternative

**All other functionality**: Pure Go with standard library only.

### Migration Path

**No migration required** - All changes are additive:

1. Existing code continues to work unchanged
2. New features require explicit imports:
   ```go
   import "github.com/codeninja55/go-radx/fhir/mapping"
   import "github.com/codeninja55/go-radx/fhir/smart"
   import "github.com/codeninja55/go-radx/fhir/ig/uscore"
   import "github.com/codeninja55/go-radx/fhir/subscriptions"
   ```
3. Incremental adoption supported (use only features you need)

### Benefits

1. **Production-Ready Healthcare Integration**
   - Complete radiology workflow support (DICOM → FHIR → clinical systems)
   - Secure authorization for enterprise deployments
   - US healthcare market compliance

2. **Developer Experience**
   - Type-safe conversion APIs with comprehensive error handling
   - Clear separation of concerns (mapping, auth, validation, events)
   - Extensive examples and documentation

3. **Performance**
   - Lazy DICOM parsing only when conversion needed
   - OAuth2 token caching reduces auth overhead
   - Subscription webhooks eliminate polling

4. **Standards Compliance**
   - SMART App Launch 2.0 specification
   - US Core Implementation Guide v6.1.0
   - FHIR R5 Subscriptions specification
   - DICOM PS3.16 Content Mapping Resources

5. **Extensibility**
   - Framework for additional IGs (IPA, mCODE, C-CDA on FHIR)
   - Pluggable subscription notification backends
   - Custom DICOM → FHIR mapping rules

### Risks & Mitigation

**Risk 1: OAuth2 Security**
- **Mitigation**: Follow SMART App Launch security best practices, mandate PKCE, comprehensive security testing

**Risk 2: US Core Profile Complexity**
- **Mitigation**: Start with essential profiles (Patient, Observation, DiagnosticReport), expand incrementally

**Risk 3: Subscription Delivery Reliability**
- **Mitigation**: Implement retry logic, exponential backoff, dead letter queues, monitoring hooks

**Risk 4: DICOM Mapping Edge Cases**
- **Mitigation**: Comprehensive test suite with real-world DICOM files, graceful handling of missing/invalid data

**Risk 5: Dependency Management**
- **Mitigation**: Minimal dependencies (only oauth2 and uuid), regular security audits, pin versions

## Success Metrics

1. ✅ **DICOM Conversion**: Successfully converts 100+ real-world DICOM studies to ImagingStudy
2. ✅ **SMART Authorization**: Passes official SMART conformance tests
3. ✅ **US Core Validation**: Validates 95%+ of US Core examples from HL7 spec
4. ✅ **Subscriptions**: Delivers 99.9% of webhook notifications within 5 seconds
5. ✅ **Performance**: All operations <100ms for typical resources
6. ✅ **Test Coverage**: >85% coverage for all new packages
7. ✅ **Documentation**: Complete user guides with working examples for each feature
8. ✅ **Zero Breaking Changes**: Existing users unaffected

## Implementation Sequence

Recommended order for maximum value with minimum risk:

### Phase 1: DICOM to FHIR Mapping (Weeks 1-3)
**Why First**: Highest value for radiology workflows, leverages existing DICOM implementation, no external
dependencies.

**Deliverables**:
- `fhir/mapping/imagingstudy.go` - DICOM Study → ImagingStudy
- `fhir/mapping/patient.go` - DICOM Patient → Patient
- Comprehensive test suite with real DICOM files
- Documentation and examples

**Validation**: Convert 100+ DICOM studies, verify FHIR JSON validity

---

### Phase 2: SMART on FHIR (Weeks 4-7)
**Why Second**: Unblocks enterprise security, enables remaining features (Bulk FHIR requires Backend Services).

**Deliverables**:
- `fhir/smart/auth/oauth2.go` - OAuth2 client
- `fhir/smart/launch/` - EHR + standalone launch
- `fhir/smart/auth/backend.go` - Backend Services (JWT)
- Token management with refresh
- SMART conformance tests

**Validation**: Pass official SMART App Launch conformance suite

---

### Phase 3: US Core IG (Weeks 8-10)
**Why Third**: Builds on SMART authorization, enables US market compliance.

**Deliverables**:
- `fhir/ig/uscore/profiles.go` - US Core profiles (Patient, Observation, DiagnosticReport)
- `fhir/ig/uscore/validator.go` - Profile validation
- `fhir/ig/uscore/extensions.go` - Race, ethnicity, birthsex
- Must Support element validation
- Value set bindings

**Validation**: Validate all US Core examples from HL7 spec

---

### Phase 4: FHIR Subscriptions (Weeks 11-13)
**Why Last**: Most complex, depends on solid foundation from earlier phases.

**Deliverables**:
- `fhir/subscriptions/manager.go` - Subscription lifecycle
- `fhir/subscriptions/webhook.go` - HTTP delivery with retry
- `fhir/subscriptions/filters.go` - Query parameter filters
- R5 topic-based subscriptions
- R4 backport support

**Validation**: Deliver 10,000 test webhooks with 99.9% success rate

---

**Total Timeline: ~13 weeks (~3 months)**

**Parallel Work Opportunities:**
- Documentation can be written alongside implementation
- Test suites can be developed in parallel with code
- Examples can be created as features stabilize

## Open Questions

1. **Q: Should DICOM SR → DiagnosticReport mapping be in Phase 1 or deferred?**
   - **Proposed**: Include basic SR support in Phase 1, advanced templates (TID 1500, 1501) in future iteration
   - **Rationale**: Basic SR covers 80% of use cases, keeps Phase 1 timeline reasonable

2. **Q: Which US Core profiles are minimum viable?**
   - **Proposed**: Patient, Observation, DiagnosticReport, ImagingStudy (radiology focus)
   - **Rationale**: Covers primary radiology workflows, other profiles added incrementally

3. **Q: Should we support FHIR R4 Subscriptions or only R5?**
   - **Proposed**: R5 first (cleaner API), R4 backport via optional adapter
   - **Rationale**: R5 is future-focused, but R4 still widely deployed - provide migration path

4. **Q: What subscription persistence backends should we support?**
   - **Proposed**: In-memory (default), pluggable interface for SQL/Redis/etc.
   - **Rationale**: Simple default for testing, production users can integrate their storage

5. **Q: Should SMART client credentials be in config files or environment variables?**
   - **Proposed**: Environment variables (12-factor), config file support optional
   - **Rationale**: Follows security best practices, prevents credential leaks

6. **Q: Do we need CLI tools for these features?**
   - **Proposed**: Library-first, optional CLI as future enhancement
   - **Rationale**: Library provides maximum flexibility, CLI adds convenience later

## Related Changes

This proposal establishes foundation for future enhancements:

- **Bulk FHIR** - Requires SMART Backend Services (Phase 2)
- **IPA IG** - Uses US Core profile framework (Phase 3)
- **mCODE IG** - Oncology-specific profiles using same IG pattern
- **DICOM SEG Mapping** - DICOM Segmentation → FHIR ImagingSelection
- **DICOM GSPS Mapping** - Grayscale Presentation State → FHIR annotations
- **CDA Support** - Clinical Document Architecture using similar mapping patterns

## References

**Standards:**
- [SMART App Launch 2.0](http://hl7.org/fhir/smart-app-launch/)
- [US Core Implementation Guide v6.1.0](http://hl7.org/fhir/us/core/)
- [FHIR R5 Subscriptions](http://hl7.org/fhir/R5/subscriptions.html)
- [DICOM PS3.16 Content Mapping](https://dicom.nema.org/medical/dicom/current/output/chtml/part16/chapter_A.html)

**Reference Implementations:**
- [HAPI FHIR](https://github.com/hapifhir/hapi-fhir) - Java FHIR server (SMART + Subscriptions)
- [pydicom](https://github.com/pydicom/pydicom) - Python DICOM library
- [dicom-fhir-converter](https://github.com/microsoft/dicom-server) - Microsoft DICOM → FHIR

**Testing Resources:**
- [SMART Sandbox](https://launch.smarthealthit.org/) - SMART conformance testing
- [US Core Examples](https://hl7.org/fhir/us/core/examples.html) - Official test fixtures
- [DICOM Test Files](https://www.dicomlibrary.com/) - Public DICOM datasets
