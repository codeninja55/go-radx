// Package anonymize implements DICOM PS3.15 compliant de-identification profiles.
package anonymize

import (
	"github.com/codeninja55/go-radx/dicom/tag"
)

// initializeBasicProfile sets up actions for the Basic Application Level Confidentiality Profile.
//
// This implements DICOM PS3.15 Annex E Table E.1-1:
// Application Level Confidentiality Profile Attributes
//
// Reference: https://dicom.nema.org/medical/dicom/current/output/html/part15.html#table_E.1-1
func (a *Anonymizer) initializeBasicProfile() {
	// Patient Module attributes
	a.actions[tag.PatientName] = ActionDummy                // D
	a.actions[tag.PatientID] = ActionDummy                  // D
	a.actions[tag.PatientBirthDate] = ActionEmpty           // Z
	a.actions[tag.PatientBirthTime] = ActionRemove          // X
	a.actions[tag.PatientSex] = ActionKeep                  // Keep if RetainPatientCharacteristics
	a.actions[tag.PatientAge] = ActionKeep                  // Keep if RetainPatientCharacteristics
	a.actions[tag.PatientSize] = ActionKeep                 // Keep if RetainPatientCharacteristics
	a.actions[tag.PatientWeight] = ActionKeep               // Keep if RetainPatientCharacteristics
	a.actions[tag.OtherPatientIDs] = ActionRemove           // X
	a.actions[tag.OtherPatientNames] = ActionRemove         // X
	a.actions[tag.PatientBirthName] = ActionRemove          // X
	a.actions[tag.PatientMotherBirthName] = ActionRemove    // X
	a.actions[tag.MedicalRecordLocator] = ActionRemove      // X
	a.actions[tag.EthnicGroup] = ActionRemove               // X
	a.actions[tag.PatientComments] = ActionRemove           // X
	a.actions[tag.PatientSpeciesDescription] = ActionRemove // X
	a.actions[tag.PatientBreedDescription] = ActionRemove   // X
	a.actions[tag.ResponsiblePerson] = ActionRemove         // X
	a.actions[tag.ResponsibleOrganization] = ActionRemove   // X
	a.actions[tag.PatientIdentityRemoved] = ActionDummy     // Set to YES

	// General Study Module
	a.actions[tag.StudyInstanceUID] = ActionUID                      // U
	a.actions[tag.StudyDate] = ActionEmpty                           // Z/D
	a.actions[tag.StudyTime] = ActionEmpty                           // Z/D
	a.actions[tag.ReferringPhysicianName] = ActionEmpty              // Z
	a.actions[tag.ReferringPhysicianAddress] = ActionRemove          // X
	a.actions[tag.ReferringPhysicianTelephoneNumbers] = ActionRemove // X
	a.actions[tag.StudyID] = ActionEmpty                             // Z
	a.actions[tag.AccessionNumber] = ActionEmpty                     // Z
	a.actions[tag.IssuerOfAccessionNumberSequence] = ActionRemove    // X
	a.actions[tag.StudyDescription] = ActionClean                    // C - Clean descriptors
	a.actions[tag.PhysiciansOfRecord] = ActionRemove                 // X
	a.actions[tag.NameOfPhysiciansReadingStudy] = ActionRemove       // X
	a.actions[tag.RequestingPhysician] = ActionRemove                // X
	a.actions[tag.ConsultingPhysicianName] = ActionRemove            // X
	a.actions[tag.AdmittingDiagnosesDescription] = ActionRemove      // X
	a.actions[tag.ReferencedStudySequence] = ActionKeep              // Keep UIDs handled separately

	// General Series Module
	a.actions[tag.SeriesInstanceUID] = ActionUID            // U
	a.actions[tag.SeriesNumber] = ActionKeep                // K
	a.actions[tag.SeriesDate] = ActionEmpty                 // Z/D
	a.actions[tag.SeriesTime] = ActionEmpty                 // Z/D
	a.actions[tag.SeriesDescription] = ActionClean          // C
	a.actions[tag.PerformingPhysicianName] = ActionEmpty    // Z
	a.actions[tag.OperatorsName] = ActionEmpty              // Z
	a.actions[tag.ProtocolName] = ActionClean               // C
	a.actions[tag.RequestAttributesSequence] = ActionRemove // X

	// General Equipment Module
	a.actions[tag.InstitutionName] = ActionRemove             // X/D based on RetainDeviceIdentity
	a.actions[tag.InstitutionAddress] = ActionRemove          // X
	a.actions[tag.InstitutionalDepartmentName] = ActionRemove // X
	a.actions[tag.StationName] = ActionKeep                   // Keep if RetainDeviceIdentity
	a.actions[tag.DeviceSerialNumber] = ActionRemove          // X/D

	// General Image Module
	a.actions[tag.SOPInstanceUID] = ActionUID          // U
	a.actions[tag.AcquisitionDate] = ActionEmpty       // Z/D
	a.actions[tag.AcquisitionTime] = ActionEmpty       // Z/D
	a.actions[tag.AcquisitionDateTime] = ActionEmpty   // Z/D
	a.actions[tag.ContentDate] = ActionEmpty           // Z
	a.actions[tag.ContentTime] = ActionEmpty           // Z
	a.actions[tag.InstanceCreationDate] = ActionEmpty  // Z
	a.actions[tag.InstanceCreationTime] = ActionEmpty  // Z
	a.actions[tag.InstanceCreatorUID] = ActionRemove   // X
	a.actions[tag.DerivationDescription] = ActionClean // C

	// SOP Common Module
	a.actions[tag.InstanceNumber] = ActionKeep              // K
	a.actions[tag.TimezoneOffsetFromUTC] = ActionRemove     // X
	a.actions[tag.DigitalSignaturesSequence] = ActionRemove // X

	// Patient Study Module
	a.actions[tag.PatientSexNeutered] = ActionRemove // X

	// Overlay Identification (if present)
	// Note: Overlays are handled via RemoveOverlays option

	// Curve Identification (if present)
	// Note: Curves are handled via RemoveCurves option

	// Additional identifying attributes
	a.actions[tag.ImageComments] = ActionRemove               // X
	a.actions[tag.FrameComments] = ActionRemove               // X
	a.actions[tag.RequestingService] = ActionRemove           // X
	a.actions[tag.CurrentPatientLocation] = ActionRemove      // X
	a.actions[tag.PatientInstitutionResidence] = ActionRemove // X

	// Modified Attributes Sequence
	a.actions[tag.ModifiedAttributesSequence] = ActionRemove // X

	// Original Attributes Sequence
	a.actions[tag.OriginalAttributesSequence] = ActionRemove // X

	// Person Identification
	a.actions[tag.PersonName] = ActionRemove             // X
	a.actions[tag.PersonAddress] = ActionRemove          // X
	a.actions[tag.PersonTelephoneNumbers] = ActionRemove // X

	// Text observations and comments
	a.actions[tag.TextComments] = ActionRemove // X
	a.actions[tag.TextString] = ActionRemove   // X

	// Study and series comments
	a.actions[tag.AdditionalPatientHistory] = ActionRemove // X
	a.actions[tag.Occupation] = ActionRemove               // X
	a.actions[tag.MilitaryRank] = ActionRemove             // X
	a.actions[tag.BranchOfService] = ActionRemove          // X
	a.actions[tag.CountryOfResidence] = ActionRemove       // X
	a.actions[tag.RegionOfResidence] = ActionRemove        // X

	// Dates and times that may identify temporal patterns
	a.actions[tag.PerformedProcedureStepStartDate] = ActionEmpty // Z/D
	a.actions[tag.PerformedProcedureStepStartTime] = ActionEmpty // Z/D
	a.actions[tag.PerformedProcedureStepEndDate] = ActionEmpty   // Z/D
	a.actions[tag.PerformedProcedureStepEndTime] = ActionEmpty   // Z/D

	// File metadata that may contain identifying information
	a.actions[tag.MediaStorageSOPInstanceUID] = ActionUID // U - should match SOPInstanceUID

	// Apply options-based modifications
	if a.config.Options.RetainDeviceIdentity {
		a.actions[tag.InstitutionName] = ActionKeep
		a.actions[tag.StationName] = ActionKeep
		a.actions[tag.DeviceSerialNumber] = ActionKeep
		a.actions[tag.InstitutionalDepartmentName] = ActionKeep
	}

	if a.config.Options.RetainPatientCharacteristics {
		a.actions[tag.PatientAge] = ActionKeep
		a.actions[tag.PatientSex] = ActionKeep
		a.actions[tag.PatientSize] = ActionKeep
		a.actions[tag.PatientWeight] = ActionKeep
	} else {
		a.actions[tag.PatientAge] = ActionEmpty
		a.actions[tag.PatientSex] = ActionEmpty
		a.actions[tag.PatientSize] = ActionRemove
		a.actions[tag.PatientWeight] = ActionRemove
	}

	if a.config.Options.RetainUIDs {
		a.actions[tag.StudyInstanceUID] = ActionKeep
		a.actions[tag.SeriesInstanceUID] = ActionKeep
		a.actions[tag.SOPInstanceUID] = ActionKeep
		a.actions[tag.MediaStorageSOPInstanceUID] = ActionKeep
	}

	if a.config.Options.RetainLongitudinalTemporalInfo {
		// Apply offset to dates/times instead of removing
		a.actions[tag.StudyDate] = ActionCallback
		a.actions[tag.StudyTime] = ActionCallback
		a.actions[tag.SeriesDate] = ActionCallback
		a.actions[tag.SeriesTime] = ActionCallback
		a.actions[tag.AcquisitionDate] = ActionCallback
		a.actions[tag.AcquisitionTime] = ActionCallback
		a.actions[tag.AcquisitionDateTime] = ActionCallback
		a.actions[tag.ContentDate] = ActionCallback
		a.actions[tag.ContentTime] = ActionCallback
	}
}

// initializeCleanPixelDataProfile adds actions for the Clean Pixel Data Option.
//
// This removes burned-in annotations and overlays from pixel data.
func (a *Anonymizer) initializeCleanPixelDataProfile() {
	// This is handled via the CleanPixelData option
	// Actual pixel data cleaning would require image processing
	// For now, we document the requirement
}

// initializeCleanDescriptorsProfile adds actions for the Clean Descriptors Option.
//
// This cleans text fields of identifying information while preserving clinical content.
func (a *Anonymizer) initializeCleanDescriptorsProfile() {
	// Text fields that should be cleaned rather than removed
	a.actions[tag.StudyDescription] = ActionClean
	a.actions[tag.SeriesDescription] = ActionClean
	a.actions[tag.ProtocolName] = ActionClean
	a.actions[tag.DerivationDescription] = ActionClean
	a.actions[tag.ImageComments] = ActionClean
	a.actions[tag.RequestedProcedureDescription] = ActionClean
	a.actions[tag.PerformedProcedureStepDescription] = ActionClean
}

// Additional tag definitions that may not be in the main tag package
// These would need to be added to the tag package for complete coverage

// Tags used in anonymization that should be in tag package:
// - PatientBirthTime (0010,0032)
// - OtherPatientIDs (0010,1000)
// - OtherPatientNames (0010,1001)
// - PatientBirthName (0010,1005)
// - PatientAge (0010,1010)
// - PatientSize (0010,1020)
// - PatientWeight (0010,1030)
// - PatientMotherBirthName (0010,1060)
// - MedicalRecordLocator (0010,1090)
// - EthnicGroup (0010,2160)
// - PatientComments (0010,4000)
// - PatientSpeciesDescription (0010,2201)
// - PatientBreedDescription (0010,2292)
// - ResponsiblePerson (0010,2297)
// - ResponsibleOrganization (0010,2299)
// - PatientIdentityRemoved (0012,0062)
// - ReferringPhysicianAddress (0008,0092)
// - ReferringPhysicianTelephoneNumbers (0008,0094)
// - PhysiciansOfRecord (0008,1048)
// - NameOfPhysiciansReadingStudy (0008,1060)
// - RequestingPhysician (0032,1032)
// - ConsultingPhysicianName (0008,009C)
// - AdmittingDiagnosesDescription (0008,1080)
// - IssuerOfAccessionNumberSequence (0008,0051)
// - RequestAttributesSequence (0040,0275)
// - InstitutionAddress (0008,0081)
// - FrameComments (0020,9158)
// - CurrentPatientLocation (0038,0300)
// - PatientInstitutionResidence (0038,0400)
// - ModifiedAttributesSequence (0400,0550)
// - OriginalAttributesSequence (0400,0561)
// - PersonName (0040,A123)
// - PersonAddress (0040,A353)
// - PersonTelephoneNumbers (0040,A354)
// - TextComments (4000,4000)
// - TextString (2030,0020)
// - AdditionalPatientHistory (0010,21B0)
// - Occupation (0010,2180)
// - MilitaryRank (0010,1080)
// - BranchOfService (0010,1081)
// - CountryOfResidence (0010,2150)
// - RegionOfResidence (0010,2152)
// - PatientSexNeutered (0010,2203)
// - PerformedProcedureStepStartDate (0040,0244)
// - PerformedProcedureStepStartTime (0040,0245)
// - PerformedProcedureStepEndDate (0040,0250)
// - PerformedProcedureStepEndTime (0040,0251)
// - RequestedProcedureDescription (0032,1060)
// - PerformedProcedureStepDescription (0040,0254)
