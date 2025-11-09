# Security Policy

## Overview

go-radx is healthcare software that may be used in medical device applications or systems handling Protected Health
Information (PHI). We take security seriously and encourage responsible disclosure of security vulnerabilities.

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

**Note**: Once version 1.0.0 is released, we will maintain security updates for the current major version and the
previous major version for 12 months after the new major version release.

## Reporting a Vulnerability

### Private Disclosure

**DO NOT** report security vulnerabilities through public GitHub issues, discussions, or pull requests.

Instead, please report security vulnerabilities by emailing:

**Security Contact**: security@go-radx.dev *(or create a private security advisory via GitHub)*

### What to Include

Please include as much of the following information as possible:

1. **Type of vulnerability** (e.g., buffer overflow, SQL injection, cross-site scripting, PHI exposure, etc.)
2. **Full paths of affected source files**
3. **Location of affected source code** (tag/branch/commit or direct URL)
4. **Special configuration required** to reproduce the issue
5. **Step-by-step instructions** to reproduce the issue
6. **Proof-of-concept or exploit code** (if possible)
7. **Impact of the issue** including how an attacker might exploit it
8. **Affected versions**

### Healthcare-Specific Security Concerns

Please report if you discover any of the following:

- **PHI Exposure**: Protected Health Information in logs, error messages, or debug output
- **Data Integrity**: Issues that could corrupt medical imaging data or FHIR resources
- **Authentication/Authorization**: Bypass of security controls in DIMSE or DICOMweb
- **Validation**: Input validation issues that could lead to data corruption or injection attacks
- **Compliance**: HIPAA, GDPR, or medical device regulatory compliance issues

## Response Timeline

- **Initial Response**: Within 48 hours of report
- **Triage**: Within 5 business days
- **Status Updates**: Every 7 days until resolution
- **Fix Development**: Depends on severity (see below)
- **Public Disclosure**: After fix is released and users have time to update

### Severity Levels

| Severity | Response Time | Fix Timeline | Examples |
|----------|--------------|--------------|----------|
| **Critical** | < 24 hours | < 7 days | Remote code execution, PHI exposure, authentication bypass |
| **High** | < 48 hours | < 14 days | Data corruption, privilege escalation, significant DoS |
| **Medium** | < 5 days | < 30 days | Local DoS, information disclosure (non-PHI) |
| **Low** | < 10 days | < 90 days | Minor information disclosure, edge case issues |

## Security Update Process

1. **Vulnerability Confirmed** - We confirm and reproduce the issue
2. **Fix Developed** - We develop and test a fix
3. **Security Advisory** - We draft a security advisory (kept private)
4. **Release Prepared** - We prepare a patch release
5. **Coordinated Disclosure** - We coordinate disclosure with reporter
6. **Public Release** - We release the fix and publish the advisory
7. **Notification** - We notify users via GitHub Security Advisories

## Security Best Practices

### For Contributors

**Code Review**:
- All code changes require review by maintainers
- Security-sensitive changes require additional review
- No `--no-verify` commits (bypasses pre-commit security checks)

**Dependencies**:
- Keep dependencies up to date
- Review dependency security advisories
- Use `mise security:vulncheck` regularly

**Input Validation**:
- Validate all external input (DICOM files, FHIR resources, HL7 messages)
- Sanitize data before logging
- Use type-safe parsing

**Error Handling**:
- Never expose sensitive information in error messages
- Never log PHI (patient names, MRNs, DOB, etc.)
- Handle errors gracefully without leaking implementation details

### For Users

**PHI Protection**:
- Never log PHI using this library's logging facilities
- Use DICOM anonymization features before sharing test data
- Ensure proper access controls on systems using go-radx

**Network Security**:
- Use TLS for DIMSE connections when possible
- Use HTTPS for DICOMweb endpoints
- Implement proper authentication and authorization

**Validation**:
- Always validate DICOM files from untrusted sources
- Validate FHIR resources before processing
- Use conformance testing to verify standards compliance

**Updates**:
- Subscribe to GitHub Security Advisories for this repository
- Apply security updates promptly
- Test updates in non-production environments first

## Known Security Considerations

### DICOM File Parsing

DICOM files can contain complex nested structures and compressed data. When parsing DICOM files from untrusted sources:

- Files may consume excessive memory (malformed or malicious files)
- Decompression of image data may be resource-intensive
- Tag values should be validated before use

**Mitigations**:
- Implement size limits for DICOM files
- Use resource limits (memory, CPU time) when processing
- Validate tag values before processing

### DIMSE Networking

DIMSE protocol operates over TCP without built-in encryption:

- Communications are unencrypted by default
- Authentication is basic (AE Title only)
- No built-in authorization mechanism

**Mitigations**:
- Use TLS for DIMSE when possible
- Implement network-level controls (firewalls, VPNs)
- Add application-level authentication and authorization
- Validate all DIMSE messages

### DICOMweb

DICOMweb uses HTTP/HTTPS but requires proper security configuration:

- Authentication is not mandated by the standard
- Authorization is implementation-dependent
- CORS policies must be carefully configured

**Mitigations**:
- Always use HTTPS in production
- Implement OAuth2 or API key authentication
- Configure CORS policies restrictively
- Rate limit API endpoints

### FHIR Resources

FHIR resources may contain sensitive patient information:

- PHI in resource fields
- Sensitive data in extensions
- References that could leak information

**Mitigations**:
- Implement fine-grained access controls
- Use FHIR security labels
- Audit access to sensitive resources
- Validate all input data

## Vulnerability Disclosure Policy

### Our Commitments

We commit to:

- Respond promptly to security reports
- Keep reporters informed of progress
- Credit reporters in security advisories (unless they prefer anonymity)
- Work with reporters to understand and fix issues
- Coordinate public disclosure timing with reporters

### We Request That You:

- Give us reasonable time to fix issues before public disclosure
- Make a good faith effort to avoid privacy violations, data destruction, and service disruption
- Do not access or modify data beyond what is necessary to demonstrate the vulnerability
- Do not exploit the vulnerability for any purpose other than verification

### Scope

This security policy applies to:

- The go-radx library itself (all packages)
- Official documentation and examples
- Build and development tooling in this repository

This security policy does **not** apply to:

- Third-party dependencies (report to those projects directly)
- Forks of this repository (unless maintained by official maintainers)
- Applications built using go-radx (those are separate projects)

## Security Features

### Current Features

- **Input Validation**: Comprehensive validation for FHIR resources
- **Safe Parsing**: Safe parsing of DICOM files with error handling
- **No Logging of PHI**: Library does not log sensitive patient data
- **Standards Compliance**: Adherence to FHIR, DICOM, and HL7 specifications

### Planned Features

- **TLS Support**: TLS for DIMSE networking
- **OAuth2 Support**: OAuth2 for DICOMweb clients
- **DICOM Anonymization**: Built-in DICOM tag anonymization
- **FHIR Security Labels**: Support for FHIR security and privacy tags
- **Audit Logging**: Structured audit logging (without PHI)

## Compliance

### HIPAA Considerations

This library may be used in systems subject to HIPAA regulations. Users are responsible for:

- Implementing appropriate technical safeguards
- Maintaining audit trails
- Ensuring proper access controls
- Implementing encryption for data at rest and in transit

### GDPR Considerations

When processing data of EU residents:

- Implement data minimization
- Support data subject rights (access, deletion, portability)
- Maintain records of processing activities
- Ensure lawful basis for processing

### Medical Device Software

If using go-radx in medical device software:

- Maintain risk management documentation
- Implement appropriate validation and verification
- Follow relevant standards (IEC 62304, etc.)
- Maintain traceability of requirements

## Security Audits

We welcome security audits of this codebase. If you are conducting a security audit:

1. Contact us in advance to coordinate
2. Provide your timeline and scope
3. We will provide assistance as needed
4. Share findings privately before public disclosure

## Contact

- **Security Issues**: security@go-radx.dev
- **General Questions**: GitHub Discussions
- **GitHub Security Advisories**: https://github.com/codeninja55/go-radx/security/advisories

## Acknowledgments

We thank all security researchers who have responsibly disclosed vulnerabilities to us. Your contributions help make
go-radx safer for the healthcare community.

## Updates to This Policy

This security policy may be updated from time to time. Material changes will be announced via:

- GitHub Security Advisories
- Release notes
- Project README

---

**Last Updated**: 2025-01-09
**Version**: 1.0
