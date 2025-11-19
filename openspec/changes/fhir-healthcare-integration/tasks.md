# Implementation Tasks: FHIR Healthcare Integration

## Overview

Implementation broken into 4 phases following proposal sequence. Each task is small, verifiable, and delivers
user-visible progress. Dependencies clearly marked. Estimated timeline: ~13 weeks total.

---

## Phase 1: DICOM to FHIR Mapping (Weeks 1-3)

### 1.1 Package Setup & Scaffolding

- [ ] **Create `fhir/mapping/` package structure**
  - Create directory: `fhir/mapping/`
  - Create files: `options.go`, `errors.go`, `doc.go` (package docs)
  - Define `MappingOptions` struct with common configuration
  - Define sentinel errors: `ErrMissingStudyUID`, `ErrMissingPatientID`, `ErrInvalidModality`
  - **Validation**: Package compiles with `go build ./fhir/mapping`
  - **Time**: 1 hour

- [ ] **Add mapping package documentation**
  - Write package-level godoc in `doc.go` with examples
  - Document common use cases and patterns
  - Add links to DICOM/FHIR specs
  - **Validation**: `go doc github.com/codeninja55/go-radx/fhir/mapping` shows docs
  - **Time**: 1 hour

### 1.2 Patient Mapping

- [ ] **Implement `DICOMPatientToFHIRPatient()` function**
  - File: `fhir/mapping/patient.go`
  - Extract DICOM tags: PatientName, PatientID, PatientBirthDate, PatientSex
  - Map to FHIR Patient fields: name, identifier, birthDate, gender
  - Handle missing optional fields gracefully
  - Map DICOM sex codes to FHIR gender codes
  - **Validation**: Unit tests for full/partial patient data
  - **Time**: 3 hours

- [ ] **Add patient mapping tests**
  - File: `fhir/mapping/patient_test.go`
  - Test full patient demographics mapping
  - Test minimal patient (ID only)
  - Test DICOM sex → FHIR gender code mappings
  - Test missing PatientName handling
  - Test invalid/empty patient ID error
  - **Validation**: `go test ./fhir/mapping -run TestPatient -v` passes
  - **Time**: 2 hours

### 1.3 ImagingStudy Mapping

- [ ] **Implement `DICOMStudyToImagingStudy()` core function**
  - File: `fhir/mapping/imagingstudy.go`
  - Extract StudyInstanceUID (required, fail if missing)
  - Map Study-level metadata: StudyDate, StudyTime, StudyDescription
  - Map Modality to FHIR coding
  - Create ImagingStudy identifier from UID
  - **Validation**: Unit test for single-series CT study
  - **Time**: 4 hours

- [ ] **Add series and instance mapping to ImagingStudy**
  - Iterate DICOM series within study
  - Map SeriesInstanceUID, SeriesNumber, Modality, SeriesDescription
  - Map instance-level metadata: SOPInstanceUID, SOPClassUID
  - Create `ImagingStudy.series[]` array with all series
  - **Validation**: Test multi-series study (3+ series)
  - **Time**: 3 hours

- [ ] **Add WADO-RS endpoint configuration**
  - Accept `WADOEndpoint` in `MappingOptions`
  - Generate Endpoint resource with WADO-RS URL
  - Link Endpoint to ImagingStudy
  - Format: `{baseURL}/studies/{studyUID}`
  - **Validation**: Test with/without WADO endpoint
  - **Time**: 2 hours

- [ ] **Add provenance tracking**
  - Accept `IncludeProvenance` option
  - Set `meta.source` with DICOM SOP Instance UID
  - Add extension with mapping timestamp, library version
  - **Validation**: Test provenance presence/absence
  - **Time**: 2 hours

- [ ] **Add ImagingStudy mapping tests**
  - File: `fhir/mapping/imagingstudy_test.go`
  - Test single-modality study (CT)
  - Test multi-modality study (CR + US)
  - Test study with 100+ instances
  - Test missing StudyInstanceUID error
  - Test missing optional fields (StudyDescription)
  - Test WADO endpoint generation
  - Test provenance extension
  - **Validation**: `go test ./fhir/mapping -run TestImagingStudy -v` passes
  - **Time**: 4 hours

### 1.4 DiagnosticReport (SR) Mapping

- [ ] **Implement basic DICOM SR → DiagnosticReport**
  - File: `fhir/mapping/diagnosticreport.go`
  - Support TID 1500 (Measurement Report) initially
  - Extract DocumentTitle → DiagnosticReport.code
  - Map ContentSequence to Observations (basic)
  - Set DiagnosticReport.status from SR Verification Flag
  - **Validation**: Test with sample TID 1500 SR
  - **Time**: 5 hours

- [ ] **Add measurement extraction to Observations**
  - Parse numeric measurements from SR ContentSequence
  - Create Observation resources with valueQuantity
  - Map measurement codes to Observation.code
  - Link Observations to DiagnosticReport via `result` references
  - **Validation**: Test SR with 3+ measurements
  - **Time**: 3 hours

- [ ] **Add reverse mapping: DiagnosticReport → DICOM SR**
  - Function: `DiagnosticReportToDICOMSR()`
  - Create DICOM SR dataset with TID 1500 structure
  - Map DiagnosticReport.code → DocumentTitle
  - Map Observations → ContentSequence measurements
  - Populate required DICOM tags (SOPClassUID, etc.)
  - **Validation**: Roundtrip test (DICOM → FHIR → DICOM)
  - **Time**: 6 hours

- [ ] **Add DiagnosticReport mapping tests**
  - File: `fhir/mapping/diagnosticreport_test.go`
  - Test TID 1500 SR → DiagnosticReport
  - Test measurement Observations extraction
  - Test unsupported TID error (TID 9999)
  - Test DiagnosticReport → SR conversion
  - Test roundtrip metadata preservation
  - **Validation**: `go test ./fhir/mapping -run TestDiagnosticReport -v` passes
  - **Time**: 3 hours

### 1.5 Integration & Validation

- [ ] **Add optional FHIR output validation**
  - Accept `ValidateOutput` option in `MappingOptions`
  - Call existing `fhir/validation` on generated resources
  - Return validation errors if present
  - Default: skip validation for performance
  - **Validation**: Test with valid/invalid FHIR output
  - **Time**: 2 hours

- [ ] **Add real-world DICOM file tests**
  - Download sample DICOM files to `testdata/dicom/mapping/`
  - Test with CT, MR, CR, US studies from DICOM Library
  - Test with Orthanc-generated studies
  - Verify FHIR JSON validity with validator
  - **Validation**: 10+ real DICOM files convert successfully
  - **Time**: 4 hours

- [ ] **Add performance benchmarks**
  - File: `fhir/mapping/benchmark_test.go`
  - Benchmark `DICOMStudyToImagingStudy` with 100-instance study
  - Target: <50ms per study
  - Benchmark patient mapping: <5ms
  - **Validation**: `go test -bench=. ./fhir/mapping` shows results
  - **Time**: 2 hours

- [ ] **Write mapping user guide**
  - File: `docs/user-guide/fhir/mapping.md`
  - Getting started examples
  - WADO-RS configuration
  - Handling missing DICOM tags
  - Performance tuning tips
  - **Validation**: Documentation builds with `mise docs:build`
  - **Time**: 3 hours

**Phase 1 Total: ~48 hours (3 weeks @ 16 hrs/week)**

---

## Phase 2: SMART on FHIR (Weeks 4-7)

### 2.1 Package Setup & Core Types

- [ ] **Create `fhir/smart/` package structure**
  - Create directory: `fhir/smart/`
  - Create subdirectories: `auth/`, `launch/`
  - Create files: `config.go`, `client.go`, `errors.go`, `doc.go`
  - Define `Config`, `Client`, `Token` types
  - Define sentinel errors: `ErrSMARTNotSupported`, `ErrInvalidClient`, etc.
  - **Validation**: Package compiles
  - **Time**: 2 hours

- [ ] **Add external dependencies**
  - Add `golang.org/x/oauth2` to `go.mod`
  - Add `github.com/google/uuid` to `go.mod`
  - Run `go mod tidy` and `go mod vendor` (if vendoring)
  - **Validation**: `go mod verify` succeeds
  - **Time**: 30 minutes

### 2.2 SMART Capability Discovery

- [ ] **Implement SMART endpoint discovery**
  - File: `fhir/smart/capabilities.go`
  - Fetch CapabilityStatement from `{FHIRBaseURL}/metadata`
  - Extract OAuth URIs from SMART extensions
  - Parse authorize and token URLs
  - Cache CapabilityStatement for performance
  - **Validation**: Test with mock FHIR server
  - **Time**: 3 hours

- [ ] **Add explicit endpoint configuration**
  - Support `SkipDiscovery` option in `Config`
  - Use provided `AuthorizeURL` and `TokenURL` if present
  - Fallback to discovery if not provided
  - **Validation**: Test both discovery and explicit config
  - **Time**: 1 hour

- [ ] **Add capability detection**
  - Function: `GetCapabilities()`
  - Parse SMART capability codes from CapabilityStatement
  - Return supported features: `launch-ehr`, `launch-standalone`, `client-confidential-symmetric`
  - **Validation**: Test with various CapabilityStatements
  - **Time**: 2 hours

### 2.3 EHR Launch Flow

- [ ] **Implement EHR launch authorization URL builder**
  - File: `fhir/smart/launch/ehr.go`
  - Function: `GetAuthorizationURL(launchCtx, scopes)`
  - Include `launch` parameter with launch token
  - Generate random `state` parameter (store for validation)
  - Add `aud` (FHIR base URL)
  - Format all query parameters correctly
  - **Validation**: Unit test URL structure
  - **Time**: 3 hours

- [ ] **Implement authorization code exchange**
  - File: `fhir/smart/auth/oauth2.go`
  - Function: `ExchangeCode(code, state)`
  - Validate state matches stored value
  - POST to token endpoint with authorization_code grant
  - Parse access token response
  - Extract patient ID from token claims (if present)
  - **Validation**: Test with mock OAuth2 server
  - **Time**: 4 hours

- [ ] **Implement launch context resolution**
  - File: `fhir/smart/launch/context.go`
  - Function: `ResolveContext(token)`
  - Parse JWT access token (if JWT format)
  - Extract claims: `patient`, `encounter`, `fhirUser`, `scope`
  - Return `LaunchContext` struct
  - **Validation**: Test with sample JWT tokens
  - **Time**: 2 hours

### 2.4 Standalone Launch Flow

- [ ] **Implement standalone launch authorization URL builder**
  - File: `fhir/smart/launch/standalone.go`
  - Function: `GetAuthorizationURL(nil, scopes)` (no launch token)
  - Omit `launch` parameter
  - Include all other OAuth2 parameters
  - **Validation**: Test URL without launch parameter
  - **Time**: 1 hour

- [ ] **Add patient selection handling**
  - Document that patient selection occurs during auth
  - Extract patient ID from token response
  - No special client-side logic needed
  - **Validation**: Integration test documentation
  - **Time**: 1 hour

### 2.5 PKCE Implementation

- [ ] **Implement PKCE parameter generation**
  - File: `fhir/smart/auth/pkce.go`
  - Function: `GeneratePKCE()`
  - Generate random code verifier (43-128 chars, base64url)
  - Compute SHA-256 hash of verifier
  - Base64url encode hash → code challenge
  - Return verifier and challenge
  - **Validation**: Test verifier/challenge pair validity
  - **Time**: 2 hours

- [ ] **Add PKCE to authorization URL**
  - Function: `GetAuthorizationURLWithPKCE()`
  - Include `code_challenge` and `code_challenge_method=S256`
  - Store verifier securely for later
  - **Validation**: Test PKCE URL parameters
  - **Time**: 1 hour

- [ ] **Add PKCE to token exchange**
  - Function: `ExchangeCodeWithPKCE(code, state, verifier)`
  - Include `code_verifier` in token request
  - Server validates verifier matches challenge
  - **Validation**: Test with PKCE-enabled mock server
  - **Time**: 2 hours

### 2.6 Backend Services Authorization

- [ ] **Implement JWT assertion creation**
  - File: `fhir/smart/auth/backend.go`
  - Function: `CreateJWTAssertion(claims, privateKey)`
  - Sign JWT with RS384 algorithm
  - Include claims: iss, sub, aud, exp, jti
  - Use `github.com/golang-jwt/jwt/v5` (add dependency)
  - **Validation**: Test JWT signature verification
  - **Time**: 3 hours

- [ ] **Implement Backend Services token request**
  - Function: `BackendServicesAuth(jwt, scopes)`
  - POST to token endpoint with `client_credentials` grant
  - Include `client_assertion_type` and `client_assertion` (JWT)
  - Request `system/*` scopes
  - Parse access token response
  - **Validation**: Test with mock auth server
  - **Time**: 3 hours

### 2.7 Token Management

- [ ] **Implement token caching**
  - File: `fhir/smart/auth/token_manager.go`
  - Store tokens in memory with expiry
  - Key by client ID + scope combination
  - Thread-safe with mutex
  - **Validation**: Test concurrent access
  - **Time**: 2 hours

- [ ] **Implement automatic token refresh**
  - Function: `RefreshToken(refreshToken)`
  - POST to token endpoint with `refresh_token` grant
  - Update cached token with new access token
  - Handle refresh token expiry error
  - **Validation**: Test token refresh before expiry
  - **Time**: 3 hours

- [ ] **Add proactive token refresh**
  - Check token expiry before each request
  - Refresh if <5 minutes remaining
  - Automatic background refresh (optional)
  - **Validation**: Test auto-refresh timing
  - **Time**: 2 hours

### 2.8 Scope Management

- [ ] **Implement scope parsing and validation**
  - File: `fhir/smart/scopes.go`
  - Function: `ParseScopes(scopeString)`
  - Split space-delimited scope string
  - Parse scope format: `context/ResourceType.action`
  - Function: `HasScope(token, requiredScope)`
  - Support wildcard matching: `patient/*.read` matches `patient/Observation.read`
  - **Validation**: Test scope parsing and matching
  - **Time**: 3 hours

### 2.9 Error Handling & Testing

- [ ] **Implement OAuth2 error handling**
  - Parse `error` and `error_description` from responses
  - Map to specific errors: `ErrAuthorizationDenied`, `ErrInvalidClient`, `ErrInvalidScope`
  - Include error descriptions in returned errors
  - **Validation**: Test all OAuth2 error scenarios
  - **Time**: 2 hours

- [ ] **Add comprehensive SMART tests**
  - Files: `*_test.go` across package
  - Test discovery with mock CapabilityStatement
  - Test EHR launch flow end-to-end
  - Test standalone launch flow
  - Test PKCE generation and validation
  - Test Backend Services JWT assertion
  - Test token caching and refresh
  - Test scope parsing and matching
  - **Validation**: `go test ./fhir/smart/... -v` passes (>85% coverage)
  - **Time**: 8 hours

- [ ] **Add SMART conformance tests**
  - Integration tests with [SMART Launcher](https://launch.smarthealthit.org/)
  - Test EHR launch against live server
  - Test token exchange and refresh
  - Document conformance test results
  - **Validation**: Manual testing against SMART Launcher
  - **Time**: 4 hours

- [ ] **Write SMART user guide**
  - File: `docs/user-guide/fhir/smart.md`
  - SMART App Launch quickstart
  - EHR launch example
  - Standalone launch example
  - Backend Services example
  - Token management best practices
  - Security considerations
  - **Validation**: Documentation builds successfully
  - **Time**: 4 hours

**Phase 2 Total: ~60 hours (4 weeks @ 15 hrs/week)**

---

## Phase 3: US Core IG Support (Weeks 8-10)

### 3.1 Package Setup & Generic IG Framework

- [ ] **Create `fhir/ig/` package structure**
  - Create directory: `fhir/ig/`
  - Create subdirectory: `uscore/`
  - Create files: `profile.go`, `validator.go`, `extension.go`, `doc.go`
  - Define `Profile`, `ExtensionDefinition`, `ValueSetBinding` types
  - **Validation**: Package compiles
  - **Time**: 2 hours

- [ ] **Implement generic IG validation framework**
  - File: `fhir/ig/validator.go`
  - Type: `Validator` with profile registry
  - Function: `RegisterProfile(profile)`
  - Function: `ValidateProfile(resource, profile)`
  - Call base FHIR validation first, then IG-specific rules
  - **Validation**: Test with mock profile
  - **Time**: 4 hours

- [ ] **Implement Must Support validation**
  - Function: `CheckMustSupport(resource, profile)`
  - Use reflection to check field presence
  - Return list of missing Must Support elements
  - Handle nested fields (e.g., `identifier.system`)
  - **Validation**: Test with Patient resource
  - **Time**: 3 hours

### 3.2 US Core Profiles

- [ ] **Define US Core Patient profile**
  - File: `fhir/ig/uscore/profiles.go`
  - Variable: `ProfilePatient`
  - Must Support: `identifier`, `name`, `gender`
  - Extensions: US Core Race, Ethnicity
  - Value set bindings: AdministrativeGender
  - **Validation**: Profile structure complete
  - **Time**: 2 hours

- [ ] **Define US Core Observation profile**
  - Variable: `ProfileObservation`
  - Must Support: `status`, `category`, `code`, `subject`, `value[x]`
  - Vital Signs profile (sub-profile)
  - LOINC code bindings
  - **Validation**: Profile structure complete
  - **Time**: 2 hours

- [ ] **Define US Core DiagnosticReport profile**
  - Variable: `ProfileDiagnosticReport`
  - Must Support: `status`, `category`, `code`, `subject`, `effectiveDateTime`
  - Support for `imagingStudy` references
  - **Validation**: Profile structure complete
  - **Time**: 2 hours

- [ ] **Define US Core ImagingStudy profile (if in v6.1.0)**
  - Variable: `ProfileImagingStudy`
  - Check if ImagingStudy profile exists in US Core v6.1.0
  - Define Must Support elements if present
  - **Validation**: Profile matches US Core spec
  - **Time**: 2 hours

- [ ] **Define additional US Core profiles**
  - `ProfilePractitioner`
  - `ProfileOrganization`
  - `ProfileEncounter`
  - Must Support elements for each
  - **Validation**: All profiles compile
  - **Time**: 4 hours

### 3.3 US Core Extensions

- [ ] **Implement US Core Race extension helpers**
  - File: `fhir/ig/uscore/extensions.go`
  - Type: `RaceCategory` with OMBCategory, Detailed, Text
  - Function: `AddRaceExtension(patient, race)`
  - Function: `GetRaceExtension(patient)`
  - Create proper extension structure per US Core spec
  - **Validation**: Test race extension add/extract
  - **Time**: 3 hours

- [ ] **Implement US Core Ethnicity extension helpers**
  - Type: `EthnicityCategory`
  - Function: `AddEthnicityExtension(patient, ethnicity)`
  - Function: `GetEthnicityExtension(patient)`
  - **Validation**: Test ethnicity extension add/extract
  - **Time**: 2 hours

- [ ] **Implement US Core Birth Sex extension**
  - Function: `AddBirthSexExtension(patient, birthSex)`
  - Function: `GetBirthSexExtension(patient)`
  - Value set: `M`, `F`, `UNK`
  - **Validation**: Test birth sex extension
  - **Time**: 1 hour

- [ ] **Validate extension structure**
  - Function: `ValidateRaceExtension(patient)`
  - Check required `text` field
  - Validate OMB category codes (urn:oid:2.16.840.1.113883.6.238)
  - Return validation errors
  - **Validation**: Test with valid/invalid extensions
  - **Time**: 2 hours

### 3.4 Value Set Bindings

- [ ] **Implement AdministrativeGender value set**
  - File: `fhir/ig/uscore/valuesets.go`
  - Valid codes: `male`, `female`, `other`, `unknown`
  - Function: `ValidateAdministrativeGender(code)`
  - Required binding (reject invalid codes)
  - **Validation**: Test valid/invalid gender codes
  - **Time**: 1 hour

- [ ] **Implement OMB Race Categories value set**
  - Valid codes from OMB standard (2.16.840.1.113883.6.238)
  - Function: `ValidateOMBRaceCategory(code)`
  - Extensible binding (allow other codes with warning)
  - **Validation**: Test OMB race codes
  - **Time**: 2 hours

- [ ] **Implement OMB Ethnicity Categories value set**
  - Function: `ValidateOMBEthnicityCategory(code)`
  - Extensible binding
  - **Validation**: Test OMB ethnicity codes
  - **Time**: 1 hour

- [ ] **Add extensible binding support**
  - Distinguish `required` vs `extensible` vs `preferred` bindings
  - Required: Reject invalid codes
  - Extensible: Warn but allow other codes
  - Preferred: No enforcement, just recommendation
  - **Validation**: Test all binding strengths
  - **Time**: 2 hours

### 3.5 US Core-Specific Validation

- [ ] **Implement identifier.system requirement validation**
  - US Core requires `identifier.system` for all identifiers
  - Check Patient.identifier[*].system is present
  - Return error if missing
  - **Validation**: Test with/without system
  - **Time**: 1 hour

- [ ] **Implement dataAbsentReason validation**
  - Observation must have value[x] OR dataAbsentReason (not both)
  - Validate mutual exclusion
  - **Validation**: Test all combinations
  - **Time**: 2 hours

### 3.6 Testing & Documentation

- [ ] **Add US Core validation tests**
  - File: `fhir/ig/uscore/validator_test.go`
  - Test US Core Patient validation (full and partial)
  - Test US Core Observation validation
  - Test US Core DiagnosticReport validation
  - Test Must Support checking
  - Test extension validation
  - Test value set bindings
  - **Validation**: `go test ./fhir/ig/uscore -v` passes (>85% coverage)
  - **Time**: 6 hours

- [ ] **Test with official US Core examples**
  - Download US Core examples from HL7 spec
  - Add to `testdata/fhir/uscore/`
  - Validate all examples pass US Core validation
  - Target: 95%+ examples pass
  - **Validation**: All US Core examples validate successfully
  - **Time**: 4 hours

- [ ] **Add custom IG tests**
  - Test generic IG framework with custom profile
  - Test profile registration and validation
  - Test extensibility for non-US-Core IGs
  - **Validation**: Custom IG validation works
  - **Time**: 2 hours

- [ ] **Write US Core user guide**
  - File: `docs/user-guide/fhir/uscore.md`
  - US Core validation quickstart
  - Must Support element checking
  - US Core extensions (race, ethnicity, birthsex)
  - Value set bindings
  - Custom IG development guide
  - **Validation**: Documentation builds successfully
  - **Time**: 3 hours

**Phase 3 Total: ~48 hours (3 weeks @ 16 hrs/week)**

---

## Phase 4: FHIR Subscriptions (Weeks 11-13)

### 4.1 Package Setup & Core Types

- [ ] **Create `fhir/subscriptions/` package structure**
  - Create directory: `fhir/subscriptions/`
  - Create files: `manager.go`, `subscription.go`, `errors.go`, `doc.go`
  - Define `Manager`, `Subscription`, `ResourceEvent` types
  - Define sentinel errors: `ErrSubscriptionNotFound`, `ErrInvalidWebhookURL`, etc.
  - **Validation**: Package compiles
  - **Time**: 2 hours

- [ ] **Implement subscription storage interface**
  - File: `fhir/subscriptions/store.go`
  - Interface: `SubscriptionStore` with Save, Get, List, Delete, FindByTopic
  - Document interface contract
  - **Validation**: Interface compiles
  - **Time**: 1 hour

- [ ] **Implement in-memory store**
  - File: `fhir/subscriptions/memory_store.go`
  - Type: `MemoryStore` implementing `SubscriptionStore`
  - Use `map[string]*Subscription` with mutex
  - Thread-safe operations
  - **Validation**: Test concurrent access
  - **Time**: 3 hours

### 4.2 Subscription Lifecycle Management

- [ ] **Implement subscription creation**
  - File: `fhir/subscriptions/manager.go`
  - Function: `Subscribe(ctx, subscription)`
  - Validate subscription structure
  - Store in persistence layer
  - Assign unique subscription ID
  - Set initial status: `"requested"` or `"active"`
  - **Validation**: Test subscription creation
  - **Time**: 2 hours

- [ ] **Implement subscription activation**
  - Function: `ActivateSubscription(ctx, id)`
  - Change status from `"requested"` to `"active"`
  - Start monitoring for matching events
  - **Validation**: Test activation flow
  - **Time**: 1 hour

- [ ] **Implement subscription deactivation**
  - Function: `DeactivateSubscription(ctx, id)`
  - Change status to `"off"`
  - Stop event monitoring
  - Preserve subscription in storage
  - **Validation**: Test deactivation
  - **Time**: 1 hour

- [ ] **Implement subscription deletion**
  - Function: `DeleteSubscription(ctx, id)`
  - Remove from storage completely
  - Stop all deliveries
  - **Validation**: Test deletion
  - **Time**: 1 hour

### 4.3 Event Monitoring & Filtering

- [ ] **Implement resource change notification**
  - Function: `NotifyChange(ctx, event)`
  - Accept `ResourceEvent` with eventType and resource
  - Find matching active subscriptions
  - Apply filters to determine matches
  - Trigger notifications for matches
  - **Validation**: Test event notification flow
  - **Time**: 3 hours

- [ ] **Implement subscription filter matching**
  - File: `fhir/subscriptions/filters.go`
  - Function: `MatchesFilter(resource, filter)`
  - Parse FHIR search parameters from filter string
  - Check resource fields match filter criteria
  - Support: identifier, tag, date, status filters
  - **Validation**: Test filter matching logic
  - **Time**: 4 hours

- [ ] **Implement topic-based matching**
  - File: `fhir/subscriptions/topic.go`
  - Type: `SubscriptionTopic` (R5 resource)
  - Function: `RegisterTopic(topic)`
  - Match events against topic `resourceTrigger` rules
  - Check `supportedInteraction` (create, update, delete)
  - **Validation**: Test topic matching
  - **Time**: 3 hours

### 4.4 Webhook Delivery

- [ ] **Implement webhook delivery**
  - File: `fhir/subscriptions/webhook.go`
  - Function: `DeliverWebhook(ctx, subscription, bundle)`
  - POST notification bundle to subscriber endpoint
  - Set `Content-Type: application/fhir+json`
  - Handle HTTP response codes
  - **Validation**: Test with mock HTTP server
  - **Time**: 3 hours

- [ ] **Implement retry logic with exponential backoff**
  - Retry failed webhooks: 3 attempts
  - Delays: 1s, 2s, 4s (exponential backoff)
  - Configurable retry policy
  - **Validation**: Test retry behavior
  - **Time**: 3 hours

- [ ] **Implement webhook timeout**
  - Set request timeout (default: 5 seconds)
  - Cancel request on timeout
  - Count as delivery failure
  - **Validation**: Test timeout handling
  - **Time**: 1 hour

- [ ] **Add dead letter queue logging**
  - Log persistent delivery failures after max retries
  - Include subscription ID, endpoint, error details
  - Provide hook for custom DLQ handlers
  - **Validation**: Test DLQ logging
  - **Time**: 2 hours

### 4.5 Notification Bundle Building

- [ ] **Implement notification bundle builder**
  - File: `fhir/subscriptions/bundle.go`
  - Function: `BuildNotificationBundle(event, subscription)`
  - Create `history` bundle type
  - Add SubscriptionStatus as first entry
  - Add resource entries based on `content` setting
  - Set bundle timestamp and ID
  - **Validation**: Test bundle structure
  - **Time**: 4 hours

- [ ] **Implement content modes**
  - Support `empty` (no resource data)
  - Support `id-only` (URL only)
  - Support `full-resource` (complete resource)
  - **Validation**: Test all content modes
  - **Time**: 2 hours

- [ ] **Add SubscriptionStatus generation**
  - Create SubscriptionStatus resource
  - Set `type` = `"event-notification"` or `"heartbeat"`
  - Include subscription reference, topic URL
  - Add event sequence number
  - Add focus resource reference
  - **Validation**: Test SubscriptionStatus structure
  - **Time**: 2 hours

### 4.6 Heartbeat & Error Handling

- [ ] **Implement heartbeat notifications**
  - Function: `SendHeartbeat(ctx, subscription)`
  - Send when no events for `heartbeatPeriod` seconds
  - Create heartbeat bundle with SubscriptionStatus
  - Verify subscriber endpoint health
  - **Validation**: Test heartbeat timing
  - **Time**: 3 hours

- [ ] **Implement subscription error state**
  - Track consecutive delivery failures
  - Change status to `"error"` after threshold (5 failures)
  - Stop further delivery attempts
  - Log error state
  - **Validation**: Test error state transition
  - **Time**: 2 hours

- [ ] **Implement subscription recovery**
  - Function: `ReactivateSubscription(ctx, id)`
  - Manually recover from error state
  - Reset failure counter
  - Resume deliveries
  - **Validation**: Test recovery flow
  - **Time**: 1 hour

### 4.7 R4 Backport Compatibility

- [ ] **Implement R4 subscription support**
  - File: `fhir/subscriptions/r4_compat.go`
  - Function: `SubscribeR4(ctx, r4Subscription)`
  - Convert R4 `criteria` to R5 `filter`
  - Map R4 `channel` to R5 `endpoint` and `channelType`
  - Map R4 `payload` to R5 `contentType`
  - **Validation**: Test R4 → R5 conversion
  - **Time**: 3 hours

- [ ] **Add R4 compatibility tests**
  - Test R4 subscription creation
  - Test criteria → filter mapping
  - Test channel → endpoint mapping
  - **Validation**: R4 subscriptions work correctly
  - **Time**: 2 hours

### 4.8 Testing & Documentation

- [ ] **Add comprehensive subscription tests**
  - Files: `*_test.go` across package
  - Test subscription lifecycle (create, activate, delete)
  - Test event notification and filtering
  - Test webhook delivery (success, failure, retry)
  - Test notification bundle building
  - Test heartbeat notifications
  - Test error state and recovery
  - Test R4 compatibility
  - **Validation**: `go test ./fhir/subscriptions -v` passes (>85% coverage)
  - **Time**: 8 hours

- [ ] **Add performance tests**
  - Benchmark notification delivery
  - Load test: 10,000 notifications/second
  - Test concurrent subscription management
  - **Validation**: Performance targets met
  - **Time**: 3 hours

- [ ] **Add integration tests**
  - Mock HTTP server for webhook delivery
  - Test end-to-end subscription flow
  - Test retry logic with controlled failures
  - **Validation**: Integration tests pass
  - **Time**: 4 hours

- [ ] **Write subscriptions user guide**
  - File: `docs/user-guide/fhir/subscriptions.md`
  - Creating and managing subscriptions
  - Webhook endpoint implementation
  - Subscription filters and topics
  - Monitoring and troubleshooting
  - R4 compatibility
  - **Validation**: Documentation builds successfully
  - **Time**: 4 hours

**Phase 4 Total: ~60 hours (3 weeks @ 20 hrs/week)**

---

## Cross-Cutting Tasks (Ongoing)

- [ ] **Update CHANGELOG.md**
  - Add entry for each phase completion
  - Follow Keep a Changelog format
  - Link to relevant PRs
  - **Validation**: CHANGELOG updated
  - **Time**: 1 hour (spread across phases)

- [ ] **Update README.md**
  - Add new features to feature list
  - Update examples with SMART, US Core, Subscriptions
  - Update documentation links
  - **Validation**: README accurate
  - **Time**: 2 hours (after Phase 4)

- [ ] **Add examples directory**
  - Create `examples/` directory with runnable examples
  - DICOM mapping example
  - SMART authorization example
  - US Core validation example
  - Subscription management example
  - **Validation**: Examples run successfully
  - **Time**: 4 hours (after Phase 4)

- [ ] **Update API documentation**
  - Ensure 100% godoc coverage for all exported types
  - Add package-level examples
  - Cross-link related packages
  - **Validation**: `go doc` shows complete docs
  - **Time**: 3 hours (ongoing)

- [ ] **Run security audit**
  - Run `govulncheck` on all new dependencies
  - Review OAuth2 security implementation
  - Audit webhook URL validation (SSRF prevention)
  - Review token storage (no plaintext logging)
  - **Validation**: No security issues found
  - **Time**: 2 hours (after Phase 2 & 4)

- [ ] **Performance profiling**
  - Profile DICOM mapping with pprof
  - Profile subscription notification delivery
  - Optimize hot paths if needed
  - **Validation**: Performance targets met
  - **Time**: 3 hours (after each phase)

---

## Validation & Release (Week 14)

- [ ] **Run full test suite**
  - `go test ./fhir/mapping/... -v -cover`
  - `go test ./fhir/smart/... -v -cover`
  - `go test ./fhir/ig/uscore/... -v -cover`
  - `go test ./fhir/subscriptions/... -v -cover`
  - **Validation**: All tests pass, >85% coverage
  - **Time**: 1 hour

- [ ] **Run linters**
  - `golangci-lint run ./fhir/...`
  - Fix any linter issues
  - **Validation**: 100% linter pass
  - **Time**: 1 hour

- [ ] **Build documentation**
  - `mise docs:build`
  - Review generated docs for completeness
  - Fix broken links
  - **Validation**: Documentation builds without errors
  - **Time**: 1 hour

- [ ] **Create release PR**
  - Merge all changes to `main`
  - Tag release (e.g., `v0.5.0`)
  - Generate release notes from CHANGELOG
  - **Validation**: Release tagged successfully
  - **Time**: 1 hour

- [ ] **Publish release**
  - Push tag to GitHub
  - GitHub Actions publishes documentation
  - Monitor for issues
  - **Validation**: Release published
  - **Time**: 30 minutes

---

## Total Effort Summary

| Phase | Tasks | Time Estimate |
|-------|-------|---------------|
| Phase 1: DICOM Mapping | 17 tasks | ~48 hours (3 weeks) |
| Phase 2: SMART on FHIR | 25 tasks | ~60 hours (4 weeks) |
| Phase 3: US Core IG | 22 tasks | ~48 hours (3 weeks) |
| Phase 4: Subscriptions | 26 tasks | ~60 hours (3 weeks) |
| Cross-Cutting | 8 tasks | ~15 hours (ongoing) |
| Validation & Release | 6 tasks | ~5 hours (1 week) |
| **Total** | **104 tasks** | **~236 hours (~13-14 weeks)** |

---

## Parallelization Opportunities

Tasks that can be done in parallel (different developers):

- **Phase 1 & 2**: After Phase 1 setup, SMART work can start independently
- **Documentation**: Can be written alongside implementation
- **Testing**: Test files can be written while implementation is in progress
- **Examples**: Can be created as features stabilize

With 2-3 developers working in parallel, timeline can be compressed to ~8-10 weeks.

---

## Dependencies

**External Packages** (added during implementation):
1. `golang.org/x/oauth2` - Phase 2.1
2. `github.com/google/uuid` - Phase 2.1
3. `github.com/golang-jwt/jwt/v5` - Phase 2.6 (optional)

**Internal Dependencies**:
- All phases depend on existing `fhir/r5/resources` and `fhir/validation` packages
- Phase 3 (US Core) depends on `fhir/validation` framework
- Phases are otherwise independent and can proceed in any order

---

## Success Criteria

Each phase must meet:

1. ✅ **All tests pass** with >85% coverage
2. ✅ **All linters pass** (golangci-lint)
3. ✅ **Documentation complete** (godoc + user guide)
4. ✅ **Examples working** (runnable code)
5. ✅ **Performance targets met** (benchmarks)
6. ✅ **No breaking changes** to existing code

Final release must meet:

7. ✅ **Full integration test suite passes**
8. ✅ **Security audit complete**
9. ✅ **CHANGELOG updated**
10. ✅ **README accurate and up-to-date**
