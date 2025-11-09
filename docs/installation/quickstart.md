# Quick Start

Get up and running with go-radx in minutes.

## Installation

```bash
go get github.com/codeninja55/go-radx
```

## Your First FHIR Resource

Create a simple Patient resource:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
)

func main() {
    // Create a patient
    birthDate := primitives.MustDate("1974-12-25")
    patient := &resources.Patient{
        ID:     stringPtr("example-patient"),
        Active: boolPtr(true),
        Name: []resources.HumanName{
            {
                Use:    stringPtr("official"),
                Family: stringPtr("Smith"),
                Given:  []string{"John", "Robert"},
            },
        },
        Gender:    stringPtr("male"),
        BirthDate: &birthDate,
        Telecom: []resources.ContactPoint{
            {
                System: stringPtr("phone"),
                Value:  stringPtr("+1-555-0123"),
                Use:    stringPtr("mobile"),
            },
            {
                System: stringPtr("email"),
                Value:  stringPtr("john.smith@example.com"),
            },
        },
    }

    // Serialize to JSON
    data, err := json.MarshalIndent(patient, "", "  ")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(string(data))
}

// Helper functions
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
```

**Output**:

```json
{
  "resourceType": "Patient",
  "id": "example-patient",
  "active": true,
  "name": [
    {
      "use": "official",
      "family": "Smith",
      "given": ["John", "Robert"]
    }
  ],
  "gender": "male",
  "birthDate": "1974-12-25",
  "telecom": [
    {
      "system": "phone",
      "value": "+1-555-0123",
      "use": "mobile"
    },
    {
      "system": "email",
      "value": "john.smith@example.com"
    }
  ]
}
```

## Validating Resources

Ensure your FHIR resources are valid:

```go
package main

import (
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/validation"
)

func main() {
    // Create a patient
    birthDate := primitives.MustDate("1990-05-15")
    patient := &resources.Patient{
        ID:        stringPtr("example"),
        Active:    boolPtr(true),
        BirthDate: &birthDate,
        Name: []resources.HumanName{
            {
                Use:    stringPtr("official"),
                Family: stringPtr("Doe"),
                Given:  []string{"Jane"},
            },
        },
        Gender: stringPtr("female"),
    }

    // Validate the resource
    validator := validation.NewFHIRValidator()
    if err := validator.Validate(patient); err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    fmt.Println("✓ Patient resource is valid")
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
```

## Reading FHIR from JSON

Parse FHIR resources from JSON:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

func main() {
    jsonData := `{
      "resourceType": "Patient",
      "id": "example",
      "active": true,
      "name": [{
        "use": "official",
        "family": "Doe",
        "given": ["John"]
      }],
      "gender": "male",
      "birthDate": "1974-12-25"
    }`

    var patient resources.Patient
    if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
        log.Fatal(err)
    }

    // Access fields
    fmt.Printf("Patient ID: %s\n", *patient.ID)
    fmt.Printf("Active: %v\n", *patient.Active)
    fmt.Printf("Name: %s, %s\n", *patient.Name[0].Family, patient.Name[0].Given[0])
    fmt.Printf("Gender: %s\n", *patient.Gender)
    fmt.Printf("Birth Date: %s\n", patient.BirthDate.String())
}
```

**Output**:

```
Patient ID: example
Active: true
Name: Doe, John
Gender: male
Birth Date: 1974-12-25
```

## Working with Observations

Create and validate an Observation resource:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
    "github.com/codeninja55/go-radx/fhir/validation"
)

func main() {
    // Create an observation
    obs := &resources.Observation{
        ID:     stringPtr("blood-pressure"),
        Status: "final",
        Code: resources.CodeableConcept{
            Coding: []resources.Coding{
                {
                    System:  stringPtr("http://loinc.org"),
                    Code:    stringPtr("85354-9"),
                    Display: stringPtr("Blood pressure panel"),
                },
            },
            Text: stringPtr("Blood Pressure"),
        },
        Subject: resources.Reference{
            Reference: stringPtr("Patient/example"),
            Display:   stringPtr("John Doe"),
        },
        EffectiveDateTime: primitives.MustDateTime("2024-01-15T10:30:00Z"),
        Component: []resources.ObservationComponent{
            {
                Code: resources.CodeableConcept{
                    Coding: []resources.Coding{
                        {
                            System:  stringPtr("http://loinc.org"),
                            Code:    stringPtr("8480-6"),
                            Display: stringPtr("Systolic blood pressure"),
                        },
                    },
                },
                ValueQuantity: &resources.Quantity{
                    Value:  float64Ptr(120),
                    Unit:   stringPtr("mmHg"),
                    System: stringPtr("http://unitsofmeasure.org"),
                    Code:   stringPtr("mm[Hg]"),
                },
            },
            {
                Code: resources.CodeableConcept{
                    Coding: []resources.Coding{
                        {
                            System:  stringPtr("http://loinc.org"),
                            Code:    stringPtr("8462-4"),
                            Display: stringPtr("Diastolic blood pressure"),
                        },
                    },
                },
                ValueQuantity: &resources.Quantity{
                    Value:  float64Ptr(80),
                    Unit:   stringPtr("mmHg"),
                    System: stringPtr("http://unitsofmeasure.org"),
                    Code:   stringPtr("mm[Hg]"),
                },
            },
        },
    }

    // Validate
    validator := validation.NewFHIRValidator()
    if err := validator.Validate(obs); err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    // Serialize
    data, _ := json.MarshalIndent(obs, "", "  ")
    fmt.Println(string(data))
}

func stringPtr(s string) *string    { return &s }
func float64Ptr(f float64) *float64 { return &f }
```

## Working with Bundles

Create a Bundle to group multiple resources:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
)

func main() {
    birthDate := primitives.MustDate("1974-12-25")

    bundle := &resources.Bundle{
        ID:   stringPtr("patient-bundle"),
        Type: "collection",
        Entry: []resources.BundleEntry{
            {
                FullUrl: stringPtr("urn:uuid:patient-1"),
                Resource: &resources.Patient{
                    ID:        stringPtr("patient-1"),
                    Active:    boolPtr(true),
                    BirthDate: &birthDate,
                    Name: []resources.HumanName{
                        {
                            Family: stringPtr("Smith"),
                            Given:  []string{"John"},
                        },
                    },
                },
            },
            {
                FullUrl: stringPtr("urn:uuid:patient-2"),
                Resource: &resources.Patient{
                    ID:        stringPtr("patient-2"),
                    Active:    boolPtr(true),
                    BirthDate: &birthDate,
                    Name: []resources.HumanName{
                        {
                            Family: stringPtr("Doe"),
                            Given:  []string{"Jane"},
                        },
                    },
                },
            },
        },
    }

    // Serialize
    data, err := json.MarshalIndent(bundle, "", "  ")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(string(data))
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
```

## Next Steps

Now that you've created your first FHIR resources:

1. **Learn More**: Explore the [FHIR User Guide](../user-guide/fhir/index.md)
2. **See Examples**: Check out [FHIR Examples](../examples/fhir-examples.md)
3. **Dive Deep**: Read about [Validation](../user-guide/fhir/validation.md), [Bundles](../user-guide/fhir/bundles.md),
   and [Primitives](../user-guide/fhir/primitives.md)
4. **Contribute**: See the [Contributing Guide](../development/contributing.md)

## Common Patterns

### Helper Functions

You'll often need these helper functions:

```go
func stringPtr(s string) *string       { return &s }
func boolPtr(b bool) *bool             { return &b }
func intPtr(i int) *int                { return &i }
func int64Ptr(i int64) *int64          { return &i }
func float64Ptr(f float64) *float64    { return &f }
```

### Error Handling

Always validate resources before sending to FHIR servers:

```go
validator := validation.NewFHIRValidator()
if err := validator.Validate(resource); err != nil {
    return fmt.Errorf("invalid resource: %w", err)
}
```

### Reading from Files

```go
data, err := os.ReadFile("patient.json")
if err != nil {
    return err
}

var patient resources.Patient
if err := json.Unmarshal(data, &patient); err != nil {
    return err
}
```

### Writing to Files

```go
data, err := json.MarshalIndent(patient, "", "  ")
if err != nil {
    return err
}

if err := os.WriteFile("patient.json", data, 0644); err != nil {
    return err
}
```

## Troubleshooting

### Import Errors

If you see import errors, ensure you've installed the module:

```bash
go get github.com/codeninja55/go-radx
go mod tidy
```

### Validation Errors

Check that all required fields are set:

```go
// Required fields for Patient
patient.Name = []resources.HumanName{...}  // Required
patient.Gender = stringPtr("male")          // Required for some profiles
```

### JSON Marshaling Errors

Ensure primitive types use the correct constructors:

```go
// Wrong
patient.BirthDate = "1974-12-25"  // Type error

// Correct
birthDate := primitives.MustDate("1974-12-25")
patient.BirthDate = &birthDate
```

## Getting Help

- [Troubleshooting Guide](troubleshooting.md)
- [GitHub Issues](https://github.com/codeninja55/go-radx/issues)
- [Community Support](../community/support.md)
