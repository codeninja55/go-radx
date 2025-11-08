# Changelog

All notable changes to go-radx will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial FHIR R5 support with 158 resources
- Comprehensive validation framework
  - Cardinality validation
  - Choice type validation
  - Required field validation
  - Nested resource validation
- FHIR primitive types
  - Date, DateTime, Time, Instant
  - Precision-aware parsing
  - Validation on construction
- Bundle utilities
  - Bundle navigation
  - Resource extraction
  - Reference resolution
- Primitive extension support
  - Parallel extension fields
  - Extension helper methods
- Summary mode for payload optimization
  - 40-70% size reduction
  - Summary field tagging
- Comprehensive documentation
  - User guides
  - API reference
  - Code examples
  - Troubleshooting guides

### Infrastructure

- Mise task runner integration
- MkDocs documentation site
- GitHub Pages deployment
- CGo dependency management
  - macOS (Homebrew)
  - Linux (apt, yum)
- Development tooling
  - golangci-lint
  - govulncheck
  - Test coverage reporting

## Release Guidelines

### Version Format

- **Major** (X.0.0) - Breaking changes
- **Minor** (0.X.0) - New features, backward compatible
- **Patch** (0.0.X) - Bug fixes, backward compatible

### Release Process

1. Update version in go.mod
2. Update CHANGELOG.md
3. Create git tag
4. Push tag to trigger release workflow
5. Publish release notes on GitHub

## Categories

### Added

New features and capabilities

### Changed

Changes to existing functionality

### Deprecated

Features marked for removal in future releases

### Removed

Features that have been removed

### Fixed

Bug fixes

### Security

Security fixes and improvements

## How to Read This Changelog

Each release section includes:

- **Version number** - Semantic version (e.g., v0.1.0)
- **Release date** - ISO 8601 format (YYYY-MM-DD)
- **Changes** - Categorized list of changes
- **Migration notes** - Breaking changes and upgrade path

## Previous Versions

### [0.0.0] - 2025-01-09

Initial project setup and repository structure.

### Added

- Project repository initialization
- Basic Go module setup
- Documentation structure
- Mise configuration

## Future Releases

Planned features for upcoming releases:

### FHIR Enhancements

- FHIRPath expression support
- Profile-specific validation
- Terminology binding validation
- Reference integrity checking
- Search parameter support

### DICOM Support

- DICOM file reading/writing
- Tag dictionary
- Transfer syntax support
- Image decompression
  - JPEG (via libjpeg-turbo)
  - JPEG 2000 (via OpenJPEG)
  - RLE
- Pixel data handling
- DICOM networking (DIMSE)

### HL7 Support

- HL7 v2 message parsing
- HL7 v2 message generation
- Segment extraction
- Field validation

### Integration

- FHIR to DICOM mapping
- DICOM to FHIR mapping
- HL7 to FHIR mapping
- ImagingStudy generation from DICOM

### Performance

- Lazy loading for large resources
- Streaming JSON parsing
- Memory optimization
- Concurrent validation

### Developer Experience

- More code examples
- Video tutorials
- Interactive documentation
- Performance benchmarks

## Contributing

Help us keep this changelog up to date:

- Report bugs and their fixes
- Document new features
- Note breaking changes
- Update migration guides

See our [Contributing Guide](../development/contributing.md) for details.

## Links

- [Latest Release](https://github.com/codeninja55/go-radx/releases/latest)
- [All Releases](https://github.com/codeninja55/go-radx/releases)
- [Roadmap](https://github.com/codeninja55/go-radx/projects)
- [Issues](https://github.com/codeninja55/go-radx/issues)

## Stay Updated

- **Watch the repository** - Get notified of new releases
- **Star the repository** - Show your support
- **Follow discussions** - Stay informed about upcoming changes

[Unreleased]: https://github.com/codeninja55/go-radx/compare/v0.0.0...HEAD
[0.0.0]: https://github.com/codeninja55/go-radx/releases/tag/v0.0.0
