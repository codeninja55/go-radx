# FHIR Resource Inheritance

This document describes the resource inheritance pattern used in the go-radx FHIR library.

## Overview

FHIR defines a hierarchy of resource types:
- **Resource**: Base type for all FHIR resources
- **DomainResource**: Extends Resource, adds human-readable narrative
- **Specific Resources**: Patient, Observation, etc. extend DomainResource

## Implementation Using Struct Embedding

We use Go's struct embedding to implement FHIR resource inheritance:

```go
// Base Resource type
type Resource struct {
    ResourceType  string
    ID            *string
    Meta          *Meta
    ImplicitRules *string
    Language      *string
}

// DomainResource embeds Resource
type DomainResource struct {
    Resource
    Text              *Narrative
    Contained         []interface{}
    Extension         []Extension
    ModifierExtension []Extension
}

// Patient embeds DomainResource
type Patient struct {
    fhir.DomainResource
    Active   *bool
    Name     []HumanName
    // ... patient-specific fields
}
```

## Benefits

### 1. Matches FHIR Specification

The inheritance hierarchy matches FHIR's structure definition hierarchy, making it easier to understand and maintain.

### 2. Reduces Code Duplication

Base fields (ID, Meta, Text, Extension, etc.) are defined once and inherited by all resources:

```go
// Without embedding - ~10 fields repeated in every resource
type Patient struct {
    ID    *string
    Meta  *Meta
    Text  *Narrative
    // ... 7 more base fields
    // ... patient fields
}

// With embedding - base fields inherited
type Patient struct {
    fhir.DomainResource
    // ... patient fields only
}
```

### 3. Direct Field Access

Embedded fields are accessible directly:

```go
patient := &Patient{
    Active: boolPtr(true),
}

// Direct access to embedded Resource fields
patient.ID = stringPtr("123")
patient.Meta = &fhir.Meta{VersionID: stringPtr("1")}
patient.Text = &fhir.Narrative{Status: "generated", Div: "<div>..."}
patient.Extension = []fhir.Extension{{URL: "http://..."}}
```

### 4. JSON Serialization Compatibility

Go's JSON encoder automatically flattens embedded structs, maintaining FHIR JSON compatibility:

```go
patient := &Patient{
    DomainResource: fhir.DomainResource{
        Resource: fhir.Resource{
            ID: stringPtr("123"),
        },
        Text: &fhir.Narrative{Status: "generated", Div: "<div>..."},
    },
    Active: boolPtr(true),
}

json.Marshal(patient)
// Output:
// {
//   "id": "123",
//   "text": {"status": "generated", "div": "<div>..."},
//   "active": true
// }
```

## Resource Hierarchy

### Resource (Base)

All FHIR resources inherit from Resource:

**Fields:**
- `ResourceType` (string) - The type of resource
- `ID` (*string) - Logical id of the resource
- `Meta` (*Meta) - Metadata about the resource
- `ImplicitRules` (*string) - A set of rules under which this content was created
- `Language` (*string) - Language of the resource content

**Extension Fields:**
- `IDExt`, `ImplicitRulesExt`, `LanguageExt` - Primitive extensions

### DomainResource

Most FHIR resources extend DomainResource, which adds human-readable content:

**Inherits:** Resource

**Additional Fields:**
- `Text` (*Narrative) - Human-readable summary
- `Contained` ([]interface{}) - Contained inline resources
- `Extension` ([]Extension) - Additional content defined by implementations
- `ModifierExtension` ([]Extension) - Extensions that cannot be ignored

### Specific Resources

Individual resources (Patient, Observation, etc.) extend DomainResource:

```go
type Patient struct {
    fhir.DomainResource
    // Patient-specific fields
}

type Observation struct {
    fhir.DomainResource
    // Observation-specific fields
}
```

## Usage Examples

### Creating a Resource

```go
import "github.com/codeninja55/go-radx/fhir"

patient := &Patient{
    DomainResource: fhir.DomainResource{
        Resource: fhir.Resource{
            ID:       stringPtr("example"),
            Language: stringPtr("en-US"),
        },
        Text: &fhir.Narrative{
            Status: "generated",
            Div:    "<div>John Doe</div>",
        },
        Extension: []fhir.Extension{
            {
                URL:         "http://example.org/ext",
                ValueString: stringPtr("extension-value"),
            },
        },
    },
    Active: boolPtr(true),
    Name: []HumanName{
        {
            Family: stringPtr("Doe"),
            Given:  []string{"John"},
        },
    },
}
```

### Accessing Base Fields

```go
// Direct access through embedding
fmt.Printf("ID: %s\n", *patient.ID)
fmt.Printf("Language: %s\n", *patient.Language)
fmt.Printf("Text Status: %s\n", patient.Text.Status)

// Modify base fields
patient.Meta = &fhir.Meta{
    VersionID:   stringPtr("2"),
    LastUpdated: now(),
}
```

### Working with Extensions

```go
// Add extension to any resource
patient.Extension = append(patient.Extension, fhir.Extension{
    URL:          "http://hl7.org/fhir/StructureDefinition/patient-birthPlace",
    ValueAddress: &Address{City: stringPtr("Seattle")},
})

// Find extension by URL
for _, ext := range patient.Extension {
    if ext.URL == "http://example.org/target" {
        fmt.Printf("Found extension: %v\n", ext.ValueString)
    }
}
```

### Working with Contained Resources (R5)

In R5, the `Contained` field uses `json.RawMessage` for type-safe lazy deserialization. Use the generic helper functions to work with contained resources:

#### Adding Contained Resources

```go
import (
    "github.com/codeninja55/go-radx/fhir"
    "github.com/codeninja55/go-radx/fhir/r5/resources"
)

// Create a main resource (e.g., Observation)
observation := &resources.Observation{
    Status: "final",
    Code: resources.CodeableConcept{
        Text: stringPtr("Blood pressure"),
    },
}
observation.ID = stringPtr("obs-1")
observation.ResourceType = "Observation"

// Create a contained resource (e.g., Patient)
patient := &resources.Patient{
    Active: boolPtr(true),
    Gender: stringPtr("female"),
}
patient.ID = stringPtr("patient-1")
patient.ResourceType = "Patient"

// Add patient to contained resources
var err error
observation.Contained, err = fhir.AddContainedResource(observation.Contained, patient)
if err != nil {
    log.Fatal(err)
}

// Reference the contained patient
observation.Subject = &resources.Reference{
    Reference: stringPtr("#patient-1"),
}
```

#### Accessing Contained Resources by Index

```go
// Get the first contained resource as a Patient
patient, err := fhir.UnmarshalContainedResource[resources.Patient](observation.Contained, 0)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Contained patient: %s\n", *patient.ID)
```

#### Finding Contained Resources by ID

```go
// Find contained resource by ID
raw, err := fhir.GetContainedResourceByID(observation.Contained, "patient-1")
if err != nil {
    log.Fatal(err)
}

// Unmarshal to specific type
patient, err := fhir.UnmarshalResource[resources.Patient](raw)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found patient: %s\n", *patient.ID)
```

#### Multiple Contained Resources

```go
// Add multiple resources of different types
practitioner := &resources.Practitioner{
    Active: boolPtr(true),
}
practitioner.ID = stringPtr("pract-1")
practitioner.ResourceType = "Practitioner"

observation.Contained, _ = fhir.AddContainedResource(observation.Contained, patient)
observation.Contained, _ = fhir.AddContainedResource(observation.Contained, practitioner)

// Access each one
patient, _ := fhir.UnmarshalContainedResource[resources.Patient](observation.Contained, 0)
pract, _ := fhir.UnmarshalContainedResource[resources.Practitioner](observation.Contained, 1)
```

### JSON Serialization

```go
import "encoding/json"

// Marshal to JSON
data, err := json.Marshal(patient)
if err != nil {
    log.Fatal(err)
}

// Unmarshal from JSON
var loaded Patient
err = json.Unmarshal(data, &loaded)
if err != nil {
    log.Fatal(err)
}

// All fields accessible through embedding
fmt.Printf("Loaded ID: %s\n", *loaded.ID)
fmt.Printf("Loaded Active: %v\n", *loaded.Active)
```

## Type Assertions and Interface Methods

Since resources embed base types, you can work with them polymorphically:

```go
// Function accepting any resource
func printResourceID(r interface{ GetID() *string }) {
    if id := r.GetID(); id != nil {
        fmt.Printf("Resource ID: %s\n", *id)
    }
}

// Add method to Resource
func (r *Resource) GetID() *string {
    return r.ID
}

// All resources inherit GetID()
patient := &Patient{...}
observation := &Observation{...}

printResourceID(patient)
printResourceID(observation)
```

## Migration Guide

### From Flattened Structs

If you have existing code using flattened resource structs:

```go
// Old (flattened)
type Patient struct {
    ID     *string
    Meta   *Meta
    Text   *Narrative
    Active *bool
    Name   []HumanName
}

patient := &Patient{
    ID:     stringPtr("123"),
    Active: boolPtr(true),
}
```

Migrate to embedded structs:

```go
// New (embedded)
type Patient struct {
    fhir.DomainResource
    Active *bool
    Name   []HumanName
}

patient := &Patient{
    DomainResource: fhir.DomainResource{
        Resource: fhir.Resource{
            ID: stringPtr("123"),
        },
    },
    Active: boolPtr(true),
}

// Or initialize base fields after creation
patient := &Patient{Active: boolPtr(true)}
patient.ID = stringPtr("123")
```

### Field Access

Field access remains the same due to embedding:

```go
// Both old and new:
patient.ID = stringPtr("123")
patient.Active = boolPtr(true)

// No code changes needed for field access!
```

## Best Practices

### 1. Initialize Base Fields After Creation

For cleaner code, initialize resource-specific fields first, then set base fields:

```go
patient := &Patient{
    Active: boolPtr(true),
    Name:   []HumanName{{Family: stringPtr("Doe")}},
}

// Set base fields
patient.ID = stringPtr("example")
patient.Meta = &fhir.Meta{VersionID: stringPtr("1")}
```

### 2. Use Helper Functions for Base Fields

```go
func SetResourceMeta(r *fhir.Resource, versionID string) {
    r.Meta = &fhir.Meta{
        VersionID:   stringPtr(versionID),
        LastUpdated: now(),
    }
}

// Works for all resources
SetResourceMeta(&patient.Resource, "1")
SetResourceMeta(&observation.Resource, "2")
```

### 3. Validate Resources

Use the validation framework on the complete resource:

```go
validator := validation.NewFHIRValidator()
err := validator.Validate(patient)
// Validates both embedded base fields and resource-specific fields
```

## Technical Details

### JSON Tag Omission

Embedded fields don't have JSON tags, allowing Go's JSON encoder to automatically flatten the structure:

```go
type Patient struct {
    fhir.DomainResource  // No JSON tag - will be flattened
    Active *bool `json:"active,omitempty"`
}
```

### Field Shadowing

If a resource redefines a base field (not recommended), it shadows the embedded field:

```go
type CustomResource struct {
    fhir.DomainResource
    ID string  // Shadows Resource.ID (don't do this!)
}
```

### Zero Values

Embedded structs are zero-initialized. Check for nil when accessing pointer fields:

```go
patient := &Patient{}
// patient.ID is nil
// patient.Meta is nil
// patient.Text is nil

if patient.Meta != nil {
    fmt.Println(*patient.Meta.VersionID)
}
```

## Generated Code

The code generator automatically uses embedding for resources that extend DomainResource or Resource:

```go
// Generator detects BaseDefinition and adds embedding
if def.BaseDefinition == "http://hl7.org/fhir/StructureDefinition/DomainResource" {
    // Embeds fhir.DomainResource
    // Filters out base fields
}
```

Generated resources include:
1. Embedded base type (fhir.DomainResource or fhir.Resource)
2. Resource-specific fields only
3. Extension fields for primitives
4. Validation tags on all fields

## Summary

- **Resource hierarchy**: Resource → DomainResource → Specific Resources
- **Implementation**: Go struct embedding
- **Benefits**: DRY, type-safe, matches FHIR spec, maintains JSON compatibility
- **Usage**: Direct field access, no changes to existing code
- **Generated**: Automatic embedding in generated resources
