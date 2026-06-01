package dicom

// This file holds the PS3.15 Annex E Table E.1-1 action set for the Basic
// Application Level Confidentiality Profile. Each confidentiality attribute carries
// the action the profile applies to it wherever it appears, including inside
// sequence items. The prototype shipped a sparse, top-level-only subset that falsely
// claimed PS3.15 compliance; this table covers the Basic Profile attribute set and
// is applied recursively (Codex DCM-013).
//
// Reference: https://dicom.nema.org/medical/dicom/current/output/html/part15.html#table_E.1-1

// deidAction is one PS3.15 Table E.1-1 action. The single-letter codes are the
// standard's own notation (X remove, Z replace with zero length, D replace with a
// non-zero dummy, C clean, U remap UID, K keep).
type deidAction uint8

const (
	// deidKeep leaves the attribute unchanged (K). It is the implicit action for any
	// attribute not in the table, so the zero value reads naturally.
	deidKeep deidAction = iota
	// deidRemove deletes the attribute (X).
	deidRemove
	// deidReplaceZero replaces the value with a zero-length value of the same VR (Z).
	deidReplaceZero
	// deidReplaceDummy replaces the value with a non-zero-length VR-valid dummy (D).
	deidReplaceDummy
	// deidClean replaces the value with one of similar meaning stripped of identity
	// (C). v1 cannot parse free text for identity, so a clean attribute is treated as
	// replace-with-zero (the safe direction): identity is removed even though
	// clinically benign content is not preserved.
	deidClean
	// deidReplaceUID remaps the UID through the stable per-run map (U).
	deidReplaceUID
	// deidShiftDate is not a Table E.1-1 code: it is the resolved action for a
	// date/time attribute when the caller opts in to temporal retention. It keeps the
	// value (DateModeKeep) or shifts it by the per-run offset (DateModeShift).
	deidShiftDate
)

// isPatientCharacteristic reports whether t is a general patient characteristic
// retained under the PS3.15 Retain Patient Characteristics sub-option.
func isPatientCharacteristic(t Tag) bool {
	switch t {
	case TagPatientAge, TagPatientSex, TagPatientSize, TagPatientWeight,
		TagPatientSexNeutered, TagSmokingStatus, TagPregnancyStatus:
		return true
	default:
		return false
	}
}

// isDeviceIdentity reports whether t is a device or institution identity attribute
// retained under the PS3.15 Retain Device Identity sub-option.
func isDeviceIdentity(t Tag) bool {
	switch t {
	case TagInstitutionName, TagInstitutionAddress, TagInstitutionCodeSequence,
		TagInstitutionalDepartmentName, TagStationName, TagDeviceSerialNumber,
		TagDeviceLabel, TagGantryID, TagPlateID, TagDetectorID, TagCassetteID:
		return true
	default:
		return false
	}
}

// basicProfileActions maps a confidentiality attribute to its Basic Profile action.
// It is built once at init from the keyword table below. Attributes whose keyword is
// not in this dictionary build are skipped rather than panicking, so a dictionary
// revision never breaks construction.
var basicProfileActions = buildBasicProfileActions()

// basicProfileAction returns the Basic Profile action for t, or (deidKeep, false)
// when t is not a listed confidentiality attribute.
func basicProfileAction(t Tag) (deidAction, bool) {
	a, ok := basicProfileActions[t]
	return a, ok
}

// basicProfileKeywordActions is the PS3.15 Table E.1-1 Basic Profile column, keyed by
// PS3.6 keyword. Date and time attributes carry their Basic-Profile action here; the
// profile overrides them when the caller opts in to temporal retention.
//
// The Basic Profile column collapses to one of X/Z/D/C/U for these attributes; a few
// attributes the standard marks "Z/D" or "X/D" (the latter selectable by a sub-option
// such as Retain Device Identity) are listed with their default Basic action and the
// profile flips them when the option is set.
var basicProfileKeywordActions = map[string]deidAction{
	// Instance and study identity (SOP common, file meta mirror).
	"InstanceCreatorUID":                 deidReplaceUID,
	"SOPInstanceUID":                     deidReplaceUID,
	"StudyInstanceUID":                   deidReplaceUID,
	"SeriesInstanceUID":                  deidReplaceUID,
	"FrameOfReferenceUID":                deidReplaceUID,
	"SynchronizationFrameOfReferenceUID": deidReplaceUID,
	"ReferencedFrameOfReferenceUID":      deidReplaceUID,
	"ConcatenationUID":                   deidReplaceUID,
	"DimensionOrganizationUID":           deidReplaceUID,
	"PaletteColorLookupTableUID":         deidReplaceUID,
	"ReferencedSOPInstanceUID":           deidReplaceUID,
	"StorageMediaFileSetUID":             deidReplaceUID,
	"FiducialUID":                        deidReplaceUID,
	"IrradiationEventUID":                deidReplaceUID,
	"TargetUID":                          deidReplaceUID,
	"TransactionUID":                     deidReplaceUID,
	"DeviceUID":                          deidReplaceUID,
	"FailedSOPInstanceUIDList":           deidReplaceUID,

	// Patient module.
	"PatientName":                        deidReplaceDummy,
	"PatientID":                          deidReplaceDummy,
	"IssuerOfPatientID":                  deidRemove,
	"TypeOfPatientID":                    deidRemove,
	"PatientBirthDate":                   deidReplaceZero,
	"PatientBirthTime":                   deidRemove,
	"PatientSex":                         deidReplaceZero,
	"PatientInsurancePlanCodeSequence":   deidRemove,
	"PatientPrimaryLanguageCodeSequence": deidRemove,
	"OtherPatientIDs":                    deidRemove,
	"OtherPatientIDsSequence":            deidRemove,
	"OtherPatientNames":                  deidRemove,
	"PatientBirthName":                   deidRemove,
	"PatientAge":                         deidRemove,
	"PatientSize":                        deidRemove,
	"PatientWeight":                      deidRemove,
	"PatientAddress":                     deidRemove,
	"InsurancePlanIdentification":        deidRemove,
	"PatientMotherBirthName":             deidRemove,
	"MilitaryRank":                       deidRemove,
	"BranchOfService":                    deidRemove,
	"MedicalRecordLocator":               deidRemove,
	"MedicalAlerts":                      deidRemove,
	"Allergies":                          deidRemove,
	"CountryOfResidence":                 deidRemove,
	"RegionOfResidence":                  deidRemove,
	"PatientTelephoneNumbers":            deidRemove,
	"EthnicGroup":                        deidRemove,
	"Occupation":                         deidRemove,
	"SmokingStatus":                      deidRemove,
	"AdditionalPatientHistory":           deidRemove,
	"PregnancyStatus":                    deidRemove,
	"LastMenstrualDate":                  deidRemove,
	"PatientReligiousPreference":         deidRemove,
	"PatientComments":                    deidRemove,
	"PatientSpeciesDescription":          deidRemove,
	"PatientBreedDescription":            deidRemove,
	"BreedRegistrationNumber":            deidRemove,
	"ResponsiblePerson":                  deidRemove,
	"ResponsibleOrganization":            deidRemove,
	"PatientSexNeutered":                 deidReplaceZero,
	"PatientState":                       deidRemove,
	"CurrentPatientLocation":             deidRemove,
	"PatientInstitutionResidence":        deidRemove,
	"ConfidentialityConstraintOnPatientDataDescription": deidRemove,

	// Study and visit identity.
	"AccessionNumber":                              deidReplaceZero,
	"IssuerOfAccessionNumberSequence":              deidRemove,
	"StudyID":                                      deidReplaceZero,
	"StudyIDIssuer":                                deidRemove,
	"StudyDate":                                    deidReplaceZero,
	"StudyTime":                                    deidReplaceZero,
	"StudyDescription":                             deidClean,
	"ReferringPhysicianName":                       deidReplaceZero,
	"ReferringPhysicianAddress":                    deidRemove,
	"ReferringPhysicianTelephoneNumbers":           deidRemove,
	"ReferringPhysicianIdentificationSequence":     deidRemove,
	"ConsultingPhysicianName":                      deidRemove,
	"ConsultingPhysicianIdentificationSequence":    deidRemove,
	"PhysiciansOfRecord":                           deidRemove,
	"PhysiciansOfRecordIdentificationSequence":     deidRemove,
	"NameOfPhysiciansReadingStudy":                 deidRemove,
	"PhysiciansReadingStudyIdentificationSequence": deidRemove,
	"RequestingPhysician":                          deidRemove,
	"RequestingService":                            deidRemove,
	"RequestedProcedureDescription":                deidClean,
	"AdmittingDiagnosesDescription":                deidRemove,
	"AdmittingDiagnosesCodeSequence":               deidRemove,
	"PatientHospitalDiscussion":                    deidRemove,
	"AdmissionID":                                  deidRemove,
	"IssuerOfAdmissionID":                          deidRemove,
	"IssuerOfAdmissionIDSequence":                  deidRemove,
	"ServiceEpisodeID":                             deidRemove,
	"ServiceEpisodeDescription":                    deidRemove,
	"IssuerOfServiceEpisodeID":                     deidRemove,
	"IssuerOfServiceEpisodeIDSequence":             deidRemove,
	"ReferencedPatientAliasSequence":               deidRemove,
	"AdmittingDate":                                deidRemove,
	"AdmittingTime":                                deidRemove,
	"DischargeDiagnosisDescription":                deidRemove,

	// Series module.
	"SeriesDate":              deidReplaceZero,
	"SeriesTime":              deidReplaceZero,
	"SeriesDescription":       deidClean,
	"PerformingPhysicianName": deidReplaceZero,
	"PerformingPhysicianIdentificationSequence": deidRemove,
	"OperatorsName":                          deidReplaceZero,
	"OperatorIdentificationSequence":         deidRemove,
	"ProtocolName":                           deidClean,
	"BodyPartExamined":                       deidKeep,
	"RequestAttributesSequence":              deidReplaceZero,
	"PerformedProcedureStepDescription":      deidClean,
	"PerformedProcedureStepID":               deidRemove,
	"PerformedProcedureStepStartDate":        deidReplaceZero,
	"PerformedProcedureStepStartTime":        deidReplaceZero,
	"PerformedProcedureStepEndDate":          deidReplaceZero,
	"PerformedProcedureStepEndTime":          deidReplaceZero,
	"CommentsOnThePerformedProcedureStep":    deidRemove,
	"ScheduledProcedureStepStartDate":        deidRemove,
	"ScheduledProcedureStepStartTime":        deidRemove,
	"ScheduledProcedureStepEndDate":          deidRemove,
	"ScheduledProcedureStepEndTime":          deidRemove,
	"ScheduledPerformingPhysicianName":       deidRemove,
	"ScheduledStationName":                   deidRemove,
	"ScheduledStationAETitle":                deidRemove,
	"ScheduledProcedureStepDescription":      deidRemove,
	"RequestedProcedureID":                   deidRemove,
	"FillerOrderNumberImagingServiceRequest": deidRemove,
	"PlacerOrderNumberImagingServiceRequest": deidRemove,
	"OrderCallbackPhoneNumber":               deidRemove,
	"OrderEnteredBy":                         deidRemove,
	"OrderEntererLocation":                   deidRemove,

	// Equipment module.
	"InstitutionName":             deidRemove,
	"InstitutionAddress":          deidRemove,
	"InstitutionCodeSequence":     deidRemove,
	"InstitutionalDepartmentName": deidRemove,
	"StationName":                 deidRemove,
	"DeviceSerialNumber":          deidRemove,
	"DeviceLabel":                 deidRemove,
	"GantryID":                    deidRemove,
	"PlateID":                     deidRemove,
	"DetectorID":                  deidRemove,
	"CassetteID":                  deidRemove,
	"SourceManufacturer":          deidRemove,

	// Image and acquisition module.
	"AcquisitionDate":           deidReplaceZero,
	"AcquisitionTime":           deidReplaceZero,
	"AcquisitionDateTime":       deidReplaceZero,
	"ContentDate":               deidReplaceZero,
	"ContentTime":               deidReplaceZero,
	"OverlayDate":               deidRemove,
	"OverlayTime":               deidRemove,
	"InstanceCreationDate":      deidReplaceZero,
	"InstanceCreationTime":      deidReplaceZero,
	"DateOfLastCalibration":     deidRemove,
	"TimeOfLastCalibration":     deidRemove,
	"AcquisitionComments":       deidRemove,
	"DerivationDescription":     deidClean,
	"ImageComments":             deidRemove,
	"FrameComments":             deidRemove,
	"ImagePresentationComments": deidRemove,
	"ImageType":                 deidKeep,

	// SOP common and provenance.
	"TimezoneOffsetFromUTC":                    deidRemove,
	"DigitalSignaturesSequence":                deidRemove,
	"DigitalSignatureUID":                      deidRemove,
	"MACParametersSequence":                    deidRemove,
	"DataSetTrailingPadding":                   deidRemove,
	"ContributionDescription":                  deidRemove,
	"ModifiedAttributesSequence":               deidRemove,
	"OriginalAttributesSequence":               deidRemove,
	"ReferencedImageSequence":                  deidReplaceZero,
	"SourceImageSequence":                      deidReplaceZero,
	"ReferencedPerformedProcedureStepSequence": deidReplaceZero,
	"DerivationCodeSequence":                   deidKeep,

	// Content, person, and free text.
	"PersonName":                                        deidRemove,
	"PersonAddress":                                     deidRemove,
	"PersonTelephoneNumbers":                            deidRemove,
	"PersonIdentificationCodeSequence":                  deidRemove,
	"VerifyingObserverName":                             deidReplaceDummy,
	"VerifyingObserverIdentificationCodeSequence":       deidRemove,
	"VerifyingObserverSequence":                         deidReplaceDummy,
	"VerifyingOrganization":                             deidRemove,
	"AuthorObserverSequence":                            deidRemove,
	"ParticipantSequence":                               deidRemove,
	"CustodialOrganizationSequence":                     deidRemove,
	"ContentCreatorName":                                deidReplaceZero,
	"ContentCreatorIdentificationCodeSequence":          deidRemove,
	"NameOfPhysiciansReadingStudyCodeSequence":          deidRemove,
	"TextComments":                                      deidRemove,
	"TextString":                                        deidRemove,
	"ContentSequence":                                   deidKeep,
	"TelephoneNumberTrial":                              deidRemove,
	"DistributionName":                                  deidRemove,
	"DistributionAddress":                               deidRemove,
	"NamesOfIntendedRecipientsOfResults":                deidRemove,
	"IntendedRecipientsOfResultsIdentificationSequence": deidRemove,
	"ImpressionsTrial":                                  deidRemove,
	"ResultsComments":                                   deidRemove,
	"InterpretationApproverSequence":                    deidRemove,
	"InterpretationAuthor":                              deidRemove,
	"InterpretationDiagnosisDescription":                deidRemove,
	"InterpretationIDIssuer":                            deidRemove,
	"InterpretationRecorder":                            deidRemove,
	"InterpretationTranscriber":                         deidRemove,
	"InterpretationText":                                deidRemove,
	"ReviewerName":                                      deidRemove,
	"DataSetName":                                       deidRemove,
	"ArbitraryText":                                     deidRemove,

	// Specimen and protocol-context identity.
	"BarcodeValue":                           deidRemove,
	"SpecimenAccessionNumber":                deidRemove,
	"SpecimenIdentifier":                     deidRemove,
	"SlideIdentifier":                        deidRemove,
	"AcquisitionDeviceProcessingDescription": deidClean,
	"DischargeDate":                          deidRemove,
	"DischargeTime":                          deidRemove,

	// The de-identification metadata itself: PatientIdentityRemoved is forced to YES
	// by the profile, so it must not be removed by the table walk.
	"PatientIdentityRemoved":             deidKeep,
	"DeidentificationMethod":             deidKeep,
	"DeidentificationMethodCodeSequence": deidKeep,
}

// buildBasicProfileActions resolves the keyword action table to tag-keyed actions
// once, dropping keywords absent from this dictionary build.
func buildBasicProfileActions() map[Tag]deidAction {
	m := make(map[Tag]deidAction, len(basicProfileKeywordActions))
	for keyword, action := range basicProfileKeywordActions {
		if t, ok := LookupKeyword(keyword); ok {
			m[t] = action
		}
	}
	return m
}
