# FHIR User Guide

Welcome to the FHIR (Fast Healthcare Interoperability Resources) user guide for go-radx.

## What is FHIR?

FHIR is a standard for exchanging healthcare information electronically. go-radx provides comprehensive support for FHIR R5, the latest version of the specification.

## Quick Navigation

- [Overview](overview.md) - Complete FHIR library documentation
- [Resources](resources.md) - Working with FHIR resources
- [Bundles](bundles.md) - Bundle navigation and utilities
- [Validation](validation.md) - Resource validation
- [Primitives](primitives.md) - FHIR primitive types (Date, DateTime, Time, Instant)
- [Extensions](extensions.md) - FHIR extensions support
- [Choice Types](choice-types.md) - Handling polymorphic fields
- [Summary Mode](summary-mode.md) - Payload optimization

## Key Features

### Type-Safe Resources
All 158 FHIR R5 resources are generated from official FHIR StructureDefinitions, providing compile-time safety.

### Comprehensive Validation
Built-in validation for cardinality, required fields, enums, and choice types.

### Performance Optimized
Summary mode provides 40-70% payload reduction for bandwidth-constrained scenarios.

## Getting Started

```go
import "github.com/codeninja55/go-radx/fhir/r5/resources"
import "github.com/codeninja55/go-radx/fhir/primitives"

// Create a patient
birthDate := primitives.MustDate("1974-12-25")
patient := &resources.Patient{
    ID:     stringPtr("example"),
    Active: boolPtr(true),
    Name: []resources.HumanName{
        {
            Use:    stringPtr("official"),
            Family: stringPtr("Doe"),
            Given:  []string{"John"},
        },
    },
    Gender:    stringPtr("male"),
    BirthDate: &birthDate,
}
```

See the [Overview](overview.md) for complete examples and API documentation.
