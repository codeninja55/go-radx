# CLAUDE.md

This document provides context for AI assistants working on the `codeninja55/go-radx` project.

## Project Overview

go-radx is a comprehensive Go library for medical imaging and healthcare interoperability standards. This project provides robust, production-ready implementations of FHIR R4/R5, DICOM, DICOMweb, HL7 v2.x, and DIMSE networking protocols with a focus on type safety, performance, and developer experience.

### Purpose and Goals

**Primary Goals:**
- Provide type-safe, idiomatic Go implementations of healthcare standards
- Enable seamless integration between FHIR, DICOM, and HL7 systems
- Support radiology workflows, clinical systems integration, and medical imaging applications
- Deliver production-ready libraries with comprehensive validation and error handling
- Create developer-friendly APIs with excellent documentation and examples

**Target Users:**
- Healthcare software developers building clinical systems
- Radiology workflow automation engineers
- Medical imaging application developers
- PACS and RIS system integrators
- Healthcare interoperability specialists

### Domain Context

You must understand these healthcare standards and workflows:

**FHIR (Fast Healthcare Interoperability Resources)**
- HL7 FHIR R5 specification with 158 resource types
- RESTful API design with JSON/XML serialization
- Cardinality constraints (0..1, 1..1, 0..*, 1..*)
- Choice types (polymorphic fields with `[x]` suffix)
- Bundle types: document, message, transaction, collection, searchset
- Reference integrity between resources

**DICOM (Digital Imaging and Communications in Medicine)**
- Medical imaging standard (NEMA PS3 series)
- Part 10: File format (.dcm files)
- Part 7 & 8: DIMSE networking protocol
- Data Elements with VR (Value Representation)
- Transfer Syntaxes (compression methods)
- Service-Object Pairs (SOP Classes)

**DIMSE Protocol Concepts**
- Application Entity (AE) - DICOM network endpoint
- Association - Network connection between AEs
- Presentation Context - Agreement on what data can be sent
- SCP (Service Class Provider) - Receives DICOM services
- SCU (Service Class User) - Initiates DICOM services
- DIMSE services: C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE
- Normalized services: N-CREATE, N-SET, N-GET, N-DELETE, N-ACTION, N-EVENT-REPORT

**DICOMweb (RESTful DICOM Services)**
- Modern web-based DICOM services using HTTP/HTTPS
- WADO-RS (Web Access to DICOM Objects) - RESTful retrieval
- STOW-RS (Store Over the Web) - RESTful storage via HTTP POST
- QIDO-RS (Query based on ID) - RESTful search and query
- Firewall-friendly, standard HTTP, OAuth2 authentication support
- JSON/XML metadata responses, multi-part/related for bulk transfers
- Complements traditional DIMSE networking

**HL7 v2.x (Health Level Seven)**
- Legacy messaging standard (still widely used)
- Pipe-delimited format: `|^~\\&` delimiters
- Message types: ADT (admissions), ORM (orders), ORU (results)
- Segments: MSH (header), PID (patient), OBX (observation)
- MLLP (Minimal Lower Layer Protocol) for transport
- ACK/NACK acknowledgment messages

**Radiology Workflow Integration:**
1. Order placed (HL7 ORM message or FHIR ServiceRequest)
2. Modality worklist query (DIMSE C-FIND)
3. Image acquisition (DICOM instance creation)
4. Image storage (DIMSE C-STORE to PACS)
5. Image viewing (WADO-RS or DIMSE C-GET/C-MOVE)
6. Report creation (FHIR DiagnosticReport)
7. Results delivery (HL7 ORU message)

**Key System Integrations:**
- RIS (Radiology Information System) ↔ PACS via DIMSE
- EMR (Electronic Medical Record) ↔ RIS via HL7 v2
- Clinical systems ↔ FHIR API
- DICOM metadata → FHIR ImagingStudy conversion
- HL7 ADT → FHIR Patient/Encounter conversion

### Critical Constraints

**Medical Device Considerations:**
- Code may be used in medical device software
- Must be deterministic and testable
- Comprehensive validation required
- Audit trail for data modifications
- Error handling must be explicit and safe

**Data Privacy & Security:**
- **NEVER** log PHI (Protected Health Information) unless explicitly set by the user
- Support for DICOM anonymization
- Secure handling of patient data
- No telemetry or data collection
- Input validation for all external data

**Standards Compliance:**
- FHIR R4/R5 specification conformance (and future R6)
- DICOM standard (NEMA PS3) conformance
- HL7 v2.x specification conformance
- Must pass conformance testing tools

**Performance & Reliability:**
- Minimize allocations in hot paths
- Streaming for large files where possible
- Concurrent-safe APIs
- No global mutable state
- Memory efficient implementations

## Software Design Philosophy Principles

- See `@/.claude/SOFTWARE_DESIGN_PRINCIPLES.md`

## Golang Development Rules

### Development Environment

- **Go Version**: 1.26.x (managed via mise - see `mise.toml`)
- **Module**: `github.com/codeninja55/go-radx`

### Coding Best Practices

- **Early Returns**: Use to avoid nested conditions
- **Descriptive Names**: Use clear variable/function names (prefix handlers with "handle")
- **Constants Over Functions**: Use constants where possible
- **DRY Code**: Don't repeat yourself
- **Functional Style**: Prefer functional, immutable approaches when not verbose
- **Minimal Changes**: Only modify code related to the task at hand
- **Function Ordering**: Define composing functions before their components
- **TODO Comments**: Mark issues in existing code with "TODO:" prefix
- **Simplicity**: Prioritize simplicity and readability over clever solutions
- **Build Iteratively** Start with minimal functionality and verify it works before adding complexity
- **Run Tests**: Test your code frequently with realistic inputs and validate outputs
- **Build Test Environments**: Create testing environments for components that are challenging and difficult to validate directly
- **Functional Code**: Use functional and stateless approaches where they improve clarity
- **Clean logic**: Keep core logic clean and push implementation details to the edges
- **File Organziation**: Balance file organization with simplicity - use an appropriate number of files for the project scale

### Golang Development Style Guidelines

- See `@/.claude/UBER_GO.md` for Golang development style guidelines.

### Modernization Notes

- Use `errors.Is()` and `errors.As()` for error checking
- Replace `interface{}` with `any` type alias
- Replace type assertions with type switches where appropriate
- Use generics for type-safe operations
- Implement context cancellation handling for long operations
- Add proper docstring comments for exported functions and types
- Use `go.uber.org/zap` for structured logging
- Add linting and static analysis tools

### Testing

- See `@/.claude/TESTING.md` for testing guidelines.

## Content Strategy

- Document just enough for user success - not too much, not too little.
- Prioritize accuracy and usability of information.
- Make content evergreen when possible.
- Search for existing information before adding new content.
- Check existing patterns for consistency
- Start by making the smallest reasonable changes.
- When writing in Markdown, ensure the content does not exceed 120 characters per line.

## Writing standards

- Second-person voice ("you")
- Prerequisites at the start of procedural content.
- Test all code examples before publishing.
- Match style and formatting of existing pages.
- Include both basic and advanced use cases.
- Language tags on all code blocks.
- Relative paths for internal links.
- Use broadly applicable examples rather than overly specific business cases.
- Lead with context when helpful, - explain what something is before diving into implementation detail.
- Use sentence case for all headers ("Getting started" not "Getting Started").
- Use sentence case for code block titles ("Expanded example" not "Expanded Example")
- Prefer active voice and direct language.
- Remove unnecessary words while maintaining clarity.
- Break complex instructions into clear numbered steps.
- Make language more precise and contextual.

### Language and tone standards

- Avoid promotional language. You are a technical writing assistant, not a marketer or marketing person. Never use phrases like "breathtaking" or "exceptional value."
- Reduce conjunction overuse. Limit use of "moreover," "furthermore," "additionally," "on the other hand," and "consequently." Favour direct, clear statements.
- Avoid editorializing. Remove phrases like "it's important to note," "this article will," "in conclusion," or personal interpretations.
- No undue emphasis. Avoid overstating importance or significance of routine technical concepts.

### Technical accuracy standards

- Verify all links. Every link, both internal and external, must be tested and functional before publication.
- Maintain consistency. Use consistent terminology, formatting, and language variety throughout all documentation.
- Valid technical references. Ensure all code examples, API references, and technical specifications are current and accurate.

## Pull Requests

- Create a detailed message of what changed. Focus on the high level description of
  the problem it tries to solve, and how it is solved. Don't go into the specifics of the
  code unless it adds clarity.
- **IMPORTANT**: After creating a pull request, ALWAYS update the CHANGELOG.md file:
  - Move changes from `[Unreleased]` section to a new version section if appropriate
  - Add the PR number and link to each relevant change entry
  - Follow Keep a Changelog format: `- Description (#123)` where #123 is the PR number
  - Commit the CHANGELOG.md update with message: `docs: update CHANGELOG for PR #123`

## Documentation Format and Standards

### File Naming Conventions

- **Use lowercase with hyphens**: `fhir-r4-to-r5-migration.md`, `performance-benchmarks.md`
- **NOT uppercase with underscores**: ~~`CGO_TROUBLESHOOTING.md`~~, ~~`DOCKER_DEVELOPMENT.md`~~
- **Exception**: Project root files use UPPERCASE: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `LICENSE`

### Documentation Structure

```
docs/
├── index.md                    # Main documentation index
├── installation/               # Installation guides
│   ├── index.md
│   ├── prerequisites.md
│   ├── quickstart.md
│   └── troubleshooting.md
├── user-guide/                 # User-facing guides
│   └── fhir/                   # FHIR-specific guides
├── examples/                   # Code examples
├── development/                # Developer guides
│   ├── contributing.md
│   └── testing.md
├── community/                  # Community resources
└── [topic].md                  # Topic-specific docs at root
```

### Content Standards

#### Header Structure
- Use sentence case: "Quick start" not "Quick Start"
- Start with H1 title and brief introduction
- Use hierarchical heading structure (H1 → H2 → H3)

#### Code Examples
- Always include language tags: ` ```go`, ` ```bash`
- Provide complete, runnable examples
- Include necessary imports
- Show expected output when helpful

#### Standard Helper Functions (Go)
```go
func stringPtr(s string) *string       { return &s }
func boolPtr(b bool) *bool             { return &b }
func intPtr(i int) *int                { return &i }
func int64Ptr(i int64) *int64          { return &i }
func float64Ptr(f float64) *float64    { return &f }
```

#### Cross-References
- Use relative paths: `[FHIR User Guide](../user-guide/fhir/index.md)`
- Include "Next Steps" or "See Also" sections

### Testing Documentation

Before committing documentation:
- Build docs locally: `mise docs:build` or `mkdocs build`
- Preview docs: `mise docs:serve` or `mkdocs serve`
- Verify all links work
- Check formatting renders correctly
- Ensure no broken cross-references

## Reference documentation

- [DICOM](https://dicom.nema.org/medical/dicom/current/output/html/part03.html)
- [DICOMweb](https://www.dicomstandard.org/using/dicomweb)
- [DICOMweb Resources](https://www.dicomstandard.org/using/dicomweb/restful-structure)
- [FHIR](https://www.hl7.org/fhir/overview.html)
- [FHIR Resources](https://www.hl7.org/fhir/resourcelist.html)
- [FHIR R5](https://www.hl7.org/fhir/R5/)
- [FHIR R5 Resources](https://www.hl7.org/fhir/R5/resourcelist.html)

## Reference implementation

- [dcmtk](https://github.com/DCMTK/dcmtk)
- [pynetdicom](https://github.com/pydicom/pynetdicom)
- [pydicom](https://github.com/pydicom/pydicom)
- [hl7](https://github.com/johnpaulett/python-hl7)
- [dicom-standard](https://github.com/innolitics/dicom-standard.git)
- [dicomweb-client](https://github.com/ImagingDataCommons/dicomweb-client.git)
- [dicom-rs](https://github.com/Enet4/dicom-rs)
- [fhir.resources](https://github.com/nazrulworld/fhir.resources)
- [golang-fhir-models](https://github.com/samply/golang-fhir-models)
