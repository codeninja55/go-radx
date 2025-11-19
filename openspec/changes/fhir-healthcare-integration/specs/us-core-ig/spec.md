# Specification: US Core Implementation Guide Support

## Overview

Implementation Guide validation framework with US Core IG v6.1.0 profiles for US healthcare compliance. Provides
profile-aware validation, Must Support element checking, US-specific extensions, and value set bindings required for
EHR integration and regulatory compliance.

## ADDED Requirements

### Requirement: Validate resources against US Core profiles

Enforce US Core profile constraints on top of base FHIR validation.

#### Scenario: Validate US Core Patient with all required elements

**Given** a Patient resource with:
```json
{
  "resourceType": "Patient",
  "identifier": [{"system": "http://example.org/mrn", "value": "123456"}],
  "name": [{"family": "Doe", "given": ["John"]}],
  "gender": "male",
  "birthDate": "1974-12-25"
}
```

**When** validating against US Core Patient profile:
```go
validator := uscore.NewValidator()
err := validator.ValidateProfile(patient, uscore.ProfilePatient)
```

**Then** validation passes (no error returned)

#### Scenario: Detect missing Must Support element

**Given** a Patient resource missing `identifier` (Must Support in US Core)

**When** validating against US Core Patient

**Then** validation returns error:
- Error type: `validation.Errors`
- Error contains: `"Missing Must Support element: identifier"`
- Field path: `"Patient.identifier"`

#### Scenario: Validate US Core Observation with vital signs

**Given** an Observation resource with:
```json
{
  "resourceType": "Observation",
  "status": "final",
  "category": [{
    "coding": [{
      "system": "http://terminology.hl7.org/CodeSystem/observation-category",
      "code": "vital-signs"
    }]
  }],
  "code": {
    "coding": [{
      "system": "http://loinc.org",
      "code": "8867-4",
      "display": "Heart rate"
    }]
  },
  "subject": {"reference": "Patient/123"},
  "valueQuantity": {
    "value": 80,
    "unit": "beats/minute",
    "system": "http://unitsofmeasure.org",
    "code": "/min"
  }
}
```

**When** validating against US Core Vital Signs profile

**Then** validation passes with all required elements present

---

### Requirement: Check Must Support elements

Verify presence of elements marked as Must Support in US Core profiles.

#### Scenario: Identify all Must Support elements for US Core Patient

**Given** US Core Patient profile

**When** querying Must Support elements:
```go
elements := uscore.GetMustSupportElements(uscore.ProfilePatient)
```

**Then** elements include:
- `"identifier"`
- `"identifier.system"`
- `"identifier.value"`
- `"name"`
- `"name.family"`
- `"name.given"`
- `"gender"`
- `"birthDate"` (if known)

#### Scenario: Report missing Must Support elements

**Given** a Patient with:
- `name` present
- `gender` missing
- `identifier` missing

**When** checking Must Support:
```go
missing := validator.CheckMustSupport(patient, uscore.ProfilePatient)
```

**Then** missing contains:
- `"identifier"` (required Must Support)
- `"gender"` (required Must Support)
- Does NOT contain `"birthDate"` (conditional Must Support)

---

### Requirement: Validate US Core extensions

Support US-specific extensions for race, ethnicity, and birth sex.

#### Scenario: Validate US Core Race extension

**Given** a Patient with race extension:
```json
{
  "extension": [{
    "url": "http://hl7.org/fhir/us/core/StructureDefinition/us-core-race",
    "extension": [{
      "url": "ombCategory",
      "valueCoding": {
        "system": "urn:oid:2.16.840.1.113883.6.238",
        "code": "2106-3",
        "display": "White"
      }
    }, {
      "url": "text",
      "valueString": "White"
    }]
  }]
}
```

**When** validating race extension:
```go
err := uscore.ValidateRaceExtension(patient)
```

**Then** validation passes with:
- `ombCategory` present
- Valid OMB race code
- `text` field present (required)

#### Scenario: Validate US Core Ethnicity extension

**Given** a Patient with ethnicity extension:
```json
{
  "extension": [{
    "url": "http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity",
    "extension": [{
      "url": "ombCategory",
      "valueCoding": {
        "system": "urn:oid:2.16.840.1.113883.6.238",
        "code": "2135-2",
        "display": "Hispanic or Latino"
      }
    }, {
      "url": "text",
      "valueString": "Hispanic or Latino"
    }]
  }]
}
```

**When** validating ethnicity extension

**Then** validation passes with OMB ethnicity code

#### Scenario: Detect invalid extension structure

**Given** a race extension missing required `text` field

**When** validating race extension

**Then** validation returns error:
- Error: `"US Core Race extension missing required 'text' element"`

---

### Requirement: Enforce value set bindings

Validate coded values against US Core-required value sets.

#### Scenario: Validate administrative-gender binding

**Given** a Patient with `gender` = `"male"`

**When** validating against US Core Patient profile

**Then** validation passes (`"male"` is in AdministrativeGender value set)

#### Scenario: Reject invalid gender code

**Given** a Patient with `gender` = `"unknown-value"`

**When** validating against US Core Patient

**Then** validation returns error:
- Error: `"Invalid gender code 'unknown-value', must be from AdministrativeGender value set"`
- Valid codes: `["male", "female", "other", "unknown"]`

#### Scenario: Validate LOINC codes for observations

**Given** an Observation with:
- `code.coding[0].system` = `"http://loinc.org"`
- `code.coding[0].code` = `"8867-4"` (Heart rate)

**When** validating against US Core Observation profile

**Then** validation passes (LOINC code valid for vital signs)

#### Scenario: Enforce extensible binding with warning

**Given** a resource with code not in preferred value set (extensible binding)

**When** validating

**Then** validation:
- Does NOT fail (extensible allows other codes)
- Issues warning: `"Code not in preferred value set, consider using recommended codes"`

---

### Requirement: Support profile-specific validation rules

Implement US Core-specific constraints beyond base FHIR.

#### Scenario: Enforce US Core Patient identifier system requirement

**Given** a Patient with identifier lacking `system`

**When** validating against US Core Patient

**Then** validation returns error:
- Error: `"US Core Patient requires identifier.system for all identifiers"`

#### Scenario: Validate Observation data absent reason

**Given** an Observation with:
- `value[x]` absent
- `dataAbsentReason` present

**When** validating against US Core Observation

**Then** validation passes (dataAbsentReason explains missing value)

#### Scenario: Reject Observation with both value and dataAbsentReason

**Given** an Observation with:
- `valueQuantity` = `{"value": 80}`
- `dataAbsentReason` = `{"coding": [...]}`

**When** validating

**Then** validation returns error:
- Error: `"Cannot have both value and dataAbsentReason"`

---

### Requirement: Provide US Core profile definitions

Define US Core profiles as Go structs for programmatic access.

#### Scenario: Access US Core Patient profile metadata

**Given** US Core Patient profile

**When** accessing profile:
```go
profile := uscore.ProfilePatient
```

**Then** profile contains:
- `URL` = `"http://hl7.org/fhir/us/core/StructureDefinition/us-core-patient"`
- `BaseProfile` = `"Patient"`
- `MustSupport` = `["identifier", "name", "gender", ...]`
- `Extensions` = US Core Race, Ethnicity
- `ValueSetBindings` = AdministrativeGender for gender

#### Scenario: List all US Core profiles

**Given** US Core IG implementation

**When** querying available profiles:
```go
profiles := uscore.GetAllProfiles()
```

**Then** profiles include:
- `ProfilePatient` - US Core Patient
- `ProfileObservation` - US Core Observation
- `ProfileDiagnosticReport` - US Core DiagnosticReport
- `ProfileImagingStudy` - US Core ImagingStudy (if in v6.1.0)
- `ProfilePractitioner` - US Core Practitioner
- `ProfileOrganization` - US Core Organization
- `ProfileEncounter` - US Core Encounter

---

### Requirement: Support generic IG validation framework

Provide extensible framework for custom IGs beyond US Core.

#### Scenario: Define custom IG profile

**Given** a custom IG profile definition:
```go
customProfile := &ig.Profile{
    URL:         "http://example.org/fhir/StructureDefinition/custom-patient",
    BaseProfile: "Patient",
    MustSupport: []string{"identifier", "name", "telecom"},
    Extensions: []ig.ExtensionDefinition{{
        URL:         "http://example.org/custom-extension",
        Cardinality: "0..1",
    }},
}
```

**When** registering and validating:
```go
validator := ig.NewValidator()
validator.RegisterProfile(customProfile)
err := validator.ValidateProfile(patient, customProfile)
```

**Then** validation enforces custom profile rules

#### Scenario: Load IG profiles from StructureDefinition JSON

**Given** a StructureDefinition JSON file

**When** loading profile:
```go
profile, err := ig.LoadProfileFromJSON(jsonData)
validator.RegisterProfile(profile)
```

**Then** profile is dynamically loaded and available for validation

---

### Requirement: Validate US Core DiagnosticReport for imaging

Ensure imaging-related DiagnosticReports conform to US Core.

#### Scenario: Validate radiology DiagnosticReport

**Given** a DiagnosticReport with:
```json
{
  "resourceType": "DiagnosticReport",
  "status": "final",
  "category": [{
    "coding": [{
      "system": "http://loinc.org",
      "code": "LP29684-5",
      "display": "Radiology"
    }]
  }],
  "code": {
    "coding": [{
      "system": "http://loinc.org",
      "code": "30954-2",
      "display": "CT Chest"
    }]
  },
  "subject": {"reference": "Patient/123"},
  "effectiveDateTime": "2024-01-15T10:30:00Z",
  "issued": "2024-01-15T14:00:00Z"
}
```

**When** validating against US Core DiagnosticReport profile

**Then** validation passes with all required elements

#### Scenario: Link DiagnosticReport to ImagingStudy

**Given** a DiagnosticReport with:
- `imagingStudy` = `[{"reference": "ImagingStudy/456"}]`

**When** validating

**Then** validation passes (imagingStudy is valid reference)

---

### Requirement: Provide helper functions for US Core extensions

Simplify adding and extracting US Core extensions.

#### Scenario: Add US Core Race extension to Patient

**Given** a Patient resource

**When** adding race:
```go
err := uscore.AddRaceExtension(patient, uscore.RaceCategory{
    OMBCategory: "2106-3", // White
    Detailed:    []string{"2108-9"}, // European
    Text:        "White",
})
```

**Then** the Patient has race extension with proper structure

#### Scenario: Extract race from Patient

**Given** a Patient with race extension

**When** extracting:
```go
race, err := uscore.GetRaceExtension(patient)
```

**Then** race contains:
- `OMBCategory` = `"2106-3"`
- `Text` = `"White"`

---

## MODIFIED Requirements

None (new capability)

---

## REMOVED Requirements

None (new capability)

---

## Cross-References

- **Depends on**: Base FHIR validation in `fhir/validation` package
- **Related to**: `dicom-fhir-mapping` spec - Mapped resources should validate against US Core
- **Related to**: `smart-on-fhir` spec - US Core profiles accessed via SMART-authorized requests
- **Enables**: EHR integration, FDA regulatory compliance

---

## Implementation Notes

### Package Structure
```
fhir/ig/
├── validator.go          # Generic IG validation framework
├── profile.go            # Profile definition types
├── extension.go          # Extension validation
└── uscore/
    ├── profiles.go       # US Core profile definitions
    ├── validator.go      # US Core-specific validation
    ├── extensions.go     # Race, ethnicity, birthsex helpers
    ├── valuesets.go      # US Core value set bindings
    └── helpers.go        # Convenience functions
```

### Key Types
```go
// Generic IG framework
type Profile struct {
    URL             string
    BaseProfile     string
    MustSupport     []string
    Extensions      []ExtensionDefinition
    ValueSetBindings map[string]ValueSetBinding
    Constraints     []Constraint
}

type ExtensionDefinition struct {
    URL         string
    Cardinality string  // e.g., "0..1", "1..*"
    Type        string  // e.g., "Coding", "string"
}

type ValueSetBinding struct {
    Strength string  // "required", "extensible", "preferred"
    ValueSet string  // Value set URL
}

type Validator struct {
    profiles map[string]*Profile
    baseValidator *validation.FHIRValidator
}

// US Core specific
type RaceCategory struct {
    OMBCategory string
    Detailed    []string
    Text        string
}

type EthnicityCategory struct {
    OMBCategory string
    Detailed    []string
    Text        string
}

var (
    ProfilePatient          *Profile
    ProfileObservation      *Profile
    ProfileDiagnosticReport *Profile
    ProfileImagingStudy     *Profile
    ProfilePractitioner     *Profile
    ProfileOrganization     *Profile
    ProfileEncounter        *Profile
)
```

### Testing Approach
- Validate all [US Core examples](https://hl7.org/fhir/us/core/examples.html) from HL7 spec
- Must Support element coverage tests (100% of profiles)
- Extension validation tests (race, ethnicity, birthsex)
- Value set binding tests (required vs extensible)
- Custom IG profile registration and validation
- Performance: Validate 1000 patients in <10 seconds

### Dependencies
- Existing `fhir/validation` package for base validation
- Existing `fhir/r5/resources` for FHIR types
- No external dependencies

### US Core Version
- **Target**: US Core v6.1.0 (FHIR R4)
- **Future**: Support US Core v7.0.0 when released
- **Multi-version**: Consider `uscore/v6` and `uscore/v7` packages if needed

### Value Sets to Implement
Priority value sets for US Core compliance:
1. **AdministrativeGender** (required binding)
2. **OMB Race Categories** (extensible)
3. **OMB Ethnicity Categories** (extensible)
4. **US Core Birth Sex** (required)
5. **LOINC** (for Observation codes)
6. **SNOMED CT** (for clinical findings)