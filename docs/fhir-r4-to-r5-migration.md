# FHIR R4 to R5 Migration Guide

This guide helps you migrate from FHIR R4 to FHIR R5 in go-radx.

## Overview

FHIR R5 is the latest version of the FHIR specification with several improvements and new resources. While R4 and R5 are largely similar, there are important differences to be aware of.

## Package Structure

```go
// R4
import "github.com/codeninja55/go-radx/fhir/r4/resources"

// R5
import "github.com/codeninja55/go-radx/fhir/r5/resources"
```

## Key Differences

### New Resources in R5

R5 introduces 12 new resources not present in R4:

1. **ActorDefinition** - Describes a participant in a process
2. **ArtifactAssessment** - Assessment of an artifact's quality, suitability, or appropriateness
3. **BiologicallyDerivedProductDispense** - Record of dispensing biologically derived product
4. **EncounterHistory** - Historical record of encounter statuses
5. **FormularyItem** - Definition of a medication product in a formulary
6. **GenomicStudy** - Genomic study investigation
7. **InventoryItem** - Functional description of an inventory item
8. **InventoryReport** - Report of inventory quantities
9. **Permission** - Access permission for a resource or system
10. **Requirements** - Requirements specification for a system
11. **SubscriptionStatus** - Status notification for subscriptions
12. **SubscriptionTopic** - Definition of subscription topic
13. **TestPlan** - Test plan for testing conformance
14. **Transport** - Transport of resources between systems

### Removed Resources from R4

The following R4 resources were removed in R5:

1. **CatalogEntry** - Replaced by other mechanisms
2. **EffectEvidenceSynthesis** - Merged into Evidence resource
3. **MedicinalProduct** - Renamed to MedicinalProductDefinition
4. **MedicinalProductAuthorization** - Renamed to RegulatedAuthorization
5. **MedicinalProductContraindication** - Moved to ClinicalUseDefinition
6. **MedicinalProductIndication** - Moved to ClinicalUseDefinition
7. **MedicinalProductIngredient** - Renamed to Ingredient
8. **MedicinalProductInteraction** - Moved to ClinicalUseDefinition
9. **MedicinalProductManufactured** - Merged into ManufacturedItemDefinition
10. **MedicinalProductPackaged** - Renamed to PackagedProductDefinition
11. **MedicinalProductPharmaceutical** - Renamed to AdministrableProductDefinition
12. **MedicinalProductUndesirableEffect** - Moved to ClinicalUseDefinition
13. **RiskEvidenceSynthesis** - Merged into Evidence resource
14. **SubstanceSpecification** - Renamed to SubstanceDefinition

### Type Changes

#### Integer64 Type

R5 introduces the `integer64` primitive type for large integers:

```go
// R4 - uses int/int32
type MolecularSequence struct {
    // ...
    ReferenceSeqStart *int32
    ReferenceSeqEnd   *int32
}

// R5 - uses int64 for large coordinates
type MolecularSequence struct {
    // ...
    ReferenceSeqStart *int64  // Can represent larger genomic coordinates
    ReferenceSeqEnd   *int64
}
```

#### CodeableReference Type

R5 adds `CodeableReference` which combines concept and reference:

```go
// R4 - separate fields
type Observation struct {
    Code    CodeableConcept
    Subject *Reference
}

// R5 - can use CodeableReference
type Observation struct {
    Code    CodeableConcept
    Subject *CodeableReference  // Can be either concept or reference
}
```

### Field Renaming

Common field name changes between R4 and R5:

| Resource | R4 Field | R5 Field | Notes |
|----------|----------|----------|-------|
| Patient | `deceased[x]` | `deceased[x]` | Same name, but type handling improved |
| Observation | `referenceRange` | `referenceRange` | Structure updated |
| Medication | `ingredient` | `ingredient` | BackboneElement structure changed |

### Cardinality Changes

Some fields have different cardinality in R5:

```go
// R4 - single value
type Patient struct {
    MultipleBirth *bool  // or *int32
}

// R5 - choice type handling improved
type Patient struct {
    MultipleBirth any  // bool or int64
}
```

## Migration Steps

### Step 1: Update Imports

```go
// Before (R4)
import "github.com/codeninja55/go-radx/fhir/r4/resources"

// After (R5)
import "github.com/codeninja55/go-radx/fhir/r5/resources"
```

### Step 2: Update Resource Construction

Check for renamed or removed resources:

```go
// R4
product := resources.MedicinalProduct{
    // ...
}

// R5 - Renamed
product := resources.MedicinalProductDefinition{
    // ... (fields may have changed)
}
```

### Step 3: Handle Type Changes

Update code that uses changed primitive types:

```go
// R4
sequence := resources.MolecularSequence{
    ReferenceSeqStart: int32Ptr(1000000),
}

// R5 - use int64
sequence := resources.MolecularSequence{
    ReferenceSeqStart: int64Ptr(1000000),
}
```

### Step 4: Test Thoroughly

Run comprehensive tests after migration:

```bash
mise run test
mise run test-coverage
```

## Common Gotchas

### 1. JSON Serialization Compatibility

R4 and R5 JSON may not be directly compatible due to:
- Renamed fields
- Different data types
- New required fields

Always validate against the target FHIR version.

### 2. Validation Rules

R5 has stricter validation rules in some areas:

```go
// R4 - may accept partial dates
birthDate := primitives.MustDate("2024")  // OK

// R5 - same, but more validation on related fields
patient := resources.Patient{
    BirthDate: &birthDate,
    Deceased:  &deceasedDate,  // Must be after birthDate
}
```

### 3. Extension Handling

Extension handling is more strict in R5:

```go
// Both R4 and R5 support extensions, but R5 validates more strictly
extension := resources.Extension{
    URL:   stringPtr("http://example.org/ext"),
    Value: "string value",  // R5 validates type more strictly
}
```

## Compatibility Layer

To support both R4 and R5 in the same codebase:

```go
package myapp

import (
    r4 "github.com/codeninja55/go-radx/fhir/r4/resources"
    r5 "github.com/codeninja55/go-radx/fhir/r5/resources"
)

// Create conversion functions
func ConvertPatientR4ToR5(r4Patient *r4.Patient) *r5.Patient {
    // Manual mapping
    r5Patient := &r5.Patient{
        ID:        r4Patient.ID,
        Active:    r4Patient.Active,
        Name:      convertHumanNames(r4Patient.Name),
        // ... map other fields
    }
    return r5Patient
}
```

## Resources

- [FHIR R4 Specification](https://hl7.org/fhir/R4/)
- [FHIR R5 Specification](https://hl7.org/fhir/R5/)
- [R4/R5 Conversion Spec](https://hl7.org/fhir/R5/diff.html)
- [go-radx FHIR Documentation](user-guide/fhir/index.md)

## Need Help?

- Check existing examples in `fhir/examples/`
- Review test files for usage patterns
- Open an issue on GitHub for migration questions
