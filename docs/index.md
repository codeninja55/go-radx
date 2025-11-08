# go-radx

A comprehensive Go implementation of DICOM, FHIR R5, and medical imaging standards.

## Features

### FHIR R5 Support
- **Type-Safe Resources** - All 158 R5 resources generated from official FHIR StructureDefinitions
- **Comprehensive Validation** - Cardinality, required fields, enums, and choice type validation
- **Primitive Extensions** - Full support for FHIR primitive extensions with automatic serialization
- **Choice Types** - Type-safe handling of FHIR choice types with mutual exclusion
- **Summary Mode** - 40-70% payload reduction for bandwidth optimization
- **Bundle Navigation** - Rich utilities for searching, filtering, and reference resolution

### DICOM Support
- **DICOM File Parsing** - Part 10 format
- **Network Communication** - DIMSE protocol (Part 7 & 8)
- **Transfer Syntax Support** - Explicit VR, Implicit VR, Deflated
- **Data Element Handling** - Full VR support
- **Dataset Operations** - Collections and management

## Quick Links

- [Installation Guide](installation/index.md)
- [FHIR User Guide](user-guide/fhir/index.md)
- [API Reference](api-reference/index.md)
- [Examples](examples/index.md)

## Getting Started

```bash
go get github.com/codeninja55/go-radx
```

See the [Installation Guide](installation/index.md) for detailed setup instructions.

## Project Status

go-radx is under active development. See the [Changelog](community/changelog.md) for recent updates.

## License

See [License](community/license.md) for details.
