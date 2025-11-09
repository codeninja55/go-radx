# FHIR Choice Types

This document explains FHIR choice types and their implementation in go-radx.

## Overview

In FHIR, **choice types** (also called polymorphic elements) are fields that can have different data types.
In the FHIR specification, these are denoted with `[x]` suffix, such as `deceased[x]` or `value[x]`.

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
import (
    "github.com/harrison-ai/go-radx/fhir/primitives"
    "github.com/harrison-ai/go-radx/fhir/r5/resources"
)

// Option 1: Boolean
patient := &resources.Patient{
    DeceasedBoolean: boolPtr(true),
}
patient.ID = stringPtr("example")
patient.ResourceType = "Patient"

// Option 2: DateTime
patient := &resources.Patient{
    DeceasedDateTime: &primitives.DateTime{Time: time.Now()},
}
patient.ID = stringPtr("example")
patient.ResourceType = "Patient"

// INVALID: Setting both
patient := &resources.Patient{
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
patient := &resources.Patient{
    DeceasedBoolean: boolPtr(true),
}
patient.ID = stringPtr("example")
patient.ResourceType = "Patient"

// JSON output
{
  "resourceType": "Patient",
  "id": "example",
  "deceasedBoolean": true
}

// Alternative choice
patient := &resources.Patient{
    DeceasedDateTime: primitives.MustDateTime("2024-01-01T10:00:00Z"),
}
patient.ID = stringPtr("example")
patient.ResourceType = "Patient"

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

## Working with json.RawMessage Choice Types (R5)

Some choice types are truly polymorphic and can hold many different types of values. In these cases, the field uses `json.RawMessage` for type-safe lazy deserialization instead of being expanded into multiple fields.

### Examples of json.RawMessage Choice Types

**Extension.value[x]**:
```go
type Extension struct {
    URL   string            `json:"url"`
    Value json.RawMessage   `json:"value,omitempty"`  // Can be any FHIR type
}
```

**Parameters.parameter.value[x]**:
```go
type ParametersParameter struct {
    Name  string            `json:"name"`
    Value json.RawMessage   `json:"value,omitempty"`  // Can be any FHIR type
}
```

**UsageContext.value[x]**:
```go
type UsageContext struct {
    Code  CodeableConcept   `json:"code"`
    Value json.RawMessage   `json:"value,omitempty"`  // CodeableConcept, Quantity, Range, or Reference
}
```

### Using Generic Helper Functions

Use `fhir.UnmarshalResource[T]()` to unmarshal json.RawMessage choice types in a type-safe way:

```go
import (
    "github.com/harrison-ai/go-radx/fhir"
    "github.com/harrison-ai/go-radx/fhir/r5/types"
)

// Unmarshal Extension value as a string
ext := &types.Extension{
    URL:   "http://example.org/extension",
    Value: json.RawMessage(`{"valueString": "example value"}`),
}

// Type-safe unmarshaling with compile-time checking
var stringValue struct {
    ValueString string `json:"valueString"`
}
err := json.Unmarshal(ext.Value, &stringValue)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Extension value: %s\n", stringValue.ValueString)
```

### Creating json.RawMessage Choice Values

Marshal the value first, then assign to the json.RawMessage field:

```go
// Create a CodeableConcept value
concept := &types.CodeableConcept{
    Text: stringPtr("Example concept"),
    Coding: []types.Coding{
        {
            System: stringPtr("http://example.org"),
            Code:   stringPtr("example"),
        },
    },
}

// Marshal to json.RawMessage
conceptJSON, err := json.Marshal(map[string]interface{}{
    "valueCodeableConcept": concept,
})
if err != nil {
    log.Fatal(err)
}

// Assign to Extension
ext := &types.Extension{
    URL:   "http://example.org/extension",
    Value: conceptJSON,
}
```

### Type Checking json.RawMessage Choice Values

To determine which type a json.RawMessage contains, unmarshal to a map and check for type indicators:

```go
// Check which type the Extension value contains
var valueMap map[string]interface{}
if err := json.Unmarshal(ext.Value, &valueMap); err != nil {
    log.Fatal(err)
}

switch {
case valueMap["valueString"] != nil:
    var v struct{ ValueString string `json:"valueString"` }
    json.Unmarshal(ext.Value, &v)
    fmt.Printf("String value: %s\n", v.ValueString)

case valueMap["valueCodeableConcept"] != nil:
    var v struct{ ValueCodeableConcept types.CodeableConcept `json:"valueCodeableConcept"` }
    json.Unmarshal(ext.Value, &v)
    fmt.Printf("Concept: %s\n", *v.ValueCodeableConcept.Text)

case valueMap["valueInteger"] != nil:
    var v struct{ ValueInteger int `json:"valueInteger"` }
    json.Unmarshal(ext.Value, &v)
    fmt.Printf("Integer value: %d\n", v.ValueInteger)

default:
    fmt.Println("Unknown value type")
}
```

### Benefits of json.RawMessage for Choice Types

1. **Type Safety**: Compile-time type checking when unmarshaling
2. **Lazy Deserialization**: Only unmarshal when needed
3. **Memory Efficiency**: Store as bytes until accessed
4. **Flexibility**: Can handle any FHIR type without code generation for every combination
5. **Forward Compatibility**: New types can be added without regenerating code

## Summary

- **Choice types** represent fields that can have different types
- **Implementation**: Multiple Go fields with suffixed names (for simple choices) or json.RawMessage (for complex choices)
- **Constraint**: Only one field in each choice group can be set
- **Validation**: Automatic mutual exclusion checking
- **JSON**: Uses specific field names (e.g., `deceasedBoolean`)
- **Type Safety**: Go's type system and generics enforce correct usage
