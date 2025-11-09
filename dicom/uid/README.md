# DICOM UID Package

This package provides comprehensive DICOM Unique Identifier (UID) definitions including all known MediaStorageSOPClass and TransferSyntaxUID enumerations.

## Features

### ✅ Complete UID Coverage

All **519 known DICOM UIDs** from DICOM Standard 2024b are available:
- **64 Transfer Syntax UIDs** - All compression formats and encodings
- **322 SOP Class UIDs** - All imaging modalities and storage classes
- Plus: Well-known SOP Instances, Coding Schemes, Service Classes, etc.

### ✅ Three Ways to Access UIDs

#### 1. Package-Level Constants (25 most common UIDs)

For convenient access to frequently-used UIDs:

```go
import "github.com/harrison-ai/go-radx/dicom/uid"

// Transfer Syntaxes
tsUID := uid.ImplicitVRLittleEndian      // 1.2.840.10008.1.2
tsUID = uid.ExplicitVRLittleEndian       // 1.2.840.10008.1.2.1
tsUID = uid.JPEGBaseline8Bit             // 1.2.840.10008.1.2.4.50
tsUID = uid.JPEG2000Lossless             // 1.2.840.10008.1.2.4.90
tsUID = uid.RLELossless                  // 1.2.840.10008.1.2.5

// SOP Classes
sopUID := uid.CTImageStorage             // 1.2.840.10008.5.1.4.1.1.2
sopUID = uid.MRImageStorage              // 1.2.840.10008.5.1.4.1.1.4
sopUID = uid.SecondaryCaptureImageStorage // 1.2.840.10008.5.1.4.1.1.7

// Query/Retrieve
sopUID = uid.StudyRootQRFind             // 1.2.840.10008.5.1.4.1.2.2.1
sopUID = uid.StudyRootQRMove             // 1.2.840.10008.5.1.4.1.2.2.2

// Verification
sopUID = uid.VerificationSOPClass        // 1.2.840.10008.1.1
```

**Available Package Constants:**

Transfer Syntaxes (13):
- `ImplicitVRLittleEndian`, `ExplicitVRLittleEndian`, `DeflatedExplicitVRLittleEndian`
- `ExplicitVRBigEndian` (retired)
- `JPEGBaseline8Bit`, `JPEGExtended12Bit`, `JPEGLossless`, `JPEGLosslessSV1`
- `JPEGLSLossless`, `JPEGLSNearLossless`
- `JPEG2000Lossless`, `JPEG2000`
- `RLELossless`

SOP Classes (11):
- `VerificationSOPClass`
- `ComputedRadiographyImageStorage`
- `CTImageStorage`, `MRImageStorage`
- `SecondaryCaptureImageStorage`
- `UltrasoundImageStorage` (in uid.go)
- `PatientRootQRFind`, `PatientRootQRMove`, `PatientRootQRGet`
- `StudyRootQRFind`, `StudyRootQRMove`, `StudyRootQRGet`
- `ModalityWorklistInformationFind`

#### 2. UID Map (All 519 UIDs with metadata)

For comprehensive access with full metadata:

```go
import "github.com/harrison-ai/go-radx/dicom/uid"

// Lookup by UID string
info, ok := uid.Lookup("1.2.840.10008.1.2")
if ok {
    fmt.Println(info.Name)     // "Implicit VR Little Endian"
    fmt.Println(info.Type)     // TypeTransferSyntax
    fmt.Println(info.Retired)  // false
    fmt.Println(info.Info)     // "Default Transfer Syntax for DICOM"
}

// Find by name
info, err := uid.FindByName("CT Image Storage")
if err == nil {
    fmt.Println(info.UID)      // "1.2.840.10008.5.1.4.1.1.2"
}

// Find all Transfer Syntaxes
allTS := uid.FindAllByType(uid.TypeTransferSyntax)
fmt.Printf("Found %d Transfer Syntaxes\n", len(allTS))  // 64

// Find all SOP Classes
allSOPClasses := uid.FindAllByType(uid.TypeSOPClass)
fmt.Printf("Found %d SOP Classes\n", len(allSOPClasses))  // 322
```

**UID Type Categories:**

```go
const (
    TypeSOPClass                        // 322 SOP Classes
    TypeMetaSOPClass                    // Meta SOP Classes
    TypeTransferSyntax                  // 64 Transfer Syntaxes
    TypeWellKnownSOPInstance            // Well-known instances
    TypeWellKnownFrameOfReference       // Frame references
    TypeCodingScheme                    // Coding schemes
    TypeApplicationContextName          // Application contexts
    TypeServiceClass                    // Service classes
    TypeDICOMUIDsAsACodingScheme       // DICOM UIDs as coding
    TypeLDAPOID                         // LDAP OIDs
    TypeSynchronizationFrameOfReference // Sync frames
    TypeApplicationHostingModel         // Hosting models
    TypeMappingResource                 // Mapping resources
)
```

#### 3. Helper Functions

```go
import "github.com/harrison-ai/go-radx/dicom/uid"

// Check if UID is a Transfer Syntax
if uid.IsTransferSyntax("1.2.840.10008.1.2") {
    fmt.Println("This is a Transfer Syntax")
}

// Check if UID is a SOP Class
if uid.IsSOPClass("1.2.840.10008.5.1.4.1.1.2") {
    fmt.Println("This is a SOP Class")
}

// Get human-readable name
name := uid.Name("1.2.840.10008.1.2")
fmt.Println(name)  // "Implicit VR Little Endian"

// Check if retired
if uid.IsRetired("1.2.840.10008.1.2.2") {
    fmt.Println("Explicit VR Big Endian is RETIRED")
}

// Get UID type
uidType := uid.GetType("1.2.840.10008.1.2")
fmt.Println(uidType)  // "Transfer Syntax"
```

## Common Transfer Syntax UIDs

### Uncompressed
- **1.2.840.10008.1.2** - Implicit VR Little Endian (default)
- **1.2.840.10008.1.2.1** - Explicit VR Little Endian (most common)
- **1.2.840.10008.1.2.1.99** - Deflated Explicit VR Little Endian
- **1.2.840.10008.1.2.2** - Explicit VR Big Endian (RETIRED)

### JPEG Compression
- **1.2.840.10008.1.2.4.50** - JPEG Baseline (Process 1) - 8-bit lossy
- **1.2.840.10008.1.2.4.51** - JPEG Extended (Process 2 & 4) - 12-bit lossy
- **1.2.840.10008.1.2.4.57** - JPEG Lossless (Process 14)
- **1.2.840.10008.1.2.4.70** - JPEG Lossless (Process 14, SV1) - DEFAULT lossless

### JPEG-LS Compression
- **1.2.840.10008.1.2.4.80** - JPEG-LS Lossless
- **1.2.840.10008.1.2.4.81** - JPEG-LS Near-Lossless

### JPEG 2000 Compression
- **1.2.840.10008.1.2.4.90** - JPEG 2000 Lossless
- **1.2.840.10008.1.2.4.91** - JPEG 2000 (lossy/lossless)
- **1.2.840.10008.1.2.4.92** - JPEG 2000 Part 2 Lossless
- **1.2.840.10008.1.2.4.93** - JPEG 2000 Part 2

### Other Compression
- **1.2.840.10008.1.2.5** - RLE Lossless

## Common SOP Class UIDs (Imaging)

### Computed Radiography (CR)
- **1.2.840.10008.5.1.4.1.1.1** - CR Image Storage

### Computed Tomography (CT)
- **1.2.840.10008.5.1.4.1.1.2** - CT Image Storage
- **1.2.840.10008.5.1.4.1.1.2.1** - Enhanced CT Image Storage

### Magnetic Resonance (MR)
- **1.2.840.10008.5.1.4.1.1.4** - MR Image Storage
- **1.2.840.10008.5.1.4.1.1.4.1** - Enhanced MR Image Storage

### Ultrasound (US)
- **1.2.840.10008.5.1.4.1.1.3.1** - Ultrasound Multi-frame Image Storage
- **1.2.840.10008.5.1.4.1.1.6.1** - Ultrasound Image Storage

### Nuclear Medicine & PET
- **1.2.840.10008.5.1.4.1.1.20** - Nuclear Medicine Image Storage
- **1.2.840.10008.5.1.4.1.1.128** - PET Image Storage
- **1.2.840.10008.5.1.4.1.1.130** - Enhanced PET Image Storage

### X-Ray (Digital)
- **1.2.840.10008.5.1.4.1.1.1.1** - Digital X-Ray - For Presentation
- **1.2.840.10008.5.1.4.1.1.1.1.1** - Digital X-Ray - For Processing
- **1.2.840.10008.5.1.4.1.1.1.2** - Digital Mammography - For Presentation

### X-Ray Angiography & Fluoroscopy
- **1.2.840.10008.5.1.4.1.1.12.1** - X-Ray Angiographic Image Storage
- **1.2.840.10008.5.1.4.1.1.12.2** - X-Ray Radiofluoroscopic Image Storage

### Secondary Capture
- **1.2.840.10008.5.1.4.1.1.7** - Secondary Capture Image Storage

## Common SOP Class UIDs (Non-Imaging)

### Verification
- **1.2.840.10008.1.1** - Verification SOP Class (C-ECHO)

### Query/Retrieve
- **1.2.840.10008.5.1.4.1.2.1.1** - Patient Root QR - FIND
- **1.2.840.10008.5.1.4.1.2.1.2** - Patient Root QR - MOVE
- **1.2.840.10008.5.1.4.1.2.1.3** - Patient Root QR - GET
- **1.2.840.10008.5.1.4.1.2.2.1** - Study Root QR - FIND
- **1.2.840.10008.5.1.4.1.2.2.2** - Study Root QR - MOVE
- **1.2.840.10008.5.1.4.1.2.2.3** - Study Root QR - GET

### Workflow Management
- **1.2.840.10008.5.1.4.31** - Modality Worklist - FIND
- **1.2.840.10008.3.1.2.3.3** - Modality Performed Procedure Step

## Data Source

All UIDs are auto-generated from **DICOM PS3.6 Part 6 - Data Dictionary** (Standard Version 2024b).

Source: https://dicom.nema.org/medical/dicom/current/source/docbook/part06/part06.xml

## UID Validation

The package provides UID validation per DICOM Part 5 Section 9.1:

```go
import "github.com/harrison-ai/go-radx/dicom/uid"

// Check if valid UID format
if uid.IsValid("1.2.840.10008.1.2") {
    fmt.Println("Valid UID")
}

// Parse and validate
parsedUID, err := uid.Parse("1.2.840.10008.1.2")
if err != nil {
    fmt.Printf("Invalid UID: %v\n", err)
}
fmt.Println(parsedUID.String())  // "1.2.840.10008.1.2"

// MustParse for known-valid UIDs (panics on error)
knownUID := uid.MustParse("1.2.840.10008.1.2")
```

### Validation Rules

UIDs must conform to:
- Maximum length: 64 characters
- Character set: Digits (0-9) and periods (.)
- No leading or trailing periods
- No consecutive periods
- No leading zeros in components (except "0" alone)
- Minimum 2 components (e.g., "1.2")

## References

- **DICOM Standard Part 5**: https://dicom.nema.org/medical/dicom/current/output/html/part05.html
- **DICOM Standard Part 6**: https://dicom.nema.org/medical/dicom/current/output/html/part06.html
- **UID Registry**: https://dicom.nema.org/medical/dicom/current/output/html/part06.html#chapter_A
- **pydicom UID module**: https://pydicom.github.io/pydicom/stable/reference/uid.html

## Examples

### Example 1: Check Transfer Syntax

```go
import "github.com/harrison-ai/go-radx/dicom/uid"

func printTransferSyntaxInfo(tsUID string) {
    if !uid.IsTransferSyntax(tsUID) {
        fmt.Printf("%s is not a Transfer Syntax\n", tsUID)
        return
    }

    info, _ := uid.Lookup(tsUID)
    fmt.Printf("Transfer Syntax: %s\n", info.Name)
    fmt.Printf("  UID: %s\n", info.UID)
    fmt.Printf("  Info: %s\n", info.Info)
    if info.Retired {
        fmt.Println("  [RETIRED]")
    }
}

// Output for "1.2.840.10008.1.2":
// Transfer Syntax: Implicit VR Little Endian
//   UID: 1.2.840.10008.1.2
//   Info: Default Transfer Syntax for DICOM
```

### Example 2: Find All CT Storage SOP Classes

```go
import (
    "fmt"
    "strings"
    "github.com/harrison-ai/go-radx/dicom/uid"
)

func findCTStorageClasses() {
    allSOPClasses := uid.FindAllByType(uid.TypeSOPClass)

    for _, info := range allSOPClasses {
        if strings.Contains(info.Name, "CT") &&
           strings.Contains(info.Name, "Storage") {
            fmt.Printf("%s - %s\n", info.UID, info.Name)
            if info.Retired {
                fmt.Println("  [RETIRED]")
            }
        }
    }
}

// Output:
// 1.2.840.10008.5.1.4.1.1.2 - CT Image Storage
// 1.2.840.10008.5.1.4.1.1.2.1 - Enhanced CT Image Storage
// 1.2.840.10008.5.1.4.1.1.2.2 - Legacy Converted Enhanced CT Image Storage
```

### Example 3: Validate and Parse UID

```go
import "github.com/harrison-ai/go-radx/dicom/uid"

func processUID(uidStr string) error {
    // Validate format
    if !uid.IsValid(uidStr) {
        return fmt.Errorf("invalid UID format: %s", uidStr)
    }

    // Parse
    parsedUID, err := uid.Parse(uidStr)
    if err != nil {
        return err
    }

    // Lookup metadata
    info, ok := uid.Lookup(uidStr)
    if !ok {
        fmt.Printf("Unknown UID: %s\n", parsedUID.String())
        return nil
    }

    fmt.Printf("UID: %s\n", info.UID)
    fmt.Printf("Name: %s\n", info.Name)
    fmt.Printf("Type: %s\n", info.Type)

    return nil
}
```

## Summary

✅ **ALL** known MediaStorageSOPClass and TransferSyntaxUID values are available in this package
✅ **519 total UIDs** including 64 Transfer Syntaxes and 322 SOP Classes
✅ **25 commonly-used UIDs** exported as package constants for convenience
✅ **Complete metadata** available via uidMap with name, type, info, and retirement status
✅ **Helper functions** for type checking, name lookup, and validation
✅ **Auto-generated** from DICOM Standard 2024b (Part 6)

This provides equivalent functionality to pydicom's `pydicom.uid` module.