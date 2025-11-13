# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **RadX CLI Enhancements**:
  - Group filtering for DICOM dump command (`--group`/`-g` flag)
  - Named group aliases (patient, study, image, pixel, metadata, overlay)
  - Combined group and tag filtering support
  - Improved tag name display using DICOM dictionary keywords instead of tag notation
- **FHIR R5 Type Safety Refactoring** (#10):
  - Generic helper functions for type-safe resource operations
  - `UnmarshalResource[T]()` for type-safe unmarshaling of polymorphic fields
  - `AddContainedResource[T]()` for type-safe addition of contained resources
  - `GetContainedResourceByID()` for resource retrieval by ID
  - Comprehensive recursive validation for nested array choice types
  - Generated code headers with metadata (FHIR version, timestamp, generator version)
  - R5 schema download script for reproducible builds
  - Comprehensive generator testing (12 test functions for type mapper, 5 for code generator)
  - Choice type serialization tests (14 test cases covering Patient, Observation)
  - Bundle operation tests (11 test cases for entry/outcome/issues handling)
  - Validation framework tests (80+ test cases, 100% pass rate)
- CODE_OF_CONDUCT.md using Contributor Covenant v2.1
- SUPPORT.md for community support guidance
- Comprehensive pkg.go.dev examples for FHIR resources
- CI/CD status badges to README
- Dependabot configuration for automated dependency updates
- Benchmark documentation and results
- FHIR R5 implementation with 13 resource types tested
  - Patient, Observation, Bundle (existing)
  - DiagnosticReport, ImagingStudy (radiology-focused)
  - Encounter, Condition, Procedure, MedicationRequest, ServiceRequest (clinical)
  - Organization, Practitioner, Location (administrative)
- Comprehensive test coverage improvements:
  - fhir package: 82.8% coverage
  - fhir/validation package: 84.1% coverage
  - fhir/primitives package: 90.9% coverage
- Automatic SemVer tagging and release workflow
- **DICOM Core Implementation** (#9):
  - Complete DICOM Part 10 file format support (read/write)
  - Full DICOM data dictionary with tag lookup
  - Support for all Value Representation (VR) types
  - Dataset operations and manipulation API
  - Transfer syntax support (Explicit VR LE, Implicit VR LE, Deflated)
  - Comprehensive UID generation following ISO 8824 standard
  - Directory reader for batch DICOM file processing
- **DIMSE Networking Protocol (DICOM Part 7 & 8)** (#9):
  - Association management (A-ASSOCIATE, A-RELEASE, A-ABORT)
  - Presentation context negotiation
  - Protocol Data Unit (PDU) encoding/decoding
  - DIMSE message services (C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE)
  - Service Class User (SCU) implementation
  - Service Class Provider (SCP) implementation with configurable handlers
  - Message fragmentation and reassembly
  - State machine for association lifecycle
  - Integration tests against Orthanc PACS
  - Fuzz testing for protocol robustness (21 fuzz tests covering PDU, DIMSE, DUL layers)
- **DICOM Test Infrastructure** (#9):
  - Synthetic DICOM test data generator (testdata/generate_nested_dicom.go)
  - PHI-free test fixtures with ~1,195 generated files
  - Organized testdata structure (testdata/dicom/ with nested series)
  - Fuzz test corpora for PDU and DIMSE message fuzzing
- **Benchmark Suite and CI/CD** (#9):
  - Comprehensive benchmark suite for DICOM operations (anonymize, dataset, LUT)
  - Comparative benchmarks for optimization validation
  - Memory profiling benchmarks for allocation analysis
  - GitHub Actions benchmark CI/CD workflow
  - Automated performance regression detection (>20% threshold)
  - Benchmark result comparison and PR comments
  - Baseline tracking for main branch

### Changed
- **BREAKING: FHIR R5 Type Safety** (#10):
  - `DomainResource.Contained` changed from `[]interface{}` to `[]json.RawMessage`
  - `Bundle.entry.resource` changed from `*any` to `json.RawMessage`
  - `BundleEntryResponse.outcome` changed from `*any` to `json.RawMessage`
  - `Bundle.issues` changed from `*any` to `json.RawMessage`
  - Regenerated all 158 R5 resources with updated generator
  - Regenerated all 44 R5 complex types with updated generator
  - Code generator now maps `Resource` type to `json.RawMessage` instead of `any`
  - Non-choice polymorphic types use `json.RawMessage` as fallback (was `any`)
- Updated Go version to 1.25.4
- Updated golangci-lint to v2.4.0 for Go 1.25 compatibility
- Coverage threshold set to informational only (not blocking)
- Improved CI/CD workflows for better reliability
- **Test data reorganization** (#9): Moved all DICOM files to testdata/dicom/ subdirectory
- **Benchmark improvements** (#9):
  - Fixed sub-benchmark naming for readable output
  - Optimized dataset creation using Copy() instead of recreation
  - Added throughput reporting (MB/s) for data-intensive operations
  - Improved benchmark accuracy by eliminating setup overhead

### Fixed
- golangci-lint compatibility with Go 1.25.4
- Test compilation errors in summary and validation tests
- Coverage calculation to exclude generated resource definitions
- **Security fix** (#9): PDU size limit validation to prevent DoS via memory exhaustion (dimse/pdu/data.go)

### Security
- Added PDU size limit enforcement in DIMSE protocol decoder (#9)
- Removed potentially PHI-containing test data (CTC_2 directory) (#9)
- All test data is now clearly marked as synthetic and PHI-free (#9)

## [0.1.0] - 2025-01-09

### Added
- Initial FHIR R4 backwards compatibility
- FHIR R5 complete resource implementation (158 resources)
- Core validation framework with:
  - Cardinality constraints (0..1, 1..1, 0..*, 1..*)
  - Required field validation
  - Choice type mutual exclusion
  - Enum/coded value validation
- FHIR primitive types with validation:
  - Date, DateTime, Time, Instant
  - ISO 8601 parsing and formatting
- Bundle navigation utilities:
  - Resource extraction and filtering
  - Reference resolution
  - Pagination support
- Summary mode serialization (40-70% payload reduction)
- SMART on FHIR support:
  - OAuth2 authorization framework
  - EHR launch flow
  - Standalone app launch
  - Token management
- DICOM Structured Report (SR) support:
  - FHIR Observation mapping
  - ImagingStudy integration
  - DiagnosticReport generation
- Comprehensive documentation with MkDocs
- Essential open source project files:
  - README.md with detailed feature documentation
  - CONTRIBUTING.md with contribution guidelines
  - SECURITY.md with security policy
  - LICENSE (MIT)
  - Issue and PR templates
- CI/CD workflows:
  - Automated testing on Ubuntu and macOS
  - golangci-lint integration
  - Code formatting checks
  - Coverage reporting to Codecov

### Infrastructure
- Go Modules setup with Go 1.25.4
- Mise task runner integration for development workflow
- MkDocs documentation site with Material theme
- GitHub Actions CI/CD pipeline

[Unreleased]: https://github.com/codeninja55/go-radx/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/codeninja55/go-radx/releases/tag/v0.1.0
