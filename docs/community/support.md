# Community Support

Need help with go-radx? Here's how to get support and connect with the community.

## Getting Help

### Documentation

Start with our comprehensive documentation:

- **[Installation Guide](../installation/index.md)** - Setup and prerequisites
- **[Quick Start](../installation/quickstart.md)** - Get started quickly
- **[FHIR User Guide](../user-guide/fhir/index.md)** - FHIR concepts and APIs
- **[Examples](../examples/fhir-examples.md)** - Code examples
- **[Troubleshooting](../installation/troubleshooting.md)** - Common issues

### GitHub Issues

For bug reports, feature requests, and technical questions:

1. **Search existing issues** - Your question may already be answered
2. **Create a new issue** - Use the appropriate template
3. **Provide details** - Include version, OS, code samples, and error messages

[Open an Issue](https://github.com/codeninja55/go-radx/issues/new)

### GitHub Discussions

For general questions, ideas, and community discussions:

- **Q&A** - Ask questions and help others
- **Ideas** - Propose new features
- **Show and Tell** - Share what you've built
- **General** - Community chat

[Join Discussions](https://github.com/codeninja55/go-radx/discussions)

## Contributing

We welcome contributions! Here's how you can help:

### Code Contributions

- **Bug fixes** - Fix issues and improve stability
- **Features** - Add new capabilities
- **Tests** - Improve test coverage
- **Documentation** - Enhance docs and examples

See our [Contributing Guide](../development/contributing.md) for details.

### Documentation

- Fix typos and errors
- Clarify explanations
- Add examples
- Improve tutorials

### Community Help

- Answer questions in GitHub Discussions
- Help with issue triage
- Share your use cases and experiences

## Communication Channels

### Primary Channels

- **GitHub Issues** - Bug reports, feature requests
  - [github.com/codeninja55/go-radx/issues](https://github.com/codeninja55/go-radx/issues)

- **GitHub Discussions** - Questions, ideas, community
  - [github.com/codeninja55/go-radx/discussions](https://github.com/codeninja55/go-radx/discussions)

### Social Media

Stay updated with project news:

- **GitHub** - Watch the repository for updates
  - [github.com/codeninja55/go-radx](https://github.com/codeninja55/go-radx)

## Reporting Issues

### Bug Reports

When reporting a bug, include:

1. **Go version** - `go version`
2. **OS and version** - macOS, Linux, Windows
3. **go-radx version** - Module version or commit
4. **Steps to reproduce** - Minimal code example
5. **Expected behavior** - What should happen
6. **Actual behavior** - What actually happens
7. **Error messages** - Complete stack trace

**Example**:

```
**Go Version**: go1.25.4 darwin/arm64
**OS**: macOS 15.1.1
**go-radx version**: v0.1.0

**Steps to Reproduce**:
1. Create patient with birth date
2. Validate patient
3. Marshal to JSON

**Expected**: Validation succeeds
**Actual**: Validation fails with "invalid date format"

**Error**:
```
validation error: birth date invalid
```

**Code**:
```go
birthDate := primitives.MustDate("01/15/2024")
patient := &resources.Patient{BirthDate: &birthDate}
```
```

### Feature Requests

When requesting a feature, include:

1. **Use case** - What problem does this solve?
2. **Proposed solution** - How should it work?
3. **Alternatives** - Other approaches considered
4. **Additional context** - Links, examples, references

### Security Issues

For security vulnerabilities, **DO NOT** open a public issue.

Instead, email security concerns to the maintainers privately or use GitHub's security advisory feature.

[Report a Security Vulnerability](https://github.com/codeninja55/go-radx/security/advisories/new)

## Code of Conduct

We are committed to providing a welcoming and inclusive community.

### Our Standards

- **Be respectful** - Treat everyone with respect
- **Be constructive** - Provide helpful feedback
- **Be patient** - Help others learn
- **Be inclusive** - Welcome diverse perspectives

### Unacceptable Behavior

- Harassment, discrimination, or hate speech
- Trolling, insulting, or derogatory comments
- Personal attacks
- Publishing others' private information

### Enforcement

Violations may result in temporary or permanent bans from the community.

Report violations to the project maintainers.

## Response Times

While we strive to respond quickly, please be patient:

- **Critical bugs** - Within 1-3 days
- **Bug reports** - Within 1 week
- **Feature requests** - Within 2 weeks
- **Questions** - Community-driven, varies

Response times are not guaranteed and depend on maintainer availability.

## Commercial Support

For commercial support, training, or consulting:

- Contact the project maintainers
- Discuss your requirements
- Explore partnership opportunities

## Acknowledgments

Thank you to all contributors, users, and supporters of go-radx!

### Contributors

See our [Contributors](https://github.com/codeninja55/go-radx/graphs/contributors) page.

### Sponsors

Interested in sponsoring go-radx development? Contact the maintainers.

## Resources

### External Resources

- **FHIR Specification** - [hl7.org/fhir/R5](http://hl7.org/fhir/R5/)
- **DICOM Standard** - [dicomstandard.org](https://www.dicomstandard.org/)
- **Go Documentation** - [go.dev/doc](https://go.dev/doc/)

### Related Projects

- **pydicom** - Python DICOM library
- **pynetdicom** - Python DICOM networking
- **fhir.resources** - Python FHIR library
- **golang-fhir-models** - Go FHIR models

## Stay Updated

- **Watch the repository** - Get notified of releases
- **Star the repository** - Show your support
- **Follow discussions** - Stay informed

## Next Steps

- [Start Contributing](../development/contributing.md)
- [View Changelog](changelog.md)
- [Read License](license.md)
