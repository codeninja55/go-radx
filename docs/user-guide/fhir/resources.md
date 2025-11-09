# Working with FHIR Resources

This guide covers how to work with FHIR resources in go-radx.

## Resource Basics

All FHIR resources inherit from a common base structure:

```go
type Resource struct {
    ResourceType string  `json:"resourceType"`
    ID           *string `json:"id,omitempty"`
    Meta         *Meta   `json:"meta,omitempty"`
    Language     *string `json:"language,omitempty"`
}
```

## Creating Resources

### Patient Resource

```go
import (
    "github.com/codeninja55/go-radx/fhir/r5/resources"
    "github.com/codeninja55/go-radx/fhir/primitives"
)

birthDate := primitives.MustDate("1990-05-15")
patient := &resources.Patient{
    ID:     stringPtr("patient-123"),
    Active: boolPtr(true),
    Name: []resources.HumanName{
        {
            Use:    stringPtr("official"),
            Family: stringPtr("Smith"),
            Given:  []string{"Jane", "Marie"},
        },
    },
    Gender:    stringPtr("female"),
    BirthDate: &birthDate,
    Telecom: []resources.ContactPoint{
        {
            System: stringPtr("phone"),
            Value:  stringPtr("+1-555-0123"),
            Use:    stringPtr("mobile"),
        },
        {
            System: stringPtr("email"),
            Value:  stringPtr("jane.smith@example.com"),
        },
    },
}
```

### Observation Resource

```go
obs := &resources.Observation{
    ID:     stringPtr("obs-123"),
    Status: "final",
    Code: resources.CodeableConcept{
        Coding: []resources.Coding{
            {
                System:  stringPtr("http://loinc.org"),
                Code:    stringPtr("85354-9"),
                Display: stringPtr("Blood pressure panel"),
            },
        },
    },
    Subject: resources.Reference{
        Reference: stringPtr("Patient/patient-123"),
    },
    EffectiveDateTime: primitives.MustDateTime("2024-01-15T10:30:00Z"),
    ValueQuantity: &resources.Quantity{
        Value:  float64Ptr(120),
        Unit:   stringPtr("mmHg"),
        System: stringPtr("http://unitsofmeasure.org"),
        Code:   stringPtr("mm[Hg]"),
    },
}
```

## Reading Resources

### From JSON

```go
import "encoding/json"

jsonData := `{
  "resourceType": "Patient",
  "id": "example",
  "active": true,
  "name": [{
    "family": "Doe",
    "given": ["John"]
  }]
}`

var patient resources.Patient
if err := json.Unmarshal([]byte(jsonData), &patient); err != nil {
    log.Fatal(err)
}

// Access fields
fmt.Printf("Patient ID: %s\n", *patient.ID)
fmt.Printf("Family Name: %s\n", *patient.Name[0].Family)
```

### From File

```go
data, err := os.ReadFile("patient.json")
if err != nil {
    log.Fatal(err)
}

var patient resources.Patient
if err := json.Unmarshal(data, &patient); err != nil {
    log.Fatal(err)
}
```

## Writing Resources

### To JSON

```go
data, err := json.MarshalIndent(patient, "", "  ")
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(data))
```

### To File

```go
data, err := json.MarshalIndent(patient, "", "  ")
if err != nil {
    log.Fatal(err)
}

if err := os.WriteFile("patient.json", data, 0644); err != nil {
    log.Fatal(err)
}
```

## Resource References

FHIR uses references to link resources:

```go
// Create a reference
patientRef := resources.Reference{
    Reference: stringPtr("Patient/patient-123"),
    Display:   stringPtr("Jane Smith"),
}

// Use in another resource
obs := &resources.Observation{
    Subject: patientRef,
    // ... other fields
}
```

### Contained Resources

Resources can contain other resources inline:

```go
patient := &resources.Patient{
    ID: stringPtr("example"),
    Contained: []any{
        &resources.Organization{
            ID: stringPtr("org1"),
            Name: stringPtr("Example Clinic"),
        },
    },
    ManagingOrganization: &resources.Reference{
        Reference: stringPtr("#org1"),
    },
}
```

## Resource Metadata

All resources support metadata:

```go
patient.Meta = &resources.Meta{
    VersionId:  stringPtr("1"),
    LastUpdated: primitives.MustInstant("2024-01-15T10:30:00Z"),
    Tag: []resources.Coding{
        {
            System: stringPtr("http://example.org/tags"),
            Code:   stringPtr("test"),
        },
    },
}
```

## Helper Functions

```go
func stringPtr(s string) *string {
    return &s
}

func boolPtr(b bool) *bool {
    return &b
}

func intPtr(i int) *int {
    return &i
}

func float64Ptr(f float64) *float64 {
    return &f
}
```

## Best Practices

1. **Always validate** resources before sending to FHIR servers
2. **Use pointers** for optional fields (nil indicates absence)
3. **Check for nil** before accessing pointer fields
4. **Use primitives package** for dates, times, and other FHIR primitives
5. **Validate references** to ensure they point to valid resources

## Next Steps

- Learn about [Bundles](bundles.md) for working with collections
- See [Validation](validation.md) for ensuring resource correctness
- Explore [Primitives](primitives.md) for FHIR data types
