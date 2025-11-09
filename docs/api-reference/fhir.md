# FHIR API reference

Complete FHIR package API documentation.

## Online documentation

The complete API reference is available on pkg.go.dev:

**[FHIR R5 Resources](https://pkg.go.dev/github.com/codeninja55/go-radx/fhir/r5/resources)**

## Key packages

### Resources (`fhir/r5/resources`)

All 158 FHIR R5 resource types:

- Administrative resources: Patient, Practitioner, Organization, Location
- Clinical resources: Observation, Condition, Procedure, DiagnosticReport
- Diagnostic resources: ImagingStudy, ServiceRequest
- Workflow resources: Encounter, Appointment, Task

### Primitives (`fhir/primitives`)

FHIR primitive types with validation:

- `Date` - FHIR date with partial precision support
- `DateTime` - FHIR dateTime with timezone support
- `Time` - 24-hour time format
- `Instant` - Precise timestamp with timezone

### Validation (`fhir/validation`)

Resource validation framework:

- `FHIRValidator` - Main validator type
- Cardinality validation (0..1, 1..1, 0..*, 1..*)
- Required field validation
- Choice type validation
- Enum validation

### Bundle (`fhir/bundle`)

Bundle utilities:

- Resource extraction and filtering
- Reference resolution
- Pagination support
- Type detection

## Example usage

```go
package main

import (
    "encoding/json"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/validation"
)

func main() {
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
        BirthDate: &birthDate,
    }

    // Validate
    validator := validation.NewFHIRValidator()
    if err := validator.Validate(patient); err != nil {
        log.Fatal(err)
    }

    // Serialize
    data, _ := json.MarshalIndent(patient, "", "  ")
    println(string(data))
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool { return &b }
```

## See also

- [FHIR Overview](../user-guide/fhir/overview.md)
- [FHIR Examples](../examples/fhir-examples.md)
- [Installation Guide](../installation/index.md)
