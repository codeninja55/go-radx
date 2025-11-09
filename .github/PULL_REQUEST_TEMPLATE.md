## Summary

Brief description of what this PR does and why.

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Code refactoring
- [ ] Test coverage improvement
- [ ] Dependency update

## Related Issues

- Closes #(issue number)
- Related to #(issue number)

## Changes Made

Detailed list of changes:

-
-
-

## Testing

### Test Coverage

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] All tests passing locally
- [ ] Test coverage >= 80% for new code

### Manual Testing

Describe the testing you performed:

1.
2.
3.

## Standards Compliance

If this PR relates to healthcare standards:

- [ ] FHIR R5 specification compliant
- [ ] DICOM standard compliant
- [ ] HL7 v2.x specification compliant
- [ ] DICOMweb standard compliant
- [ ] No PHI in code, tests, or examples

## Code Quality

- [ ] Code follows [Uber Go Style Guide](https://github.com/uber-go/guide)
- [ ] Code formatted with `gofmt` (or `mise fmt`)
- [ ] All linters passing (`mise lint` or `golangci-lint run`)
- [ ] No new warnings introduced
- [ ] Godoc comments added for public APIs
- [ ] Line length under 120 characters

## Documentation

- [ ] README updated (if needed)
- [ ] User guide updated (if needed)
- [ ] API documentation updated (godoc)
- [ ] CHANGELOG updated
- [ ] Examples added/updated (if applicable)

## Security & Privacy

- [ ] No hardcoded credentials or secrets
- [ ] Input validation added for external data
- [ ] No PHI logged or exposed
- [ ] Security considerations documented (if applicable)
- [ ] Ran `govulncheck` (no new vulnerabilities)

## Performance

- [ ] No performance regression
- [ ] Benchmarks added for performance-critical code (if applicable)
- [ ] Memory usage acceptable

## Breaking Changes

If this is a breaking change:

- [ ] Documented in CHANGELOG with migration guide
- [ ] Major version bump required
- [ ] Deprecation warnings added (if gradual migration)

### Migration Instructions

If breaking changes exist, provide migration instructions:

```go
// Before
// ...

// After
// ...
```

## Screenshots/Examples

If applicable, add screenshots or code examples demonstrating the changes.

## Checklist

- [ ] I have read the [CONTRIBUTING](../CONTRIBUTING.md) guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published

## Additional Context

Add any other context about the pull request here.
