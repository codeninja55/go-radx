# FHIR Resource Validation

This guide covers validating FHIR resources in go-radx.

## Overview

The validation framework ensures FHIR resources comply with the specification by checking:

- **Cardinality** - Required fields and maximum occurrences
- **Choice Types** - Mutual exclusion of polymorphic fields
- **Enumerations** - Valid coded values
- **Data Types** - Correct primitive types

## Basic Validation

```go
import "github.com/codeninja55/go-radx/fhir/validation"

// Create validator
validator := validation.NewFHIRValidator()

// Validate a resource
patient := &resources.Patient{
    ID:     stringPtr("example"),
    Active: boolPtr(true),
}

if err := validator.Validate(patient); err != nil {
    log.Printf("Validation failed: %v", err)
    return err
}

fmt.Println("Resource is valid!")
```

## Validation Errors

Validation errors provide detailed information about what failed:

```go
patient := &resources.Patient{
    // Missing required fields
}

err := validator.Validate(patient)
if err != nil {
    fmt.Printf("Validation errors: %v\n", err)
    // Output: "2 validation error(s):
    //   1. field 'Name' is required but missing
    //   2. field 'Gender' is required but missing"
}
```

## Cardinality Validation

### Required Fields (1..1 or 1..*)

```go
// Missing required status field
obs := &resources.Observation{
    ID: stringPtr("example"),
    // Status is required but missing
}

err := validator.Validate(obs)
// Error: field 'Status' is required (cardinality 1..1)
```

### Optional Fields (0..1 or 0..*)

```go
// Optional fields can be nil
patient := &resources.Patient{
    ID:     stringPtr("example"),
    Active: nil, // OK - optional field (0..1)
}
```

## Choice Type Validation

Choice types allow multiple possible types but only one can be set:

```go
// INVALID - multiple choice fields set
patient := &resources.Patient{
    DeceasedBoolean:  boolPtr(true),
    DeceasedDateTime: primitives.MustDateTime("2024-01-01T00:00:00Z"),
}

err := validator.Validate(patient)
// Error: choice type 'deceased' has multiple fields set

// VALID - only one choice field set
patient := &resources.Patient{
    DeceasedBoolean: boolPtr(true),
}
```

See [Choice Types](choice-types.md) for more details.

## Nested Resource Validation

Validation recurses into nested structures:

```go
patient := &resources.Patient{
    Name: []resources.HumanName{
        {
            // Missing required 'use' field in HumanName
            Family: stringPtr("Doe"),
        },
    },
}

err := validator.Validate(patient)
// Error includes validation of nested HumanName structure
```

## Custom Validation

You can add custom validation logic:

```go
func validatePatient(patient *resources.Patient) error {
    // Standard validation first
    validator := validation.NewFHIRValidator()
    if err := validator.Validate(patient); err != nil {
        return err
    }

    // Custom business rules
    if patient.BirthDate != nil {
        birthTime, _ := patient.BirthDate.Time()
        if birthTime.After(time.Now()) {
            return fmt.Errorf("birth date cannot be in the future")
        }
    }

    return nil
}
```

## Validation Best Practices

### 1. Validate Before Sending

```go
func createPatient(patient *resources.Patient) error {
    // Validate before sending to server
    if err := validator.Validate(patient); err != nil {
        return fmt.Errorf("invalid patient: %w", err)
    }

    // Send to FHIR server
    return sendToServer(patient)
}
```

### 2. Validate After Receiving

```go
func receivePatient(data []byte) (*resources.Patient, error) {
    var patient resources.Patient
    if err := json.Unmarshal(data, &patient); err != nil {
        return nil, err
    }

    // Validate received data
    if err := validator.Validate(&patient); err != nil {
        return nil, fmt.Errorf("received invalid patient: %w", err)
    }

    return &patient, nil
}
```

### 3. Handle Validation Errors Gracefully

```go
err := validator.Validate(patient)
if err != nil {
    // Log for debugging
    log.Printf("Validation failed: %v", err)

    // Return user-friendly message
    return fmt.Errorf("patient data is incomplete or invalid")
}
```

### 4. Use in Tests

```go
func TestCreatePatient(t *testing.T) {
    patient := createTestPatient()

    validator := validation.NewFHIRValidator()
    if err := validator.Validate(patient); err != nil {
        t.Errorf("Patient should be valid: %v", err)
    }
}
```

## Validation Coverage

The validator checks:

✅ Required fields (cardinality 1..1, 1..*)
✅ Choice type mutual exclusion
✅ Nested resource validation
✅ Primitive type validation (via primitive constructors)

Not currently checked:
- Value ranges and constraints
- CodeableConcept bindings
- Reference target types
- Invariants and constraints

## Performance

Validation uses reflection and has minimal overhead:

- Simple resources: <1ms
- Complex resources with many fields: 1-5ms
- Nested resources: 5-10ms

For high-performance scenarios, consider:
- Caching validation results
- Validating only on boundaries (API entry/exit)
- Using validation selectively in production

## Future Enhancements

Planned validation features:

- FHIRPath expression validation
- Profile-specific validation
- Terminology binding validation
- Reference integrity checking
- Custom constraint validation

## Summary

- Use `validation.NewFHIRValidator()` to create a validator
- Call `validator.Validate(resource)` to check compliance
- Validation errors provide detailed information
- Always validate before sending resources to servers
- Validation catches most FHIR specification violations

See the [Overview](overview.md) for complete API documentation.
