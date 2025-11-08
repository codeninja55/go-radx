# Contributing to go-radx

Thank you for your interest in contributing to go-radx! This guide will help you get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Documentation](#documentation)

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- **Be respectful** - Treat everyone with respect
- **Be constructive** - Provide helpful feedback
- **Be patient** - Help others learn
- **Be inclusive** - Welcome diverse perspectives

See [Community Support](../community/support.md#code-of-conduct) for full details.

## Getting Started

### Prerequisites

- **Go 1.25.4 or later** - [Install Go](https://go.dev/dl/)
- **Mise** - [Install Mise](https://mise.jdx.dev/getting-started.html)
- **Git** - Version control
- **GitHub account** - For pull requests

### Finding Something to Work On

1. **Browse issues** - Look for `good first issue` or `help wanted` labels
2. **Check discussions** - See what features are being discussed
3. **Ask** - Not sure? Open a discussion to ask!

[View Open Issues](https://github.com/codeninja55/go-radx/issues)

## Development Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub

# Clone your fork
git clone https://github.com/YOUR_USERNAME/go-radx.git
cd go-radx

# Add upstream remote
git remote add upstream https://github.com/codeninja55/go-radx.git
```

### 2. Install Dependencies

```bash
# Install Go version and tools via mise
mise install

# Verify installation
mise doctor

# Install CGo dependencies (optional, for DICOM image support)
# macOS
mise cgo:install:macos

# Linux
mise cgo:install:linux
```

### 3. Set Up Development Tools

```bash
# Install linters and tools
mise tool:install:golangci-lint
mise tool:install:govulncheck

# Verify tools
golangci-lint --version
govulncheck --version
```

### 4. Run Tests

```bash
# Run all tests
mise test

# Run tests with coverage
mise test:coverage

# Run tests verbosely
mise test:verbose
```

### 5. Verify Installation

```bash
# Build the project
mise build

# Run linter
mise lint

# Run formatter
mise fmt
```

## Making Changes

### 1. Create a Branch

```bash
# Update main branch
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/bug-description
```

### 2. Make Your Changes

- Write clean, readable code
- Follow existing code style
- Add tests for new functionality
- Update documentation

### 3. Commit Your Changes

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "Add feature: description of change"
```

**Commit Message Format**:

```
<type>: <description>

[optional body]

[optional footer]
```

**Types**:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples**:

```bash
git commit -m "feat: add FHIR Bundle pagination support"
git commit -m "fix: correct Date primitive parsing for partial dates"
git commit -m "docs: add examples for Observation resources"
```

## Testing

### Running Tests

```bash
# All tests
mise test

# Specific package
go test ./fhir/validation/...

# With coverage
mise test:coverage

# Verbose output
mise test:verbose
```

### Writing Tests

```go
package validation

import (
    "testing"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func TestValidatePatient(t *testing.T) {
    tests := []struct {
        name    string
        patient *resources.Patient
        wantErr bool
    }{
        {
            name: "valid patient",
            patient: &resources.Patient{
                Name: []resources.HumanName{
                    {Family: stringPtr("Doe")},
                },
            },
            wantErr: false,
        },
        {
            name:    "missing required name",
            patient: &resources.Patient{},
            wantErr: true,
        },
    }

    validator := NewFHIRValidator()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.Validate(tt.patient)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func stringPtr(s string) *string { return &s }
```

### Test Coverage

Aim for:

- **Unit tests** - 80%+ coverage for new code
- **Integration tests** - Major workflows
- **Edge cases** - Error handling, boundary conditions

## Submitting Changes

### 1. Push Your Branch

```bash
git push origin feature/your-feature-name
```

### 2. Create a Pull Request

1. Go to [github.com/codeninja55/go-radx](https://github.com/codeninja55/go-radx)
2. Click "Pull requests" → "New pull request"
3. Select your fork and branch
4. Fill out the PR template
5. Click "Create pull request"

### 3. PR Description

Include:

- **Description** - What changes are being made and why
- **Related issues** - Link to related issues
- **Testing** - How you tested the changes
- **Screenshots** - If applicable
- **Breaking changes** - If any

**Example**:

```markdown
## Description

Adds support for FHIR Bundle pagination with helper methods for retrieving next/previous page links.

Closes #42

## Changes

- Add `GetNextLink()` and `GetPreviousLink()` methods to Bundle helper
- Add tests for pagination scenarios
- Update Bundle documentation with pagination examples

## Testing

- Added unit tests for link extraction
- Tested with paginated search results from FHIR server
- All existing tests pass

## Breaking Changes

None
```

### 4. Review Process

1. **Automated checks** - CI runs tests and linters
2. **Code review** - Maintainers review your code
3. **Address feedback** - Make requested changes
4. **Approval** - PR gets approved
5. **Merge** - Maintainer merges your PR

## Code Style

### Go Code Style

Follow the [Uber Go Style Guide](https://github.com/uber-go/guide):

```go
// Good - clear, concise, idiomatic Go
func (v *Validator) Validate(resource interface{}) error {
    if resource == nil {
        return ErrNilResource
    }

    // Early returns for error cases
    if err := v.checkRequired(resource); err != nil {
        return fmt.Errorf("required field check failed: %w", err)
    }

    return nil
}

// Bad - unclear, non-idiomatic
func (v *Validator) Validate(resource interface{}) error {
    if resource != nil {
        err := v.checkRequired(resource)
        if err == nil {
            return nil
        } else {
            return fmt.Errorf("required field check failed: %w", err)
        }
    } else {
        return ErrNilResource
    }
}
```

### Formatting

```bash
# Format code (gofmt)
mise fmt

# Or manually
go fmt ./...
```

### Linting

```bash
# Run linter
mise lint

# Fix auto-fixable issues
golangci-lint run --fix
```

### Naming Conventions

- **Exported** - `PascalCase` for public APIs
- **Unexported** - `camelCase` for internal
- **Acronyms** - `ID`, `HTTP`, `URL` (all caps)
- **Getters** - No `Get` prefix (e.g., `Name()` not `GetName()`)

## Documentation

### Code Comments

```go
// NewValidator creates a new FHIR validator with default settings.
// The validator checks cardinality, required fields, and choice types.
func NewValidator() *Validator {
    return &Validator{
        checkCardinality: true,
        checkRequired:    true,
        checkChoiceTypes: true,
    }
}
```

### Package Documentation

```go
// Package validation provides FHIR resource validation.
//
// The validator checks resources against the FHIR specification:
//   - Required field presence
//   - Cardinality constraints
//   - Choice type mutual exclusion
//   - Enum value validity
//
// Example usage:
//
//   validator := validation.NewFHIRValidator()
//   if err := validator.Validate(patient); err != nil {
//       log.Fatal(err)
//   }
package validation
```

### Documentation Files

When adding features, update:

- **User guides** - `docs/user-guide/`
- **Examples** - `docs/examples/`
- **API reference** - `docs/api-reference/`
- **CHANGELOG** - `docs/community/changelog.md`

### Building Documentation

```bash
# Install MkDocs dependencies
mise docs:install

# Serve documentation locally
mise docs:serve

# Build documentation
mise docs:build
```

Visit [http://localhost:8000](http://localhost:8000) to view.

## Areas for Contribution

### High Priority

- **FHIR Resources** - Additional resource support
- **Validation** - Profile-specific validation
- **Tests** - Improve test coverage
- **Documentation** - More examples and guides

### Medium Priority

- **DICOM** - DICOM file support
- **HL7** - HL7 v2 message support
- **Performance** - Optimization opportunities
- **Tools** - Developer tooling

### Low Priority

- **Examples** - More code examples
- **Benchmarks** - Performance benchmarks
- **Integration** - Third-party integrations

## Questions?

- **GitHub Discussions** - [Ask questions](https://github.com/codeninja55/go-radx/discussions)
- **GitHub Issues** - [Report bugs](https://github.com/codeninja55/go-radx/issues)
- **Community Support** - [Get help](../community/support.md)

## Thank You!

Your contributions make go-radx better for everyone. Thank you for taking the time to contribute!

## Resources

- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)
