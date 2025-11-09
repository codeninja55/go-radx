// Package anonymize provides DICOM PS3.15 compliant de-identification and anonymization.
//
// This package implements the DICOM standard's security profiles for protecting patient
// privacy while preserving clinical and research utility of medical images.
//
// # Supported Profiles
//
// The package implements multiple de-identification profiles from DICOM PS3.15:
//
//   - Basic Application Level Confidentiality Profile (E.1)
//   - Clean Pixel Data Option
//   - Clean Descriptors Option
//   - Retain UIDs Option
//   - Retain Device Identity Option
//   - Retain Patient Characteristics Option
//   - Retain Longitudinal Temporal Information Options
//
// # Basic Usage
//
// Apply the Basic Application Level Confidentiality Profile:
//
//	anonymizer := anonymize.NewAnonymizer(anonymize.ProfileBasic)
//	anonymizedDS, err := anonymizer.Anonymize(ds)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Custom Configuration
//
// Create a custom anonymizer with specific options:
//
//	config := anonymize.Config{
//	    Profile: anonymize.ProfileBasic,
//	    Options: anonymize.Options{
//	        RetainUIDs:                false,
//	        RetainDeviceIdentity:      false,
//	        RetainPatientCharacteristics: true,
//	        CleanPixelData:           false,
//	        CleanDescriptors:         true,
//	    },
//	    PatientName: "ANONYMOUS",
//	    PatientID:   "ANON001",
//	}
//	anonymizer := anonymize.NewAnonymizerWithConfig(config)
//
// # Action Types
//
// The package uses standard DICOM PS3.15 action types:
//
//   - D: Replace with dummy value
//   - Z: Replace with zero-length value or remove
//   - X: Remove
//   - K: Keep (no action)
//   - C: Clean (replace with values of similar meaning)
//   - U: Replace UIDs with new generated values
//
// # Compliance
//
// This implementation follows DICOM PS3.15 Attribute Confidentiality Profiles:
// https://dicom.nema.org/medical/dicom/current/output/html/part15.html#chapter_E
//
// # Important Notes
//
// De-identification cannot guarantee complete anonymity. Additional steps may be
// required depending on your use case:
//   - Review for burned-in annotations in pixel data
//   - Check for identifying information in private tags
//   - Validate against your institutional requirements
//   - Consider additional scrubbing of free-text fields
package anonymize
