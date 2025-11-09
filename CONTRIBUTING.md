# Contributing to go-radx

Thank you for your interest in contributing to go-radx! This document provides guidelines and instructions for
contributing to this project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Reporting Bugs](#reporting-bugs)
- [Requesting Features](#requesting-features)
- [License](#license)

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please report
unacceptable behavior to the project maintainers.

## Getting Started

go-radx is a comprehensive Go library for medical imaging and healthcare interoperability standards including FHIR R5,
DICOM, HL7 v2.x, and DIMSE networking protocols.

Before contributing:

1. Read the [README](README.md) to understand the project scope
2. Review the [documentation](https://codeninja55.github.io/go-radx/)
3. Check [existing issues](https://github.com/codeninja55/go-radx/issues) for similar work
4. Join [GitHub Discussions](https://github.com/codeninja55/go-radx/discussions) to ask questions

## Development Setup

### Prerequisites

- **Go 1.25.4+** - [Download](https://go.dev/dl/)
- **mise** - Task runner and version manager ([Installation](https://mise.jdx.dev/getting-started.html))
- **Git** - Version control
- **Optional**: libjpeg-turbo and OpenJPEG for image decompression support

### Setup Instructions

1. **Fork and clone the repository**:

   ```bash
   git clone https://github.com/YOUR_USERNAME/go-radx.git
   cd go-radx
   ```

2. **Install mise and dependencies**:

   ```bash
   # Install mise (if not already installed)
   curl https://mise.run | sh

   # Install project dependencies
   mise install
   ```

3. **Verify setup**:

   ```bash
   # Run tests
   mise test

   # Run linters
   mise lint
   ```

4. **Optional: Install image codec dependencies** (for JPEG/JPEG2000 support):

   **macOS**:
   ```bash
   brew install jpeg-turbo openjpeg
   ```

   **Linux (Ubuntu/Debian)**:
   ```bash
   sudo apt-get install libjpeg-turbo8-dev libopenjp2-7-dev
   ```

## How to Contribute

### Types of Contributions

We welcome various types of contributions:

- **Bug fixes** - Fix issues in existing code
- **New features** - Implement new functionality (discuss first in an issue)
- **Documentation** - Improve or add documentation
- **Tests** - Add test coverage
- **Examples** - Add code examples and use cases
- **Performance** - Optimize existing code
- **Refactoring** - Improve code quality

### Contribution Workflow

1. **Create an issue** (for new features or significant changes)
2. **Fork the repository**
3. **Create a feature branch** from `main`
4. **Make your changes** following coding standards
5. **Write tests** for your changes
6. **Run tests and linters** locally
7. **Commit your changes** following commit guidelines
8. **Push to your fork**
9. **Create a Pull Request**

## Coding Standards

### Go Style Guide

This project strictly follows the [Uber Go Style Guide](https://github.com/uber-go/guide).

**Key principles**:

- **KISS** - Keep It Simple, Stupid
- **YAGNI** - You Aren't Gonna Need It
- **SOLID** - All five principles

### Code Style

**Naming Conventions**:

- Exported identifiers: `PascalCase`
- Unexported identifiers: `camelCase`
- Acronyms: All caps (e.g., `ID`, `HTTP`, `URL`)
- No `Get` prefix for getters (e.g., `Name()` not `GetName()`)

**Error Handling**:

- Handle each error exactly once (either log or return, not both)
- Wrap errors with context using `fmt.Errorf` with `%w`
- Error variables start with `Err` or `err`
- Error types end with `Error` suffix

**Concurrency**:

- Never use fire-and-forget goroutines
- Always provide wait mechanism (`sync.WaitGroup` or context)
- Channel size should be 0 (unbuffered) or 1

**Interfaces**:

- Never use pointers to interfaces
- Verify interface compliance with zero-value assertions
- Small, focused interfaces (Interface Segregation Principle)

**Formatting**:

- Use `gofmt` (run via `mise fmt`)
- Line length: 120 characters maximum
- Consistent import ordering (stdlib, third-party, local)

### Healthcare-Specific Considerations

**Medical Device Software**:

- Code must be deterministic and testable
- Comprehensive validation required
- Error handling must be explicit and safe
- Audit trail for data modifications

**Data Privacy**:

- **NEVER** log PHI (Protected Health Information)
- Support for DICOM anonymization
- Secure handling of patient data
- No telemetry or data collection

**Standards Compliance**:

- FHIR R5 specification conformance
- DICOM standard (NEMA PS3) conformance
- HL7 v2.x specification conformance

## Testing Requirements

### Test Coverage

- **Overall**: 80%+ coverage
- **Critical paths**: 90%+ coverage
- **New code**: 80%+ coverage minimum
- **Bug fixes**: Must include regression test

### Test Organization

- Unit tests in `*_test.go` files alongside code
- Table-driven tests for multiple test cases
- Use subtests with `t.Run()` for better organization
- Test fixtures in `testdata/` directories

### Running Tests

```bash
# Run all tests
mise test

# Run with coverage
mise test:coverage

# Run with verbose output
mise test:verbose

# Run benchmarks
go test -bench=. ./...
```

### Test Best Practices

- Test behavior, not implementation
- Keep tests simple and readable
- Use descriptive test names
- Clean up resources with `defer` or `t.Cleanup()`
- Don't test external services (use mocks/stubs)

## Commit Guidelines

### Commit Message Format

```
<type>: <description>

[optional body]

[optional footer]
```

### Commit Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code style changes (formatting, no code change)
- `refactor` - Code refactoring
- `test` - Adding or updating tests
- `chore` - Maintenance tasks (dependencies, build, etc.)

### Examples

```
feat: add DICOM SR to FHIR DiagnosticReport mapping

Implements bidirectional mapping between DICOM Structured Reports
and FHIR DiagnosticReport resources with full validation support.
```

```
fix: correct FHIR primitive extension handling

Fixed issue where primitive extensions were not being serialized
correctly in JSON output for DateTime fields.

Fixes #123
```

### Critical Rules

- **NEVER** use `--no-verify` flag when committing
- **NEVER** use `--no-hooks` or bypass pre-commit hooks
- **ALWAYS** run linters and tests before committing
- **ALWAYS** write descriptive commit messages
- Pre-commit hooks are mandatory quality gates

## Pull Request Process

### Before Submitting

1. **Update your branch** with latest `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout your-feature-branch
   git rebase main
   ```

2. **Run all checks locally**:
   ```bash
   mise test
   mise lint
   ```

3. **Update documentation** if needed

4. **Add or update tests** for your changes

### PR Guidelines

1. **Title**: Use conventional commit format (e.g., `feat: add DICOMweb WADO-RS client`)

2. **Description**: Include:
   - What changed and why
   - How to test the changes
   - Link to related issues
   - Screenshots/examples if applicable

3. **Scope**: Keep PRs focused on a single feature or fix

4. **Size**: Prefer smaller PRs (< 500 lines changed)

5. **Tests**: All PRs must include tests

6. **Documentation**: Update docs if API changes

### PR Template

```markdown
## Summary

Brief description of what this PR does.

## Changes

- List of key changes
- Another change

## Testing

How to test these changes:

1. Step one
2. Step two

## Related Issues

Closes #123
Related to #456

## Checklist

- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Code follows style guide
- [ ] All tests passing
- [ ] Linters passing
```

### Review Process

1. **Automated checks** must pass (linting, tests)
2. **Code review** by maintainers
3. **Address feedback** - make requested changes
4. **Final approval** from maintainer
5. **Merge** - Squash merge to `main` (maintainers will handle)

### Review Timeline

- Initial review: Within 3-5 business days
- Follow-up reviews: Within 2-3 business days
- Large PRs may take longer

## Reporting Bugs

### Before Reporting

1. Check [existing issues](https://github.com/codeninja55/go-radx/issues)
2. Update to latest version and verify bug still exists
3. Search [GitHub Discussions](https://github.com/codeninja55/go-radx/discussions)

### Bug Report Template

```markdown
## Bug Description

Clear description of the bug.

## Steps to Reproduce

1. Step one
2. Step two
3. See error

## Expected Behavior

What should happen.

## Actual Behavior

What actually happens.

## Environment

- go-radx version:
- Go version: (`go version`)
- OS: (macOS/Linux/Windows)
- Architecture: (amd64/arm64)

## Additional Context

Any other relevant information.

## Code Sample

```go
// Minimal code to reproduce
```
```

## Requesting Features

### Feature Request Guidelines

1. **Search first** - Check if feature already requested
2. **Describe use case** - Explain the problem you're solving
3. **Propose solution** - Suggest how it might work
4. **Consider alternatives** - What other approaches did you consider?
5. **Standards compliance** - How does it relate to FHIR/DICOM/HL7 standards?

### Feature Request Template

```markdown
## Feature Description

Brief description of the feature.

## Use Case

What problem does this solve?

## Proposed Solution

How should this work?

## Alternatives Considered

What other approaches could work?

## Standards Reference

Links to relevant FHIR/DICOM/HL7 specification sections.

## Additional Context

Any other relevant information.
```

## Development Tips

### Useful Commands

```bash
# Format code
mise fmt

# Run linters
mise lint

# Run tests with coverage
mise test:coverage

# Build documentation
mise docs:serve

# Run specific test
go test -run TestFunctionName ./path/to/package

# Run benchmarks
go test -bench=. ./...

# Check vulnerabilities
mise security:vulncheck
```

### Common Patterns

**Factory Functions**:
```go
func NewValidator() *Validator {
    return &Validator{
        // initialization
    }
}
```

**Error Wrapping**:
```go
if err != nil {
    return fmt.Errorf("read config: %w", err)
}
```

**Interface Compliance Verification**:
```go
var _ http.Handler = (*Handler)(nil)
```

## Resources

### Documentation

- [go-radx Documentation](https://codeninja55.github.io/go-radx/)
- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Effective Go](https://go.dev/doc/effective_go)

### Standards

- [FHIR R5](http://hl7.org/fhir/R5/)
- [DICOM Standard](https://www.dicomstandard.org/)
- [HL7 v2.x](http://www.hl7.org/implement/standards/product_brief.cfm?product_id=185)

### Reference Implementations

- [pydicom](https://github.com/pydicom/pydicom)
- [pynetdicom](https://github.com/pydicom/pynetdicom)
- [fhir.resources](https://github.com/nazrulworld/fhir.resources)

## License

By contributing to go-radx, you agree that your contributions will be licensed under the [MIT License](LICENSE).

## Questions?

- Join [GitHub Discussions](https://github.com/codeninja55/go-radx/discussions)
- Review [existing issues](https://github.com/codeninja55/go-radx/issues)
- Check the [documentation](https://codeninja55.github.io/go-radx/)

Thank you for contributing to go-radx! 🎉
