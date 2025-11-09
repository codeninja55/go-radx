# Project Context

## Purpose

go-radx is a comprehensive Go library for medical imaging and healthcare interoperability standards. The project aims to provide robust, production-ready implementations of FHIR R5, DICOM, HL7 v2.x, and DIMSE networking protocols with a focus on type safety, performance, and developer experience.

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

## Tech Stack

### Core Language
- **Go 1.25.4+** - Modern Go with generics and latest features
- **CGo** (optional) - For JPEG/JPEG2000 image decompression via C libraries

### Build & Development Tools
- **Mise** - Task runner and tool version manager (replaces Make)
  - Configured via `mise.toml` with 35+ development tasks
  - Manages Go, Python, and tool versions
- **uv** - Fast Python package manager for documentation tooling
- **golangci-lint** - Comprehensive Go linter
- **govulncheck** - Go vulnerability scanner

### Documentation
- **MkDocs** - Static site generator
- **Material for MkDocs** - Documentation theme
- **Python 3.14** - For documentation tooling
- **GitHub Pages** - Documentation hosting

### CGo Dependencies (Optional)
- **libjpeg-turbo** - JPEG compression/decompression
- **OpenJPEG 2.5+** - JPEG 2000 compression/decompression
- **pkg-config** - Build configuration

### Testing & Quality
- Go's built-in testing framework
- Table-driven tests
- Benchmark tests
- Coverage reporting
- GitHub Actions for CI/CD

### Version Control
- **Git** with strict pre-commit hooks
- **GitHub** for hosting and collaboration
- **gh CLI** for GitHub operations

## Project Conventions

### Code Style

**Go Style Guidelines:**
- Follow [Uber Go Style Guide](https://github.com/uber-go/guide) strictly
- Additional rules from `~/.claude/UBER_GO.md`

**Key Principles:**
- **KISS** - Keep It Simple, Stupid
- **YAGNI** - You Aren't Gonna Need It
- **SOLID** - Single Responsibility, Open-Closed, Liskov Substitution, Interface Segregation, Dependency Inversion
- **12-Factor App** - Configuration, dependencies, dev/prod parity

**Formatting:**
- Use `gofmt` (via `mise fmt`)
- Line length: 120 characters maximum
- Consistent import ordering (stdlib, third-party, local)

**Naming Conventions:**
- Exported: `PascalCase`
- Unexported: `camelCase`
- Acronyms: All caps (e.g., `ID`, `HTTP`, `URL`)
- No `Get` prefix for getters (e.g., `Name()` not `GetName()`)

**Error Handling:**
- Handle each error exactly once (either log or return, not both)
- Wrap errors with context using `fmt.Errorf` with `%w`
- Error variables start with `Err` or `err`
- Error types end with `Error` suffix

**Concurrency:**
- Never use fire-and-forget goroutines
- Always provide wait mechanism (`sync.WaitGroup` or context)
- Zero-value mutexes are valid
- Channel size should be 0 (unbuffered) or 1

**Interface Design:**
- Never use pointers to interfaces
- Verify interface compliance with zero-value assertions
- Small, focused interfaces (Interface Segregation Principle)

### Architecture Patterns

**Package Structure:**
```
go-radx/
├── fhir/          # FHIR R5 implementation
│   ├── r5/        # Generated R5 resources
│   ├── primitives/# FHIR primitive types
│   ├── validation/# Validation framework
│   └── bundle/    # Bundle utilities
├── dicom/         # DICOM core
│   ├── dataset/   # Dataset operations
│   ├── transfer/  # Transfer syntaxes
│   └── tag/       # Data dictionary
├── dimse/         # DIMSE networking
│   ├── scp/       # Service Class Providers
│   ├── scu/       # Service Class Users
│   └── pdu/       # Protocol Data Units
├── dicomweb/      # DICOMweb RESTful services
│   ├── wado/      # WADO-RS client
│   ├── stow/      # STOW-RS client
│   └── qido/      # QIDO-RS client
├── hl7/           # HL7 v2.x
│   ├── message/   # Message parsing
│   ├── segment/   # Segment handling
│   └── mllp/      # MLLP protocol
└── cmd/           # CLI utilities
    └── radx/      # Main CLI tool
```

**Design Patterns:**
- Dependency injection over global state
- Factory functions (e.g., `NewValidator()`)
- Builder pattern for complex object construction
- Strategy pattern for extensible behavior
- Interface-based abstraction for testability

**Resource Management:**
- Use `defer` for cleanup
- Exit only in `main()` or `init()`
- Avoid `panic` except for unrecoverable errors
- Prefer returning errors over panicking

### Testing Strategy

**Test Organization:**
- Unit tests in `*_test.go` files alongside code
- Integration tests in separate files or packages
- Benchmark tests for performance-critical code
- Example tests for documentation

**Test Patterns:**
- Table-driven tests for multiple test cases
- Subtests using `t.Run()` for better organization
- Test fixtures in `testdata/` directories
- Helper functions in `testdata/helpers.go`

**Coverage Goals:**
- Overall: 80%+ coverage
- Critical paths: 90%+ coverage
- New code: 80%+ coverage
- Bug fixes: Include regression test

**Test Execution:**
```bash
mise test              # Run all tests
mise test:coverage     # Run with coverage
mise test:verbose      # Run with verbose output
go test -bench=. ./... # Run benchmarks
```

**Best Practices:**
- Test behavior, not implementation
- Keep tests simple and readable
- Use descriptive test names
- Clean up resources with `defer` or `t.Cleanup()`
- Don't test external services (use mocks/stubs)

### Git Workflow

**Branching Strategy:**
- `main` - Production-ready code
- `feat/*` - Feature branches
- `fix/*` - Bug fix branches
- `docs/*` - Documentation-only changes
- `refactor/*` - Code refactoring
- `chore/*` - Maintenance tasks

**Commit Conventions:**
```
<type>: <description>

[optional body]

[optional footer]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code style changes (formatting)
- `refactor` - Code refactoring
- `test` - Adding or updating tests
- `chore` - Maintenance tasks

**Critical Rules:**
- **NEVER** use `--no-verify` flag when committing
- **NEVER** use `--no-hooks` or bypass pre-commit hooks
- **ALWAYS** run linters and tests before committing
- **ALWAYS** write descriptive commit messages
- Pre-commit hooks are mandatory quality gates

**Pull Request Process:**
1. Create feature branch from `main`
2. Make changes with descriptive commits
3. Run `mise test` and `mise lint` locally
4. Push branch and create PR
5. Address review feedback
6. Squash merge to `main` (or rebase if appropriate)

## Domain Context

### Medical Imaging Standards

**FHIR (Fast Healthcare Interoperability Resources)**
- HL7 FHIR R5 specification
- 158 resource types covering clinical, administrative, and infrastructure needs
- RESTful API design with JSON/XML serialization
- Extensibility via primitive extensions and custom extensions
- Cardinality constraints (0..1, 1..1, 0..*, 1..*)
- Choice types (polymorphic fields with `[x]` suffix)
- Reference integrity between resources
- Bundle types: document, message, transaction, collection, searchset
- Summary mode for bandwidth optimization

**SMART on FHIR**
- OAuth2-based authorization framework for FHIR applications
- EHR launch flow - Launch FHIR apps from within EHR systems
- Standalone launch flow - Independent application launch
- Scopes and permissions - Fine-grained access control (patient/*, user/*, system/*)
- SMART Backend Services - System-to-system authorization with JWT assertions
- Token management - Automatic refresh, caching, PKCE support
- Context resolution - Patient, encounter, location, and user context
- Conformance - SMART App Launch 2.0 specification, OpenID Connect integration
- Enables third-party app integration with EHR systems

**DICOM (Digital Imaging and Communications in Medicine)**
- Medical imaging standard (NEMA PS3 series)
- Part 10: File format (.dcm files)
- Part 7 & 8: DIMSE networking protocol
- Data Elements with VR (Value Representation)
- Transfer Syntaxes (compression methods)
- Service-Object Pairs (SOP Classes)
- Service Class Users (SCU) and Providers (SCP)
- Association establishment and management
- DIMSE services: C-ECHO, C-STORE, C-FIND, C-GET, C-MOVE
- Normalized services: N-CREATE, N-SET, N-GET, N-DELETE, N-ACTION, N-EVENT-REPORT

**DICOM Structured Reporting (SR)**
- DICOM Part 3 Annex C - Structured Reporting
- Content tree structure - Hierarchical organization of report content
- Content items - Containers, text, numeric measurements, coded concepts, image references
- Templates (TIDs) - Standardized report structures (TID 1500, TID 1501, etc.)
- Measurement reports - Quantitative analysis results
- CAD results - Computer-Aided Detection findings
- Key image notes - Selected images with annotations
- SR ↔ FHIR mapping - Bidirectional conversion to DiagnosticReport + Observations
- Preserves coded terminology and relationships

**HL7 v2.x (Health Level Seven)**
- Legacy messaging standard (still widely used)
- Pipe-delimited format: `|^~\&` delimiters
- Message types: ADT (admissions), ORM (orders), ORU (results), etc.
- Segments: MSH (header), PID (patient), OBX (observation), etc.
- MLLP (Minimal Lower Layer Protocol) for transport
- ACK/NACK acknowledgment messages

**DIMSE Protocol Concepts**
- Application Entity (AE) - DICOM network endpoint
- Association - Network connection between AEs
- Presentation Context - Agreement on what data can be sent
- Transfer Syntax - How data is encoded
- PDU (Protocol Data Unit) - Network message format
- SOP Class - Defines what operations are available
- SOP Instance - Specific piece of data (e.g., an image)

**DICOMweb (RESTful DICOM Services)**
- Modern web-based DICOM services using HTTP/HTTPS
- WADO-RS (Web Access to DICOM Objects) - RESTful retrieval of studies, series, instances
- STOW-RS (Store Over the Web) - RESTful storage of DICOM instances via HTTP POST
- QIDO-RS (Query based on ID for DICOM Objects) - RESTful search and query
- Advantages: firewall-friendly, uses standard HTTP, supports authentication (OAuth2)
- JSON/XML metadata responses
- Multi-part/related for bulk transfers
- Complements traditional DIMSE networking

### Healthcare Workflow Context

**Radiology Workflow:**
1. Order placed (HL7 ORM message or FHIR ServiceRequest)
2. Modality worklist query (DIMSE C-FIND)
3. Image acquisition (DICOM instance creation)
4. Image storage (DIMSE C-STORE to PACS)
5. Image viewing (WADO-RS or DIMSE C-GET/C-MOVE)
6. Report creation (FHIR DiagnosticReport)
7. Results delivery (HL7 ORU message)

**Key Integrations:**
- RIS (Radiology Information System) ↔ PACS via DIMSE
- EMR (Electronic Medical Record) ↔ RIS via HL7 v2
- Clinical systems ↔ FHIR API
- DICOM metadata → FHIR ImagingStudy conversion
- DICOM SR ↔ FHIR DiagnosticReport + Observations (bidirectional)
- HL7 ADT → FHIR Patient/Encounter conversion
- SMART on FHIR apps ↔ EHR systems via OAuth2

## Important Constraints

### Technical Constraints

**Go Version:**
- Minimum: Go 1.25.4
- Must use modern Go features (generics, errors.Is/As, any type)

**Platform Support:**
- **Primary**: macOS (ARM64), Linux (x86_64, ARM64)
- **CGo**: Optional, platform-specific build for image decompression
- **Windows**: Best-effort support (use WSL2 recommended)

**Memory & Performance:**
- Minimize allocations in hot paths
- Streaming for large files where possible
- Concurrent-safe APIs
- No global mutable state

**Backward Compatibility:**
- Follow semantic versioning (SemVer)
- Breaking changes only in major versions
- Deprecation warnings before removal
- Migration guides for breaking changes

### Regulatory & Compliance

**Medical Device Considerations:**
- Code may be used in medical device software
- Must be deterministic and testable
- Comprehensive validation required
- Audit trail for data modifications
- Error handling must be explicit and safe

**Data Privacy:**
- No logging of PHI (Protected Health Information)
- Support for DICOM anonymization
- Secure handling of patient data
- No telemetry or data collection

**Standards Compliance:**
- FHIR R5 specification conformance
- DICOM standard (NEMA PS3) conformance
- HL7 v2.x specification conformance
- Must pass conformance testing tools

### Project Constraints

**No External Runtime Dependencies:**
- Pure Go where possible
- CGo only for image codecs (optional)
- No system-specific binaries in production

**Documentation Requirements:**
- All public APIs must have godoc comments
- User guides for major features
- Code examples for common use cases
- Migration guides for version upgrades

**Security:**
- No hardcoded credentials
- Input validation for all external data
- Safe handling of untrusted input
- Regular dependency vulnerability scanning

## External Dependencies

### Go Dependencies (Managed via go.mod)
- Standard library only for core functionality
- Third-party libraries must be:
  - Well-maintained (active development)
  - Widely used (proven stability)
  - Compatible license (MIT, Apache, BSD)
  - Minimal transitive dependencies

### CGo Dependencies (Optional)
- **libjpeg-turbo** - JPEG codec (BSD-3-Clause, IJG License)
  - macOS: via Homebrew
  - Linux: via apt/yum
  - Purpose: JPEG transfer syntax support
- **OpenJPEG 2.5+** - JPEG 2000 codec (BSD-2-Clause)
  - macOS: via Homebrew
  - Linux: via apt/yum
  - Purpose: JPEG 2000 transfer syntax support

### Development Dependencies
- **golangci-lint** - Go linter (GPL-3.0)
- **govulncheck** - Vulnerability scanner (BSD-3-Clause)
- **MkDocs** - Documentation generator (BSD-2-Clause)
- **Material for MkDocs** - Documentation theme (MIT)
- **mise** - Task runner and version manager (MIT)
- **uv** - Python package manager (Apache-2.0, MIT)

### External Services (Development)
- **GitHub** - Version control and CI/CD
- **GitHub Pages** - Documentation hosting
- **pkg.go.dev** - Go package documentation
- **GitHub Actions** - Continuous integration

### Reference Implementations (Study Only)
- **pydicom** - Python DICOM library
- **pynetdicom** - Python DIMSE implementation
- **fhir.resources** - Python FHIR library
- **golang-fhir-models** - Go FHIR models (samply)
- **dcm4che** - Java DICOM toolkit
- **dicom-standard** - DICOM specification (Innolitics)

### Standards Organizations
- **HL7 International** - FHIR specification
- **NEMA** - DICOM standard
- **Health Level Seven** - HL7 v2.x specification

## Reference Documentation

### Internal Documentation
- Main README: `README.md`
- FHIR User Guide: `docs/user-guide/fhir/`
- Installation Guide: `docs/installation/`
- Contributing Guide: `docs/development/contributing.md`
- Testing Guide: `docs/development/testing.md`

### External Standards
- FHIR R5: http://hl7.org/fhir/R5/
- DICOM Standard: https://www.dicomstandard.org/
- HL7 v2.x: http://www.hl7.org/implement/standards/product_brief.cfm?product_id=185
- Uber Go Style Guide: https://github.com/uber-go/guide

## Project Roadmap

### Current Status (v0.1.0 - In Progress)
- ✅ FHIR R5 documentation and structure
- ✅ Documentation infrastructure (MkDocs, GitHub Pages)
- ✅ Development tooling (Mise, linters, testing)
- 🚧 FHIR primitives implementation
- 🚧 FHIR resources generation
- 🚧 FHIR validation framework

### Near-Term (v0.2.0 - v0.5.0)
- DICOM file I/O (Part 10)
- DICOM data dictionary
- Transfer syntax support
- DIMSE association management
- Basic DIMSE services (C-ECHO, C-STORE)

### Mid-Term (v0.6.0 - v1.0.0)
- Full DIMSE protocol implementation
- SCP/SCU implementations
- DICOMweb client library (WADO-RS, STOW-RS, QIDO-RS)
- DICOMweb CLI tool integration
- HL7 v2.x message parsing
- MLLP protocol
- CLI utilities (`radx` command)

### Long-Term (v1.1.0+)
- SMART on FHIR implementation (OAuth2, app launch, backend services)
- DICOM SR ↔ FHIR DiagnosticReport + Observations mapping
- DICOM metadata ↔ FHIR ImagingStudy conversion
- HL7 ↔ FHIR integration (ADT, ORM, ORU messages)
- Advanced image processing (segmentation, presentation states)
- Performance optimizations
- Production hardening
