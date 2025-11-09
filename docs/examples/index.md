# Examples

Welcome to the go-radx examples! This section provides practical code examples for working with FHIR, DICOM, and other
medical imaging standards.

## Quick Links

- **[FHIR Examples](fhir-examples.md)** - Comprehensive FHIR resource examples

## Overview

The examples are organized by feature area and demonstrate real-world use cases:

### FHIR Examples

- **Basic Operations**
  - Creating Patient resources
  - Creating Observation resources
  - Reading from JSON files
  - Writing to JSON files

- **Bundle Operations**
  - Searching and processing results
  - Handling paginated results
  - Resolving references
  - Filtering resources

- **Complete Workflows**
  - Healthcare encounter workflow
  - Patient registration
  - Vital signs recording
  - Medication management

### Coming Soon

- **DICOM Examples** - Working with DICOM files, tags, and images
- **HL7 Examples** - HL7 message parsing and generation
- **Integration Examples** - Connecting FHIR, DICOM, and HL7

## Running Examples

All examples can be run directly with Go:

```bash
# Run directly
go run example.go

# Or build first
go build -o example example.go
./example
```

## Example Structure

Each example includes:

1. **Imports** - Required packages
2. **Helper functions** - Common utilities
3. **Main logic** - The example code
4. **Comments** - Explaining key concepts
5. **Expected output** - What you should see

## Learning Path

If you're new to go-radx, we recommend this learning path:

1. Start with **[FHIR Examples](fhir-examples.md)** - Basic resource creation
2. Read **[FHIR User Guide](../user-guide/fhir/index.md)** - Core concepts
3. Explore **[Validation](../user-guide/fhir/validation.md)** - Ensuring correctness
4. Try **[Bundles](../user-guide/fhir/bundles.md)** - Working with collections

## Contributing Examples

Have a useful example to share? We welcome contributions!

1. Create your example following the existing patterns
2. Include clear comments and documentation
3. Test the example thoroughly
4. Submit a pull request

See our [Contributing Guide](../development/contributing.md) for details.

## Getting Help

- [Troubleshooting Guide](../installation/troubleshooting.md)
- [GitHub Issues](https://github.com/codeninja55/go-radx/issues)
- [Community Support](../community/support.md)
