# Installation

This guide covers installing go-radx and its dependencies.

## Overview

go-radx is a Go library for working with medical imaging standards including FHIR R5, DICOM, and HL7. The library has
some platform-specific dependencies for image compression support.

## Quick Install

```bash
go get github.com/codeninja55/go-radx
```

## Requirements

### Go Version

- **Go 1.25.4 or later** (managed via [mise](https://mise.jdx.dev))

### CGo Dependencies (Optional)

For DICOM image decompression support, you'll need:

- **libjpeg-turbo** - JPEG compression/decompression
- **OpenJPEG** - JPEG 2000 compression/decompression

These are only required if you need to work with compressed DICOM images. FHIR and basic DICOM functionality work
without these dependencies.

See [Prerequisites](prerequisites.md) for detailed installation instructions.

## Installation Options

### Option 1: Using Mise (Recommended)

If you're developing with go-radx or contributing to the project:

```bash
# Install mise
curl https://mise.run | sh

# Install Go and dependencies
mise install

# Verify installation
mise doctor
```

### Option 2: Standard Go Install

For using go-radx as a library in your project:

```bash
go get github.com/codeninja55/go-radx
```

### Option 3: From Source

```bash
# Clone repository
git clone https://github.com/codeninja55/go-radx.git
cd go-radx

# Install dependencies (if using mise)
mise install

# Build
go build ./...

# Run tests
go test ./...
```

## Feature Support

### FHIR R5

✅ Full support - no additional dependencies required

- All 158 FHIR R5 resources
- Validation framework
- Bundle navigation
- Primitives (Date, DateTime, Time, Instant)
- Extensions support
- Summary mode

### DICOM

⚠️ Partial CGo dependencies

- **Basic DICOM**: No CGo required
  - Reading/writing DICOM files
  - Tag parsing
  - Metadata extraction

- **Image Decompression**: Requires CGo
  - JPEG compression (libjpeg-turbo)
  - JPEG 2000 compression (OpenJPEG)

### HL7

🚧 Coming soon

## Verification

After installation, verify go-radx is working:

```go
package main

import (
    "fmt"
    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
)

func main() {
    birthDate := primitives.MustDate("1990-05-15")
    patient := &resources.Patient{
        ID:        stringPtr("example"),
        Active:    boolPtr(true),
        BirthDate: &birthDate,
    }

    fmt.Printf("Created patient: %s\n", *patient.ID)
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool { return &b }
```

## Next Steps

- Read the [Quick Start](quickstart.md) guide
- Explore [FHIR Examples](../examples/fhir-examples.md)
- See [Troubleshooting](troubleshooting.md) if you encounter issues

## Platform-Specific Notes

### macOS

CGo dependencies can be installed via Homebrew:

```bash
mise cgo:install:macos
```

Or manually:

```bash
brew install jpeg-turbo openjpeg
```

### Linux

CGo dependencies can be installed via apt:

```bash
mise cgo:install:linux
```

Or manually:

```bash
sudo apt-get update
sudo apt-get install -y libjpeg-turbo8-dev libopenjp2-7-dev
```

### Windows

CGo support on Windows requires additional setup. See [Prerequisites](prerequisites.md) for details.

## Getting Help

- [Troubleshooting Guide](troubleshooting.md)
- [GitHub Issues](https://github.com/codeninja55/go-radx/issues)
- [Community Support](../community/support.md)
