# Prerequisites

This guide covers the prerequisites for installing and using go-radx.

## Core Requirements

### Go

**Minimum Version**: Go 1.25.4

go-radx uses modern Go features and requires Go 1.25.4 or later.

#### Installation

**Using mise (recommended)**:

```bash
# mise will automatically install the correct Go version
mise install
```

**Manual installation**:

Download from [go.dev/dl](https://go.dev/dl/)

#### Verification

```bash
go version
# Should output: go version go1.25.4 or later
```

## CGo Dependencies (Optional)

CGo dependencies are **only required** for DICOM image decompression. If you're only using FHIR or working with
uncompressed DICOM files, you can skip this section.

### Why CGo?

DICOM images can use various compression formats:

- **JPEG** - Lossy compression (common in radiology)
- **JPEG 2000** - Lossy or lossless compression (common in pathology)
- **RLE** - Run-length encoding (lossless)

To decompress these images, go-radx uses proven C libraries via CGo:

- **libjpeg-turbo** - Fast JPEG codec
- **OpenJPEG** - JPEG 2000 codec

### Platform-Specific Installation

#### macOS

**Using mise** (recommended):

```bash
mise cgo:install:macos
```

**Manual installation** via Homebrew:

```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install jpeg-turbo openjpeg
```

**Verification**:

```bash
# Check libjpeg-turbo
brew list jpeg-turbo

# Check OpenJPEG
brew list openjpeg
```

#### Linux (Ubuntu/Debian)

**Using mise** (recommended):

```bash
mise cgo:install:linux
```

**Manual installation** via apt:

```bash
sudo apt-get update
sudo apt-get install -y \
    libjpeg-turbo8-dev \
    libopenjp2-7-dev \
    build-essential
```

**Verification**:

```bash
# Check libjpeg-turbo
dpkg -l | grep libjpeg-turbo

# Check OpenJPEG
dpkg -l | grep libopenjp2
```

#### Linux (RHEL/CentOS/Fedora)

```bash
sudo yum install -y \
    libjpeg-turbo-devel \
    openjpeg2-devel \
    gcc \
    gcc-c++
```

#### Windows

CGo support on Windows requires additional setup:

1. **Install MinGW-w64**:
   - Download from [mingw-w64.org](https://www.mingw-w64.org/)
   - Or use [MSYS2](https://www.msys2.org/)

2. **Install dependencies** via MSYS2:

```bash
pacman -S mingw-w64-x86_64-libjpeg-turbo
pacman -S mingw-w64-x86_64-openjpeg2
```

3. **Set environment variables**:

```cmd
set CGO_ENABLED=1
set CC=gcc
```

### Verifying CGo Setup

After installing CGo dependencies, verify they're working:

```bash
# Check if CGo is enabled
go env CGO_ENABLED
# Should output: 1

# Try building with CGo
go build -tags cgo ./...
```

### Disabling CGo

If you don't need image decompression, you can disable CGo:

```bash
CGO_ENABLED=0 go build ./...
```

This will skip all CGo-dependent functionality but still allow you to:

- Work with FHIR resources
- Parse DICOM metadata
- Read uncompressed DICOM pixel data

## Development Tools (Optional)

For contributing to go-radx or running the full development workflow:

### mise

**Purpose**: Task runner and tool version manager

**Installation**:

```bash
curl https://mise.run | sh
```

**Usage**:

```bash
# See available tasks
mise tasks

# Run tests
mise test

# Run linter
mise lint

# Build documentation
mise docs:build
```

### Python 3.14

**Purpose**: Documentation generation (MkDocs)

**Installation via mise**:

```bash
# mise automatically installs Python 3.14
mise install
```

### uv

**Purpose**: Fast Python package manager

**Installation via mise**:

```bash
# mise automatically installs uv
mise install
```

### golangci-lint

**Purpose**: Go linter

**Installation**:

```bash
mise tool:install:golangci-lint
```

### govulncheck

**Purpose**: Go vulnerability checker

**Installation**:

```bash
mise tool:install:govulncheck
```

## Environment Variables

### CGO_ENABLED

Controls whether CGo is enabled:

```bash
export CGO_ENABLED=1  # Enable CGo (default)
export CGO_ENABLED=0  # Disable CGo
```

### GOFLAGS

Build flags for Go:

```bash
export GOFLAGS="-buildvcs=false"  # Disable VCS stamping
```

### PKG_CONFIG_PATH

Path to pkg-config files for CGo libraries:

```bash
# macOS (Homebrew)
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"

# Linux
export PKG_CONFIG_PATH="/usr/lib/pkgconfig"
```

## Troubleshooting

### CGo Not Working

**Error**: `gcc: command not found`

**Solution**: Install build tools

```bash
# macOS
xcode-select --install

# Ubuntu/Debian
sudo apt-get install build-essential

# RHEL/CentOS
sudo yum groupinstall "Development Tools"
```

### Library Not Found

**Error**: `library not found for -ljpeg` or similar

**Solution**: Set PKG_CONFIG_PATH or library paths

```bash
# macOS
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"
export DYLD_LIBRARY_PATH="/opt/homebrew/lib"

# Linux
export PKG_CONFIG_PATH="/usr/lib/pkgconfig"
export LD_LIBRARY_PATH="/usr/lib"
```

### Go Version Too Old

**Error**: `go: go.mod requires go >= 1.25.4`

**Solution**: Update Go

```bash
# Using mise
mise install go@1.25.4

# Or download from go.dev/dl
```

## Next Steps

- [Quick Start Guide](quickstart.md)
- [Installation Overview](index.md)
- [Troubleshooting Guide](troubleshooting.md)
