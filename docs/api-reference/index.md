# API reference

Complete API documentation for go-radx packages.

## Go package documentation

The complete Go API reference is available on pkg.go.dev:

**[pkg.go.dev/github.com/codeninja55/go-radx](https://pkg.go.dev/github.com/codeninja55/go-radx)**

## Main packages

### FHIR

- `github.com/codeninja55/go-radx/fhir/r5/resources` - All 158 FHIR R5 resource types
- `github.com/codeninja55/go-radx/fhir/primitives` - FHIR primitive types (Date, DateTime, Time, Instant)
- `github.com/codeninja55/go-radx/fhir/validation` - Resource validation framework
- `github.com/codeninja55/go-radx/fhir/bundle` - Bundle navigation and utilities

### DICOM

- `github.com/codeninja55/go-radx/dicom` - DICOM file I/O and dataset operations
- `github.com/codeninja55/go-radx/dicom/tag` - DICOM data dictionary
- `github.com/codeninja55/go-radx/dicom/uid` - UID generation and management
- `github.com/codeninja55/go-radx/dicom/vr` - Value Representation types

### DIMSE

- `github.com/codeninja55/go-radx/dimse/scu` - Service Class User (SCU) implementation
- `github.com/codeninja55/go-radx/dimse/scp` - Service Class Provider (SCP) implementation
- `github.com/codeninja55/go-radx/dimse/pdu` - Protocol Data Units
- `github.com/codeninja55/go-radx/dimse/dul` - DICOM Upper Layer service
- `github.com/codeninja55/go-radx/dimse/dimse` - DIMSE message handling

## Usage

View package documentation:

```bash
# View locally
go doc github.com/codeninja55/go-radx/fhir/r5/resources

# View specific type
go doc github.com/codeninja55/go-radx/fhir/r5/resources.Patient

# View online
open https://pkg.go.dev/github.com/codeninja55/go-radx
```
