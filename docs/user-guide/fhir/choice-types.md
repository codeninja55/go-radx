# FHIR Choice Types

This document explains FHIR choice types and their implementation in go-radx.

## Overview

In FHIR, **choice types** (also called polymorphic elements) are fields that can have different data types. In the FHIR specification, these are denoted with `[x]` suffix, such as `deceased[x]` or `value[x]`.

For example, `Patient.deceased[x]` can be either:
- `deceasedBoolean` - A boolean indicating if the patient is deceased
- `deceasedDateTime` - The date/time of death

**Constraint:** Only ONE of the options can be set at a time (mutual exclusion).

## Implementation

### Generated Fields

Choice types are expanded into multiple Go fields with suffixed names:

```go
type Patient struct {
    fhir.DomainResource
    
    // Choice type: deceased[x]
    DeceasedBoolean  *bool              `json:"deceasedBoolean,omitempty" fhir:"cardinality=0..1,summary,choice=deceased"`
    DeceasedDateTime *primitives.DateTime `json:"deceasedDateTime,omitempty" fhir:"cardinality=0..1,summary,choice=deceased"`
    
    // Choice type: multipleBirth[x]
    MultipleBirthBoolean *bool `json:"multipleBirthBoolean,omitempty" fhir:"cardinality=0..1,choice=multipleBirth"`
    MultipleBirthInteger *int  `json:"multipleBirthInteger,omitempty" fhir:"cardinality=0..1,choice=multipleBirth"`
    
    // ... other fields
}
```

### Field Naming Convention

- Base name (e.g., "deceased") + Type suffix (e.g., "Boolean") = `DeceasedBoolean`
- JSON serialization uses camelCase: `deceasedBoolean`, `deceasedDateTime`
- All options in a choice group have the same `choice=groupname` tag

### Validation

The validation framework enforces mutual exclusion - only one field in each choice group can be set:

```go
validator := validation.NewFHIRValidator()

patient := &Patient{
    DeceasedBoolean:  boolPtr(true),
    DeceasedDateTime: dateTimePtr("2024-01-01"),  // ERROR: Can't set both
}

err := validator.Validate(patient)
// Error: choice type 'deceased' has multiple fields set
```

## Usage Examples

### Setting a Choice Field

Only set **one** of the options:

```go
import "github.com/harrison-ai/go-radx/fhir/primitives"

// Option 1: Boolean
patient := &Patient{
    DeceasedBoolean: boolPtr(true),
}

// Option 2: DateTime
patient := &Patient{
    DeceasedDateTime: &primitives.DateTime{Time: time.Now()},
}

// INVALID: Setting both
patient := &Patient{
    DeceasedBoolean:  boolPtr(true),
    DeceasedDateTime: &primitives.DateTime{Time: time.Now()},  // Validation error!
}
```

### Reading a Choice Field

Check which option is set:

```go
if patient.DeceasedBoolean != nil {
    fmt.Printf("Patient deceased (boolean): %v\n", *patient.DeceasedBoolean)
} else if patient.DeceasedDateTime != nil {
    fmt.Printf("Patient deceased at: %v\n", patient.DeceasedDateTime)
} else {
    fmt.Println("Deceased status unknown")
}
```

### JSON Serialization

Choice fields serialize to JSON with their specific field names:

```go
// Go struct
patient := &Patient{
    ID:              stringPtr("example"),
    DeceasedBoolean: boolPtr(true),
}

// JSON output
{
  "resourceType": "Patient",
  "id": "example",
  "deceasedBoolean": true
}

// Alternative choice
patient := &Patient{
    ID:               stringPtr("example"),
    DeceasedDateTime: primitives.MustDateTime("2024-01-01T10:00:00Z"),
}

// JSON output
{
  "resourceType": "Patient",
  "id": "example",
  "deceasedDateTime": "2024-01-01T10:00:00Z"
}
```

### JSON Deserialization

The JSON decoder automatically populates the correct field:

```go
jsonData := `{
  "resourceType": "Patient",
  "id": "example",
  "deceasedDateTime": "2024-01-01T10:00:00Z"
}
`

var patient Patient
json.Unmarshal([]byte(jsonData), &patient)

// patient.DeceasedBoolean is nil
// patient.DeceasedDateTime is set
```

## Common Choice Types

### Patient Resource

- **deceased[x]**: `deceasedBoolean`, `deceasedDateTime`
- **multipleBirth[x]**: `multipleBirthBoolean`, `multipleBirthInteger`

### Observation Resource

- **value[x]**: `valueQuantity`, `valueCodeableConcept`, `valueString`, `valueBoolean`, `valueInteger`, `valueRange`, `valueRatio`, `valueSampledData`, `valueTime`, `valueDateTime`, `valuePeriod`
- **effective[x]**: `effectiveDateTime`, `effectivePeriod`, `effectiveTiming`, `effectiveInstant`

### Medication Resource

- **ingredient.item[x]**: `itemCodeableConcept`, `itemReference`

## Validation Details

### Mutual Exclusion

The validator checks that only one field in each choice group is set:

```go
validator := validation.NewFHIRValidator()

type Resource struct {
    ValueBoolean *bool   `fhir:"choice=value"`
    ValueString  *string `fhir:"choice=value"`
    ValueInteger *int    `fhir:"choice=value"`
}

// Valid: Only one set
resource := &Resource{ValueBoolean: boolPtr(true)}
validator.Validate(resource)  // No error

// Invalid: Multiple set
resource := &Resource{
    ValueBoolean: boolPtr(true),
    ValueString:  stringPtr("test"),
}
validator.Validate(resource)  // Error: choice type 'value' has multiple fields set
```

### Zero Values

Zero values (false, 0, "") are considered "set" if they're explicit pointers:

```go
falseVal := false
resource := &Resource{
    ValueBoolean: &falseVal,  // This is SET (even though false)
}
```

### Nested Structs

Validation recurses into nested structures:

```go
type Outer struct {
    Inner *Inner
}

type Inner struct {
    ValueBoolean *bool   `fhir:"choice=value"`
    ValueString  *string `fhir:"choice=value"`
}

outer := &Outer{
    Inner: &Inner{
        ValueBoolean: boolPtr(true),
        ValueString:  stringPtr("invalid"),  // Error in nested struct
    },
}

validator.Validate(outer)  // Error: choice type 'value' has multiple fields set
```

## Helper Functions

### Checking Which Choice is Set

Create helper methods to check which option is active:

```go
func (p *Patient) GetDeceasedType() string {
    if p.DeceasedBoolean != nil {
        return "boolean"
    }
    if p.DeceasedDateTime != nil {
        return "dateTime"
    }
    return ""
}

func (p *Patient) IsDeceased() bool {
    if p.DeceasedBoolean != nil {
        return *p.DeceasedBoolean
    }
    if p.DeceasedDateTime != nil {
        return true  // If date/time is set, patient is deceased
    }
    return false
}
```

### Setting Choice Fields Safely

Create helper methods to ensure mutual exclusion:

```go
func (p *Patient) SetDeceasedBoolean(value bool) {
    p.DeceasedBoolean = &value
    p.DeceasedDateTime = nil  // Clear other option
}

func (p *Patient) SetDeceasedDateTime(value primitives.DateTime) {
    p.DeceasedDateTime = &value
    p.DeceasedBoolean = nil  // Clear other option
}
```

## Best Practices

### 1. Always Validate After Setting

```go
patient.DeceasedBoolean = boolPtr(true)

if err := validator.Validate(patient); err != nil {
    log.Fatal(err)
}
```

### 2. Clear Other Options When Setting

To avoid validation errors, explicitly nil out other options:

```go
// Setting deceased as boolean
patient.DeceasedBoolean = boolPtr(true)
patient.DeceasedDateTime = nil  // Explicitly clear
```

### 3. Check Before Access

Always check for nil before accessing:

```go
if patient.DeceasedBoolean != nil {
    if *patient.DeceasedBoolean {
        fmt.Println("Patient is deceased")
    }
}
```

### 4. Use Type Switches for Complex Logic

For choice types with many options (like `Observation.value[x]`):

```go
func processValue(obs *Observation) {
    switch {
    case obs.ValueQuantity != nil:
        processQuantity(obs.ValueQuantity)
    case obs.ValueCodeableConcept != nil:
        processConcept(obs.ValueCodeableConcept)
    case obs.ValueString != nil:
        processString(*obs.ValueString)
    default:
        fmt.Println("No value set")
    }
}
```

### 5. Document Expected Choice in Comments

```go
// CreatePatient creates a patient with deceased status.
// deceased should be ONE of: boolean or dateTime
func CreatePatient(name string, deceased interface{}) (*Patient, error) {
    patient := &Patient{Name: []HumanName{{Text: &name}}}
    
    switch v := deceased.(type) {
    case bool:
        patient.DeceasedBoolean = &v
    case primitives.DateTime:
        patient.DeceasedDateTime = &v
    default:
        return nil, fmt.Errorf("invalid deceased type")
    }
    
    return patient, validator.Validate(patient)
}
```

## Migration Guide

### From Single Field with interface{}

Old implementation (using `any`):

```go
type Patient struct {
    Deceased *any `json:"deceased,omitempty"`
}

// Usage was unclear
patient.Deceased = true  // Type information lost
```

New implementation (multiple fields):

```go
type Patient struct {
    DeceasedBoolean  *bool     `json:"deceasedBoolean,omitempty" fhir:"choice=deceased"`
    DeceasedDateTime *DateTime `json:"deceasedDateTime,omitempty" fhir:"choice=deceased"`
}

// Usage is type-safe and clear
patient.DeceasedBoolean = boolPtr(true)
```

### Updating Code

1. **Identify choice fields** in your code (fields using `any` or `interface{}`)
2. **Replace with specific fields** based on FHIR spec
3. **Update JSON tags** to use specific names (e.g., `deceasedBoolean`)
4. **Add validation** to ensure mutual exclusion

## Error Messages

Validation errors clearly indicate which choice group has issues:

```
choice type 'deceased' has multiple fields set, only one is allowed: DeceasedBoolean, DeceasedDateTime
```

```
2 validation error(s):
  1. choice type 'deceased' has multiple fields set, only one is allowed: DeceasedBoolean, DeceasedDateTime
  2. choice type 'multipleBirth' has multiple fields set, only one is allowed: MultipleBirthBoolean, MultipleBirthInteger
```

## Summary

- **Choice types** represent fields that can have different types
- **Implementation**: Multiple Go fields with suffixed names
- **Constraint**: Only one field in each choice group can be set
- **Validation**: Automatic mutual exclusion checking
- **JSON**: Uses specific field names (e.g., `deceasedBoolean`)
- **Type Safety**: Go's type system enforces correct usage
