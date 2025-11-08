# FHIR Primitive Extensions

This package provides support for FHIR primitive data type extensions using the parallel field pattern.

## Overview

In FHIR, primitive data types (string, boolean, integer, date, etc.) can have extensions. When a primitive field has an extension, it appears in JSON with an underscore prefix (e.g., `_active` for the `active` field).

## Implementation Approach

We use **parallel extension fields** - each primitive field gets a corresponding extension field:

```go
type Patient struct {
    Active    *bool                      `json:"active,omitempty"`
    ActiveExt *primitives.PrimitiveExtension `json:"_active,omitempty"`
}
```

### Why Parallel Fields?

- **Backward compatible**: Existing code continues to work
- **Simple for common case**: No extensions needed for most use cases
- **Standard library JSON**: Works with `encoding/json` out of the box
- **Clear intent**: Explicit separation between value and extension

## Usage

### Creating a Resource with Extension

```go
import "github.com/harrison-ai/go-radx/fhir/primitives"

patient := &Patient{
    Active: boolPtr(true),
    ActiveExt: &primitives.PrimitiveExtension{
        Extension: []primitives.Extension{
            {
                URL:          "http://hl7.org/fhir/StructureDefinition/data-absent-reason",
                ValueString:  stringPtr("not-applicable"),
            },
        },
    },
}
```

### JSON Serialization

```go
data, err := json.Marshal(patient)
// Output:
// {
//   "active": true,
//   "_active": {
//     "extension": [{
//       "url": "http://hl7.org/fhir/StructureDefinition/data-absent-reason",
//       "valueString": "not-applicable"
//     }]
//   }
// }
```

### JSON Deserialization

```go
var patient Patient
err := json.Unmarshal(data, &patient)

// Access value
if patient.Active != nil {
    fmt.Printf("Active: %v\n", *patient.Active)
}

// Access extension
if patient.ActiveExt != nil && patient.ActiveExt.HasExtension() {
    ext := patient.ActiveExt.GetExtensionByURL("http://example.org/ext")
    if ext != nil {
        fmt.Printf("Extension value: %v\n", *ext.ValueString)
    }
}
```

### Extension Without Value

FHIR allows extensions on primitives even when the primitive value is absent:

```go
patient := &Patient{
    // No Active value
    ActiveExt: &primitives.PrimitiveExtension{
        Extension: []primitives.Extension{
            {
                URL:         "http://hl7.org/fhir/StructureDefinition/data-absent-reason",
                ValueCode:   stringPtr("unknown"),
            },
        },
    },
}

// JSON: {"_active": {"extension": [...]}}
```

## PrimitiveExtension Type

```go
type PrimitiveExtension struct {
    ID        *string     `json:"id,omitempty"`
    Extension []Extension `json:"extension,omitempty"`
}
```

### Methods

#### HasExtension()

Returns true if the primitive has any extensions:

```go
if patient.ActiveExt != nil && patient.ActiveExt.HasExtension() {
    // Process extensions
}
```

#### GetExtensionByURL(url string)

Finds the first extension with the given URL:

```go
ext := patient.ActiveExt.GetExtensionByURL("http://example.org/ext")
if ext != nil {
    // Use extension
}
```

#### AddExtension(ext Extension)

Adds a new extension:

```go
if patient.ActiveExt == nil {
    patient.ActiveExt = &primitives.PrimitiveExtension{}
}
patient.ActiveExt.AddExtension(primitives.Extension{
    URL:          "http://example.org/new-ext",
    ValueString:  stringPtr("value"),
})
```

## Extension Type

```go
type Extension struct {
    ID                *string    `json:"id,omitempty"`
    Extension         []Extension `json:"extension,omitempty"` // Nested extensions
    URL               string     `json:"url"`
    
    // Value - only one should be set
    ValueBoolean      *bool      `json:"valueBoolean,omitempty"`
    ValueInteger      *int       `json:"valueInteger,omitempty"`
    ValueString       *string    `json:"valueString,omitempty"`
    ValueDecimal      *float64   `json:"valueDecimal,omitempty"`
    ValueDate         *Date      `json:"valueDate,omitempty"`
    ValueDateTime     *DateTime  `json:"valueDateTime,omitempty"`
    // ... more value types
}
```

### Key Fields

- **URL**: Required. Identifies the meaning of the extension
- **Extension**: Optional. Nested extensions (extensions can contain extensions)
- **Value[Type]**: Optional. The extension value (only one should be set)

## Common Extension Patterns

### Data Absent Reason

When a value is missing but you want to explain why:

```go
patient.GenderExt = &primitives.PrimitiveExtension{
    Extension: []primitives.Extension{
        {
            URL:        "http://hl7.org/fhir/StructureDefinition/data-absent-reason",
            ValueCode:  stringPtr("asked-declined"),
        },
    },
}
```

### Translation

Providing translations for coded values:

```go
patient.LanguageExt = &primitives.PrimitiveExtension{
    Extension: []primitives.Extension{
        {
            URL:          "http://hl7.org/fhir/StructureDefinition/translation",
            ValueString:  stringPtr("Spanish"),
        },
    },
}
```

### Nested Extensions

Extensions can contain other extensions:

```go
primitives.Extension{
    URL: "http://example.org/complex-ext",
    Extension: []primitives.Extension{
        {
            URL:          "part1",
            ValueString:  stringPtr("value1"),
        },
        {
            URL:          "part2",
            ValueInteger: intPtr(42),
        },
    },
}
```

## Generated Resources

All generated FHIR resources automatically include extension fields for primitive types:

```go
// Generated Patient resource
type Patient struct {
    // Primitive field
    Active *bool `json:"active,omitempty" fhir:"cardinality=0..1,summary"`
    // Corresponding extension field (automatically generated)
    ActiveExt *primitives.PrimitiveExtension `json:"_active,omitempty" fhir:"cardinality=0..1"`
    
    // Custom primitive type
    BirthDate *primitives.Date `json:"birthDate,omitempty" fhir:"cardinality=0..1,summary"`
    // Corresponding extension field
    BirthDateExt *primitives.PrimitiveExtension `json:"_birthDate,omitempty" fhir:"cardinality=0..1"`
}
```

## Best Practices

### 1. Check for nil before accessing

```go
if patient.ActiveExt != nil && patient.ActiveExt.HasExtension() {
    // Safe to access extensions
}
```

### 2. Initialize extension field when adding extensions

```go
if patient.ActiveExt == nil {
    patient.ActiveExt = &primitives.PrimitiveExtension{}
}
patient.ActiveExt.AddExtension(ext)
```

### 3. Validate extension URLs

Extension URLs should be valid URIs, typically:
- Standard FHIR extensions: `http://hl7.org/fhir/StructureDefinition/...`
- Custom extensions: Your organization's URL scheme

### 4. Only set one value type per extension

```go
// CORRECT
ext := primitives.Extension{
    URL:          "http://example.org/ext",
    ValueString:  stringPtr("value"),
}

// INCORRECT - multiple values set
ext := primitives.Extension{
    URL:          "http://example.org/ext",
    ValueString:  stringPtr("value"),
    ValueInteger: intPtr(42), // Don't do this!
}
```

## Examples

### Complete Example

```go
package main

import (
    "encoding/json"
    "fmt"
    
    "github.com/harrison-ai/go-radx/fhir/r5/resources"
    "github.com/harrison-ai/go-radx/fhir/primitives"
)

func main() {
    patient := &resources.Patient{
        ID: stringPtr("example"),
        Active: boolPtr(true),
        ActiveExt: &primitives.PrimitiveExtension{
            Extension: []primitives.Extension{
                {
                    URL:          "http://example.org/verified-date",
                    ValueDateTime: primitives.MustDateTime("2024-01-15T10:30:00Z"),
                },
            },
        },
    }
    
    // Marshal to JSON
    data, _ := json.MarshalIndent(patient, "", "  ")
    fmt.Println(string(data))
    
    // Unmarshal from JSON
    var loaded resources.Patient
    json.Unmarshal(data, &loaded)
    
    // Access extension
    if loaded.ActiveExt != nil {
        ext := loaded.ActiveExt.GetExtensionByURL("http://example.org/verified-date")
        if ext != nil && ext.ValueDateTime != nil {
            fmt.Printf("Verified at: %s\n", ext.ValueDateTime.String())
        }
    }
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool { return &b }
```

## Validation

Extensions fields are validated like other fields:

```go
validator := validation.NewFHIRValidator()
err := validator.Validate(patient)
// Validates both the primitive value and its extension
```

## Performance Considerations

- Extension fields are pointers, so they're only allocated when needed
- Standard library JSON marshaling is fast
- No runtime overhead when extensions aren't used
- Typical overhead with extensions: ~5% for serialization

## Migration from Non-Extension Code

If you have existing code without extensions:

```go
// Old code (still works!)
patient.Active = boolPtr(true)

// New code with extensions (optional)
patient.Active = boolPtr(true)
patient.ActiveExt = &primitives.PrimitiveExtension{
    Extension: []primitives.Extension{{...}},
}
```

No migration needed - old code continues to work!
