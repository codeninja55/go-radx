# Troubleshooting

Common issues and solutions when installing and using go-radx.

## Installation Issues

### Go Version Too Old

**Problem**: Error message: `go: go.mod requires go >= 1.25.4`

**Solution**:

```bash
# Using mise (recommended)
mise install go@1.25.4

# Verify version
go version
```

### Module Not Found

**Problem**: `cannot find module providing package github.com/codeninja55/go-radx`

**Solution**:

```bash
# Ensure module is installed
go get github.com/codeninja55/go-radx

# Tidy dependencies
go mod tidy

# Verify go.mod
cat go.mod | grep go-radx
```

### Import Cycle Detected

**Problem**: `import cycle not allowed`

**Solution**: This usually indicates a circular dependency in your code, not in go-radx. Check your imports:

```go
// Bad - creates cycle
package a
import "myproject/b"

package b
import "myproject/a"
```

## CGo Issues

### CGo Not Enabled

**Problem**: `C compiler not found` or `CGo is disabled`

**Solution**:

```bash
# Enable CGo
export CGO_ENABLED=1

# Verify
go env CGO_ENABLED
# Should output: 1

# Install build tools if needed
# macOS
xcode-select --install

# Ubuntu/Debian
sudo apt-get install build-essential

# RHEL/CentOS
sudo yum groupinstall "Development Tools"
```

### Library Not Found - libjpeg

**Problem**: `ld: library not found for -ljpeg` or `cannot find -ljpeg`

**Solution**:

```bash
# macOS
brew install jpeg-turbo
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"

# Ubuntu/Debian
sudo apt-get install libjpeg-turbo8-dev

# RHEL/CentOS
sudo yum install libjpeg-turbo-devel
```

### Library Not Found - OpenJPEG

**Problem**: `ld: library not found for -lopenjp2` or `cannot find -lopenjp2`

**Solution**:

```bash
# macOS
brew install openjpeg
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"

# Ubuntu/Debian
sudo apt-get install libopenjp2-7-dev

# RHEL/CentOS
sudo yum install openjpeg2-devel
```

### pkg-config Not Found

**Problem**: `Package 'libjpeg' not found` or `pkg-config not found`

**Solution**:

```bash
# Install pkg-config
# macOS
brew install pkg-config

# Ubuntu/Debian
sudo apt-get install pkg-config

# RHEL/CentOS
sudo yum install pkgconfig

# Set PKG_CONFIG_PATH
# macOS
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"

# Linux
export PKG_CONFIG_PATH="/usr/lib/pkgconfig:/usr/lib/x86_64-linux-gnu/pkgconfig"
```

### GCC Not Found

**Problem**: `gcc: command not found`

**Solution**:

```bash
# macOS
xcode-select --install

# Ubuntu/Debian
sudo apt-get install gcc

# RHEL/CentOS
sudo yum install gcc

# Windows
# Install MinGW-w64 or MSYS2
```

## Runtime Issues

### Validation Errors

**Problem**: `field 'Name' is required but missing`

**Solution**: Ensure all required fields are set:

```go
// Check FHIR specification for required fields
patient := &resources.Patient{
    Name: []resources.HumanName{  // Required
        {
            Family: stringPtr("Doe"),
        },
    },
}
```

### JSON Unmarshaling Errors

**Problem**: `json: cannot unmarshal string into Go struct field`

**Solution**: Use correct primitive types:

```go
// Wrong - direct string assignment
patient.BirthDate = "1974-12-25"  // Type error

// Correct - use primitives package
birthDate := primitives.MustDate("1974-12-25")
patient.BirthDate = &birthDate
```

### Choice Type Errors

**Problem**: `choice type 'deceased' has multiple fields set`

**Solution**: Only set one field in a choice type:

```go
// Wrong - multiple fields set
patient.DeceasedBoolean = boolPtr(true)
patient.DeceasedDateTime = primitives.MustDateTime("2024-01-01T00:00:00Z")

// Correct - only one field
patient.DeceasedBoolean = boolPtr(true)
```

### Nil Pointer Dereference

**Problem**: `panic: runtime error: invalid memory address or nil pointer dereference`

**Solution**: Always check for nil before dereferencing:

```go
// Wrong - doesn't check for nil
fmt.Printf("Name: %s\n", *patient.Name[0].Family)  // Panic if nil

// Correct - check for nil
if len(patient.Name) > 0 && patient.Name[0].Family != nil {
    fmt.Printf("Name: %s\n", *patient.Name[0].Family)
}
```

### Invalid Date Format

**Problem**: `invalid date format: "01/15/2024"`

**Solution**: Use ISO 8601 format (YYYY-MM-DD):

```go
// Wrong
birthDate := primitives.MustDate("01/15/2024")  // Panic

// Correct
birthDate := primitives.MustDate("2024-01-15")
```

### Invalid DateTime Format

**Problem**: `invalid datetime format`

**Solution**: Use ISO 8601 format with timezone:

```go
// Wrong
dt := primitives.MustDateTime("2024-01-15 10:30:00")  // Panic

// Correct
dt := primitives.MustDateTime("2024-01-15T10:30:00Z")
// Or with timezone offset
dt := primitives.MustDateTime("2024-01-15T10:30:00+10:00")
```

## Build Issues

### Build Tags Not Recognized

**Problem**: CGo code not compiling even with CGo enabled

**Solution**: Use build tags:

```bash
# Build with CGo tag
go build -tags cgo ./...

# Or set in environment
export CGO_ENABLED=1
go build ./...
```

### Version Conflicts

**Problem**: `module requires go >= 1.25.4 but go is 1.21.0`

**Solution**:

```bash
# Update Go version
# Using mise
mise install go@1.25.4

# Or download from go.dev/dl

# Update go.mod
go mod edit -go=1.25.4
```

### Dependency Issues

**Problem**: `ambiguous import: found package in multiple modules`

**Solution**:

```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

## Testing Issues

### Tests Failing

**Problem**: Tests fail with validation errors

**Solution**: Check test data matches FHIR specification:

```go
func TestCreatePatient(t *testing.T) {
    patient := &resources.Patient{
        Name: []resources.HumanName{  // Required field
            {
                Family: stringPtr("Doe"),
            },
        },
    }

    validator := validation.NewFHIRValidator()
    if err := validator.Validate(patient); err != nil {
        t.Errorf("Expected valid patient: %v", err)
    }
}
```

### Test Coverage Low

**Problem**: `coverage: 0.0% of statements`

**Solution**: Ensure tests are in same package or use test files:

```bash
# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Documentation Issues

### MkDocs Build Fails

**Problem**: `mkdocs: command not found`

**Solution**:

```bash
# Install dependencies
mise docs:install

# Or manually with uv
uv pip install -r docs/requirements.txt
```

### MkDocs Serve Fails

**Problem**: `Address already in use`

**Solution**:

```bash
# Use different port
uv run mkdocs serve -a localhost:8001

# Or kill process using port 8000
lsof -ti:8000 | xargs kill -9
```

### Missing Python Dependencies

**Problem**: `ModuleNotFoundError: No module named 'mkdocs'`

**Solution**:

```bash
# Install with mise
mise docs:install

# Verify installation
uv pip list | grep mkdocs
```

## Performance Issues

### Slow Validation

**Problem**: Validation takes too long for large resources

**Solution**: Cache validators and validate selectively:

```go
// Cache validator (singleton pattern)
var validator = validation.NewFHIRValidator()

// Validate only on boundaries
func CreatePatient(p *resources.Patient) error {
    // Validate once at entry point
    if err := validator.Validate(p); err != nil {
        return err
    }
    // ... rest of logic
}
```

### Large JSON Files

**Problem**: Out of memory when parsing large bundles

**Solution**: Use streaming JSON decoder:

```go
import "encoding/json"

func LoadBundles(r io.Reader) error {
    decoder := json.NewDecoder(r)

    // Process one bundle at a time
    for decoder.More() {
        var bundle resources.Bundle
        if err := decoder.Decode(&bundle); err != nil {
            return err
        }
        // Process bundle
    }
    return nil
}
```

## Platform-Specific Issues

### macOS: Library Not Found

**Problem**: `dyld: Library not loaded`

**Solution**:

```bash
# Add Homebrew libraries to path
export DYLD_LIBRARY_PATH="/opt/homebrew/lib:$DYLD_LIBRARY_PATH"

# Or install with mise
mise cgo:install:macos
```

### Linux: Permission Denied

**Problem**: `permission denied` when installing dependencies

**Solution**:

```bash
# Use sudo for system packages
sudo apt-get install libjpeg-turbo8-dev

# Or install in user directory
export CGO_CFLAGS="-I$HOME/local/include"
export CGO_LDFLAGS="-L$HOME/local/lib"
```

### Windows: MinGW Issues

**Problem**: CGo fails on Windows

**Solution**:

```cmd
# Ensure MinGW is in PATH
set PATH=C:\msys64\mingw64\bin;%PATH%

# Set CGo environment
set CGO_ENABLED=1
set CC=gcc

# Use MSYS2 shell for building
# Or use WSL2 for better compatibility
```

## Getting More Help

If you're still experiencing issues:

1. **Check GitHub Issues**: [github.com/codeninja55/go-radx/issues](https://github.com/codeninja55/go-radx/issues)
2. **Search Discussions**: Look for similar problems in GitHub Discussions
3. **Create an Issue**: Provide:
   - Go version (`go version`)
   - OS and version
   - Complete error message
   - Minimal reproduction code
   - Steps to reproduce

4. **Community Support**: See [Community Support](../community/support.md) for more help options

## Debugging Tips

### Enable Verbose Output

```bash
# Verbose build
go build -v ./...

# Verbose test
go test -v ./...

# Show all commands
go build -x ./...
```

### Check Environment

```bash
# Go environment
go env

# CGo settings
go env CGO_ENABLED
go env CGO_CFLAGS
go env CGO_LDFLAGS
```

### Verify Dependencies

```bash
# List dependencies
go list -m all

# Check for updates
go list -u -m all

# Dependency graph
go mod graph
```

## Next Steps

- [Installation Guide](index.md)
- [Prerequisites](prerequisites.md)
- [Quick Start](quickstart.md)
- [Community Support](../community/support.md)
