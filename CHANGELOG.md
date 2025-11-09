# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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

### Changed
- Updated Go version to 1.25.4
- Updated golangci-lint to v2.4.0 for Go 1.25 compatibility
- Coverage threshold set to informational only (not blocking)
- Improved CI/CD workflows for better reliability

### Fixed
- golangci-lint compatibility with Go 1.25.4
- Test compilation errors in summary and validation tests
- Coverage calculation to exclude generated resource definitions

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
