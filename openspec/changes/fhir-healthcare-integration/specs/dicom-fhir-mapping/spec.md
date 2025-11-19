# Specification: DICOM to FHIR Mapping

## Overview

Provides bidirectional conversion between DICOM imaging data and FHIR resources, enabling radiology workflow
integration with clinical systems. Focuses on Study → ImagingStudy, Patient → Patient, and SR → DiagnosticReport
mappings.

## ADDED Requirements

### Requirement: Convert DICOM Study to FHIR ImagingStudy

Map DICOM Study/Series/Instance hierarchy to FHIR ImagingStudy resource with complete metadata preservation.

#### Scenario: Convert CT study with 3 series to ImagingStudy

**Given** a DICOM CT dataset with:
- StudyInstanceUID: `1.2.840.113619.2.55.3.604688119`
- StudyDescription: `CT CHEST W/ CONTRAST`
- 3 Series with distinct SeriesInstanceUIDs
- Each series has 100+ instances

**When** calling `DICOMStudyToImagingStudy(dataset, opts)`

**Then** the function returns an ImagingStudy resource where:
- `identifier[0].system` = `"urn:dicom:uid"`
- `identifier[0].value` = `"urn:oid:1.2.840.113619.2.55.3.604688119"`
- `description` = `"CT CHEST W/ CONTRAST"`
- `series` contains 3 elements with mapped SeriesInstanceUIDs
- `series[*].instance` contains instance references
- `status` = `"available"`

#### Scenario: Handle missing optional DICOM tags gracefully

**Given** a DICOM dataset with:
- StudyInstanceUID present
- StudyDescription missing (empty tag)
- PatientName missing

**When** calling `DICOMStudyToImagingStudy(dataset, opts)` with lenient mode

**Then** the function returns an ImagingStudy where:
- `identifier` is populated (required)
- `description` is `nil` (optional field)
- `subject` reference is created with identifier only (no display name)
- No error is returned

#### Scenario: Fail validation on missing StudyInstanceUID

**Given** a DICOM dataset missing StudyInstanceUID tag

**When** calling `DICOMStudyToImagingStudy(dataset, opts)`

**Then** the function returns error:
- Error contains `ErrMissingStudyUID`
- Error message includes context: `"DICOM study missing StudyInstanceUID (0020,000D)"`

---

### Requirement: Add WADO-RS endpoint references to ImagingStudy

Configure WADO-RS endpoint URLs for retrieving DICOM instances via web access.

#### Scenario: Add WADO-RS endpoint for study retrieval

**Given** a DICOM study and mapping options with:
```go
opts := MappingOptions{
    WADOEndpoint: "https://pacs.example.com/dicomweb",
}
```

**When** calling `DICOMStudyToImagingStudy(dataset, opts)`

**Then** the ImagingStudy resource includes:
- `endpoint[0].reference` = `"Endpoint/wado-rs"`
- `endpoint[0].type.code` = `"dicom-wado-rs"`
- Endpoint resource URL = `"https://pacs.example.com/dicomweb/studies/1.2.840..."`

#### Scenario: Omit endpoint when not configured

**Given** mapping options with no WADO endpoint:
```go
opts := MappingOptions{} // Empty
```

**When** calling `DICOMStudyToImagingStudy(dataset, opts)`

**Then** the ImagingStudy resource has:
- `endpoint` = `nil` (not populated)
- Study is valid but lacks retrieval capability

---

### Requirement: Convert DICOM Patient to FHIR Patient

Map DICOM patient demographic tags to FHIR Patient resource fields.

#### Scenario: Map patient demographics with full data

**Given** a DICOM dataset with:
- PatientName: `"Doe^John^^^"`
- PatientID: `"MRN12345"`
- PatientBirthDate: `"19741225"`
- PatientSex: `"M"`

**When** calling `DICOMPatientToFHIRPatient(dataset)`

**Then** the function returns a Patient resource where:
- `identifier[0].value` = `"MRN12345"`
- `name[0].family` = `"Doe"`
- `name[0].given[0]` = `"John"`
- `birthDate` = `"1974-12-25"` (FHIR date format)
- `gender` = `"male"` (FHIR code)

#### Scenario: Map patient with only PatientID (minimal data)

**Given** a DICOM dataset with:
- PatientID: `"MRN12345"`
- All other patient tags empty

**When** calling `DICOMPatientToFHIRPatient(dataset)`

**Then** the function returns a Patient where:
- `identifier[0].value` = `"MRN12345"`
- `name` = `nil`
- `birthDate` = `nil`
- `gender` = `nil`
- Patient is valid (identifier is minimum required)

#### Scenario: Map DICOM sex to FHIR gender codes

**Given** DICOM PatientSex values

**When** mapping to FHIR gender

**Then** the mappings are:
| DICOM Sex | FHIR Gender |
|-----------|-------------|
| `"M"`     | `"male"`    |
| `"F"`     | `"female"`  |
| `"O"`     | `"other"`   |
| `""`      | `nil`       |
| `"UNKNOWN"` | `"unknown"` |

---

### Requirement: Convert DICOM SR to FHIR DiagnosticReport

Map DICOM Structured Reports to FHIR DiagnosticReport with Observations for measurements.

#### Scenario: Convert TID 1500 Measurement Report to DiagnosticReport

**Given** a DICOM SR with:
- SOPClassUID = `"1.2.840.10008.5.1.4.1.1.88.33"` (Enhanced SR)
- ContentTemplateSequence TID = `1500` (Measurement Report)
- ContentSequence contains 3 measurements (e.g., tumor diameter)

**When** calling `DICOMSRToDiagnosticReport(dataset, TID1500)`

**Then** the function returns:
- DiagnosticReport with `status` = `"final"`
- `code` from SR DocumentTitle
- `result` contains 3 Observation references
- Each Observation has:
  - `valueQuantity` with numeric value and unit
  - `code` from measurement concept

#### Scenario: Handle unsupported SR template ID

**Given** a DICOM SR with:
- ContentTemplateSequence TID = `9999` (custom/unknown)

**When** calling `DICOMSRToDiagnosticReport(dataset, TID9999)`

**Then** the function returns error:
- Error = `ErrUnsupportedSRTemplate`
- Error message includes: `"SR template TID 9999 not supported"`

---

### Requirement: Preserve provenance and metadata

Track mapping provenance for audit trails and debugging.

#### Scenario: Add provenance extension to mapped ImagingStudy

**Given** a DICOM study being mapped

**When** calling `DICOMStudyToImagingStudy(dataset, opts)` with provenance enabled:
```go
opts := MappingOptions{
    IncludeProvenance: true,
}
```

**Then** the ImagingStudy includes:
- `meta.source` = `"urn:dicom:sop:1.2.840..."`
- `extension` with URL `"http://go-radx.dev/fhir/StructureDefinition/dicom-mapping"`
- Extension contains:
  - Mapping timestamp
  - Mapping library version
  - Original DICOM SOPInstanceUID

---

### Requirement: Support reverse mapping (FHIR to DICOM)

Convert FHIR DiagnosticReport back to DICOM SR for archival in PACS.

#### Scenario: Convert DiagnosticReport to DICOM SR TID 1500

**Given** a FHIR DiagnosticReport with:
- `code` = LOINC code for radiology report
- `result` contains 2 Observation resources with measurements
- `conclusionCode` with findings

**When** calling `DiagnosticReportToDICOMSR(report, TID1500)`

**Then** the function returns a DICOM dataset where:
- SOPClassUID = `"1.2.840.10008.5.1.4.1.1.88.33"` (Enhanced SR)
- ContentTemplateSequence TID = `1500`
- ContentSequence contains:
  - 2 measurement nodes from Observations
  - Finding nodes from conclusionCode
- All required DICOM tags populated

---

### Requirement: Handle modality-specific metadata

Map modality-specific DICOM tags to FHIR appropriately.

#### Scenario: Map CT acquisition parameters to ImagingStudy

**Given** a CT DICOM dataset with:
- Modality: `"CT"`
- KVP: `120`
- ExposureTime: `500`
- SeriesDescription: `"AXIAL 5MM"`

**When** calling `DICOMStudyToImagingStudy(dataset, opts)`

**Then** the ImagingStudy series includes:
- `modality.code` = `"CT"` (DICOM code)
- `description` = `"AXIAL 5MM"`
- `bodySite` mapped from BodyPartExamined if present
- CT-specific parameters in extensions (optional)

#### Scenario: Map MR sequence parameters

**Given** an MR DICOM dataset with:
- Modality: `"MR"`
- MagneticFieldStrength: `3.0`
- EchoTime: `30`
- RepetitionTime: `500`

**When** calling `DICOMStudyToImagingStudy(dataset, opts)`

**Then** the ImagingStudy series includes:
- `modality.code` = `"MR"`
- MR-specific parameters in extensions or omitted based on options

---

### Requirement: Validate FHIR output conformance

Ensure generated FHIR resources pass validation before returning.

#### Scenario: Validate generated ImagingStudy before return

**Given** DICOM study mapped to ImagingStudy

**When** calling `DICOMStudyToImagingStudy(dataset, opts)` with validation enabled:
```go
opts := MappingOptions{
    ValidateOutput: true,
}
```

**Then**:
- The function runs FHIR validation on ImagingStudy
- If validation fails, returns error with validation details
- If validation passes, returns valid ImagingStudy

#### Scenario: Skip validation for performance

**Given** high-throughput mapping scenario

**When** calling `DICOMStudyToImagingStudy(dataset, opts)` with:
```go
opts := MappingOptions{
    ValidateOutput: false, // Default
}
```

**Then** the function:
- Skips FHIR validation
- Returns ImagingStudy immediately
- User can validate separately if needed

---

## MODIFIED Requirements

None (new capability)

---

## REMOVED Requirements

None (new capability)

---

## Cross-References

- **Related to**: `smart-on-fhir` spec - May need authorization to access DICOM/FHIR endpoints
- **Related to**: `us-core-ig` spec - ImagingStudy and DiagnosticReport should validate against US Core profiles
- **Enables**: Future bulk export of imaging studies (requires mapping layer)

---

## Implementation Notes

### Package Structure
```
fhir/mapping/
├── imagingstudy.go       # DICOM Study → ImagingStudy
├── patient.go            # DICOM Patient → Patient
├── diagnosticreport.go   # DICOM SR ↔ DiagnosticReport
├── options.go            # MappingOptions configuration
├── errors.go             # Mapping-specific errors
└── provenance.go         # Provenance tracking
```

### Key Types
```go
type MappingOptions struct {
    WADOEndpoint        string
    IncludeProvenance   bool
    ValidateOutput      bool
    LenientMode         bool
    PreferredIdentifierSystem string
}

var (
    ErrMissingStudyUID       = errors.New("DICOM study missing StudyInstanceUID")
    ErrMissingPatientID      = errors.New("DICOM patient missing PatientID")
    ErrUnsupportedSRTemplate = errors.New("SR template not supported")
    ErrInvalidModality       = errors.New("invalid DICOM modality")
)
```

### Testing Approach
- Use real DICOM files from testdata/dicom/
- Test with Orthanc-generated DICOM studies
- Roundtrip tests (DICOM → FHIR → DICOM metadata)
- Edge case testing (missing tags, invalid values)
- Performance benchmarks (100+ instance studies)

### Dependencies
- Existing `dicom` package for dataset parsing
- Existing `fhir/r5/resources` for FHIR types
- Existing `fhir/validation` for output validation (optional)
