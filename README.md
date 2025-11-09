# go-radx

A comprehensive Go library for medical imaging and healthcare interoperability standards including FHIR R5, DICOM, HL7, and
DIMSE networking protocols.

[![Go Version](https://img.shields.io/badge/Go-1.25.4+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-latest-blue)](https://codeninja55.github.io/go-radx/)

## Overview

go-radx provides robust, production-ready implementations of healthcare and medical imaging standards with a focus on
type safety, performance, and developer experience. Built for radiology workflows, clinical systems integration, and
medical imaging applications.

## Features

### FHIR R5 Support

Complete implementation of HL7 FHIR Release 5 specification:

- **158 FHIR Resources** - All R5 resources generated from official HL7 StructureDefinitions
- **Type-Safe API** - Compile-time safety with Go structs matching FHIR specification exactly
- **Comprehensive Validation** - Built-in validation for:
  - Cardinality constraints (0..1, 1..1, 0..*, 1..*)
  - Required fields and mandatory elements
  - Choice type mutual exclusion (e.g., deceased[x])
  - Enum/coded value validation
  - Nested resource validation
- **Primitive Types** - Full FHIR primitive type support:
  - Date, DateTime, Time, Instant with precision handling
  - Validation on construction
  - ISO 8601 parsing and formatting
- **Primitive Extensions** - Automatic handling of extensions on primitive fields
- **Bundle Navigation** - Rich utilities for working with FHIR Bundles:
  - Resource extraction and filtering
  - Reference resolution within bundles
  - Pagination support (searchset bundles)
  - Resource counting and type detection
- **Summary Mode** - 40-70% payload reduction for bandwidth-constrained scenarios
- **JSON Serialization** - Full compatibility with FHIR JSON format

#### SMART on FHIR

OAuth2-based authorization framework for FHIR applications:

- **SMART App Launch** - Full SMART on FHIR launch framework support:
  - EHR launch flow - Launch from within EHR systems
  - Standalone launch flow - Independent application launch
  - OAuth2 authorization code flow
  - Token introspection and refresh
- **Scopes and Permissions** - Fine-grained access control:
  - Patient context scopes (`patient/*.read`, `patient/*.write`)
  - User context scopes (`user/*.read`, `user/*.write`)
  - System context scopes (`system/*.read`, `system/*.write`)
  - Launch context (`launch`, `launch/patient`, `launch/encounter`)
- **SMART Backend Services** - Backend system-to-system authorization:
  - Client credentials flow with JWT assertions
  - Bulk data access authorization
  - Asymmetric key authentication (RS384)
- **Token Management** - Comprehensive token handling:
  - Automatic token refresh
  - Token caching and storage
  - Secure token management
  - PKCE (Proof Key for Code Exchange) support
- **Context Resolution** - Automatic resolution of launch context:
  - Patient context from launch parameters
  - Encounter context for visit-based apps
  - Location and user context
- **Conformance** - Standards-compliant implementation:
  - SMART App Launch 2.0 specification
  - FHIR R5 OAuth endpoints
  - OpenID Connect integration

### DICOM Support

#### Core DICOM Features

- **DICOM File I/O** - Complete Part 10 file format support:
  - Read/write DICOM files (.dcm)
  - Parse DICOM headers and metadata
  - Extract and modify data elements
  - Support for all VR (Value Representation) types
- **Transfer Syntax Support**:
  - Explicit VR Little Endian
  - Implicit VR Little Endian
  - Deflated Explicit VR Little Endian
  - JPEG Baseline (via libjpeg-turbo)
  - JPEG 2000 (via OpenJPEG)
  - RLE Lossless
- **Data Dictionary** - Complete DICOM data dictionary with tag lookup
- **Dataset Operations** - Flexible dataset manipulation and querying

#### DIMSE Networking (DICOM Part 7 & 8)

Full implementation of DICOM Message Service Element protocol for network communication:

- **Association Management**:
  - A-ASSOCIATE request/accept/reject
  - A-RELEASE request/response
  - A-ABORT
  - Presentation context negotiation
  - Transfer syntax selection
- **DIMSE Services**:
  - C-ECHO (verification)
  - C-STORE (storage)
  - C-FIND (query)
  - C-GET (retrieval)
  - C-MOVE (move)
  - N-CREATE, N-SET, N-GET, N-DELETE, N-ACTION, N-EVENT-REPORT
- **Service Class Providers (SCP)**:
  - Storage SCP - Receive and store DICOM instances
  - Query/Retrieve SCP - Handle C-FIND and C-MOVE requests
  - Verification SCP - Respond to C-ECHO requests
  - Worklist SCP - Manage modality worklists
- **Service Class Users (SCU)**:
  - Storage SCU - Send DICOM instances to PACS/servers
  - Query/Retrieve SCU - Query and retrieve studies/series/instances
  - Verification SCU - Test connectivity with C-ECHO
  - Worklist SCU - Query modality worklists
- **Connection Pooling** - Efficient connection management for high-throughput scenarios
- **TLS Support** - Secure DICOM communication
- **Concurrent Operations** - Thread-safe implementations for parallel processing

#### Image Processing

- **Pixel Data Handling**:
  - Decode compressed transfer syntaxes
  - Support for multi-frame images
  - Photometric interpretation handling
  - Window/level adjustments
- **CGo Integration**:
  - libjpeg-turbo for fast JPEG decompression
  - OpenJPEG for JPEG 2000 support
  - Optional CGo - library works without image decompression

#### DICOMweb Support

Modern RESTful web services for DICOM:

- **WADO-RS (Web Access to DICOM Objects)**:
  - Retrieve studies, series, and instances via HTTP/HTTPS
  - Support for rendered image formats (JPEG, PNG)
  - Metadata retrieval in JSON/XML
  - Multi-part responses for bulk retrieval
  - Frame-level access for multi-frame images
- **STOW-RS (Store Over the Web)**:
  - Store DICOM instances via HTTP POST
  - Multi-part/related content type support
  - Bulk upload capabilities
  - Store response with success/failure details
- **QIDO-RS (Query based on ID for DICOM Objects)**:
  - RESTful search for studies, series, and instances
  - Query parameter support (patient name, study date, modality, etc.)
  - Fuzzy matching and wildcards
  - Pagination support
  - Response in JSON or XML format
- **DICOMweb Client Library**:
  - Type-safe Go API for all DICOMweb services
  - Connection pooling and retry logic
  - Automatic authentication (OAuth2, API keys)
  - Progress tracking for large transfers
  - Configurable timeouts and error handling
- **DICOMweb CLI Tool**:
  - Command-line interface for DICOMweb operations
  - Retrieve, store, and query commands
  - Batch operations for multiple files
  - Integration with radx command suite

### HL7 v2 Support

Implementation of HL7 Version 2.x messaging standard:

- **Message Parsing**:
  - Parse HL7 v2.x messages (ADT, ORM, ORU, etc.)
  - Support for all standard message types
  - Custom message type definitions
  - Segment extraction and field access
- **Message Generation**:
  - Build HL7 messages programmatically
  - Fluent API for message construction
  - Automatic field formatting
  - MSH segment generation
- **Validation**:
  - Segment validation
  - Required field checking
  - Data type validation
  - Cardinality enforcement
- **MLLP Protocol**:
  - Minimal Lower Layer Protocol client/server
  - Message framing and transmission
  - Acknowledgment handling (ACK/NACK)
- **Encoding Support**:
  - Standard HL7 v2 encoding (pipe delimiters)
  - Escape sequence handling
  - Character set support

### DICOM Utilities CLI

Command-line tools for DICOM operations:

```bash
# DICOM file inspection
radx dicom dump image.dcm                    # Display DICOM metadata
radx dicom tags image.dcm --tag 0010,0010    # Extract specific tags
radx dicom validate directory/               # Validate DICOM files

# Network operations
radx dicom echo AE_TITLE@host:port           # C-ECHO verification
radx dicom store *.dcm AE_TITLE@host:port    # C-STORE send images
radx dicom find --patient "Doe^John" pacs    # C-FIND query
radx dicom move --study 1.2.3.4 dest-ae pacs # C-MOVE retrieve

# DICOM server
radx dicom serve --port 11112 --ae MY_SCP    # Start SCP server
radx dicom worklist --config worklist.yaml   # Worklist SCP

# Conversion utilities
radx dicom to-fhir image.dcm                 # DICOM to FHIR ImagingStudy
radx dicom anonymize input.dcm output.dcm    # Anonymize DICOM files
radx dicom compress input.dcm --syntax jpeg  # Compress pixel data

# DICOMweb operations
radx dicomweb retrieve --study 1.2.840.113... pacs.example.com  # WADO-RS retrieve study
radx dicomweb store *.dcm https://pacs.example.com/dicomweb     # STOW-RS store
radx dicomweb search --patient "Doe^John" pacs.example.com      # QIDO-RS search
radx dicomweb metadata --series 1.2.840... pacs.example.com     # Retrieve metadata
```

### Integration & Utilities

- **DICOM to FHIR Mapping**:
  - Convert DICOM studies to FHIR ImagingStudy resources
  - Patient demographic mapping
  - Series and instance references
  - Endpoint configuration for WADO-RS
  - **DICOM SR to FHIR DiagnosticReport** - Comprehensive structured report mapping:
    - Convert DICOM Structured Reports to FHIR DiagnosticReport resources
    - Map SR content tree to FHIR Observations
    - Support for measurement reports, CAD results, key image notes
    - Preserve coded concepts and relationships
    - Handle numeric measurements with units
    - Map image references to ImagingSelection
    - Bidirectional conversion with validation
- **FHIR to DICOM**:
  - Generate DICOM from FHIR resources
  - Patient mapping to DICOM tags
  - **FHIR DiagnosticReport to DICOM SR** - Reverse mapping:
    - Convert FHIR DiagnosticReport and Observations to DICOM SR
    - Generate compliant SR content trees
    - Map coded values to DICOM terminology
    - Support for various SR templates (TID 1500, TID 1501, etc.)
    - Preserve provenance and authorship
- **HL7 to FHIR**:
  - ADT messages to Patient/Encounter resources
  - ORM/ORU to ServiceRequest/DiagnosticReport
  - Custom message mapping

## Installation

### Prerequisites

- **Go 1.25.4+** - [Download](https://go.dev/dl/)
- **CGo (optional)** - Required for JPEG/JPEG2000 image decompression

### Basic Installation

```bash
go get github.com/codeninja55/go-radx
```

### With Image Decompression Support

#### macOS

```bash
brew install jpeg-turbo openjpeg
go get github.com/codeninja55/go-radx
```

#### Linux (Ubuntu/Debian)

```bash
sudo apt-get install libjpeg-turbo8-dev libopenjp2-7-dev
go get github.com/codeninja55/go-radx
```

See the [Installation Guide](https://codeninja55.github.io/go-radx/installation/) for detailed setup instructions.

## Quick Start

### FHIR Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/validation"
)

func main() {
    // Create a patient
    birthDate := primitives.MustDate("1974-12-25")
    patient := &resources.Patient{
        ID:     stringPtr("example"),
        Active: boolPtr(true),
        Name: []resources.HumanName{
            {
                Use:    stringPtr("official"),
                Family: stringPtr("Doe"),
                Given:  []string{"John"},
            },
        },
        Gender:    stringPtr("male"),
        BirthDate: &birthDate,
    }

    // Validate the resource
    validator := validation.NewFHIRValidator()
    if err := validator.Validate(patient); err != nil {
        log.Fatal(err)
    }

    // Serialize to JSON
    data, _ := json.MarshalIndent(patient, "", "  ")
    fmt.Println(string(data))
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
```

### DICOM Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/dicom"
)

func main() {
    // Read DICOM file
    dataset, err := dicom.ReadFile("image.dcm")
    if err != nil {
        log.Fatal(err)
    }

    // Extract patient information
    patientName, _ := dataset.GetString(dicom.TagPatientName)
    studyDate, _ := dataset.GetDate(dicom.TagStudyDate)

    fmt.Printf("Patient: %s\n", patientName)
    fmt.Printf("Study Date: %s\n", studyDate)

    // Modify and save
    dataset.SetString(dicom.TagInstitutionName, "Example Hospital")
    if err := dicom.WriteFile("modified.dcm", dataset); err != nil {
        log.Fatal(err)
    }
}
```

### DIMSE Networking Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/dimse"
)

func main() {
    // Create association
    assoc, err := dimse.NewAssociation(dimse.Config{
        CallingAE:  "MY_SCU",
        CalledAE:   "PACS_SCP",
        RemoteAddr: "pacs.example.com:11112",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer assoc.Release()

    // Perform C-ECHO
    if err := assoc.CEcho(); err != nil {
        log.Fatal(err)
    }
    fmt.Println("C-ECHO successful!")

    // Send DICOM file via C-STORE
    dataset, _ := dicom.ReadFile("image.dcm")
    if err := assoc.CStore(dataset); err != nil {
        log.Fatal(err)
    }
    fmt.Println("C-STORE successful!")
}
```

### DICOMweb Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/dicomweb"
)

func main() {
    // Create DICOMweb client
    client := dicomweb.NewClient(dicomweb.Config{
        BaseURL: "https://pacs.example.com/dicomweb",
        Auth: dicomweb.OAuth2("client_id", "client_secret"),
    })

    ctx := context.Background()

    // QIDO-RS: Search for studies
    studies, err := client.SearchStudies(ctx, dicomweb.SearchParams{
        PatientName: "Doe^John",
        StudyDate:   "20240101-20241231",
        Modality:    "CT",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d studies\n", len(studies))

    // WADO-RS: Retrieve study
    for _, study := range studies {
        instances, err := client.RetrieveStudy(ctx, study.StudyInstanceUID)
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Retrieved %d instances\n", len(instances))
    }

    // STOW-RS: Store DICOM files
    files := []string{"image1.dcm", "image2.dcm"}
    response, err := client.Store(ctx, files)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Stored %d instances successfully\n", response.SuccessCount)
}
```

### HL7 Example

```go
package main

import (
    "fmt"

    "github.com/codeninja55/go-radx/hl7"
)

func main() {
    // Parse HL7 message
    msg, err := hl7.Parse([]byte(`MSH|^~\&|SENDING_APP|SENDING_FAC|...`))
    if err != nil {
        panic(err)
    }

    // Extract patient information
    pid := msg.Segment("PID")
    patientName := pid.Field(5)
    dob := pid.Field(7)

    fmt.Printf("Patient: %s\n", patientName)
    fmt.Printf("DOB: %s\n", dob)

    // Build new HL7 message
    newMsg := hl7.NewMessage("ADT", "A01")
    newMsg.AddSegment("PID").
        SetField(3, "12345").        // Patient ID
        SetField(5, "Doe^John").     // Patient Name
        SetField(7, "19741225")      // DOB

    fmt.Println(newMsg.String())
}
```

## Documentation

- **[User Guide](https://codeninja55.github.io/go-radx/user-guide/fhir/)** - Comprehensive guides and tutorials
- **[API Reference](https://pkg.go.dev/github.com/codeninja55/go-radx)** - Complete API documentation
- **[Examples](https://codeninja55.github.io/go-radx/examples/)** - Code examples and patterns
- **[Installation](https://codeninja55.github.io/go-radx/installation/)** - Setup and configuration
- **[Contributing](https://codeninja55.github.io/go-radx/development/contributing/)** - Contribution guidelines

## Project Status

go-radx is under active development. Current implementation status:

- ✅ **FHIR R5** - Complete (158 resources, validation, bundles, primitives)
- 🚧 **DICOM Core** - In progress (file I/O, data dictionary)
- 🚧 **DIMSE** - In progress (association, C-ECHO, C-STORE)
- 📋 **DICOMweb** - Planned (WADO-RS, STOW-RS, QIDO-RS client and CLI)
- 📋 **SMART on FHIR** - Planned (OAuth2, app launch, backend services)
- 📋 **HL7 v2** - Planned
- 📋 **CLI Tools** - Planned (radx command)
- 📋 **Integration** - Planned (DICOM SR↔FHIR DiagnosticReport, HL7↔FHIR)

See the [Changelog](https://codeninja55.github.io/go-radx/community/changelog/) for recent updates and
[Roadmap](https://github.com/codeninja55/go-radx/projects) for planned features.

## Architecture

go-radx is designed with modularity and performance in mind:

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

## Performance

go-radx is built for production use with performance in mind:

- **FHIR Validation**: <1ms for simple resources, 1-5ms for complex resources
- **DICOM Parsing**: ~10-50ms per file depending on size
- **DIMSE Throughput**: 100+ instances/second for C-STORE
- **Memory Efficient**: Minimal allocations, streaming where possible
- **Concurrent Safe**: Thread-safe APIs for parallel processing

Benchmarks available in each package.

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Using mise
mise test
mise test:coverage
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](https://codeninja55.github.io/go-radx/development/contributing/)
for guidelines.

### Development Setup

```bash
# Install mise (task runner)
curl https://mise.run | sh

# Clone repository
git clone https://github.com/codeninja55/go-radx.git
cd go-radx

# Install dependencies
mise install

# Run tests
mise test

# Build documentation
mise docs:serve
```

## Community & Support

- **GitHub Issues** - [Report bugs or request features](https://github.com/codeninja55/go-radx/issues)
- **GitHub Discussions** - [Ask questions and share ideas](https://github.com/codeninja55/go-radx/discussions)
- **Documentation** - [Read the docs](https://codeninja55.github.io/go-radx/)

## References

### Standards

- **FHIR** - [HL7 FHIR R5](http://hl7.org/fhir/R5/)
- **DICOM** - [DICOM Standard](https://www.dicomstandard.org/)
- **HL7 v2** - [HL7 Version 2](http://www.hl7.org/implement/standards/product_brief.cfm?product_id=185)

### Inspired By

- **pydicom** - Python DICOM library
- **pynetdicom** - Python DIMSE implementation
- **fhir.resources** - Python FHIR library
- **golang-fhir-models** - Go FHIR models
- **dcm4che** - Java DICOM toolkit

## License

go-radx is licensed under the [MIT License](LICENSE).

Copyright (c) 2025 Andru Manuel-Che

## Acknowledgments

- HL7 International for the FHIR specification
- NEMA for the DICOM standard
- The open source medical imaging community
- Contributors and users of go-radx
